package postgres

import (
	"context"
	"time"

	"github.com/jackc/pgx/v4"
	"github.com/pkg/errors"

	en "github.com/100bench/avito_tech_assignment_autumn_2025/internal/entities"
)

func (p *PgxStorage) CreatePRWithReviewers(ctx context.Context, pr *en.PullRequest, reviewerIDs []string) error {
	tx, err := p.pool.Begin(ctx)
	if err != nil {
		return errors.Wrap(err, "PgxStorage.CreatePRWithReviewers.BeginTx")
	}
	defer func() { _ = tx.Rollback(ctx) }()

	const qPR = `
		INSERT INTO pull_requests (pull_request_id, pull_request_name, author_id, status, created_at, merged_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`
	_, err = tx.Exec(ctx, qPR, pr.PullRequestID, pr.PullRequestName, pr.AuthorID, string(pr.Status), pr.CreatedAt, pr.MergedAt)
	if err != nil {
		return errors.Wrap(err, "PgxStorage.CreatePRWithReviewers.CreatePR")
	}

	const qReviewers = `INSERT INTO pr_reviewers (pull_request_id, user_id) VALUES ($1, $2)`
	for _, reviewerID := range reviewerIDs {
		_, err = tx.Exec(ctx, qReviewers, pr.PullRequestID, reviewerID)
		if err != nil {
			return errors.Wrap(err, "PgxStorage.CreatePRWithReviewers.AssignReviewer")
		}
	}

	if err = tx.Commit(ctx); err != nil {
		return errors.Wrap(err, "PgxStorage.CreatePRWithReviewers.Commit")
	}

	return nil
}

func (p *PgxStorage) GetPR(ctx context.Context, prID string) (*en.PullRequest, error) {
	const qPR = `
		SELECT pull_request_id, pull_request_name, author_id, status, created_at, merged_at
		FROM pull_requests
		WHERE pull_request_id = $1
	`
	var pr en.PullRequest
	var status string
	err := p.pool.QueryRow(ctx, qPR, prID).Scan(
		&pr.PullRequestID, &pr.PullRequestName, &pr.AuthorID, &status, &pr.CreatedAt, &pr.MergedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, errors.Wrap(err, "PgxStorage.GetPR")
	}
	pr.Status = en.PRStatus(status)

	const qReviewers = `SELECT user_id FROM pr_reviewers WHERE pull_request_id = $1`
	rows, err := p.pool.Query(ctx, qReviewers, prID)
	if err != nil {
		return nil, errors.Wrap(err, "PgxStorage.GetPR.GetReviewers")
	}
	defer rows.Close()

	var reviewers []string
	for rows.Next() {
		var userID string
		if err := rows.Scan(&userID); err != nil {
			return nil, errors.Wrap(err, "PgxStorage.GetPR.ScanReviewer")
		}
		reviewers = append(reviewers, userID)
	}
	if rows.Err() != nil {
		return nil, errors.Wrap(rows.Err(), "PgxStorage.GetPR.RowsError")
	}

	pr.AssignedReviewers = reviewers
	return &pr, nil
}

func (p *PgxStorage) MergePR(ctx context.Context, prID string, mergedAt time.Time) (*en.PullRequest, error) {
	const q = `
		UPDATE pull_requests
		SET status = $2, merged_at = $3
		WHERE pull_request_id = $1
		RETURNING pull_request_id, pull_request_name, author_id, status, created_at, merged_at
	`
	var pr en.PullRequest
	var status string
	err := p.pool.QueryRow(ctx, q, prID, string(en.StatusMerged), mergedAt).Scan(
		&pr.PullRequestID, &pr.PullRequestName, &pr.AuthorID, &status, &pr.CreatedAt, &pr.MergedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, errors.Wrap(err, "PgxStorage.MergePR")
	}
	pr.Status = en.PRStatus(status)

	const qReviewers = `SELECT user_id FROM pr_reviewers WHERE pull_request_id = $1`
	rows, err := p.pool.Query(ctx, qReviewers, prID)
	if err != nil {
		return nil, errors.Wrap(err, "PgxStorage.MergePR.GetReviewers")
	}
	defer rows.Close()

	var reviewers []string
	for rows.Next() {
		var userID string
		if err := rows.Scan(&userID); err != nil {
			return nil, errors.Wrap(err, "PgxStorage.MergePR.ScanReviewer")
		}
		reviewers = append(reviewers, userID)
	}
	pr.AssignedReviewers = reviewers

	return &pr, nil
}

func (p *PgxStorage) PRExists(ctx context.Context, prID string) (bool, error) {
	const q = `SELECT EXISTS(SELECT 1 FROM pull_requests WHERE pull_request_id = $1)`
	var exists bool
	err := p.pool.QueryRow(ctx, q, prID).Scan(&exists)
	if err != nil {
		return false, errors.Wrap(err, "PgxStorage.PRExists")
	}
	return exists, nil
}
