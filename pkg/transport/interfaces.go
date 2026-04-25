// Package transport defines the interfaces through which the controller
// communicates with agents. Concrete implementations live in
// /controller/transport/. Nothing outside the controller should depend on
// this package directly.
package transport

import (
	"context"

	"github.com/google/uuid"
	"github.com/donnie-ellis/aop/pkg/types"
)

// ---------------------------------------------------------------------------
// AgentTransport
// ---------------------------------------------------------------------------

// AgentTransport abstracts the wire protocol used to dispatch jobs to agents.
// The reconciliation loop only ever calls Dispatch and Cancel — it never
// constructs HTTP (or gRPC) requests directly.
//
// v1 implementation: RESTTransport in /controller/transport/rest.go
//   Push model — controller makes a direct HTTP POST to the agent's listener
//   address (Agent.Address). The agent must be reachable from the controller.
//
// v2 seam: a pull/long-poll or gRPC-stream transport can implement this same
//   interface. The reconciliation loop never changes; only the implementation
//   differs. A pull implementation would write the assignment to a queue and
//   return once the agent has acknowledged it via its poll endpoint.
//
// Switching implementations is a config change; the reconciliation loop
// never changes.
type AgentTransport interface {
	// Dispatch sends a job to the specified agent. It returns an error if the
	// agent cannot be reached or rejects the job. On success the agent is
	// expected to begin execution and report status via the callback URL
	// embedded in the payload.
	Dispatch(ctx context.Context, agentID uuid.UUID, job types.JobPayload) error

	// Cancel instructs the agent to abort a running job. The agent may not
	// honour the request immediately; the job status transitions to cancelled
	// asynchronously when the agent confirms.
	Cancel(ctx context.Context, agentID uuid.UUID, jobID uuid.UUID) error
}

// ---------------------------------------------------------------------------
// AgentSelector
// ---------------------------------------------------------------------------

// AgentSelector chooses one agent from a slice of available agents.
// "Available" is pre-filtered by the caller (heartbeat TTL, capacity).
//
// v1 implementation: LeastBusySelector — fewest running jobs; recency as
// tiebreak. Future implementations can add label affinity, weighted
// round-robin, or capability matching without touching the dispatch loop.
type AgentSelector interface {
	// Select returns the chosen agent, or an error if no suitable agent
	// exists in the provided slice (e.g. the slice is empty).
	Select(ctx context.Context, available []types.Agent) (*types.Agent, error)
}
