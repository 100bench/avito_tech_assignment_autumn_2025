package public

import (
	"context"

	"github.com/100bench/avito_tech_assignment_autumn_2025/internal/entities"
)

// интерфейс юзкейса имплементируется в хендлерах и используется для вызова бизнес-логики
type PRReviewService interface {
	CreateTeam(ctx context.Context, teamName string, members []entities.TeamMember) (*entities.Team, error)
	GetTeam(ctx context.Context, teamName string) (*entities.Team, error)

	SetUserActive(ctx context.Context, userID string, isActive bool) (*entities.User, error)
	GetUserReviews(ctx context.Context, userID string) ([]*entities.PullRequestShort, error)

	CreatePullRequest(ctx context.Context, prID, prName, authorID string) (*entities.PullRequest, error)
	MergePullRequest(ctx context.Context, prID string) (*entities.PullRequest, error)
	ReassignReviewer(ctx context.Context, prID, oldUserID string) (pr *entities.PullRequest, newReviewerID string, err error)

	DeactivateTeamMembers(ctx context.Context, teamName string, userIDs []string) (*entities.DeactivateResult, error)

	GetStats(ctx context.Context) (*entities.Stats, error)
}
