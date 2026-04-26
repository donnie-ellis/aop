package secrets

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/json"
	"errors"
	"io"
	"testing"

	"github.com/donnie-ellis/aop/pkg/types"
	"github.com/google/uuid"
)

// ---------------------------------------------------------------------------
// Test helpers
// ---------------------------------------------------------------------------

func newKey(t *testing.T) []byte {
	t.Helper()
	key := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, key); err != nil {
		t.Fatalf("generate key: %v", err)
	}
	return key
}

// encrypt mirrors the API's AES-256-GCM encrypt so we can produce test blobs.
func encrypt(key, plaintext []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}
	return gcm.Seal(nonce, nonce, plaintext, nil), nil
}

type mockDB struct {
	cred *types.Credential
	err  error
}

func (m *mockDB) GetCredentialWithSecret(_ context.Context, _ uuid.UUID) (*types.Credential, error) {
	return m.cred, m.err
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

func TestResolve_Success(t *testing.T) {
	key := newKey(t)
	fields := map[string]string{"private_key": "PEM-DATA"}
	plaintext, _ := json.Marshal(fields)
	ciphertext, err := encrypt(key, plaintext)
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}

	cred := &types.Credential{
		ID:            uuid.New(),
		Kind:          types.CredentialKindSSHKey,
		EncryptedData: ciphertext,
	}
	p := NewProvider(&mockDB{cred: cred}, key)

	secret, err := p.Resolve(cred.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if secret.Kind != types.CredentialKindSSHKey {
		t.Errorf("kind: got %q", secret.Kind)
	}
	if secret.Fields["private_key"] != "PEM-DATA" {
		t.Errorf("field: got %q", secret.Fields["private_key"])
	}
}

func TestResolve_NilKey(t *testing.T) {
	p := NewProvider(&mockDB{cred: &types.Credential{}}, nil)
	_, err := p.Resolve(uuid.New())
	if err == nil {
		t.Fatal("expected error when key is nil")
	}
}

func TestResolve_EmptyKey(t *testing.T) {
	p := NewProvider(&mockDB{cred: &types.Credential{}}, []byte{})
	_, err := p.Resolve(uuid.New())
	if err == nil {
		t.Fatal("expected error when key is empty")
	}
}

func TestResolve_DBError(t *testing.T) {
	p := NewProvider(&mockDB{err: errors.New("not found")}, newKey(t))
	_, err := p.Resolve(uuid.New())
	if err == nil {
		t.Fatal("expected error when DB lookup fails")
	}
}

func TestResolve_WrongKey(t *testing.T) {
	encKey := newKey(t)
	fields := map[string]string{"token": "abc"}
	plaintext, _ := json.Marshal(fields)
	ciphertext, _ := encrypt(encKey, plaintext)

	cred := &types.Credential{ID: uuid.New(), Kind: types.CredentialKindHTTPSToken, EncryptedData: ciphertext}
	wrongKey := newKey(t)
	p := NewProvider(&mockDB{cred: cred}, wrongKey)

	_, err := p.Resolve(cred.ID)
	if err == nil {
		t.Fatal("expected error when decrypting with wrong key")
	}
}

func TestResolve_TruncatedCiphertext(t *testing.T) {
	cred := &types.Credential{
		ID:            uuid.New(),
		Kind:          types.CredentialKindVaultPassword,
		EncryptedData: []byte{0x01, 0x02}, // too short for any GCM nonce
	}
	p := NewProvider(&mockDB{cred: cred}, newKey(t))
	_, err := p.Resolve(cred.ID)
	if err == nil {
		t.Fatal("expected error for truncated ciphertext")
	}
}

func TestResolve_AllCredentialKinds(t *testing.T) {
	cases := []struct {
		kind   types.CredentialKind
		fields map[string]string
	}{
		{types.CredentialKindSSHKey, map[string]string{"private_key": "PEM"}},
		{types.CredentialKindHTTPSToken, map[string]string{"token": "ghp_abc"}},
		{types.CredentialKindVaultPassword, map[string]string{"password": "s3cr3t"}},
		{types.CredentialKindMachineUser, map[string]string{"username": "deploy", "private_key": "PEM"}},
	}

	key := newKey(t)
	for _, tc := range cases {
		t.Run(string(tc.kind), func(t *testing.T) {
			plaintext, _ := json.Marshal(tc.fields)
			ciphertext, _ := encrypt(key, plaintext)
			cred := &types.Credential{ID: uuid.New(), Kind: tc.kind, EncryptedData: ciphertext}
			p := NewProvider(&mockDB{cred: cred}, key)

			secret, err := p.Resolve(cred.ID)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if secret.Kind != tc.kind {
				t.Errorf("kind: got %q, want %q", secret.Kind, tc.kind)
			}
			for k, want := range tc.fields {
				if got := secret.Fields[k]; got != want {
					t.Errorf("field %q: got %q, want %q", k, got, want)
				}
			}
		})
	}
}
