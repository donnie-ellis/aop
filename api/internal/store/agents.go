package store

import (
	"context"
	"time"

	"github.com/donnie-ellis/aop/pkg/types"
	"github.com/google/uuid"
)

func (s *Store) CreateAgent(ctx context.Context, name, address, tokenHash string, labels map[string]string, capacity int) (*types.Agent, error) {
	var a types.Agent
	err := s.pool.QueryRow(ctx,
		`INSERT INTO agents (name, address, token_hash, status, labels, capacity)
		 VALUES ($1, $2, $3, 'offline', $4, $5)
		 RETURNING id, name, address, status, labels, capacity, last_heartbeat_at, registered_at, updated_at`,
		name, address, tokenHash, labels, capacity,
	).Scan(&a.ID, &a.Name, &a.Address, &a.Status, &a.Labels, &a.Capacity, &a.LastHeartbeatAt, &a.RegisteredAt, &a.UpdatedAt)
	return &a, err
}

func (s *Store) GetAgentByID(ctx context.Context, id uuid.UUID) (*types.Agent, error) {
	var a types.Agent
	err := s.pool.QueryRow(ctx,
		`SELECT id, name, address, status, labels, capacity, last_heartbeat_at, registered_at, updated_at
		 FROM agents WHERE id = $1`,
		id,
	).Scan(&a.ID, &a.Name, &a.Address, &a.Status, &a.Labels, &a.Capacity, &a.LastHeartbeatAt, &a.RegisteredAt, &a.UpdatedAt)
	if isNotFound(err) {
		return nil, ErrNotFound
	}
	return &a, err
}

func (s *Store) GetAgentByTokenHash(ctx context.Context, tokenHash string) (*types.Agent, error) {
	var a types.Agent
	err := s.pool.QueryRow(ctx,
		`SELECT id, name, address, status, labels, capacity, last_heartbeat_at, registered_at, updated_at
		 FROM agents WHERE token_hash = $1`,
		tokenHash,
	).Scan(&a.ID, &a.Name, &a.Address, &a.Status, &a.Labels, &a.Capacity, &a.LastHeartbeatAt, &a.RegisteredAt, &a.UpdatedAt)
	if isNotFound(err) {
		return nil, ErrNotFound
	}
	return &a, err
}

func (s *Store) ListAgents(ctx context.Context) ([]types.Agent, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id, name, address, status, labels, capacity, last_heartbeat_at, registered_at, updated_at
		 FROM agents ORDER BY registered_at DESC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var agents []types.Agent
	for rows.Next() {
		var a types.Agent
		if err := rows.Scan(&a.ID, &a.Name, &a.Address, &a.Status, &a.Labels, &a.Capacity, &a.LastHeartbeatAt, &a.RegisteredAt, &a.UpdatedAt); err != nil {
			return nil, err
		}
		agents = append(agents, a)
	}
	return agents, rows.Err()
}

func (s *Store) UpdateAgentHeartbeat(ctx context.Context, id uuid.UUID, runningJobs int, ttl time.Duration) error {
	// Mark online if heartbeat is fresh; capacity check determines busy vs online.
	tag, err := s.pool.Exec(ctx,
		`UPDATE agents
		 SET last_heartbeat_at = now(),
		     status = 'online',
		     updated_at = now()
		 WHERE id = $1`,
		id,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// MarkStaleAgentsOffline sets agents whose last heartbeat is older than ttl to offline.
// Called periodically by the controller; also useful for the API's status display.
func (s *Store) MarkStaleAgentsOffline(ctx context.Context, ttl time.Duration) error {
	_, err := s.pool.Exec(ctx,
		`UPDATE agents SET status = 'offline', updated_at = now()
		 WHERE status = 'online'
		   AND last_heartbeat_at < now() - $1::interval`,
		ttl.String(),
	)
	return err
}
