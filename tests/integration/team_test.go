package integration

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/100bench/avito_tech_assignment_autumn_2025/internal/entities"
)

type CreateTeamRequest struct {
	TeamName string                `json:"team_name"`
	Members  []entities.TeamMember `json:"members"`
}

type CreateTeamResponse struct {
	Team struct {
		TeamName string                `json:"team_name"`
		Members  []entities.TeamMember `json:"members"`
	} `json:"team"`
}

type ErrorResponse struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

func TestCreateTeam_Success(t *testing.T) {
	env := SetupTestEnv(t)
	defer env.Cleanup(t)

	req := CreateTeamRequest{
		TeamName: "backend",
		Members: []entities.TeamMember{
			{UserID: "u1", Username: "Alice", IsActive: true},
			{UserID: "u2", Username: "Bob", IsActive: true},
			{UserID: "u3", Username: "Charlie", IsActive: true},
		},
	}

	body, err := json.Marshal(req)
	require.NoError(t, err)

	resp, err := env.Client.Post(
		env.Server.URL+"/team/add",
		"application/json",
		bytes.NewReader(body),
	)
	require.NoError(t, err)
	defer func() {
		if err := resp.Body.Close(); err != nil {
			t.Logf("close body: %v", err)
		}
	}()

	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	var result CreateTeamResponse
	err = json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(t, err)

	assert.Equal(t, "backend", result.Team.TeamName)
	assert.Len(t, result.Team.Members, 3)
	assert.Equal(t, "u1", result.Team.Members[0].UserID)
	assert.Equal(t, "Alice", result.Team.Members[0].Username)
	assert.True(t, result.Team.Members[0].IsActive)
}

func TestCreateTeam_AlreadyExists(t *testing.T) {
	env := SetupTestEnv(t)
	defer env.Cleanup(t)

	req := CreateTeamRequest{
		TeamName: "backend",
		Members: []entities.TeamMember{
			{UserID: "u1", Username: "Alice", IsActive: true},
		},
	}

	body, err := json.Marshal(req)
	require.NoError(t, err)

	resp1, err := env.Client.Post(
		env.Server.URL+"/team/add",
		"application/json",
		bytes.NewReader(body),
	)
	require.NoError(t, err)
	if err := resp1.Body.Close(); err != nil {
		t.Logf("close body: %v", err)
	}

	resp2, err := env.Client.Post(
		env.Server.URL+"/team/add",
		"application/json",
		bytes.NewReader(body),
	)
	require.NoError(t, err)
	defer func() {
		if err := resp2.Body.Close(); err != nil {
			t.Logf("close body: %v", err)
		}
	}()

	assert.Equal(t, http.StatusBadRequest, resp2.StatusCode)

	var errResp ErrorResponse
	err = json.NewDecoder(resp2.Body).Decode(&errResp)
	require.NoError(t, err)

	assert.Equal(t, "TEAM_EXISTS", errResp.Error.Code)
}

func TestCreateTeam_UpsertUsers(t *testing.T) {
	env := SetupTestEnv(t)
	defer env.Cleanup(t)

	req1 := CreateTeamRequest{
		TeamName: "frontend",
		Members: []entities.TeamMember{
			{UserID: "u1", Username: "Alice", IsActive: true},
			{UserID: "u2", Username: "Bob", IsActive: true},
		},
	}

	body1, err := json.Marshal(req1)
	require.NoError(t, err)

	resp1, err := env.Client.Post(
		env.Server.URL+"/team/add",
		"application/json",
		bytes.NewReader(body1),
	)
	require.NoError(t, err)
	if err := resp1.Body.Close(); err != nil {
		t.Logf("close body: %v", err)
	}

	req2 := CreateTeamRequest{
		TeamName: "backend",
		Members: []entities.TeamMember{
			{UserID: "u1", Username: "Alice", IsActive: false},
			{UserID: "u3", Username: "Charlie", IsActive: true},
		},
	}

	body2, err := json.Marshal(req2)
	require.NoError(t, err)

	resp2, err := env.Client.Post(
		env.Server.URL+"/team/add",
		"application/json",
		bytes.NewReader(body2),
	)
	require.NoError(t, err)
	defer func() {
		if err := resp2.Body.Close(); err != nil {
			t.Logf("close body: %v", err)
		}
	}()

	assert.Equal(t, http.StatusCreated, resp2.StatusCode)

	resp4, err := env.Client.Get(env.Server.URL + "/team/get?team_name=frontend")
	require.NoError(t, err)
	defer func() {
		if err := resp4.Body.Close(); err != nil {
			t.Logf("close body: %v", err)
		}
	}()

	var frontendResult CreateTeamResponse
	err = json.NewDecoder(resp4.Body).Decode(&frontendResult)
	require.NoError(t, err)

	assert.Len(t, frontendResult.Team.Members, 1)
	assert.Equal(t, "u2", frontendResult.Team.Members[0].UserID)

	resp3, err := env.Client.Get(env.Server.URL + "/team/get?team_name=backend")
	require.NoError(t, err)
	defer func() {
		if err := resp3.Body.Close(); err != nil {
			t.Logf("close body: %v", err)
		}
	}()

	var result CreateTeamResponse
	err = json.NewDecoder(resp3.Body).Decode(&result)
	require.NoError(t, err)

	assert.Len(t, result.Team.Members, 2)

	var u1Found bool
	for _, m := range result.Team.Members {
		if m.UserID == "u1" {
			u1Found = true
			assert.False(t, m.IsActive)
		}
	}
	assert.True(t, u1Found)
}

func TestGetTeam_Success(t *testing.T) {
	env := SetupTestEnv(t)
	defer env.Cleanup(t)

	req := CreateTeamRequest{
		TeamName: "backend",
		Members: []entities.TeamMember{
			{UserID: "u1", Username: "Alice", IsActive: true},
			{UserID: "u2", Username: "Bob", IsActive: true},
		},
	}

	body, err := json.Marshal(req)
	require.NoError(t, err)

	resp1, err := env.Client.Post(
		env.Server.URL+"/team/add",
		"application/json",
		bytes.NewReader(body),
	)
	require.NoError(t, err)
	if err := resp1.Body.Close(); err != nil {
		t.Logf("close body: %v", err)
	}

	resp2, err := env.Client.Get(env.Server.URL + "/team/get?team_name=backend")
	require.NoError(t, err)
	defer func() {
		if err := resp2.Body.Close(); err != nil {
			t.Logf("close body: %v", err)
		}
	}()

	assert.Equal(t, http.StatusOK, resp2.StatusCode)

	var result CreateTeamResponse
	err = json.NewDecoder(resp2.Body).Decode(&result)
	require.NoError(t, err)

	assert.Equal(t, "backend", result.Team.TeamName)
	assert.Len(t, result.Team.Members, 2)
}

func TestGetTeam_NotFound(t *testing.T) {
	env := SetupTestEnv(t)
	defer env.Cleanup(t)

	resp, err := env.Client.Get(env.Server.URL + "/team/get?team_name=nonexistent")
	require.NoError(t, err)
	defer func() {
		if err := resp.Body.Close(); err != nil {
			t.Logf("close body: %v", err)
		}
	}()

	assert.Equal(t, http.StatusNotFound, resp.StatusCode)

	var errResp ErrorResponse
	err = json.NewDecoder(resp.Body).Decode(&errResp)
	require.NoError(t, err)

	assert.Equal(t, "NOT_FOUND", errResp.Error.Code)
}
