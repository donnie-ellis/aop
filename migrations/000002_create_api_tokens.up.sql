CREATE TABLE api_tokens (
  id           uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id      uuid        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  name         text        NOT NULL,
  token_hash   text        NOT NULL UNIQUE,
  created_at   timestamptz NOT NULL DEFAULT now(),
  last_used_at timestamptz,
  expires_at   timestamptz
);

CREATE INDEX api_tokens_user_id_idx    ON api_tokens(user_id);
CREATE INDEX api_tokens_token_hash_idx ON api_tokens(token_hash);
