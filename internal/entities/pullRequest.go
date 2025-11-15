package entities

import "time"

type PRStatus string

const (
    StatusOpen   PRStatus = "OPEN"
    StatusMerged PRStatus = "MERGED"
)

type PullRequest struct {
	PullRequestID     string     `json:"pull_request_id"`
	PullRequestName   string     `json:"pull_request_name"`
	AuthorID          string     `json:"author_id"`
	Status            PRStatus   `json:"status"`
	AssignedReviewers []string   `json:"assigned_reviewers"`
	CreatedAt         time.Time  `json:"created_at,omitempty"`
	MergedAt          *time.Time `json:"merged_at,omitempty"`
}

func NewPullRequest(id string, name string, authorID string, status PRStatus, reviewers []string, createdAt time.Time, mergedAt *time.Time) *PullRequest {
	return &PullRequest{
		PullRequestID:     id,
		PullRequestName:   name,
		AuthorID:          authorID,
		Status:            status,
		AssignedReviewers: reviewers,
		CreatedAt:         createdAt,
		MergedAt:          mergedAt,
	}
}