package entities

import (
	"fmt"
)

var (
	ErrNilDependency = fmt.Errorf("nil dependency provided")
)

type ErrorCode string

const (
	ErrCodeTeamExists       ErrorCode = "TEAM_EXISTS"
	ErrCodePRExists         ErrorCode = "PR_EXISTS"
	ErrCodePRMerged         ErrorCode = "PR_MERGED"
	ErrCodeNotAssigned      ErrorCode = "NOT_ASSIGNED"
	ErrCodeNoCandidate      ErrorCode = "NO_CANDIDATE"
	ErrCodeNotFound         ErrorCode = "NOT_FOUND"
	ErrCodeInvalidTeamUser  ErrorCode = "INVALID_TEAM_USER"
)

type AppError struct {
	Code    ErrorCode
	Message string
}

func (e *AppError) Error() string {
	return e.Message
}

func NewTeamExistsError(teamName string) *AppError {
	return &AppError{
		Code:    ErrCodeTeamExists,
		Message: fmt.Sprintf("team '%s' already exists", teamName),
	}
}

func NewPRExistsError(prID string) *AppError {
	return &AppError{
		Code:    ErrCodePRExists,
		Message: fmt.Sprintf("PR '%s' already exists", prID),
	}
}

func NewPRMergedError(prID string) *AppError {
	return &AppError{
		Code:    ErrCodePRMerged,
		Message: "cannot modify merged PR",
	}
}

func NewNotAssignedError(userID, prID string) *AppError {
	return &AppError{
		Code:    ErrCodeNotAssigned,
		Message: fmt.Sprintf("user '%s' is not assigned to PR '%s'", userID, prID),
	}
}

func NewNoCandidateError(teamName string) *AppError {
	return &AppError{
		Code:    ErrCodeNoCandidate,
		Message: fmt.Sprintf("no active candidates in team '%s'", teamName),
	}
}

func NewNotFoundError(resource, id string) *AppError {
	return &AppError{
		Code:    ErrCodeNotFound,
		Message: fmt.Sprintf("%s '%s' not found", resource, id),
	}
}

func NewInvalidTeamUserError(userID, teamName string, reason string) *AppError {
	return &AppError{
		Code:    ErrCodeInvalidTeamUser,
		Message: fmt.Sprintf("user '%s' %s team '%s'", userID, reason, teamName),
	}
}
