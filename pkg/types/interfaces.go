package types

import "context"

// AgentTransport abstracts the communication protocol between the
// controller and agents.
// V1 implementation: REST (/controller/transport/rest.go)
// Future: gRPC (/controller/transport/grpc.go)
type AgentTransport interface {
	Dispatch(ctx context.Context, agentID string, job JobPayload) error
	Cancel(ctx context.Context, agentID string, jobID string) error
}

// SecretsProvider resolves a credential reference to its secret value.
// V1 implementation: Postgres-encrypted (/api/secrets/postgres.go)
// Future: HashiCorp Vault, AWS Secrets Manager
type SecretsProvider interface {
	Get(ctx context.Context, ref string) (SecretValue, error)
	Ping(ctx context.Context) error
}

// InventorySource fetches raw inventory data from an external source.
// V1 implementation: git_file (/controller/inventory/gitfile.go)
// Future: AWS EC2, Terraform state
type InventorySource interface {
	Sync(ctx context.Context) (RawInventory, error)
	Schema() SourceSchema
}

type RawInventory struct {
	Content string
}

type SourceSchema struct {
	Fields []SourceField
}

type SourceField struct {
	Name     string
	Label    string
	Type     string
	Required bool
}

// AgentSelector chooses which agent should receive a job.
// V1 implementation: least-busy (/controller/selector/leastbusy.go)
// Future: label affinity, weighted round-robin
type AgentSelector interface {
	Select(ctx context.Context, available []Agent) (*Agent, error)
}

// NodeExecutor executes a single workflow node.
// V1 implementations: JobTemplateExecutor, ApprovalExecutor
// Future: SubWorkflowExecutor, DecisionGateExecutor
type NodeExecutor interface {
	Execute(ctx context.Context, node WorkflowNode, wfCtx *WorkflowContext) error
}
