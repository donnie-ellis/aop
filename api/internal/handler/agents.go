package handler

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"

	"github.com/donnie-ellis/aop/api/internal/middleware"
	"github.com/donnie-ellis/aop/pkg/types"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

func (h *Handlers) RegisterAgent(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name     string            `json:"name"`
		Address  string            `json:"address"`
		Labels   map[string]string `json:"labels"`
		Capacity int               `json:"capacity"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request")
		return
	}
	if body.Name == "" || body.Address == "" {
		respondError(w, http.StatusBadRequest, "name and address are required")
		return
	}

	regToken := r.Header.Get("X-Registration-Token")
	if !h.reg.Valid(regToken) {
		respondError(w, http.StatusUnauthorized, "invalid registration token")
		return
	}

	if body.Capacity <= 0 {
		body.Capacity = 1
	}
	if body.Labels == nil {
		body.Labels = map[string]string{}
	}

	raw := make([]byte, 32)
	rand.Read(raw) //nolint:errcheck
	rawStr := hex.EncodeToString(raw)
	sum := sha256.Sum256([]byte(rawStr))
	tokenHash := hex.EncodeToString(sum[:])

	agent, err := h.store.CreateAgent(r.Context(), body.Name, body.Address, tokenHash, body.Labels, body.Capacity)
	if err != nil {
		h.log.Error().Err(err).Msg("create agent")
		respondError(w, http.StatusInternalServerError, "internal error")
		return
	}

	respond(w, http.StatusCreated, map[string]any{
		"agent_id": agent.ID,
		"token":    rawStr, // returned once; agent stores it in memory
	})
}

func (h *Handlers) AgentHeartbeat(w http.ResponseWriter, r *http.Request) {
	agent, ok := middleware.AgentFromCtx(r.Context())
	if !ok {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	var body types.AgentHeartbeat
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request")
		return
	}
	if err := h.store.UpdateAgentHeartbeat(r.Context(), agent.ID, body.RunningJobs, h.cfg.HeartbeatTTL); err != nil {
		respondStoreErr(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handlers) ListAgents(w http.ResponseWriter, r *http.Request) {
	agents, err := h.store.ListAgents(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, "internal error")
		return
	}
	respond(w, http.StatusOK, agents)
}

func (h *Handlers) GetAgent(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid id")
		return
	}
	agent, err := h.store.GetAgentByID(r.Context(), id)
	if err != nil {
		respondStoreErr(w, err)
		return
	}
	respond(w, http.StatusOK, agent)
}
