package handler

import (
	"encoding/json"
	"net/http"

	"github.com/donnie-ellis/aop/pkg/types"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

func (h *Handlers) ListWorkflows(w http.ResponseWriter, r *http.Request) {
	workflows, err := h.store.ListWorkflows(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, "internal error")
		return
	}
	respond(w, http.StatusOK, workflows)
}

func (h *Handlers) CreateWorkflow(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Name == "" {
		respondError(w, http.StatusBadRequest, "name is required")
		return
	}
	wf, err := h.store.CreateWorkflow(r.Context(), body.Name, body.Description)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "internal error")
		return
	}
	respond(w, http.StatusCreated, wf)
}

func (h *Handlers) GetWorkflow(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid id")
		return
	}
	wf, err := h.store.GetWorkflow(r.Context(), id)
	if err != nil {
		respondStoreErr(w, err)
		return
	}
	nodes, _ := h.store.GetWorkflowNodes(r.Context(), id)
	edges, _ := h.store.GetWorkflowEdges(r.Context(), id)
	respond(w, http.StatusOK, map[string]any{
		"workflow": wf,
		"nodes":    nodes,
		"edges":    edges,
	})
}

func (h *Handlers) UpdateWorkflow(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid id")
		return
	}
	var body struct {
		Name        string               `json:"name"`
		Description string               `json:"description"`
		Nodes       []types.WorkflowNode `json:"nodes"`
		Edges       []types.WorkflowEdge `json:"edges"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request")
		return
	}
	wf, err := h.store.UpdateWorkflow(r.Context(), id, body.Name, body.Description)
	if err != nil {
		respondStoreErr(w, err)
		return
	}
	if body.Nodes != nil {
		if err := h.store.UpsertWorkflowNodes(r.Context(), id, body.Nodes); err != nil {
			respondError(w, http.StatusInternalServerError, "internal error")
			return
		}
	}
	if body.Edges != nil {
		if err := h.store.UpsertWorkflowEdges(r.Context(), id, body.Edges); err != nil {
			respondError(w, http.StatusInternalServerError, "internal error")
			return
		}
	}
	respond(w, http.StatusOK, wf)
}

func (h *Handlers) DeleteWorkflow(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid id")
		return
	}
	if err := h.store.DeleteWorkflow(r.Context(), id); err != nil {
		respondStoreErr(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handlers) RunWorkflow(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid id")
		return
	}
	if _, err := h.store.GetWorkflow(r.Context(), id); err != nil {
		respondStoreErr(w, err)
		return
	}
	run, err := h.store.CreateWorkflowRun(r.Context(), id)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "internal error")
		return
	}
	respond(w, http.StatusCreated, run)
}

func (h *Handlers) ListWorkflowRuns(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid id")
		return
	}
	runs, err := h.store.ListWorkflowRuns(r.Context(), id)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "internal error")
		return
	}
	respond(w, http.StatusOK, runs)
}

func (h *Handlers) GetWorkflowRun(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid id")
		return
	}
	run, err := h.store.GetWorkflowRun(r.Context(), id)
	if err != nil {
		respondStoreErr(w, err)
		return
	}
	respond(w, http.StatusOK, run)
}
