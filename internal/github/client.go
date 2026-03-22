package github

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	gh "github.com/google/go-github/v68/github"
	"github.com/leavesster/gh-active/pkg/model"
)

type Client struct {
	gh *gh.Client
}

func NewClient(token string) *Client {
	return &Client{
		gh: gh.NewClient(nil).WithAuthToken(token),
	}
}

// AuthenticatedUser returns the login of the token owner.
func (c *Client) AuthenticatedUser(ctx context.Context) (string, error) {
	user, _, err := c.gh.Users.Get(ctx, "")
	if err != nil {
		return "", fmt.Errorf("get authenticated user: %w", err)
	}
	return user.GetLogin(), nil
}

// FetchEvents retrieves all user events within the given time range.
// GitHub limits this resource to 10 pages. We use 100 items per page to maximize coverage.
// The returned bool reports whether fetching stopped at the pagination cap.
func (c *Client) FetchEvents(ctx context.Context, user string, start, end time.Time) ([]*gh.Event, bool, error) {
	var all []*gh.Event
	opts := &gh.ListOptions{PerPage: 100, Page: 1}

	for opts.Page <= 10 {
		events, resp, err := c.gh.Activity.ListEventsPerformedByUser(ctx, user, false, opts)
		if err != nil {
			return nil, false, fmt.Errorf("fetch events page %d: %w", opts.Page, err)
		}

		all = append(all, filterEventsInRange(events, start, end)...)

		if resp.NextPage == 0 {
			break
		}
		if opts.Page == 10 {
			return all, true, nil
		}
		opts.Page = resp.NextPage
	}

	return all, false, nil
}

// FetchPRCommits returns commit SHAs for a given pull request.
func (c *Client) FetchPRCommits(ctx context.Context, owner, repo string, prNumber int) ([]string, error) {
	if c == nil || c.gh == nil {
		return nil, nil
	}

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

// FetchPR returns full pull request details for a given PR number.
func (c *Client) FetchPR(ctx context.Context, owner, repo string, prNumber int) (*gh.PullRequest, error) {
	if c == nil || c.gh == nil {
		return nil, nil
	}

	pr, _, err := c.gh.PullRequests.Get(ctx, owner, repo, prNumber)
	if err != nil {
		return nil, fmt.Errorf("fetch PR #%d: %w", prNumber, err)
	}
	return pr, nil
}

// FetchCompareCommits gets commits between two SHAs using the compare API.
func (c *Client) FetchCompareCommits(ctx context.Context, owner, repo, base, head string) ([]model.Commit, error) {
	if c == nil || c.gh == nil {
		return nil, nil
	}

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

func splitRepoName(repo string) (owner, name string, ok bool) {
	parts := strings.SplitN(repo, "/", 2)
	if len(parts) != 2 {
		return "", "", false
	}
	return parts[0], parts[1], true
}

func filterEventsInRange(events []*gh.Event, start, end time.Time) []*gh.Event {
	var filtered []*gh.Event
	for _, e := range events {
		t := e.GetCreatedAt().Time
		if t.Before(start) || t.After(end) {
			continue
		}
		filtered = append(filtered, e)
	}
	return filtered
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
// Same PR appearing multiple times is deduplicated by keeping highest-priority status.
func (c *Client) ParseEvents(ctx context.Context, events []*gh.Event) ([]model.Activity, error) {
	prCommitSHAs := make(map[string]bool)
	prByURL := make(map[string]*model.Activity) // dedup PRs by URL
	var prOrder []string                        // preserve first-seen order
	var pushEvents []*gh.Event

	// Pass 1: process PullRequestEvents, dedup by PR URL
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
			for _, sha := range shas {
				prCommitSHAs[sha] = true
			}

			existing, seen := prByURL[a.URL]
			if !seen {
				prByURL[a.URL] = a
				prOrder = append(prOrder, a.URL)
			} else if prStatusPriority(a.PRStatus) > prStatusPriority(existing.PRStatus) {
				// keep commits from merged version
				if a.PRStatus == model.PRStatusMerged && len(a.Commits) > 0 {
					existing.Commits = a.Commits
				}
				existing.PRStatus = a.PRStatus
				existing.CreatedAt = a.CreatedAt
			}
		case "PushEvent":
			pushEvents = append(pushEvents, e)
		}
	}

	// Collect deduplicated PR activities in order
	var prActivities []model.Activity
	for _, url := range prOrder {
		prActivities = append(prActivities, *prByURL[url])
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

// prStatusPriority returns priority for dedup: higher wins.
// merged > review > opened
func prStatusPriority(s model.PRStatus) int {
	switch s {
	case model.PRStatusMerged:
		return 3
	case model.PRStatusReview:
		return 2
	case model.PRStatusOpened:
		return 1
	default:
		return 0
	}
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
	number := pull.GetNumber()
	title := pull.GetTitle()
	htmlURL := pull.GetHTMLURL()
	merged := pull.GetMerged()
	apiURL := pull.GetURL()
	repo := e.GetRepo().GetName()
	owner, repoName, repoOK := splitRepoName(repo)

	// Performed-events payloads often include only a partial pull_request object.
	// Fill in missing user-facing metadata from the repo/number we already have.
	if number > 0 && repoOK && c != nil && c.gh != nil && (title == "" || htmlURL == "" || (action == "closed" && !merged)) {
		fullPR, err := c.FetchPR(ctx, owner, repoName, number)
		if err != nil {
			return nil, nil, err
		}
		if fullPR != nil {
			if title == "" {
				title = fullPR.GetTitle()
			}
			if htmlURL == "" {
				htmlURL = fullPR.GetHTMLURL()
			}
			if !merged {
				merged = fullPR.GetMerged()
			}
		}
	}

	if htmlURL == "" && repoOK && number > 0 {
		htmlURL = fmt.Sprintf("https://github.com/%s/%s/pull/%d", owner, repoName, number)
	}
	if title == "" && number > 0 {
		title = fmt.Sprintf("PR #%d", number)
	}
	if htmlURL == "" {
		htmlURL = apiURL
	}

	var status model.PRStatus
	switch {
	case action == "merged":
		status = model.PRStatusMerged
	case action == "closed" && merged:
		status = model.PRStatusMerged
	case action == "opened":
		status = model.PRStatusOpened
	case action == "review_requested":
		status = model.PRStatusReview
	case action == "closed" && !merged:
		return nil, nil, nil
	default:
		return nil, nil, nil
	}

	activity := &model.Activity{
		Type:      model.ActivityTypePR,
		Repo:      repo,
		Title:     title,
		URL:       htmlURL,
		PRStatus:  status,
		CreatedAt: e.GetCreatedAt().Time,
	}

	// For merged PRs, fetch commits for dedup
	var shas []string
	if status == model.PRStatusMerged && repoOK && number > 0 && c != nil && c.gh != nil {
		shas, err = c.FetchPRCommits(ctx, owner, repoName, number)
		if err != nil {
			return nil, nil, err
		}
		for _, sha := range shas {
			activity.Commits = append(activity.Commits, model.Commit{SHA: sha})
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
