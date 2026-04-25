package secrets

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"github.com/donnie-ellis/aop/pkg/types"
	"github.com/google/uuid"
)

// Store is a thin interface so secrets can call back into the DB without
// importing the full store package (avoids a circular dep if needed).
type credentialGetter interface {
	GetCredentialWithSecret(ctx context.Context, id uuid.UUID) (*types.Credential, error)
}

// Provider implements types.SecretsProvider using AES-256-GCM encryption
// with keys stored in Postgres. The encryption key comes from AOP_ENCRYPTION_KEY.
type Provider struct {
	db  credentialGetter
	key []byte // 32-byte AES-256 key
}

func NewProvider(db credentialGetter, key []byte) *Provider {
	return &Provider{db: db, key: key}
}

func (p *Provider) Resolve(credentialID uuid.UUID) (*types.CredentialSecret, error) {
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
		return nil, fmt.Errorf("credential %s: unmarshal: %w", credentialID, err)
	}

	return &types.CredentialSecret{Kind: cred.Kind, Fields: fields}, nil
}

func (p *Provider) Ping(_ context.Context) error { return nil }

// Encrypt returns nonce‖ciphertext using AES-256-GCM.
func Encrypt(key, plaintext []byte) ([]byte, error) {
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
