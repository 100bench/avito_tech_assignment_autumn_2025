package integration

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDeactivateMembers_WithReassignment(t *testing.T) {
	env := SetupTestEnv(t)
	defer env.Cleanup(t)

	// Создаем команду с 4 участниками
	teamPayload := map[string]interface{}{
		"team_name": "deactivate-team",
		"members": []map[string]interface{}{
			{"user_id": "author1", "username": "Author1", "is_active": true},
			{"user_id": "rev1", "username": "Rev1", "is_active": true},
			{"user_id": "rev2", "username": "Rev2", "is_active": true},
			{"user_id": "rev3", "username": "Rev3", "is_active": true},
		},
	}
	teamData, _ := json.Marshal(teamPayload)
	resp, err := env.Client.Post(env.Server.URL+"/team/add", "application/json", bytes.NewBuffer(teamData))
	require.NoError(t, err)
	if err := resp.Body.Close(); err != nil {
		t.Logf("close body: %v", err)
	}

	// Создаем PR с rev1 и rev2 как ревьюверами
	prPayload := map[string]interface{}{
		"pull_request_id":   "pr-deactivate-1",
		"pull_request_name": "Test PR",
		"author_id":         "author1",
	}
	prData, _ := json.Marshal(prPayload)
	resp, err = env.Client.Post(env.Server.URL+"/pullRequest/create", "application/json", bytes.NewBuffer(prData))
	require.NoError(t, err)
	if err := resp.Body.Close(); err != nil {
		t.Logf("close body: %v", err)
	}

	// Деактивируем rev1
	deactivatePayload := map[string]interface{}{
		"team_name": "deactivate-team",
		"user_ids":  []string{"rev1"},
	}
	deactivateData, _ := json.Marshal(deactivatePayload)
	resp, err = env.Client.Post(env.Server.URL+"/team/deactivateMembers", "application/json", bytes.NewBuffer(deactivateData))
	require.NoError(t, err)
	defer func() {
		if err := resp.Body.Close(); err != nil {
			t.Logf("close body: %v", err)
		}
	}()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(t, err)

	deactivated := result["deactivated_users"].([]interface{})
	assert.Len(t, deactivated, 1)
	assert.Equal(t, "rev1", deactivated[0])

	reassigned := result["reassigned_prs"].([]interface{})
	assert.Len(t, reassigned, 1)

	reassignment := reassigned[0].(map[string]interface{})
	assert.Equal(t, "pr-deactivate-1", reassignment["pull_request_id"])
	assert.Equal(t, "rev1", reassignment["old_reviewer"])
	assert.NotEmpty(t, reassignment["new_reviewer"])
	assert.Contains(t, []string{"rev3"}, reassignment["new_reviewer"]) // rev2 уже назначен, author1 - автор
}

func TestDeactivateMembers_NoReplacementCandidates(t *testing.T) {
	env := SetupTestEnv(t)
	defer env.Cleanup(t)

	// Создаем команду с 2 участниками (автор + 1 ревьювер)
	teamPayload := map[string]interface{}{
		"team_name": "small-deactivate-team",
		"members": []map[string]interface{}{
			{"user_id": "small-author", "username": "SmallAuthor", "is_active": true},
			{"user_id": "small-rev", "username": "SmallRev", "is_active": true},
		},
	}
	teamData, _ := json.Marshal(teamPayload)
	resp, err := env.Client.Post(env.Server.URL+"/team/add", "application/json", bytes.NewBuffer(teamData))
	require.NoError(t, err)
	if err := resp.Body.Close(); err != nil {
		t.Logf("close body: %v", err)
	}

	// Создаем PR
	prPayload := map[string]interface{}{
		"pull_request_id":   "pr-small",
		"pull_request_name": "Small PR",
		"author_id":         "small-author",
	}
	prData, _ := json.Marshal(prPayload)
	resp, err = env.Client.Post(env.Server.URL+"/pullRequest/create", "application/json", bytes.NewBuffer(prData))
	require.NoError(t, err)
	if err := resp.Body.Close(); err != nil {
		t.Logf("close body: %v", err)
	}

	// Деактивируем единственного ревьювера
	deactivatePayload := map[string]interface{}{
		"team_name": "small-deactivate-team",
		"user_ids":  []string{"small-rev"},
	}
	deactivateData, _ := json.Marshal(deactivatePayload)
	resp, err = env.Client.Post(env.Server.URL+"/team/deactivateMembers", "application/json", bytes.NewBuffer(deactivateData))
	require.NoError(t, err)
	defer func() {
		if err := resp.Body.Close(); err != nil {
			t.Logf("close body: %v", err)
		}
	}()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(t, err)

	deactivated := result["deactivated_users"].([]interface{})
	assert.Len(t, deactivated, 1)

	reassigned := result["reassigned_prs"].([]interface{})
	assert.Len(t, reassigned, 1)

	reassignment := reassigned[0].(map[string]interface{})
	assert.Equal(t, "pr-small", reassignment["pull_request_id"])
	assert.Equal(t, "small-rev", reassignment["old_reviewer"])
	assert.Empty(t, reassignment["new_reviewer"]) // нет кандидатов - удалили без замены
}

func TestDeactivateMembers_UsersNotInTeam(t *testing.T) {
	env := SetupTestEnv(t)
	defer env.Cleanup(t)

	teamPayload := map[string]interface{}{
		"team_name": "test-team-deactivate",
		"members": []map[string]interface{}{
			{"user_id": "test-user", "username": "TestUser", "is_active": true},
		},
	}
	teamData, _ := json.Marshal(teamPayload)
	resp, err := env.Client.Post(env.Server.URL+"/team/add", "application/json", bytes.NewBuffer(teamData))
	require.NoError(t, err)
	resp.Body.Close()

	// Пытаемся деактивировать несуществующего пользователя
	deactivatePayload := map[string]interface{}{
		"team_name": "test-team-deactivate",
		"user_ids":  []string{"nonexistent-user"},
	}
	deactivateData, _ := json.Marshal(deactivatePayload)
	resp, err = env.Client.Post(env.Server.URL+"/team/deactivateMembers", "application/json", bytes.NewBuffer(deactivateData))
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
}
