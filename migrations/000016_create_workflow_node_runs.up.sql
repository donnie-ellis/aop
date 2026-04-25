CREATE TABLE workflow_node_runs (
  id              uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
  workflow_run_id uuid        NOT NULL REFERENCES workflow_runs(id) ON DELETE CASCADE,
  node_id         uuid        NOT NULL REFERENCES workflow_nodes(id),
  status          text        NOT NULL DEFAULT 'waiting',
  job_id          uuid        REFERENCES jobs(id) ON DELETE SET NULL,
  created_at      timestamptz NOT NULL DEFAULT now(),
  updated_at      timestamptz NOT NULL DEFAULT now(),
  started_at      timestamptz,
  ended_at        timestamptz
);

CREATE INDEX workflow_node_runs_workflow_run_id_idx ON workflow_node_runs(workflow_run_id);
CREATE INDEX workflow_node_runs_node_id_idx         ON workflow_node_runs(node_id);
CREATE INDEX workflow_node_runs_job_id_idx          ON workflow_node_runs(job_id);

CREATE TRIGGER workflow_node_runs_updated_at
  BEFORE UPDATE ON workflow_node_runs
  FOR EACH ROW EXECUTE FUNCTION set_updated_at();
