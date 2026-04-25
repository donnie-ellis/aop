CREATE TABLE job_logs (
  id         uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
  job_id     uuid        NOT NULL REFERENCES jobs(id) ON DELETE CASCADE,
  seq        integer     NOT NULL,
  line       text        NOT NULL,
  stream     text        NOT NULL DEFAULT 'stdout',
  created_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE (job_id, seq)
);

-- Composite index covers both the streaming query (job_id ORDER BY seq)
-- and the UNIQUE constraint enforcement.
CREATE INDEX job_logs_job_id_seq_idx ON job_logs(job_id, seq);
