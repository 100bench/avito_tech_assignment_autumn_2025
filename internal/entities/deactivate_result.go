package entities

// DeactivateResult результат массовой деактивации
type DeactivateResult struct {
	DeactivatedUsers []string             `json:"deactivated_users"`
	Reassignments    []PRReassignmentInfo `json:"reassigned_prs"`
}
