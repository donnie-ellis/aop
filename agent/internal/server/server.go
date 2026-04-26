// Package server exposes the small HTTP listener that the controller uses to
// dispatch jobs and request cancellations.
//
// Routes:
//
//	POST /dispatch            – receive a JobPayload and start execution
//	POST /jobs/{id}/cancel    – cancel a running job
package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/donnie-ellis/aop/agent/internal/apiclient"
	"github.com/donnie-ellis/aop/agent/internal/runner"
	"github.com/donnie-ellis/aop/pkg/types"
	"github.com/go-chi/chi/v5"
	chiMiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// Server is the agent's inbound HTTP server.
type Server struct {
	http     *http.Server
	log      zerolog.Logger
	client   *apiclient.Client
	workDir  string
	capacity int

	mu      sync.Mutex
	running map[uuid.UUID]context.CancelFunc
	count   atomic.Int64
}

// New wires up the chi router and returns a ready-to-start Server.
func New(port, workDir string, capacity int, client *apiclient.Client, log zerolog.Logger) *Server {
	s := &Server{
		log:      log,
		client:   client,
		workDir:  workDir,
		capacity: capacity,
		running:  make(map[uuid.UUID]context.CancelFunc),
	}

	r := chi.NewRouter()
	r.Use(chiMiddleware.RequestID)
	r.Use(chiMiddleware.Recoverer)

	r.Post("/dispatch", s.handleDispatch)
	r.Post("/jobs/{id}/cancel", s.handleCancel)

	s.http = &http.Server{
		Addr:         fmt.Sprintf(":%s", port),
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
	}
	return s
}

// Start blocks until the server exits.
func (s *Server) Start() error {
	s.log.Info().Str("addr", s.http.Addr).Msg("dispatch listener started")
	return s.http.ListenAndServe()
}

// Shutdown gracefully stops the HTTP server.
func (s *Server) Shutdown(ctx context.Context) error {
	return s.http.Shutdown(ctx)
}

// RunningJobs returns the current number of in-flight jobs.
func (s *Server) RunningJobs() int {
	return int(s.count.Load())
}

// handleDispatch receives a JobPayload from the controller and starts
// executing the job in a goroutine. Returns 202 immediately.
func (s *Server) handleDispatch(w http.ResponseWriter, r *http.Request) {
	var payload types.JobPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, `{"error":"invalid payload"}`, http.StatusBadRequest)
		return
	}
	if payload.JobID == uuid.Nil {
		http.Error(w, `{"error":"job_id is required"}`, http.StatusBadRequest)
		return
	}

	if int(s.count.Load()) >= s.capacity {
		http.Error(w, `{"error":"at capacity"}`, http.StatusServiceUnavailable)
		return
	}

	jobCtx, cancel := context.WithCancel(context.Background())
	s.mu.Lock()
	s.running[payload.JobID] = cancel
	s.mu.Unlock()
	s.count.Add(1)

	go s.executeJob(jobCtx, payload)

	w.WriteHeader(http.StatusAccepted)
}

// handleCancel signals a running job to stop.
func (s *Server) handleCancel(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, `{"error":"invalid job id"}`, http.StatusBadRequest)
		return
	}

	s.mu.Lock()
	cancel, ok := s.running[id]
	s.mu.Unlock()

	if !ok {
		http.Error(w, `{"error":"job not found"}`, http.StatusNotFound)
		return
	}
	cancel()
	w.WriteHeader(http.StatusAccepted)
}

// executeJob runs the full job lifecycle: execute, stream logs, report result.
func (s *Server) executeJob(ctx context.Context, payload types.JobPayload) {
	defer func() {
		s.mu.Lock()
		delete(s.running, payload.JobID)
		s.mu.Unlock()
		s.count.Add(-1)
	}()

	log := s.log.With().Str("job_id", payload.JobID.String()).Logger()
	log.Info().Msg("job started")

	batcher := runner.NewLogBatcher(
		10,
		500*time.Millisecond,
		func(lines []types.JobLogLine) {
			if err := s.client.PostLogs(context.Background(), payload.JobID, lines); err != nil {
				log.Warn().Err(err).Msg("post logs")
			}
		},
	)

	var seq int
	onLog := func(n int, line, stream string) {
		seq = n
		batcher.Add(types.JobLogLine{Seq: seq, Line: line, Stream: stream})
	}

	result := runner.Run(ctx, runner.Config{
		Payload: payload,
		WorkDir: s.workDir,
		OnLog:   onLog,
	}, log)

	batcher.Flush()

	if err := s.client.PostResult(context.Background(), payload.JobID, result.Status, result.ExitCode, nil); err != nil {
		log.Error().Err(err).Msg("post result")
	}
	log.Info().Str("status", string(result.Status)).Int("exit_code", result.ExitCode).Msg("job finished")
}
