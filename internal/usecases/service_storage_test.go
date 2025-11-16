package usecases

import (
	"context"
	"errors"
	"testing"
	"time"

	en "github.com/100bench/avito_tech_assignment_autumn_2025/internal/entities"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// 1. CreateTeam Tests
func TestCreateTeam_Success(t *testing.T) {
	mockStorage := NewMockStorage(t)
	service := &ServiceStorage{storage: mockStorage}

	ctx := context.Background()
	teamName := "backend"
	members := []en.TeamMember{
		{UserID: "u1", Username: "Alice", IsActive: true},
		{UserID: "u2", Username: "Bob", IsActive: true},
	}

	mockStorage.EXPECT().TeamExists(ctx, teamName).Return(false, nil).Once()
	mockStorage.EXPECT().CreateTeamWithUsers(ctx, teamName, mock.AnythingOfType("[]*entities.User")).Return(nil).Once()

	team, err := service.CreateTeam(ctx, teamName, members)

	require.NoError(t, err)
	assert.NotNil(t, team)
	assert.Equal(t, teamName, team.TeamName)
	assert.Len(t, team.TeamMembers, 2)
}

func TestCreateTeam_EmptyTeamName(t *testing.T) {
	mockStorage := NewMockStorage(t)
	service := &ServiceStorage{storage: mockStorage}

	ctx := context.Background()
	members := []en.TeamMember{
		{UserID: "u1", Username: "Alice", IsActive: true},
	}

	team, err := service.CreateTeam(ctx, "", members)

	require.Error(t, err)
	assert.Nil(t, team)
	assert.Contains(t, err.Error(), "team name cannot be empty")
}

func TestCreateTeam_NoMembers(t *testing.T) {
	mockStorage := NewMockStorage(t)
	service := &ServiceStorage{storage: mockStorage}

	ctx := context.Background()
	teamName := "backend"

	team, err := service.CreateTeam(ctx, teamName, []en.TeamMember{})

	require.Error(t, err)
	assert.Nil(t, team)
	assert.Contains(t, err.Error(), "at least one member")
}

func TestCreateTeam_AlreadyExists(t *testing.T) {
	mockStorage := NewMockStorage(t)
	service := &ServiceStorage{storage: mockStorage}

	ctx := context.Background()
	teamName := "existing"
	members := []en.TeamMember{
		{UserID: "u1", Username: "Alice", IsActive: true},
	}

	mockStorage.EXPECT().TeamExists(ctx, teamName).Return(true, nil).Once()

	team, err := service.CreateTeam(ctx, teamName, members)

	require.Error(t, err)
	assert.Nil(t, team)
	var appErr *en.AppError
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, en.ErrCodeTeamExists, appErr.Code)
}

func TestCreateTeam_StorageError(t *testing.T) {
	mockStorage := NewMockStorage(t)
	service := &ServiceStorage{storage: mockStorage}

	ctx := context.Background()
	teamName := "backend"
	members := []en.TeamMember{
		{UserID: "u1", Username: "Alice", IsActive: true},
	}

	mockStorage.EXPECT().TeamExists(ctx, teamName).Return(false, nil).Once()
	mockStorage.EXPECT().CreateTeamWithUsers(ctx, teamName, mock.Anything).Return(errors.New("storage error")).Once()

	team, err := service.CreateTeam(ctx, teamName, members)

	require.Error(t, err)
	assert.Nil(t, team)
}

// 2. GetTeam Tests
func TestGetTeam_Success(t *testing.T) {
	mockStorage := NewMockStorage(t)
	service := &ServiceStorage{storage: mockStorage}

	ctx := context.Background()
	teamName := "backend"
	expectedTeam := &en.Team{
		TeamName: teamName,
		TeamMembers: []en.TeamMember{
			{UserID: "u1", Username: "Alice", IsActive: true},
		},
	}

	mockStorage.EXPECT().GetTeamByName(ctx, teamName).Return(expectedTeam, nil).Once()

	team, err := service.GetTeam(ctx, teamName)

	require.NoError(t, err)
	assert.Equal(t, expectedTeam, team)
}

func TestGetTeam_NotFound(t *testing.T) {
	mockStorage := NewMockStorage(t)
	service := &ServiceStorage{storage: mockStorage}

	ctx := context.Background()
	teamName := "nonexistent"

	mockStorage.EXPECT().GetTeamByName(ctx, teamName).Return(nil, nil).Once()

	team, err := service.GetTeam(ctx, teamName)

	require.Error(t, err)
	assert.Nil(t, team)
	var appErr *en.AppError
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, en.ErrCodeNotFound, appErr.Code)
}

func TestGetTeam_StorageError(t *testing.T) {
	mockStorage := NewMockStorage(t)
	service := &ServiceStorage{storage: mockStorage}

	ctx := context.Background()
	teamName := "backend"

	mockStorage.EXPECT().GetTeamByName(ctx, teamName).Return(nil, errors.New("storage error")).Once()

	team, err := service.GetTeam(ctx, teamName)

	require.Error(t, err)
	assert.Nil(t, team)
}

// 3. SetUserActive Tests
func TestSetUserActive_Success(t *testing.T) {
	mockStorage := NewMockStorage(t)
	service := &ServiceStorage{storage: mockStorage}

	ctx := context.Background()
	userID := "u1"
	isActive := false
	expectedUser := &en.User{
		UserID:   userID,
		Username: "Alice",
		TeamName: "backend",
		IsActive: isActive,
	}

	mockStorage.EXPECT().SetUserActiveStatus(ctx, userID, isActive).Return(expectedUser, nil).Once()

	user, err := service.SetUserActive(ctx, userID, isActive)

	require.NoError(t, err)
	assert.Equal(t, expectedUser, user)
}

func TestSetUserActive_NotFound(t *testing.T) {
	mockStorage := NewMockStorage(t)
	service := &ServiceStorage{storage: mockStorage}

	ctx := context.Background()
	userID := "nonexistent"

	mockStorage.EXPECT().SetUserActiveStatus(ctx, userID, true).Return(nil, nil).Once()

	user, err := service.SetUserActive(ctx, userID, true)

	require.Error(t, err)
	assert.Nil(t, user)
	var appErr *en.AppError
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, en.ErrCodeNotFound, appErr.Code)
}

func TestSetUserActive_StorageError(t *testing.T) {
	mockStorage := NewMockStorage(t)
	service := &ServiceStorage{storage: mockStorage}

	ctx := context.Background()
	userID := "u1"

	mockStorage.EXPECT().SetUserActiveStatus(ctx, userID, false).Return(nil, errors.New("storage error")).Once()

	user, err := service.SetUserActive(ctx, userID, false)

	require.Error(t, err)
	assert.Nil(t, user)
}

// 4. GetUserReviews Tests
func TestGetUserReviews_Success(t *testing.T) {
	mockStorage := NewMockStorage(t)
	service := &ServiceStorage{storage: mockStorage}

	ctx := context.Background()
	userID := "u1"
	expectedPRs := []*en.PullRequestShort{
		{PullRequestID: "pr1", PullRequestName: "Feature A", AuthorID: "u2", Status: "OPEN"},
		{PullRequestID: "pr2", PullRequestName: "Feature B", AuthorID: "u3", Status: "OPEN"},
	}

	mockStorage.EXPECT().GetPRsByReviewer(ctx, userID).Return(expectedPRs, nil).Once()

	prs, err := service.GetUserReviews(ctx, userID)

	require.NoError(t, err)
	assert.Len(t, prs, 2)
	assert.Equal(t, expectedPRs, prs)
}

func TestGetUserReviews_Empty(t *testing.T) {
	mockStorage := NewMockStorage(t)
	service := &ServiceStorage{storage: mockStorage}

	ctx := context.Background()
	userID := "u1"

	mockStorage.EXPECT().GetPRsByReviewer(ctx, userID).Return([]*en.PullRequestShort{}, nil).Once()

	prs, err := service.GetUserReviews(ctx, userID)

	require.NoError(t, err)
	assert.Empty(t, prs)
}

func TestGetUserReviews_StorageError(t *testing.T) {
	mockStorage := NewMockStorage(t)
	service := &ServiceStorage{storage: mockStorage}

	ctx := context.Background()
	userID := "u1"

	mockStorage.EXPECT().GetPRsByReviewer(ctx, userID).Return(nil, errors.New("storage error")).Once()

	prs, err := service.GetUserReviews(ctx, userID)

	require.Error(t, err)
	assert.Nil(t, prs)
}

// 5. CreatePullRequest Tests
func TestCreatePR_Success_TwoReviewers(t *testing.T) {
	mockStorage := NewMockStorage(t)
	service := &ServiceStorage{storage: mockStorage}

	ctx := context.Background()
	prID := "pr-1"
	prName := "Feature X"
	authorID := "u1"

	author := &en.User{UserID: authorID, Username: "Alice", TeamName: "backend", IsActive: true}
	candidates := []*en.User{
		{UserID: "u2", Username: "Bob", TeamName: "backend", IsActive: true},
		{UserID: "u3", Username: "Charlie", TeamName: "backend", IsActive: true},
		{UserID: "u4", Username: "Dave", TeamName: "backend", IsActive: true},
	}

	mockStorage.EXPECT().PRExists(ctx, prID).Return(false, nil).Once()
	mockStorage.EXPECT().GetUser(ctx, authorID).Return(author, nil).Once()
	mockStorage.EXPECT().GetUsersByTeam(ctx, "backend", true).Return(candidates, nil).Once()
	mockStorage.EXPECT().CreatePRWithReviewers(ctx, mock.MatchedBy(func(pr *en.PullRequest) bool {
		return pr.PullRequestID == prID && pr.PullRequestName == prName && pr.AuthorID == authorID
	}), mock.MatchedBy(func(reviewers []string) bool {
		return len(reviewers) == 2
	})).Return(nil).Once()

	pr, err := service.CreatePullRequest(ctx, prID, prName, authorID)

	require.NoError(t, err)
	assert.NotNil(t, pr)
	assert.Equal(t, prID, pr.PullRequestID)
}

func TestCreatePR_Success_OneReviewer(t *testing.T) {
	mockStorage := NewMockStorage(t)
	service := &ServiceStorage{storage: mockStorage}

	ctx := context.Background()
	prID := "pr-1"
	prName := "Feature X"
	authorID := "u1"

	author := &en.User{UserID: authorID, Username: "Alice", TeamName: "backend", IsActive: true}
	candidates := []*en.User{
		{UserID: "u2", Username: "Bob", TeamName: "backend", IsActive: true},
	}

	mockStorage.EXPECT().PRExists(ctx, prID).Return(false, nil).Once()
	mockStorage.EXPECT().GetUser(ctx, authorID).Return(author, nil).Once()
	mockStorage.EXPECT().GetUsersByTeam(ctx, "backend", true).Return(candidates, nil).Once()
	mockStorage.EXPECT().CreatePRWithReviewers(ctx, mock.Anything, mock.MatchedBy(func(reviewers []string) bool {
		return len(reviewers) == 1
	})).Return(nil).Once()

	pr, err := service.CreatePullRequest(ctx, prID, prName, authorID)

	require.NoError(t, err)
	assert.NotNil(t, pr)
}

func TestCreatePR_Success_NoReviewers(t *testing.T) {
	mockStorage := NewMockStorage(t)
	service := &ServiceStorage{storage: mockStorage}

	ctx := context.Background()
	prID := "pr-1"
	prName := "Feature X"
	authorID := "u1"

	author := &en.User{UserID: authorID, Username: "Alice", TeamName: "backend", IsActive: true}

	mockStorage.EXPECT().PRExists(ctx, prID).Return(false, nil).Once()
	mockStorage.EXPECT().GetUser(ctx, authorID).Return(author, nil).Once()
	mockStorage.EXPECT().GetUsersByTeam(ctx, "backend", true).Return([]*en.User{}, nil).Once()
	mockStorage.EXPECT().CreatePRWithReviewers(ctx, mock.Anything, mock.MatchedBy(func(reviewers []string) bool {
		return len(reviewers) == 0
	})).Return(nil).Once()

	pr, err := service.CreatePullRequest(ctx, prID, prName, authorID)

	require.NoError(t, err)
	assert.NotNil(t, pr)
}

func TestCreatePR_EmptyFields(t *testing.T) {
	mockStorage := NewMockStorage(t)
	service := &ServiceStorage{storage: mockStorage}

	ctx := context.Background()

	tests := []struct {
		name     string
		prID     string
		prName   string
		authorID string
	}{
		{"empty prID", "", "Feature", "u1"},
		{"empty prName", "pr-1", "", "u1"},
		{"empty authorID", "pr-1", "Feature", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pr, err := service.CreatePullRequest(ctx, tt.prID, tt.prName, tt.authorID)
			require.Error(t, err)
			assert.Nil(t, pr)
			assert.Contains(t, err.Error(), "cannot be empty")
		})
	}
}

func TestCreatePR_AlreadyExists(t *testing.T) {
	mockStorage := NewMockStorage(t)
	service := &ServiceStorage{storage: mockStorage}

	ctx := context.Background()
	prID := "existing"

	mockStorage.EXPECT().PRExists(ctx, prID).Return(true, nil).Once()

	pr, err := service.CreatePullRequest(ctx, prID, "Feature", "u1")

	require.Error(t, err)
	assert.Nil(t, pr)
	var appErr *en.AppError
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, en.ErrCodePRExists, appErr.Code)
}

func TestCreatePR_AuthorNotFound(t *testing.T) {
	mockStorage := NewMockStorage(t)
	service := &ServiceStorage{storage: mockStorage}

	ctx := context.Background()
	prID := "pr-1"
	authorID := "nonexistent"

	mockStorage.EXPECT().PRExists(ctx, prID).Return(false, nil).Once()
	mockStorage.EXPECT().GetUser(ctx, authorID).Return(nil, nil).Once()

	pr, err := service.CreatePullRequest(ctx, prID, "Feature", authorID)

	require.Error(t, err)
	assert.Nil(t, pr)
	var appErr *en.AppError
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, en.ErrCodeNotFound, appErr.Code)
}

func TestCreatePR_StorageError(t *testing.T) {
	mockStorage := NewMockStorage(t)
	service := &ServiceStorage{storage: mockStorage}

	ctx := context.Background()
	prID := "pr-1"
	authorID := "u1"

	author := &en.User{UserID: authorID, Username: "Alice", TeamName: "backend", IsActive: true}

	mockStorage.EXPECT().PRExists(ctx, prID).Return(false, nil).Once()
	mockStorage.EXPECT().GetUser(ctx, authorID).Return(author, nil).Once()
	mockStorage.EXPECT().GetUsersByTeam(ctx, "backend", true).Return([]*en.User{}, nil).Once()
	mockStorage.EXPECT().CreatePRWithReviewers(ctx, mock.Anything, mock.Anything).Return(errors.New("storage error")).Once()

	pr, err := service.CreatePullRequest(ctx, prID, "Feature", authorID)

	require.Error(t, err)
	assert.Nil(t, pr)
}

// 6. MergePullRequest Tests
func TestMergePR_Success(t *testing.T) {
	mockStorage := NewMockStorage(t)
	service := &ServiceStorage{storage: mockStorage}

	ctx := context.Background()
	prID := "pr-1"
	now := time.Now()

	openPR := &en.PullRequest{
		PullRequestID:   prID,
		PullRequestName: "Feature",
		AuthorID:        "u1",
		Status:          en.StatusOpen,
	}
	mergedPR := &en.PullRequest{
		PullRequestID:   prID,
		PullRequestName: "Feature",
		AuthorID:        "u1",
		Status:          en.StatusMerged,
		MergedAt:        &now,
	}

	mockStorage.EXPECT().GetPR(ctx, prID).Return(openPR, nil).Once()
	mockStorage.EXPECT().MergePR(ctx, prID, mock.AnythingOfType("time.Time")).Return(mergedPR, nil).Once()

	pr, err := service.MergePullRequest(ctx, prID)

	require.NoError(t, err)
	assert.Equal(t, en.StatusMerged, pr.Status)
}

func TestMergePR_Idempotent(t *testing.T) {
	mockStorage := NewMockStorage(t)
	service := &ServiceStorage{storage: mockStorage}

	ctx := context.Background()
	prID := "pr-1"
	now := time.Now()

	mergedPR := &en.PullRequest{
		PullRequestID:   prID,
		PullRequestName: "Feature",
		AuthorID:        "u1",
		Status:          en.StatusMerged,
		MergedAt:        &now,
	}

	mockStorage.EXPECT().GetPR(ctx, prID).Return(mergedPR, nil).Once()

	pr, err := service.MergePullRequest(ctx, prID)

	require.NoError(t, err)
	assert.Equal(t, en.StatusMerged, pr.Status)
}

func TestMergePR_NotFound(t *testing.T) {
	mockStorage := NewMockStorage(t)
	service := &ServiceStorage{storage: mockStorage}

	ctx := context.Background()
	prID := "nonexistent"

	mockStorage.EXPECT().GetPR(ctx, prID).Return(nil, nil).Once()

	pr, err := service.MergePullRequest(ctx, prID)

	require.Error(t, err)
	assert.Nil(t, pr)
	var appErr *en.AppError
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, en.ErrCodeNotFound, appErr.Code)
}

func TestMergePR_StorageError(t *testing.T) {
	mockStorage := NewMockStorage(t)
	service := &ServiceStorage{storage: mockStorage}

	ctx := context.Background()
	prID := "pr-1"

	openPR := &en.PullRequest{
		PullRequestID: prID,
		Status:        en.StatusOpen,
		AuthorID:      "u1",
	}

	mockStorage.EXPECT().GetPR(ctx, prID).Return(openPR, nil).Once()
	mockStorage.EXPECT().MergePR(ctx, prID, mock.AnythingOfType("time.Time")).Return(nil, errors.New("storage error")).Once()

	pr, err := service.MergePullRequest(ctx, prID)

	require.Error(t, err)
	assert.Nil(t, pr)
}

// 7. ReassignReviewer Tests
func TestReassignReviewer_Success(t *testing.T) {
	mockStorage := NewMockStorage(t)
	service := &ServiceStorage{storage: mockStorage}

	ctx := context.Background()
	prID := "pr-1"
	oldUserID := "u2"

	pr := &en.PullRequest{
		PullRequestID:     prID,
		Status:            en.StatusOpen,
		AuthorID:          "u1",
		AssignedReviewers: []string{oldUserID, "u3"},
	}
	oldUser := &en.User{UserID: oldUserID, Username: "Bob", TeamName: "backend", IsActive: true}
	candidates := []*en.User{
		{UserID: "u1", Username: "Alice", TeamName: "backend", IsActive: true},
		{UserID: "u4", Username: "Dave", TeamName: "backend", IsActive: true},
		{UserID: "u5", Username: "Eve", TeamName: "backend", IsActive: true},
	}
	updatedPR := &en.PullRequest{
		PullRequestID:     prID,
		Status:            en.StatusOpen,
		AuthorID:          "u1",
		AssignedReviewers: []string{"u4", "u3"},
	}

	mockStorage.EXPECT().GetPR(ctx, prID).Return(pr, nil).Once()
	mockStorage.EXPECT().IsUserAssignedToReviewer(ctx, prID, oldUserID).Return(true, nil).Once()
	mockStorage.EXPECT().GetUser(ctx, oldUserID).Return(oldUser, nil).Once()
	mockStorage.EXPECT().GetUsersByTeam(ctx, "backend", true).Return(candidates, nil).Once()
	mockStorage.EXPECT().ReassignReviewer(ctx, prID, oldUserID, mock.AnythingOfType("string")).Return(nil).Once()
	mockStorage.EXPECT().GetPR(ctx, prID).Return(updatedPR, nil).Once()

	result, newReviewerID, err := service.ReassignReviewer(ctx, prID, oldUserID)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.NotEmpty(t, newReviewerID)
}

func TestReassignReviewer_OnMergedPR(t *testing.T) {
	mockStorage := NewMockStorage(t)
	service := &ServiceStorage{storage: mockStorage}

	ctx := context.Background()
	prID := "pr-1"
	oldUserID := "u2"

	mergedPR := &en.PullRequest{
		PullRequestID: prID,
		Status:        en.StatusMerged,
		AuthorID:      "u1",
	}

	mockStorage.EXPECT().GetPR(ctx, prID).Return(mergedPR, nil).Once()

	result, newReviewerID, err := service.ReassignReviewer(ctx, prID, oldUserID)

	require.Error(t, err)
	assert.Nil(t, result)
	assert.Empty(t, newReviewerID)
	var appErr *en.AppError
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, en.ErrCodePRMerged, appErr.Code)
}

func TestReassignReviewer_NotAssigned(t *testing.T) {
	mockStorage := NewMockStorage(t)
	service := &ServiceStorage{storage: mockStorage}

	ctx := context.Background()
	prID := "pr-1"
	oldUserID := "u5"

	pr := &en.PullRequest{
		PullRequestID:     prID,
		Status:            en.StatusOpen,
		AuthorID:          "u1",
		AssignedReviewers: []string{"u2", "u3"},
	}

	mockStorage.EXPECT().GetPR(ctx, prID).Return(pr, nil).Once()
	mockStorage.EXPECT().IsUserAssignedToReviewer(ctx, prID, oldUserID).Return(false, nil).Once()

	result, newReviewerID, err := service.ReassignReviewer(ctx, prID, oldUserID)

	require.Error(t, err)
	assert.Nil(t, result)
	assert.Empty(t, newReviewerID)
	var appErr *en.AppError
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, en.ErrCodeNotAssigned, appErr.Code)
}

func TestReassignReviewer_NoCandidate(t *testing.T) {
	mockStorage := NewMockStorage(t)
	service := &ServiceStorage{storage: mockStorage}

	ctx := context.Background()
	prID := "pr-1"
	oldUserID := "u2"

	pr := &en.PullRequest{
		PullRequestID:     prID,
		Status:            en.StatusOpen,
		AuthorID:          "u1",
		AssignedReviewers: []string{oldUserID},
	}
	oldUser := &en.User{UserID: oldUserID, Username: "Bob", TeamName: "small-team", IsActive: true}

	mockStorage.EXPECT().GetPR(ctx, prID).Return(pr, nil).Once()
	mockStorage.EXPECT().IsUserAssignedToReviewer(ctx, prID, oldUserID).Return(true, nil).Once()
	mockStorage.EXPECT().GetUser(ctx, oldUserID).Return(oldUser, nil).Once()
	mockStorage.EXPECT().GetUsersByTeam(ctx, "small-team", true).Return([]*en.User{}, nil).Once()

	result, newReviewerID, err := service.ReassignReviewer(ctx, prID, oldUserID)

	require.Error(t, err)
	assert.Nil(t, result)
	assert.Empty(t, newReviewerID)
	var appErr *en.AppError
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, en.ErrCodeNoCandidate, appErr.Code)
}

func TestReassignReviewer_PRNotFound(t *testing.T) {
	mockStorage := NewMockStorage(t)
	service := &ServiceStorage{storage: mockStorage}

	ctx := context.Background()
	prID := "nonexistent"
	oldUserID := "u2"

	mockStorage.EXPECT().GetPR(ctx, prID).Return(nil, nil).Once()

	result, newReviewerID, err := service.ReassignReviewer(ctx, prID, oldUserID)

	require.Error(t, err)
	assert.Nil(t, result)
	assert.Empty(t, newReviewerID)
	var appErr *en.AppError
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, en.ErrCodeNotFound, appErr.Code)
}

func TestReassignReviewer_UserNotFound(t *testing.T) {
	mockStorage := NewMockStorage(t)
	service := &ServiceStorage{storage: mockStorage}

	ctx := context.Background()
	prID := "pr-1"
	oldUserID := "nonexistent"

	pr := &en.PullRequest{
		PullRequestID:     prID,
		Status:            en.StatusOpen,
		AuthorID:          "u1",
		AssignedReviewers: []string{"u2"},
	}

	mockStorage.EXPECT().GetPR(ctx, prID).Return(pr, nil).Once()
	mockStorage.EXPECT().IsUserAssignedToReviewer(ctx, prID, oldUserID).Return(true, nil).Once()
	mockStorage.EXPECT().GetUser(ctx, oldUserID).Return(nil, nil).Once()

	result, newReviewerID, err := service.ReassignReviewer(ctx, prID, oldUserID)

	require.Error(t, err)
	assert.Nil(t, result)
	assert.Empty(t, newReviewerID)
	var appErr *en.AppError
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, en.ErrCodeNotFound, appErr.Code)
}
