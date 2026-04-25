CREATE TABLE job_status_events (
  id          uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
  job_id      uuid        NOT NULL REFERENCES jobs(id) ON DELETE CASCADE,
  from_status text        NOT NULL,
  to_status   text        NOT NULL,
  reason      text        NOT NULL DEFAULT '',
  timestamp   timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX job_status_events_job_id_idx ON job_status_events(job_id);
