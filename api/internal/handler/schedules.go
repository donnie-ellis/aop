package handler

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

func (h *Handlers) ListSchedules(w http.ResponseWriter, r *http.Request) {
	schedules, err := h.store.ListSchedules(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, "internal error")
		return
	}
	respond(w, http.StatusOK, schedules)
}

func (h *Handlers) CreateSchedule(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name       string         `json:"name"`
		TemplateID string         `json:"template_id"`
		CronExpr   string         `json:"cron_expr"`
		Timezone   string         `json:"timezone"`
		Enabled    bool           `json:"enabled"`
		ExtraVars  map[string]any `json:"extra_vars"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request")
		return
	}
	if body.Name == "" || body.TemplateID == "" || body.CronExpr == "" {
		respondError(w, http.StatusBadRequest, "name, template_id, and cron_expr are required")
		return
	}
	templateID, err := uuid.Parse(body.TemplateID)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid template_id")
		return
	}
	if body.Timezone == "" {
		body.Timezone = "UTC"
	}

	sc, err := h.store.CreateSchedule(r.Context(), body.Name, templateID, body.CronExpr, body.Timezone, body.Enabled, body.ExtraVars)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "internal error")
		return
	}
	respond(w, http.StatusCreated, sc)
}

func (h *Handlers) GetSchedule(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid id")
		return
	}
	sc, err := h.store.GetSchedule(r.Context(), id)
	if err != nil {
		respondStoreErr(w, err)
		return
	}
	respond(w, http.StatusOK, sc)
}

func (h *Handlers) UpdateSchedule(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid id")
		return
	}
	var body struct {
		Name      string         `json:"name"`
		CronExpr  string         `json:"cron_expr"`
		Timezone  string         `json:"timezone"`
		Enabled   bool           `json:"enabled"`
		ExtraVars map[string]any `json:"extra_vars"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request")
		return
	}
	sc, err := h.store.UpdateSchedule(r.Context(), id, body.Name, body.CronExpr, body.Timezone, body.Enabled, body.ExtraVars)
	if err != nil {
		respondStoreErr(w, err)
		return
	}
	respond(w, http.StatusOK, sc)
}

func (h *Handlers) DeleteSchedule(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid id")
		return
	}
	if err := h.store.DeleteSchedule(r.Context(), id); err != nil {
		respondStoreErr(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
