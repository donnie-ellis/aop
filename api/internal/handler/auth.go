package handler

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"

	"github.com/donnie-ellis/aop/api/internal/auth"
	"github.com/donnie-ellis/aop/api/internal/middleware"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

func (h *Handlers) Login(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request")
		return
	}

	user, err := h.store.GetUserByEmail(r.Context(), body.Email)
	if err != nil {
		respondError(w, http.StatusUnauthorized, "invalid credentials")
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(body.Password)); err != nil {
		respondError(w, http.StatusUnauthorized, "invalid credentials")
		return
	}

	token, err := auth.IssueToken(h.cfg.JWTSecret, user.ID, user.Email, h.cfg.JWTExpiry)
	if err != nil {
		h.log.Error().Err(err).Msg("issue token")
		respondError(w, http.StatusInternalServerError, "internal error")
		return
	}

	respond(w, http.StatusOK, map[string]string{"token": token})
}

func (h *Handlers) CreateAPIToken(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromCtx(r.Context())
	if !ok {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var body struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Name == "" {
		respondError(w, http.StatusBadRequest, "name is required")
		return
	}

	raw, hash := generateToken()
	token, err := h.store.CreateAPIToken(r.Context(), userID, body.Name, hash)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "internal error")
		return
	}

	respond(w, http.StatusCreated, map[string]any{
		"id":         token.ID,
		"name":       token.Name,
		"token":      raw, // shown once
		"created_at": token.CreatedAt,
	})
}

func (h *Handlers) ListAPITokens(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromCtx(r.Context())
	if !ok {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	tokens, err := h.store.ListAPITokens(r.Context(), userID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "internal error")
		return
	}
	respond(w, http.StatusOK, tokens)
}

func (h *Handlers) DeleteAPIToken(w http.ResponseWriter, r *http.Request) {
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
	if err := h.store.DeleteAPIToken(r.Context(), id, userID); err != nil {
		respondStoreErr(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func generateToken() (raw, hash string) {
	b := make([]byte, 32)
	rand.Read(b) //nolint:errcheck
	raw = hex.EncodeToString(b)
	sum := sha256.Sum256([]byte(raw))
	hash = hex.EncodeToString(sum[:])
	return
}
