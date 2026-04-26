package handler_test

import (
	"net/http"
	"testing"
)

func TestCreateJobTemplate_Success(t *testing.T) {
	p := mustCreateProject(t)

	resp := do(t, http.MethodPost, "/job-templates", map[string]any{
		"name":       "deploy-app",
		"project_id": p.ID.String(),
		"playbook":   "deploy.yml",
	}, userJWT)
	assertStatus(t, resp, http.StatusCreated)

	var body map[string]any
	decodeJSON(t, resp, &body)
	if body["id"] == "" {
		t.Fatal("expected id in response")
	}
	if body["name"] != "deploy-app" {
		t.Fatalf("expected name deploy-app, got %v", body["name"])
	}
	if body["playbook"] != "deploy.yml" {
		t.Fatalf("expected playbook deploy.yml, got %v", body["playbook"])
	}
}

func TestCreateJobTemplate_MissingFields(t *testing.T) {
	p := mustCreateProject(t)
	cases := []map[string]any{
		{"project_id": p.ID.String(), "playbook": "x.yml"},          // missing name
		{"name": "x", "playbook": "x.yml"},                          // missing project_id
		{"name": "x", "project_id": p.ID.String()},                  // missing playbook
	}
	for _, body := range cases {
		resp := do(t, http.MethodPost, "/job-templates", body, userJWT)
		assertStatus(t, resp, http.StatusBadRequest)
		resp.Body.Close()
	}
}

func TestCreateJobTemplate_InvalidProjectID(t *testing.T) {
	resp := do(t, http.MethodPost, "/job-templates", map[string]any{
		"name":       "bad-proj",
		"project_id": "not-a-uuid",
		"playbook":   "x.yml",
	}, userJWT)
	assertStatus(t, resp, http.StatusBadRequest)
	resp.Body.Close()
}

func TestCreateJobTemplate_RequiresAuth(t *testing.T) {
	p := mustCreateProject(t)
	resp := do(t, http.MethodPost, "/job-templates", map[string]any{
		"name": "x", "project_id": p.ID.String(), "playbook": "x.yml",
	}, "")
	assertStatus(t, resp, http.StatusUnauthorized)
	resp.Body.Close()
}

func TestGetJobTemplate(t *testing.T) {
	p := mustCreateProject(t)
	tmpl := mustCreateTemplate(t, p.ID)

	resp := do(t, http.MethodGet, "/job-templates/"+tmpl.ID.String(), nil, userJWT)
	assertStatus(t, resp, http.StatusOK)

	var body map[string]any
	decodeJSON(t, resp, &body)
	if body["id"] != tmpl.ID.String() {
		t.Fatalf("expected id %s, got %v", tmpl.ID, body["id"])
	}
}

func TestGetJobTemplate_NotFound(t *testing.T) {
	resp := do(t, http.MethodGet, "/job-templates/00000000-0000-0000-0000-000000000020", nil, userJWT)
	assertStatus(t, resp, http.StatusNotFound)
	resp.Body.Close()
}

func TestListJobTemplates(t *testing.T) {
	p := mustCreateProject(t)
	mustCreateTemplate(t, p.ID)

	resp := do(t, http.MethodGet, "/job-templates", nil, userJWT)
	assertStatus(t, resp, http.StatusOK)

	var templates []map[string]any
	decodeJSON(t, resp, &templates)
	if len(templates) == 0 {
		t.Fatal("expected at least one template")
	}
}

func TestUpdateJobTemplate(t *testing.T) {
	p := mustCreateProject(t)
	tmpl := mustCreateTemplate(t, p.ID)

	resp := do(t, http.MethodPut, "/job-templates/"+tmpl.ID.String(), map[string]any{
		"name":     "updated-template",
		"playbook": "updated.yml",
		"default_extra_vars": map[string]any{"env": "staging"},
	}, userJWT)
	assertStatus(t, resp, http.StatusOK)

	var body map[string]any
	decodeJSON(t, resp, &body)
	if body["name"] != "updated-template" {
		t.Fatalf("expected updated-template, got %v", body["name"])
	}
	if body["playbook"] != "updated.yml" {
		t.Fatalf("expected updated.yml, got %v", body["playbook"])
	}
}

func TestUpdateJobTemplate_NotFound(t *testing.T) {
	resp := do(t, http.MethodPut, "/job-templates/00000000-0000-0000-0000-000000000021", map[string]any{
		"name": "x", "playbook": "x.yml",
	}, userJWT)
	assertStatus(t, resp, http.StatusNotFound)
	resp.Body.Close()
}

func TestDeleteJobTemplate(t *testing.T) {
	p := mustCreateProject(t)
	tmpl := mustCreateTemplate(t, p.ID)

	resp := do(t, http.MethodDelete, "/job-templates/"+tmpl.ID.String(), nil, userJWT)
	assertStatus(t, resp, http.StatusNoContent)
	resp.Body.Close()

	resp2 := do(t, http.MethodGet, "/job-templates/"+tmpl.ID.String(), nil, userJWT)
	assertStatus(t, resp2, http.StatusNotFound)
	resp2.Body.Close()
}
