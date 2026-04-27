package config

import (
	"testing"
	"time"
)

func setEnv(t *testing.T, pairs map[string]string) {
	t.Helper()
	for k, v := range pairs {
		t.Setenv(k, v)
	}
}

func requiredEnv(t *testing.T) {
	t.Helper()
	setEnv(t, map[string]string{
		"AOP_API_URL":            "http://api:8080",
		"AOP_REGISTRATION_TOKEN": "secret",
		"AOP_AGENT_ADDRESS":      "http://agent:7000",
	})
}

func TestLoad_RequiredVars(t *testing.T) {
	cases := []struct {
		name    string
		env     map[string]string
		wantErr string
	}{
		{
			name: "missing all",
			env: map[string]string{
				"AOP_API_URL":            "",
				"AOP_REGISTRATION_TOKEN": "",
				"AOP_AGENT_ADDRESS":      "",
			},
			wantErr: "AOP_API_URL is required",
		},
		{
			name: "missing token and address",
			env: map[string]string{
				"AOP_API_URL":            "http://api",
				"AOP_REGISTRATION_TOKEN": "",
				"AOP_AGENT_ADDRESS":      "",
			},
			wantErr: "AOP_REGISTRATION_TOKEN is required",
		},
		{
			name: "missing address",
			env: map[string]string{
				"AOP_API_URL":            "http://api",
				"AOP_REGISTRATION_TOKEN": "tok",
				"AOP_AGENT_ADDRESS":      "",
			},
			wantErr: "AOP_AGENT_ADDRESS is required",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			setEnv(t, tc.env)
			_, err := Load()
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if got := err.Error(); len(got) == 0 {
				t.Error("error message is empty")
			}
		})
	}
}

func TestLoad_Defaults(t *testing.T) {
	requiredEnv(t)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Capacity != 4 {
		t.Errorf("capacity: got %d, want 4", cfg.Capacity)
	}
	if cfg.HeartbeatInterval != 10*time.Second {
		t.Errorf("heartbeat interval: got %v, want 10s", cfg.HeartbeatInterval)
	}
	if cfg.WorkDir != "/tmp/aop" {
		t.Errorf("work dir: got %q, want /tmp/aop", cfg.WorkDir)
	}
	if cfg.Port != "7000" {
		t.Errorf("port: got %q, want 7000", cfg.Port)
	}
	if cfg.LogLevel != "info" {
		t.Errorf("log level: got %q, want info", cfg.LogLevel)
	}
	if cfg.LogFormat != "json" {
		t.Errorf("log format: got %q, want json", cfg.LogFormat)
	}
}

func TestLoad_RequiredFieldsPopulated(t *testing.T) {
	requiredEnv(t)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.APIURL != "http://api:8080" {
		t.Errorf("api url: got %q", cfg.APIURL)
	}
	if cfg.RegistrationToken != "secret" {
		t.Errorf("registration token: got %q", cfg.RegistrationToken)
	}
	if cfg.AgentAddress != "http://agent:7000" {
		t.Errorf("agent address: got %q", cfg.AgentAddress)
	}
}

func TestLoad_Capacity(t *testing.T) {
	requiredEnv(t)

	t.Run("valid", func(t *testing.T) {
		t.Setenv("AOP_CAPACITY", "8")
		cfg, err := Load()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cfg.Capacity != 8 {
			t.Errorf("got %d, want 8", cfg.Capacity)
		}
	})

	t.Run("invalid string", func(t *testing.T) {
		t.Setenv("AOP_CAPACITY", "notanumber")
		_, err := Load()
		if err == nil {
			t.Fatal("expected error for invalid capacity")
		}
	})

	t.Run("zero", func(t *testing.T) {
		t.Setenv("AOP_CAPACITY", "0")
		_, err := Load()
		if err == nil {
			t.Fatal("expected error for zero capacity")
		}
	})

	t.Run("negative", func(t *testing.T) {
		t.Setenv("AOP_CAPACITY", "-1")
		_, err := Load()
		if err == nil {
			t.Fatal("expected error for negative capacity")
		}
	})
}

func TestLoad_HeartbeatInterval(t *testing.T) {
	requiredEnv(t)

	t.Run("valid", func(t *testing.T) {
		t.Setenv("AOP_HEARTBEAT_INTERVAL", "30s")
		cfg, err := Load()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cfg.HeartbeatInterval != 30*time.Second {
			t.Errorf("got %v, want 30s", cfg.HeartbeatInterval)
		}
	})

	t.Run("invalid", func(t *testing.T) {
		t.Setenv("AOP_HEARTBEAT_INTERVAL", "notaduration")
		_, err := Load()
		if err == nil {
			t.Fatal("expected error for invalid duration")
		}
	})
}

func TestLoad_Labels(t *testing.T) {
	requiredEnv(t)

	t.Setenv("AOP_LABELS", "env=prod,region=us-east-1, tier=backend")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := map[string]string{
		"env":    "prod",
		"region": "us-east-1",
		"tier":   "backend",
	}
	for k, v := range want {
		if got := cfg.Labels[k]; got != v {
			t.Errorf("label %q: got %q, want %q", k, got, v)
		}
	}
}

func TestLoad_AgentNameDefaultsToHostname(t *testing.T) {
	requiredEnv(t)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.AgentName == "" {
		t.Error("agent name should not be empty")
	}
}

func TestLoad_AgentNameOverride(t *testing.T) {
	requiredEnv(t)
	t.Setenv("AOP_AGENT_NAME", "my-agent")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.AgentName != "my-agent" {
		t.Errorf("got %q, want my-agent", cfg.AgentName)
	}
}

func TestLoad_OverrideOptionals(t *testing.T) {
	requiredEnv(t)
	setEnv(t, map[string]string{
		"AOP_WORK_DIR":   "/var/aop",
		"AOP_PORT":       "9000",
		"AOP_LOG_LEVEL":  "debug",
		"AOP_LOG_FORMAT": "console",
	})

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.WorkDir != "/var/aop" {
		t.Errorf("work dir: got %q", cfg.WorkDir)
	}
	if cfg.Port != "9000" {
		t.Errorf("port: got %q", cfg.Port)
	}
	if cfg.LogLevel != "debug" {
		t.Errorf("log level: got %q", cfg.LogLevel)
	}
	if cfg.LogFormat != "console" {
		t.Errorf("log format: got %q", cfg.LogFormat)
	}
}
