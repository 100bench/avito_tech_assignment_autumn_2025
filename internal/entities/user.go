package entities

import "time"

type User struct {
	UserID    string    `json:"user_id"`
    Username  string    `json:"username"`
    TeamName  string    `json:"team_name"`
    IsActive  bool      `json:"is_active"`
    CreatedAt time.Time `json:"created_at,omitempty"`
    UpdatedAt time.Time `json:"updated_at,omitempty"`
}

func NewUser(id string, name string, team string, active bool, createdAt time.Time, updatedAt time.Time) *User {
	return &User{
		UserID:   id,
		Username: name,
		TeamName: team,
		IsActive: active,
		CreatedAt: createdAt,
		UpdatedAt: updatedAt,
	}
}
