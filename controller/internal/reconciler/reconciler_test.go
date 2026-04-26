package reconciler

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/donnie-ellis/aop/pkg/types"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// ---------------------------------------------------------------------------
// Mocks
// ---------------------------------------------------------------------------

type transitionCall struct {
	jobID    uuid.UUID
	from, to types.JobStatus
	agentID  *uuid.UUID
	reason   string
}

type mockStore struct {
	mu sync.Mutex

	pendingJobs         []types.Job
	stuckDispatchedJobs []types.Job
	stuckRunningJobs    []types.Job
	availableAgents     []types.Agent
	template            *types.JobTemplate
	project             *types.Project
	inventoryHosts      []types.InventoryHost

	pendingErr  error
	agentsErr   error
	templateErr error
	projectErr  error

	transitions  []transitionCall
	transitionOK bool
}

func (m *mockStore) GetPendingJobs(_ context.Context) ([]types.Job, error) {
	return m.pendingJobs, m.pendingErr
}
func (m *mockStore) GetStuckDispatchedJobs(_ context.Context, _ time.Duration) ([]types.Job, error) {
	return m.stuckDispatchedJobs, nil
}
func (m *mockStore) GetStuckRunningJobs(_ context.Context, _ time.Duration) ([]types.Job, error) {
	return m.stuckRunningJobs, nil
}
func (m *mockStore) GetAvailableAgents(_ context.Context, _ time.Duration) ([]types.Agent, error) {
	return m.availableAgents, m.agentsErr
}
func (m *mockStore) GetJobTemplate(_ context.Context, _ uuid.UUID) (*types.JobTemplate, error) {
	return m.template, m.templateErr
}
func (m *mockStore) GetProject(_ context.Context, _ uuid.UUID) (*types.Project, error) {
	return m.project, m.projectErr
}
func (m *mockStore) GetInventoryHosts(_ context.Context, _ uuid.UUID) ([]types.InventoryHost, error) {
	return m.inventoryHosts, nil
}
func (m *mockStore) GetCredentialWithSecret(_ context.Context, _ uuid.UUID) (*types.Credential, error) {
	return nil, errors.New("not implemented in mockStore")
}
func (m *mockStore) TransitionJob(_ context.Context, jobID uuid.UUID, from, to types.JobStatus, agentID *uuid.UUID, reason string) (bool, error) {
	m.mu.Lock()
	m.transitions = append(m.transitions, transitionCall{jobID, from, to, agentID, reason})
	m.mu.Unlock()
	return m.transitionOK, nil
}

type mockSecrets struct {
	secret *types.CredentialSecret
	err    error
}

func (m *mockSecrets) Resolve(_ uuid.UUID) (*types.CredentialSecret, error) {
	return m.secret, m.err
}

type mockSelector struct {
	agent *types.Agent
	err   error
}

func (m *mockSelector) Select(_ context.Context, _ []types.Agent) (*types.Agent, error) {
	return m.agent, m.err
}

type mockTransport struct {
	mu          sync.Mutex
	dispatched  []types.JobPayload
	dispatchErr error
}

func (m *mockTransport) Dispatch(_ context.Context, _ uuid.UUID, p types.JobPayload) error {
	m.mu.Lock()
	m.dispatched = append(m.dispatched, p)
	m.mu.Unlock()
	return m.dispatchErr
}
func (m *mockTransport) Cancel(_ context.Context, _ uuid.UUID, _ uuid.UUID) error { return nil }

// ---------------------------------------------------------------------------
// Fixtures
// ---------------------------------------------------------------------------

func defaultCfg() Config {
	return Config{
		APIURL:          "http://api:8080",
		HeartbeatTTL:    30 * time.Second,
		DispatchTimeout: 60 * time.Second,
		RunningTimeout:  3600 * time.Second,
	}
}

func makeJob(status types.JobStatus) types.Job {
	return types.Job{ID: uuid.New(), TemplateID: uuid.New(), Status: status, ExtraVars: map[string]any{}}
}

func makeAgent() *types.Agent {
	return &types.Agent{ID: uuid.New(), Name: "agent-1", RunningJobs: 0}
}

func makeTemplate(credentialID *uuid.UUID) *types.JobTemplate {
	return &types.JobTemplate{
		ID:               uuid.New(),
		Playbook:         "site.yml",
		ProjectID:        uuid.New(),
		DefaultExtraVars: map[string]any{"key": "default"},
		CredentialID:     credentialID,
	}
}

func makeProject() *types.Project {
	return &types.Project{ID: uuid.New(), RepoURL: "https://github.com/org/repo", Branch: "main"}
}

func nop() zerolog.Logger { return zerolog.Nop() }

// ---------------------------------------------------------------------------
// dispatchPending
// ---------------------------------------------------------------------------

func TestDispatchPending_NoPendingJobs(t *testing.T) {
	store := &mockStore{transitionOK: true}
	xport := &mockTransport{}
	r := New(store, &mockSecrets{}, &mockSelector{}, xport, defaultCfg(), nop())

	r.dispatchPending(context.Background())

	xport.mu.Lock()
	defer xport.mu.Unlock()
	if len(xport.dispatched) != 0 {
		t.Errorf("expected no dispatch calls, got %d", len(xport.dispatched))
	}
}

func TestDispatchPending_StoreError(t *testing.T) {
	store := &mockStore{pendingErr: errors.New("db down")}
	r := New(store, &mockSecrets{}, &mockSelector{}, &mockTransport{}, defaultCfg(), nop())
	r.dispatchPending(context.Background()) // must not panic
}

func TestDispatchPending_NoAgentsAvailable(t *testing.T) {
	job := makeJob(types.JobStatusPending)
	store := &mockStore{
		pendingJobs:  []types.Job{job},
		transitionOK: true,
	}
	sel := &mockSelector{err: errors.New("no agents available")}
	r := New(store, &mockSecrets{}, sel, &mockTransport{}, defaultCfg(), nop())

	r.dispatchPending(context.Background())

	store.mu.Lock()
	defer store.mu.Unlock()
	if len(store.transitions) == 0 {
		t.Fatal("expected job to be failed when no agents available")
	}
	last := store.transitions[len(store.transitions)-1]
	if last.to != types.JobStatusFailed {
		t.Errorf("expected transition to failed, got %s", last.to)
	}
}

func TestDispatchPending_SuccessfulDispatch(t *testing.T) {
	credID := uuid.New()
	job := makeJob(types.JobStatusPending)
	tmpl := makeTemplate(&credID)
	job.TemplateID = tmpl.ID

	store := &mockStore{
		pendingJobs:     []types.Job{job},
		availableAgents: []types.Agent{*makeAgent()},
		template:        tmpl,
		project:         makeProject(),
		transitionOK:    true,
	}
	secret := &mockSecrets{secret: &types.CredentialSecret{
		Kind:   types.CredentialKindSSHKey,
		Fields: map[string]string{"private_key": "PEM"},
	}}
	xport := &mockTransport{}
	r := New(store, secret, &mockSelector{agent: makeAgent()}, xport, defaultCfg(), nop())

	r.dispatchPending(context.Background())

	xport.mu.Lock()
	defer xport.mu.Unlock()
	if len(xport.dispatched) != 1 {
		t.Fatalf("expected 1 dispatch, got %d", len(xport.dispatched))
	}
	p := xport.dispatched[0]
	if p.JobID != job.ID {
		t.Errorf("payload job_id: got %v, want %v", p.JobID, job.ID)
	}
	if p.Playbook != "site.yml" {
		t.Errorf("payload playbook: got %q", p.Playbook)
	}
	if p.CallbackURL != "http://api:8080" {
		t.Errorf("callback url: got %q", p.CallbackURL)
	}
	if len(p.Credentials) != 1 {
		t.Errorf("expected 1 credential, got %d", len(p.Credentials))
	}
}

func TestDispatchPending_DispatchFails_JobTransitionedToFailed(t *testing.T) {
	job := makeJob(types.JobStatusPending)
	tmpl := makeTemplate(nil)
	job.TemplateID = tmpl.ID

	store := &mockStore{
		pendingJobs:     []types.Job{job},
		availableAgents: []types.Agent{*makeAgent()},
		template:        tmpl,
		project:         makeProject(),
		transitionOK:    true,
	}
	xport := &mockTransport{dispatchErr: errors.New("connection refused")}
	r := New(store, &mockSecrets{}, &mockSelector{agent: makeAgent()}, xport, defaultCfg(), nop())

	r.dispatchPending(context.Background())

	store.mu.Lock()
	defer store.mu.Unlock()
	// Expect pending->dispatched (claim) then dispatched->failed (revert).
	// dispatchOne returns nil after the revert so dispatchPending does NOT add
	// a third pending->failed transition.
	if len(store.transitions) != 2 {
		t.Fatalf("expected exactly 2 transitions, got %d: %v", len(store.transitions), store.transitions)
	}
	if store.transitions[0].to != types.JobStatusDispatched {
		t.Errorf("first transition should claim dispatched; got %s", store.transitions[0].to)
	}
	if store.transitions[1].from != types.JobStatusDispatched || store.transitions[1].to != types.JobStatusFailed {
		t.Errorf("second transition should be dispatched->failed; got %s->%s", store.transitions[1].from, store.transitions[1].to)
	}
}

func TestDispatchPending_AlreadyClaimed(t *testing.T) {
	job := makeJob(types.JobStatusPending)
	tmpl := makeTemplate(nil)
	job.TemplateID = tmpl.ID

	store := &mockStore{
		pendingJobs:     []types.Job{job},
		availableAgents: []types.Agent{*makeAgent()},
		template:        tmpl,
		project:         makeProject(),
		transitionOK:    false, // concurrent controller claimed it first
	}
	xport := &mockTransport{}
	r := New(store, &mockSecrets{}, &mockSelector{agent: makeAgent()}, xport, defaultCfg(), nop())

	r.dispatchPending(context.Background())

	xport.mu.Lock()
	defer xport.mu.Unlock()
	if len(xport.dispatched) != 0 {
		t.Errorf("should not dispatch when claim fails; got %d", len(xport.dispatched))
	}
}

// ---------------------------------------------------------------------------
// buildPayload
// ---------------------------------------------------------------------------

func TestBuildPayload_ExtraVarsMerge(t *testing.T) {
	job := makeJob(types.JobStatusPending)
	job.ExtraVars = map[string]any{"key": "job-override", "job-only": true}

	tmpl := makeTemplate(nil)
	tmpl.DefaultExtraVars = map[string]any{"key": "default", "tmpl-only": "yes"}

	store := &mockStore{template: tmpl, project: makeProject(), transitionOK: true}
	r := New(store, &mockSecrets{}, &mockSelector{}, &mockTransport{}, defaultCfg(), nop())

	payload, err := r.buildPayload(context.Background(), job, uuid.New())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if payload.ExtraVars["key"] != "job-override" {
		t.Errorf("job vars should override template defaults; got %v", payload.ExtraVars["key"])
	}
	if payload.ExtraVars["tmpl-only"] != "yes" {
		t.Errorf("template-only key missing; got %v", payload.ExtraVars["tmpl-only"])
	}
	if payload.ExtraVars["job-only"] != true {
		t.Errorf("job-only key missing; got %v", payload.ExtraVars["job-only"])
	}
}

func TestBuildPayload_NoCredential(t *testing.T) {
	job := makeJob(types.JobStatusPending)
	store := &mockStore{template: makeTemplate(nil), project: makeProject(), transitionOK: true}
	r := New(store, &mockSecrets{}, &mockSelector{}, &mockTransport{}, defaultCfg(), nop())

	payload, err := r.buildPayload(context.Background(), job, uuid.New())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(payload.Credentials) != 0 {
		t.Errorf("expected no credentials, got %d", len(payload.Credentials))
	}
}

func TestBuildPayload_SecretsError(t *testing.T) {
	credID := uuid.New()
	job := makeJob(types.JobStatusPending)
	store := &mockStore{template: makeTemplate(&credID), project: makeProject(), transitionOK: true}
	secrets := &mockSecrets{err: errors.New("vault unreachable")}
	r := New(store, secrets, &mockSelector{}, &mockTransport{}, defaultCfg(), nop())

	_, err := r.buildPayload(context.Background(), job, uuid.New())
	if err == nil {
		t.Fatal("expected error when secrets provider fails")
	}
}

func TestBuildPayload_RepoAndBranch(t *testing.T) {
	job := makeJob(types.JobStatusPending)
	project := makeProject()
	project.RepoURL = "https://github.com/org/ansible"
	project.Branch = "release/1.0"

	store := &mockStore{template: makeTemplate(nil), project: project, transitionOK: true}
	r := New(store, &mockSecrets{}, &mockSelector{}, &mockTransport{}, defaultCfg(), nop())

	payload, err := r.buildPayload(context.Background(), job, uuid.New())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if payload.RepoURL != "https://github.com/org/ansible" {
		t.Errorf("RepoURL: got %q", payload.RepoURL)
	}
	if payload.RepoBranch != "release/1.0" {
		t.Errorf("RepoBranch: got %q", payload.RepoBranch)
	}
}

func TestBuildPayload_TemplateError(t *testing.T) {
	job := makeJob(types.JobStatusPending)
	store := &mockStore{templateErr: errors.New("not found"), transitionOK: true}
	r := New(store, &mockSecrets{}, &mockSelector{}, &mockTransport{}, defaultCfg(), nop())

	_, err := r.buildPayload(context.Background(), job, uuid.New())
	if err == nil {
		t.Fatal("expected error when template lookup fails")
	}
}

// ---------------------------------------------------------------------------
// failStuck
// ---------------------------------------------------------------------------

func TestFailStuck_DispatchedTimeout(t *testing.T) {
	job := makeJob(types.JobStatusDispatched)
	store := &mockStore{stuckDispatchedJobs: []types.Job{job}, transitionOK: true}
	r := New(store, &mockSecrets{}, &mockSelector{}, &mockTransport{}, defaultCfg(), nop())

	r.failStuck(context.Background())

	store.mu.Lock()
	defer store.mu.Unlock()
	if len(store.transitions) == 0 {
		t.Fatal("expected stuck dispatched job to be transitioned")
	}
	tr := store.transitions[0]
	if tr.from != types.JobStatusDispatched || tr.to != types.JobStatusFailed {
		t.Errorf("expected dispatched→failed, got %s→%s", tr.from, tr.to)
	}
	if tr.reason == "" {
		t.Error("expected non-empty failure reason")
	}
}

func TestFailStuck_RunningTimeout(t *testing.T) {
	job := makeJob(types.JobStatusRunning)
	store := &mockStore{stuckRunningJobs: []types.Job{job}, transitionOK: true}
	r := New(store, &mockSecrets{}, &mockSelector{}, &mockTransport{}, defaultCfg(), nop())

	r.failStuck(context.Background())

	store.mu.Lock()
	defer store.mu.Unlock()
	if len(store.transitions) == 0 {
		t.Fatal("expected stuck running job to be transitioned")
	}
	tr := store.transitions[0]
	if tr.from != types.JobStatusRunning || tr.to != types.JobStatusFailed {
		t.Errorf("expected running→failed, got %s→%s", tr.from, tr.to)
	}
}

func TestFailStuck_NoStuckJobs(t *testing.T) {
	store := &mockStore{transitionOK: true}
	r := New(store, &mockSecrets{}, &mockSelector{}, &mockTransport{}, defaultCfg(), nop())
	r.failStuck(context.Background()) // must not panic or error

	store.mu.Lock()
	defer store.mu.Unlock()
	if len(store.transitions) != 0 {
		t.Errorf("expected no transitions, got %d", len(store.transitions))
	}
}
