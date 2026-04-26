package server

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/donnie-ellis/aop/api/internal/config"
	"github.com/donnie-ellis/aop/api/internal/handler"
	"github.com/donnie-ellis/aop/api/internal/middleware"
	"github.com/donnie-ellis/aop/api/internal/store"
	"github.com/go-chi/chi/v5"
	chiMiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/rs/zerolog"
)

type Server struct {
	http *http.Server
	log  zerolog.Logger
}

func New(cfg *config.Config, h *handler.Handlers, st *store.Store, log zerolog.Logger) *Server {
	r := chi.NewRouter()

	r.Use(chiMiddleware.RequestID)
	r.Use(chiMiddleware.RealIP)
	r.Use(chiMiddleware.Recoverer)

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`)) //nolint:errcheck
	})

	// Agent self-registration (public — validated by X-Registration-Token header)
	r.Post("/agents/register", h.RegisterAgent)

	// User auth (public)
	r.Post("/auth/login", h.Login)

	// Authenticated user routes
	r.Group(func(r chi.Router) {
		r.Use(middleware.RequireUser(cfg.JWTSecret, st))

		r.Post("/auth/tokens", h.CreateAPIToken)
		r.Get("/auth/tokens", h.ListAPITokens)
		r.Delete("/auth/tokens/{id}", h.DeleteAPIToken)

		r.Get("/agents", h.ListAgents)
		r.Get("/agents/{id}", h.GetAgent)

		r.Get("/credentials", h.ListCredentials)
		r.Post("/credentials", h.CreateCredential)
		r.Get("/credentials/{id}", h.GetCredential)
		r.Put("/credentials/{id}", h.UpdateCredential)
		r.Delete("/credentials/{id}", h.DeleteCredential)

		r.Get("/projects", h.ListProjects)
		r.Post("/projects", h.CreateProject)
		r.Get("/projects/{id}", h.GetProject)
		r.Put("/projects/{id}", h.UpdateProject)
		r.Delete("/projects/{id}", h.DeleteProject)
		r.Post("/projects/{id}/sync", h.SyncProject)
		r.Get("/projects/{id}/inventory", h.ListInventoryHosts)

		r.Get("/job-templates", h.ListJobTemplates)
		r.Post("/job-templates", h.CreateJobTemplate)
		r.Get("/job-templates/{id}", h.GetJobTemplate)
		r.Put("/job-templates/{id}", h.UpdateJobTemplate)
		r.Delete("/job-templates/{id}", h.DeleteJobTemplate)

		r.Get("/schedules", h.ListSchedules)
		r.Post("/schedules", h.CreateSchedule)
		r.Get("/schedules/{id}", h.GetSchedule)
		r.Put("/schedules/{id}", h.UpdateSchedule)
		r.Delete("/schedules/{id}", h.DeleteSchedule)

		r.Get("/jobs", h.ListJobs)
		r.Post("/jobs", h.CreateJob)
		r.Get("/jobs/{id}", h.GetJob)
		r.Post("/jobs/{id}/cancel", h.CancelJob)
		r.Get("/jobs/{id}/logs", h.GetJobLogs)

		r.Get("/workflows", h.ListWorkflows)
		r.Post("/workflows", h.CreateWorkflow)
		r.Get("/workflows/{id}", h.GetWorkflow)
		r.Put("/workflows/{id}", h.UpdateWorkflow)
		r.Delete("/workflows/{id}", h.DeleteWorkflow)
		r.Post("/workflows/{id}/run", h.RunWorkflow)
		r.Get("/workflows/{id}/runs", h.ListWorkflowRuns)
		r.Get("/workflow-runs/{id}", h.GetWorkflowRun)

		r.Get("/approvals", h.ListPendingApprovals)
		r.Get("/approvals/{id}", h.GetApproval)
		r.Post("/approvals/{id}/{action}", h.ResolveApproval)
	})

	// Agent callback routes (authenticated by agent token)
	r.Group(func(r chi.Router) {
		r.Use(middleware.RequireAgent(st))

		r.Post("/agent/heartbeat", h.AgentHeartbeat)
		r.Post("/agent/jobs/{job_id}/logs", h.AgentPostLogs)
		r.Post("/agent/jobs/{job_id}/result", h.AgentPostResult)
	})

	return &Server{
		http: &http.Server{
			Addr:         fmt.Sprintf(":%s", cfg.Port),
			Handler:      r,
			ReadTimeout:  15 * time.Second,
			WriteTimeout: 30 * time.Second,
			IdleTimeout:  60 * time.Second,
		},
		log: log,
	}
}

// Handler returns the underlying http.Handler for use in tests.
func (s *Server) Handler() http.Handler {
	return s.http.Handler
}

func (s *Server) Start() error {
	s.log.Info().Str("addr", s.http.Addr).Msg("starting api server")
	return s.http.ListenAndServe()
}

func (s *Server) Shutdown(ctx context.Context) error {
	return s.http.Shutdown(ctx)
}
