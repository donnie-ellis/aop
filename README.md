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

- Go 1.22+
- Node.js 20+
- Docker and Docker Compose
- Ansible (on any host running the agent)

### Run Locally

```bash
# Clone the repo
git clone https://github.com/yourusername/aop.git
cd aop

# Start Postgres and all services
docker compose up -d

# Apply database migrations
make migrate

# Start the full stack in development mode
make dev
```

The UI will be available at `http://localhost:5173`.
The API server will be available at `http://localhost:8080`.

### Build

```bash
make build-api        # Build the API server binary
make build-controller # Build the controller binary
make build-agent      # Build the agent binary
make build-all        # Build all three
make ui-build         # Build the React frontend
```

### Test

```bash
make test             # Run all Go tests
make test-api         # Run API server tests only
make test-controller  # Run controller tests only
make test-agent       # Run agent tests only
```

### Lint

```bash
make lint             # Run golangci-lint across all packages
```

---

## Configuration

All components are configured via environment variables. There are no config files
required — all settings have documented environment variables with sensible defaults
where applicable. Each component's README documents its full set of environment variables.

A `.env.example` file is provided at the root with all variables and descriptions.
Copy it to `.env` for local development — docker compose will pick it up automatically.

```bash
cp .env.example .env
# Edit .env with your values
```

---

## Tech Stack

| Layer | Technology |
|-------|-----------|
| Backend | Go 1.22+ |
| Frontend | React 18, TypeScript, Tailwind CSS, Vite |
| Database | PostgreSQL 16 |
| Migrations | golang-migrate |
| Logging | zerolog (structured JSON) |
| Metrics | Prometheus (`/metrics` endpoint) |
| Workflow visualization | React Flow |
| Containerization | Docker |

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
- [ ] Core data model and migrations
- [ ] API server — all resource endpoints
- [ ] Controller — reconciliation loop and job lifecycle
- [ ] Scheduler — cron-based job triggering
- [ ] Git sync — project and inventory sync
- [ ] Agent — Ansible execution and log streaming
- [ ] Workflow engine — DAG execution and approval gates
- [ ] React UI — all core screens
- [ ] Docker images and docker-compose

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