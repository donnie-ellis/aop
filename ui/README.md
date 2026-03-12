# UI (`/ui`)

## Purpose

The React frontend for AOP. A developer tool interface — functional, dark-mode-first,
and focused on the things operators actually need: seeing what's running, reading logs,
building workflows, and managing resources.

The UI talks exclusively to the AOP API server. It has no direct database access and
no direct agent communication. Everything goes through the REST API and WebSocket
endpoints.

---

## What This Component Does

- Provides the primary interface for all AOP operations
- Authenticates users and manages JWT session state
- Displays live job status and streams log output via WebSocket
- Provides a visual DAG editor for building and viewing workflows
- Surfaces pending approval requests with context for approvers
- Manages all AOP resources: projects, credentials, inventories, templates, schedules

## What This Component Does NOT Do

- Communicate with agents directly
- Access Postgres directly
- Store sensitive data in the browser (credentials, raw secrets)
- Persist application state between sessions beyond the JWT token

---

## Tech Stack

- **Framework:** React 18 + TypeScript
- **Build:** Vite
- **Routing:** React Router v6
- **Styling:** Tailwind CSS (utility-first, dark mode via class strategy)
- **HTTP client:** fetch with a typed wrapper (or axios)
- **WebSocket:** native browser WebSocket API
- **DAG visualization:** React Flow (reactflow.dev)
- **State management:** React context + hooks (no Redux in v1 — not needed at this scale)
- **Forms:** React Hook Form + Zod for validation

---

## Design Direction

Developer tool aesthetic. Dark-mode first. Clean and functional over decorative.
Think VS Code or Grafana, not a marketing website.

- Dark background (`#0d1117` or similar)
- Monospace font for log output, job IDs, status codes
- Color used for status (green=success, red=failed, yellow=running, gray=pending)
- Minimal chrome — the content is the interface
- Responsive enough to be usable on a laptop, not optimized for mobile

---

## Authentication

JWT stored **in memory only** — never in localStorage or sessionStorage. This means
the token is lost on page refresh, requiring re-login. This is the correct security
tradeoff for a developer tool where sessions are typically short-lived.

Auth context provides the token to all components. An Axios/fetch interceptor attaches
`Authorization: Bearer {token}` to every API request.

API token management (for service accounts) is a UI screen but the tokens themselves
are never displayed after creation — show once, then metadata only.

---

## Screens

### Login
Simple email/password form. On success, stores JWT in auth context, redirects to
dashboard.

### Dashboard
- Currently running jobs with live status badges
- Recently completed jobs (last 10)
- Agent status summary (online count, any offline agents)
- Quick-launch buttons for frequently used job templates (v2 — pinning)

### Projects
- List: name, git URL, branch, last synced, sync status
- Create/Edit: git URL, branch, credential for git auth
- Detail: sync history, list of inventories sourced from this project

### Credentials
- List: name, type, created/updated dates — no secret values ever shown
- Create: name, type selector, type-specific fields (SSH key upload, password input)
- No edit — delete and recreate (avoids partial secret exposure)

### Inventories
- List: name, source type, last synced, sync status (success/failed/syncing)
- Create: name, source type (v1: git_file), source config fields
- Detail: last sync output, inventory content preview (host list)
- Manual sync button

### Job Templates
- List: name, project, playbook, inventory
- Create/Edit: all fields, extra_vars editor (JSON or key-value UI)
- Detail: run history for this template, Run Now button
- Run Now: opens modal to confirm, optionally override extra_vars, then dispatches

### Jobs
- List: filterable by status, template, agent, date range
- Live status badges that update without full page refresh
- Detail / Log Viewer (see below)

### Log Viewer
The most important screen in the application. Users spend more time here than anywhere
else.

- Full-page log stream
- WebSocket connection to `GET /jobs/:id/logs`
- ANSI color code rendering (Ansible output is heavily colorized)
- Auto-scroll to bottom with a "scroll to bottom" button when manually scrolled up
- Job status header: template name, agent, start time, duration (live if running)
- Reconnect automatically if WebSocket drops
- If job is complete, loads stored logs via REST then closes WS connection
- Copy-to-clipboard button for full log output

### Workflows
- List: name, last run status, last run time
- Create/Edit: opens the DAG builder
- Run history: list of runs with status

### Workflow DAG Builder
Built with React Flow.

- Drag node types from a sidebar palette onto the canvas
- Node types: Job Template (select from dropdown), Approval
- Connect nodes by dragging between handles
- Edge condition selector on each edge: on_success / on_failure / always
- Nodes show the template name and a status badge when viewing a run
- Save workflow definition back to API

### Workflow Run Viewer
- The DAG rendered with live node status overlaid
- Nodes color-coded by status: pending/running/success/failed/waiting-approval
- Click a completed job node to open its log viewer
- Pending approval nodes show an Approve/Deny panel with context:
  - Which playbook, which inventory, which extra vars
  - What has already run in this workflow and its results
  - Approve and Deny buttons, optional justification text field

### Agents
- List: hostname, labels, status (online/offline), current job count, last heartbeat
- Online/offline determined by last heartbeat age
- No create/delete — agents self-register

### Schedules
- List: template name, cron expression, next run time, enabled/disabled toggle
- Create/Edit: template selector, cron expression input with human-readable preview
- Enable/disable without deleting

---

## API Client Layer

A typed wrapper around fetch lives in `/ui/src/api/`. It handles:
- Base URL from `VITE_API_URL` environment variable
- Attaching the Authorization header from auth context
- Parsing response JSON into typed interfaces that mirror the API's response shapes
- Translating HTTP error responses into typed error objects
- Redirecting to login on 401 responses

All API types are defined in `/ui/src/types/` and mirror the Go types in `/pkg/types`.
In v1 these are maintained in sync manually. A future improvement is generating them
from the OpenAPI spec.

---

## WebSocket Handling

The log viewer manages its own WebSocket lifecycle:
1. On mount, open WS connection to `/jobs/:id/logs`
2. Append each received line to a buffer, render to screen
3. On close (server closes when job completes), display final status
4. On unexpected close, wait 2s then reconnect (up to 3 attempts)
5. On unmount, close the connection cleanly

ANSI codes are rendered using a lightweight parser — the `ansi-to-react` library or
equivalent. Do not strip ANSI codes — Ansible's colorized output is meaningful.

---

## Configuration (Environment Variables)

```
VITE_API_URL        # AOP API base URL, e.g. http://localhost:8080 (required)
```

---

## V2 Seams to Preserve

- The DAG builder stores node `type` and `spec` as-is from the API — new node types
  (decision gates) appear in the palette and render without UI changes to the engine
- Edge condition is a string field in the UI data model — expression input replaces
  the dropdown without a data model change
- The workflow run viewer renders node status by type — new node types add a renderer,
  not a rewrite
- The approval panel is componentized — richer context display in v2 is additive