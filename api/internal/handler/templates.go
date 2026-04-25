package handler

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

func (h *Handlers) ListJobTemplates(w http.ResponseWriter, r *http.Request) {
	templates, err := h.store.ListJobTemplates(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, "internal error")
		return
	}
	respond(w, http.StatusOK, templates)
}

func (h *Handlers) CreateJobTemplate(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name             string         `json:"name"`
		Description      string         `json:"description"`
		ProjectID        string         `json:"project_id"`
		Playbook         string         `json:"playbook"`
		CredentialID     *string        `json:"credential_id"`
		DefaultExtraVars map[string]any `json:"default_extra_vars"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request")
		return
	}
	if body.Name == "" || body.ProjectID == "" || body.Playbook == "" {
		respondError(w, http.StatusBadRequest, "name, project_id, and playbook are required")
		return
	}

	projectID, err := uuid.Parse(body.ProjectID)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid project_id")
		return
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

	tmpl, err := h.store.CreateJobTemplate(r.Context(), body.Name, body.Description, projectID, body.Playbook, credID, body.DefaultExtraVars)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "internal error")
		return
	}
	respond(w, http.StatusCreated, tmpl)
}

func (h *Handlers) GetJobTemplate(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid id")
		return
	}
	tmpl, err := h.store.GetJobTemplate(r.Context(), id)
	if err != nil {
		respondStoreErr(w, err)
		return
	}
	respond(w, http.StatusOK, tmpl)
}

func (h *Handlers) UpdateJobTemplate(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid id")
		return
	}
	var body struct {
		Name             string         `json:"name"`
		Description      string         `json:"description"`
		Playbook         string         `json:"playbook"`
		CredentialID     *string        `json:"credential_id"`
		DefaultExtraVars map[string]any `json:"default_extra_vars"`
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
	tmpl, err := h.store.UpdateJobTemplate(r.Context(), id, body.Name, body.Description, body.Playbook, credID, body.DefaultExtraVars)
	if err != nil {
		respondStoreErr(w, err)
		return
	}
	respond(w, http.StatusOK, tmpl)
}

func (h *Handlers) DeleteJobTemplate(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid id")
		return
	}
	if err := h.store.DeleteJobTemplate(r.Context(), id); err != nil {
		respondStoreErr(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
