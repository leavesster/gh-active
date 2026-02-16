package report

import (
	"fmt"
	"strings"

	"github.com/leavesster/gh-active/pkg/model"
)

func GenerateMarkdown(r *model.WeeklyReport) string {
	var b strings.Builder

	b.WriteString(fmt.Sprintf("# Weekly Report: %s\n", r.Username))
	b.WriteString(fmt.Sprintf("**%s ~ %s**\n\n",
		r.StartDate.Format("2006-01-02"),
		r.EndDate.Format("2006-01-02")))

	if r.Summary != "" {
		b.WriteString("## Summary\n\n")
		b.WriteString(r.Summary)
		b.WriteString("\n\n")
	}

	completed, inProgress := splitActivities(r.Activities)

	if len(completed) > 0 {
		b.WriteString("## Completed\n\n")
		writeActivities(&b, completed)
	}

	if len(inProgress) > 0 {
		b.WriteString("## In Progress\n\n")
		writeActivities(&b, inProgress)
	}

	if len(completed) == 0 && len(inProgress) == 0 {
		b.WriteString("*No activity this week.*\n")
	}

	// stats
	b.WriteString("\n---\n")
	prCount, commitCount := countStats(r.Activities)
	b.WriteString(fmt.Sprintf("PRs: %d | Commits: %d\n", prCount, commitCount))

	return b.String()
}

func splitActivities(activities []model.Activity) (completed, inProgress []model.Activity) {
	for _, a := range activities {
		if a.Type == model.ActivityTypePR && (a.PRStatus == model.PRStatusOpened || a.PRStatus == model.PRStatusReview) {
			inProgress = append(inProgress, a)
		} else {
			completed = append(completed, a)
		}
	}
	return
}

func writeActivities(b *strings.Builder, activities []model.Activity) {
	// group by repo
	byRepo := make(map[string][]model.Activity)
	var repoOrder []string
	for _, a := range activities {
		if _, seen := byRepo[a.Repo]; !seen {
			repoOrder = append(repoOrder, a.Repo)
		}
		byRepo[a.Repo] = append(byRepo[a.Repo], a)
	}

	for _, repo := range repoOrder {
		b.WriteString(fmt.Sprintf("### %s\n\n", repo))
		for _, a := range byRepo[repo] {
			writeActivity(b, a)
		}
	}
}

func writeActivity(b *strings.Builder, a model.Activity) {
	switch a.Type {
	case model.ActivityTypePR:
		status := ""
		switch a.PRStatus {
		case model.PRStatusMerged:
			status = "merged"
		case model.PRStatusOpened:
			status = "opened"
		case model.PRStatusReview:
			status = "in review"
		}
		if a.URL != "" {
			b.WriteString(fmt.Sprintf("- **PR** [%s](%s) (%s)\n", a.Title, a.URL, status))
		} else {
			b.WriteString(fmt.Sprintf("- **PR** %s (%s)\n", a.Title, status))
		}
	case model.ActivityTypePush:
		b.WriteString(fmt.Sprintf("- %s\n", a.Title))
		for _, c := range a.Commits {
			msg := firstLine(c.Message)
			b.WriteString(fmt.Sprintf("  - `%.7s` %s\n", c.SHA, msg))
		}
	}
}

func firstLine(s string) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return s[:i]
	}
	return s
}

func countStats(activities []model.Activity) (prs, commits int) {
	for _, a := range activities {
		switch a.Type {
		case model.ActivityTypePR:
			prs++
		case model.ActivityTypePush:
			commits += len(a.Commits)
		}
	}
	return
}
