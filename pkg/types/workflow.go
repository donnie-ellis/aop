package types

import (
	"time"

	"github.com/google/uuid"
)

// ---------------------------------------------------------------------------
// Workflow
// ---------------------------------------------------------------------------

// Workflow is a directed acyclic graph of nodes connected by conditional edges.
// Running a workflow creates a WorkflowRun that the controller engine drives
// from start to completion.
type Workflow struct {
	ID          uuid.UUID `json:"id"          db:"id"`
	Name        string    `json:"name"        db:"name"`
	Description string    `json:"description" db:"description"`
	CreatedAt   time.Time `json:"created_at"  db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"  db:"updated_at"`
}

// ---------------------------------------------------------------------------
// WorkflowNode
// ---------------------------------------------------------------------------

// WorkflowNodeKind is the type of work a node performs.
type WorkflowNodeKind string

const (
	// WorkflowNodeKindJobTemplate executes an Ansible job via a JobTemplate.
	WorkflowNodeKindJobTemplate WorkflowNodeKind = "job_template"

	// WorkflowNodeKindApproval blocks the workflow until a user approves or denies.
	WorkflowNodeKindApproval WorkflowNodeKind = "approval"

	// WorkflowNodeKindSubWorkflow executes a nested workflow. v2 feature — seam
	// is present in the type system now so the schema is stable.
	WorkflowNodeKindSubWorkflow WorkflowNodeKind = "sub_workflow"
)

// WorkflowNode is a vertex in the workflow DAG. Each node has a Kind and a
// reference to the resource it will execute.
type WorkflowNode struct {
	ID         uuid.UUID        `json:"id"          db:"id"`
	WorkflowID uuid.UUID        `json:"workflow_id" db:"workflow_id"`
	Kind       WorkflowNodeKind `json:"kind"        db:"kind"`

	// ResourceID points to the entity this node executes:
	//   job_template → JobTemplate.ID
	//   approval     → (no resource, approval is inline)
	//   sub_workflow → Workflow.ID
	ResourceID *uuid.UUID `json:"resource_id" db:"resource_id"`

	// Label is a short human-readable name shown in the UI.
	Label string `json:"label" db:"label"`

	// ExtraVars injected specifically for this node, merged over the workflow context.
	ExtraVars map[string]any `json:"extra_vars" db:"extra_vars"`

	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// ---------------------------------------------------------------------------
// WorkflowEdge
// ---------------------------------------------------------------------------

// EdgeCondition controls when an edge is followed after the source node completes.
// In v1 only the three literal values are used. The field type is string (not an
// enum) so that v2 can store CEL expressions like `facts.exit_code == 0` without
// a schema migration.
type EdgeCondition = string

const (
	EdgeConditionOnSuccess EdgeCondition = "on_success"
	EdgeConditionOnFailure EdgeCondition = "on_failure"
	EdgeConditionAlways    EdgeCondition = "always"
)

// WorkflowEdge is a directed edge from one node to another with an optional condition.
type WorkflowEdge struct {
	ID           uuid.UUID `json:"id"            db:"id"`
	WorkflowID   uuid.UUID `json:"workflow_id"   db:"workflow_id"`
	SourceNodeID uuid.UUID `json:"source_node_id" db:"source_node_id"`
	TargetNodeID uuid.UUID `json:"target_node_id" db:"target_node_id"`

	// Condition is the string condition under which this edge is followed.
	// v1: one of EdgeConditionOnSuccess, EdgeConditionOnFailure, EdgeConditionAlways.
	// v2: may be a CEL expression evaluated against WorkflowContext.Facts.
	Condition EdgeCondition `json:"condition" db:"condition"`
}

// ---------------------------------------------------------------------------
// WorkflowRun
// ---------------------------------------------------------------------------

// WorkflowRunStatus mirrors JobStatus semantics at the workflow level.
type WorkflowRunStatus string

const (
	WorkflowRunStatusPending   WorkflowRunStatus = "pending"
	WorkflowRunStatusRunning   WorkflowRunStatus = "running"
	WorkflowRunStatusSuccess   WorkflowRunStatus = "success"
	WorkflowRunStatusFailed    WorkflowRunStatus = "failed"
	WorkflowRunStatusCancelled WorkflowRunStatus = "cancelled"
)

// IsTerminal returns true if no further transitions are expected.
func (s WorkflowRunStatus) IsTerminal() bool {
	switch s {
	case WorkflowRunStatusSuccess, WorkflowRunStatusFailed, WorkflowRunStatusCancelled:
		return true
	}
	return false
}

// WorkflowRun is a single execution instance of a Workflow. The workflow engine
// in the controller drives it from start to completion.
type WorkflowRun struct {
	ID         uuid.UUID         `json:"id"          db:"id"`
	WorkflowID uuid.UUID         `json:"workflow_id" db:"workflow_id"`
	Status     WorkflowRunStatus `json:"status"      db:"status"`

	// Context is the facts bag. Node completion merges Ansible set_stats output
	// into this map. In v2, decision gate nodes will read from it via CEL.
	Context map[string]any `json:"context" db:"context"`

	CreatedAt time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt time.Time  `json:"updated_at" db:"updated_at"`
	StartedAt *time.Time `json:"started_at" db:"started_at"`
	EndedAt   *time.Time `json:"ended_at"   db:"ended_at"`
}

// ---------------------------------------------------------------------------
// WorkflowNodeRun
// ---------------------------------------------------------------------------

// WorkflowNodeRunStatus tracks per-node execution state within a workflow run.
type WorkflowNodeRunStatus string

const (
	WorkflowNodeRunStatusWaiting   WorkflowNodeRunStatus = "waiting" // dependencies not yet complete
	WorkflowNodeRunStatusRunning   WorkflowNodeRunStatus = "running"
	WorkflowNodeRunStatusSuccess   WorkflowNodeRunStatus = "success"
	WorkflowNodeRunStatusFailed    WorkflowNodeRunStatus = "failed"
	WorkflowNodeRunStatusSkipped   WorkflowNodeRunStatus = "skipped" // edge condition not met
	WorkflowNodeRunStatusCancelled WorkflowNodeRunStatus = "cancelled"
)

// WorkflowNodeRun records the execution of a single node within a workflow run.
// The engine uses these records to determine which nodes are ready to execute.
type WorkflowNodeRun struct {
	ID            uuid.UUID             `json:"id"              db:"id"`
	WorkflowRunID uuid.UUID             `json:"workflow_run_id" db:"workflow_run_id"`
	NodeID        uuid.UUID             `json:"node_id"         db:"node_id"`
	Status        WorkflowNodeRunStatus `json:"status"          db:"status"`

	// JobID is set when the node creates a job (kind=job_template).
	JobID *uuid.UUID `json:"job_id" db:"job_id"`

	CreatedAt time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt time.Time  `json:"updated_at" db:"updated_at"`
	StartedAt *time.Time `json:"started_at" db:"started_at"`
	EndedAt   *time.Time `json:"ended_at"   db:"ended_at"`
}

// ---------------------------------------------------------------------------
// ApprovalRequest
// ---------------------------------------------------------------------------

// ApprovalStatus tracks the state of a manual approval gate.
type ApprovalStatus string

const (
	ApprovalStatusPending  ApprovalStatus = "pending"
	ApprovalStatusApproved ApprovalStatus = "approved"
	ApprovalStatusDenied   ApprovalStatus = "denied"
)

// ---------------------------------------------------------------------------
// WorkflowContext
// ---------------------------------------------------------------------------

// WorkflowContext is the facts bag that travels with a workflow run.
// Node completion merges Ansible set_stats output into Facts.
// In v1 nothing reads from it. In v2, decision gate nodes evaluate CEL
// expressions against it — the data is already there when v2 needs it.
type WorkflowContext struct {
	RunID string
	Facts map[string]any
}

// Merge merges the provided facts into the context, overwriting on collision.
func (c *WorkflowContext) Merge(facts map[string]any) {
	for k, v := range facts {
		c.Facts[k] = v
	}
}

// ---------------------------------------------------------------------------
// ApprovalRequest
// ---------------------------------------------------------------------------

// ApprovalRequest is created by the ApprovalExecutor when the workflow engine
// reaches an approval node. The workflow is blocked until a user resolves it
// via the API.
type ApprovalRequest struct {
	ID            uuid.UUID      `json:"id"              db:"id"`
	WorkflowRunID uuid.UUID      `json:"workflow_run_id" db:"workflow_run_id"`
	NodeID        uuid.UUID      `json:"node_id"         db:"node_id"`
	Status        ApprovalStatus `json:"status"          db:"status"`

	// ReviewedBy is the user ID of the approver/denier, if resolved.
	ReviewedBy *uuid.UUID `json:"reviewed_by" db:"reviewed_by"`
	ReviewNote *string    `json:"review_note" db:"review_note"`

	CreatedAt  time.Time  `json:"created_at"  db:"created_at"`
	ResolvedAt *time.Time `json:"resolved_at" db:"resolved_at"`
}
