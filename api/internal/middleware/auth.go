package middleware

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"strings"

	"github.com/donnie-ellis/aop/api/internal/auth"
	"github.com/donnie-ellis/aop/api/internal/store"
	"github.com/donnie-ellis/aop/pkg/types"
	"github.com/google/uuid"
)

type contextKey string

const (
	CtxKeyUserID contextKey = "user_id"
	CtxKeyAgent  contextKey = "agent"
)

func UserIDFromCtx(ctx context.Context) (uuid.UUID, bool) {
	v, ok := ctx.Value(CtxKeyUserID).(uuid.UUID)
	return v, ok
}

func AgentFromCtx(ctx context.Context) (*types.Agent, bool) {
	v, ok := ctx.Value(CtxKeyAgent).(*types.Agent)
	return v, ok
}

// RequireUser validates a JWT or API token and injects the user ID into context.
func RequireUser(jwtSecret []byte, st *store.Store) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			raw := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
			if raw == "" {
				http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
				return
			}

			// Try JWT first.
			if claims, err := auth.ParseToken(jwtSecret, raw); err == nil {
				ctx := context.WithValue(r.Context(), CtxKeyUserID, claims.UserID)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}

			// Fall back to API token (SHA-256 hash lookup).
			hash := hashToken(raw)
			token, err := st.GetAPITokenByHash(r.Context(), hash)
			if err != nil {
				http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
				return
			}
			go st.TouchAPIToken(r.Context(), token.ID) //nolint:errcheck
			ctx := context.WithValue(r.Context(), CtxKeyUserID, token.UserID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequireAgent validates an agent bearer token and injects the Agent into context.
func RequireAgent(st *store.Store) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			raw := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
			if raw == "" {
				http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
				return
			}
			agent, err := st.GetAgentByTokenHash(r.Context(), hashToken(raw))
			if err != nil {
				http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
				return
			}
			ctx := context.WithValue(r.Context(), CtxKeyAgent, agent)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func hashToken(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}
