// Package transport implements the AgentTransport interface using direct
// HTTP POST calls to each agent's listener address.
package transport

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/donnie-ellis/aop/pkg/types"
	"github.com/google/uuid"
)

// addressResolver is a narrow DB interface so the transport can look up an
// agent's HTTP address without importing the full store package.
type addressResolver interface {
	GetAgentAddress(ctx context.Context, agentID uuid.UUID) (string, error)
}

// RESTTransport dispatches jobs over HTTP to each agent's listener.
// The controller makes a direct POST to agent.Address/dispatch.
// Agents must be reachable from the controller on the network.
type RESTTransport struct {
	resolver addressResolver
	http     *http.Client
}

// New returns a RESTTransport. resolver is typically the store.
func New(resolver addressResolver) *RESTTransport {
	return &RESTTransport{
		resolver: resolver,
		http:     &http.Client{Timeout: 30 * time.Second},
	}
}

// Dispatch serialises payload and POSTs it to the agent's /dispatch endpoint.
// Returns an error if the agent is unreachable or returns a non-2xx status.
func (t *RESTTransport) Dispatch(ctx context.Context, agentID uuid.UUID, payload types.JobPayload) error {
	addr, err := t.resolver.GetAgentAddress(ctx, agentID)
	if err != nil {
		return fmt.Errorf("dispatch: resolve agent address: %w", err)
	}

	body, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, addr+"/dispatch", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("dispatch: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := t.http.Do(req)
	if err != nil {
		return fmt.Errorf("dispatch: post to agent: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusAccepted {
		return fmt.Errorf("dispatch: agent returned %d", resp.StatusCode)
	}
	return nil
}

// Cancel sends a cancellation request to the agent's /jobs/{id}/cancel endpoint.
func (t *RESTTransport) Cancel(ctx context.Context, agentID uuid.UUID, jobID uuid.UUID) error {
	addr, err := t.resolver.GetAgentAddress(ctx, agentID)
	if err != nil {
		return fmt.Errorf("cancel: resolve agent address: %w", err)
	}

	url := fmt.Sprintf("%s/jobs/%s/cancel", addr, jobID)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, nil)
	if err != nil {
		return fmt.Errorf("cancel: build request: %w", err)
	}

	resp, err := t.http.Do(req)
	if err != nil {
		return fmt.Errorf("cancel: post to agent: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusAccepted && resp.StatusCode != http.StatusNotFound {
		return fmt.Errorf("cancel: agent returned %d", resp.StatusCode)
	}
	return nil
}
