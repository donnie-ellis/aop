CREATE TABLE workflow_nodes (
  id          uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
  workflow_id uuid        NOT NULL REFERENCES workflows(id) ON DELETE CASCADE,
  kind        text        NOT NULL,
  resource_id uuid,
  label       text        NOT NULL DEFAULT '',
  extra_vars  jsonb       NOT NULL DEFAULT '{}',
  created_at  timestamptz NOT NULL DEFAULT now(),
  updated_at  timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX workflow_nodes_workflow_id_idx ON workflow_nodes(workflow_id);

CREATE TRIGGER workflow_nodes_updated_at
  BEFORE UPDATE ON workflow_nodes
  FOR EACH ROW EXECUTE FUNCTION set_updated_at();
