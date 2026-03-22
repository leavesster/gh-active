package llm

import (
	"fmt"
	"strings"
	"time"

	"github.com/leavesster/gh-active/pkg/model"
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
2. Group work by repository (one section per repo)
3. Under each repository, list events as bullet points
4. Each event should be summarized in exactly one sentence
5. Highlight impact, not just activity

Output format:
- ## Summary
- <one paragraph>
- ## By Repository
- ### owner/repo
- - <one sentence for one event>

Rules:
- Prefer one bullet for one input event (keep near 1:1 mapping).
- Do not mix multiple repositories in one bullet.
- Do not merge unrelated events into one sentence.
- If there are many events, merge only highly related events within the same repository into one sentence.
- Keep language concise and concrete.
- Do not include a top-level title heading.

Activities:

`,
		opts.Username,
		opts.Start.Format("2006-01-02"),
		opts.End.Format("2006-01-02"),
		langName(opts.Language),
	))

	for i, a := range activities {
		eventID := fmt.Sprintf("E%03d", i+1)
		switch a.Type {
		case model.ActivityTypePR:
			b.WriteString(fmt.Sprintf("[%s][PR][%s][%s] %s", eventID, a.PRStatus, a.Repo, a.Title))
			if a.URL != "" {
				b.WriteString(fmt.Sprintf(" (%s)", a.URL))
			}
			b.WriteByte('\n')
		case model.ActivityTypePush:
			b.WriteString(fmt.Sprintf("[%s][Push][%s] %s\n", eventID, a.Repo, a.Title))
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

	b.WriteString("\nOutput the report in Markdown.")
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
