package handler

import (
	"encoding/json"
	"net/http"

	"github.com/donnie-ellis/aop/api/internal/store"
)

func respond(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data) //nolint:errcheck
}

func respondError(w http.ResponseWriter, status int, msg string) {
	respond(w, status, map[string]string{"error": msg})
}

func respondStoreErr(w http.ResponseWriter, err error) {
	if err == store.ErrNotFound {
		respondError(w, http.StatusNotFound, "not found")
		return
	}
	if err == store.ErrConflict {
		respondError(w, http.StatusConflict, "conflict")
		return
	}
	respondError(w, http.StatusInternalServerError, "internal error")
}
