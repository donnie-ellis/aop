package handler_test

import (
	"net/http"
	"testing"

	"github.com/donnie-ellis/aop/pkg/types"
	"github.com/google/uuid"
)

func mustCreateWorkflow(t *testing.T) *types.Workflow {
	t.Helper()
	resp := do(t, http.MethodPost, "/workflows", map[string]any{
		"name":        "wf-" + uuid.New().String()[:8],
		"description": "test workflow",
	}, userJWT)
	assertStatus(t, resp, http.StatusCreated)
	var wf types.Workflow
	decodeJSON(t, resp, &wf)
	return &wf
}

func TestCreateWorkflow_Success(t *testing.T) {
	resp := do(t, http.MethodPost, "/workflows", map[string]any{
		"name":        "deploy-pipeline",
		"description": "full deploy",
	}, userJWT)
	assertStatus(t, resp, http.StatusCreated)

	var body map[string]any
	decodeJSON(t, resp, &body)
	if body["id"] == "" {
		t.Fatal("expected id in response")
	}
	if body["name"] != "deploy-pipeline" {
		t.Fatalf("expected name deploy-pipeline, got %v", body["name"])
	}
}

func TestCreateWorkflow_MissingName(t *testing.T) {
	resp := do(t, http.MethodPost, "/workflows", map[string]any{
		"description": "no name",
	}, userJWT)
	assertStatus(t, resp, http.StatusBadRequest)
	resp.Body.Close()
}

func TestCreateWorkflow_RequiresAuth(t *testing.T) {
	resp := do(t, http.MethodPost, "/workflows", map[string]any{"name": "x"}, "")
	assertStatus(t, resp, http.StatusUnauthorized)
	resp.Body.Close()
}

func TestGetWorkflow_ReturnsNodesAndEdges(t *testing.T) {
	wf := mustCreateWorkflow(t)

	resp := do(t, http.MethodGet, "/workflows/"+wf.ID.String(), nil, userJWT)
	assertStatus(t, resp, http.StatusOK)

	var body map[string]any
	decodeJSON(t, resp, &body)
	// Response must include workflow, nodes, edges keys.
	if _, ok := body["workflow"]; !ok {
		t.Fatal("response must contain 'workflow' key")
	}
	if _, ok := body["nodes"]; !ok {
		t.Fatal("response must contain 'nodes' key")
	}
	if _, ok := body["edges"]; !ok {
		t.Fatal("response must contain 'edges' key")
	}
}

func TestGetWorkflow_NotFound(t *testing.T) {
	resp := do(t, http.MethodGet, "/workflows/00000000-0000-0000-0000-000000000050", nil, userJWT)
	assertStatus(t, resp, http.StatusNotFound)
	resp.Body.Close()
}

func TestListWorkflows(t *testing.T) {
	mustCreateWorkflow(t)

	resp := do(t, http.MethodGet, "/workflows", nil, userJWT)
	assertStatus(t, resp, http.StatusOK)

	var workflows []map[string]any
	decodeJSON(t, resp, &workflows)
	if len(workflows) == 0 {
		t.Fatal("expected at least one workflow")
	}
}

func TestUpdateWorkflow_NameAndDescription(t *testing.T) {
	wf := mustCreateWorkflow(t)

	resp := do(t, http.MethodPut, "/workflows/"+wf.ID.String(), map[string]any{
		"name":        "updated-workflow",
		"description": "updated desc",
	}, userJWT)
	assertStatus(t, resp, http.StatusOK)

	var body map[string]any
	decodeJSON(t, resp, &body)
	if body["name"] != "updated-workflow" {
		t.Fatalf("expected updated-workflow, got %v", body["name"])
	}
}

func TestUpdateWorkflow_UpsertsNodesAndEdges(t *testing.T) {
	p := mustCreateProject(t)
	tmpl := mustCreateTemplate(t, p.ID)
	wf := mustCreateWorkflow(t)

	nodeID := uuid.New()
	resp := do(t, http.MethodPut, "/workflows/"+wf.ID.String(), map[string]any{
		"name": wf.Name,
		"nodes": []map[string]any{
			{
				"id":          nodeID.String(),
				"kind":        "job_template",
				"resource_id": tmpl.ID.String(),
				"label":       "Deploy",
			},
		},
		"edges": []map[string]any{},
	}, userJWT)
	assertStatus(t, resp, http.StatusOK)
	resp.Body.Close()

	// GET must now return the node.
	get := do(t, http.MethodGet, "/workflows/"+wf.ID.String(), nil, userJWT)
	assertStatus(t, get, http.StatusOK)
	var body map[string]any
	decodeJSON(t, get, &body)
	nodes, _ := body["nodes"].([]any)
	if len(nodes) != 1 {
		t.Fatalf("expected 1 node after upsert, got %d", len(nodes))
	}
}

func TestUpdateWorkflow_NotFound(t *testing.T) {
	resp := do(t, http.MethodPut, "/workflows/00000000-0000-0000-0000-000000000051", map[string]any{
		"name": "x",
	}, userJWT)
	assertStatus(t, resp, http.StatusNotFound)
	resp.Body.Close()
}

func TestDeleteWorkflow(t *testing.T) {
	wf := mustCreateWorkflow(t)

	resp := do(t, http.MethodDelete, "/workflows/"+wf.ID.String(), nil, userJWT)
	assertStatus(t, resp, http.StatusNoContent)
	resp.Body.Close()

	resp2 := do(t, http.MethodGet, "/workflows/"+wf.ID.String(), nil, userJWT)
	assertStatus(t, resp2, http.StatusNotFound)
	resp2.Body.Close()
}

func TestRunWorkflow(t *testing.T) {
	wf := mustCreateWorkflow(t)

	resp := do(t, http.MethodPost, "/workflows/"+wf.ID.String()+"/run", nil, userJWT)
	assertStatus(t, resp, http.StatusCreated)

	var body map[string]any
	decodeJSON(t, resp, &body)
	if body["id"] == "" {
		t.Fatal("expected workflow run id")
	}
	if body["workflow_id"] != wf.ID.String() {
		t.Fatalf("expected workflow_id %s, got %v", wf.ID, body["workflow_id"])
	}
	if body["status"] != "pending" {
		t.Fatalf("expected initial run status=pending, got %v", body["status"])
	}
}

func TestRunWorkflow_NotFound(t *testing.T) {
	resp := do(t, http.MethodPost, "/workflows/00000000-0000-0000-0000-000000000052/run", nil, userJWT)
	assertStatus(t, resp, http.StatusNotFound)
	resp.Body.Close()
}

func TestListWorkflowRuns(t *testing.T) {
	wf := mustCreateWorkflow(t)

	// Create two runs.
	for i := 0; i < 2; i++ {
		r := do(t, http.MethodPost, "/workflows/"+wf.ID.String()+"/run", nil, userJWT)
		assertStatus(t, r, http.StatusCreated)
		r.Body.Close()
	}

	resp := do(t, http.MethodGet, "/workflows/"+wf.ID.String()+"/runs", nil, userJWT)
	assertStatus(t, resp, http.StatusOK)

	var runs []map[string]any
	decodeJSON(t, resp, &runs)
	if len(runs) < 2 {
		t.Fatalf("expected at least 2 runs, got %d", len(runs))
	}
}

func TestGetWorkflowRun(t *testing.T) {
	wf := mustCreateWorkflow(t)

	cr := do(t, http.MethodPost, "/workflows/"+wf.ID.String()+"/run", nil, userJWT)
	assertStatus(t, cr, http.StatusCreated)
	var run map[string]any
	decodeJSON(t, cr, &run)
	runID := run["id"].(string)

	resp := do(t, http.MethodGet, "/workflow-runs/"+runID, nil, userJWT)
	assertStatus(t, resp, http.StatusOK)

	var body map[string]any
	decodeJSON(t, resp, &body)
	if body["id"] != runID {
		t.Fatalf("expected run id %s, got %v", runID, body["id"])
	}
}

func TestGetWorkflowRun_NotFound(t *testing.T) {
	resp := do(t, http.MethodGet, "/workflow-runs/00000000-0000-0000-0000-000000000053", nil, userJWT)
	assertStatus(t, resp, http.StatusNotFound)
	resp.Body.Close()
}
