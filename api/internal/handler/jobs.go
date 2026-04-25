package handler

import (
	"encoding/json"
	"net/http"

	"github.com/donnie-ellis/aop/api/internal/middleware"
	"github.com/donnie-ellis/aop/api/internal/store"
	"github.com/donnie-ellis/aop/pkg/types"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

func (h *Handlers) ListJobs(w http.ResponseWriter, r *http.Request) {
	f := store.ListJobsFilter{}
	if s := r.URL.Query().Get("status"); s != "" {
		f.Status = types.JobStatus(s)
	}
	if s := r.URL.Query().Get("template_id"); s != "" {
		id, err := uuid.Parse(s)
		if err == nil {
			f.TemplateID = &id
		}
	}
	if s := r.URL.Query().Get("agent_id"); s != "" {
		id, err := uuid.Parse(s)
		if err == nil {
			f.AgentID = &id
		}
	}
	jobs, err := h.store.ListJobs(r.Context(), f)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "internal error")
		return
	}
	respond(w, http.StatusOK, jobs)
}

func (h *Handlers) CreateJob(w http.ResponseWriter, r *http.Request) {
	var body struct {
		TemplateID string         `json:"template_id"`
		ExtraVars  map[string]any `json:"extra_vars"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request")
		return
	}
	templateID, err := uuid.Parse(body.TemplateID)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid template_id")
		return
	}
	if _, err := h.store.GetJobTemplate(r.Context(), templateID); err != nil {
		respondStoreErr(w, err)
		return
	}

	job, err := h.store.CreateJob(r.Context(), templateID, nil, body.ExtraVars)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "internal error")
		return
	}
	respond(w, http.StatusCreated, job)
}

func (h *Handlers) GetJob(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid id")
		return
	}
	job, err := h.store.GetJob(r.Context(), id)
	if err != nil {
		respondStoreErr(w, err)
		return
	}
	respond(w, http.StatusOK, job)
}

func (h *Handlers) CancelJob(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid id")
		return
	}
	job, err := h.store.GetJob(r.Context(), id)
	if err != nil {
		respondStoreErr(w, err)
		return
	}
	if job.Status.IsTerminal() {
		respondError(w, http.StatusConflict, "job is already in a terminal state")
		return
	}
	reason := "cancelled by user"
	if err := h.store.UpdateJobStatus(r.Context(), id, types.JobStatusCancelled, nil, &reason); err != nil {
		respondError(w, http.StatusInternalServerError, "internal error")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handlers) GetJobLogs(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid id")
		return
	}
	// TODO: upgrade to WebSocket for live streaming; for now return stored logs.
	lines, err := h.store.GetJobLogs(r.Context(), id)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "internal error")
		return
	}
	respond(w, http.StatusOK, lines)
}

// AgentPostLogs is called by the agent to submit a batch of log lines.
func (h *Handlers) AgentPostLogs(w http.ResponseWriter, r *http.Request) {
	agent, ok := middleware.AgentFromCtx(r.Context())
	if !ok {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	jobID, err := uuid.Parse(chi.URLParam(r, "job_id"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid job_id")
		return
	}

	job, err := h.store.GetJob(r.Context(), jobID)
	if err != nil {
		respondStoreErr(w, err)
		return
	}
	if job.AgentID == nil || *job.AgentID != agent.ID {
		respondError(w, http.StatusForbidden, "forbidden")
		return
	}

	var lines []types.JobLogLine
	if err := json.NewDecoder(r.Body).Decode(&lines); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request")
		return
	}
	if err := h.store.AppendJobLogs(r.Context(), jobID, lines); err != nil {
		respondError(w, http.StatusInternalServerError, "internal error")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// AgentPostResult is called by the agent when a job finishes.
func (h *Handlers) AgentPostResult(w http.ResponseWriter, r *http.Request) {
	agent, ok := middleware.AgentFromCtx(r.Context())
	if !ok {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	jobID, err := uuid.Parse(chi.URLParam(r, "job_id"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid job_id")
		return
	}

	job, err := h.store.GetJob(r.Context(), jobID)
	if err != nil {
		respondStoreErr(w, err)
		return
	}
	if job.AgentID == nil || *job.AgentID != agent.ID {
		respondError(w, http.StatusForbidden, "forbidden")
		return
	}

	var body struct {
		Status   types.JobStatus `json:"status"`
		ExitCode int             `json:"exit_code"`
		Facts    map[string]any  `json:"facts"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request")
		return
	}
	if !body.Status.IsTerminal() {
		respondError(w, http.StatusBadRequest, "status must be terminal")
		return
	}

	if err := h.store.UpdateJobStatus(r.Context(), jobID, body.Status, nil, nil); err != nil {
		respondError(w, http.StatusInternalServerError, "internal error")
		return
	}
	if len(body.Facts) > 0 {
		h.store.SetJobFacts(r.Context(), jobID, body.Facts) //nolint:errcheck
	}
	w.WriteHeader(http.StatusNoContent)
}
