package github

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	gh "github.com/google/go-github/v68/github"
	"github.com/yleaf/gh-active/pkg/model"
)

func makeEvent(typ string, createdAt time.Time, rawPayload any) *gh.Event {
	b, _ := json.Marshal(rawPayload)
	raw := json.RawMessage(b)
	repo := &gh.Repository{Name: gh.Ptr("owner/repo")}
	ts := gh.Timestamp{Time: createdAt}
	return &gh.Event{
		Type:       gh.Ptr(typ),
		RawPayload: &raw,
		Repo:       repo,
		CreatedAt:  &ts,
	}
}

func makePRPayload(action string, number int, title, htmlURL string, merged bool) *gh.PullRequestEvent {
	return &gh.PullRequestEvent{
		Action: gh.Ptr(action),
		PullRequest: &gh.PullRequest{
			Number:  gh.Ptr(number),
			Title:   gh.Ptr(title),
			HTMLURL: gh.Ptr(htmlURL),
			Merged:  gh.Ptr(merged),
		},
	}
}

// parsePushEventFromPayload is a test helper that creates a push activity
// from a pushPayload directly, simulating what the real parsePushEvent does
// but without needing the Compare API.
func parsePushFromPayload(p pushPayload, repo string, createdAt time.Time, commits []model.Commit, prCommitSHAs map[string]bool) *model.Activity {
	var filtered []model.Commit
	for _, c := range commits {
		if !prCommitSHAs[c.SHA] {
			filtered = append(filtered, c)
		}
	}
	if len(filtered) == 0 {
		return nil
	}

	branch := p.Ref
	if len(branch) > len("refs/heads/") {
		branch = branch[len("refs/heads/"):]
	}

	return &model.Activity{
		Type:      model.ActivityTypePush,
		Repo:      repo,
		Title:     fmt.Sprintf("%d commits to %s", len(filtered), branch),
		Commits:   filtered,
		CreatedAt: createdAt,
	}
}

func TestDedup_PushFiltersOutPRCommits(t *testing.T) {
	commits := []model.Commit{
		{SHA: "aaa111", Message: "fix bug"},
		{SHA: "bbb222", Message: "add feature"},
		{SHA: "ccc333", Message: "standalone commit"},
	}
	prSHAs := map[string]bool{"aaa111": true, "bbb222": true}

	a := parsePushFromPayload(
		pushPayload{Ref: "refs/heads/main", Head: "ccc333", Before: "000"},
		"owner/repo", time.Now(), commits, prSHAs,
	)

	if a == nil {
		t.Fatal("expected activity, got nil")
	}
	if len(a.Commits) != 1 {
		t.Fatalf("commits = %d, want 1", len(a.Commits))
	}
	if a.Commits[0].SHA != "ccc333" {
		t.Errorf("commit SHA = %q, want %q", a.Commits[0].SHA, "ccc333")
	}
}

func TestDedup_AllCommitsInPR_ReturnsNil(t *testing.T) {
	commits := []model.Commit{
		{SHA: "aaa111", Message: "fix bug"},
	}
	prSHAs := map[string]bool{"aaa111": true}

	a := parsePushFromPayload(
		pushPayload{Ref: "refs/heads/main", Head: "aaa111", Before: "000"},
		"owner/repo", time.Now(), commits, prSHAs,
	)

	if a != nil {
		t.Error("expected nil when all commits are in PRs")
	}
}

func TestDedup_NoPRCommits_KeepsAll(t *testing.T) {
	commits := []model.Commit{
		{SHA: "aaa111", Message: "fix bug"},
		{SHA: "bbb222", Message: "add feature"},
	}

	a := parsePushFromPayload(
		pushPayload{Ref: "refs/heads/main", Head: "bbb222", Before: "000"},
		"owner/repo", time.Now(), commits, map[string]bool{},
	)

	if a == nil {
		t.Fatal("expected activity, got nil")
	}
	if len(a.Commits) != 2 {
		t.Errorf("commits = %d, want 2", len(a.Commits))
	}
	if a.Title != "2 commits to main" {
		t.Errorf("title = %q, want %q", a.Title, "2 commits to main")
	}
}

func TestParsePREvent_IgnoresClosedNotMerged(t *testing.T) {
	client := &Client{}
	payload := makePRPayload("closed", 1, "some PR", "https://github.com/x/y/pull/1", false)
	e := makeEvent("PullRequestEvent", time.Now(), payload)

	a, shas, err := client.parsePREvent(nil, e)
	if err != nil {
		t.Fatal(err)
	}
	if a != nil {
		t.Error("expected nil for closed-not-merged PR")
	}
	if len(shas) != 0 {
		t.Error("expected no SHAs for closed-not-merged PR")
	}
}

func TestParsePREvent_Opened(t *testing.T) {
	client := &Client{}
	payload := makePRPayload("opened", 42, "Add login feature", "https://github.com/x/y/pull/42", false)
	e := makeEvent("PullRequestEvent", time.Now(), payload)

	a, _, err := client.parsePREvent(nil, e)
	if err != nil {
		t.Fatal(err)
	}
	if a == nil {
		t.Fatal("expected activity for opened PR")
	}
	if a.PRStatus != model.PRStatusOpened {
		t.Errorf("status = %q, want %q", a.PRStatus, model.PRStatusOpened)
	}
	if a.Title != "Add login feature" {
		t.Errorf("title = %q, want %q", a.Title, "Add login feature")
	}
}

func TestParsePREvent_ReviewRequested(t *testing.T) {
	client := &Client{}
	payload := makePRPayload("review_requested", 7, "Refactor auth", "https://github.com/x/y/pull/7", false)
	e := makeEvent("PullRequestEvent", time.Now(), payload)

	a, _, err := client.parsePREvent(nil, e)
	if err != nil {
		t.Fatal(err)
	}
	if a == nil {
		t.Fatal("expected activity for review_requested PR")
	}
	if a.PRStatus != model.PRStatusReview {
		t.Errorf("status = %q, want %q", a.PRStatus, model.PRStatusReview)
	}
}

func TestBranchPrefix_Stripped(t *testing.T) {
	commits := []model.Commit{
		{SHA: "xyz", Message: "test"},
	}
	a := parsePushFromPayload(
		pushPayload{Ref: "refs/heads/feature/auth", Head: "xyz", Before: "000"},
		"owner/repo", time.Now(), commits, map[string]bool{},
	)
	if a == nil {
		t.Fatal("expected activity")
	}
	if a.Title != "1 commits to feature/auth" {
		t.Errorf("title = %q, want branch prefix stripped", a.Title)
	}
}

func TestPRStatusPriority(t *testing.T) {
	if prStatusPriority(model.PRStatusMerged) <= prStatusPriority(model.PRStatusReview) {
		t.Error("merged should outrank review")
	}
	if prStatusPriority(model.PRStatusReview) <= prStatusPriority(model.PRStatusOpened) {
		t.Error("review should outrank opened")
	}
}

func TestPRDedup_SamePR_KeepsHighestStatus(t *testing.T) {
	now := time.Now()
	url := "https://github.com/x/y/pull/42"

	// Simulate: same PR fires opened then review_requested then merged
	opened := &model.Activity{
		Type: model.ActivityTypePR, Repo: "x/y", Title: "feat",
		URL: url, PRStatus: model.PRStatusOpened, CreatedAt: now.Add(-2 * time.Hour),
	}
	review := &model.Activity{
		Type: model.ActivityTypePR, Repo: "x/y", Title: "feat",
		URL: url, PRStatus: model.PRStatusReview, CreatedAt: now.Add(-1 * time.Hour),
	}
	merged := &model.Activity{
		Type: model.ActivityTypePR, Repo: "x/y", Title: "feat",
		URL: url, PRStatus: model.PRStatusMerged, CreatedAt: now,
		Commits: []model.Commit{{SHA: "abc"}},
	}

	// Feed them into the dedup map logic
	byURL := map[string]*model.Activity{}
	for _, a := range []*model.Activity{opened, review, merged} {
		existing, seen := byURL[a.URL]
		if !seen {
			cp := *a
			byURL[a.URL] = &cp
		} else if prStatusPriority(a.PRStatus) > prStatusPriority(existing.PRStatus) {
			if a.PRStatus == model.PRStatusMerged && len(a.Commits) > 0 {
				existing.Commits = a.Commits
			}
			existing.PRStatus = a.PRStatus
			existing.CreatedAt = a.CreatedAt
		}
	}

	if len(byURL) != 1 {
		t.Fatalf("expected 1 deduplicated PR, got %d", len(byURL))
	}
	result := byURL[url]
	if result.PRStatus != model.PRStatusMerged {
		t.Errorf("status = %q, want merged", result.PRStatus)
	}
	if len(result.Commits) != 1 || result.Commits[0].SHA != "abc" {
		t.Error("merged commits should be preserved")
	}
}
