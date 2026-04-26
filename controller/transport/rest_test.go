package transport

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/donnie-ellis/aop/pkg/types"
	"github.com/google/uuid"
)

// mockResolver implements addressResolver using a fixed address.
type mockResolver struct {
	address string
	err     error
}

func (m *mockResolver) GetAgentAddress(_ context.Context, _ uuid.UUID) (string, error) {
	return m.address, m.err
}

func agentID() uuid.UUID { return uuid.New() }

func TestDispatch_Success(t *testing.T) {
	var gotPayload types.JobPayload
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/dispatch" || r.Method != http.MethodPost {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &gotPayload)
		w.WriteHeader(http.StatusAccepted)
	}))
	defer srv.Close()

	xport := New(&mockResolver{address: srv.URL})
	payload := types.JobPayload{JobID: uuid.New(), Playbook: "site.yml"}

	if err := xport.Dispatch(context.Background(), agentID(), payload); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotPayload.Playbook != "site.yml" {
		t.Errorf("payload not forwarded: got %q", gotPayload.Playbook)
	}
}

func TestDispatch_AgentRejectsWithNon202(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "at capacity", http.StatusServiceUnavailable)
	}))
	defer srv.Close()

	xport := New(&mockResolver{address: srv.URL})
	err := xport.Dispatch(context.Background(), agentID(), types.JobPayload{JobID: uuid.New()})
	if err == nil {
		t.Fatal("expected error for 503 response")
	}
}

func TestDispatch_ResolverError(t *testing.T) {
	xport := New(&mockResolver{err: errResolve()})
	err := xport.Dispatch(context.Background(), agentID(), types.JobPayload{})
	if err == nil {
		t.Fatal("expected error when resolver fails")
	}
}

func TestDispatch_AgentUnreachable(t *testing.T) {
	// Port 1 is never listening.
	xport := New(&mockResolver{address: "http://127.0.0.1:1"})
	err := xport.Dispatch(context.Background(), agentID(), types.JobPayload{})
	if err == nil {
		t.Fatal("expected error for unreachable agent")
	}
}

func TestCancel_Success(t *testing.T) {
	jobID := uuid.New()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		want := "/jobs/" + jobID.String() + "/cancel"
		if r.URL.Path != want {
			t.Errorf("path: got %q, want %q", r.URL.Path, want)
		}
		w.WriteHeader(http.StatusAccepted)
	}))
	defer srv.Close()

	xport := New(&mockResolver{address: srv.URL})
	if err := xport.Cancel(context.Background(), agentID(), jobID); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCancel_JobNotFoundIsOK(t *testing.T) {
	// 404 from the agent (job already done) is not an error.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	xport := New(&mockResolver{address: srv.URL})
	if err := xport.Cancel(context.Background(), agentID(), uuid.New()); err != nil {
		t.Fatalf("expected 404 to be ignored, got: %v", err)
	}
}

func TestCancel_AgentReturnsError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	xport := New(&mockResolver{address: srv.URL})
	if err := xport.Cancel(context.Background(), agentID(), uuid.New()); err == nil {
		t.Fatal("expected error for 500 response")
	}
}

func TestCancel_ResolverError(t *testing.T) {
	xport := New(&mockResolver{err: errResolve()})
	if err := xport.Cancel(context.Background(), agentID(), uuid.New()); err == nil {
		t.Fatal("expected error when resolver fails")
	}
}

func errResolve() error {
	return &resolveError{}
}

type resolveError struct{}

func (e *resolveError) Error() string { return "resolve failed" }
