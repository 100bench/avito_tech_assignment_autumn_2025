package usecases

import (
	"context"
	"github.com/100bench/avito_tech_assignment_autumn_2025/internal/entities"
	"time"
)

// интерфейс для взаимодействия с хранилищем данных
type Storage interface {
	// Teams. createTeamWithUsers создает команду и всех пользователей атомарно в одной транзакции
	CreateTeamWithUsers(ctx context.Context, teamName string, users []*entities.User) error
	GetTeamByName(ctx context.Context, teamName string) (*entities.Team, error)
	TeamExists(ctx context.Context, teamName string) (bool, error)
	
	// Users
	GetUser(ctx context.Context, userID string) (*entities.User, error)
	GetUsersByTeam(ctx context.Context, teamName string, activeOnly bool) ([]*entities.User, error)
	SetUserActiveStatus(ctx context.Context, userID string, isActive bool) (*entities.User, error)
	
	// Pull Requests. createPRWithReviewers создает PR и назначает ревьюверов атомарно
	CreatePRWithReviewers(ctx context.Context, pr *entities.PullRequest, reviewerIDs []string) error
	GetPR(ctx context.Context, prID string) (*entities.PullRequest, error)
	MergePR(ctx context.Context, prID string, mergedAt time.Time) (*entities.PullRequest, error)
	PRExists(ctx context.Context, prID string) (bool, error)
	
	// Reviewers. reassignReviewer заменяет ревьювера атомарно (удаление старого + добавление нового)
	ReassignReviewer(ctx context.Context, prID string, oldUserID string, newUserID string) error
	GetPRsByReviewer(ctx context.Context, userID string) ([]*entities.PullRequestShort, error)
	IsUserAssignedToReviewer(ctx context.Context, prID string, userID string) (bool, error)
}