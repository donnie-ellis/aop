package handler_test

import (
	"net/http"
	"testing"
)

func TestCreateCredential_Success(t *testing.T) {
	resp := do(t, http.MethodPost, "/credentials", map[string]any{
		"name": "my-ssh-key",
		"kind": "ssh_key",
		"fields": map[string]string{
			"private_key": "-----BEGIN OPENSSH PRIVATE KEY-----\nfake\n-----END OPENSSH PRIVATE KEY-----",
		},
	}, userJWT)
	assertStatus(t, resp, http.StatusCreated)

	var body map[string]any
	decodeJSON(t, resp, &body)
	if body["id"] == "" {
		t.Fatal("expected id in response")
	}
	if body["name"] != "my-ssh-key" {
		t.Fatalf("expected name my-ssh-key, got %v", body["name"])
	}
	// encrypted_data and raw fields must never appear.
	for _, banned := range []string{"encrypted_data", "fields", "private_key"} {
		if _, ok := body[banned]; ok {
			t.Fatalf("field %q must not appear in credential create response", banned)
		}
	}
}

func TestCreateCredential_MissingName(t *testing.T) {
	resp := do(t, http.MethodPost, "/credentials", map[string]any{
		"kind":   "ssh_key",
		"fields": map[string]string{},
	}, userJWT)
	assertStatus(t, resp, http.StatusBadRequest)
	resp.Body.Close()
}

func TestCreateCredential_MissingKind(t *testing.T) {
	resp := do(t, http.MethodPost, "/credentials", map[string]any{
		"name":   "no-kind",
		"fields": map[string]string{},
	}, userJWT)
	assertStatus(t, resp, http.StatusBadRequest)
	resp.Body.Close()
}

func TestCreateCredential_RequiresAuth(t *testing.T) {
	resp := do(t, http.MethodPost, "/credentials", map[string]any{
		"name": "x", "kind": "ssh_key",
	}, "")
	assertStatus(t, resp, http.StatusUnauthorized)
	resp.Body.Close()
}

func TestGetCredential(t *testing.T) {
	cr := do(t, http.MethodPost, "/credentials", map[string]any{
		"name": "get-me",
		"kind": "vault_password",
		"fields": map[string]string{
			"password": "s3cr3t",
		},
	}, userJWT)
	assertStatus(t, cr, http.StatusCreated)
	var created map[string]any
	decodeJSON(t, cr, &created)
	id := created["id"].(string)

	resp := do(t, http.MethodGet, "/credentials/"+id, nil, userJWT)
	assertStatus(t, resp, http.StatusOK)

	var body map[string]any
	decodeJSON(t, resp, &body)
	if body["id"] != id {
		t.Fatalf("expected id %s, got %v", id, body["id"])
	}
	// Secrets must not be in the GET response either.
	for _, banned := range []string{"encrypted_data", "fields", "password"} {
		if _, ok := body[banned]; ok {
			t.Fatalf("field %q must not appear in credential GET response", banned)
		}
	}
}

func TestGetCredential_NotFound(t *testing.T) {
	resp := do(t, http.MethodGet, "/credentials/00000000-0000-0000-0000-000000000002", nil, userJWT)
	assertStatus(t, resp, http.StatusNotFound)
	resp.Body.Close()
}

func TestListCredentials(t *testing.T) {
	resp := do(t, http.MethodGet, "/credentials", nil, userJWT)
	assertStatus(t, resp, http.StatusOK)

	var creds []map[string]any
	decodeJSON(t, resp, &creds)
	for _, c := range creds {
		for _, banned := range []string{"encrypted_data", "fields"} {
			if _, ok := c[banned]; ok {
				t.Fatalf("field %q must not appear in credentials list", banned)
			}
		}
	}
}

func TestUpdateCredential(t *testing.T) {
	cr := do(t, http.MethodPost, "/credentials", map[string]any{
		"name":   "update-me",
		"kind":   "ssh_key",
		"fields": map[string]string{"private_key": "old"},
	}, userJWT)
	assertStatus(t, cr, http.StatusCreated)
	var created map[string]any
	decodeJSON(t, cr, &created)
	id := created["id"].(string)

	resp := do(t, http.MethodPut, "/credentials/"+id, map[string]any{
		"name":   "updated-name",
		"kind":   "ssh_key",
		"fields": map[string]string{"private_key": "new-key-data"},
	}, userJWT)
	assertStatus(t, resp, http.StatusOK)

	var body map[string]any
	decodeJSON(t, resp, &body)
	if body["name"] != "updated-name" {
		t.Fatalf("expected updated name, got %v", body["name"])
	}
}

func TestUpdateCredential_NotFound(t *testing.T) {
	resp := do(t, http.MethodPut, "/credentials/00000000-0000-0000-0000-000000000003", map[string]any{
		"name": "x", "kind": "ssh_key", "fields": map[string]string{},
	}, userJWT)
	assertStatus(t, resp, http.StatusNotFound)
	resp.Body.Close()
}

func TestDeleteCredential(t *testing.T) {
	cr := do(t, http.MethodPost, "/credentials", map[string]any{
		"name":   "delete-me",
		"kind":   "ssh_key",
		"fields": map[string]string{"private_key": "x"},
	}, userJWT)
	assertStatus(t, cr, http.StatusCreated)
	var created map[string]any
	decodeJSON(t, cr, &created)
	id := created["id"].(string)

	resp := do(t, http.MethodDelete, "/credentials/"+id, nil, userJWT)
	assertStatus(t, resp, http.StatusNoContent)
	resp.Body.Close()

	// Confirm it's gone.
	resp2 := do(t, http.MethodGet, "/credentials/"+id, nil, userJWT)
	assertStatus(t, resp2, http.StatusNotFound)
	resp2.Body.Close()
}
