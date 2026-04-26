package handler_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/donnie-ellis/aop/api/internal/auth"
	"github.com/donnie-ellis/aop/api/internal/config"
	"github.com/donnie-ellis/aop/api/internal/handler"
	"github.com/donnie-ellis/aop/api/internal/secrets"
	"github.com/donnie-ellis/aop/api/internal/server"
	"github.com/donnie-ellis/aop/api/internal/store"
	"github.com/donnie-ellis/aop/pkg/types"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
	"golang.org/x/crypto/bcrypt"
)

const (
	testPassword  = "hunter2"
	testRegToken  = "test-registration-token-abc"
	testEncKeyHex = "0102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f20"
)

var (
	testJWTSecret = []byte("test-jwt-secret-for-integration-tests")
	testDBURL     = "postgresql://postgres:postgres@localhost:5432/postgres?sslmode=disable"

	ts        *httptest.Server
	testPool  *pgxpool.Pool
	testStore *store.Store
	testCfg   *config.Config
	testUser  *types.User
	userJWT   string
)

func TestMain(m *testing.M) {
	if v := os.Getenv("AOP_DB_URL"); v != "" {
		testDBURL = v
	}

	ctx := context.Background()
	pool, err := store.Connect(ctx, testDBURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "connect to test DB: %v\n", err)
		os.Exit(1)
	}
	testPool = pool
	testStore = store.New(pool)

	truncateAll(ctx, pool)

	encKey := mustDecodeHex(testEncKeyHex)

	testCfg = &config.Config{
		DBUrl:         testDBURL,
		EncryptionKey: encKey,
		JWTSecret:     testJWTSecret,
		JWTExpiry:     time.Hour,
		HeartbeatTTL:  30 * time.Second,
		RegToken:      testRegToken,
		Port:          "0",
		LogLevel:      "disabled",
		LogFormat:     "json",
	}

	hash, _ := bcrypt.GenerateFromPassword([]byte(testPassword), bcrypt.MinCost)
	testUser, err = testStore.CreateUser(ctx, "test@example.com", string(hash))
	if err != nil {
		fmt.Fprintf(os.Stderr, "create test user: %v\n", err)
		os.Exit(1)
	}

	userJWT, err = auth.IssueToken(testJWTSecret, testUser.ID, testUser.Email, time.Hour)
	if err != nil {
		fmt.Fprintf(os.Stderr, "issue test JWT: %v\n", err)
		os.Exit(1)
	}

	log := zerolog.Nop()
	sp := secrets.NewProvider(testStore, encKey)
	reg := auth.NewStaticTokenValidator(testRegToken)
	h := handler.New(testStore, sp, testCfg, reg, log)
	srv := server.New(testCfg, h, testStore, log)
	ts = httptest.NewServer(srv.Handler())

	code := m.Run()

	ts.Close()
	pool.Close()
	os.Exit(code)
}

// truncateAll clears every table in dependency order (leaf → root).
func truncateAll(ctx context.Context, pool *pgxpool.Pool) {
	tables := []string{
		"job_logs", "job_status_events", "workflow_node_runs",
		"approval_requests", "jobs", "workflow_edges", "workflow_nodes",
		"workflow_runs", "workflows", "schedules", "job_templates",
		"inventory_hosts", "projects", "credentials", "api_tokens",
		"agents", "users",
	}
	for _, tbl := range tables {
		if _, err := pool.Exec(ctx, "TRUNCATE TABLE "+tbl+" CASCADE"); err != nil {
			fmt.Fprintf(os.Stderr, "truncate %s: %v\n", tbl, err)
		}
	}
}

// do makes an HTTP request to the test server with an optional Bearer token.
func do(t *testing.T, method, path string, body any, token string) *http.Response {
	t.Helper()
	return doHeaders(t, method, path, body, map[string]string{
		"Authorization": "Bearer " + token,
	})
}

// doHeaders makes an HTTP request with explicit headers (Bearer not set automatically).
func doHeaders(t *testing.T, method, path string, body any, headers map[string]string) *http.Response {
	t.Helper()
	var r io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal request body: %v", err)
		}
		r = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, ts.URL+path, r)
	if err != nil {
		t.Fatalf("new request %s %s: %v", method, path, err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	for k, v := range headers {
		if v != "" && v != "Bearer " {
			req.Header.Set(k, v)
		}
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("do %s %s: %v", method, path, err)
	}
	return resp
}

// decodeJSON decodes the response body into dst and closes the body.
func decodeJSON(t *testing.T, resp *http.Response, dst any) {
	t.Helper()
	defer resp.Body.Close()
	if err := json.NewDecoder(resp.Body).Decode(dst); err != nil {
		t.Fatalf("decode response body: %v", err)
	}
}

// assertStatus fails the test if the response status doesn't match.
func assertStatus(t *testing.T, resp *http.Response, want int) {
	t.Helper()
	if resp.StatusCode != want {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		t.Fatalf("expected status %d, got %d: %s", want, resp.StatusCode, body)
	}
}

// registerAgent calls POST /agents/register with the shared registration token.
func registerAgent(t *testing.T, body map[string]any) *http.Response {
	t.Helper()
	return doHeaders(t, http.MethodPost, "/agents/register", body, map[string]string{
		"X-Registration-Token": testRegToken,
	})
}

// mustCreateProject inserts a project directly via the store.
func mustCreateProject(t *testing.T) *types.Project {
	t.Helper()
	p, err := testStore.CreateProject(context.Background(),
		"proj-"+uuid.New().String()[:8],
		"https://github.com/example/repo",
		"main",
		"inventory/hosts.ini",
		nil,
	)
	if err != nil {
		t.Fatalf("create project fixture: %v", err)
	}
	return p
}

// mustCreateTemplate inserts a job template directly via the store.
func mustCreateTemplate(t *testing.T, projectID uuid.UUID) *types.JobTemplate {
	t.Helper()
	tmpl, err := testStore.CreateJobTemplate(context.Background(),
		"tmpl-"+uuid.New().String()[:8],
		"",
		projectID,
		"site.yml",
		nil,
		map[string]any{},
	)
	if err != nil {
		t.Fatalf("create template fixture: %v", err)
	}
	return tmpl
}

// mustCreateAgent inserts an agent directly via the store and returns the raw token.
func mustCreateAgent(t *testing.T) (*types.Agent, string) {
	t.Helper()
	raw := "agent-token-" + uuid.New().String()
	sum := sha256.Sum256([]byte(raw))
	tokenHash := hex.EncodeToString(sum[:])
	agent, err := testStore.CreateAgent(context.Background(),
		"agent-"+uuid.New().String()[:8],
		"http://127.0.0.1:9999",
		tokenHash,
		map[string]string{"env": "test"},
		5,
	)
	if err != nil {
		t.Fatalf("create agent fixture: %v", err)
	}
	return agent, raw
}

func mustDecodeHex(s string) []byte {
	b, err := hex.DecodeString(s)
	if err != nil {
		panic(fmt.Sprintf("mustDecodeHex: %v", err))
	}
	return b
}
