CREATE TABLE job_templates (
  id                 uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
  name               text        NOT NULL,
  description        text        NOT NULL DEFAULT '',
  project_id         uuid        NOT NULL REFERENCES projects(id),
  playbook           text        NOT NULL,
  credential_id      uuid        REFERENCES credentials(id) ON DELETE SET NULL,
  default_extra_vars jsonb       NOT NULL DEFAULT '{}',
  created_at         timestamptz NOT NULL DEFAULT now(),
  updated_at         timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX job_templates_project_id_idx ON job_templates(project_id);

CREATE TRIGGER job_templates_updated_at
  BEFORE UPDATE ON job_templates
  FOR EACH ROW EXECUTE FUNCTION set_updated_at();
