package entities

// PRReassignmentInfo информация о переназначении ревьювера
type PRReassignmentInfo struct {
	PullRequestID string `json:"pull_request_id"`
	OldReviewer   string `json:"old_reviewer"`
	NewReviewer   string `json:"new_reviewer"` // пустая строка если удален без замены
}
