package integration

import (
    "bytes"
    "encoding/json"
    "net/http"
    "testing"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

// Server currently validates that user_ids is non-empty and returns 500 if empty.
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

    // Деактивируем единственного возможного ревьюера (u2) -> замены нет.
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