package handler_test

import (
	"net/http"
	"testing"
)

func TestCreateProject_Success(t *testing.T) {
	resp := do(t, http.MethodPost, "/projects", map[string]any{
		"name":           "infra-repo",
		"repo_url":       "https://github.com/example/infra",
		"branch":         "main",
		"inventory_path": "inventory/production.ini",
	}, userJWT)
	assertStatus(t, resp, http.StatusCreated)

	var body map[string]any
	decodeJSON(t, resp, &body)
	if body["id"] == "" {
		t.Fatal("expected id in response")
	}
	if body["name"] != "infra-repo" {
		t.Fatalf("expected name infra-repo, got %v", body["name"])
	}
	if body["sync_status"] != "pending" {
		t.Fatalf("expected initial sync_status=pending, got %v", body["sync_status"])
	}
}

func TestCreateProject_DefaultBranch(t *testing.T) {
	resp := do(t, http.MethodPost, "/projects", map[string]any{
		"name":           "no-branch",
		"repo_url":       "https://github.com/example/repo2",
		"inventory_path": "inv/hosts",
		// branch omitted — should default to "main"
	}, userJWT)
	assertStatus(t, resp, http.StatusCreated)

	var body map[string]any
	decodeJSON(t, resp, &body)
	if body["branch"] != "main" {
		t.Fatalf("expected branch=main, got %v", body["branch"])
	}
}

func TestCreateProject_MissingRequiredFields(t *testing.T) {
	cases := []map[string]any{
		{"repo_url": "https://github.com/x", "inventory_path": "inv/hosts"},          // missing name
		{"name": "x", "inventory_path": "inv/hosts"},                                  // missing repo_url
		{"name": "x", "repo_url": "https://github.com/x"},                             // missing inventory_path
	}
	for _, body := range cases {
		resp := do(t, http.MethodPost, "/projects", body, userJWT)
		assertStatus(t, resp, http.StatusBadRequest)
		resp.Body.Close()
	}
}

func TestCreateProject_RequiresAuth(t *testing.T) {
	resp := do(t, http.MethodPost, "/projects", map[string]any{
		"name": "x", "repo_url": "https://github.com/x", "inventory_path": "inv",
	}, "")
	assertStatus(t, resp, http.StatusUnauthorized)
	resp.Body.Close()
}

func TestGetProject(t *testing.T) {
	p := mustCreateProject(t)

	resp := do(t, http.MethodGet, "/projects/"+p.ID.String(), nil, userJWT)
	assertStatus(t, resp, http.StatusOK)

	var body map[string]any
	decodeJSON(t, resp, &body)
	if body["id"] != p.ID.String() {
		t.Fatalf("expected id %s, got %v", p.ID, body["id"])
	}
}

func TestGetProject_NotFound(t *testing.T) {
	resp := do(t, http.MethodGet, "/projects/00000000-0000-0000-0000-000000000010", nil, userJWT)
	assertStatus(t, resp, http.StatusNotFound)
	resp.Body.Close()
}

func TestListProjects(t *testing.T) {
	mustCreateProject(t)

	resp := do(t, http.MethodGet, "/projects", nil, userJWT)
	assertStatus(t, resp, http.StatusOK)

	var projects []map[string]any
	decodeJSON(t, resp, &projects)
	if len(projects) == 0 {
		t.Fatal("expected at least one project in list")
	}
}

func TestUpdateProject(t *testing.T) {
	p := mustCreateProject(t)

	resp := do(t, http.MethodPut, "/projects/"+p.ID.String(), map[string]any{
		"name":           "updated-name",
		"repo_url":       "https://github.com/example/updated",
		"branch":         "develop",
		"inventory_path": "inventory/staging.ini",
	}, userJWT)
	assertStatus(t, resp, http.StatusOK)

	var body map[string]any
	decodeJSON(t, resp, &body)
	if body["name"] != "updated-name" {
		t.Fatalf("expected name updated-name, got %v", body["name"])
	}
}

func TestUpdateProject_NotFound(t *testing.T) {
	resp := do(t, http.MethodPut, "/projects/00000000-0000-0000-0000-000000000011", map[string]any{
		"name": "x", "repo_url": "https://github.com/x", "inventory_path": "inv",
	}, userJWT)
	assertStatus(t, resp, http.StatusNotFound)
	resp.Body.Close()
}

func TestDeleteProject(t *testing.T) {
	p := mustCreateProject(t)

	resp := do(t, http.MethodDelete, "/projects/"+p.ID.String(), nil, userJWT)
	assertStatus(t, resp, http.StatusNoContent)
	resp.Body.Close()

	resp2 := do(t, http.MethodGet, "/projects/"+p.ID.String(), nil, userJWT)
	assertStatus(t, resp2, http.StatusNotFound)
	resp2.Body.Close()
}

func TestSyncProject(t *testing.T) {
	p := mustCreateProject(t)

	resp := do(t, http.MethodPost, "/projects/"+p.ID.String()+"/sync", nil, userJWT)
	assertStatus(t, resp, http.StatusAccepted)
	resp.Body.Close()
}

func TestSyncProject_NotFound(t *testing.T) {
	resp := do(t, http.MethodPost, "/projects/00000000-0000-0000-0000-000000000012/sync", nil, userJWT)
	assertStatus(t, resp, http.StatusNotFound)
	resp.Body.Close()
}

func TestListInventoryHosts_Empty(t *testing.T) {
	p := mustCreateProject(t)

	resp := do(t, http.MethodGet, "/projects/"+p.ID.String()+"/inventory", nil, userJWT)
	assertStatus(t, resp, http.StatusOK)

	var hosts []map[string]any
	decodeJSON(t, resp, &hosts)
	// Empty inventory is valid.
	if hosts == nil {
		hosts = []map[string]any{}
	}
}
