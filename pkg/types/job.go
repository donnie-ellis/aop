// Package types defines shared domain types used across the AOP api, controller,
// and agent components. Nothing in this package imports from any other AOP package
// — it is the foundation layer with zero internal dependencies.
package types

import (
	"time"

	"github.com/google/uuid"
)

// ---------------------------------------------------------------------------
// JobStatus
// ---------------------------------------------------------------------------

// JobStatus represents the lifecycle state of a Job.
// Transitions are strictly validated; see the state machine constants below.
type JobStatus string

const (
	// JobStatusPending is the initial state. The job is waiting to be dispatched.
	JobStatusPending JobStatus = "pending"

	// JobStatusDispatched means a controller has selected an agent and sent the job.
	// The agent has not yet confirmed it is running.
	JobStatusDispatched JobStatus = "dispatched"

	// JobStatusRunning means the agent has confirmed execution has started.
	JobStatusRunning JobStatus = "running"

	// JobStatusSuccess is a terminal state. The job completed without errors.
	JobStatusSuccess JobStatus = "success"

	// JobStatusFailed is a terminal state. The job failed, was timed out, or
	// could not be dispatched.
	JobStatusFailed JobStatus = "failed"

	// JobStatusCancelled is a terminal state. A user or workflow engine cancelled
	// the job before or during execution.
	JobStatusCancelled JobStatus = "cancelled"
)

// IsTerminal returns true if the status is a final state from which no further
// transitions are permitted.
func (s JobStatus) IsTerminal() bool {
	switch s {
	case JobStatusSuccess, JobStatusFailed, JobStatusCancelled:
		return true
	}
	return false
}

// ValidTransitions defines the allowed next states for each job status.
// The controller validates every status change against this map and rejects
// invalid transitions.
var ValidTransitions = map[JobStatus][]JobStatus{
	JobStatusPending:    {JobStatusDispatched, JobStatusFailed},
	JobStatusDispatched: {JobStatusRunning, JobStatusFailed, JobStatusCancelled},
	JobStatusRunning:    {JobStatusSuccess, JobStatusFailed, JobStatusCancelled},
	// Terminal states have no valid outbound transitions.
	JobStatusSuccess:   {},
	JobStatusFailed:    {},
	JobStatusCancelled: {},
}

// CanTransitionTo returns true if transitioning from s to next is a valid move.
func (s JobStatus) CanTransitionTo(next JobStatus) bool {
	for _, allowed := range ValidTransitions[s] {
		if allowed == next {
			return true
		}
	}
	return false
}

// ---------------------------------------------------------------------------
// Job
// ---------------------------------------------------------------------------

// Job is the core execution unit. A job represents a single Ansible playbook
// run dispatched to a specific agent.
type Job struct {
	ID         uuid.UUID  `json:"id"          db:"id"`
	TemplateID uuid.UUID  `json:"template_id" db:"template_id"`
	AgentID    *uuid.UUID `json:"agent_id"    db:"agent_id"`    // nil until dispatched
	WorkflowRunID *uuid.UUID `json:"workflow_run_id" db:"workflow_run_id"` // nil for standalone jobs

	Status    JobStatus  `json:"status"     db:"status"`
	CreatedAt time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt time.Time  `json:"updated_at" db:"updated_at"`
	StartedAt *time.Time `json:"started_at" db:"started_at"` // set when agent reports running
	EndedAt   *time.Time `json:"ended_at"   db:"ended_at"`   // set on terminal transition

	// ExtraVars is a freeform JSON object of variables merged at dispatch time.
	// These come from the triggering request (API, scheduler, workflow context).
	ExtraVars map[string]any `json:"extra_vars" db:"extra_vars"`

	// Facts holds values emitted by Ansible set_stats after completion.
	// These are merged into the WorkflowContext bag for downstream nodes.
	Facts map[string]any `json:"facts" db:"facts"`

	// FailureReason is a short human-readable explanation set on failed transitions.
	FailureReason *string `json:"failure_reason" db:"failure_reason"`
}

// JobStatusEvent records a single state transition for audit and display.
type JobStatusEvent struct {
	ID        uuid.UUID `json:"id"         db:"id"`
	JobID     uuid.UUID `json:"job_id"     db:"job_id"`
	FromStatus JobStatus `json:"from_status" db:"from_status"`
	ToStatus  JobStatus `json:"to_status"  db:"to_status"`
	Timestamp time.Time `json:"timestamp"  db:"timestamp"`
	Reason    string    `json:"reason"     db:"reason"`
}

// ---------------------------------------------------------------------------
// JobPayload — what the controller sends to an agent
// ---------------------------------------------------------------------------

// JobLogLine is a single line of output captured from ansible-playbook.
// Batches of these are POSTed by the agent and stored in job_logs.
type JobLogLine struct {
	Seq    int    `json:"seq"`
	Line   string `json:"line"`
	Stream string `json:"stream"` // "stdout" | "stderr"
}

// JobPayload is the wire structure the controller sends to an agent when
// dispatching a job. It contains everything the agent needs to execute;
// the agent never queries the database or any secrets backend.
//
// The controller resolves and decrypts credentials before building this
// payload. The agent receives plaintext material, materializes it into
// temp files, and deletes those files in a deferred cleanup after every run.
type JobPayload struct {
	JobID      uuid.UUID       `json:"job_id"`
	TemplateID uuid.UUID       `json:"template_id"`
	Playbook   string          `json:"playbook"`    // path within the project repo
	Inventory  []InventoryHost `json:"inventory"`   // resolved and pre-flattened
	ExtraVars  map[string]any  `json:"extra_vars"`
	Credentials []CredentialSecret `json:"credentials"` // decrypted by controller before dispatch
	CallbackURL string         `json:"callback_url"` // agent POSTs status/logs here (API server URL)
}
