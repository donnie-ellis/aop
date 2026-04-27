package config

import (
	"testing"
	"time"
)

func requiredEnv(t *testing.T) {
	t.Helper()
	t.Setenv("AOP_DB_URL", "postgres://localhost/aop")
	t.Setenv("AOP_API_URL", "http://api:8080")
}

func TestLoad_MissingDBURL(t *testing.T) {
	t.Setenv("AOP_DB_URL", "")
	t.Setenv("AOP_API_URL", "http://api")
	_, err := Load()
	if err == nil {
		t.Fatal("expected error for missing AOP_DB_URL")
	}
}

func TestLoad_MissingAPIURL(t *testing.T) {
	t.Setenv("AOP_DB_URL", "postgres://localhost/aop")
	t.Setenv("AOP_API_URL", "")
	_, err := Load()
	if err == nil {
		t.Fatal("expected error for missing AOP_API_URL")
	}
}

func TestLoad_MissingBoth(t *testing.T) {
	t.Setenv("AOP_DB_URL", "")
	t.Setenv("AOP_API_URL", "")
	_, err := Load()
	if err == nil {
		t.Fatal("expected error when both required vars are missing")
	}
}

func TestLoad_Defaults(t *testing.T) {
	requiredEnv(t)
	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	checks := []struct {
		name string
		got  any
		want any
	}{
		{"LogLevel", cfg.LogLevel, "info"},
		{"LogFormat", cfg.LogFormat, "json"},
		{"WorkspaceDir", cfg.WorkspaceDir, "/tmp/aop-workspace"},
		{"ReconcileInterval", cfg.ReconcileInterval, 5 * time.Second},
		{"HeartbeatTTL", cfg.HeartbeatTTL, 30 * time.Second},
		{"DispatchTimeout", cfg.DispatchTimeout, 60 * time.Second},
		{"RunningTimeout", cfg.RunningTimeout, 3600 * time.Second},
		{"SyncInterval", cfg.SyncInterval, 5 * time.Minute},
	}
	for _, c := range checks {
		if c.got != c.want {
			t.Errorf("%s: got %v, want %v", c.name, c.got, c.want)
		}
	}
}

func TestLoad_RequiredFieldsPopulated(t *testing.T) {
	requiredEnv(t)
	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.DBURL != "postgres://localhost/aop" {
		t.Errorf("DBURL: got %q", cfg.DBURL)
	}
	if cfg.APIURL != "http://api:8080" {
		t.Errorf("APIURL: got %q", cfg.APIURL)
	}
}

func TestLoad_DurationOverrides(t *testing.T) {
	requiredEnv(t)
	t.Setenv("AOP_RECONCILE_INTERVAL", "10s")
	t.Setenv("AOP_AGENT_HEARTBEAT_TTL", "1m")
	t.Setenv("AOP_JOB_DISPATCH_TIMEOUT", "2m")
	t.Setenv("AOP_JOB_RUNNING_TIMEOUT", "30m")
	t.Setenv("AOP_SYNC_INTERVAL", "15m")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.ReconcileInterval != 10*time.Second {
		t.Errorf("ReconcileInterval: got %v", cfg.ReconcileInterval)
	}
	if cfg.HeartbeatTTL != time.Minute {
		t.Errorf("HeartbeatTTL: got %v", cfg.HeartbeatTTL)
	}
	if cfg.DispatchTimeout != 2*time.Minute {
		t.Errorf("DispatchTimeout: got %v", cfg.DispatchTimeout)
	}
	if cfg.RunningTimeout != 30*time.Minute {
		t.Errorf("RunningTimeout: got %v", cfg.RunningTimeout)
	}
	if cfg.SyncInterval != 15*time.Minute {
		t.Errorf("SyncInterval: got %v", cfg.SyncInterval)
	}
}

func TestLoad_InvalidDuration(t *testing.T) {
	requiredEnv(t)
	t.Setenv("AOP_RECONCILE_INTERVAL", "notaduration")
	_, err := Load()
	if err == nil {
		t.Fatal("expected error for invalid duration")
	}
}

func TestLoad_OptionalOverrides(t *testing.T) {
	requiredEnv(t)
	t.Setenv("AOP_LOG_LEVEL", "debug")
	t.Setenv("AOP_LOG_FORMAT", "pretty")
	t.Setenv("AOP_WORKSPACE_DIR", "/var/aop-workspace")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.LogLevel != "debug" {
		t.Errorf("LogLevel: got %q", cfg.LogLevel)
	}
	if cfg.LogFormat != "pretty" {
		t.Errorf("LogFormat: got %q", cfg.LogFormat)
	}
	if cfg.WorkspaceDir != "/var/aop-workspace" {
		t.Errorf("WorkspaceDir: got %q", cfg.WorkspaceDir)
	}
}
