package handler

import (
	"encoding/json"
	"net/http"

	"github.com/donnie-ellis/aop/api/internal/middleware"
	"github.com/donnie-ellis/aop/pkg/types"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

func (h *Handlers) ListPendingApprovals(w http.ResponseWriter, r *http.Request) {
	approvals, err := h.store.ListPendingApprovals(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, "internal error")
		return
	}
	respond(w, http.StatusOK, approvals)
}

func (h *Handlers) GetApproval(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid id")
		return
	}
	a, err := h.store.GetApprovalRequest(r.Context(), id)
	if err != nil {
		respondStoreErr(w, err)
		return
	}
	respond(w, http.StatusOK, a)
}

func (h *Handlers) ResolveApproval(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromCtx(r.Context())
	if !ok {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid id")
		return
	}
	action := chi.URLParam(r, "action") // "approve" or "deny"
	var status types.ApprovalStatus
	switch action {
	case "approve":
		status = types.ApprovalStatusApproved
	case "deny":
		status = types.ApprovalStatusDenied
	default:
		respondError(w, http.StatusBadRequest, "action must be approve or deny")
		return
	}

	var body struct {
		Note *string `json:"note"`
	}
	// note is optional; ignore decode errors
	json.NewDecoder(r.Body).Decode(&body) //nolint:errcheck

	if err := h.store.ResolveApproval(r.Context(), id, status, userID, body.Note); err != nil {
		respondStoreErr(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
