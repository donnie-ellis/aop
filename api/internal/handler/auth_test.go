package handler_test

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"testing"
)

func TestLogin_Success(t *testing.T) {
	resp := do(t, http.MethodPost, "/auth/login", map[string]string{
		"email":    testUser.Email,
		"password": testPassword,
	}, "")
	assertStatus(t, resp, http.StatusOK)

	var body map[string]string
	decodeJSON(t, resp, &body)
	if body["token"] == "" {
		t.Fatal("expected token in response, got empty string")
	}
}

func TestLogin_WrongPassword(t *testing.T) {
	resp := do(t, http.MethodPost, "/auth/login", map[string]string{
		"email":    testUser.Email,
		"password": "wrong-password",
	}, "")
	assertStatus(t, resp, http.StatusUnauthorized)
}

func TestLogin_UnknownUser(t *testing.T) {
	resp := do(t, http.MethodPost, "/auth/login", map[string]string{
		"email":    "nobody@example.com",
		"password": testPassword,
	}, "")
	assertStatus(t, resp, http.StatusUnauthorized)
}

func TestLogin_BadBody(t *testing.T) {
	resp := do(t, http.MethodPost, "/auth/login", nil, "")
	assertStatus(t, resp, http.StatusBadRequest)
}

func TestCreateAPIToken(t *testing.T) {
	resp := do(t, http.MethodPost, "/auth/tokens", map[string]string{
		"name": "ci-token",
	}, userJWT)
	assertStatus(t, resp, http.StatusCreated)

	var body map[string]any
	decodeJSON(t, resp, &body)
	if body["token"] == "" {
		t.Fatal("raw token must be returned on creation")
	}
	if body["id"] == "" {
		t.Fatal("id must be present")
	}
}

func TestCreateAPIToken_MissingName(t *testing.T) {
	resp := do(t, http.MethodPost, "/auth/tokens", map[string]string{"name": ""}, userJWT)
	assertStatus(t, resp, http.StatusBadRequest)
}

func TestCreateAPIToken_RequiresAuth(t *testing.T) {
	resp := do(t, http.MethodPost, "/auth/tokens", map[string]string{"name": "x"}, "")
	assertStatus(t, resp, http.StatusUnauthorized)
}

func TestListAPITokens(t *testing.T) {
	// Create one token first so the list is non-empty.
	cr := do(t, http.MethodPost, "/auth/tokens", map[string]string{"name": "list-test"}, userJWT)
	assertStatus(t, cr, http.StatusCreated)
	cr.Body.Close()

	resp := do(t, http.MethodGet, "/auth/tokens", nil, userJWT)
	assertStatus(t, resp, http.StatusOK)

	var tokens []map[string]any
	decodeJSON(t, resp, &tokens)
	if len(tokens) == 0 {
		t.Fatal("expected at least one token in list")
	}
	// token_hash must never be exposed.
	for _, tok := range tokens {
		if _, ok := tok["token_hash"]; ok {
			t.Fatal("token_hash must not appear in list response")
		}
	}
}

func TestDeleteAPIToken(t *testing.T) {
	// Create a token to delete.
	cr := do(t, http.MethodPost, "/auth/tokens", map[string]string{"name": "delete-me"}, userJWT)
	assertStatus(t, cr, http.StatusCreated)
	var created map[string]any
	decodeJSON(t, cr, &created)
	id := created["id"].(string)

	resp := do(t, http.MethodDelete, "/auth/tokens/"+id, nil, userJWT)
	assertStatus(t, resp, http.StatusNoContent)
	resp.Body.Close()

	// Second delete should 404.
	resp2 := do(t, http.MethodDelete, "/auth/tokens/"+id, nil, userJWT)
	assertStatus(t, resp2, http.StatusNotFound)
	resp2.Body.Close()
}

func TestAPIToken_AuthWorks(t *testing.T) {
	// Create an API token, then use the raw value to authenticate.
	cr := do(t, http.MethodPost, "/auth/tokens", map[string]string{"name": "api-auth-test"}, userJWT)
	assertStatus(t, cr, http.StatusCreated)
	var created map[string]any
	decodeJSON(t, cr, &created)
	rawToken := created["token"].(string)

	// Use raw token (not JWT) to call a protected endpoint.
	resp := do(t, http.MethodGet, "/agents", nil, rawToken)
	assertStatus(t, resp, http.StatusOK)
	resp.Body.Close()
}

func TestAuth_InvalidToken(t *testing.T) {
	resp := do(t, http.MethodGet, "/agents", nil, "not-a-valid-token")
	assertStatus(t, resp, http.StatusUnauthorized)
	resp.Body.Close()
}

func TestAuth_TamperedJWT(t *testing.T) {
	// Flip a character in the signature section of a valid JWT.
	tampered := userJWT[:len(userJWT)-4] + "XXXX"
	resp := do(t, http.MethodGet, "/agents", nil, tampered)
	assertStatus(t, resp, http.StatusUnauthorized)
	resp.Body.Close()
}

func TestAuth_HashLookup_WrongToken(t *testing.T) {
	// A correctly formatted hex string that isn't a real token.
	sum := sha256.Sum256([]byte("nonexistent"))
	fake := hex.EncodeToString(sum[:])
	resp := do(t, http.MethodGet, "/agents", nil, fake)
	assertStatus(t, resp, http.StatusUnauthorized)
	resp.Body.Close()
}
