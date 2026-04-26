package handler_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/google/uuid"
)

// seedApproval creates a workflow run and an approval_request row directly,
// returning the approval request ID.
func seedApproval(t *testing.T) uuid.UUID {
	t.Helper()
	// Create a workflow.
	wf, err := testStore.CreateWorkflow(context.Background(), "approval-wf-"+uuid.New().String()[:8], "")
	if err != nil {
		t.Fatalf("create workflow for approval: %v", err)
	}
	// Create a run.
	run, err := testStore.CreateWorkflowRun(context.Background(), wf.ID)
	if err != nil {
		t.Fatalf("create workflow run for approval: %v", err)
	}
	// Create a workflow node so the approval FK is satisfied.
	nodeID := uuid.New()
	if _, err := testPool.Exec(context.Background(),
		`INSERT INTO workflow_nodes (id, workflow_id, kind, resource_id, label, extra_vars)
		 VALUES ($1, $2, 'approval', $3, 'Gate', '{}')`,
		nodeID, wf.ID, uuid.New(),
	); err != nil {
		t.Fatalf("seed workflow node: %v", err)
	}

	// Insert an approval_requests row directly — in production this is done by the controller.
	var approvalID uuid.UUID
	err = testPool.QueryRow(context.Background(),
		`INSERT INTO approval_requests (workflow_run_id, node_id, status)
		 VALUES ($1, $2, 'pending')
		 RETURNING id`,
		run.ID, nodeID,
	).Scan(&approvalID)
	if err != nil {
		t.Fatalf("seed approval request: %v", err)
	}
	return approvalID
}

func TestListPendingApprovals(t *testing.T) {
	seedApproval(t)

	resp := do(t, http.MethodGet, "/approvals", nil, userJWT)
	assertStatus(t, resp, http.StatusOK)

	var approvals []map[string]any
	decodeJSON(t, resp, &approvals)
	if len(approvals) == 0 {
		t.Fatal("expected at least one pending approval")
	}
	for _, a := range approvals {
		if a["status"] != "pending" {
			t.Fatalf("list returned non-pending approval: status=%v", a["status"])
		}
	}
}

func TestListPendingApprovals_RequiresAuth(t *testing.T) {
	resp := do(t, http.MethodGet, "/approvals", nil, "")
	assertStatus(t, resp, http.StatusUnauthorized)
	resp.Body.Close()
}

func TestGetApproval(t *testing.T) {
	id := seedApproval(t)

	resp := do(t, http.MethodGet, "/approvals/"+id.String(), nil, userJWT)
	assertStatus(t, resp, http.StatusOK)

	var body map[string]any
	decodeJSON(t, resp, &body)
	if body["id"] != id.String() {
		t.Fatalf("expected id %s, got %v", id, body["id"])
	}
	if body["status"] != "pending" {
		t.Fatalf("expected status=pending, got %v", body["status"])
	}
}

func TestGetApproval_NotFound(t *testing.T) {
	resp := do(t, http.MethodGet, "/approvals/00000000-0000-0000-0000-000000000060", nil, userJWT)
	assertStatus(t, resp, http.StatusNotFound)
	resp.Body.Close()
}

func TestGetApproval_InvalidID(t *testing.T) {
	resp := do(t, http.MethodGet, "/approvals/not-a-uuid", nil, userJWT)
	assertStatus(t, resp, http.StatusBadRequest)
	resp.Body.Close()
}

func TestResolveApproval_Approve(t *testing.T) {
	id := seedApproval(t)

	resp := do(t, http.MethodPost, "/approvals/"+id.String()+"/approve", map[string]any{
		"note": "LGTM",
	}, userJWT)
	assertStatus(t, resp, http.StatusNoContent)
	resp.Body.Close()

	// Confirm status changed.
	get := do(t, http.MethodGet, "/approvals/"+id.String(), nil, userJWT)
	assertStatus(t, get, http.StatusOK)
	var body map[string]any
	decodeJSON(t, get, &body)
	if body["status"] != "approved" {
		t.Fatalf("expected status=approved, got %v", body["status"])
	}
	if body["review_note"] != "LGTM" {
		t.Fatalf("expected review_note=LGTM, got %v", body["review_note"])
	}
	if body["reviewed_by"] == nil {
		t.Fatal("expected reviewed_by to be set")
	}
}

func TestResolveApproval_Deny(t *testing.T) {
	id := seedApproval(t)

	resp := do(t, http.MethodPost, "/approvals/"+id.String()+"/deny", map[string]any{
		"note": "Not safe to deploy",
	}, userJWT)
	assertStatus(t, resp, http.StatusNoContent)
	resp.Body.Close()

	get := do(t, http.MethodGet, "/approvals/"+id.String(), nil, userJWT)
	assertStatus(t, get, http.StatusOK)
	var body map[string]any
	decodeJSON(t, get, &body)
	if body["status"] != "denied" {
		t.Fatalf("expected status=denied, got %v", body["status"])
	}
}

func TestResolveApproval_InvalidAction(t *testing.T) {
	id := seedApproval(t)

	resp := do(t, http.MethodPost, "/approvals/"+id.String()+"/maybe", nil, userJWT)
	assertStatus(t, resp, http.StatusBadRequest)
	resp.Body.Close()
}

func TestResolveApproval_AlreadyResolved(t *testing.T) {
	id := seedApproval(t)

	// Approve first.
	r1 := do(t, http.MethodPost, "/approvals/"+id.String()+"/approve", nil, userJWT)
	assertStatus(t, r1, http.StatusNoContent)
	r1.Body.Close()

	// Try to deny the already-approved request — must 404 (WHERE status='pending' matches nothing).
	r2 := do(t, http.MethodPost, "/approvals/"+id.String()+"/deny", nil, userJWT)
	assertStatus(t, r2, http.StatusNotFound)
	r2.Body.Close()
}

func TestResolveApproval_RequiresAuth(t *testing.T) {
	id := seedApproval(t)

	resp := do(t, http.MethodPost, "/approvals/"+id.String()+"/approve", nil, "")
	assertStatus(t, resp, http.StatusUnauthorized)
	resp.Body.Close()
}
