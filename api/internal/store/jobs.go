package store

import (
	"context"
	"strconv"

	"github.com/donnie-ellis/aop/pkg/types"
	"github.com/google/uuid"
)

func (s *Store) CreateJob(ctx context.Context, templateID uuid.UUID, workflowRunID *uuid.UUID, extraVars map[string]any) (*types.Job, error) {
	if extraVars == nil {
		extraVars = map[string]any{}
	}
	var j types.Job
	err := s.pool.QueryRow(ctx,
		`INSERT INTO jobs (template_id, workflow_run_id, extra_vars)
		 VALUES ($1, $2, $3)
		 RETURNING id, template_id, agent_id, workflow_run_id, status, extra_vars, facts, failure_reason, created_at, updated_at, started_at, ended_at`,
		templateID, workflowRunID, extraVars,
	).Scan(&j.ID, &j.TemplateID, &j.AgentID, &j.WorkflowRunID, &j.Status, &j.ExtraVars, &j.Facts, &j.FailureReason, &j.CreatedAt, &j.UpdatedAt, &j.StartedAt, &j.EndedAt)
	return &j, err
}

func (s *Store) GetJob(ctx context.Context, id uuid.UUID) (*types.Job, error) {
	var j types.Job
	err := s.pool.QueryRow(ctx,
		`SELECT id, template_id, agent_id, workflow_run_id, status, extra_vars, facts, failure_reason, created_at, updated_at, started_at, ended_at
		 FROM jobs WHERE id = $1`,
		id,
	).Scan(&j.ID, &j.TemplateID, &j.AgentID, &j.WorkflowRunID, &j.Status, &j.ExtraVars, &j.Facts, &j.FailureReason, &j.CreatedAt, &j.UpdatedAt, &j.StartedAt, &j.EndedAt)
	if isNotFound(err) {
		return nil, ErrNotFound
	}
	return &j, err
}

type ListJobsFilter struct {
	Status     types.JobStatus
	TemplateID *uuid.UUID
	AgentID    *uuid.UUID
}

func (s *Store) ListJobs(ctx context.Context, f ListJobsFilter) ([]types.Job, error) {
	query := `SELECT id, template_id, agent_id, workflow_run_id, status, extra_vars, facts, failure_reason, created_at, updated_at, started_at, ended_at
	          FROM jobs WHERE 1=1`
	args := []any{}
	n := 1

	if f.Status != "" {
		query += ` AND status = $` + itoa(n)
		args = append(args, f.Status)
		n++
	}
	if f.TemplateID != nil {
		query += ` AND template_id = $` + itoa(n)
		args = append(args, *f.TemplateID)
		n++
	}
	if f.AgentID != nil {
		query += ` AND agent_id = $` + itoa(n)
		args = append(args, *f.AgentID)
		n++
	}
	query += ` ORDER BY created_at DESC LIMIT 200`

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var jobs []types.Job
	for rows.Next() {
		var j types.Job
		if err := rows.Scan(&j.ID, &j.TemplateID, &j.AgentID, &j.WorkflowRunID, &j.Status, &j.ExtraVars, &j.Facts, &j.FailureReason, &j.CreatedAt, &j.UpdatedAt, &j.StartedAt, &j.EndedAt); err != nil {
			return nil, err
		}
		jobs = append(jobs, j)
	}
	return jobs, rows.Err()
}

func (s *Store) UpdateJobStatus(ctx context.Context, id uuid.UUID, status types.JobStatus, agentID *uuid.UUID, failureReason *string) error {
	_, err := s.pool.Exec(ctx,
		`UPDATE jobs
		 SET status=$1,
		     agent_id=COALESCE($2, agent_id),
		     failure_reason=COALESCE($3, failure_reason),
		     started_at=CASE WHEN $1='running' AND started_at IS NULL THEN now() ELSE started_at END,
		     ended_at=CASE WHEN $4 THEN now() ELSE ended_at END
		 WHERE id=$5`,
		status, agentID, failureReason, status == types.JobStatusSuccess || status == types.JobStatusFailed || status == types.JobStatusCancelled, id,
	)
	return err
}

func (s *Store) SetJobFacts(ctx context.Context, id uuid.UUID, facts map[string]any) error {
	_, err := s.pool.Exec(ctx,
		`UPDATE jobs SET facts=$1 WHERE id=$2`,
		facts, id,
	)
	return err
}

func (s *Store) AppendJobLogs(ctx context.Context, jobID uuid.UUID, lines []types.JobLogLine) error {
	for _, l := range lines {
		_, err := s.pool.Exec(ctx,
			`INSERT INTO job_logs (job_id, seq, line, stream)
			 VALUES ($1, $2, $3, $4)
			 ON CONFLICT (job_id, seq) DO NOTHING`,
			jobID, l.Seq, l.Line, l.Stream,
		)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) GetJobLogs(ctx context.Context, jobID uuid.UUID) ([]types.JobLogLine, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT seq, line, stream FROM job_logs WHERE job_id=$1 ORDER BY seq`,
		jobID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var lines []types.JobLogLine
	for rows.Next() {
		var l types.JobLogLine
		if err := rows.Scan(&l.Seq, &l.Line, &l.Stream); err != nil {
			return nil, err
		}
		lines = append(lines, l)
	}
	return lines, rows.Err()
}

func (s *Store) AddJobStatusEvent(ctx context.Context, jobID uuid.UUID, from, to types.JobStatus, reason string) error {
	_, err := s.pool.Exec(ctx,
		`INSERT INTO job_status_events (job_id, from_status, to_status, reason) VALUES ($1, $2, $3, $4)`,
		jobID, from, to, reason,
	)
	return err
}

func itoa(n int) string {
	return strconv.Itoa(n)
}
