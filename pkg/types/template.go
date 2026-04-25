package types

import (
	"time"

	"github.com/google/uuid"
)

// ---------------------------------------------------------------------------
// JobTemplate
// ---------------------------------------------------------------------------

// JobTemplate is a reusable recipe for a job. A job is always created from a
// template. Templates store the playbook path, default variables, and the
// credential to use. The controller clones the associated project to resolve
// the playbook at dispatch time.
type JobTemplate struct {
	ID           uuid.UUID  `json:"id"            db:"id"`
	Name         string     `json:"name"          db:"name"`
	Description  string     `json:"description"   db:"description"`
	ProjectID    uuid.UUID  `json:"project_id"    db:"project_id"`
	Playbook     string     `json:"playbook"      db:"playbook"` // relative path within repo
	CredentialID *uuid.UUID `json:"credential_id" db:"credential_id"`

	// DefaultExtraVars are merged with job-level ExtraVars at dispatch time.
	// Job-level values win on key collision.
	DefaultExtraVars map[string]any `json:"default_extra_vars" db:"default_extra_vars"`

	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// ---------------------------------------------------------------------------
// Schedule
// ---------------------------------------------------------------------------

// Schedule attaches a cron expression to a JobTemplate. When the expression
// fires, the controller scheduler creates a Job in pending status.
type Schedule struct {
	ID         uuid.UUID      `json:"id"          db:"id"`
	Name       string         `json:"name"        db:"name"`
	TemplateID uuid.UUID      `json:"template_id" db:"template_id"`
	CronExpr   string         `json:"cron_expr"   db:"cron_expr"` // standard 5-field cron
	Timezone   string         `json:"timezone"    db:"timezone"`  // IANA tz, e.g. "America/New_York"
	Enabled    bool           `json:"enabled"     db:"enabled"`
	ExtraVars  map[string]any `json:"extra_vars" db:"extra_vars"`

	LastRunAt *time.Time `json:"last_run_at" db:"last_run_at"`
	NextRunAt *time.Time `json:"next_run_at" db:"next_run_at"`
	CreatedAt time.Time  `json:"created_at"  db:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"  db:"updated_at"`
}
