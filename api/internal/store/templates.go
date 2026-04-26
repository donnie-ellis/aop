package store

import (
	"context"

	"github.com/donnie-ellis/aop/pkg/types"
	"github.com/google/uuid"
)

func (s *Store) CreateJobTemplate(ctx context.Context, name, description string, projectID uuid.UUID, playbook string, credentialID *uuid.UUID, defaultExtraVars map[string]any) (*types.JobTemplate, error) {
	if defaultExtraVars == nil {
		defaultExtraVars = map[string]any{}
	}
	var t types.JobTemplate
	err := s.pool.QueryRow(ctx,
		`INSERT INTO job_templates (name, description, project_id, playbook, credential_id, default_extra_vars)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 RETURNING id, name, description, project_id, playbook, credential_id, default_extra_vars, created_at, updated_at`,
		name, description, projectID, playbook, credentialID, defaultExtraVars,
	).Scan(&t.ID, &t.Name, &t.Description, &t.ProjectID, &t.Playbook, &t.CredentialID, &t.DefaultExtraVars, &t.CreatedAt, &t.UpdatedAt)
	return &t, err
}

func (s *Store) GetJobTemplate(ctx context.Context, id uuid.UUID) (*types.JobTemplate, error) {
	var t types.JobTemplate
	err := s.pool.QueryRow(ctx,
		`SELECT id, name, description, project_id, playbook, credential_id, default_extra_vars, created_at, updated_at
		 FROM job_templates WHERE id = $1`,
		id,
	).Scan(&t.ID, &t.Name, &t.Description, &t.ProjectID, &t.Playbook, &t.CredentialID, &t.DefaultExtraVars, &t.CreatedAt, &t.UpdatedAt)
	if isNotFound(err) {
		return nil, ErrNotFound
	}
	return &t, err
}

func (s *Store) ListJobTemplates(ctx context.Context) ([]types.JobTemplate, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id, name, description, project_id, playbook, credential_id, default_extra_vars, created_at, updated_at
		 FROM job_templates ORDER BY name`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var templates []types.JobTemplate
	for rows.Next() {
		var t types.JobTemplate
		if err := rows.Scan(&t.ID, &t.Name, &t.Description, &t.ProjectID, &t.Playbook, &t.CredentialID, &t.DefaultExtraVars, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, err
		}
		templates = append(templates, t)
	}
	return templates, rows.Err()
}

func (s *Store) UpdateJobTemplate(ctx context.Context, id uuid.UUID, name, description, playbook string, credentialID *uuid.UUID, defaultExtraVars map[string]any) (*types.JobTemplate, error) {
	var t types.JobTemplate
	err := s.pool.QueryRow(ctx,
		`UPDATE job_templates
		 SET name=$1, description=$2, playbook=$3, credential_id=$4, default_extra_vars=$5
		 WHERE id=$6
		 RETURNING id, name, description, project_id, playbook, credential_id, default_extra_vars, created_at, updated_at`,
		name, description, playbook, credentialID, defaultExtraVars, id,
	).Scan(&t.ID, &t.Name, &t.Description, &t.ProjectID, &t.Playbook, &t.CredentialID, &t.DefaultExtraVars, &t.CreatedAt, &t.UpdatedAt)
	if isNotFound(err) {
		return nil, ErrNotFound
	}
	return &t, err
}

func (s *Store) DeleteJobTemplate(ctx context.Context, id uuid.UUID) error {
	tag, err := s.pool.Exec(ctx, `DELETE FROM job_templates WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
