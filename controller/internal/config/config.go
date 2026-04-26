package config

import (
	"errors"
	"fmt"
	"os"
	"time"
)

// Config holds all runtime configuration for the controller process.
type Config struct {
	DBURL   string
	APIURL  string // base URL of the API server — embedded in job payloads as CallbackURL
	LogLevel  string
	LogFormat string

	ReconcileInterval time.Duration // how often to poll for pending jobs
	HeartbeatTTL      time.Duration // agent considered offline after this
	DispatchTimeout   time.Duration // stuck dispatched job → failed after this
	RunningTimeout    time.Duration // stuck running job → failed after this

	WorkspaceDir string // base directory for git clones
	SyncInterval time.Duration // how often to sync all project inventories
}

func Load() (*Config, error) {
	cfg := &Config{
		DBURL:             os.Getenv("AOP_DB_URL"),
		APIURL:            os.Getenv("AOP_API_URL"),
		LogLevel:          "info",
		LogFormat:         "json",
		ReconcileInterval: 5 * time.Second,
		HeartbeatTTL:      30 * time.Second,
		DispatchTimeout:   60 * time.Second,
		RunningTimeout:    3600 * time.Second,
		WorkspaceDir:      "/tmp/aop-workspace",
		SyncInterval:      5 * time.Minute,
	}

	var errs []error
	if cfg.DBURL == "" {
		errs = append(errs, errors.New("AOP_DB_URL is required"))
	}
	if cfg.APIURL == "" {
		errs = append(errs, errors.New("AOP_API_URL is required"))
	}

	if v := os.Getenv("AOP_LOG_LEVEL"); v != "" {
		cfg.LogLevel = v
	}
	if v := os.Getenv("AOP_LOG_FORMAT"); v != "" {
		cfg.LogFormat = v
	}
	if v := os.Getenv("AOP_WORKSPACE_DIR"); v != "" {
		cfg.WorkspaceDir = v
	}

	parseDuration := func(env string, dest *time.Duration) {
		if v := os.Getenv(env); v != "" {
			d, err := time.ParseDuration(v)
			if err != nil {
				errs = append(errs, fmt.Errorf("%s invalid: %w", env, err))
			} else {
				*dest = d
			}
		}
	}
	parseDuration("AOP_RECONCILE_INTERVAL", &cfg.ReconcileInterval)
	parseDuration("AOP_AGENT_HEARTBEAT_TTL", &cfg.HeartbeatTTL)
	parseDuration("AOP_JOB_DISPATCH_TIMEOUT", &cfg.DispatchTimeout)
	parseDuration("AOP_JOB_RUNNING_TIMEOUT", &cfg.RunningTimeout)
	parseDuration("AOP_SYNC_INTERVAL", &cfg.SyncInterval)

	if len(errs) > 0 {
		return nil, errors.Join(errs...)
	}
	return cfg, nil
}
