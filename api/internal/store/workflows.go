package store

import (
	"context"

	"github.com/donnie-ellis/aop/pkg/types"
	"github.com/google/uuid"
)

func (s *Store) CreateWorkflow(ctx context.Context, name, description string) (*types.Workflow, error) {
	var w types.Workflow
	err := s.pool.QueryRow(ctx,
		`INSERT INTO workflows (name, description) VALUES ($1, $2)
		 RETURNING id, name, description, created_at, updated_at`,
		name, description,
	).Scan(&w.ID, &w.Name, &w.Description, &w.CreatedAt, &w.UpdatedAt)
	return &w, err
}

func (s *Store) GetWorkflow(ctx context.Context, id uuid.UUID) (*types.Workflow, error) {
	var w types.Workflow
	err := s.pool.QueryRow(ctx,
		`SELECT id, name, description, created_at, updated_at FROM workflows WHERE id = $1`,
		id,
	).Scan(&w.ID, &w.Name, &w.Description, &w.CreatedAt, &w.UpdatedAt)
	if isNotFound(err) {
		return nil, ErrNotFound
	}
	return &w, err
}

func (s *Store) ListWorkflows(ctx context.Context) ([]types.Workflow, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id, name, description, created_at, updated_at FROM workflows ORDER BY name`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var workflows []types.Workflow
	for rows.Next() {
		var w types.Workflow
		if err := rows.Scan(&w.ID, &w.Name, &w.Description, &w.CreatedAt, &w.UpdatedAt); err != nil {
			return nil, err
		}
		workflows = append(workflows, w)
	}
	return workflows, rows.Err()
}

func (s *Store) UpdateWorkflow(ctx context.Context, id uuid.UUID, name, description string) (*types.Workflow, error) {
	var w types.Workflow
	err := s.pool.QueryRow(ctx,
		`UPDATE workflows SET name=$1, description=$2 WHERE id=$3
		 RETURNING id, name, description, created_at, updated_at`,
		name, description, id,
	).Scan(&w.ID, &w.Name, &w.Description, &w.CreatedAt, &w.UpdatedAt)
	if isNotFound(err) {
		return nil, ErrNotFound
	}
	return &w, err
}

func (s *Store) DeleteWorkflow(ctx context.Context, id uuid.UUID) error {
	tag, err := s.pool.Exec(ctx, `DELETE FROM workflows WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) UpsertWorkflowNodes(ctx context.Context, workflowID uuid.UUID, nodes []types.WorkflowNode) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, `DELETE FROM workflow_nodes WHERE workflow_id=$1`, workflowID); err != nil {
		return err
	}
	for _, n := range nodes {
		if n.ExtraVars == nil {
			n.ExtraVars = map[string]any{}
		}
		if _, err := tx.Exec(ctx,
			`INSERT INTO workflow_nodes (id, workflow_id, kind, resource_id, label, extra_vars)
			 VALUES ($1, $2, $3, $4, $5, $6)`,
			n.ID, workflowID, n.Kind, n.ResourceID, n.Label, n.ExtraVars,
		); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

func (s *Store) UpsertWorkflowEdges(ctx context.Context, workflowID uuid.UUID, edges []types.WorkflowEdge) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, `DELETE FROM workflow_edges WHERE workflow_id=$1`, workflowID); err != nil {
		return err
	}
	for _, e := range edges {
		if _, err := tx.Exec(ctx,
			`INSERT INTO workflow_edges (id, workflow_id, source_node_id, target_node_id, condition)
			 VALUES ($1, $2, $3, $4, $5)`,
			e.ID, workflowID, e.SourceNodeID, e.TargetNodeID, e.Condition,
		); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

func (s *Store) GetWorkflowNodes(ctx context.Context, workflowID uuid.UUID) ([]types.WorkflowNode, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id, workflow_id, kind, resource_id, label, extra_vars, created_at, updated_at
		 FROM workflow_nodes WHERE workflow_id=$1`,
		workflowID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var nodes []types.WorkflowNode
	for rows.Next() {
		var n types.WorkflowNode
		if err := rows.Scan(&n.ID, &n.WorkflowID, &n.Kind, &n.ResourceID, &n.Label, &n.ExtraVars, &n.CreatedAt, &n.UpdatedAt); err != nil {
			return nil, err
		}
		nodes = append(nodes, n)
	}
	return nodes, rows.Err()
}

func (s *Store) GetWorkflowEdges(ctx context.Context, workflowID uuid.UUID) ([]types.WorkflowEdge, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id, workflow_id, source_node_id, target_node_id, condition
		 FROM workflow_edges WHERE workflow_id=$1`,
		workflowID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var edges []types.WorkflowEdge
	for rows.Next() {
		var e types.WorkflowEdge
		if err := rows.Scan(&e.ID, &e.WorkflowID, &e.SourceNodeID, &e.TargetNodeID, &e.Condition); err != nil {
			return nil, err
		}
		edges = append(edges, e)
	}
	return edges, rows.Err()
}

func (s *Store) CreateWorkflowRun(ctx context.Context, workflowID uuid.UUID) (*types.WorkflowRun, error) {
	var r types.WorkflowRun
	err := s.pool.QueryRow(ctx,
		`INSERT INTO workflow_runs (workflow_id) VALUES ($1)
		 RETURNING id, workflow_id, status, context, created_at, updated_at, started_at, ended_at`,
		workflowID,
	).Scan(&r.ID, &r.WorkflowID, &r.Status, &r.Context, &r.CreatedAt, &r.UpdatedAt, &r.StartedAt, &r.EndedAt)
	return &r, err
}

func (s *Store) GetWorkflowRun(ctx context.Context, id uuid.UUID) (*types.WorkflowRun, error) {
	var r types.WorkflowRun
	err := s.pool.QueryRow(ctx,
		`SELECT id, workflow_id, status, context, created_at, updated_at, started_at, ended_at
		 FROM workflow_runs WHERE id=$1`,
		id,
	).Scan(&r.ID, &r.WorkflowID, &r.Status, &r.Context, &r.CreatedAt, &r.UpdatedAt, &r.StartedAt, &r.EndedAt)
	if isNotFound(err) {
		return nil, ErrNotFound
	}
	return &r, err
}

func (s *Store) ListWorkflowRuns(ctx context.Context, workflowID uuid.UUID) ([]types.WorkflowRun, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id, workflow_id, status, context, created_at, updated_at, started_at, ended_at
		 FROM workflow_runs WHERE workflow_id=$1 ORDER BY created_at DESC`,
		workflowID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var runs []types.WorkflowRun
	for rows.Next() {
		var r types.WorkflowRun
		if err := rows.Scan(&r.ID, &r.WorkflowID, &r.Status, &r.Context, &r.CreatedAt, &r.UpdatedAt, &r.StartedAt, &r.EndedAt); err != nil {
			return nil, err
		}
		runs = append(runs, r)
	}
	return runs, rows.Err()
}
