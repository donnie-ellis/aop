CREATE TABLE approval_requests (
  id              uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
  workflow_run_id uuid        NOT NULL REFERENCES workflow_runs(id) ON DELETE CASCADE,
  node_id         uuid        NOT NULL REFERENCES workflow_nodes(id),
  status          text        NOT NULL DEFAULT 'pending',
  reviewed_by     uuid        REFERENCES users(id) ON DELETE SET NULL,
  review_note     text,
  created_at      timestamptz NOT NULL DEFAULT now(),
  resolved_at     timestamptz
);

CREATE INDEX approval_requests_workflow_run_id_idx ON approval_requests(workflow_run_id);
CREATE INDEX approval_requests_status_idx          ON approval_requests(status) WHERE status = 'pending';
