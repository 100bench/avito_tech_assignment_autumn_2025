package usecases

//go:generate mockery --name=Storage --output=. --outpkg=usecases --filename=mock_storage_test.go --with-expecter

import (
	"context"
	"math/rand"
	"time"

	en "github.com/100bench/avito_tech_assignment_autumn_2025/internal/entities"
	"github.com/pkg/errors"
)

type ServiceStorage struct {
	storage Storage
}

func NewServiceStorage(storage Storage) (*ServiceStorage, error) {
	if storage == nil {
		return nil, errors.New("storage cannot be nil")
	}
	return &ServiceStorage{storage: storage}, nil
}

// createTeam создает команду с участниками
func (s *ServiceStorage) CreateTeam(ctx context.Context, teamName string, members []en.TeamMember) (*en.Team, error) {
	if teamName == "" {
		return nil, errors.New("team name cannot be empty")
	}
	if len(members) == 0 {
		return nil, errors.New("team must have at least one member")
	}

	exists, err := s.storage.TeamExists(ctx, teamName)
	if err != nil {
		return nil, errors.Wrap(err, "failed to check team existence")
	}
	if exists {
		return nil, en.NewTeamExistsError(teamName)
	}

	users := make([]*en.User, len(members))
	for i, m := range members {
		users[i] = &en.User{
			UserID:   m.UserID,
			Username: m.Username,
			TeamName: teamName,
			IsActive: m.IsActive,
		}
	}

	err = s.storage.CreateTeamWithUsers(ctx, teamName, users)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create team with users")
	}

	return &en.Team{
		TeamName:    teamName,
		TeamMembers: members,
	}, nil
}

// getTeam возвращает команду с участниками
func (s *ServiceStorage) GetTeam(ctx context.Context, teamName string) (*en.Team, error) {
	team, err := s.storage.GetTeamByName(ctx, teamName)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get team")
	}
	if team == nil {
		return nil, en.NewNotFoundError("team", teamName)
	}
	return team, nil
}

// setUserActive устанавливает флаг активности пользователя
func (s *ServiceStorage) SetUserActive(ctx context.Context, userID string, isActive bool) (*en.User, error) {
	user, err := s.storage.SetUserActiveStatus(ctx, userID, isActive)
	if err != nil {
		return nil, errors.Wrap(err, "failed to set user active status")
	}
	if user == nil {
		return nil, en.NewNotFoundError("user", userID)
	}
	return user, nil
}

// getUserReviews возвращает список PR где пользователь назначен ревьювером
func (s *ServiceStorage) GetUserReviews(ctx context.Context, userID string) ([]*en.PullRequestShort, error) {
	prs, err := s.storage.GetPRsByReviewer(ctx, userID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get user reviews")
	}
	return prs, nil
}

// createPullRequest создает PR и автоматически назначает до 2 ревьюверов из команды автора
func (s *ServiceStorage) CreatePullRequest(ctx context.Context, prID, prName, authorID string) (*en.PullRequest, error) {
	if prID == "" || prName == "" || authorID == "" {
		return nil, errors.New("prID, prName and authorID cannot be empty")
	}

	exists, err := s.storage.PRExists(ctx, prID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to check PR existence")
	}
	if exists {
		return nil, en.NewPRExistsError(prID)
	}

	author, err := s.storage.GetUser(ctx, authorID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get author")
	}
	if author == nil {
		return nil, en.NewNotFoundError("author", authorID)
	}

	// получаем только активных участников команды
	teamMembers, err := s.storage.GetUsersByTeam(ctx, author.TeamName, true)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get team members")
	}

	// исключаем автора из кандидатов
	var candidates []*en.User
	for _, member := range teamMembers {
		if member.UserID != authorID {
			candidates = append(candidates, member)
		}
	}

	reviewerIDs := selectRandomReviewers(candidates, 2)

	pr := &en.PullRequest{
		PullRequestID:     prID,
		PullRequestName:   prName,
		AuthorID:          authorID,
		Status:            en.StatusOpen,
		AssignedReviewers: reviewerIDs,
		CreatedAt:         time.Now(),
		MergedAt:          nil,
	}

	err = s.storage.CreatePRWithReviewers(ctx, pr, reviewerIDs)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create PR with reviewers")
	}

	return pr, nil
}

// mergePullRequest помечает PR как MERGED. Операция идемпотентная
func (s *ServiceStorage) MergePullRequest(ctx context.Context, prID string) (*en.PullRequest, error) {
	pr, err := s.storage.GetPR(ctx, prID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get PR")
	}
	if pr == nil {
		return nil, en.NewNotFoundError("pull request", prID)
	}

	// если уже смержен, возвращаем текущее состояние
	if pr.Status == en.StatusMerged {
		return pr, nil
	}

	mergedPR, err := s.storage.MergePR(ctx, prID, time.Now())
	if err != nil {
		return nil, errors.Wrap(err, "failed to merge PR")
	}

	return mergedPR, nil
}

// reassignReviewer заменяет ревьювера на случайного активного участника из команды заменяемого
func (s *ServiceStorage) ReassignReviewer(ctx context.Context, prID, oldUserID string) (*en.PullRequest, string, error) {
	pr, err := s.storage.GetPR(ctx, prID)
	if err != nil {
		return nil, "", errors.Wrap(err, "failed to get PR")
	}
	if pr == nil {
		return nil, "", en.NewNotFoundError("pull request", prID)
	}

	if pr.Status == en.StatusMerged {
		return nil, "", en.NewPRMergedError(prID)
	}

	isAssigned, err := s.storage.IsUserAssignedToReviewer(ctx, prID, oldUserID)
	if err != nil {
		return nil, "", errors.Wrap(err, "failed to check reviewer assignment")
	}
	if !isAssigned {
		return nil, "", en.NewNotAssignedError(oldUserID, prID)
	}

	oldUser, err := s.storage.GetUser(ctx, oldUserID)
	if err != nil {
		return nil, "", errors.Wrap(err, "failed to get old reviewer")
	}
	if oldUser == nil {
		return nil, "", en.NewNotFoundError("user", oldUserID)
	}

	// получаем активных участников команды заменяемого ревьювера
	teamMembers, err := s.storage.GetUsersByTeam(ctx, oldUser.TeamName, true)
	if err != nil {
		return nil, "", errors.Wrap(err, "failed to get team members")
	}

	// исключаем автора и уже назначенных ревьюверов
	var candidates []*en.User
	for _, member := range teamMembers {
		if member.UserID == pr.AuthorID {
			continue
		}
		if contains(pr.AssignedReviewers, member.UserID) {
			continue
		}
		candidates = append(candidates, member)
	}

	if len(candidates) == 0 {
		return nil, "", en.NewNoCandidateError(oldUser.TeamName)
	}

	newUserID := candidates[rand.Intn(len(candidates))].UserID

	err = s.storage.ReassignReviewer(ctx, prID, oldUserID, newUserID)
	if err != nil {
		return nil, "", errors.Wrap(err, "failed to reassign reviewer")
	}

	updatedPR, err := s.storage.GetPR(ctx, prID)
	if err != nil {
		return nil, "", errors.Wrap(err, "failed to get updated PR")
	}

	return updatedPR, newUserID, nil
}

func selectRandomReviewers(candidates []*en.User, max int) []string {
	if len(candidates) == 0 {
		return []string{}
	}

	count := min(len(candidates), max)

	shuffled := make([]*en.User, len(candidates))
	copy(shuffled, candidates)
	rand.Shuffle(len(shuffled), func(i, j int) {
		shuffled[i], shuffled[j] = shuffled[j], shuffled[i]
	})

	result := make([]string, count)
	for i := 0; i < count; i++ {
		result[i] = shuffled[i].UserID
	}

	return result
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// DeactivateTeamMembers массово деактивирует пользователей команды и переназначает их открытые PR
func (s *ServiceStorage) DeactivateTeamMembers(ctx context.Context, teamName string, userIDs []string) (*en.DeactivateResult, error) {
	if teamName == "" {
		return nil, errors.New("team name cannot be empty")
	}
	if len(userIDs) == 0 {
		return nil, errors.New("user IDs cannot be empty")
	}

	result, err := s.storage.DeactivateTeamMembersWithReassignment(ctx, teamName, userIDs)
	if err != nil {
		return nil, errors.Wrap(err, "failed to deactivate team members with reassignment")
	}

	return result, nil
}

// GetStats возвращает статистику по назначениям ревьюверов и PR
func (s *ServiceStorage) GetStats(ctx context.Context) (*en.Stats, error) {
	stats, err := s.storage.GetStats(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get stats")
	}
	return stats, nil
}
