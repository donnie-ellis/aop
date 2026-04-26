// Package secrets implements types.SecretsProvider for the controller.
// The controller only ever decrypts — encryption is handled by the API server.
package secrets

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/donnie-ellis/aop/pkg/types"
	"github.com/google/uuid"
)

type credentialGetter interface {
	GetCredentialWithSecret(ctx context.Context, id uuid.UUID) (*types.Credential, error)
}

// Provider implements types.SecretsProvider using AES-256-GCM.
// The key must be exactly 32 bytes (from AOP_ENCRYPTION_KEY decoded from hex).
type Provider struct {
	db  credentialGetter
	key []byte
}

func NewProvider(db credentialGetter, key []byte) *Provider {
	return &Provider{db: db, key: key}
}

func (p *Provider) Resolve(credentialID uuid.UUID) (*types.CredentialSecret, error) {
	if len(p.key) == 0 {
		return nil, errors.New("AOP_ENCRYPTION_KEY is not set; cannot decrypt credentials")
	}

	cred, err := p.db.GetCredentialWithSecret(context.Background(), credentialID)
	if err != nil {
		return nil, fmt.Errorf("credential %s: %w", credentialID, err)
	}

	plaintext, err := decrypt(p.key, cred.EncryptedData)
	if err != nil {
		return nil, fmt.Errorf("credential %s: decrypt: %w", credentialID, err)
	}

	var fields map[string]string
	if err := json.Unmarshal(plaintext, &fields); err != nil {
		return nil, fmt.Errorf("credential %s: unmarshal fields: %w", credentialID, err)
	}

	return &types.CredentialSecret{Kind: cred.Kind, Fields: fields}, nil
}

// decrypt unwraps a nonce‖ciphertext blob produced by the API's AES-256-GCM encryptor.
func decrypt(key, ciphertext []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	if len(ciphertext) < gcm.NonceSize() {
		return nil, errors.New("ciphertext too short")
	}
	nonce, data := ciphertext[:gcm.NonceSize()], ciphertext[gcm.NonceSize():]
	return gcm.Open(nil, nonce, data, nil)
}
