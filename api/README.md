# API Server (`/api`)

## Purpose

The API server is the single entry point for all external communication with AOP.
Clients — the React UI, future CLI, and external webhooks — talk only to this component.
It owns the REST API, WebSocket log streaming, authentication, and all database reads
and writes that originate from external requests.

The controller and scheduler read from Postgres directly for efficiency, but they never
expose HTTP endpoints. If something needs to be accessible from outside the system,
it lives here.

---

## What This Component Does

- Serves the REST API for all AOP resources
- Handles JWT authentication and API token validation
- Encrypts and decrypts credentials at rest
- Accepts agent registration and heartbeat requests
- Creates job records when clients request execution
- Streams job logs to clients via WebSocket
- Triggers Git sync operations
- Exposes workflow run state and approval endpoints
- Serves the Prometheus metrics endpoint

## What This Component Does NOT Do

- Execute Ansible (that is the agent's job)
- Select which agent receives a job (that is the controller's job)
- Evaluate cron schedules (that is the scheduler's job)
- Clone Git repos (that is git sync's job)
- Make decisions about workflow progression (that is the workflow engine's job)

---

## Tech Stack

- **Language:** Go
- **Router:** chi (or stdlib net/http)
- **Database:** pgx v5 for Postgres
- **Auth:** golang-jwt/jwt
- **Logging:** zerolog (structured JSON in production)
- **Encryption:** AES-GCM, key from `AOP_ENCRYPTION_KEY` env var
- **WebSocket:** gorilla/websocket or nhooyr.io/websocket

---

## API Surface

### Auth
```
POST   /auth/login              # Returns JWT
POST   /auth/token              # Create API token for service accounts
DELETE /auth/token/:id          # Revoke API token
```

### Projects
```
GET    /projects
POST   /projects
GET    /projects/:id
PUT    /projects/:id
DELETE /projects/:id
POST   /projects/:id/sync       # Trigger git pull + inventory sync
```

### Credentials
```
GET    /credentials             # Returns metadata only, never raw secrets
POST   /credentials
GET    /credentials/:id         # Returns metadata only
DELETE /credentials/:id
```

### Inventories
```
GET    /inventories
POST   /inventories
GET    /inventories/:id
PUT    /inventories/:id
DELETE /inventories/:id
POST   /inventories/:id/sync    # Trigger sync from source
```

### Job Templates
```
GET    /job-templates
POST   /job-templates
GET    /job-templates/:id
PUT    /job-templates/:id
DELETE /job-templates/:id
GET    /job-templates/:id/jobs  # Job history for this template
```

### Jobs
```
POST   /jobs                    # Dispatch a job from a template
GET    /jobs                    # List with filters (status, template, agent, date)
GET    /jobs/:id
DELETE /jobs/:id                # Cancel a running job
GET    /jobs/:id/logs           # WebSocket — streams log lines
```

### Schedules
```
GET    /schedules
POST   /schedules
GET    /schedules/:id
PUT    /schedules/:id
DELETE /schedules/:id
```

### Agents
```
POST   /agents/register         # Called by agent on startup
GET    /agents                  # List with online/offline status
GET    /agents/:id
POST   /agents/:id/heartbeat    # Called by agent on interval
GET    /agents/:id/jobs/next    # Long-poll for job assignment (if REST protocol)
POST   /agents/:id/jobs/:job_id/logs    # Agent posts log lines
POST   /agents/:id/jobs/:job_id/result  # Agent posts final result
```

### Workflows
```
GET    /workflows
POST   /workflows
GET    /workflows/:id
PUT    /workflows/:id
DELETE /workflows/:id
POST   /workflows/:id/run       # Start a workflow run
GET    /workflows/runs          # List runs
GET    /workflows/runs/:id      # Full run state including node statuses
```

### Approvals
```
GET    /approvals               # Pending approvals for current user
GET    /approvals/:id
POST   /approvals/:id/approve
POST   /approvals/:id/deny
```

### System
```
GET    /health
GET    /metrics                 # Prometheus
```

---

## Authentication

**JWT** for interactive users. Short-lived (configurable, default 8h). Middleware
validates on all protected routes. Token contains user ID and basic claims — no roles
in v1.

**API Tokens** for service accounts (webhooks, future CLI). Long-lived, stored hashed
in DB. Passed as Bearer token in Authorization header. Same middleware handles both
JWT and API token validation.

**Agent tokens** are a separate concern — issued at registration, stored hashed, presented
on heartbeat and job receipt. Validated by the agent-specific middleware on agent endpoints.

---

## Credential Encryption

All sensitive credential fields encrypted with AES-GCM before writing to Postgres.
Encryption key loaded from `AOP_ENCRYPTION_KEY` environment variable at startup.
Server refuses to start if key is missing or wrong length.

GET responses for credentials return only metadata: id, name, type, created_at,
updated_at. The raw secret values are never returned. Credentials are only decrypted
internally when materializing a job for dispatch to an agent.

**CredentialType** defines how a credential is used:
- `ssh_key` — written to a temp file, passed via --private-key
- `vault_password` — written to a temp file, passed via --vault-password-file
- `username_password` — injected as extra vars or env vars depending on config
- Future types implement the same interface

---

## WebSocket Log Streaming

`GET /jobs/:id/logs` upgrades to WebSocket. The server streams log lines as they
arrive from the agent. Lines are newline-delimited text, including ANSI escape codes
from Ansible output.

If the job is already complete when a client connects, the server replays stored logs
from Postgres then closes the connection. If the job is still running, the server
streams live and closes when the job completes.

Clients should implement reconnect with backoff — WebSocket connections can drop.

---

## Configuration (Environment Variables)

```
AOP_DB_URL              # Postgres connection string (required)
AOP_ENCRYPTION_KEY      # 32-byte hex key for AES-GCM (required)
AOP_JWT_SECRET          # JWT signing secret (required)
AOP_JWT_EXPIRY          # JWT expiry duration, e.g. "8h" (default: 8h)
AOP_PORT                # HTTP port (default: 8080)
AOP_LOG_LEVEL           # debug|info|warn|error (default: info)
AOP_LOG_FORMAT          # json|pretty (default: json)
AOP_AGENT_HEARTBEAT_TTL # How long before an agent is marked offline (default: 30s)
```

Server validates all required vars on startup and exits with a clear error message
if any are missing.

---

## Key Interfaces Owned

The API server instantiates and owns the SecretsProvider. In v1 this is always the
Postgres-encrypted implementation. The interface is defined in `/pkg/types`.

```go
type SecretsProvider interface {
    Get(ctx context.Context, ref string) (SecretValue, error)
    Ping(ctx context.Context) error
}
```

New providers (HashiCorp Vault etc.) register themselves and are selected via config.
The API server code never changes when a new provider is added.

---

## V2 Seams to Preserve

- SecretsProvider is an interface — new vault backends plug in without touching API code
- Credential type handling is data-driven — new types add a DB record, not code changes
- Agent endpoints are versioned (`/v1/agents/...`) — protocol changes don't break existing agents immediately