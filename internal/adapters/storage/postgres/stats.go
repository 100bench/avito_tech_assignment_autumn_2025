package postgres

import (
	"context"

	"github.com/pkg/errors"

	en "github.com/100bench/avito_tech_assignment_autumn_2025/internal/entities"
)

func (p *PgxStorage) GetStats(ctx context.Context) (*en.Stats, error) {
	const qAssignments = `
		SELECT user_id, COUNT(*) as count
		FROM pr_reviewers
		GROUP BY user_id
	`
	rows, err := p.pool.Query(ctx, qAssignments)
	if err != nil {
		return nil, errors.Wrap(err, "PgxStorage.GetStats.QueryAssignments")
	}
	defer rows.Close()

	userAssignments := make(map[string]int)
	for rows.Next() {
		var userID string
		var count int
		if err := rows.Scan(&userID, &count); err != nil {
			return nil, errors.Wrap(err, "PgxStorage.GetStats.ScanAssignment")
		}
		userAssignments[userID] = count
	}
	if rows.Err() != nil {
		return nil, errors.Wrap(rows.Err(), "PgxStorage.GetStats.AssignmentsRowsError")
	}

	const qPRStats = `
		SELECT status, COUNT(*) as count
		FROM pull_requests
		WHERE status IN ('OPEN', 'MERGED')
		GROUP BY status
	`
	prRows, err := p.pool.Query(ctx, qPRStats)
	if err != nil {
		return nil, errors.Wrap(err, "PgxStorage.GetStats.QueryPRStats")
	}
	defer prRows.Close()

	var openCount, mergedCount int
	for prRows.Next() {
		var status string
		var count int
		if err := prRows.Scan(&status, &count); err != nil {
			return nil, errors.Wrap(err, "PgxStorage.GetStats.ScanPRStat")
		}
		if status == "OPEN" {
			openCount = count
		} else if status == "MERGED" {
			mergedCount = count
		}
	}
	if prRows.Err() != nil {
		return nil, errors.Wrap(prRows.Err(), "PgxStorage.GetStats.PRStatsRowsError")
	}

	return &en.Stats{
		UserAssignments: userAssignments,
		PRStats: en.PRStats{
			Open:   openCount,
			Merged: mergedCount,
		},
	}, nil
}
