CREATE TABLE inventory_hosts (
  id         uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
  project_id uuid        NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
  hostname   text        NOT NULL,
  groups     text[]      NOT NULL DEFAULT '{}',
  vars       jsonb       NOT NULL DEFAULT '{}',
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX inventory_hosts_project_id_idx ON inventory_hosts(project_id);

CREATE TRIGGER inventory_hosts_updated_at
  BEFORE UPDATE ON inventory_hosts
  FOR EACH ROW EXECUTE FUNCTION set_updated_at();
