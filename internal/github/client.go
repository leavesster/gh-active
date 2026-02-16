package github

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	gh "github.com/google/go-github/v68/github"
	"github.com/yleaf/gh-active/pkg/model"
)

type Client struct {
	gh *gh.Client
}

func NewClient(token string) *Client {
	return &Client{
		gh: gh.NewClient(nil).WithAuthToken(token),
	}
}

// FetchEvents retrieves all user events within the given time range.
// Events API returns max 300 events (10 pages x 30 per page).
func (c *Client) FetchEvents(ctx context.Context, user string, start, end time.Time) ([]*gh.Event, error) {
	var all []*gh.Event
	opts := &gh.ListOptions{PerPage: 30, Page: 1}

	for opts.Page <= 10 {
		events, resp, err := c.gh.Activity.ListEventsPerformedByUser(ctx, user, false, opts)
		if err != nil {
			return nil, fmt.Errorf("fetch events page %d: %w", opts.Page, err)
		}

		for _, e := range events {
			t := e.GetCreatedAt().Time
			if t.Before(start) {
				return all, nil
			}
			if t.After(end) {
				continue
			}
			all = append(all, e)
		}

		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}

	return all, nil
}

// FetchPRCommits returns commit SHAs for a given pull request.
func (c *Client) FetchPRCommits(ctx context.Context, owner, repo string, prNumber int) ([]string, error) {
	var shas []string
	opts := &gh.ListOptions{PerPage: 100}

	for {
		commits, resp, err := c.gh.PullRequests.ListCommits(ctx, owner, repo, prNumber, opts)
		if err != nil {
			return nil, fmt.Errorf("fetch PR #%d commits: %w", prNumber, err)
		}
		for _, c := range commits {
			shas = append(shas, c.GetSHA())
		}
		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}

	return shas, nil
}

// FetchCompareCommits gets commits between two SHAs using the compare API.
func (c *Client) FetchCompareCommits(ctx context.Context, owner, repo, base, head string) ([]model.Commit, error) {
	comp, _, err := c.gh.Repositories.CompareCommits(ctx, owner, repo, base, head, &gh.ListOptions{PerPage: 100})
	if err != nil {
		return nil, fmt.Errorf("compare %s...%s: %w", base[:7], head[:7], err)
	}

	var commits []model.Commit
	for _, rc := range comp.Commits {
		msg := ""
		if rc.Commit != nil {
			msg = rc.Commit.GetMessage()
		}
		commits = append(commits, model.Commit{
			SHA:     rc.GetSHA(),
			Message: msg,
		})
	}
	return commits, nil
}

// pushPayload extracts fields from Events API PushEvent raw payload,
// which only contains ref/head/before (no commits array).
type pushPayload struct {
	Ref    string `json:"ref"`
	Head   string `json:"head"`
	Before string `json:"before"`
}

// ParseEvents converts raw GitHub events into deduplicated Activities.
// Two-pass approach: first collect PR commit SHAs, then filter pushes.
func (c *Client) ParseEvents(ctx context.Context, events []*gh.Event) ([]model.Activity, error) {
	prCommitSHAs := make(map[string]bool)
	var prActivities []model.Activity
	var pushEvents []*gh.Event

	// Pass 1: process PullRequestEvents
	for _, e := range events {
		switch e.GetType() {
		case "PullRequestEvent":
			a, shas, err := c.parsePREvent(ctx, e)
			if a == nil {
				continue
			}
			if err != nil {
				return nil, err
			}
			prActivities = append(prActivities, *a)
			for _, sha := range shas {
				prCommitSHAs[sha] = true
			}
		case "PushEvent":
			pushEvents = append(pushEvents, e)
		}
	}

	// Pass 2: process PushEvents, skip commits already in PRs
	var pushActivities []model.Activity
	for _, e := range pushEvents {
		a := c.parsePushEvent(ctx, e, prCommitSHAs)
		if a != nil {
			pushActivities = append(pushActivities, *a)
		}
	}

	all := append(prActivities, pushActivities...)
	return all, nil
}

func (c *Client) parsePREvent(ctx context.Context, e *gh.Event) (*model.Activity, []string, error) {
	payload, err := e.ParsePayload()
	if err != nil {
		return nil, nil, fmt.Errorf("parse PR payload: %w", err)
	}
	pr, ok := payload.(*gh.PullRequestEvent)
	if !ok {
		return nil, nil, nil
	}

	action := pr.GetAction()
	pull := pr.GetPullRequest()

	var status model.PRStatus
	switch {
	case action == "closed" && pull.GetMerged():
		status = model.PRStatusMerged
	case action == "opened":
		status = model.PRStatusOpened
	case action == "review_requested":
		status = model.PRStatusReview
	case action == "closed" && !pull.GetMerged():
		return nil, nil, nil
	default:
		return nil, nil, nil
	}

	repo := e.GetRepo().GetName()
	activity := &model.Activity{
		Type:      model.ActivityTypePR,
		Repo:      repo,
		Title:     pull.GetTitle(),
		URL:       pull.GetHTMLURL(),
		PRStatus:  status,
		CreatedAt: e.GetCreatedAt().Time,
	}

	// For merged PRs, fetch commits for dedup
	var shas []string
	if status == model.PRStatusMerged {
		parts := strings.SplitN(repo, "/", 2)
		if len(parts) == 2 {
			shas, err = c.FetchPRCommits(ctx, parts[0], parts[1], pull.GetNumber())
			if err != nil {
				return nil, nil, err
			}
			for _, sha := range shas {
				activity.Commits = append(activity.Commits, model.Commit{SHA: sha})
			}
		}
	}

	return activity, shas, nil
}

// parsePushEvent handles Events API PushEvent which only has before/head SHAs.
// It uses the Compare API to get the actual commits.
func (c *Client) parsePushEvent(ctx context.Context, e *gh.Event, prCommitSHAs map[string]bool) *model.Activity {
	raw := e.GetRawPayload()
	var p pushPayload
	if err := json.Unmarshal(raw, &p); err != nil {
		return nil
	}

	if p.Head == "" || p.Before == "" {
		return nil
	}

	repo := e.GetRepo().GetName()
	parts := strings.SplitN(repo, "/", 2)
	if len(parts) != 2 {
		return nil
	}

	commits, err := c.FetchCompareCommits(ctx, parts[0], parts[1], p.Before, p.Head)
	if err != nil {
		// Compare may fail for force pushes or deleted branches — skip silently
		return nil
	}

	// Filter out PR commits
	var filtered []model.Commit
	for _, commit := range commits {
		if !prCommitSHAs[commit.SHA] {
			filtered = append(filtered, commit)
		}
	}

	if len(filtered) == 0 {
		return nil
	}

	branch := strings.TrimPrefix(p.Ref, "refs/heads/")

	return &model.Activity{
		Type:      model.ActivityTypePush,
		Repo:      repo,
		Title:     fmt.Sprintf("%d commits to %s", len(filtered), branch),
		Commits:   filtered,
		CreatedAt: e.GetCreatedAt().Time,
	}
}
