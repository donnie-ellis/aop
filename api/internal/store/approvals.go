package store

import (
	"context"

	"github.com/donnie-ellis/aop/pkg/types"
	"github.com/google/uuid"
)

func (s *Store) GetApprovalRequest(ctx context.Context, id uuid.UUID) (*types.ApprovalRequest, error) {
	var a types.ApprovalRequest
	err := s.pool.QueryRow(ctx,
		`SELECT id, workflow_run_id, node_id, status, reviewed_by, review_note, created_at, resolved_at
		 FROM approval_requests WHERE id=$1`,
		id,
	).Scan(&a.ID, &a.WorkflowRunID, &a.NodeID, &a.Status, &a.ReviewedBy, &a.ReviewNote, &a.CreatedAt, &a.ResolvedAt)
	if isNotFound(err) {
		return nil, ErrNotFound
	}
	return &a, err
}

func (s *Store) ListPendingApprovals(ctx context.Context) ([]types.ApprovalRequest, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id, workflow_run_id, node_id, status, reviewed_by, review_note, created_at, resolved_at
		 FROM approval_requests WHERE status='pending' ORDER BY created_at`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var approvals []types.ApprovalRequest
	for rows.Next() {
		var a types.ApprovalRequest
		if err := rows.Scan(&a.ID, &a.WorkflowRunID, &a.NodeID, &a.Status, &a.ReviewedBy, &a.ReviewNote, &a.CreatedAt, &a.ResolvedAt); err != nil {
			return nil, err
		}
		approvals = append(approvals, a)
	}
	return approvals, rows.Err()
}

func (s *Store) ResolveApproval(ctx context.Context, id uuid.UUID, status types.ApprovalStatus, reviewerID uuid.UUID, note *string) error {
	tag, err := s.pool.Exec(ctx,
		`UPDATE approval_requests
		 SET status=$1, reviewed_by=$2, review_note=$3, resolved_at=now()
		 WHERE id=$4 AND status='pending'`,
		status, reviewerID, note, id,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
