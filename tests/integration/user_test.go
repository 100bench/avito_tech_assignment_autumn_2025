package integration

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSetUserIsActive(t *testing.T) {
	env := SetupTestEnv(t)
	defer env.Cleanup(t)

	teamPayload := map[string]interface{}{
		"team_name": "test-team",
		"members": []map[string]interface{}{
			{"user_id": "u1", "username": "Alice", "is_active": true},
		},
	}
	teamData, _ := json.Marshal(teamPayload)
	resp, err := env.Client.Post(env.Server.URL+"/team/add", "application/json", bytes.NewBuffer(teamData))
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	if err := resp.Body.Close(); err != nil {
		t.Logf("close body: %v", err)
	}

	payload := map[string]interface{}{"user_id": "u1", "is_active": false}
	data, _ := json.Marshal(payload)
	resp, err = env.Client.Post(env.Server.URL+"/users/setIsActive", "application/json", bytes.NewBuffer(data))
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(t, err)
	user := result["user"].(map[string]interface{})
	assert.False(t, user["is_active"].(bool))
}

func TestSetUserIsActive_NotFound(t *testing.T) {
	env := SetupTestEnv(t)
	defer env.Cleanup(t)

	payload := map[string]interface{}{"user_id": "nonexistent", "is_active": true}
	data, _ := json.Marshal(payload)
	resp, err := env.Client.Post(env.Server.URL+"/users/setIsActive", "application/json", bytes.NewBuffer(data))
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func TestGetUserReviews(t *testing.T) {
	env := SetupTestEnv(t)
	defer env.Cleanup(t)

	teamPayload := map[string]interface{}{
		"team_name": "dev-team",
		"members": []map[string]interface{}{
			{"user_id": "author1", "username": "Author", "is_active": true},
			{"user_id": "reviewer1", "username": "Reviewer", "is_active": true},
		},
	}
	teamData, _ := json.Marshal(teamPayload)
	resp, err := env.Client.Post(env.Server.URL+"/team/add", "application/json", bytes.NewBuffer(teamData))
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	if err := resp.Body.Close(); err != nil {
		t.Logf("close body: %v", err)
	}

	t.Run("GetReviewsForUserWithNoPRs", func(t *testing.T) {
		resp, err := env.Client.Get(env.Server.URL + "/users/getReview?user_id=reviewer1")
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var result map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&result)
		require.NoError(t, err)

		assert.Equal(t, "reviewer1", result["user_id"])
		prs := result["pull_requests"].([]interface{})
		assert.Empty(t, prs)
	})

	t.Run("GetReviewsAfterPRCreation", func(t *testing.T) {
		prPayload := map[string]interface{}{
			"pull_request_id":   "pr1",
			"pull_request_name": "Test PR",
			"author_id":         "author1",
		}
		prData, _ := json.Marshal(prPayload)
		resp, err := env.Client.Post(env.Server.URL+"/pullRequest/create", "application/json", bytes.NewBuffer(prData))
		require.NoError(t, err)
		require.Equal(t, http.StatusCreated, resp.StatusCode)
		resp.Body.Close()

		resp, err = env.Client.Get(env.Server.URL + "/users/getReview?user_id=reviewer1")
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var result map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&result)
		require.NoError(t, err)

		assert.Equal(t, "reviewer1", result["user_id"])
		prs := result["pull_requests"].([]interface{})
		assert.Len(t, prs, 1)

		pr := prs[0].(map[string]interface{})
		assert.Equal(t, "pr1", pr["pull_request_id"])
		assert.Equal(t, "Test PR", pr["pull_request_name"])
		assert.Equal(t, "OPEN", pr["status"])
	})
}
