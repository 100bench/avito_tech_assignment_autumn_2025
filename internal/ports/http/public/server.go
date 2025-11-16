package public

import (
	"encoding/json"
	"net/http"

	"github.com/100bench/avito_tech_assignment_autumn_2025/internal/entities"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/pkg/errors"
)

type Server struct {
	service PRReviewService
	router  *chi.Mux
}

func NewServer(service PRReviewService) (*Server, error) {
	if service == nil {
		return nil, errors.Wrap(entities.ErrNilDependency, "public server service")
	}
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	s := &Server{
		service: service,
		router:  r,
	}
	s.setupRoutes()
	return s, nil
}

func (s *Server) GetRouter() *chi.Mux {
	return s.router
}

func (s *Server) setupRoutes() {
	s.router.Post("/team/add", s.handleCreateTeam)
	s.router.Get("/team/get", s.handleGetTeam)

	s.router.Post("/users/setIsActive", s.handleSetUserActive)
	s.router.Get("/users/getReview", s.handleGetUserReviews)

	s.router.Post("/pullRequest/create", s.handleCreatePR)
	s.router.Post("/pullRequest/merge", s.handleMergePR)
	s.router.Post("/pullRequest/reassign", s.handleReassignReviewer)

	s.router.Post("/team/deactivateMembers", s.handleDeactivateMembers)

	s.router.Get("/stats", s.handleGetStats)
}

type CreateTeamRequest struct {
	TeamName string                `json:"team_name"`
	Members  []entities.TeamMember `json:"members"`
}

type TeamResponse struct {
	TeamName string                `json:"team_name"`
	Members  []entities.TeamMember `json:"members"`
}

type CreateTeamResponse struct {
	Team TeamResponse `json:"team"`
}

type GetTeamResponse struct {
	Team TeamResponse `json:"team"`
}

func (s *Server) handleCreateTeam(w http.ResponseWriter, r *http.Request) {
	var req CreateTeamRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.respondWithError(w, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
		return
	}

	team, err := s.service.CreateTeam(r.Context(), req.TeamName, req.Members)
	if err != nil {
		s.handleError(w, err)
		return
	}

	resp := CreateTeamResponse{
		Team: TeamResponse{
			TeamName: team.TeamName,
			Members:  team.TeamMembers,
		},
	}
	s.respondWithJSON(w, http.StatusCreated, resp)
}

func (s *Server) handleGetTeam(w http.ResponseWriter, r *http.Request) {
	teamName := r.URL.Query().Get("team_name")
	if teamName == "" {
		s.respondWithError(w, http.StatusBadRequest, "INVALID_REQUEST", "missing team_name query parameter")
		return
	}

	team, err := s.service.GetTeam(r.Context(), teamName)
	if err != nil {
		s.handleError(w, err)
		return
	}

	resp := GetTeamResponse{
		Team: TeamResponse{
			TeamName: team.TeamName,
			Members:  team.TeamMembers,
		},
	}
	s.respondWithJSON(w, http.StatusOK, resp)
}

type SetUserActiveRequest struct {
	UserID   string `json:"user_id"`
	IsActive bool   `json:"is_active"`
}

type UserResponse struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	TeamName string `json:"team_name"`
	IsActive bool   `json:"is_active"`
}

type SetUserActiveResponse struct {
	User UserResponse `json:"user"`
}

func (s *Server) handleSetUserActive(w http.ResponseWriter, r *http.Request) {
	var req SetUserActiveRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.respondWithError(w, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
		return
	}

	user, err := s.service.SetUserActive(r.Context(), req.UserID, req.IsActive)
	if err != nil {
		s.handleError(w, err)
		return
	}

	resp := SetUserActiveResponse{
		User: UserResponse{
			UserID:   user.UserID,
			Username: user.Username,
			TeamName: user.TeamName,
			IsActive: user.IsActive,
		},
	}
	s.respondWithJSON(w, http.StatusOK, resp)
}

type UserReviewsResponse struct {
	UserID       string                      `json:"user_id"`
	PullRequests []entities.PullRequestShort `json:"pull_requests"`
}

func (s *Server) handleGetUserReviews(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		s.respondWithError(w, http.StatusBadRequest, "INVALID_REQUEST", "missing user_id query parameter")
		return
	}

	prs, err := s.service.GetUserReviews(r.Context(), userID)
	if err != nil {
		s.handleError(w, err)
		return
	}

	resp := UserReviewsResponse{
		UserID:       userID,
		PullRequests: make([]entities.PullRequestShort, 0, len(prs)),
	}
	for _, pr := range prs {
		resp.PullRequests = append(resp.PullRequests, *pr)
	}
	s.respondWithJSON(w, http.StatusOK, resp)
}

type CreatePRRequest struct {
	PullRequestID   string `json:"pull_request_id"`
	PullRequestName string `json:"pull_request_name"`
	AuthorID        string `json:"author_id"`
}

type PullRequestResponse struct {
	PullRequestID     string   `json:"pull_request_id"`
	PullRequestName   string   `json:"pull_request_name"`
	AuthorID          string   `json:"author_id"`
	Status            string   `json:"status"`
	AssignedReviewers []string `json:"assigned_reviewers"`
}

type CreatePRResponse struct {
	PR PullRequestResponse `json:"pr"`
}

func (s *Server) handleCreatePR(w http.ResponseWriter, r *http.Request) {
	var req CreatePRRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.respondWithError(w, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
		return
	}

	pr, err := s.service.CreatePullRequest(r.Context(), req.PullRequestID, req.PullRequestName, req.AuthorID)
	if err != nil {
		s.handleError(w, err)
		return
	}

	resp := CreatePRResponse{
		PR: PullRequestResponse{
			PullRequestID:     pr.PullRequestID,
			PullRequestName:   pr.PullRequestName,
			AuthorID:          pr.AuthorID,
			Status:            string(pr.Status),
			AssignedReviewers: pr.AssignedReviewers,
		},
	}
	s.respondWithJSON(w, http.StatusCreated, resp)
}

type MergePRRequest struct {
	PullRequestID string `json:"pull_request_id"`
}

type MergePRResponse struct {
	PR PullRequestResponse `json:"pr"`
}

func (s *Server) handleMergePR(w http.ResponseWriter, r *http.Request) {
	var req MergePRRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.respondWithError(w, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
		return
	}

	pr, err := s.service.MergePullRequest(r.Context(), req.PullRequestID)
	if err != nil {
		s.handleError(w, err)
		return
	}

	resp := MergePRResponse{
		PR: PullRequestResponse{
			PullRequestID:     pr.PullRequestID,
			PullRequestName:   pr.PullRequestName,
			AuthorID:          pr.AuthorID,
			Status:            string(pr.Status),
			AssignedReviewers: pr.AssignedReviewers,
		},
	}
	s.respondWithJSON(w, http.StatusOK, resp)
}

type ReassignReviewerRequest struct {
	PullRequestID string `json:"pull_request_id"`
	OldUserID     string `json:"old_user_id"`
}

type ReassignReviewerResponse struct {
	PR         PullRequestResponse `json:"pr"`
	ReplacedBy string              `json:"replaced_by"`
}

func (s *Server) handleReassignReviewer(w http.ResponseWriter, r *http.Request) {
	var req ReassignReviewerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.respondWithError(w, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
		return
	}

	pr, newReviewerID, err := s.service.ReassignReviewer(r.Context(), req.PullRequestID, req.OldUserID)
	if err != nil {
		s.handleError(w, err)
		return
	}

	resp := ReassignReviewerResponse{
		PR: PullRequestResponse{
			PullRequestID:     pr.PullRequestID,
			PullRequestName:   pr.PullRequestName,
			AuthorID:          pr.AuthorID,
			Status:            string(pr.Status),
			AssignedReviewers: pr.AssignedReviewers,
		},
		ReplacedBy: newReviewerID,
	}
	s.respondWithJSON(w, http.StatusOK, resp)
}

type ErrorResponse struct {
	Error ErrorDetail `json:"error"`
}

type ErrorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (s *Server) respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	response, err := json.Marshal(payload)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_, err = w.Write(response)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

func (s *Server) respondWithError(w http.ResponseWriter, code int, errCode string, message string) {
	resp := ErrorResponse{
		Error: ErrorDetail{
			Code:    errCode,
			Message: message,
		},
	}
	s.respondWithJSON(w, code, resp)
}

type deactivateMembersRequest struct {
    TeamName string   `json:"team_name"`
    UserIDs  []string `json:"user_ids"`
}

type DeactivateMembersResponse struct {
	DeactivatedUsers []string                      `json:"deactivated_users"`
	ReassignedPRs    []entities.PRReassignmentInfo `json:"reassigned_prs"`
}

func (s *Server) handleDeactivateMembers(w http.ResponseWriter, r *http.Request) {
    var req deactivateMembersRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        s.respondWithError(w, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
        return
    }

    // Новое поведение: пустой список — это валидный noop
    if len(req.UserIDs) == 0 {
        w.WriteHeader(http.StatusOK)
        _ = json.NewEncoder(w).Encode(map[string]any{
            "deactivated_users": []any{},
            "reassigned_prs":    []any{},
        })
        return
    }

    result, err := s.service.DeactivateTeamMembers(r.Context(), req.TeamName, req.UserIDs)
    if err != nil {
        s.handleError(w, err)
        return
    }

    deactivated := result.DeactivatedUsers
    if deactivated == nil {
        deactivated = []string{}
    }
    reassigned := result.Reassignments
    if reassigned == nil {
        reassigned = []entities.PRReassignmentInfo{}
    }

    resp := DeactivateMembersResponse{
        DeactivatedUsers: deactivated,
        ReassignedPRs:    reassigned,
    }
    s.respondWithJSON(w, http.StatusOK, resp)
}


func (s *Server) handleError(w http.ResponseWriter, err error) {
	if appErr, ok := err.(*entities.AppError); ok {
		statusCode := s.getHTTPStatusForError(appErr)
		s.respondWithError(w, statusCode, string(appErr.Code), appErr.Message)
	} else {
		s.respondWithError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
	}
}

func (s *Server) getHTTPStatusForError(appErr *entities.AppError) int {
	switch appErr.Code {
	case entities.ErrCodeTeamExists:
		return http.StatusBadRequest
	case entities.ErrCodePRExists:
		return http.StatusConflict
	case entities.ErrCodePRMerged, entities.ErrCodeNotAssigned, entities.ErrCodeNoCandidate:
		return http.StatusConflict
	case entities.ErrCodeInvalidTeamUser:
		return http.StatusConflict
	case entities.ErrCodeNotFound:
		return http.StatusNotFound
	default:
		return http.StatusInternalServerError
	}
}

func (s *Server) handleGetStats(w http.ResponseWriter, r *http.Request) {
	stats, err := s.service.GetStats(r.Context())
	if err != nil {
		s.handleError(w, err)
		return
	}

	s.respondWithJSON(w, http.StatusOK, stats)
}
