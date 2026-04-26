// Package store provides all database operations for the controller.
// Every method maps to a single database interaction; no business logic lives here.
package store

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/donnie-ellis/aop/pkg/types"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Store wraps a pgx connection pool and exposes controller-specific queries.
type Store struct {
	db *pgxpool.Pool
}

func New(db *pgxpool.Pool) *Store {
	return &Store{db: db}
}

// ---------------------------------------------------------------------------
// Jobs
// ---------------------------------------------------------------------------

// GetPendingJobs returns up to 50 pending jobs, locking them with SKIP LOCKED
// so concurrent controller instances don't double-dispatch.
func (s *Store) GetPendingJobs(ctx context.Context) ([]types.Job, error) {
	rows, err := s.db.Query(ctx, `
		SELECT id, template_id, agent_id, workflow_run_id, status, extra_vars, facts, failure_reason,
		       created_at, updated_at, started_at, ended_at
		FROM jobs
		WHERE status = 'pending'
		ORDER BY created_at
		LIMIT 50
		FOR UPDATE SKIP LOCKED
	`)
	if err != nil {
		return nil, fmt.Errorf("get pending jobs: %w", err)
	}
	defer rows.Close()
	return scanJobs(rows)
}

// GetStuckDispatchedJobs returns dispatched jobs that haven't moved to running
// within the given timeout, indicating the agent never acknowledged them.
func (s *Store) GetStuckDispatchedJobs(ctx context.Context, timeout time.Duration) ([]types.Job, error) {
	rows, err := s.db.Query(ctx, `
		SELECT id, template_id, agent_id, workflow_run_id, status, extra_vars, facts, failure_reason,
		       created_at, updated_at, started_at, ended_at
		FROM jobs
		WHERE status = 'dispatched' AND updated_at < now() - $1::interval
		FOR UPDATE SKIP LOCKED
	`, timeout.String())
	if err != nil {
		return nil, fmt.Errorf("get stuck dispatched jobs: %w", err)
	}
	defer rows.Close()
	return scanJobs(rows)
}

// GetStuckRunningJobs returns running jobs that haven't received a log or
// heartbeat update within the given timeout, indicating the agent died.
func (s *Store) GetStuckRunningJobs(ctx context.Context, timeout time.Duration) ([]types.Job, error) {
	rows, err := s.db.Query(ctx, `
		SELECT id, template_id, agent_id, workflow_run_id, status, extra_vars, facts, failure_reason,
		       created_at, updated_at, started_at, ended_at
		FROM jobs
		WHERE status = 'running' AND updated_at < now() - $1::interval
		FOR UPDATE SKIP LOCKED
	`, timeout.String())
	if err != nil {
		return nil, fmt.Errorf("get stuck running jobs: %w", err)
	}
	defer rows.Close()
	return scanJobs(rows)
}

// TransitionJob atomically moves a job from fromStatus to toStatus if the
// current status matches. Returns false (no error) if the row was not updated
// (e.g. another controller already transitioned it).
func (s *Store) TransitionJob(ctx context.Context, jobID uuid.UUID, fromStatus, toStatus types.JobStatus, agentID *uuid.UUID, reason string) (bool, error) {
	tag, err := s.db.Exec(ctx, `
		UPDATE jobs
		SET status = $2,
		    agent_id = COALESCE($3, agent_id),
		    failure_reason = CASE WHEN $4 != '' THEN $4 ELSE failure_reason END,
		    started_at = CASE WHEN $2 = 'running' AND started_at IS NULL THEN now() ELSE started_at END,
		    ended_at   = CASE WHEN $2 IN ('success','failed','cancelled') THEN now() ELSE ended_at END,
		    updated_at = now()
		WHERE id = $1 AND status = $5
	`, jobID, toStatus, agentID, reason, fromStatus)
	if err != nil {
		return false, fmt.Errorf("transition job %s %s→%s: %w", jobID, fromStatus, toStatus, err)
	}
	return tag.RowsAffected() == 1, nil
}

// CreateJob inserts a new job in pending status for the given template.
// extraVars are merged on top of the template's defaults at dispatch time.
func (s *Store) CreateJob(ctx context.Context, templateID uuid.UUID, extraVars map[string]any) (*types.Job, error) {
	if extraVars == nil {
		extraVars = map[string]any{}
	}
	varsJSON, err := json.Marshal(extraVars)
	if err != nil {
		return nil, fmt.Errorf("marshal extra_vars: %w", err)
	}

	var job types.Job
	row := s.db.QueryRow(ctx, `
		INSERT INTO jobs (template_id, extra_vars, status)
		VALUES ($1, $2, 'pending')
		RETURNING id, template_id, agent_id, workflow_run_id, status, extra_vars, facts,
		          failure_reason, created_at, updated_at, started_at, ended_at
	`, templateID, varsJSON)
	if err := scanJob(row, &job); err != nil {
		return nil, fmt.Errorf("create job: %w", err)
	}
	return &job, nil
}

// ---------------------------------------------------------------------------
// Agents
// ---------------------------------------------------------------------------

// GetAvailableAgents returns online agents whose last heartbeat is within ttl
// and who have spare capacity (running_jobs < capacity).
func (s *Store) GetAvailableAgents(ctx context.Context, ttl time.Duration) ([]types.Agent, error) {
	rows, err := s.db.Query(ctx, `
		SELECT a.id, a.name, a.address, a.status, a.labels, a.capacity, a.last_heartbeat_at,
		       COALESCE(COUNT(j.id), 0)::int AS running_jobs
		FROM agents a
		LEFT JOIN jobs j ON j.agent_id = a.id AND j.status IN ('dispatched','running')
		WHERE a.status = 'online'
		  AND a.last_heartbeat_at > now() - $1::interval
		GROUP BY a.id
		HAVING COALESCE(COUNT(j.id), 0) < a.capacity
		ORDER BY COALESCE(COUNT(j.id), 0) ASC, a.last_heartbeat_at DESC
	`, ttl.String())
	if err != nil {
		return nil, fmt.Errorf("get available agents: %w", err)
	}
	defer rows.Close()

	var agents []types.Agent
	for rows.Next() {
		var a types.Agent
		var labelsJSON []byte
		err := rows.Scan(
			&a.ID, &a.Name, &a.Address, &a.Status, &labelsJSON,
			&a.Capacity, &a.LastHeartbeatAt, &a.RunningJobs,
		)
		if err != nil {
			return nil, fmt.Errorf("scan agent: %w", err)
		}
		if err := json.Unmarshal(labelsJSON, &a.Labels); err != nil {
			a.Labels = map[string]string{}
		}
		agents = append(agents, a)
	}
	return agents, rows.Err()
}

// GetAgentAddress returns just the dispatch address for an agent by ID.
// Used by the transport layer to resolve an address from an agentID.
func (s *Store) GetAgentAddress(ctx context.Context, agentID uuid.UUID) (string, error) {
	var address string
	err := s.db.QueryRow(ctx, `SELECT address FROM agents WHERE id = $1`, agentID).Scan(&address)
	if err != nil {
		return "", fmt.Errorf("get agent address %s: %w", agentID, err)
	}
	return address, nil
}

// ---------------------------------------------------------------------------
// Job Templates + Projects
// ---------------------------------------------------------------------------

// GetJobTemplate returns a job template by ID.
func (s *Store) GetJobTemplate(ctx context.Context, id uuid.UUID) (*types.JobTemplate, error) {
	var t types.JobTemplate
	var varsJSON []byte
	err := s.db.QueryRow(ctx, `
		SELECT id, name, description, project_id, playbook, credential_id, default_extra_vars,
		       created_at, updated_at
		FROM job_templates WHERE id = $1
	`, id).Scan(
		&t.ID, &t.Name, &t.Description, &t.ProjectID, &t.Playbook, &t.CredentialID, &varsJSON,
		&t.CreatedAt, &t.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("get job template %s: %w", id, err)
	}
	if err := json.Unmarshal(varsJSON, &t.DefaultExtraVars); err != nil {
		t.DefaultExtraVars = map[string]any{}
	}
	return &t, nil
}

// GetProject returns a project by ID.
func (s *Store) GetProject(ctx context.Context, id uuid.UUID) (*types.Project, error) {
	var p types.Project
	err := s.db.QueryRow(ctx, `
		SELECT id, name, repo_url, branch, inventory_path, credential_id,
		       sync_status, last_synced_at, sync_error, created_at, updated_at
		FROM projects WHERE id = $1
	`, id).Scan(
		&p.ID, &p.Name, &p.RepoURL, &p.Branch, &p.InventoryPath, &p.CredentialID,
		&p.SyncStatus, &p.LastSyncedAt, &p.SyncError, &p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("get project %s: %w", id, err)
	}
	return &p, nil
}

// GetAllProjects returns every project for the git sync loop.
func (s *Store) GetAllProjects(ctx context.Context) ([]types.Project, error) {
	rows, err := s.db.Query(ctx, `
		SELECT id, name, repo_url, branch, inventory_path, credential_id,
		       sync_status, last_synced_at, sync_error, created_at, updated_at
		FROM projects ORDER BY created_at
	`)
	if err != nil {
		return nil, fmt.Errorf("get all projects: %w", err)
	}
	defer rows.Close()

	var projects []types.Project
	for rows.Next() {
		var p types.Project
		if err := rows.Scan(
			&p.ID, &p.Name, &p.RepoURL, &p.Branch, &p.InventoryPath, &p.CredentialID,
			&p.SyncStatus, &p.LastSyncedAt, &p.SyncError, &p.CreatedAt, &p.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan project: %w", err)
		}
		projects = append(projects, p)
	}
	return projects, rows.Err()
}

// UpdateProjectSyncStatus records the result of a git sync attempt.
func (s *Store) UpdateProjectSyncStatus(ctx context.Context, id uuid.UUID, status types.InventorySyncStatus, syncErr string) error {
	var errPtr *string
	if syncErr != "" {
		errPtr = &syncErr
	}
	_, err := s.db.Exec(ctx, `
		UPDATE projects
		SET sync_status = $2, sync_error = $3, last_synced_at = now(), updated_at = now()
		WHERE id = $1
	`, id, status, errPtr)
	return err
}

// ---------------------------------------------------------------------------
// Inventory
// ---------------------------------------------------------------------------

// GetInventoryHosts returns all inventory hosts for a project.
func (s *Store) GetInventoryHosts(ctx context.Context, projectID uuid.UUID) ([]types.InventoryHost, error) {
	rows, err := s.db.Query(ctx, `
		SELECT id, project_id, hostname, groups, vars, created_at, updated_at
		FROM inventory_hosts WHERE project_id = $1
	`, projectID)
	if err != nil {
		return nil, fmt.Errorf("get inventory hosts: %w", err)
	}
	defer rows.Close()

	var hosts []types.InventoryHost
	for rows.Next() {
		var h types.InventoryHost
		var varsJSON []byte
		if err := rows.Scan(&h.ID, &h.ProjectID, &h.Hostname, &h.Groups, &varsJSON, &h.CreatedAt, &h.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan inventory host: %w", err)
		}
		if err := json.Unmarshal(varsJSON, &h.Vars); err != nil {
			h.Vars = map[string]any{}
		}
		hosts = append(hosts, h)
	}
	return hosts, rows.Err()
}

// UpsertInventoryHosts replaces all inventory hosts for a project within a
// single transaction. The old rows are deleted and new ones are inserted.
func (s *Store) UpsertInventoryHosts(ctx context.Context, projectID uuid.UUID, hosts []types.InventoryHost) error {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, `DELETE FROM inventory_hosts WHERE project_id = $1`, projectID); err != nil {
		return fmt.Errorf("delete old hosts: %w", err)
	}

	for _, h := range hosts {
		varsJSON, _ := json.Marshal(h.Vars)
		_, err := tx.Exec(ctx, `
			INSERT INTO inventory_hosts (project_id, hostname, groups, vars)
			VALUES ($1, $2, $3, $4)
		`, projectID, h.Hostname, h.Groups, varsJSON)
		if err != nil {
			return fmt.Errorf("insert host %s: %w", h.Hostname, err)
		}
	}

	return tx.Commit(ctx)
}

// ---------------------------------------------------------------------------
// Credentials (SecretsProvider implementation)
// ---------------------------------------------------------------------------

// GetCredentialWithSecret returns the credential record including encrypted data.
// This is used by the SecretsProvider — do not expose it via API handlers.
func (s *Store) GetCredentialWithSecret(ctx context.Context, id uuid.UUID) (*types.Credential, error) {
	var c types.Credential
	err := s.db.QueryRow(ctx, `
		SELECT id, name, kind, description, encrypted_data, created_at, updated_at
		FROM credentials WHERE id = $1
	`, id).Scan(&c.ID, &c.Name, &c.Kind, &c.Description, &c.EncryptedData, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("get credential %s: %w", id, err)
	}
	return &c, nil
}

// ---------------------------------------------------------------------------
// Schedules
// ---------------------------------------------------------------------------

// GetActiveSchedules returns all enabled schedules.
func (s *Store) GetActiveSchedules(ctx context.Context) ([]types.Schedule, error) {
	rows, err := s.db.Query(ctx, `
		SELECT id, name, template_id, cron_expr, timezone, enabled, extra_vars,
		       last_run_at, next_run_at, created_at, updated_at
		FROM schedules
		WHERE enabled = true
		ORDER BY created_at
	`)
	if err != nil {
		return nil, fmt.Errorf("get active schedules: %w", err)
	}
	defer rows.Close()

	var schedules []types.Schedule
	for rows.Next() {
		var sc types.Schedule
		var varsJSON []byte
		if err := rows.Scan(
			&sc.ID, &sc.Name, &sc.TemplateID, &sc.CronExpr, &sc.Timezone, &sc.Enabled, &varsJSON,
			&sc.LastRunAt, &sc.NextRunAt, &sc.CreatedAt, &sc.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan schedule: %w", err)
		}
		if err := json.Unmarshal(varsJSON, &sc.ExtraVars); err != nil {
			sc.ExtraVars = map[string]any{}
		}
		schedules = append(schedules, sc)
	}
	return schedules, rows.Err()
}

// UpdateScheduleLastRun records that a schedule fired and sets next_run_at.
func (s *Store) UpdateScheduleLastRun(ctx context.Context, scheduleID uuid.UUID, nextRunAt *time.Time) error {
	_, err := s.db.Exec(ctx, `
		UPDATE schedules
		SET last_run_at = now(), next_run_at = $2, updated_at = now()
		WHERE id = $1
	`, scheduleID, nextRunAt)
	return err
}

// ---------------------------------------------------------------------------
// Internal scan helpers
// ---------------------------------------------------------------------------

func scanJobs(rows pgx.Rows) ([]types.Job, error) {
	var jobs []types.Job
	for rows.Next() {
		var j types.Job
		if err := scanJobRow(rows, &j); err != nil {
			return nil, err
		}
		jobs = append(jobs, j)
	}
	return jobs, rows.Err()
}

// scanJobRow scans a pgx.Rows cursor into a Job (used inside scanJobs).
func scanJobRow(rows pgx.Rows, j *types.Job) error {
	var extraVarsJSON, factsJSON []byte
	err := rows.Scan(
		&j.ID, &j.TemplateID, &j.AgentID, &j.WorkflowRunID, &j.Status,
		&extraVarsJSON, &factsJSON, &j.FailureReason,
		&j.CreatedAt, &j.UpdatedAt, &j.StartedAt, &j.EndedAt,
	)
	if err != nil {
		return fmt.Errorf("scan job: %w", err)
	}
	if err := json.Unmarshal(extraVarsJSON, &j.ExtraVars); err != nil {
		j.ExtraVars = map[string]any{}
	}
	if err := json.Unmarshal(factsJSON, &j.Facts); err != nil {
		j.Facts = map[string]any{}
	}
	return nil
}

// scanJob scans a pgx.Row (single row) into a Job.
func scanJob(row pgx.Row, j *types.Job) error {
	var extraVarsJSON, factsJSON []byte
	err := row.Scan(
		&j.ID, &j.TemplateID, &j.AgentID, &j.WorkflowRunID, &j.Status,
		&extraVarsJSON, &factsJSON, &j.FailureReason,
		&j.CreatedAt, &j.UpdatedAt, &j.StartedAt, &j.EndedAt,
	)
	if err != nil {
		return fmt.Errorf("scan job: %w", err)
	}
	if err := json.Unmarshal(extraVarsJSON, &j.ExtraVars); err != nil {
		j.ExtraVars = map[string]any{}
	}
	if err := json.Unmarshal(factsJSON, &j.Facts); err != nil {
		j.Facts = map[string]any{}
	}
	return nil
}
