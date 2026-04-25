package types

import (
	"time"

	"github.com/google/uuid"
)

// User is an AOP operator who can log in, manage resources, and approve workflow gates.
type User struct {
	ID           uuid.UUID `json:"id"         db:"id"`
	Email        string    `json:"email"      db:"email"`
	PasswordHash string    `json:"-"          db:"password_hash"` // never serialized
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time `json:"updated_at" db:"updated_at"`
}

// APIToken is a long-lived bearer token for service accounts (webhooks, CLI, CI).
// The raw token is shown once on creation and never stored — only the hash is persisted.
type APIToken struct {
	ID         uuid.UUID  `json:"id"          db:"id"`
	UserID     uuid.UUID  `json:"user_id"     db:"user_id"`
	Name       string     `json:"name"        db:"name"`
	TokenHash  string     `json:"-"           db:"token_hash"` // never serialized
	CreatedAt  time.Time  `json:"created_at"  db:"created_at"`
	LastUsedAt *time.Time `json:"last_used_at" db:"last_used_at"`
	ExpiresAt  *time.Time `json:"expires_at"  db:"expires_at"`
}
