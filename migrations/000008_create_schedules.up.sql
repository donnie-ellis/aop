CREATE TABLE schedules (
  id          uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
  name        text        NOT NULL,
  template_id uuid        NOT NULL REFERENCES job_templates(id) ON DELETE CASCADE,
  cron_expr   text        NOT NULL,
  timezone    text        NOT NULL DEFAULT 'UTC',
  enabled     boolean     NOT NULL DEFAULT true,
  extra_vars  jsonb       NOT NULL DEFAULT '{}',
  last_run_at timestamptz,
  next_run_at timestamptz,
  created_at  timestamptz NOT NULL DEFAULT now(),
  updated_at  timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX schedules_template_id_idx ON schedules(template_id);
CREATE INDEX schedules_enabled_idx     ON schedules(enabled) WHERE enabled = true;

CREATE TRIGGER schedules_updated_at
  BEFORE UPDATE ON schedules
  FOR EACH ROW EXECUTE FUNCTION set_updated_at();
