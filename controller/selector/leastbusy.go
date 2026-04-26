// Package selector implements the AgentSelector interface.
package selector

import (
	"context"
	"errors"

	"github.com/donnie-ellis/aop/pkg/types"
)

// LeastBusySelector picks the agent with the fewest in-flight jobs.
// When tied, it picks the agent that heartbeated most recently,
// which skews toward agents that are known to be alive.
type LeastBusySelector struct{}

func New() *LeastBusySelector { return &LeastBusySelector{} }

// Select returns the best agent from the available slice. The slice is
// pre-filtered by the reconciler (heartbeat TTL + capacity check), so
// this method only needs to pick among viable candidates.
func (s *LeastBusySelector) Select(_ context.Context, available []types.Agent) (*types.Agent, error) {
	if len(available) == 0 {
		return nil, errors.New("no agents available")
	}

	best := &available[0]
	for i := 1; i < len(available); i++ {
		a := &available[i]
		if a.RunningJobs < best.RunningJobs {
			best = a
			continue
		}
		if a.RunningJobs == best.RunningJobs {
			if a.LastHeartbeatAt != nil && (best.LastHeartbeatAt == nil || a.LastHeartbeatAt.After(*best.LastHeartbeatAt)) {
				best = a
			}
		}
	}
	return best, nil
}
