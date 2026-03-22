package llm

import (
	"strings"
	"testing"
	"time"

	"github.com/leavesster/gh-active/pkg/model"
)

func TestBuildPrompt_IncludesRepoAndOneSentenceRules(t *testing.T) {
	activities := []model.Activity{
		{
			Type:     model.ActivityTypePR,
			Repo:     "acme/api",
			Title:    "Add idempotency key support",
			URL:      "https://github.com/acme/api/pull/123",
			PRStatus: model.PRStatusMerged,
		},
		{
			Type:  model.ActivityTypePush,
			Repo:  "acme/web",
			Title: "2 commits to main",
			Commits: []model.Commit{
				{SHA: "abc1234def", Message: "fix: cache invalidation\n\nextra body"},
			},
		},
	}

	prompt := BuildPrompt(activities, SummarizeOpts{
		Language: "zh-CN",
		Username: "yleaf",
		Start:    time.Date(2026, 2, 16, 0, 0, 0, 0, time.UTC),
		End:      time.Date(2026, 2, 22, 23, 59, 59, 0, time.UTC),
	})

	for _, want := range []string{
		"Group work by repository (one section per repo)",
		"Each event should be summarized in exactly one sentence",
		"Prefer one bullet for one input event (keep near 1:1 mapping).",
		"If there are many events, merge only highly related events within the same repository into one sentence.",
		"[E001][PR][merged][acme/api]",
		"[E002][Push][acme/web]",
	} {
		if !strings.Contains(prompt, want) {
			t.Fatalf("prompt missing %q\nprompt:\n%s", want, prompt)
		}
	}
}

func TestBuildPrompt_NoActivity(t *testing.T) {
	prompt := BuildPrompt(nil, SummarizeOpts{
		Language: "en",
		Username: "yleaf",
		Start:    time.Date(2026, 2, 16, 0, 0, 0, 0, time.UTC),
		End:      time.Date(2026, 2, 22, 23, 59, 59, 0, time.UTC),
	})

	if !strings.Contains(prompt, "(No activity this week)") {
		t.Fatalf("prompt should include no-activity marker, got:\n%s", prompt)
	}
}
