package types

import "context"

// InventorySource fetches raw inventory data from an external source and
// returns it as text for the controller to parse into InventoryHost records.
//
// v1 implementation: GitFileSource in /controller/inventory/gitfile.go
// Future: AWS EC2 dynamic inventory, Terraform state, static upload
type InventorySource interface {
	Sync(ctx context.Context) (RawInventory, error)
	Schema() SourceSchema
}

// RawInventory is the unparsed text returned by an InventorySource sync.
// Content is standard Ansible INI or YAML inventory format.
type RawInventory struct {
	Content string
}

// SourceSchema describes the configuration fields an InventorySource requires.
// The UI renders this as a dynamic form when creating an inventory of this type.
type SourceSchema struct {
	Fields []SourceField
}

// SourceField describes a single configuration field for an InventorySource.
type SourceField struct {
	Name     string
	Label    string
	Type     string // "string" | "secret" | "boolean"
	Required bool
}

// NodeExecutor executes a single node within a workflow run.
// The workflow engine in the controller calls Execute for each ready node and
// waits for it to return before evaluating outbound edges.
//
// v1 implementations: JobTemplateExecutor, ApprovalExecutor
//   (both in /controller/workflow/)
// Future: SubWorkflowExecutor, DecisionGateExecutor
type NodeExecutor interface {
	Execute(ctx context.Context, node WorkflowNode, wfCtx *WorkflowContext) error
}
