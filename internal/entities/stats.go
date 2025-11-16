package entities

type Stats struct {
	UserAssignments map[string]int `json:"user_assignments"`
	PRStats         PRStats        `json:"pr_stats"`
}

type PRStats struct {
	Open   int `json:"open"`
	Merged int `json:"merged"`
}
