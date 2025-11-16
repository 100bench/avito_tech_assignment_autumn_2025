package postgres

import (
    "context"
    "math/rand"

    en "github.com/100bench/avito_tech_assignment_autumn_2025/internal/entities"
    "github.com/pkg/errors"
)

//nolint:funlen
func (p *PgxStorage) DeactivateTeamMembersWithReassignment(ctx context.Context, teamName string, userIDs []string) (*en.DeactivateResult, error) {
    if len(userIDs) == 0 {
        return &en.DeactivateResult{
            DeactivatedUsers: []string{},
            Reassignments:    []en.PRReassignmentInfo{},
        }, nil
    }

    tx, err := p.pool.Begin(ctx)
    if err != nil {
        return nil, errors.Wrap(err, "begin tx")
    }
    defer func() { _ = tx.Rollback(ctx) }()

    // Проверяем существование и принадлежность пользователей
    const qCheckUsers = `
        SELECT user_id, team_name, is_active
        FROM users
        WHERE user_id = ANY($1)`
    checkRows, err := tx.Query(ctx, qCheckUsers, userIDs)
    if err != nil {
        return nil, errors.Wrap(err, "check users")
    }
    foundUsers := make(map[string]struct {
        teamName string
        isActive bool
    })
    for checkRows.Next() {
        var uid, team string
        var active bool
        if err := checkRows.Scan(&uid, &team, &active); err != nil {
            checkRows.Close()
            return nil, errors.Wrap(err, "scan user check")
        }
        foundUsers[uid] = struct {
            teamName string
            isActive bool
        }{teamName: team, isActive: active}
    }
    checkRows.Close()
    for _, uid := range userIDs {
        user, ok := foundUsers[uid]
        if !ok {
            return nil, en.NewNotFoundError("user", uid)
        }
        if user.teamName != teamName {
            return nil, en.NewInvalidTeamUserError(uid, teamName, "does not belong to")
        }
        if !user.isActive {
            return nil, en.NewInvalidTeamUserError(uid, teamName, "is not an active member of")
        }
    }

    // Активные члены команды для кандидатов
    const qMembers = `
        SELECT user_id, username, team_name, is_active, created_at, updated_at
        FROM users
        WHERE team_name = $1 AND is_active = true
        ORDER BY user_id`
    rows, err := tx.Query(ctx, qMembers, teamName)
    if err != nil {
        return nil, errors.Wrap(err, "select members")
    }
    var allActive []*en.User
    for rows.Next() {
        var u en.User
        if err := rows.Scan(&u.UserID, &u.Username, &u.TeamName, &u.IsActive, &u.CreatedAt, &u.UpdatedAt); err != nil {
            rows.Close()
            return nil, errors.Wrap(err, "scan member")
        }
        allActive = append(allActive, &u)
    }
    rows.Close()

    // Один запрос: PR с любым из деактивируемых ревьюеров + полный список ревьюеров
    const qPRsWithAllRevs = `
        WITH target_prs AS (
            SELECT DISTINCT pr.pull_request_id, pr.pull_request_name, pr.author_id, pr.status, pr.created_at, pr.merged_at
            FROM pull_requests pr
            JOIN pr_reviewers r ON pr.pull_request_id = r.pull_request_id
            WHERE r.user_id = ANY($1) AND pr.status = 'OPEN'
        )
        SELECT t.pull_request_id, t.pull_request_name, t.author_id, t.status, t.created_at, t.merged_at,
               COALESCE(array_agg(r2.user_id ORDER BY r2.user_id) FILTER (WHERE r2.user_id IS NOT NULL), '{}') AS reviewers
        FROM target_prs t
        LEFT JOIN pr_reviewers r2 ON r2.pull_request_id = t.pull_request_id
        GROUP BY t.pull_request_id, t.pull_request_name, t.author_id, t.status, t.created_at, t.merged_at`
    prRows, err := tx.Query(ctx, qPRsWithAllRevs, userIDs)
    if err != nil {
        return nil, errors.Wrap(err, "select prs with reviewers")
    }
    var openPRs []*en.PullRequest
    for prRows.Next() {
        var pr en.PullRequest
        var status string
        var reviewers []string
        if err := prRows.Scan(&pr.PullRequestID, &pr.PullRequestName, &pr.AuthorID, &status, &pr.CreatedAt, &pr.MergedAt, &reviewers); err != nil {
            prRows.Close()
            return nil, errors.Wrap(err, "scan pr with reviewers")
        }
        pr.Status = en.PRStatus(status)
        pr.AssignedReviewers = reviewers
        openPRs = append(openPRs, &pr)
    }
    prRows.Close()

    toDeactivate := make(map[string]bool, len(userIDs))
    for _, id := range userIDs {
        toDeactivate[id] = true
    }

    var infos []en.PRReassignmentInfo
    for _, pr := range openPRs {
        for _, old := range pr.AssignedReviewers {
            if !toDeactivate[old] {
                continue
            }

            // кандидаты: активные, не автор, не деактивируемые, не уже назначенные
            var cands []*en.User
            for _, m := range allActive {
                if m.UserID == pr.AuthorID || toDeactivate[m.UserID] || containsStr(pr.AssignedReviewers, m.UserID) {
                    continue
                }
                cands = append(cands, m)
            }

            var newID string
            if len(cands) > 0 {
                newID = cands[rand.Intn(len(cands))].UserID
            }

            const qDel = `DELETE FROM pr_reviewers WHERE pull_request_id = $1 AND user_id = $2`
            if _, err := tx.Exec(ctx, qDel, pr.PullRequestID, old); err != nil {
                return nil, errors.Wrap(err, "delete reviewer")
            }
            if newID != "" {
                const qIns = `INSERT INTO pr_reviewers (pull_request_id, user_id) VALUES ($1, $2)
                              ON CONFLICT (pull_request_id, user_id) DO NOTHING`
                if _, err := tx.Exec(ctx, qIns, pr.PullRequestID, newID); err != nil {
                    return nil, errors.Wrap(err, "insert reviewer")
                }
                // обновим локальный список, чтобы не выбрать того же кандидата повторно
                pr.AssignedReviewers = append(pr.AssignedReviewers, newID)
            }

            infos = append(infos, en.PRReassignmentInfo{
                PullRequestID: pr.PullRequestID,
                OldReviewer:   old,
                NewReviewer:   newID,
            })
        }
    }

    const qDeactivate = `
        UPDATE users
        SET is_active = false, updated_at = NOW()
        WHERE team_name = $1 AND user_id = ANY($2)
        RETURNING user_id`
    dr, err := tx.Query(ctx, qDeactivate, teamName, userIDs)
    if err != nil {
        return nil, errors.Wrap(err, "deactivate users")
    }
    var deactivated []string
    for dr.Next() {
        var id string
        if err := dr.Scan(&id); err != nil {
            dr.Close()
            return nil, errors.Wrap(err, "scan deactivated")
        }
        deactivated = append(deactivated, id)
    }
    dr.Close()

    if err := tx.Commit(ctx); err != nil {
        return nil, errors.Wrap(err, "commit")
    }

    // гарантируем не-nil слайсы
    if deactivated == nil {
        deactivated = []string{}
    }
    if infos == nil {
        infos = []en.PRReassignmentInfo{}
    }
    return &en.DeactivateResult{DeactivatedUsers: deactivated, Reassignments: infos}, nil
}

func containsStr(ss []string, x string) bool {
    for _, s := range ss {
        if s == x {
            return true
        }
    }
    return false
}