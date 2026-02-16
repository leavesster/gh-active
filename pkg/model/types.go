package model

import "time"

type ActivityType string

const (
	ActivityTypePR   ActivityType = "pr"
	ActivityTypePush ActivityType = "push"
)

type PRStatus string

const (
	PRStatusOpened PRStatus = "opened"
	PRStatusReview PRStatus = "review"
	PRStatusMerged PRStatus = "merged"
	PRStatusClosed PRStatus = "closed"
)

type Commit struct {
	SHA     string
	Message string
}

type Activity struct {
	Type      ActivityType
	Repo      string // "owner/repo"
	Title     string
	URL       string
	PRStatus  PRStatus // only valid for ActivityTypePR
	Commits   []Commit
	CreatedAt time.Time
}

type WeeklyReport struct {
	Username   string
	StartDate  time.Time
	EndDate    time.Time
	Summary    string // LLM-generated
	Activities []Activity
}
