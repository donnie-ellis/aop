# AOP — Ansible Orchestration Platform

AOP is a distributed Ansible orchestration platform built to replace AWX OSS. It provides
on-demand and scheduled Ansible playbook execution, workflow automation with approval gates,
and a clean web interface — designed around simplicity, observability, and genuine
extensibility through well-defined interfaces.

> **Status: Early Development — V1 in progress**

---

## Why AOP

AWX OSS has stagnated. AOP takes the core concept — a platform for running Ansible at
scale — and rebuilds it with a Kubernetes-inspired architecture: loosely coupled components,
a central API server, a controller that reconciles desired state, and agents that do the
actual work. Secrets providers, inventory sources, and agent runtimes are all pluggable
interfaces. New backends slot in without touching core logic.

---

## Features (V1)

- **On-demand execution** — run any job template immediately from the UI or API
- **Scheduled execution** — attach cron schedules to job templates
- **Workflows** — build DAGs of job templates with on_success/on_failure/always edges
- **Approval gates** — pause a workflow and require human sign-off before continuing
- **Git-sourced playbooks and inventory** — projects point to a Git repo; AOP handles the rest
- **Encrypted credential storage** — SSH keys, vault passwords, and passwords stored
  encrypted at rest, never exposed in API responses
- **Live log streaming** — watch Ansible output in real time in the browser
- **Pluggable agents** — agents run anywhere: bare VMs, containers, Kubernetes pods.
  Anything that implements the agent protocol is a valid agent.

---

## Architecture

AOP is a monorepo containing five components that work together as a distributed system.

```
┌─────────────────────────────────────────────────┐
│                    Clients                       │
│         React UI   │   CLI   │   Webhooks        │
└───────────────────────┬─────────────────────────┘
                        │ HTTPS / REST + WebSocket
┌───────────────────────┴─────────────────────────┐
│                   API Server                     │
│      REST API · WebSocket Logs · Auth · DB       │
└───────────────────────┬─────────────────────────┘
                        │ PostgreSQL
┌───────────────────────┴─────────────────────────┐
│                 Control Plane                    │
│       Controller · Scheduler · Git Sync         │
└───────────────────────┬─────────────────────────┘
                        │ Agent Protocol
┌───────────────────────┴─────────────────────────┐
│              Agents (Pluggable Runtime)          │
│     Systemd Agent · Container · K8s Agent       │
└─────────────────────────────────────────────────┘
```

For full architecture documentation including design decisions, plugin interface
definitions, and v1 scope — see [ARCHITECTURE.md](./ARCHITECTURE.md).

---

## Repository Structure

```
/
├── api/              # REST API server, auth, WebSocket log streaming
├── controller/       # Reconciliation loop, scheduler, git sync, workflow engine
├── agent/            # Ansible execution agent
├── ui/               # React + TypeScript frontend
├── pkg/              # Shared Go types and plugin interfaces
│   └── types/
├── migrations/       # PostgreSQL migration files (golang-migrate)
├── proto/            # Protobuf definitions (reserved for future gRPC migration)
├── docker-compose.yml
├── Makefile
└── ARCHITECTURE.md   # Full system design and decision log
```

### Component Documentation

| Component | README | Description |
|-----------|--------|-------------|
| API Server | [api/README.md](./api/README.md) | REST API, auth, credential encryption, log streaming |
| Controller | [controller/README.md](./controller/README.md) | Job scheduling, workflow engine, cron, git sync |
| Agent | [agent/README.md](./agent/README.md) | Ansible execution, log streaming, credential materialization |
| UI | [ui/README.md](./ui/README.md) | React frontend, screens, WebSocket, DAG builder |
| Shared Types | [pkg/README.md](./pkg/README.md) | Go types, plugin interfaces, shared constants |

---

## Getting Started

### Prerequisites

- [VS Code](https://code.visualstudio.com/) with the [Dev Containers extension](https://marketplace.visualstudio.com/items?itemName=ms-vscode-remote.remote-containers)
- Docker Desktop (or Docker Engine + Docker Compose)
- Ansible on any host that will run the agent

The devcontainer provides Go, Node 22, pnpm, golangci-lint, golang-migrate, and a
PostgreSQL 17 sidecar. Everything else is installed on first open.

### 1. Start the devcontainer

```bash
git clone https://github.com/yourusername/aop.git
cd aop

# Create the required env file (edit values if needed)
cp .devcontainer/.env.example .devcontainer/.env
```

Open the folder in VS Code and choose **Reopen in Container** when prompted (or run
`Dev Containers: Reopen in Container` from the command palette). PostgreSQL starts
automatically as a sidecar service.

### 2. Apply migrations

```bash
make migrate-up
```

`AOP_DB_URL` defaults to the devcontainer Postgres instance
(`postgresql://appuser:apppassword@localhost:5432/appdb?sslmode=disable`) via the
`.env` file loaded by the devcontainer.

### 3. Build the binaries

```bash
make build-all          # → bin/api, bin/controller, bin/agent
```

### 4. Set environment variables

Each process reads configuration from environment variables. For local development,
export these in your shell (or add them to `.devcontainer/.env`):

```bash
# Shared
export AOP_DB_URL="postgresql://appuser:apppassword@localhost:5432/appdb?sslmode=disable"

# API server
export AOP_ENCRYPTION_KEY="$(openssl rand -hex 32)"   # 32-byte hex — keep stable across restarts
export AOP_JWT_SECRET="$(openssl rand -hex 32)"

# Agent
export AOP_API_URL="http://localhost:8080"
export AOP_REGISTRATION_TOKEN="change-me"             # must match the token seeded in the DB
```

> Generate `AOP_ENCRYPTION_KEY` once and store it — changing it invalidates all
> encrypted credentials in the database.

### 5. Start the stack

Open three terminal tabs (or use a process manager):

```bash
# Tab 1 — API server
./bin/api

# Tab 2 — Controller
./bin/controller

# Tab 3 — Agent (requires ansible-playbook on PATH)
./bin/agent
```

### 6. Start the UI dev server

```bash
cd ui
pnpm install        # first time only
pnpm dev
```

The UI is available at `http://localhost:5173` and proxies API calls to `http://localhost:8080`.

---

### Smoke test

The smoke test script builds the binaries, seeds a test user, wires up a local git
repo with a no-op playbook, and asserts a job reaches `success` end-to-end.

```bash
# Postgres must be reachable at $AOP_DB_URL (devcontainer default works)
bash scripts/smoke_test.sh
```

---

## Commands Reference

### Build

```bash
make build-api        # → bin/api
make build-controller # → bin/controller
make build-agent      # → bin/agent
make build-all        # all three
```

### Test

```bash
make test             # all packages via go workspace
make test-api         # API server only
make test-controller  # controller only
make test-agent       # agent only
```

### Lint

```bash
make lint             # golangci-lint across all packages
```

### Migrations

```bash
make migrate-up       # apply all pending migrations
make migrate-down     # roll back one step
make migrate-create   # create a new migration (prompts for name)
```

### UI

```bash
cd ui
pnpm dev              # dev server with HMR at http://localhost:5173
pnpm build            # production build → ui/dist/
pnpm exec biome check # lint + format check
```

---

## Configuration

All components are configured via environment variables. No config files are required.

| Component  | Required variables |
|------------|--------------------|
| API        | `AOP_DB_URL`, `AOP_ENCRYPTION_KEY` (32-byte hex), `AOP_JWT_SECRET` |
| Controller | `AOP_DB_URL` |
| Agent      | `AOP_API_URL`, `AOP_REGISTRATION_TOKEN` |
| UI         | `VITE_API_URL` (defaults to `http://localhost:8080`) |

Dev environment values live in `.devcontainer/.env` (gitignored). Copy from
`.devcontainer/.env.example` — the devcontainer loads it automatically.

---

## Tech Stack

| Layer | Technology |
|-------|-----------|
| Backend | Go 1.22+ |
| Frontend | React 18, TypeScript, Vite, Tailwind CSS v4 |
| UI components | shadcn/ui (new-york) + Radix UI |
| Forms | TanStack Form v1 + Zod |
| Data fetching | TanStack Query v5 |
| Routing | React Router v7 |
| Package manager | pnpm |
| Linter/formatter | Biome |
| Database | PostgreSQL 17 |
| Migrations | golang-migrate |
| Logging | zerolog (structured JSON) |
| Workflow visualization | React Flow (DAG builder) |
| Containerization | Docker / VS Code Dev Containers |

---

## Plugin System

AOP is built around pluggable interfaces. V1 ships with sensible defaults; new
backends implement the interface and register themselves — no changes to core logic required.

| Interface | V1 Implementation | Future Examples |
|-----------|------------------|-----------------|
| `AgentTransport` | REST long-poll + batched POST | gRPC bidirectional stream |
| `SecretsProvider` | Postgres (AES-GCM encrypted) | HashiCorp Vault, AWS Secrets Manager |
| `InventorySource` | Git file | AWS EC2, Terraform state |
| `AgentSelector` | Least-busy | Label affinity, weighted round-robin |
| `NodeExecutor` | Job template, Approval | Decision gates, sub-workflows |

Interface definitions live in [pkg/README.md](./pkg/README.md).

---

## Roadmap

### V1 (current)
- [x] Architecture and design
- [x] Core data model and migrations
- [x] API server — all resource endpoints
- [x] Controller — reconciliation loop and job lifecycle
- [x] Agent — Ansible execution and log streaming
- [x] React UI — all core screens (login, dashboard, jobs, templates, projects, agents)
- [x] End-to-end smoke test
- [ ] Scheduler — cron-based job triggering
- [ ] Git sync — project and inventory sync
- [ ] Workflow engine — DAG execution and approval gates
- [ ] Docker images for deployment

### V2 (planned)
- Decision gates in workflows (expression-based branching via CEL)
- Sub-workflows (workflow nodes that are themselves workflows)
- External vault providers (HashiCorp Vault, AWS Secrets Manager)
- Dynamic inventory sources (AWS EC2, Terraform state)
- Label-based agent affinity and scheduling
- Notification integrations (Slack, email, webhooks)
- CLI tooling
- RBAC

---

## Contributing

This project is in early development. The architecture and interfaces are still being
established. See [ARCHITECTURE.md](./ARCHITECTURE.md) for the full design rationale
before contributing.

1. Fork the repo
2. Create a feature branch (`git checkout -b feature/your-feature`)
3. Commit your changes with clear messages
4. Open a pull request against `main`

Please read the relevant component README before working on a component — each one
documents what the component does and does not do, and the seams that must be
preserved for v2 compatibility.

---

## License

[MIT](./LICENSE)