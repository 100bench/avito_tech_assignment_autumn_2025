package integration

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

//nolint:funlen
func TestCreatePullRequest(t *testing.T) {
	env := SetupTestEnv(t)
	defer env.Cleanup(t)

	teamPayload := map[string]interface{}{
		"team_name": "backend",
		"members": []map[string]interface{}{
			{"user_id": "author", "username": "Author", "is_active": true},
			{"user_id": "rev1", "username": "Reviewer1", "is_active": true},
			{"user_id": "rev2", "username": "Reviewer2", "is_active": true},
		},
	}
	teamData, _ := json.Marshal(teamPayload)
	resp, err := env.Client.Post(env.Server.URL+"/team/add", "application/json", bytes.NewBuffer(teamData))
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	defer func() {
		if err := resp.Body.Close(); err != nil {
			t.Logf("close body: %v", err)
		}
	}()

	t.Run("CreatePRWithAutoAssignment", func(t *testing.T) {
		payload := map[string]interface{}{
			"pull_request_id":   "pr-001",
			"pull_request_name": "Add feature X",
			"author_id":         "author",
		}
		data, _ := json.Marshal(payload)
		resp, err := env.Client.Post(env.Server.URL+"/pullRequest/create", "application/json", bytes.NewBuffer(data))
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, http.StatusCreated, resp.StatusCode)

		var result map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			t.Fatalf("decode response: %v", err)
		}

		pr := result["pr"].(map[string]interface{})
		assert.Equal(t, "pr-001", pr["pull_request_id"])
		assert.Equal(t, "Add feature X", pr["pull_request_name"])
		assert.Equal(t, "author", pr["author_id"])
		assert.Equal(t, "OPEN", pr["status"])

		reviewers := pr["assigned_reviewers"].([]interface{})
		assert.Len(t, reviewers, 2)

		for _, rev := range reviewers {
			assert.NotEqual(t, "author", rev)
		}
	})

	t.Run("CreatePRWithOnlyOneAvailableReviewer", func(t *testing.T) {
		teamPayload := map[string]interface{}{
			"team_name": "small-team",
			"members": []map[string]interface{}{
				{"user_id": "author2", "username": "Author2", "is_active": true},
				{"user_id": "rev3", "username": "Reviewer3", "is_active": true},
			},
		}
		teamData, _ := json.Marshal(teamPayload)
		resp, err := env.Client.Post(env.Server.URL+"/team/add", "application/json", bytes.NewBuffer(teamData))
		require.NoError(t, err)
		require.Equal(t, http.StatusCreated, resp.StatusCode)
		defer func() {
			if err := resp.Body.Close(); err != nil {
				t.Logf("close body: %v", err)
			}
		}()

		payload := map[string]interface{}{
			"pull_request_id":   "pr-002",
			"pull_request_name": "Fix bug Y",
			"author_id":         "author2",
		}
		data, _ := json.Marshal(payload)
		resp, err = env.Client.Post(env.Server.URL+"/pullRequest/create", "application/json", bytes.NewBuffer(data))
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, http.StatusCreated, resp.StatusCode)

		var result map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			t.Fatalf("decode response: %v", err)
		}

		pr := result["pr"].(map[string]interface{})
		reviewers := pr["assigned_reviewers"].([]interface{})
		assert.Len(t, reviewers, 1)
		assert.Equal(t, "rev3", reviewers[0])
	})

	t.Run("CreatePRWithNonExistentAuthor", func(t *testing.T) {
		payload := map[string]interface{}{
			"pull_request_id":   "pr-404",
			"pull_request_name": "Ghost PR",
			"author_id":         "nonexistent",
		}
		data, _ := json.Marshal(payload)
		resp, err := env.Client.Post(env.Server.URL+"/pullRequest/create", "application/json", bytes.NewBuffer(data))
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, http.StatusNotFound, resp.StatusCode)

		var result map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			t.Fatalf("decode response: %v", err)
		}

		errObj := result["error"].(map[string]interface{})
		assert.Equal(t, "NOT_FOUND", errObj["code"])
	})

	t.Run("CreateDuplicatePR", func(t *testing.T) {
		payload := map[string]interface{}{
			"pull_request_id":   "pr-001",
			"pull_request_name": "Duplicate",
			"author_id":         "author",
		}
		data, _ := json.Marshal(payload)
		resp, err := env.Client.Post(env.Server.URL+"/pullRequest/create", "application/json", bytes.NewBuffer(data))
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, http.StatusConflict, resp.StatusCode)

		var result map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			t.Fatalf("decode response: %v", err)
		}

		errObj := result["error"].(map[string]interface{})
		assert.Equal(t, "PR_EXISTS", errObj["code"])
	})

	t.Run("CreatePRWithNoActiveReviewers", func(t *testing.T) {
		teamPayload := map[string]interface{}{
			"team_name": "solo-team",
			"members": []map[string]interface{}{
				{"user_id": "solo-author", "username": "Solo", "is_active": true},
				{"user_id": "inactive1", "username": "Inactive1", "is_active": false},
			},
		}
		teamData, _ := json.Marshal(teamPayload)
		resp, err := env.Client.Post(env.Server.URL+"/team/add", "application/json", bytes.NewBuffer(teamData))
		require.NoError(t, err)
		_ = resp.Body.Close()

		payload := map[string]interface{}{
			"pull_request_id":   "pr-solo",
			"pull_request_name": "Solo PR",
			"author_id":         "solo-author",
		}
		data, _ := json.Marshal(payload)
		resp, err = env.Client.Post(env.Server.URL+"/pullRequest/create", "application/json", bytes.NewBuffer(data))
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, http.StatusCreated, resp.StatusCode)

		var result map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			t.Fatalf("decode response: %v", err)
		}
		pr := result["pr"].(map[string]interface{})
		reviewers := pr["assigned_reviewers"].([]interface{})
		assert.Empty(t, reviewers)
	})
}

//nolint:funlen
func TestMergePullRequest(t *testing.T) {
	env := SetupTestEnv(t)
	defer env.Cleanup(t)

	teamPayload := map[string]interface{}{
		"team_name": "qa-team",
		"members": []map[string]interface{}{
			{"user_id": "qa-author", "username": "QAAuthor", "is_active": true},
			{"user_id": "qa-rev", "username": "QAReviewer", "is_active": true},
		},
	}
	teamData, _ := json.Marshal(teamPayload)
	resp, err := env.Client.Post(env.Server.URL+"/team/add", "application/json", bytes.NewBuffer(teamData))
	require.NoError(t, err)
	_ = resp.Body.Close()

	prPayload := map[string]interface{}{
		"pull_request_id":   "pr-merge-1",
		"pull_request_name": "Ready to merge",
		"author_id":         "qa-author",
	}
	prData, _ := json.Marshal(prPayload)
	resp, err = env.Client.Post(env.Server.URL+"/pullRequest/create", "application/json", bytes.NewBuffer(prData))
	require.NoError(t, err)
	_ = resp.Body.Close()

	t.Run("MergePR", func(t *testing.T) {
		payload := map[string]interface{}{
			"pull_request_id": "pr-merge-1",
		}
		data, _ := json.Marshal(payload)
		resp, err := env.Client.Post(env.Server.URL+"/pullRequest/merge", "application/json", bytes.NewBuffer(data))
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var result map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			t.Fatalf("decode response: %v", err)
		}

		pr := result["pr"].(map[string]interface{})
		assert.Equal(t, "pr-merge-1", pr["pull_request_id"])
		assert.Equal(t, "MERGED", pr["status"])
	})

	t.Run("MergeAlreadyMergedPR_Idempotent", func(t *testing.T) {
		payload := map[string]interface{}{
			"pull_request_id": "pr-merge-1",
		}
		data, _ := json.Marshal(payload)
		resp, err := env.Client.Post(env.Server.URL+"/pullRequest/merge", "application/json", bytes.NewBuffer(data))
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var result map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			t.Fatalf("decode response: %v", err)
		}

		pr := result["pr"].(map[string]interface{})
		assert.Equal(t, "MERGED", pr["status"])
	})

	t.Run("MergeNonExistentPR", func(t *testing.T) {
		payload := map[string]interface{}{
			"pull_request_id": "pr-nonexistent",
		}
		data, _ := json.Marshal(payload)
		resp, err := env.Client.Post(env.Server.URL+"/pullRequest/merge", "application/json", bytes.NewBuffer(data))
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, http.StatusNotFound, resp.StatusCode)

		var result map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			t.Fatalf("decode response: %v", err)
		}

		errObj := result["error"].(map[string]interface{})
		assert.Equal(t, "NOT_FOUND", errObj["code"])
	})
}

//nolint:funlen
func TestReassignReviewer(t *testing.T) {
	env := SetupTestEnv(t)
	defer env.Cleanup(t)

	teamPayload := map[string]interface{}{
		"team_name": "big-team",
		"members": []map[string]interface{}{
			{"user_id": "big-author", "username": "BigAuthor", "is_active": true},
			{"user_id": "big-rev1", "username": "BigRev1", "is_active": true},
			{"user_id": "big-rev2", "username": "BigRev2", "is_active": true},
			{"user_id": "big-rev3", "username": "BigRev3", "is_active": true},
		},
	}
	teamData, _ := json.Marshal(teamPayload)
	resp, err := env.Client.Post(env.Server.URL+"/team/add", "application/json", bytes.NewBuffer(teamData))
	require.NoError(t, err)
	_ = resp.Body.Close()

	prPayload := map[string]interface{}{
		"pull_request_id":   "pr-reassign",
		"pull_request_name": "Need reassignment",
		"author_id":         "big-author",
	}
	prData, _ := json.Marshal(prPayload)
	resp, err = env.Client.Post(env.Server.URL+"/pullRequest/create", "application/json", bytes.NewBuffer(prData))
	require.NoError(t, err)

	var createResult map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&createResult); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	resp.Body.Close()

	pr := createResult["pr"].(map[string]interface{})
	reviewers := pr["assigned_reviewers"].([]interface{})
	firstReviewer := reviewers[0].(string)

	t.Run("ReassignReviewer", func(t *testing.T) {
		payload := map[string]interface{}{
			"pull_request_id": "pr-reassign",
			"old_user_id":     firstReviewer,
		}
		data, _ := json.Marshal(payload)
		resp, err := env.Client.Post(env.Server.URL+"/pullRequest/reassign", "application/json", bytes.NewBuffer(data))
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var result map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			t.Fatalf("decode response: %v", err)
		}

		pr := result["pr"].(map[string]interface{})
		newReviewers := pr["assigned_reviewers"].([]interface{})

		assert.Len(t, newReviewers, 2)

		for _, rev := range newReviewers {
			assert.NotEqual(t, firstReviewer, rev, "Old reviewer should not be in the list")
		}

		for _, rev := range newReviewers {
			assert.NotEqual(t, "big-author", rev, "Author should not be assigned as reviewer")
		}
	})

	t.Run("ReassignOnMergedPR", func(t *testing.T) {
		mergePayload := map[string]interface{}{
			"pull_request_id": "pr-reassign",
		}
		mergeData, _ := json.Marshal(mergePayload)
		resp, err := env.Client.Post(env.Server.URL+"/pullRequest/merge", "application/json", bytes.NewBuffer(mergeData))
		require.NoError(t, err)
		_ = resp.Body.Close()

		payload := map[string]interface{}{
			"pull_request_id": "pr-reassign",
			"old_user_id":     "big-rev3",
		}
		data, _ := json.Marshal(payload)
		resp, err = env.Client.Post(env.Server.URL+"/pullRequest/reassign", "application/json", bytes.NewBuffer(data))
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, http.StatusConflict, resp.StatusCode)

		var result map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			t.Fatalf("decode response: %v", err)
		}

		errObj := result["error"].(map[string]interface{})
		assert.Equal(t, "PR_MERGED", errObj["code"])
	})

	t.Run("ReassignNonAssignedReviewer", func(t *testing.T) {
		prPayload := map[string]interface{}{
			"pull_request_id":   "pr-reassign-2",
			"pull_request_name": "Another PR",
			"author_id":         "big-author",
		}
		prData, _ := json.Marshal(prPayload)
		resp, err := env.Client.Post(env.Server.URL+"/pullRequest/create", "application/json", bytes.NewBuffer(prData))
		require.NoError(t, err)
		_ = resp.Body.Close()

		payload := map[string]interface{}{
			"pull_request_id": "pr-reassign-2",
			"old_user_id":     "big-author",
		}
		data, _ := json.Marshal(payload)
		resp, err = env.Client.Post(env.Server.URL+"/pullRequest/reassign", "application/json", bytes.NewBuffer(data))
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, http.StatusConflict, resp.StatusCode)

		var result map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			t.Fatalf("decode response: %v", err)
		}

		errObj := result["error"].(map[string]interface{})
		assert.Equal(t, "NOT_ASSIGNED", errObj["code"])
	})

	t.Run("ReassignWithNoCandidate", func(t *testing.T) {
		teamPayload := map[string]interface{}{
			"team_name": "tiny-team",
			"members": []map[string]interface{}{
				{"user_id": "tiny-author", "username": "TinyAuthor", "is_active": true},
				{"user_id": "tiny-rev", "username": "TinyRev", "is_active": true},
			},
		}
		teamData, _ := json.Marshal(teamPayload)
		resp, err := env.Client.Post(env.Server.URL+"/team/add", "application/json", bytes.NewBuffer(teamData))
		require.NoError(t, err)
		_ = resp.Body.Close()

		prPayload := map[string]interface{}{
			"pull_request_id":   "pr-tiny",
			"pull_request_name": "Tiny PR",
			"author_id":         "tiny-author",
		}
		prData, _ := json.Marshal(prPayload)
		resp, err = env.Client.Post(env.Server.URL+"/pullRequest/create", "application/json", bytes.NewBuffer(prData))
		require.NoError(t, err)
		_ = resp.Body.Close()

		payload := map[string]interface{}{
			"pull_request_id": "pr-tiny",
			"old_user_id":     "tiny-rev",
		}
		data, _ := json.Marshal(payload)
		resp, err = env.Client.Post(env.Server.URL+"/pullRequest/reassign", "application/json", bytes.NewBuffer(data))
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, http.StatusConflict, resp.StatusCode)

		var result map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			t.Fatalf("decode response: %v", err)
		}
		errObj := result["error"].(map[string]interface{})
		assert.Equal(t, "NO_CANDIDATE", errObj["code"])
	})
}
