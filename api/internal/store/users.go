package store

import (
	"context"

	"github.com/donnie-ellis/aop/pkg/types"
	"github.com/google/uuid"
)

func (s *Store) CreateUser(ctx context.Context, email, passwordHash string) (*types.User, error) {
	var u types.User
	err := s.pool.QueryRow(ctx,
		`INSERT INTO users (email, password_hash)
		 VALUES ($1, $2)
		 RETURNING id, email, password_hash, created_at, updated_at`,
		email, passwordHash,
	).Scan(&u.ID, &u.Email, &u.PasswordHash, &u.CreatedAt, &u.UpdatedAt)
	return &u, err
}

func (s *Store) GetUserByEmail(ctx context.Context, email string) (*types.User, error) {
	var u types.User
	err := s.pool.QueryRow(ctx,
		`SELECT id, email, password_hash, created_at, updated_at
		 FROM users WHERE email = $1`,
		email,
	).Scan(&u.ID, &u.Email, &u.PasswordHash, &u.CreatedAt, &u.UpdatedAt)
	if isNotFound(err) {
		return nil, ErrNotFound
	}
	return &u, err
}

func (s *Store) GetUserByID(ctx context.Context, id uuid.UUID) (*types.User, error) {
	var u types.User
	err := s.pool.QueryRow(ctx,
		`SELECT id, email, password_hash, created_at, updated_at
		 FROM users WHERE id = $1`,
		id,
	).Scan(&u.ID, &u.Email, &u.PasswordHash, &u.CreatedAt, &u.UpdatedAt)
	if isNotFound(err) {
		return nil, ErrNotFound
	}
	return &u, err
}

func (s *Store) CreateAPIToken(ctx context.Context, userID uuid.UUID, name, tokenHash string) (*types.APIToken, error) {
	var t types.APIToken
	err := s.pool.QueryRow(ctx,
		`INSERT INTO api_tokens (user_id, name, token_hash)
		 VALUES ($1, $2, $3)
		 RETURNING id, user_id, name, token_hash, created_at, last_used_at, expires_at`,
		userID, name, tokenHash,
	).Scan(&t.ID, &t.UserID, &t.Name, &t.TokenHash, &t.CreatedAt, &t.LastUsedAt, &t.ExpiresAt)
	return &t, err
}

func (s *Store) GetAPITokenByHash(ctx context.Context, tokenHash string) (*types.APIToken, error) {
	var t types.APIToken
	err := s.pool.QueryRow(ctx,
		`SELECT id, user_id, name, token_hash, created_at, last_used_at, expires_at
		 FROM api_tokens WHERE token_hash = $1`,
		tokenHash,
	).Scan(&t.ID, &t.UserID, &t.Name, &t.TokenHash, &t.CreatedAt, &t.LastUsedAt, &t.ExpiresAt)
	if isNotFound(err) {
		return nil, ErrNotFound
	}
	return &t, err
}

func (s *Store) ListAPITokens(ctx context.Context, userID uuid.UUID) ([]types.APIToken, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id, user_id, name, token_hash, created_at, last_used_at, expires_at
		 FROM api_tokens WHERE user_id = $1 ORDER BY created_at DESC`,
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tokens []types.APIToken
	for rows.Next() {
		var t types.APIToken
		if err := rows.Scan(&t.ID, &t.UserID, &t.Name, &t.TokenHash, &t.CreatedAt, &t.LastUsedAt, &t.ExpiresAt); err != nil {
			return nil, err
		}
		tokens = append(tokens, t)
	}
	return tokens, rows.Err()
}

func (s *Store) DeleteAPIToken(ctx context.Context, id, userID uuid.UUID) error {
	tag, err := s.pool.Exec(ctx,
		`DELETE FROM api_tokens WHERE id = $1 AND user_id = $2`,
		id, userID,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) TouchAPIToken(ctx context.Context, id uuid.UUID) error {
	_, err := s.pool.Exec(ctx,
		`UPDATE api_tokens SET last_used_at = now() WHERE id = $1`,
		id,
	)
	return err
}
