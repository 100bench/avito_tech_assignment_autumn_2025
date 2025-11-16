package postgres

import (
	"context"

	"github.com/jackc/pgx/v4"
	"github.com/pkg/errors"

	en "github.com/100bench/avito_tech_assignment_autumn_2025/internal/entities"
)

func (p *PgxStorage) ReassignReviewer(ctx context.Context, prID string, oldUserID string, newUserID string) error {
	tx, err := p.pool.Begin(ctx)
	if err != nil {
		return errors.Wrap(err, "PgxStorage.ReassignReviewer.BeginTx")
	}
	defer func() { _ = tx.Rollback(ctx) }()

	const qLock = `SELECT pull_request_id FROM pull_requests WHERE pull_request_id = $1 FOR UPDATE`
	var lockedPRID string
	err = tx.QueryRow(ctx, qLock, prID).Scan(&lockedPRID)
	if err != nil {
		return errors.Wrap(err, "PgxStorage.ReassignReviewer.LockPR")
	}

	const qDelete = `DELETE FROM pr_reviewers WHERE pull_request_id = $1 AND user_id = $2`
	commandTag, err := tx.Exec(ctx, qDelete, prID, oldUserID)
	if err != nil {
		return errors.Wrap(err, "PgxStorage.ReassignReviewer.RemoveOld")
	}
	if commandTag.RowsAffected() == 0 {
		return errors.Wrap(pgx.ErrNoRows, "PgxStorage.ReassignReviewer.OldReviewerNotFound")
	}

	const qInsert = `INSERT INTO pr_reviewers (pull_request_id, user_id) VALUES ($1, $2)`
	_, err = tx.Exec(ctx, qInsert, prID, newUserID)
	if err != nil {
		return errors.Wrap(err, "PgxStorage.ReassignReviewer.AssignNew")
	}

	if err = tx.Commit(ctx); err != nil {
		return errors.Wrap(err, "PgxStorage.ReassignReviewer.Commit")
	}

	return nil
}

func (p *PgxStorage) GetPRsByReviewer(ctx context.Context, userID string) ([]*en.PullRequestShort, error) {
	const q = `
		SELECT pr.pull_request_id, pr.pull_request_name, pr.author_id, pr.status
		FROM pull_requests pr
		JOIN pr_reviewers r ON pr.pull_request_id = r.pull_request_id
		WHERE r.user_id = $1
		ORDER BY pr.created_at DESC
	`
	rows, err := p.pool.Query(ctx, q, userID)
	if err != nil {
		return nil, errors.Wrap(err, "PgxStorage.GetPRsByReviewer")
	}
	defer rows.Close()

	var prs []*en.PullRequestShort
	for rows.Next() {
		var pr en.PullRequestShort
		if err := rows.Scan(&pr.PullRequestID, &pr.PullRequestName, &pr.AuthorID, &pr.Status); err != nil {
			return nil, errors.Wrap(err, "PgxStorage.GetPRsByReviewer.Scan")
		}
		prs = append(prs, &pr)
	}
	if rows.Err() != nil {
		return nil, errors.Wrap(rows.Err(), "PgxStorage.GetPRsByReviewer.RowsError")
	}

	return prs, nil
}

func (p *PgxStorage) IsUserAssignedToReviewer(ctx context.Context, prID string, userID string) (bool, error) {
	const q = `SELECT EXISTS(SELECT 1 FROM pr_reviewers WHERE pull_request_id = $1 AND user_id = $2)`
	var exists bool
	err := p.pool.QueryRow(ctx, q, prID, userID).Scan(&exists)
	if err != nil {
		return false, errors.Wrap(err, "PgxStorage.IsUserAssignedToReviewer")
	}
	return exists, nil
}
