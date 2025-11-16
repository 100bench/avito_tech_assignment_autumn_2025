package entities

type PullRequestShort struct {
	PullRequestID   string `json:"pull_request_id"`
	PullRequestName string `json:"pull_request_name"`
	AuthorID        string `json:"author_id"`
	Status          string `json:"status"`
}

func NewPullRequestShort(id string, name string, authorID string, status string) *PullRequestShort {
	return &PullRequestShort{
		PullRequestID:   id,
		PullRequestName: name,
		AuthorID:        authorID,
		Status:          status,
	}
}
