CREATE TABLE workflow_runs (
  id          uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
  workflow_id uuid        NOT NULL REFERENCES workflows(id),
  status      text        NOT NULL DEFAULT 'pending',
  context     jsonb       NOT NULL DEFAULT '{}',
  created_at  timestamptz NOT NULL DEFAULT now(),
  updated_at  timestamptz NOT NULL DEFAULT now(),
  started_at  timestamptz,
  ended_at    timestamptz
);

CREATE INDEX workflow_runs_workflow_id_idx ON workflow_runs(workflow_id);
CREATE INDEX workflow_runs_status_idx      ON workflow_runs(status);

CREATE TRIGGER workflow_runs_updated_at
  BEFORE UPDATE ON workflow_runs
  FOR EACH ROW EXECUTE FUNCTION set_updated_at();
