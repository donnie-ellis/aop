# Controller (`/controller`)

## Purpose

The controller is the brain of AOP's job execution system. It is a Go process running
a continuous reconciliation loop — watching for work that needs to be done and making
sure it gets done. It is inspired directly by the Kubernetes controller-manager pattern.

The controller never talks to external clients. It reads state from Postgres, makes
decisions, and updates state back to Postgres. Agents are the only external systems
it communicates with directly.

---

## Sub-components

The controller binary contains three logical sub-components. In v1 they run in the
same process. If they need to scale independently in the future, the boundaries are
already clean.

### Reconciliation Loop
Watches for jobs in `pending` status. Selects an available agent. Dispatches the job.
Updates job status through its lifecycle.

### Scheduler
Evaluates cron schedules attached to job templates. Creates job records for templates
that are due to run. The reconciliation loop then picks those up like any other job.

### Git Sync
Clones and pulls project Git repos on demand or on a schedule. Parses inventory files
from configured paths. Writes inventory data into Postgres. Reports sync status.

---

## What This Component Does

- Polls Postgres (or listens via NOTIFY) for jobs in `pending` status
- Queries available agents and selects one using the AgentSelector
- Dispatches jobs to selected agents via the agent protocol
- Updates job status through the full lifecycle state machine
- Detects stuck jobs and marks them failed after a configurable timeout
- Evaluates cron expressions and creates job records for due schedules
- Clones/pulls Git repos and syncs inventory into Postgres
- Drives workflow execution — walks the DAG, dispatches node jobs, follows edges

## What This Component Does NOT Do

- Serve HTTP to external clients (that is the API server's job)
- Execute Ansible (that is the agent's job)
- Make auth decisions (that is the API server's job)
- Store or encrypt credentials (that is the API server's job)

---

## Tech Stack

- **Language:** Go
- **Database:** pgx v5 for Postgres
- **Cron:** robfig/cron v3
- **Git:** go-git or exec to system git
- **Logging:** zerolog

---

## Job Lifecycle State Machine

Jobs move through a defined set of states. Transitions are validated — not every state
can transition to every other state. Invalid transitions are rejected and logged.

```
pending → dispatched → running → success
                               → failed
                               → cancelled
         → failed (agent selection failed, no agents available)
dispatched → failed (timeout — agent accepted but never reported running)
running → failed (timeout — agent died mid-run)
```

State transitions are recorded with timestamps. The API exposes the full history.

**Stuck job detection:** The controller periodically checks for jobs that have been
in `dispatched` or `running` state longer than a configurable timeout without an
update from the agent. These are marked `failed` with a system-generated log entry.

---

## Agent Selection

Agent selection is behind the `AgentSelector` interface:

```go
type AgentSelector interface {
    Select(ctx context.Context, available []Agent) (*Agent, error)
}
```

**Available** means: agent's last heartbeat is within `AOP_AGENT_HEARTBEAT_TTL`,
and the agent has not reported being at capacity.

**V1 implementation:** LeastBusySelector — picks the agent with the fewest currently
running jobs. If tied, picks the agent that heartbeated most recently.

Future selectors can implement label affinity (run this job on agents tagged
`env=prod`), weighted round-robin, or capability matching. The selector is chosen
via config. The reconciliation loop never changes.

---

## Workflow Execution Engine

The workflow engine lives in `/controller/workflow`. It is responsible for driving
workflow runs from start to completion.

**How it works:**
1. A workflow run is created (by the API when a client triggers execution)
2. The engine finds the starting node(s) — nodes with no inbound edges
3. For each ready node (all dependencies complete), it calls the appropriate NodeExecutor
4. On node completion, it evaluates outbound edges and enqueues the next ready nodes
5. A workflow run is complete when all terminal nodes have finished

**NodeExecutor interface:**
```go
type NodeExecutor interface {
    Execute(ctx context.Context, node WorkflowNode, wfCtx WorkflowContext) error
}
```

V1 implementations:
- `JobTemplateExecutor` — creates a job record and waits for completion
- `ApprovalExecutor` — creates an approval_request and blocks until approved or denied

**WorkflowContext (the facts bag):**
Every workflow run carries a `map[string]any` context object. When a job node
completes, any facts emitted via Ansible's `set_stats` module are captured and
merged into this bag. In v1, nothing reads from it. In v2, decision gate nodes
will evaluate CEL expressions against it. This costs nothing now — the data is
already there when v2 needs it.

**Edge conditions in v1:** `on_success`, `on_failure`, `always`. Stored as a string
field, not an enum, so expression strings slot in for v2 without schema changes.

---

## Scheduler

Uses robfig/cron v3. On startup, loads all active schedules from Postgres and
registers them. When a schedule fires, creates a job record in `pending` status
— the reconciliation loop picks it up like any manually triggered job.

Schedule changes (create, update, delete, enable, disable) are picked up without
restart — the controller watches for schedule changes in Postgres and updates the
cron registry accordingly.

Timezone support: schedules store a timezone string. All cron expressions evaluated
in the schedule's configured timezone.

Missed runs: in v1, if the controller is down when a schedule fires, the run is
missed. Catch-up behavior is a v2 feature.

---

## Git Sync

Clones project repos to a local workspace directory (`AOP_WORKSPACE_DIR`). On sync:
1. If repo not present locally, clone it
2. If present, fetch and reset to configured branch HEAD
3. Parse inventory file(s) from configured paths within the repo
4. Write/update inventory data in Postgres
5. Update project's `last_synced_at` and `sync_status`

Sync is triggered:
- On demand via API (POST /projects/:id/sync)
- On a configurable schedule per project (v1: global interval from config)

Git credentials (SSH key or HTTPS token) are stored as a Credential record and
resolved via SecretsProvider before cloning.

---

## Configuration (Environment Variables)

```
AOP_DB_URL                  # Postgres connection string (required)
AOP_LOG_LEVEL               # debug|info|warn|error (default: info)
AOP_LOG_FORMAT              # json|pretty (default: json)
AOP_RECONCILE_INTERVAL      # How often to poll for pending jobs (default: 5s)
AOP_AGENT_HEARTBEAT_TTL     # Agent considered offline after this (default: 30s)
AOP_JOB_DISPATCH_TIMEOUT    # Stuck dispatched job timeout (default: 60s)
AOP_JOB_RUNNING_TIMEOUT     # Stuck running job timeout (default: 3600s)
AOP_WORKSPACE_DIR           # Local directory for git clones (default: /tmp/aop-workspace)
```

---

## V2 Seams to Preserve

- `AgentSelector` is an interface — label affinity and other strategies slot in without
  touching the reconciliation loop
- `NodeExecutor` is an interface — sub-workflows and decision gates are new implementations,
  not modifications to the engine
- Edge `condition` is a string field — expression evaluation plugs in without schema changes
- The facts/context bag travels with every workflow run — decision gates in v2 read from
  what is already being populated
- Scheduler change detection is live — no restart required for schedule updates, which
  will remain true as schedule management gets more sophisticated