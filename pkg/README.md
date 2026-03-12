# Shared Types (`/pkg`)

## Purpose

The `/pkg` package contains Go types, interfaces, and constants shared across the
api, controller, and agent binaries. It is the contract between components.

This package contains **no business logic**. No functions that do work, no database
calls, no HTTP calls. Only type definitions, interface definitions, enums/constants,
and simple value methods (e.g. `func (s JobStatus) IsTerminal() bool`).

If you find yourself wanting to put logic in here, it belongs in one of the component
packages instead.

---

## Package Structure

```
/pkg/
└── types/
    ├── agent.go
    ├── credential.go
    ├── inventory.go
    ├── job.go
    ├── project.go
    ├── workflow.go
    ├── schedule.go
    └── interfaces.go     ← all plugin interfaces live here
```

---

## Core Types

### Agent

```go
type Agent struct {
    ID               string            `json:"id"`
    Hostname         string            `json:"hostname"`
    Labels           map[string]string `json:"labels"`
    Capacity         int               `json:"capacity"`
    RunningJobs      int               `json:"running_jobs"`
    Status           AgentStatus       `json:"status"`
    LastHeartbeatAt  time.Time         `json:"last_heartbeat_at"`
    RegisteredAt     time.Time         `json:"registered_at"`
    Version          string            `json:"version"`
}

type AgentStatus string
const (
    AgentStatusOnline  AgentStatus = "online"
    AgentStatusOffline AgentStatus = "offline"
    AgentStatusBusy    AgentStatus = "busy"
)
```

### Job

```go
type Job struct {
    ID             string            `json:"id"`
    TemplateID     string            `json:"template_id"`
    AgentID        *string           `json:"agent_id"`
    Status         JobStatus         `json:"status"`
    ExtraVars      map[string]any    `json:"extra_vars"`
    TriggeredBy    string            `json:"triggered_by"` // user_id, "scheduler", "workflow"
    WorkflowRunID  *string           `json:"workflow_run_id"`
    CreatedAt      time.Time         `json:"created_at"`
    StartedAt      *time.Time        `json:"started_at"`
    FinishedAt     *time.Time        `json:"finished_at"`
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
```

### JobTemplate

```go
type JobTemplate struct {
    ID          string         `json:"id"`
    Name        string         `json:"name"`
    ProjectID   string         `json:"project_id"`
    Playbook    string         `json:"playbook"`    // relative path within repo
    InventoryID string         `json:"inventory_id"`
    Credentials []string       `json:"credential_ids"`
    ExtraVars   map[string]any `json:"extra_vars"`
    CreatedAt   time.Time      `json:"created_at"`
    UpdatedAt   time.Time      `json:"updated_at"`
}
```

### Credential

```go
type Credential struct {
    ID          string         `json:"id"`
    Name        string         `json:"name"`
    Type        CredentialType `json:"type"`
    // Sensitive fields are never in this struct at the API layer
    // They exist only transiently when resolved by SecretsProvider
    CreatedAt   time.Time      `json:"created_at"`
    UpdatedAt   time.Time      `json:"updated_at"`
}

type CredentialType string
const (
    CredentialTypeSSHKey          CredentialType = "ssh_key"
    CredentialTypeVaultPassword   CredentialType = "vault_password"
    CredentialTypeUsernamePassword CredentialType = "username_password"
)

// SecretValue holds resolved credential data transiently — never persisted in this form
type SecretValue struct {
    CredentialID string
    Type         CredentialType
    Data         map[string]string  // key varies by type: "private_key", "password", etc.
}
```

### Inventory

```go
type Inventory struct {
    ID           string          `json:"id"`
    Name         string          `json:"name"`
    SourceType   string          `json:"source_type"`  // string, not enum — pluggable
    SourceConfig map[string]any  `json:"source_config"` // JSONB — varies by source type
    Content      string          `json:"content"`       // synced inventory text
    SyncStatus   SyncStatus      `json:"sync_status"`
    LastSyncedAt *time.Time      `json:"last_synced_at"`
    CreatedAt    time.Time       `json:"created_at"`
}

type SyncStatus string
const (
    SyncStatusPending  SyncStatus = "pending"
    SyncStatusSyncing  SyncStatus = "syncing"
    SyncStatusSuccess  SyncStatus = "success"
    SyncStatusFailed   SyncStatus = "failed"
)
```

### Project

```go
type Project struct {
    ID           string     `json:"id"`
    Name         string     `json:"name"`
    GitURL       string     `json:"git_url"`
    Branch       string     `json:"branch"`
    CredentialID *string    `json:"credential_id"` // for git auth
    SyncStatus   SyncStatus `json:"sync_status"`
    LastSyncedAt *time.Time `json:"last_synced_at"`
    CreatedAt    time.Time  `json:"created_at"`
    UpdatedAt    time.Time  `json:"updated_at"`
}
```

### Workflow

```go
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
    Type       string         `json:"type"` // "job_template" | "approval" | future types
    Spec       map[string]any `json:"spec"` // JSONB — content varies by type
}

type WorkflowEdge struct {
    ID           string `json:"id"`
    WorkflowID   string `json:"workflow_id"`
    SourceNodeID string `json:"source_node_id"`
    TargetNodeID string `json:"target_node_id"`
    Condition    string `json:"condition"` // "on_success" | "on_failure" | "always" | future expressions
}

type WorkflowRun struct {
    ID         string              `json:"id"`
    WorkflowID string              `json:"workflow_id"`
    Status     JobStatus           `json:"status"`
    NodeRuns   []WorkflowNodeRun   `json:"node_runs"`
    Facts      map[string]any      `json:"facts"` // context bag — populated by jobs, read by v2 gates
    StartedAt  time.Time           `json:"started_at"`
    FinishedAt *time.Time          `json:"finished_at"`
}

type WorkflowNodeRun struct {
    ID         string     `json:"id"`
    RunID      string     `json:"run_id"`
    NodeID     string     `json:"node_id"`
    JobID      *string    `json:"job_id"` // set for job_template nodes
    Status     JobStatus  `json:"status"`
    StartedAt  *time.Time `json:"started_at"`
    FinishedAt *time.Time `json:"finished_at"`
}
```

### Schedule

```go
type Schedule struct {
    ID         string    `json:"id"`
    TemplateID string    `json:"template_id"`
    Cron       string    `json:"cron"`      // standard cron expression
    Timezone   string    `json:"timezone"`  // e.g. "America/New_York"
    Enabled    bool      `json:"enabled"`
    CreatedAt  time.Time `json:"created_at"`
    UpdatedAt  time.Time `json:"updated_at"`
}
```

---

## Plugin Interfaces

All plugin interfaces are defined here. Implementations live in the component that
owns them — the API server owns `SecretsProvider`, the controller owns `AgentSelector`
and `NodeExecutor`, the controller's git sync owns `InventorySource`.

```go
// AgentTransport abstracts the communication protocol between the controller and agents.
// The reconciliation loop calls Dispatch() — it never constructs HTTP requests directly.
// V1 implementation: REST (in /controller/transport/rest.go)
// Future: gRPC bidirectional stream (in /controller/transport/grpc.go)
type AgentTransport interface {
    // Dispatch sends a job assignment to a specific agent.
    Dispatch(ctx context.Context, agentID string, job JobPayload) error

    // Cancel signals a running job on an agent to stop.
    Cancel(ctx context.Context, agentID string, jobID string) error
}

// JobPayload is everything an agent needs to execute a job.
// Constructed by the controller from DB records before calling Dispatch().
type JobPayload struct {
    JobID       string
    Playbook    string
    Inventory   string          // inventory file content as text
    Credentials []SecretValue   // decrypted for transit, agent re-encrypts locally
    ExtraVars   map[string]any
    WorkspaceDir string         // where the repo is cloned on the agent host
}

// SecretsProvider resolves a credential reference to its secret value.
// V1 implementation: postgres-encrypted (in /api/secrets/postgres.go)
// Future: HashiCorp Vault, AWS Secrets Manager, Azure Key Vault
type SecretsProvider interface {
    Get(ctx context.Context, ref string) (SecretValue, error)
    Ping(ctx context.Context) error
}

// InventorySource fetches raw inventory data from an external source.
// V1 implementation: git_file (in /controller/inventory/gitfile.go)
// Future: AWS EC2, Terraform state, static upload
type InventorySource interface {
    Sync(ctx context.Context) (RawInventory, error)
    Schema() SourceSchema
}

type RawInventory struct {
    Content string // INI or YAML format ansible inventory
}

type SourceSchema struct {
    Fields []SourceField
}

type SourceField struct {
    Name     string
    Label    string
    Type     string // "string" | "secret" | "boolean"
    Required bool
}

// AgentSelector chooses which agent should receive a job from a list of available agents.
// V1 implementation: least-busy (in /controller/selector/leastbusy.go)
// Future: label affinity, weighted round-robin, capability matching
type AgentSelector interface {
    Select(ctx context.Context, available []Agent) (*Agent, error)
}

// NodeExecutor executes a single workflow node.
// V1 implementations: JobTemplateExecutor, ApprovalExecutor
// Future: SubWorkflowExecutor, DecisionGateExecutor
type NodeExecutor interface {
    Execute(ctx context.Context, node WorkflowNode, wfCtx *WorkflowContext) error
}

// WorkflowContext is the facts bag that travels with a workflow run.
// Jobs populate it via set_stats output. Decision gates (v2) read from it.
type WorkflowContext struct {
    RunID string
    Facts map[string]any
}

func (c *WorkflowContext) Merge(facts map[string]any) {
    for k, v := range facts {
        c.Facts[k] = v
    }
}
```

---

## Rules for This Package

1. **No imports from component packages** (`/api`, `/controller`, `/agent`). This package
   is a dependency of those packages, not the other way around.

2. **No business logic.** Type definitions, interface definitions, constants, and
   simple value methods only.

3. **No database imports.** Types here are Go structs. Database scanning is the
   responsibility of each component's repository layer.

4. **JSON tags on everything.** These types cross the API boundary.

5. **String types for extensible fields.** `WorkflowNode.Type`, `WorkflowEdge.Condition`,
   `Inventory.SourceType`, `Credential.Type` are all strings — not enums with a fixed
   set of values. This is intentional. New values are added via new implementations,
   not by changing the type definition.