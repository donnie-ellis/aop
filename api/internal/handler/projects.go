package handler

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

func (h *Handlers) ListProjects(w http.ResponseWriter, r *http.Request) {
	projects, err := h.store.ListProjects(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, "internal error")
		return
	}
	respond(w, http.StatusOK, projects)
}

func (h *Handlers) CreateProject(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name          string  `json:"name"`
		RepoURL       string  `json:"repo_url"`
		Branch        string  `json:"branch"`
		InventoryPath string  `json:"inventory_path"`
		CredentialID  *string `json:"credential_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request")
		return
	}
	if body.Name == "" || body.RepoURL == "" || body.InventoryPath == "" {
		respondError(w, http.StatusBadRequest, "name, repo_url, and inventory_path are required")
		return
	}
	if body.Branch == "" {
		body.Branch = "main"
	}

	var credID *uuid.UUID
	if body.CredentialID != nil {
		id, err := uuid.Parse(*body.CredentialID)
		if err != nil {
			respondError(w, http.StatusBadRequest, "invalid credential_id")
			return
		}
		credID = &id
	}

	project, err := h.store.CreateProject(r.Context(), body.Name, body.RepoURL, body.Branch, body.InventoryPath, credID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "internal error")
		return
	}
	respond(w, http.StatusCreated, project)
}

func (h *Handlers) GetProject(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid id")
		return
	}
	project, err := h.store.GetProject(r.Context(), id)
	if err != nil {
		respondStoreErr(w, err)
		return
	}
	respond(w, http.StatusOK, project)
}

func (h *Handlers) UpdateProject(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid id")
		return
	}
	var body struct {
		Name          string  `json:"name"`
		RepoURL       string  `json:"repo_url"`
		Branch        string  `json:"branch"`
		InventoryPath string  `json:"inventory_path"`
		CredentialID  *string `json:"credential_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request")
		return
	}

	var credID *uuid.UUID
	if body.CredentialID != nil {
		parsed, err := uuid.Parse(*body.CredentialID)
		if err != nil {
			respondError(w, http.StatusBadRequest, "invalid credential_id")
			return
		}
		credID = &parsed
	}

	project, err := h.store.UpdateProject(r.Context(), id, body.Name, body.RepoURL, body.Branch, body.InventoryPath, credID)
	if err != nil {
		respondStoreErr(w, err)
		return
	}
	respond(w, http.StatusOK, project)
}

func (h *Handlers) DeleteProject(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid id")
		return
	}
	if err := h.store.DeleteProject(r.Context(), id); err != nil {
		respondStoreErr(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handlers) SyncProject(w http.ResponseWriter, r *http.Request) {
	// Enqueues a sync request; the controller's git sync sub-component picks it up.
	// For now, mark the project as pending sync and return 202.
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid id")
		return
	}
	if _, err := h.store.GetProject(r.Context(), id); err != nil {
		respondStoreErr(w, err)
		return
	}
	syncErr := (*string)(nil)
	if err := h.store.UpdateProjectSyncStatus(r.Context(), id, "pending", syncErr); err != nil {
		respondError(w, http.StatusInternalServerError, "internal error")
		return
	}
	w.WriteHeader(http.StatusAccepted)
}

func (h *Handlers) ListInventoryHosts(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid id")
		return
	}
	hosts, err := h.store.ListInventoryHosts(r.Context(), id)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "internal error")
		return
	}
	respond(w, http.StatusOK, hosts)
}
