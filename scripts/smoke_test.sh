#!/usr/bin/env bash
# smoke_test.sh — end-to-end dispatch sanity check
# Starts API + controller + agent, runs a no-op Ansible playbook, asserts success.
# Prerequisites: Postgres reachable at $AOP_DB_URL (devcontainer default is fine),
#                binaries in ./bin/, ansible-playbook on PATH.
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
API_PORT=18080
AGENT_PORT=17000
AGENT_ADDR="http://127.0.0.1:${AGENT_PORT}"
API_URL="http://127.0.0.1:${API_PORT}"
POLL_TIMEOUT=60  # seconds to wait for job completion

# ── secrets ──────────────────────────────────────────────────────────────────
ENCRYPTION_KEY=$(openssl rand -hex 32)
JWT_SECRET=$(openssl rand -hex 32)
REG_TOKEN=$(openssl rand -hex 16)

# ── temp workspace ────────────────────────────────────────────────────────────
WORK_DIR=$(mktemp -d /tmp/aop-smoke-XXXXXX)
AGENT_WORK_DIR="${WORK_DIR}/agent"
REPO_DIR="${WORK_DIR}/playbook-repo"
LOG_DIR="${WORK_DIR}/logs"
mkdir -p "$AGENT_WORK_DIR" "$REPO_DIR" "$LOG_DIR"

# ── cleanup on exit ───────────────────────────────────────────────────────────
cleanup() {
    local ec=$?
    echo ""
    echo "── cleanup ──"
    kill "$API_PID"    2>/dev/null || true
    kill "$CTRL_PID"   2>/dev/null || true
    kill "$AGENT_PID"  2>/dev/null || true
    wait 2>/dev/null || true
    rm -rf "$WORK_DIR"

    # Remove all DB records created by this run (order matters for FKs)
    psql "$DB_URL" -q <<SQL 2>/dev/null || true
DELETE FROM jobs          WHERE template_id = '${TEMPLATE_ID:-00000000-0000-0000-0000-000000000000}';
DELETE FROM job_templates WHERE id          = '${TEMPLATE_ID:-00000000-0000-0000-0000-000000000000}';
DELETE FROM projects      WHERE id          = '${PROJECT_ID:-00000000-0000-0000-0000-000000000000}';
DELETE FROM agents        WHERE address     = '${AGENT_ADDR}';
DELETE FROM users         WHERE email       = '${TEST_EMAIL}';
SQL

    if [[ $ec -eq 0 ]]; then
        echo "SMOKE TEST PASSED"
    else
        echo "SMOKE TEST FAILED (exit $ec)"
    fi
}
trap cleanup EXIT

# ── database: run migrations ──────────────────────────────────────────────────
echo "── migrations ──"
DB_URL="${AOP_DB_URL:-postgresql://postgres:postgres@localhost:5432/postgres?sslmode=disable}"
(cd "$REPO_ROOT" && AOP_DB_URL="$DB_URL" make migrate-up 2>&1)

# ── seed a user (bcrypt hash of "password") ───────────────────────────────────
TEST_EMAIL="smoke@test.local"
BCRYPT_HASH=$(cd "$REPO_ROOT" && go run ./api/cmd/genhash/)
psql "$DB_URL" -q <<SQL
INSERT INTO users (id, email, password_hash, created_at, updated_at)
VALUES (gen_random_uuid(), '${TEST_EMAIL}', '${BCRYPT_HASH}', now(), now())
ON CONFLICT (email) DO UPDATE SET password_hash = EXCLUDED.password_hash;
SQL

# ── create a local git repo with a no-op playbook ─────────────────────────────
echo "── creating local playbook repo ──"
git -C "$REPO_DIR" init -q -b main
cat > "${REPO_DIR}/noop.yaml" <<'YAML'
---
- name: Smoke test no-op
  hosts: localhost
  gather_facts: false
  tasks:
    - name: ping
      ansible.builtin.ping:
YAML
git -C "$REPO_DIR" add noop.yaml
git -C "$REPO_DIR" -c user.email="ci@aop" -c user.name="CI" commit -qm "add no-op playbook"

# ── start API ─────────────────────────────────────────────────────────────────
echo "── starting API on :${API_PORT} ──"
AOP_DB_URL="$DB_URL" \
AOP_ENCRYPTION_KEY="$ENCRYPTION_KEY" \
AOP_JWT_SECRET="$JWT_SECRET" \
AOP_REGISTRATION_TOKEN="$REG_TOKEN" \
AOP_PORT="$API_PORT" \
AOP_LOG_FORMAT=pretty \
    "${REPO_ROOT}/bin/api" >"${LOG_DIR}/api.log" 2>&1 &
API_PID=$!

# wait for health
for i in $(seq 1 30); do
    if curl -sf "${API_URL}/health" >/dev/null 2>&1; then
        echo "  API ready"
        break
    fi
    sleep 1
    if [[ $i -eq 30 ]]; then
        echo "ERROR: API did not become healthy after 30s"
        cat "${LOG_DIR}/api.log"
        exit 1
    fi
done

# ── start controller ──────────────────────────────────────────────────────────
echo "── starting controller ──"
AOP_DB_URL="$DB_URL" \
AOP_API_URL="$API_URL" \
AOP_ENCRYPTION_KEY="$ENCRYPTION_KEY" \
AOP_RECONCILE_INTERVAL=2s \
AOP_LOG_FORMAT=pretty \
    "${REPO_ROOT}/bin/controller" >"${LOG_DIR}/controller.log" 2>&1 &
CTRL_PID=$!

# ── start agent ───────────────────────────────────────────────────────────────
echo "── starting agent ──"
AOP_API_URL="$API_URL" \
AOP_REGISTRATION_TOKEN="$REG_TOKEN" \
AOP_AGENT_ADDRESS="$AGENT_ADDR" \
AOP_PORT="$AGENT_PORT" \
AOP_WORK_DIR="$AGENT_WORK_DIR" \
AOP_HEARTBEAT_INTERVAL=2s \
AOP_LOG_FORMAT=console \
    "${REPO_ROOT}/bin/agent" >"${LOG_DIR}/agent.log" 2>&1 &
AGENT_PID=$!

# give processes a moment to settle, then poll until we see an online agent
sleep 2

# ── login ─────────────────────────────────────────────────────────────────────
echo "── logging in ──"
JWT=$(curl -sf -X POST "${API_URL}/auth/login" \
    -H "Content-Type: application/json" \
    -d "{\"email\":\"${TEST_EMAIL}\",\"password\":\"password\"}" \
    | grep -o '"token":"[^"]*"' | cut -d'"' -f4)

if [[ -z "$JWT" ]]; then
    echo "ERROR: login failed"
    cat "${LOG_DIR}/api.log"
    exit 1
fi
echo "  token acquired"

AUTH="Authorization: Bearer ${JWT}"

# ── wait for agent to be online ───────────────────────────────────────────────
echo "── waiting for agent to be online ──"
for i in $(seq 1 20); do
    AGENT_STATUS=$(curl -sf "${API_URL}/agents" -H "$AUTH" \
        | grep -o '"status":"[^"]*"' | head -1 | cut -d'"' -f4 || echo "")
    if [[ "$AGENT_STATUS" == "online" ]]; then
        echo "  agent is online"
        break
    fi
    sleep 1
    if [[ $i -eq 20 ]]; then
        echo "ERROR: agent never came online after 20s"
        cat "${LOG_DIR}/agent.log"
        exit 1
    fi
done

# ── create project ────────────────────────────────────────────────────────────
echo "── creating project ──"
PROJECT_ID=$(curl -sf -X POST "${API_URL}/projects" \
    -H "Content-Type: application/json" \
    -H "$AUTH" \
    -d "{\"name\":\"smoke-project\",\"repo_url\":\"file://${REPO_DIR}\",\"branch\":\"main\",\"inventory_path\":\"inventory.ini\"}" \
    | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)

if [[ -z "$PROJECT_ID" ]]; then
    echo "ERROR: create project failed"
    exit 1
fi
echo "  project: $PROJECT_ID"

# ── create template ───────────────────────────────────────────────────────────
echo "── creating job template ──"
TEMPLATE_ID=$(curl -sf -X POST "${API_URL}/job-templates" \
    -H "Content-Type: application/json" \
    -H "$AUTH" \
    -d "{\"name\":\"smoke-template\",\"project_id\":\"${PROJECT_ID}\",\"playbook\":\"noop.yaml\"}" \
    | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)

if [[ -z "$TEMPLATE_ID" ]]; then
    echo "ERROR: create template failed"
    exit 1
fi
echo "  template: $TEMPLATE_ID"

# ── dispatch a job ────────────────────────────────────────────────────────────
echo "── dispatching job ──"
JOB_ID=$(curl -sf -X POST "${API_URL}/jobs" \
    -H "Content-Type: application/json" \
    -H "$AUTH" \
    -d "{\"template_id\":\"${TEMPLATE_ID}\"}" \
    | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)

if [[ -z "$JOB_ID" ]]; then
    echo "ERROR: create job failed"
    exit 1
fi
echo "  job: $JOB_ID"

# ── poll until terminal ───────────────────────────────────────────────────────
echo "── polling for completion (timeout ${POLL_TIMEOUT}s) ──"
DEADLINE=$(( $(date +%s) + POLL_TIMEOUT ))
STATUS=""
while [[ $(date +%s) -lt $DEADLINE ]]; do
    STATUS=$(curl -sf "${API_URL}/jobs/${JOB_ID}" \
        -H "$AUTH" \
        | grep -o '"status":"[^"]*"' | cut -d'"' -f4)
    echo "  status: ${STATUS}"
    case "$STATUS" in
        success|failed|cancelled) break ;;
    esac
    sleep 2
done

# ── assert ────────────────────────────────────────────────────────────────────
echo ""
echo "── result ──"
echo "  job ${JOB_ID} finished with status: ${STATUS}"

if [[ "$STATUS" != "success" ]]; then
    echo ""
    echo "=== API log ==="
    cat "${LOG_DIR}/api.log"
    echo ""
    echo "=== controller log ==="
    cat "${LOG_DIR}/controller.log"
    echo ""
    echo "=== agent log ==="
    cat "${LOG_DIR}/agent.log"
    exit 1
fi
