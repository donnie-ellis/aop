# Agent (`/agent`)

## Purpose

The agent is the execution runtime for AOP. It is intentionally the simplest component
in the system. Its entire job is: register, stay registered, accept a job, run it,
report what happened, repeat.

The agent has no business logic. It does not know about workflows, schedules, other
agents, or what comes before or after the job it is running. It knows how to execute
an Ansible playbook and get the results back to the API server.

This simplicity is a feature. It makes the agent small, auditable, and easy to embed
in any runtime environment.

---

## What This Component Does

- Registers itself with the API server on startup
- Sends periodic heartbeats to stay marked as available
- Receives job assignments from the controller (via the agent protocol)
- Materializes credentials into the execution environment (temp files, env vars)
- Executes ansible-playbook as a subprocess
- Captures stdout and stderr line by line and streams them to the API
- Reports the final result (exit code, status, duration) to the API
- Cleans up temp files and credentials after every run, regardless of outcome
- Reconnects to the API server if the connection is lost

## What This Component Does NOT Do

- Make scheduling decisions
- Know about other agents
- Store credentials beyond the lifetime of a single job
- Know about workflows or which workflow a job belongs to
- Modify playbooks or inventory — it runs what it is given

---

## Tech Stack

- **Language:** Go
- **Logging:** zerolog
- **Execution:** os/exec (ansible-playbook subprocess)
- **Protocol:** REST over HTTP. Transport logic isolated in `/agent/transport/` behind
  an interface — see AgentTransport in ARCHITECTURE.md

---

## Agent Lifecycle

```
Start
  └─ Load config from environment
  └─ Call POST /agents/register
       └─ Receive agent ID + session token
  └─ Start heartbeat loop (background goroutine)
  └─ Start job receive loop
       └─ Wait for job assignment
       └─ Execute job
            └─ Materialize credentials
            └─ Write inventory to temp file
            └─ exec ansible-playbook
            └─ Stream stdout/stderr to API
            └─ Capture exit code
            └─ Clean up temp files
            └─ POST result to API
       └─ Return to waiting for next job
```

---

## Agent Registration

On startup the agent calls `POST /agents/register` with:
- `hostname` — the machine hostname
- `labels` — key/value pairs from config (e.g. `env=prod`, `region=us-east-1`)
- `capacity` — max concurrent jobs this agent will accept (from config, default: 1)
- `version` — agent binary version

The API returns:
- `agent_id` — UUID assigned to this agent instance
- `token` — session token used on all subsequent calls

The agent ID and token are held in memory. They are not persisted to disk — if the
agent restarts it re-registers and gets a new ID and token. The API handles this
gracefully (old agent record is marked offline via heartbeat TTL expiry).

---

## Heartbeat

A background goroutine calls `POST /agents/:id/heartbeat` every `AOP_HEARTBEAT_INTERVAL`
(default: 10s). The heartbeat payload includes:
- `running_jobs` — count of currently executing jobs
- `status` — `available` or `busy` (busy when at capacity)

If the heartbeat fails (API unreachable), the agent retries with exponential backoff.
It does not stop executing current jobs during a connectivity outage — it continues
running and tries to reconnect. If connectivity is restored, it resumes heartbeating
and can receive new jobs.

---

## Job Receipt

The agent polls `GET /agents/:id/jobs/next` with a 30s long-poll timeout. If a job
has been dispatched to this agent, the server responds immediately with the job payload.
If no job is ready, the server holds the connection open until one arrives or the
timeout expires, then the agent polls again. This gives near-instant job start without
a persistent connection.

The transport logic lives in `/agent/transport/rest.go` and implements the agent-side
of the `AgentTransport` contract. A future gRPC implementation would live in
`/agent/transport/grpc.go` and replace this without touching the execution code.
- Playbook path (relative to the cloned repo workspace)
- Inventory (as text, pre-fetched from Postgres by the API)
- Credential refs with encrypted values for transit
- Extra vars (JSON)
- Job ID (used for log and result endpoints)
- Workspace path (where the repo is already cloned by git sync)

---

## Credential Materialization

The agent receives credentials as encrypted values. It decrypts them and materializes
them into the execution environment for the duration of the job run.

Materialization by credential type:
- `ssh_key` — written to `{tmpdir}/ssh_key`, chmod 600, passed via `--private-key`
- `vault_password` — written to `{tmpdir}/vault_pass`, passed via `--vault-password-file`
- `username_password` — injected as extra vars or environment variables

All temp files are written to a job-scoped temp directory: `{AOP_WORK_DIR}/{job_id}/`.
This directory is created at job start and deleted at job end — always, in a deferred
cleanup function, regardless of whether the job succeeds or fails or panics.

Credentials are never logged. The temp directory path is never logged at debug level
in production builds.

---

## Ansible Execution

```go
cmd := exec.CommandContext(ctx, "ansible-playbook",
    "-i", inventoryFile,
    "--private-key", sshKeyFile,
    "--vault-password-file", vaultPassFile,
    "-e", extraVarsJSON,
    playbookPath,
)
```

stdout and stderr are captured line by line using a pipe and a scanner goroutine.
Each line is immediately sent to the API as it arrives — not buffered until completion.

Exit code is captured from `cmd.Wait()`. Mapping:
- Exit 0 → `success`
- Exit 1 → `failed` (Ansible task failure)
- Exit 2 → `failed` (Ansible internal error)
- Context cancelled → `cancelled`
- Any other non-zero → `failed`

---

## Log Streaming

Log lines are batched and POSTed to `POST /jobs/:id/logs`. The agent flushes the
buffer when either 10 lines have accumulated or 500ms has elapsed since the last
flush — whichever comes first. This keeps log output feeling real-time in the UI
while avoiding a separate HTTP request per line.

Each line in the batch is a JSON object:

```json
[
  { "line": "TASK [Install nginx] ***", "stream": "stdout", "seq": 42 },
  { "line": "ok: [web01]", "stream": "stdout", "seq": 43 }
]
```

`seq` is a monotonically increasing sequence number per job, assigned by the agent.
The API uses this to order lines correctly and detect any gaps.

If a log POST fails due to a transient network issue, the agent retries up to 3 times
with backoff before dropping the batch and continuing. Dropped line ranges are noted
in the final result payload.

---

## Result Reporting

On job completion the agent calls `POST /jobs/:id/result`:

```json
{
  "status": "success",
  "exit_code": 0,
  "duration_ms": 12453,
  "facts": { }
}
```

`facts` contains any variables emitted by Ansible's `set_stats` module, parsed from
the playbook output. These are forwarded to the workflow engine's context bag.

---

## Pluggable Runtime

The agent binary is the same regardless of where it runs. What varies is the runtime
environment:

**Systemd agent** — the binary runs as a systemd service on a bare VM or server.
Config from environment variables or a config file. Suitable for persistent workers.

**Container agent** — the binary runs in a Docker container. Config from environment
variables. Can be co-located with the API server for simple deployments.

**Kubernetes agent** — the binary runs as a Kubernetes Deployment. Can optionally
spawn per-job Pods for isolation (v2 feature). Reports labels from the Pod's own
labels via the downward API.

All three are the same binary. The runtime differences are in how the binary is
started and configured, not in the binary itself.

---

## Configuration (Environment Variables)

```
AOP_API_URL             # Base URL of the API server (required)
AOP_REGISTRATION_TOKEN  # Pre-shared token for initial registration (required)
AOP_CAPACITY            # Max concurrent jobs (default: 1)
AOP_HEARTBEAT_INTERVAL  # Heartbeat frequency (default: 10s)
AOP_WORK_DIR            # Temp directory for job workspaces (default: /tmp/aop-work)
AOP_WORKSPACE_DIR       # Where git repos are cloned (must match controller config)
AOP_LOG_LEVEL           # debug|info|warn|error (default: info)
AOP_LOG_FORMAT          # json|pretty (default: json)
AOP_LABELS              # Comma-separated key=value labels, e.g. env=prod,region=us-east-1
```

---

## V2 Seams to Preserve

- Transport logic lives in `/agent/transport/` separate from execution logic — a gRPC
  transport implementation replaces the REST one without touching job execution code
- Credential materialization is type-driven — new credential types add a case, not a rewrite
- The `facts` field in result reporting is already defined and forwarded — the workflow
  engine uses it in v1 for the context bag
- Capacity reporting on heartbeat supports future multi-job concurrency — v1 defaults
  to 1 but the field is already there
- Log batch format is an array — the gRPC equivalent streams the same message shape,
  just over a different transport