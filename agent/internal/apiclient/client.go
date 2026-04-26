// Package apiclient handles all HTTP communication from the agent back to the
// API server: registration, heartbeats, log uploads, and result reporting.
package apiclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/donnie-ellis/aop/pkg/types"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// Client wraps the API server's agent-facing endpoints.
// Call Register first — it populates AgentID and the bearer token used by
// all subsequent requests.
type Client struct {
	base    string
	token   string
	agentID uuid.UUID
	http    *http.Client
	log     zerolog.Logger
}

func New(baseURL string, log zerolog.Logger) *Client {
	return &Client{
		base: baseURL,
		http: &http.Client{Timeout: 30 * time.Second},
		log:  log,
	}
}

func (c *Client) AgentID() uuid.UUID { return c.agentID }

// Register calls POST /agents/register and stores the returned agent ID and
// token for use in subsequent requests.
func (c *Client) Register(ctx context.Context, regToken, name, address string, labels map[string]string, capacity int) error {
	body, _ := json.Marshal(map[string]any{
		"name":     name,
		"address":  address,
		"labels":   labels,
		"capacity": capacity,
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.base+"/agents/register", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Registration-Token", regToken)

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("register: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("register: api returned %d", resp.StatusCode)
	}

	var r struct {
		AgentID uuid.UUID `json:"agent_id"`
		Token   string    `json:"token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		return fmt.Errorf("register: decode response: %w", err)
	}
	c.agentID = r.AgentID
	c.token = r.Token
	c.log.Info().Str("agent_id", r.AgentID.String()).Msg("registered with API")
	return nil
}

// Heartbeat calls POST /agent/heartbeat.
func (c *Client) Heartbeat(ctx context.Context, runningJobs int) error {
	body, _ := json.Marshal(types.AgentHeartbeat{
		AgentID:     c.agentID,
		RunningJobs: runningJobs,
		Version:     "v0.1.0",
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.base+"/agent/heartbeat", bytes.NewReader(body))
	if err != nil {
		return err
	}
	c.setBearer(req)
	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("heartbeat: %w", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("heartbeat: api returned %d", resp.StatusCode)
	}
	return nil
}

// PostLogs ships a batch of log lines to POST /agent/jobs/{id}/logs.
func (c *Client) PostLogs(ctx context.Context, jobID uuid.UUID, lines []types.JobLogLine) error {
	if len(lines) == 0 {
		return nil
	}
	body, _ := json.Marshal(lines)
	url := fmt.Sprintf("%s/agent/jobs/%s/logs", c.base, jobID)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	c.setBearer(req)
	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("post logs: %w", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("post logs: api returned %d", resp.StatusCode)
	}
	return nil
}

// PostResult calls POST /agent/jobs/{id}/result to report job completion.
func (c *Client) PostResult(ctx context.Context, jobID uuid.UUID, status types.JobStatus, exitCode int, facts map[string]any) error {
	if facts == nil {
		facts = map[string]any{}
	}
	body, _ := json.Marshal(map[string]any{
		"status":    status,
		"exit_code": exitCode,
		"facts":     facts,
	})
	url := fmt.Sprintf("%s/agent/jobs/%s/result", c.base, jobID)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	c.setBearer(req)
	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("post result: %w", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("post result: api returned %d", resp.StatusCode)
	}
	return nil
}

func (c *Client) setBearer(req *http.Request) {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.token)
}
