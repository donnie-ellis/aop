package store

import (
	"context"

	"github.com/donnie-ellis/aop/pkg/types"
	"github.com/google/uuid"
)

func (s *Store) CreateCredential(ctx context.Context, name, description string, kind types.CredentialKind, encryptedData []byte) (*types.Credential, error) {
	var c types.Credential
	err := s.pool.QueryRow(ctx,
		`INSERT INTO credentials (name, description, kind, encrypted_data)
		 VALUES ($1, $2, $3, $4)
		 RETURNING id, name, description, kind, created_at, updated_at`,
		name, description, kind, encryptedData,
	).Scan(&c.ID, &c.Name, &c.Description, &c.Kind, &c.CreatedAt, &c.UpdatedAt)
	return &c, err
}

func (s *Store) GetCredential(ctx context.Context, id uuid.UUID) (*types.Credential, error) {
	var c types.Credential
	err := s.pool.QueryRow(ctx,
		`SELECT id, name, description, kind, created_at, updated_at
		 FROM credentials WHERE id = $1`,
		id,
	).Scan(&c.ID, &c.Name, &c.Description, &c.Kind, &c.CreatedAt, &c.UpdatedAt)
	if isNotFound(err) {
		return nil, ErrNotFound
	}
	return &c, err
}

// GetCredentialWithSecret returns the credential including encrypted_data for decryption.
// Only called internally (controller dispatch, not API responses).
func (s *Store) GetCredentialWithSecret(ctx context.Context, id uuid.UUID) (*types.Credential, error) {
	var c types.Credential
	err := s.pool.QueryRow(ctx,
		`SELECT id, name, description, kind, encrypted_data, created_at, updated_at
		 FROM credentials WHERE id = $1`,
		id,
	).Scan(&c.ID, &c.Name, &c.Description, &c.Kind, &c.EncryptedData, &c.CreatedAt, &c.UpdatedAt)
	if isNotFound(err) {
		return nil, ErrNotFound
	}
	return &c, err
}

func (s *Store) ListCredentials(ctx context.Context) ([]types.Credential, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id, name, description, kind, created_at, updated_at
		 FROM credentials ORDER BY name`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var creds []types.Credential
	for rows.Next() {
		var c types.Credential
		if err := rows.Scan(&c.ID, &c.Name, &c.Description, &c.Kind, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, err
		}
		creds = append(creds, c)
	}
	return creds, rows.Err()
}

func (s *Store) UpdateCredential(ctx context.Context, id uuid.UUID, name, description string, kind types.CredentialKind, encryptedData []byte) (*types.Credential, error) {
	var c types.Credential
	err := s.pool.QueryRow(ctx,
		`UPDATE credentials SET name=$1, description=$2, kind=$3, encrypted_data=$4
		 WHERE id=$5
		 RETURNING id, name, description, kind, created_at, updated_at`,
		name, description, kind, encryptedData, id,
	).Scan(&c.ID, &c.Name, &c.Description, &c.Kind, &c.CreatedAt, &c.UpdatedAt)
	if isNotFound(err) {
		return nil, ErrNotFound
	}
	return &c, err
}

func (s *Store) DeleteCredential(ctx context.Context, id uuid.UUID) error {
	tag, err := s.pool.Exec(ctx, `DELETE FROM credentials WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
