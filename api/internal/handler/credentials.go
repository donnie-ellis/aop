package handler

import (
	"encoding/json"
	"net/http"

	"github.com/donnie-ellis/aop/api/internal/secrets"
	"github.com/donnie-ellis/aop/pkg/types"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

func (h *Handlers) ListCredentials(w http.ResponseWriter, r *http.Request) {
	creds, err := h.store.ListCredentials(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, "internal error")
		return
	}
	respond(w, http.StatusOK, creds)
}

func (h *Handlers) CreateCredential(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name        string                 `json:"name"`
		Description string                 `json:"description"`
		Kind        types.CredentialKind   `json:"kind"`
		Fields      map[string]string      `json:"fields"` // plaintext — encrypted before storage
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request")
		return
	}
	if body.Name == "" || body.Kind == "" {
		respondError(w, http.StatusBadRequest, "name and kind are required")
		return
	}

	plaintext, err := json.Marshal(body.Fields)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid fields")
		return
	}
	encrypted, err := secrets.Encrypt(h.cfg.EncryptionKey, plaintext)
	if err != nil {
		h.log.Error().Err(err).Msg("encrypt credential")
		respondError(w, http.StatusInternalServerError, "internal error")
		return
	}

	cred, err := h.store.CreateCredential(r.Context(), body.Name, body.Description, body.Kind, encrypted)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "internal error")
		return
	}
	respond(w, http.StatusCreated, cred)
}

func (h *Handlers) GetCredential(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid id")
		return
	}
	cred, err := h.store.GetCredential(r.Context(), id)
	if err != nil {
		respondStoreErr(w, err)
		return
	}
	respond(w, http.StatusOK, cred)
}

func (h *Handlers) UpdateCredential(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid id")
		return
	}
	var body struct {
		Name        string            `json:"name"`
		Description string            `json:"description"`
		Kind        types.CredentialKind `json:"kind"`
		Fields      map[string]string `json:"fields"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request")
		return
	}
	if body.Name == "" || body.Kind == "" {
		respondError(w, http.StatusBadRequest, "name and kind are required")
		return
	}
	plaintext, err := json.Marshal(body.Fields)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid fields")
		return
	}
	encrypted, err := secrets.Encrypt(h.cfg.EncryptionKey, plaintext)
	if err != nil {
		h.log.Error().Err(err).Msg("encrypt credential")
		respondError(w, http.StatusInternalServerError, "internal error")
		return
	}
	cred, err := h.store.UpdateCredential(r.Context(), id, body.Name, body.Description, body.Kind, encrypted)
	if err != nil {
		respondStoreErr(w, err)
		return
	}
	respond(w, http.StatusOK, cred)
}

func (h *Handlers) DeleteCredential(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid id")
		return
	}
	if err := h.store.DeleteCredential(r.Context(), id); err != nil {
		respondStoreErr(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
