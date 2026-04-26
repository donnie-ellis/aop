package handler_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/donnie-ellis/aop/pkg/types"
	"github.com/google/uuid"
)

func mustCreateJob(t *testing.T, templateID uuid.UUID) *types.Job {
	t.Helper()
	job, err := testStore.CreateJob(context.Background(), templateID, nil, nil)
	if err != nil {
		t.Fatalf("create job fixture: %v", err)
	}
	return job
}

func TestCreateJob_Success(t *testing.T) {
	p := mustCreateProject(t)
	tmpl := mustCreateTemplate(t, p.ID)

	resp := do(t, http.MethodPost, "/jobs", map[string]any{
		"template_id": tmpl.ID.String(),
		"extra_vars":  map[string]any{"target": "web01"},
	}, userJWT)
	assertStatus(t, resp, http.StatusCreated)

	var body map[string]any
	decodeJSON(t, resp, &body)
	if body["id"] == "" {
		t.Fatal("expected id in response")
	}
	if body["status"] != "pending" {
		t.Fatalf("expected initial status=pending, got %v", body["status"])
	}
	if body["template_id"] != tmpl.ID.String() {
		t.Fatalf("expected template_id %s, got %v", tmpl.ID, body["template_id"])
	}
}

func TestCreateJob_InvalidTemplateID(t *testing.T) {
	resp := do(t, http.MethodPost, "/jobs", map[string]any{
		"template_id": "not-a-uuid",
	}, userJWT)
	assertStatus(t, resp, http.StatusBadRequest)
	resp.Body.Close()
}

func TestCreateJob_TemplateNotFound(t *testing.T) {
	resp := do(t, http.MethodPost, "/jobs", map[string]any{
		"template_id": "00000000-0000-0000-0000-000000000040",
	}, userJWT)
	assertStatus(t, resp, http.StatusNotFound)
	resp.Body.Close()
}

func TestCreateJob_RequiresAuth(t *testing.T) {
	p := mustCreateProject(t)
	tmpl := mustCreateTemplate(t, p.ID)
	resp := do(t, http.MethodPost, "/jobs", map[string]any{
		"template_id": tmpl.ID.String(),
	}, "")
	assertStatus(t, resp, http.StatusUnauthorized)
	resp.Body.Close()
}

func TestGetJob(t *testing.T) {
	p := mustCreateProject(t)
	tmpl := mustCreateTemplate(t, p.ID)
	job := mustCreateJob(t, tmpl.ID)

	resp := do(t, http.MethodGet, "/jobs/"+job.ID.String(), nil, userJWT)
	assertStatus(t, resp, http.StatusOK)

	var body map[string]any
	decodeJSON(t, resp, &body)
	if body["id"] != job.ID.String() {
		t.Fatalf("expected id %s, got %v", job.ID, body["id"])
	}
}

func TestGetJob_NotFound(t *testing.T) {
	resp := do(t, http.MethodGet, "/jobs/00000000-0000-0000-0000-000000000041", nil, userJWT)
	assertStatus(t, resp, http.StatusNotFound)
	resp.Body.Close()
}

func TestListJobs_All(t *testing.T) {
	p := mustCreateProject(t)
	tmpl := mustCreateTemplate(t, p.ID)
	mustCreateJob(t, tmpl.ID)

	resp := do(t, http.MethodGet, "/jobs", nil, userJWT)
	assertStatus(t, resp, http.StatusOK)

	var jobs []map[string]any
	decodeJSON(t, resp, &jobs)
	if len(jobs) == 0 {
		t.Fatal("expected at least one job")
	}
}

func TestListJobs_FilterByStatus(t *testing.T) {
	p := mustCreateProject(t)
	tmpl := mustCreateTemplate(t, p.ID)
	mustCreateJob(t, tmpl.ID) // status defaults to pending

	resp := do(t, http.MethodGet, "/jobs?status=pending", nil, userJWT)
	assertStatus(t, resp, http.StatusOK)

	var jobs []map[string]any
	decodeJSON(t, resp, &jobs)
	for _, j := range jobs {
		if j["status"] != "pending" {
			t.Fatalf("filter by status=pending returned job with status=%v", j["status"])
		}
	}
}

func TestListJobs_FilterByTemplate(t *testing.T) {
	p := mustCreateProject(t)
	tmpl := mustCreateTemplate(t, p.ID)
	mustCreateJob(t, tmpl.ID)

	resp := do(t, http.MethodGet, "/jobs?template_id="+tmpl.ID.String(), nil, userJWT)
	assertStatus(t, resp, http.StatusOK)

	var jobs []map[string]any
	decodeJSON(t, resp, &jobs)
	for _, j := range jobs {
		if j["template_id"] != tmpl.ID.String() {
			t.Fatalf("filter by template_id returned wrong template: %v", j["template_id"])
		}
	}
}

func TestCancelJob(t *testing.T) {
	p := mustCreateProject(t)
	tmpl := mustCreateTemplate(t, p.ID)
	job := mustCreateJob(t, tmpl.ID)

	resp := do(t, http.MethodPost, "/jobs/"+job.ID.String()+"/cancel", nil, userJWT)
	assertStatus(t, resp, http.StatusNoContent)
	resp.Body.Close()

	// Confirm status is cancelled.
	get := do(t, http.MethodGet, "/jobs/"+job.ID.String(), nil, userJWT)
	assertStatus(t, get, http.StatusOK)
	var body map[string]any
	decodeJSON(t, get, &body)
	if body["status"] != "cancelled" {
		t.Fatalf("expected status=cancelled after cancel, got %v", body["status"])
	}
}

func TestCancelJob_AlreadyTerminal(t *testing.T) {
	p := mustCreateProject(t)
	tmpl := mustCreateTemplate(t, p.ID)
	job := mustCreateJob(t, tmpl.ID)

	resp := do(t, http.MethodPost, "/jobs/"+job.ID.String()+"/cancel", nil, userJWT)
	assertStatus(t, resp, http.StatusNoContent)
	resp.Body.Close()

	// Second cancel — job is terminal, expect 409.
	resp2 := do(t, http.MethodPost, "/jobs/"+job.ID.String()+"/cancel", nil, userJWT)
	assertStatus(t, resp2, http.StatusConflict)
	resp2.Body.Close()
}

func TestGetJobLogs_Empty(t *testing.T) {
	p := mustCreateProject(t)
	tmpl := mustCreateTemplate(t, p.ID)
	job := mustCreateJob(t, tmpl.ID)

	resp := do(t, http.MethodGet, "/jobs/"+job.ID.String()+"/logs", nil, userJWT)
	assertStatus(t, resp, http.StatusOK)

	var lines []map[string]any
	decodeJSON(t, resp, &lines)
}

func TestAgentPostLogs(t *testing.T) {
	p := mustCreateProject(t)
	tmpl := mustCreateTemplate(t, p.ID)
	agent, rawToken := mustCreateAgent(t)

	job := mustCreateJob(t, tmpl.ID)
	if err := testStore.UpdateJobStatus(context.Background(), job.ID, types.JobStatusRunning, &agent.ID, nil); err != nil {
		t.Fatalf("assign job to agent: %v", err)
	}

	logs := []map[string]any{
		{"seq": 1, "line": "PLAY [all] *****", "stream": "stdout"},
		{"seq": 2, "line": "TASK [ping] ****", "stream": "stdout"},
	}
	resp := do(t, http.MethodPost, "/agent/jobs/"+job.ID.String()+"/logs", logs, rawToken)
	assertStatus(t, resp, http.StatusNoContent)
	resp.Body.Close()

	// Fetch back as user and verify ordering.
	get := do(t, http.MethodGet, "/jobs/"+job.ID.String()+"/logs", nil, userJWT)
	assertStatus(t, get, http.StatusOK)
	var stored []map[string]any
	decodeJSON(t, get, &stored)
	if len(stored) != 2 {
		t.Fatalf("expected 2 log lines, got %d", len(stored))
	}
	if stored[0]["seq"] != float64(1) {
		t.Fatalf("expected seq=1 first, got %v", stored[0]["seq"])
	}
}

func TestAgentPostLogs_DuplicateSeqIgnored(t *testing.T) {
	p := mustCreateProject(t)
	tmpl := mustCreateTemplate(t, p.ID)
	agent, rawToken := mustCreateAgent(t)

	job := mustCreateJob(t, tmpl.ID)
	if err := testStore.UpdateJobStatus(context.Background(), job.ID, types.JobStatusRunning, &agent.ID, nil); err != nil {
		t.Fatalf("assign job: %v", err)
	}

	logs := []map[string]any{{"seq": 1, "line": "first", "stream": "stdout"}}

	// Post the same seq twice — second should be silently ignored (ON CONFLICT DO NOTHING).
	for i := 0; i < 2; i++ {
		resp := do(t, http.MethodPost, "/agent/jobs/"+job.ID.String()+"/logs", logs, rawToken)
		assertStatus(t, resp, http.StatusNoContent)
		resp.Body.Close()
	}

	get := do(t, http.MethodGet, "/jobs/"+job.ID.String()+"/logs", nil, userJWT)
	assertStatus(t, get, http.StatusOK)
	var stored []map[string]any
	decodeJSON(t, get, &stored)
	if len(stored) != 1 {
		t.Fatalf("expected exactly 1 log line after duplicate post, got %d", len(stored))
	}
}

func TestAgentPostResult_Success(t *testing.T) {
	p := mustCreateProject(t)
	tmpl := mustCreateTemplate(t, p.ID)
	agent, rawToken := mustCreateAgent(t)

	job := mustCreateJob(t, tmpl.ID)
	if err := testStore.UpdateJobStatus(context.Background(), job.ID, types.JobStatusRunning, &agent.ID, nil); err != nil {
		t.Fatalf("assign job: %v", err)
	}

	resp := do(t, http.MethodPost, "/agent/jobs/"+job.ID.String()+"/result", map[string]any{
		"status":    "success",
		"exit_code": 0,
		"facts":     map[string]any{"ansible_version": "2.16"},
	}, rawToken)
	assertStatus(t, resp, http.StatusNoContent)
	resp.Body.Close()

	get := do(t, http.MethodGet, "/jobs/"+job.ID.String(), nil, userJWT)
	assertStatus(t, get, http.StatusOK)
	var body map[string]any
	decodeJSON(t, get, &body)
	if body["status"] != "success" {
		t.Fatalf("expected status=success, got %v", body["status"])
	}
}

func TestAgentPostResult_NonTerminalStatus(t *testing.T) {
	p := mustCreateProject(t)
	tmpl := mustCreateTemplate(t, p.ID)
	agent, rawToken := mustCreateAgent(t)

	job := mustCreateJob(t, tmpl.ID)
	if err := testStore.UpdateJobStatus(context.Background(), job.ID, types.JobStatusRunning, &agent.ID, nil); err != nil {
		t.Fatalf("assign job: %v", err)
	}

	// "running" is not terminal — handler must reject it.
	resp := do(t, http.MethodPost, "/agent/jobs/"+job.ID.String()+"/result", map[string]any{
		"status":    "running",
		"exit_code": 0,
	}, rawToken)
	assertStatus(t, resp, http.StatusBadRequest)
	resp.Body.Close()
}

func TestAgentPostLogs_WrongAgent(t *testing.T) {
	p := mustCreateProject(t)
	tmpl := mustCreateTemplate(t, p.ID)
	agent, _ := mustCreateAgent(t)
	_, otherToken := mustCreateAgent(t)

	job := mustCreateJob(t, tmpl.ID)
	if err := testStore.UpdateJobStatus(context.Background(), job.ID, types.JobStatusRunning, &agent.ID, nil); err != nil {
		t.Fatalf("assign job: %v", err)
	}

	// Post with the other agent's token — must get 403.
	resp := do(t, http.MethodPost, "/agent/jobs/"+job.ID.String()+"/logs",
		[]map[string]any{{"seq": 1, "line": "x", "stream": "stdout"}}, otherToken)
	assertStatus(t, resp, http.StatusForbidden)
	resp.Body.Close()
}
