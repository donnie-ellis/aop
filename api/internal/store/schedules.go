package store

import (
	"context"

	"github.com/donnie-ellis/aop/pkg/types"
	"github.com/google/uuid"
)

func (s *Store) CreateSchedule(ctx context.Context, name string, templateID uuid.UUID, cronExpr, timezone string, enabled bool, extraVars map[string]any) (*types.Schedule, error) {
	if extraVars == nil {
		extraVars = map[string]any{}
	}
	var sc types.Schedule
	err := s.pool.QueryRow(ctx,
		`INSERT INTO schedules (name, template_id, cron_expr, timezone, enabled, extra_vars)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 RETURNING id, name, template_id, cron_expr, timezone, enabled, extra_vars, last_run_at, next_run_at, created_at, updated_at`,
		name, templateID, cronExpr, timezone, enabled, extraVars,
	).Scan(&sc.ID, &sc.Name, &sc.TemplateID, &sc.CronExpr, &sc.Timezone, &sc.Enabled, &sc.ExtraVars, &sc.LastRunAt, &sc.NextRunAt, &sc.CreatedAt, &sc.UpdatedAt)
	return &sc, err
}

func (s *Store) GetSchedule(ctx context.Context, id uuid.UUID) (*types.Schedule, error) {
	var sc types.Schedule
	err := s.pool.QueryRow(ctx,
		`SELECT id, name, template_id, cron_expr, timezone, enabled, extra_vars, last_run_at, next_run_at, created_at, updated_at
		 FROM schedules WHERE id = $1`,
		id,
	).Scan(&sc.ID, &sc.Name, &sc.TemplateID, &sc.CronExpr, &sc.Timezone, &sc.Enabled, &sc.ExtraVars, &sc.LastRunAt, &sc.NextRunAt, &sc.CreatedAt, &sc.UpdatedAt)
	if isNotFound(err) {
		return nil, ErrNotFound
	}
	return &sc, err
}

func (s *Store) ListSchedules(ctx context.Context) ([]types.Schedule, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id, name, template_id, cron_expr, timezone, enabled, extra_vars, last_run_at, next_run_at, created_at, updated_at
		 FROM schedules ORDER BY name`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var schedules []types.Schedule
	for rows.Next() {
		var sc types.Schedule
		if err := rows.Scan(&sc.ID, &sc.Name, &sc.TemplateID, &sc.CronExpr, &sc.Timezone, &sc.Enabled, &sc.ExtraVars, &sc.LastRunAt, &sc.NextRunAt, &sc.CreatedAt, &sc.UpdatedAt); err != nil {
			return nil, err
		}
		schedules = append(schedules, sc)
	}
	return schedules, rows.Err()
}

func (s *Store) UpdateSchedule(ctx context.Context, id uuid.UUID, name, cronExpr, timezone string, enabled bool, extraVars map[string]any) (*types.Schedule, error) {
	if extraVars == nil {
		extraVars = map[string]any{}
	}
	var sc types.Schedule
	err := s.pool.QueryRow(ctx,
		`UPDATE schedules
		 SET name=$1, cron_expr=$2, timezone=$3, enabled=$4, extra_vars=$5
		 WHERE id=$6
		 RETURNING id, name, template_id, cron_expr, timezone, enabled, extra_vars, last_run_at, next_run_at, created_at, updated_at`,
		name, cronExpr, timezone, enabled, extraVars, id,
	).Scan(&sc.ID, &sc.Name, &sc.TemplateID, &sc.CronExpr, &sc.Timezone, &sc.Enabled, &sc.ExtraVars, &sc.LastRunAt, &sc.NextRunAt, &sc.CreatedAt, &sc.UpdatedAt)
	if isNotFound(err) {
		return nil, ErrNotFound
	}
	return &sc, err
}

func (s *Store) DeleteSchedule(ctx context.Context, id uuid.UUID) error {
	tag, err := s.pool.Exec(ctx, `DELETE FROM schedules WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
