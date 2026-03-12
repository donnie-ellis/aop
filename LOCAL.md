# Getting Started

This guide covers setting up a local development environment using the included dev
container. The dev container provides a fully configured Go environment with Node.js,
Ansible, golang-migrate, and golangci-lint pre-installed, alongside a PostgreSQL 17
instance — no local tooling required beyond VS Code and Docker.

---

## Prerequisites

| Tool | Purpose | Min Version |
|------|---------|-------------|
| [VS Code](https://code.visualstudio.com/) | Editor | 1.80+ |
| [Docker Desktop](https://www.docker.com/products/docker-desktop/) | Runs the container and Postgres | 4.0+ |
| [Dev Containers extension](https://marketplace.visualstudio.com/items?itemName=ms-vscode-remote.remote-containers) | `ms-vscode-remote.remote-containers` | 0.300+ |
| Git | Source control | 2.x |

> **Not using the dev container?** See the [manual setup](#manual-setup-without-dev-container)
> section at the bottom of this guide.

---

## 1. Clone the Repository

```bash
git clone https://github.com/yourusername/aop.git
cd aop
```

---

## 2. Create the .env File

The dev container requires a `.devcontainer/.env` file before it can start. This file
is excluded from version control — you must create it manually after cloning.

```bash
cp .devcontainer/.env.example .devcontainer/.env
```

Then open `.devcontainer/.env` and set your values:

```dotenv
# ── PostgreSQL container init ──────────────────────────────────────────────────
# Read by the postgres:17 image on first boot to create the user and database.
# Changing these after the volume exists has no effect — see "Resetting the
# Database" below if you need to start fresh.
POSTGRES_USER=appuser
POSTGRES_PASSWORD=apppassword
POSTGRES_DB=appdb

# ── Application database connection ───────────────────────────────────────────
# Used by the Go services and golang-migrate. Must match the values above.
DATABASE_URL=postgresql://appuser:apppassword@localhost:5432/appdb?sslmode=disable
```

> **Never commit `.devcontainer/.env` to version control.** It is listed in `.gitignore`.
> Use strong, unique passwords in any shared or production-adjacent environment.

### Variable Reference

| Variable | Example | Description |
|----------|---------|-------------|
| `POSTGRES_USER` | `appuser` | Superuser created by Postgres on first boot |
| `POSTGRES_PASSWORD` | `apppassword` | Password for the Postgres superuser |
| `POSTGRES_DB` | `appdb` | Default database created on first boot |
| `DATABASE_URL` | `postgresql://…` | Full DSN used by Go services and the migrate CLI |

---

## 3. Open in the Dev Container

With Docker Desktop running:

1. Open VS Code in the repository root.
2. Open the Command Palette (`Ctrl+Shift+P` / `Cmd+Shift+P`) and run
   **Dev Containers: Reopen in Container**.
3. VS Code will build the image (first run takes a few minutes) and attach. The
   Postgres container starts automatically alongside the app container.

> **Tip:** On subsequent opens VS Code skips the build and attaches to the existing
> container. After changes to the `Dockerfile` or `docker-compose.yml`, run
> **Dev Containers: Rebuild Container** to pick them up.

---

## 4. Apply Database Migrations

From the integrated terminal inside the container:

```bash
make migrate
```

This runs `golang-migrate` against `$DATABASE_URL`. Migration files live in
`migrations/` and follow the naming convention:

```
000001_create_users.up.sql
000001_create_users.down.sql
```

Check the current migration version at any time:

```bash
migrate -path ./migrations -database "$DATABASE_URL" version
```

---

## 5. Start the Development Stack

```bash
make dev
```

| Service | URL |
|---------|-----|
| React UI | http://localhost:5173 |
| API server | http://localhost:8080 |

---

## 6. Verify the Setup

```bash
# Go toolchain
go version

# Confirm Postgres connectivity
psql "$DATABASE_URL" -c '\l'

# Run the test suite
make test

# Run the linter
make lint
```

---

## Resetting the Database

The Postgres image only reads the `POSTGRES_*` variables on first boot. If you need
to change credentials or start with a clean schema, remove the volume:

```bash
# From outside the container
docker compose -f .devcontainer/docker-compose.yml down -v
```

The `-v` flag removes the `postgres-data` volume. On the next container start Postgres
re-initializes using the current `.env` values. Re-run `make migrate` afterward.

---

## Forwarding the Postgres Port

Port 5432 is not forwarded by default. To expose it locally (useful for GUI clients
like TablePlus or DBeaver), add the following to `.devcontainer/devcontainer.json` and
rebuild the container:

```jsonc
"forwardPorts": [5432]
```

---

## Troubleshooting

**Container fails to start — `.env not found`**
The `.devcontainer/.env` file is missing. Follow [step 2](#2-create-the-env-file).

**`FATAL: password authentication failed` connecting to Postgres**
The credentials in `DATABASE_URL` don't match what Postgres was initialized with.
Either update `DATABASE_URL` to match, or [reset the volume](#resetting-the-database)
and restart.

**Migrations fail after the container starts**
- Confirm Postgres is running: `docker ps | grep postgres`
- Confirm the variable is set: `echo $DATABASE_URL`
- Verify the migrations directory exists at `./migrations/`

**Changes to the Dockerfile aren't reflected**
Open the Command Palette and run **Dev Containers: Rebuild Container**.

---

## Manual Setup (Without Dev Container)

If you prefer to run services directly on your machine, ensure the following are
installed and on your `PATH`:

- Go 1.22+
- Node.js 20+
- PostgreSQL 16+ (or point `DATABASE_URL` at a remote instance)
- [golang-migrate CLI](https://github.com/golang-migrate/migrate)
- [golangci-lint v2](https://golangci-lint.run/welcome/install/)
- Ansible (required on any host running the agent)

Then follow the [Run Locally](./README.md#run-locally) steps in the README.