package entities

type Team struct {
	TeamName    string       `json:"team_name"`
	TeamMembers []TeamMember `json:"members"`
}

func NewTeam(name string, members []TeamMember) *Team {
	return &Team{
		TeamName:    name,
		TeamMembers: members,
	}
}
