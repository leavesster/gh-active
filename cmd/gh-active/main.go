package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/spf13/cobra"
	"github.com/leavesster/gh-active/internal/config"
	ghclient "github.com/leavesster/gh-active/internal/github"
	"github.com/leavesster/gh-active/internal/llm"
	"github.com/leavesster/gh-active/internal/report"
	"github.com/leavesster/gh-active/pkg/model"
)

func main() {
	root := &cobra.Command{
		Use:   "gh-active",
		Short: "Generate weekly reports from GitHub activity",
	}

	root.AddCommand(reportCmd())
	root.AddCommand(initCmd())

	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}

func reportCmd() *cobra.Command {
	var (
		user    string
		start   string
		end     string
		week    string
		llmName string
		output  string
		noLLM   bool
	)

	cmd := &cobra.Command{
		Use:   "report",
		Short: "Generate a weekly report",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}

			if cfg.GitHub.Token == "" {
				return fmt.Errorf("GitHub token required: set GITHUB_TOKEN or configure ~/.gh-active.yaml")
			}

			startTime, endTime, err := parseTimeRange(start, end, week)
			if err != nil {
				return err
			}

			ctx := context.Background()
			client := ghclient.NewClient(cfg.GitHub.Token)

			if user == "" {
				user, err = client.AuthenticatedUser(ctx)
				if err != nil {
					return fmt.Errorf("resolve current user (try --user flag): %w", err)
				}
			}

			fmt.Fprintf(os.Stderr, "Fetching events for %s (%s ~ %s)...\n",
				user, startTime.Format("2006-01-02"), endTime.Format("2006-01-02"))

			events, err := client.FetchEvents(ctx, user, startTime, endTime)
			if err != nil {
				return fmt.Errorf("fetch events: %w", err)
			}
			fmt.Fprintf(os.Stderr, "Found %d events\n", len(events))

			activities, err := client.ParseEvents(ctx, events)
			if err != nil {
				return fmt.Errorf("parse events: %w", err)
			}

			sort.Slice(activities, func(i, j int) bool {
				return activities[i].CreatedAt.Before(activities[j].CreatedAt)
			})

			fmt.Fprintf(os.Stderr, "Parsed %d activities\n", len(activities))

			r := &model.WeeklyReport{
				Username:   user,
				StartDate:  startTime,
				EndDate:    endTime,
				Activities: activities,
			}

			if !noLLM {
				backend := llmName
				if backend == "" {
					backend = cfg.LLM.Default
				}

				provider, err := newLLM(cfg, backend)
				if err != nil {
					return err
				}

				fmt.Fprintf(os.Stderr, "Generating summary with %s...\n", backend)
				summary, err := provider.Summarize(activities, llm.SummarizeOpts{
					Language: cfg.Report.Language,
					Username: user,
					Start:    startTime,
					End:      endTime,
				})
				if err != nil {
					return fmt.Errorf("LLM summarize: %w", err)
				}
				r.Summary = summary
			}

			md := report.GenerateMarkdown(r)

			if output != "" {
				if err := os.WriteFile(output, []byte(md), 0644); err != nil {
					return fmt.Errorf("write output: %w", err)
				}
				fmt.Fprintf(os.Stderr, "Report written to %s\n", output)
			} else {
				fmt.Print(md)
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&user, "user", "", "GitHub username (default: authenticated user)")
	cmd.Flags().StringVar(&start, "start", "", "Start date (YYYY-MM-DD), default: last Monday")
	cmd.Flags().StringVar(&end, "end", "", "End date (YYYY-MM-DD), default: last Sunday")
	cmd.Flags().StringVar(&week, "week", "", "\"previous\" for last week, or any date (YYYY-MM-DD) to pick that week")
	cmd.Flags().StringVar(&llmName, "llm", "", "LLM backend (claude, openai)")
	cmd.Flags().StringVarP(&output, "output", "o", "", "Output file path")
	cmd.Flags().BoolVar(&noLLM, "no-llm", false, "Skip LLM summarization")

	return cmd
}

func initCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Create default config file at ~/.gh-active.yaml",
		RunE: func(cmd *cobra.Command, args []string) error {
			home, err := os.UserHomeDir()
			if err != nil {
				return err
			}
			path := filepath.Join(home, ".gh-active.yaml")

			if _, err := os.Stat(path); err == nil {
				return fmt.Errorf("%s already exists", path)
			}

			if err := os.WriteFile(path, []byte(config.DefaultYAML()), 0600); err != nil {
				return err
			}
			fmt.Printf("Config created at %s\n", path)
			return nil
		},
	}
}

func parseTimeRange(start, end, week string) (time.Time, time.Time, error) {
	now := time.Now()

	// --week=previous: last completed week (Mon~Sun)
	if week == "previous" {
		mon := weekMonday(now).AddDate(0, 0, -7)
		sun := mon.AddDate(0, 0, 7).Add(-time.Second)
		return mon, sun, nil
	}

	// --week=YYYY-MM-DD: Monday~Sunday of that week
	if week != "" {
		d, err := time.Parse("2006-01-02", week)
		if err != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("invalid week date: %w", err)
		}
		mon := weekMonday(d)
		sun := mon.AddDate(0, 0, 7).Add(-time.Second)
		return mon, sun, nil
	}

	if start != "" && end != "" {
		s, err := time.Parse("2006-01-02", start)
		if err != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("invalid start date: %w", err)
		}
		e, err := time.Parse("2006-01-02", end)
		if err != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("invalid end date: %w", err)
		}
		return s, e.Add(24*time.Hour - time.Second), nil
	}

	// default: current week (this Monday ~ now)
	return weekMonday(now), now, nil
}

// weekMonday returns Monday 00:00:00 of the week containing d.
func weekMonday(d time.Time) time.Time {
	wd := d.Weekday()
	if wd == time.Sunday {
		wd = 7
	}
	mon := d.AddDate(0, 0, -int(wd-time.Monday))
	return time.Date(mon.Year(), mon.Month(), mon.Day(), 0, 0, 0, 0, d.Location())
}

func newLLM(cfg *config.Config, backend string) (llm.LLM, error) {
	switch backend {
	case "claude":
		if cfg.LLM.Claude.APIKey == "" {
			return nil, fmt.Errorf("Claude API key required: set ANTHROPIC_API_KEY or configure ~/.gh-active.yaml")
		}
		return llm.NewClaude(cfg.LLM.Claude.APIKey, cfg.LLM.Claude.Model, cfg.LLM.Claude.BaseURL), nil
	case "openai":
		if cfg.LLM.OpenAI.APIKey == "" {
			return nil, fmt.Errorf("OpenAI API key required: set OPENAI_API_KEY or configure ~/.gh-active.yaml")
		}
		return llm.NewOpenAI(cfg.LLM.OpenAI.APIKey, cfg.LLM.OpenAI.Model, cfg.LLM.OpenAI.BaseURL), nil
	default:
		return nil, fmt.Errorf("unknown LLM backend: %q (supported: claude, openai)", backend)
	}
}
