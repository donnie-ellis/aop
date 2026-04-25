CREATE TABLE agents (
  id                uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
  name              text        NOT NULL,
  address           text        NOT NULL,
  token_hash        text        NOT NULL UNIQUE,
  status            text        NOT NULL DEFAULT 'offline',
  labels            jsonb       NOT NULL DEFAULT '{}',
  capacity          int         NOT NULL DEFAULT 1,
  last_heartbeat_at timestamptz,
  registered_at     timestamptz NOT NULL DEFAULT now(),
  updated_at        timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX agents_status_idx ON agents(status);

CREATE TRIGGER agents_updated_at
  BEFORE UPDATE ON agents
  FOR EACH ROW EXECUTE FUNCTION set_updated_at();
