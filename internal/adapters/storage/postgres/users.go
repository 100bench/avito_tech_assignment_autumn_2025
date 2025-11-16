package postgres

import (
	"context"

	"github.com/jackc/pgx/v4"
	"github.com/pkg/errors"

	en "github.com/100bench/avito_tech_assignment_autumn_2025/internal/entities"
)

func (p *PgxStorage) GetUser(ctx context.Context, userID string) (*en.User, error) {
	const q = `
		SELECT user_id, username, team_name, is_active, created_at, updated_at
		FROM users
		WHERE user_id = $1
	`
	var user en.User
	err := p.pool.QueryRow(ctx, q, userID).Scan(
		&user.UserID, &user.Username, &user.TeamName, &user.IsActive, &user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, errors.Wrap(err, "PgxStorage.GetUser")
	}
	return &user, nil
}

func (p *PgxStorage) GetUsersByTeam(ctx context.Context, teamName string, activeOnly bool) ([]*en.User, error) {
	var q string
	if activeOnly {
		q = `
			SELECT user_id, username, team_name, is_active, created_at, updated_at
			FROM users
			WHERE team_name = $1 AND is_active = true
			ORDER BY user_id
		`
	} else {
		q = `
			SELECT user_id, username, team_name, is_active, created_at, updated_at
			FROM users
			WHERE team_name = $1
			ORDER BY user_id
		`
	}

	rows, err := p.pool.Query(ctx, q, teamName)

	if err != nil {
		return nil, errors.Wrap(err, "PgxStorage.GetUsersByTeam")
	}
	defer rows.Close()

	var users []*en.User
	for rows.Next() {
		var user en.User
		if err := rows.Scan(
			&user.UserID, &user.Username, &user.TeamName, &user.IsActive, &user.CreatedAt, &user.UpdatedAt,
		); err != nil {
			return nil, errors.Wrap(err, "PgxStorage.GetUsersByTeam.Scan")
		}
		users = append(users, &user)
	}
	if rows.Err() != nil {
		return nil, errors.Wrap(rows.Err(), "PgxStorage.GetUsersByTeam.RowsError")
	}

	return users, nil
}

func (p *PgxStorage) SetUserActiveStatus(ctx context.Context, userID string, isActive bool) (*en.User, error) {
	const q = `
		UPDATE users
		SET is_active = $2, updated_at = NOW()
		WHERE user_id = $1
		RETURNING user_id, username, team_name, is_active, created_at, updated_at
	`
	var user en.User
	err := p.pool.QueryRow(ctx, q, userID, isActive).Scan(
		&user.UserID, &user.Username, &user.TeamName, &user.IsActive, &user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, errors.Wrap(err, "PgxStorage.SetUserActiveStatus")
	}
	return &user, nil
}
