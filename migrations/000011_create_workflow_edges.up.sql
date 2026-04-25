CREATE TABLE workflow_edges (
  id             uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  workflow_id    uuid NOT NULL REFERENCES workflows(id) ON DELETE CASCADE,
  source_node_id uuid NOT NULL REFERENCES workflow_nodes(id) ON DELETE CASCADE,
  target_node_id uuid NOT NULL REFERENCES workflow_nodes(id) ON DELETE CASCADE,
  condition      text NOT NULL DEFAULT 'on_success'
);

CREATE INDEX workflow_edges_workflow_id_idx    ON workflow_edges(workflow_id);
CREATE INDEX workflow_edges_source_node_id_idx ON workflow_edges(source_node_id);
CREATE INDEX workflow_edges_target_node_id_idx ON workflow_edges(target_node_id);
