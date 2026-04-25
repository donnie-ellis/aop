CREATE TABLE jobs (
  id              uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
  template_id     uuid        NOT NULL REFERENCES job_templates(id),
  agent_id        uuid        REFERENCES agents(id) ON DELETE SET NULL,
  workflow_run_id uuid        REFERENCES workflow_runs(id) ON DELETE SET NULL,
  status          text        NOT NULL DEFAULT 'pending',
  extra_vars      jsonb       NOT NULL DEFAULT '{}',
  facts           jsonb       NOT NULL DEFAULT '{}',
  failure_reason  text,
  created_at      timestamptz NOT NULL DEFAULT now(),
  updated_at      timestamptz NOT NULL DEFAULT now(),
  started_at      timestamptz,
  ended_at        timestamptz
);

CREATE INDEX jobs_status_idx          ON jobs(status);
CREATE INDEX jobs_template_id_idx     ON jobs(template_id);
CREATE INDEX jobs_agent_id_idx        ON jobs(agent_id);
CREATE INDEX jobs_workflow_run_id_idx ON jobs(workflow_run_id);

CREATE TRIGGER jobs_updated_at
  BEFORE UPDATE ON jobs
  FOR EACH ROW EXECUTE FUNCTION set_updated_at();
