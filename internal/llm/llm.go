package llm

import (
	"fmt"
	"strings"
	"time"

	"github.com/yleaf/gh-active/pkg/model"
)

type LLM interface {
	Summarize(activities []model.Activity, opts SummarizeOpts) (string, error)
}

type SummarizeOpts struct {
	Language string
	Username string
	Start    time.Time
	End      time.Time
}

func BuildPrompt(activities []model.Activity, opts SummarizeOpts) string {
	var b strings.Builder

	b.WriteString(fmt.Sprintf(`You are a technical writer. Generate a concise weekly report for GitHub user "%s" (%s to %s).
The report is for team leads / managers. Write in %s.

Structure:
1. One-paragraph executive summary of key achievements
2. Group work by theme/project, not chronologically
3. Highlight impact, not just activity

Activities:

`,
		opts.Username,
		opts.Start.Format("2006-01-02"),
		opts.End.Format("2006-01-02"),
		langName(opts.Language),
	))

	for _, a := range activities {
		switch a.Type {
		case model.ActivityTypePR:
			b.WriteString(fmt.Sprintf("[PR][%s][%s] %s", a.PRStatus, a.Repo, a.Title))
			if a.URL != "" {
				b.WriteString(fmt.Sprintf(" (%s)", a.URL))
			}
			b.WriteByte('\n')
		case model.ActivityTypePush:
			b.WriteString(fmt.Sprintf("[Push][%s] %s\n", a.Repo, a.Title))
			for _, c := range a.Commits {
				msg := c.Message
				if i := strings.IndexByte(msg, '\n'); i >= 0 {
					msg = msg[:i]
				}
				b.WriteString(fmt.Sprintf("  - %.7s %s\n", c.SHA, msg))
			}
		}
	}

	if len(activities) == 0 {
		b.WriteString("(No activity this week)\n")
	}

	b.WriteString("\nOutput the report in Markdown. Do not include a title heading — it will be added separately.")
	return b.String()
}

func langName(code string) string {
	switch code {
	case "zh-CN", "zh":
		return "Chinese (Simplified)"
	case "en", "en-US":
		return "English"
	case "ja":
		return "Japanese"
	default:
		return code
	}
}
