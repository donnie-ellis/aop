package store

import (
	"context"

	"github.com/donnie-ellis/aop/pkg/types"
	"github.com/google/uuid"
)

func (s *Store) CreateProject(ctx context.Context, name, repoURL, branch, inventoryPath string, credentialID *uuid.UUID) (*types.Project, error) {
	var p types.Project
	err := s.pool.QueryRow(ctx,
		`INSERT INTO projects (name, repo_url, branch, inventory_path, credential_id)
		 VALUES ($1, $2, $3, $4, $5)
		 RETURNING id, name, repo_url, branch, inventory_path, credential_id, sync_status, last_synced_at, sync_error, created_at, updated_at`,
		name, repoURL, branch, inventoryPath, credentialID,
	).Scan(&p.ID, &p.Name, &p.RepoURL, &p.Branch, &p.InventoryPath, &p.CredentialID, &p.SyncStatus, &p.LastSyncedAt, &p.SyncError, &p.CreatedAt, &p.UpdatedAt)
	return &p, err
}

func (s *Store) GetProject(ctx context.Context, id uuid.UUID) (*types.Project, error) {
	var p types.Project
	err := s.pool.QueryRow(ctx,
		`SELECT id, name, repo_url, branch, inventory_path, credential_id, sync_status, last_synced_at, sync_error, created_at, updated_at
		 FROM projects WHERE id = $1`,
		id,
	).Scan(&p.ID, &p.Name, &p.RepoURL, &p.Branch, &p.InventoryPath, &p.CredentialID, &p.SyncStatus, &p.LastSyncedAt, &p.SyncError, &p.CreatedAt, &p.UpdatedAt)
	if isNotFound(err) {
		return nil, ErrNotFound
	}
	return &p, err
}

func (s *Store) ListProjects(ctx context.Context) ([]types.Project, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id, name, repo_url, branch, inventory_path, credential_id, sync_status, last_synced_at, sync_error, created_at, updated_at
		 FROM projects ORDER BY name`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var projects []types.Project
	for rows.Next() {
		var p types.Project
		if err := rows.Scan(&p.ID, &p.Name, &p.RepoURL, &p.Branch, &p.InventoryPath, &p.CredentialID, &p.SyncStatus, &p.LastSyncedAt, &p.SyncError, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, err
		}
		projects = append(projects, p)
	}
	return projects, rows.Err()
}

func (s *Store) UpdateProject(ctx context.Context, id uuid.UUID, name, repoURL, branch, inventoryPath string, credentialID *uuid.UUID) (*types.Project, error) {
	var p types.Project
	err := s.pool.QueryRow(ctx,
		`UPDATE projects
		 SET name=$1, repo_url=$2, branch=$3, inventory_path=$4, credential_id=$5
		 WHERE id=$6
		 RETURNING id, name, repo_url, branch, inventory_path, credential_id, sync_status, last_synced_at, sync_error, created_at, updated_at`,
		name, repoURL, branch, inventoryPath, credentialID, id,
	).Scan(&p.ID, &p.Name, &p.RepoURL, &p.Branch, &p.InventoryPath, &p.CredentialID, &p.SyncStatus, &p.LastSyncedAt, &p.SyncError, &p.CreatedAt, &p.UpdatedAt)
	if isNotFound(err) {
		return nil, ErrNotFound
	}
	return &p, err
}

func (s *Store) DeleteProject(ctx context.Context, id uuid.UUID) error {
	tag, err := s.pool.Exec(ctx, `DELETE FROM projects WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) UpdateProjectSyncStatus(ctx context.Context, id uuid.UUID, status types.InventorySyncStatus, syncErr *string) error {
	_, err := s.pool.Exec(ctx,
		`UPDATE projects SET sync_status=$1, sync_error=$2, last_synced_at=CASE WHEN $1='ok' THEN now() ELSE last_synced_at END WHERE id=$3`,
		status, syncErr, id,
	)
	return err
}

func (s *Store) ListInventoryHosts(ctx context.Context, projectID uuid.UUID) ([]types.InventoryHost, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id, project_id, hostname, groups, vars, created_at, updated_at
		 FROM inventory_hosts WHERE project_id = $1 ORDER BY hostname`,
		projectID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var hosts []types.InventoryHost
	for rows.Next() {
		var h types.InventoryHost
		if err := rows.Scan(&h.ID, &h.ProjectID, &h.Hostname, &h.Groups, &h.Vars, &h.CreatedAt, &h.UpdatedAt); err != nil {
			return nil, err
		}
		hosts = append(hosts, h)
	}
	return hosts, rows.Err()
}
