package types

import (
	"time"

	"github.com/google/uuid"
)

// ---------------------------------------------------------------------------
// Credential
// ---------------------------------------------------------------------------

// CredentialKind describes the authentication mechanism a credential provides.
type CredentialKind string

const (
	// CredentialKindSSHKey is a PEM-encoded private key used for SSH Git auth
	// and for Ansible connections over SSH.
	CredentialKindSSHKey CredentialKind = "ssh_key"

	// CredentialKindHTTPSToken is a personal access token or app password used
	// for HTTPS Git clones.
	CredentialKindHTTPSToken CredentialKind = "https_token"

	// CredentialKindVaultPassword is an Ansible Vault decryption password.
	CredentialKindVaultPassword CredentialKind = "vault_password"

	// CredentialKindMachineUser bundles a username + SSH key for Ansible
	// machine connections where a non-default remote user is required.
	CredentialKindMachineUser CredentialKind = "machine_user"
)

// Credential is the metadata record for a stored secret. The actual secret
// material is never returned in API responses and is stored encrypted at rest.
// Secret retrieval goes through SecretsProvider, not direct DB reads.
type Credential struct {
	ID          uuid.UUID      `json:"id"          db:"id"`
	Name        string         `json:"name"        db:"name"`
	Kind        CredentialKind `json:"kind"        db:"kind"`
	Description string         `json:"description" db:"description"`

	// EncryptedData holds the ciphertext blob. The shape of the plaintext
	// after decryption depends on Kind:
	//   ssh_key         → {"private_key": "<PEM>", "passphrase": "<opt>"}
	//   https_token     → {"token": "<token>"}
	//   vault_password  → {"password": "<password>"}
	//   machine_user    → {"username": "<user>", "private_key": "<PEM>"}
	// This field is never serialised in JSON responses (db-only).
	EncryptedData []byte `json:"-" db:"encrypted_data"`

	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// CredentialSecret is the decrypted secret payload returned by SecretsProvider.
// The map keys match the shape documented on Credential.EncryptedData above.
// This type is intentionally kept separate from Credential so callers are never
// accidentally serialising secrets.
type CredentialSecret struct {
	Kind   CredentialKind
	Fields map[string]string
}

// ---------------------------------------------------------------------------
// SecretsProvider interface
// ---------------------------------------------------------------------------

// SecretsProvider resolves a Credential's encrypted data into usable secret
// fields. Implementations may back this with local KMS, HashiCorp Vault, AWS
// Secrets Manager, etc. The controller and agent both consume this interface;
// neither has direct access to raw encrypted bytes.
type SecretsProvider interface {
	// Resolve decrypts and returns the secret fields for the given credential ID.
	// Returns an error if the credential does not exist or decryption fails.
	Resolve(credentialID uuid.UUID) (*CredentialSecret, error)
}
