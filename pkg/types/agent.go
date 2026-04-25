package types

import (
	"time"

	"github.com/google/uuid"
)

// ---------------------------------------------------------------------------
// Agent
// ---------------------------------------------------------------------------

// AgentStatus reflects whether an agent is reachable and able to accept work.
type AgentStatus string

const (
	// AgentStatusOnline means the agent has heartbeated within AOP_AGENT_HEARTBEAT_TTL.
	AgentStatusOnline AgentStatus = "online"

	// AgentStatusOffline means the agent's last heartbeat is older than the TTL.
	AgentStatusOffline AgentStatus = "offline"

	// AgentStatusDraining means the agent will not accept new jobs but is finishing
	// current work. Set by an operator before maintenance.
	AgentStatusDraining AgentStatus = "draining"
)

// Agent represents a registered AOP agent process. The agent self-registers on
// startup and sends periodic heartbeats. The controller reads agent records from
// Postgres to make dispatch decisions.
type Agent struct {
	ID      uuid.UUID   `json:"id"      db:"id"`
	Name    string      `json:"name"    db:"name"` // human-readable, e.g. "agent-prod-01"
	Address string      `json:"address" db:"address"` // base URL for REST dispatch, e.g. "http://10.0.1.5:8080"
	Status  AgentStatus `json:"status"  db:"status"`

	// Labels are arbitrary key/value pairs used for agent selection affinity.
	// Example: {"env": "prod", "region": "us-east-1"}
	Labels map[string]string `json:"labels" db:"labels"`

	// Capacity is the maximum number of concurrent jobs this agent will accept.
	// 0 means unlimited (not recommended for production).
	Capacity int `json:"capacity" db:"capacity"`

	// RunningJobs is the count of jobs currently in running state on this agent.
	// Populated at query time, not stored directly.
	RunningJobs int `json:"running_jobs" db:"-"`

	LastHeartbeatAt *time.Time `json:"last_heartbeat_at" db:"last_heartbeat_at"`
	RegisteredAt    time.Time  `json:"registered_at"     db:"registered_at"`
	UpdatedAt       time.Time  `json:"updated_at"        db:"updated_at"`
}

// IsAvailable returns true if the agent can accept a new job dispatch.
// This is a convenience check; the authoritative evaluation is in AgentSelector.
func (a *Agent) IsAvailable(heartbeatTTL time.Duration) bool {
	if a.Status != AgentStatusOnline {
		return false
	}
	if a.LastHeartbeatAt == nil {
		return false
	}
	if time.Since(*a.LastHeartbeatAt) > heartbeatTTL {
		return false
	}
	if a.Capacity > 0 && a.RunningJobs >= a.Capacity {
		return false
	}
	return true
}

// ---------------------------------------------------------------------------
// AgentHeartbeat — sent by the agent, written by the API server
// ---------------------------------------------------------------------------

// AgentHeartbeat is the payload the agent sends on its heartbeat tick.
// The API server validates it and upserts the agent record in Postgres.
type AgentHeartbeat struct {
	AgentID     uuid.UUID `json:"agent_id"`
	RunningJobs int       `json:"running_jobs"`
	// Version is the agent binary version string for observability.
	Version string `json:"version"`
}
