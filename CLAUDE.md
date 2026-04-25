# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What This Is

AOP (Ansible Orchestration Platform) is a distributed system for running Ansible playbooks at scale. It is a monorepo with four Go binaries and a React frontend. Status: **early development — V1 in progress**. Most component directories exist with stubs; the architecture and shared types are defined but most logic is not yet written.

---

## Commands

```bash
# Build
make build-api        # → bin/api
make build-controller # → bin/controller
make build-agent      # → bin/agent
make build-all

# Test
make test             # all packages via go workspace
make test-api / test-controller / test-agent

# Lint
make lint             # golangci-lint across all packages

# Migrations (uses $AOP_DB_URL)
make migrate-up
make migrate-down          # rolls back 1 step
make migrate-create        # prompts for name, creates up/down files in ./migrations/

# Dev environment
cp .devcontainer/.env.example .devcontainer/.env  # required before first container start
make dev                                           # starts full stack
```

Single test:
```bash
go test ./api/... -run TestFunctionName
go test github.com/donnie-ellis/aop/pkg/... -run TestFunctionName
```

Migration files follow: `./migrations/NNNNNN_name.up.sql` / `./migrations/NNNNNN_name.down.sql`

---

## Architecture

Four processes plus a frontend. They share nothing at runtime except a Postgres database and the agent protocol over HTTP.

```
React UI → API Server → Postgres ← Controller → Agents (push via HTTP)
                                  ↑
                              (same DB)
```

**API Server** (`/api`) — the only process that external clients talk to. Owns auth (JWT + API tokens + agent tokens), credential encryption (AES-GCM), WebSocket log streaming, and all CRUD. Creates job records but never dispatches them — that is the controller's job.

**Controller** (`/controller`) — runs three loops in a single process: (1) reconciliation loop that picks up `pending` jobs and dispatches them to agents, (2) scheduler that fires cron jobs, (3) git sync that clones repos and parses inventory. Never serves HTTP to external clients. The controller **decrypts credentials** before dispatch and includes plaintext material in the `JobPayload`.

**Agent** (`/agent`) — executes `ansible-playbook` as a subprocess and streams output back. No business logic. Registers with the API on startup, exposes a small HTTP listener for job dispatch (`Agent.Address`), receives `JobPayload` directly from the controller via HTTP POST, then reports results and logs back to the API via `JobPayload.CallbackURL`. Each job runs in an isolated temp directory that is deleted on completion regardless of outcome.

**pkg** (`/pkg/types`) — shared Go types and plugin interfaces. **No logic, no DB imports, no imports from component packages.** Only type defs, interface defs, constants, and value methods like `IsTerminal()`.

**UI** (`/ui`) — React 18 + TypeScript + Vite + Tailwind. Dark-mode developer-tool aesthetic. JWT stored in memory only (lost on page refresh — intentional). React Flow for workflow DAG builder.

---

## Go Module Layout

This repo uses a **Go workspace** (`go.work`) with four modules:

```
go.work
agent/go.mod     (module github.com/donnie-ellis/aop/agent)
api/go.mod       (module github.com/donnie-ellis/aop/api)
controller/go.mod(module github.com/donnie-ellis/aop/controller)
pkg/go.mod       (module github.com/donnie-ellis/aop/pkg)
```

Each binary entrypoint is at `{component}/cmd/{component}/`. The `pkg` module is a shared dependency — all other modules import it. Run `go test` commands from the repo root using the workspace-qualified paths (see Makefile).

---

## Plugin Interfaces

Interfaces are split across two packages. Don't bypass them.

**`/pkg/transport/interfaces.go`** — wire protocol between controller and agents:

| Interface | Owner | V1 Location | V2 Examples |
|-----------|-------|-------------|-------------|
| `AgentTransport` | controller | `/controller/transport/rest.go` | gRPC stream, long-poll (for NAT'd agents) |
| `AgentSelector` | controller | `/controller/selector/leastbusy.go` | label affinity, round-robin |

**`/pkg/types/interfaces.go`** — domain interfaces shared by multiple components:

| Interface | Owner | V1 Location | V2 Examples |
|-----------|-------|-------------|-------------|
| `InventorySource` | controller | `/controller/inventory/gitfile.go` | AWS EC2, Terraform state |
| `NodeExecutor` | controller | `/controller/workflow/` | sub-workflows, decision gates |

**`/pkg/types/credential.go`** — secrets backend:

| Interface | Owner | V1 Location | V2 Examples |
|-----------|-------|-------------|-------------|
| `SecretsProvider` | controller + api | `/api/secrets/postgres.go` | HashiCorp Vault, AWS SM |

**Dispatch model (v1 — push):** The controller calls `AgentTransport.Dispatch()` which makes a direct HTTP POST to `Agent.Address`. Agents must be reachable from the controller. The `AgentTransport` interface is the v2 seam: a pull/long-poll or gRPC-stream implementation slots in without changing the reconciliation loop.

---

## Job Lifecycle

```
pending → dispatched → running → success
                               → failed
                               → cancelled
         → failed  (no agents available)
dispatched → failed (dispatch timeout)
running    → failed (running timeout / agent died)
```

State transitions are validated. Invalid transitions are rejected, not silently ignored. `JobStatus.IsTerminal()` returns true for `success`, `failed`, `cancelled`.

---

## Credential Handling Rules

- The `Credential` struct **never carries raw secret values** — it's metadata only.
- Raw secrets live only in `CredentialSecret`, which is transient (never persisted in that form).
- The API returns credential metadata only — secrets are never returned in API responses.
- The **controller** decrypts credentials via `SecretsProvider` when building a `JobPayload`. The decrypted `[]CredentialSecret` travels in the payload to the agent over TLS.
- The **agent** receives plaintext material, writes it to `{AOP_WORK_DIR}/{job_id}/` temp files, chmod 600, and deletes them in a deferred cleanup regardless of job outcome.
- Credential temp paths are never logged, even at debug level.

---

## Key Design Decisions to Preserve

These fields are intentionally strings rather than enums — do not change them to typed enums:
- `WorkflowEdge.Condition` — currently `on_success | on_failure | always`, reserved for CEL expressions in v2
- `WorkflowNode.Type` — currently `job_template | approval`, extensible without schema changes
- `Inventory.SourceType` — currently `git_file`, new source types are new implementations
- `Credential.Type` — currently `ssh_key | vault_password | username_password`

The `WorkflowRun.Facts` map (context bag) is populated by jobs now and read by decision gates in v2 — don't remove or rename it even though nothing reads it in v1.

Log batches use a `seq` field for ordering and gap detection — the sequence number is per-job, assigned by the agent, and must be monotonically increasing.

---

## Environment Variables Quick Reference

| Component | Required vars |
|-----------|--------------|
| API | `AOP_DB_URL`, `AOP_ENCRYPTION_KEY` (32-byte hex), `AOP_JWT_SECRET` |
| Controller | `AOP_DB_URL` |
| Agent | `AOP_API_URL`, `AOP_REGISTRATION_TOKEN` |
| UI | `VITE_API_URL` |

Dev container config lives in `.devcontainer/.env` (gitignored). Copy from `.devcontainer/.env.example`.
