CREATE TABLE projects (
  id             uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
  name           text        NOT NULL,
  repo_url       text        NOT NULL,
  branch         text        NOT NULL DEFAULT 'main',
  inventory_path text        NOT NULL,
  credential_id  uuid        REFERENCES credentials(id) ON DELETE SET NULL,
  sync_status    text        NOT NULL DEFAULT 'pending',
  last_synced_at timestamptz,
  sync_error     text,
  created_at     timestamptz NOT NULL DEFAULT now(),
  updated_at     timestamptz NOT NULL DEFAULT now()
);

CREATE TRIGGER projects_updated_at
  BEFORE UPDATE ON projects
  FOR EACH ROW EXECUTE FUNCTION set_updated_at();
