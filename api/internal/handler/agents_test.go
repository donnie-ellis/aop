package handler_test

import (
	"net/http"
	"testing"
)

func TestRegisterAgent_Success(t *testing.T) {
	resp := registerAgent(t, map[string]any{
		"name":     "agent-alpha",
		"address":  "http://10.0.0.1:7000",
		"labels":   map[string]string{"region": "us-east"},
		"capacity": 4,
	})
	assertStatus(t, resp, http.StatusCreated)

	var body map[string]any
	decodeJSON(t, resp, &body)
	if body["agent_id"] == "" {
		t.Fatal("expected agent_id in response")
	}
	if body["token"] == "" {
		t.Fatal("expected token in response (shown once)")
	}
}

func TestRegisterAgent_DefaultCapacity(t *testing.T) {
	// capacity omitted — handler should default to 1
	resp := registerAgent(t, map[string]any{
		"name":    "agent-no-capacity",
		"address": "http://10.0.0.3:7000",
	})
	assertStatus(t, resp, http.StatusCreated)
	resp.Body.Close()
}

func TestRegisterAgent_MissingName(t *testing.T) {
	resp := registerAgent(t, map[string]any{
		"address": "http://10.0.0.4:7000",
	})
	assertStatus(t, resp, http.StatusBadRequest)
	resp.Body.Close()
}

func TestRegisterAgent_MissingAddress(t *testing.T) {
	resp := registerAgent(t, map[string]any{
		"name": "nameless",
	})
	assertStatus(t, resp, http.StatusBadRequest)
	resp.Body.Close()
}

func TestRegisterAgent_WrongRegistrationToken(t *testing.T) {
	resp := doHeaders(t, http.MethodPost, "/agents/register", map[string]any{
		"name":    "sneaky",
		"address": "http://evil:7000",
	}, map[string]string{
		"X-Registration-Token": "wrong-token",
	})
	assertStatus(t, resp, http.StatusUnauthorized)
	resp.Body.Close()
}

func TestRegisterAgent_NoRegistrationToken(t *testing.T) {
	resp := do(t, http.MethodPost, "/agents/register", map[string]any{
		"name":    "no-token",
		"address": "http://10.0.0.5:7000",
	}, "") // no token header at all
	assertStatus(t, resp, http.StatusUnauthorized)
	resp.Body.Close()
}

func TestListAgents_RequiresAuth(t *testing.T) {
	resp := do(t, http.MethodGet, "/agents", nil, "")
	assertStatus(t, resp, http.StatusUnauthorized)
	resp.Body.Close()
}

func TestListAgents(t *testing.T) {
	resp := do(t, http.MethodGet, "/agents", nil, userJWT)
	assertStatus(t, resp, http.StatusOK)

	// Must be a JSON array (possibly empty).
	var agents []map[string]any
	decodeJSON(t, resp, &agents)
}

func TestGetAgent_Found(t *testing.T) {
	agent, _ := mustCreateAgent(t)

	resp := do(t, http.MethodGet, "/agents/"+agent.ID.String(), nil, userJWT)
	assertStatus(t, resp, http.StatusOK)

	var body map[string]any
	decodeJSON(t, resp, &body)
	if body["id"] != agent.ID.String() {
		t.Fatalf("expected id %s, got %v", agent.ID, body["id"])
	}
	// token_hash must never be returned to users.
	if _, ok := body["token_hash"]; ok {
		t.Fatal("token_hash must not appear in agent response")
	}
}

func TestGetAgent_NotFound(t *testing.T) {
	resp := do(t, http.MethodGet, "/agents/00000000-0000-0000-0000-000000000001", nil, userJWT)
	assertStatus(t, resp, http.StatusNotFound)
	resp.Body.Close()
}

func TestGetAgent_InvalidID(t *testing.T) {
	resp := do(t, http.MethodGet, "/agents/not-a-uuid", nil, userJWT)
	assertStatus(t, resp, http.StatusBadRequest)
	resp.Body.Close()
}

func TestAgentHeartbeat(t *testing.T) {
	agent, rawToken := mustCreateAgent(t)
	_ = agent

	resp := do(t, http.MethodPost, "/agent/heartbeat", map[string]any{
		"agent_id":     agent.ID.String(),
		"running_jobs": 1,
		"version":      "v0.1.0",
	}, rawToken)
	assertStatus(t, resp, http.StatusNoContent)
	resp.Body.Close()
}

func TestAgentHeartbeat_RequiresAgentToken(t *testing.T) {
	// A user JWT must not be accepted on the agent-only route.
	resp := do(t, http.MethodPost, "/agent/heartbeat", map[string]any{
		"running_jobs": 0,
	}, userJWT)
	assertStatus(t, resp, http.StatusUnauthorized)
	resp.Body.Close()
}

func TestAgentHeartbeat_NoToken(t *testing.T) {
	resp := do(t, http.MethodPost, "/agent/heartbeat", map[string]any{
		"running_jobs": 0,
	}, "")
	assertStatus(t, resp, http.StatusUnauthorized)
	resp.Body.Close()
}
