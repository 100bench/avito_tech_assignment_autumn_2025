package integration

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDeactivateMembers_EmptyUserIDs_ShouldSucceed(t *testing.T) {
	env := SetupTestEnv(t)
	defer env.Cleanup(t)

	teamPayload := map[string]interface{}{
		"team_name": "empty-input-team",
		"members": []map[string]interface{}{
			{"user_id": "e1", "username": "E1", "is_active": true},
			{"user_id": "e2", "username": "E2", "is_active": true},
		},
	}
	teamData, _ := json.Marshal(teamPayload)
	resp, err := env.Client.Post(env.Server.URL+"/team/add", "application/json", bytes.NewBuffer(teamData))
	require.NoError(t, err)
	_ = resp.Body.Close()

	deactivatePayload := map[string]interface{}{
		"team_name": "empty-input-team",
		"user_ids":  []string{},
	}
	deactivateData, _ := json.Marshal(deactivatePayload)
	resp, err = env.Client.Post(env.Server.URL+"/team/deactivateMembers", "application/json", bytes.NewBuffer(deactivateData))
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))

	deactivated, ok := result["deactivated_users"].([]interface{})
	require.True(t, ok)
	assert.Len(t, deactivated, 0)

	reassigned, ok := result["reassigned_prs"].([]interface{})
	require.True(t, ok)
	assert.Len(t, reassigned, 0)
}

func TestDeactivateMembers_NoReplacementCandidates(t *testing.T) {
	env := SetupTestEnv(t)
	defer env.Cleanup(t)

	teamPayload := map[string]interface{}{
		"team_name": "no-cand-team",
		"members": []map[string]interface{}{
			{"user_id": "u1", "username": "U1", "is_active": true}, // автор
			{"user_id": "u2", "username": "U2", "is_active": true},
		},
	}
	data, _ := json.Marshal(teamPayload)
	resp, err := env.Client.Post(env.Server.URL+"/team/add", "application/json", bytes.NewBuffer(data))
	require.NoError(t, err)
	_ = resp.Body.Close()

	prPayload := map[string]string{
		"pull_request_id":   "pr-x",
		"pull_request_name": "PR X",
		"author_id":         "u1",
	}
	prData, _ := json.Marshal(prPayload)
	resp, err = env.Client.Post(env.Server.URL+"/pullRequest/create", "application/json", bytes.NewBuffer(prData))
	require.NoError(t, err)
	_ = resp.Body.Close()

	deactPayload := map[string]interface{}{
		"team_name": "no-cand-team",
		"user_ids":  []string{"u2"},
	}
	deactData, _ := json.Marshal(deactPayload)
	resp, err = env.Client.Post(env.Server.URL+"/team/deactivateMembers", "application/json", bytes.NewBuffer(deactData))
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode)
	var result map[string]interface{}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))

	reassigned := result["reassigned_prs"].([]interface{})
	require.Len(t, reassigned, 1)
	entry := reassigned[0].(map[string]interface{})
	require.Equal(t, "u2", entry["old_reviewer"])
	assert.Equal(t, "", entry["new_reviewer"])
}

func TestDeactivateMembers_MultiplePRs(t *testing.T) {
	env := SetupTestEnv(t)
	defer env.Cleanup(t)

	teamPayload := map[string]interface{}{
		"team_name": "multi-pr-team",
		"members": []map[string]interface{}{
			{"user_id": "m1", "username": "M1", "is_active": true},
			{"user_id": "m2", "username": "M2", "is_active": true},
			{"user_id": "m3", "username": "M3", "is_active": true},
			{"user_id": "m4", "username": "M4", "is_active": true},
		},
	}
	data, _ := json.Marshal(teamPayload)
	resp, err := env.Client.Post(env.Server.URL+"/team/add", "application/json", bytes.NewBuffer(data))
	require.NoError(t, err)
	_ = resp.Body.Close()

	for _, prID := range []string{"mpr-1", "mpr-2"} {
		prPayload := map[string]string{
			"pull_request_id":   prID,
			"pull_request_name": prID + " name",
			"author_id":         "m1",
		}
		prData, _ := json.Marshal(prPayload)
		resp, err = env.Client.Post(env.Server.URL+"/pullRequest/create", "application/json", bytes.NewBuffer(prData))
		require.NoError(t, err)
		_ = resp.Body.Close()
	}

	deactPayload := map[string]interface{}{
		"team_name": "multi-pr-team",
		"user_ids":  []string{"m2", "m3"},
	}
	deactData, _ := json.Marshal(deactPayload)
	resp, err = env.Client.Post(env.Server.URL+"/team/deactivateMembers", "application/json", bytes.NewBuffer(deactData))
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))

	deactivated := result["deactivated_users"].([]interface{})
	require.Len(t, deactivated, 2)

	reassigned := result["reassigned_prs"].([]interface{})

	require.GreaterOrEqual(t, len(reassigned), 2)
	for _, raw := range reassigned {
		entry := raw.(map[string]interface{})
		require.NotEmpty(t, entry["old_reviewer"])
		require.NotNil(t, entry["new_reviewer"])
	}
}

func TestDeactivateMembers_UserNotFound_Should404(t *testing.T) {
	env := SetupTestEnv(t)
	defer env.Cleanup(t)

	teamPayload := map[string]interface{}{
		"team_name": "nf-team",
		"members": []map[string]interface{}{
			{"user_id": "a1", "username": "A1", "is_active": true},
		},
	}
	data, _ := json.Marshal(teamPayload)
	resp, err := env.Client.Post(env.Server.URL+"/team/add", "application/json", bytes.NewBuffer(data))
	require.NoError(t, err)
	_ = resp.Body.Close()

	deactPayload := map[string]interface{}{
		"team_name": "nf-team",
		"user_ids":  []string{"u99"}, // несуществующий пользователь
	}
	deactData, _ := json.Marshal(deactPayload)
	resp, err = env.Client.Post(env.Server.URL+"/team/deactivateMembers", "application/json", bytes.NewBuffer(deactData))
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusNotFound, resp.StatusCode)

	var errResp map[string]interface{}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&errResp))
	errObj := errResp["error"].(map[string]interface{})
	require.Equal(t, "NOT_FOUND", errObj["code"])
	require.NotEmpty(t, errObj["message"])
}

func TestDeactivateMembers_UserFromOtherTeam_Should409(t *testing.T) {
	env := SetupTestEnv(t)
	defer env.Cleanup(t)

	// team t1
	team1 := map[string]interface{}{
		"team_name": "t1",
		"members": []map[string]interface{}{
			{"user_id": "t1a", "username": "T1A", "is_active": true},
			{"user_id": "t1b", "username": "T1B", "is_active": true},
		},
	}
	// team t2
	team2 := map[string]interface{}{
		"team_name": "t2",
		"members": []map[string]interface{}{
			{"user_id": "t2a", "username": "T2A", "is_active": true},
		},
	}
	for _, p := range []map[string]interface{}{team1, team2} {
		b, _ := json.Marshal(p)
		resp, err := env.Client.Post(env.Server.URL+"/team/add", "application/json", bytes.NewBuffer(b))
		require.NoError(t, err)
		_ = resp.Body.Close()
	}

	deactPayload := map[string]interface{}{
		"team_name": "t1",
		"user_ids":  []string{"t2a"}, // пользователь из другой команды
	}
	deactData, _ := json.Marshal(deactPayload)
	resp, err := env.Client.Post(env.Server.URL+"/team/deactivateMembers", "application/json", bytes.NewBuffer(deactData))
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusConflict, resp.StatusCode)

	var errResp map[string]interface{}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&errResp))
	errObj := errResp["error"].(map[string]interface{})
	require.Equal(t, "INVALID_TEAM_USER", errObj["code"])
	require.NotEmpty(t, errObj["message"])
}

func TestDeactivateMembers_UserAlreadyInactive_Should409(t *testing.T) {
	env := SetupTestEnv(t)
	defer env.Cleanup(t)

	teamPayload := map[string]interface{}{
		"team_name": "inactive-team",
		"members": []map[string]interface{}{
			{"user_id": "ia1", "username": "IA1", "is_active": true},
			{"user_id": "ia2", "username": "IA2", "is_active": false}, // уже неактивен
		},
	}
	b, _ := json.Marshal(teamPayload)
	resp, err := env.Client.Post(env.Server.URL+"/team/add", "application/json", bytes.NewBuffer(b))
	require.NoError(t, err)
	_ = resp.Body.Close()

	deactPayload := map[string]interface{}{
		"team_name": "inactive-team",
		"user_ids":  []string{"ia2"}, // уже неактивный участник
	}
	deactData, _ := json.Marshal(deactPayload)
	resp, err = env.Client.Post(env.Server.URL+"/team/deactivateMembers", "application/json", bytes.NewBuffer(deactData))
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusConflict, resp.StatusCode)

	var errResp map[string]interface{}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&errResp))
	errObj := errResp["error"].(map[string]interface{})
	require.Equal(t, "INVALID_TEAM_USER", errObj["code"])
	require.NotEmpty(t, errObj["message"])
}

func TestDeactivateMembers_RepeatDeactivation_Should409(t *testing.T) {
	env := SetupTestEnv(t)
	defer env.Cleanup(t)

	teamPayload := map[string]interface{}{
		"team_name": "repeat-team",
		"members": []map[string]interface{}{
			{"user_id": "r1", "username": "R1", "is_active": true}, // автор
			{"user_id": "r2", "username": "R2", "is_active": true}, // будет деактивирован
		},
	}
	b, _ := json.Marshal(teamPayload)
	resp, err := env.Client.Post(env.Server.URL+"/team/add", "application/json", bytes.NewBuffer(b))
	require.NoError(t, err)
	_ = resp.Body.Close()

	// Первый вызов — успешная деактивация
	deact := map[string]interface{}{"team_name": "repeat-team", "user_ids": []string{"r2"}}
	db1, _ := json.Marshal(deact)
	resp, err = env.Client.Post(env.Server.URL+"/team/deactivateMembers", "application/json", bytes.NewBuffer(db1))
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	_ = resp.Body.Close()

	// Повтор — должен вернуть 409 INVALID_TEAM_USER
	db2, _ := json.Marshal(deact)
	resp, err = env.Client.Post(env.Server.URL+"/team/deactivateMembers", "application/json", bytes.NewBuffer(db2))
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusConflict, resp.StatusCode)
	var errResp map[string]interface{}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&errResp))
	errObj := errResp["error"].(map[string]interface{})
	require.Equal(t, "INVALID_TEAM_USER", errObj["code"])
	require.NotEmpty(t, errObj["message"])
}