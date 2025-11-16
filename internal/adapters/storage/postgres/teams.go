package postgres

import (
	"context"

	"github.com/jackc/pgx/v4"
	"github.com/pkg/errors"

	en "github.com/100bench/avito_tech_assignment_autumn_2025/internal/entities"
)

func (p *PgxStorage) CreateTeamWithUsers(ctx context.Context, teamName string, users []*en.User) error {
	tx, err := p.pool.Begin(ctx)
	if err != nil {
		return errors.Wrap(err, "PgxStorage.CreateTeamWithUsers.BeginTx")
	}
	defer func() { _ = tx.Rollback(ctx) }()

	const qTeam = `INSERT INTO teams (team_name) VALUES ($1)`
	_, err = tx.Exec(ctx, qTeam, teamName)
	if err != nil {
		return errors.Wrap(err, "PgxStorage.CreateTeamWithUsers.CreateTeam")
	}

	const qUser = `
		INSERT INTO users (user_id, username, team_name, is_active)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (user_id)
		DO UPDATE SET
			team_name = EXCLUDED.team_name,
			is_active = EXCLUDED.is_active,
			updated_at = NOW()
	`
	for _, user := range users {
		_, err = tx.Exec(ctx, qUser, user.UserID, user.Username, teamName, user.IsActive)
		if err != nil {
			return errors.Wrap(err, "PgxStorage.CreateTeamWithUsers.UpsertUser")
		}
	}

	if err = tx.Commit(ctx); err != nil {
		return errors.Wrap(err, "PgxStorage.CreateTeamWithUsers.Commit")
	}

	return nil
}

func (p *PgxStorage) GetTeamByName(ctx context.Context, teamName string) (*en.Team, error) {
	const qTeam = `SELECT team_name FROM teams WHERE team_name = $1`
	var tn string
	err := p.pool.QueryRow(ctx, qTeam, teamName).Scan(&tn)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, errors.Wrap(err, "PgxStorage.GetTeamByName")
	}

	const qUsers = `
		SELECT user_id, username, is_active
		FROM users
		WHERE team_name = $1
		ORDER BY user_id
	`
	rows, err := p.pool.Query(ctx, qUsers, teamName)
	if err != nil {
		return nil, errors.Wrap(err, "PgxStorage.GetTeamByName.GetMembers")
	}
	defer rows.Close()

	var members []en.TeamMember
	for rows.Next() {
		var member en.TeamMember
		if err := rows.Scan(&member.UserID, &member.Username, &member.IsActive); err != nil {
			return nil, errors.Wrap(err, "PgxStorage.GetTeamByName.Scan")
		}
		members = append(members, member)
	}
	if rows.Err() != nil {
		return nil, errors.Wrap(rows.Err(), "PgxStorage.GetTeamByName.RowsError")
	}

	return &en.Team{TeamName: teamName, TeamMembers: members}, nil
}

func (p *PgxStorage) TeamExists(ctx context.Context, teamName string) (bool, error) {
	const q = `SELECT EXISTS(SELECT 1 FROM teams WHERE team_name = $1)`
	var exists bool
	err := p.pool.QueryRow(ctx, q, teamName).Scan(&exists)
	if err != nil {
		return false, errors.Wrap(err, "PgxStorage.TeamExists")
	}
	return exists, nil
}
