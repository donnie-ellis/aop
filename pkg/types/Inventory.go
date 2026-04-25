package types

import (
	"time"

	"github.com/google/uuid"
)

// ---------------------------------------------------------------------------
// Inventory
// ---------------------------------------------------------------------------

// InventoryHost represents a single managed host as parsed from a project's
// inventory file and stored in Postgres. This is the canonical in-database
// representation; the agent receives a pre-flattened slice in JobPayload.
type InventoryHost struct {
	ID        uuid.UUID `json:"id"         db:"id"`
	ProjectID uuid.UUID `json:"project_id" db:"project_id"`

	// Hostname is the value Ansible will connect to — either a DNS name or IP.
	Hostname string `json:"hostname" db:"hostname"`

	// Groups is the list of inventory groups this host belongs to.
	Groups []string `json:"groups" db:"groups"`

	// Vars holds host variables parsed from the inventory source.
	// These are merged with group vars and extra vars at dispatch time.
	Vars map[string]any `json:"vars" db:"vars"`

	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// InventorySyncStatus describes the result of the most recent Git sync for
// a project's inventory.
type InventorySyncStatus string

const (
	InventorySyncStatusOK      InventorySyncStatus = "ok"
	InventorySyncStatusFailed  InventorySyncStatus = "failed"
	InventorySyncStatusPending InventorySyncStatus = "pending" // initial state before first sync
)

// Project holds metadata for a source-controlled Ansible project.
// The controller's Git Sync sub-component manages cloning and inventory
// ingestion for each project.
type Project struct {
	ID           uuid.UUID  `json:"id"            db:"id"`
	Name         string     `json:"name"          db:"name"`
	RepoURL      string     `json:"repo_url"      db:"repo_url"`
	Branch       string     `json:"branch"        db:"branch"`
	InventoryPath string    `json:"inventory_path" db:"inventory_path"` // path within repo
	CredentialID *uuid.UUID `json:"credential_id" db:"credential_id"`    // for private repos

	SyncStatus   InventorySyncStatus `json:"sync_status"    db:"sync_status"`
	LastSyncedAt *time.Time          `json:"last_synced_at" db:"last_synced_at"`
	SyncError    *string             `json:"sync_error"     db:"sync_error"` // last error message, if any

	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}
