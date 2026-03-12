package types

import "time"

// ============================================================================
// Agent
// ============================================================================

type Agent struct {
	ID              string            `json:"id"`
	Hostname        string            `json:"hostname"`
	Labels          map[string]string `json:"labels"`
	Capacity        int               `json:"capacity"`
	RunningJobs     int               `json:"running_jobs"`
	Status          AgentStatus       `json:"status"`
	LastHeartbeatAt time.Time         `json:"last_heartbeat_at"`
	RegisteredAt    time.Time         `json:"registered_at"`
	Version         string            `json:"version"`
}

type AgentStatus string

const (
	AgentStatusOnline  AgentStatus = "online"
	AgentStatusOffline AgentStatus = "offline"
	AgentStatusBusy    AgentStatus = "busy"
)

// ============================================================================
// Job
// ============================================================================

type Job struct {
	ID            string         `json:"id"`
	TemplateID    string         `json:"template_id"`
	AgentID       *string        `json:"agent_id"`
	Status        JobStatus      `json:"status"`
	ExtraVars     map[string]any `json:"extra_vars"`
	TriggeredBy   string         `json:"triggered_by"`
	WorkflowRunID *string        `json:"workflow_run_id"`
	CreatedAt     time.Time      `json:"created_at"`
	StartedAt     *time.Time     `json:"started_at"`
	FinishedAt    *time.Time     `json:"finished_at"`
}

type JobStatus string

const (
	JobStatusPending    JobStatus = "pending"
	JobStatusDispatched JobStatus = "dispatched"
	JobStatusRunning    JobStatus = "running"
	JobStatusSuccess    JobStatus = "success"
	JobStatusFailed     JobStatus = "failed"
	JobStatusCancelled  JobStatus = "cancelled"
)

func (s JobStatus) IsTerminal() bool {
	return s == JobStatusSuccess || s == JobStatusFailed || s == JobStatusCancelled
}

// JobPayload is everything an agent needs to execute a job.
type JobPayload struct {
	JobID        string
	Playbook     string
	Inventory    string
	Credentials  []SecretValue
	ExtraVars    map[string]any
	WorkspaceDir string
}

// ============================================================================
// Job Template
// ============================================================================

type JobTemplate struct {
	ID          string         `json:"id"`
	Name        string         `json:"name"`
	ProjectID   string         `json:"project_id"`
	Playbook    string         `json:"playbook"`
	InventoryID string         `json:"inventory_id"`
	Credentials []string       `json:"credential_ids"`
	ExtraVars   map[string]any `json:"extra_vars"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
}

// ============================================================================
// Credential
// ============================================================================

type Credential struct {
	ID        string         `json:"id"`
	Name      string         `json:"name"`
	Type      CredentialType `json:"type"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
}

type CredentialType string

const (
	CredentialTypeSSHKey           CredentialType = "ssh_key"
	CredentialTypeVaultPassword    CredentialType = "vault_password"
	CredentialTypeUsernamePassword CredentialType = "username_password"
)

// SecretValue holds resolved credential data transiently — never persisted.
type SecretValue struct {
	CredentialID string
	Type         CredentialType
	Data         map[string]string
}

// ============================================================================
// Inventory
// ============================================================================

type Inventory struct {
	ID           string         `json:"id"`
	Name         string         `json:"name"`
	SourceType   string         `json:"source_type"`
	SourceConfig map[string]any `json:"source_config"`
	Content      string         `json:"content"`
	SyncStatus   SyncStatus     `json:"sync_status"`
	LastSyncedAt *time.Time     `json:"last_synced_at"`
	CreatedAt    time.Time      `json:"created_at"`
}

type SyncStatus string

const (
	SyncStatusPending SyncStatus = "pending"
	SyncStatusSyncing SyncStatus = "syncing"
	SyncStatusSuccess SyncStatus = "success"
	SyncStatusFailed  SyncStatus = "failed"
)

// ============================================================================
// Project
// ============================================================================

type Project struct {
	ID           string     `json:"id"`
	Name         string     `json:"name"`
	GitURL       string     `json:"git_url"`
	Branch       string     `json:"branch"`
	CredentialID *string    `json:"credential_id"`
	SyncStatus   SyncStatus `json:"sync_status"`
	LastSyncedAt *time.Time `json:"last_synced_at"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

// ============================================================================
// Workflow
// ============================================================================

type Workflow struct {
	ID        string         `json:"id"`
	Name      string         `json:"name"`
	Nodes     []WorkflowNode `json:"nodes"`
	Edges     []WorkflowEdge `json:"edges"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
}

type WorkflowNode struct {
	ID         string         `json:"id"`
	WorkflowID string         `json:"workflow_id"`
	Type       string         `json:"type"`
	Spec       map[string]any `json:"spec"`
}

type WorkflowEdge struct {
	ID           string `json:"id"`
	WorkflowID   string `json:"workflow_id"`
	SourceNodeID string `json:"source_node_id"`
	TargetNodeID string `json:"target_node_id"`
	Condition    string `json:"condition"`
}

type WorkflowRun struct {
	ID         string            `json:"id"`
	WorkflowID string            `json:"workflow_id"`
	Status     JobStatus         `json:"status"`
	NodeRuns   []WorkflowNodeRun `json:"node_runs"`
	Facts      map[string]any    `json:"facts"`
	StartedAt  time.Time         `json:"started_at"`
	FinishedAt *time.Time        `json:"finished_at"`
}

type WorkflowNodeRun struct {
	ID         string     `json:"id"`
	RunID      string     `json:"run_id"`
	NodeID     string     `json:"node_id"`
	JobID      *string    `json:"job_id"`
	Status     JobStatus  `json:"status"`
	StartedAt  *time.Time `json:"started_at"`
	FinishedAt *time.Time `json:"finished_at"`
}

// WorkflowContext is the facts bag that travels with a workflow run.
type WorkflowContext struct {
	RunID string
	Facts map[string]any
}

func (c *WorkflowContext) Merge(facts map[string]any) {
	for k, v := range facts {
		c.Facts[k] = v
	}
}

// ============================================================================
// Schedule
// ============================================================================

type Schedule struct {
	ID         string    `json:"id"`
	TemplateID string    `json:"template_id"`
	Cron       string    `json:"cron"`
	Timezone   string    `json:"timezone"`
	Enabled    bool      `json:"enabled"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}
