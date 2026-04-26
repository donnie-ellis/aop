package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	APIURL            string
	RegistrationToken string
	AgentName         string
	AgentAddress      string        // full URL the controller uses to reach this agent
	Capacity          int           // max concurrent jobs
	HeartbeatInterval time.Duration
	WorkDir           string // base directory for job workspaces
	Port              string // HTTP listener port
	LogLevel          string
	LogFormat         string
	Labels            map[string]string
}

func Load() (*Config, error) {
	cfg := &Config{
		APIURL:            os.Getenv("AOP_API_URL"),
		RegistrationToken: os.Getenv("AOP_REGISTRATION_TOKEN"),
		AgentName:         os.Getenv("AOP_AGENT_NAME"),
		AgentAddress:      os.Getenv("AOP_AGENT_ADDRESS"),
		Capacity:          4,
		HeartbeatInterval: 10 * time.Second,
		WorkDir:           "/tmp/aop",
		Port:              "7000",
		LogLevel:          "info",
		LogFormat:         "json",
		Labels:            map[string]string{},
	}

	var errs []error

	if cfg.APIURL == "" {
		errs = append(errs, errors.New("AOP_API_URL is required"))
	}
	if cfg.RegistrationToken == "" {
		errs = append(errs, errors.New("AOP_REGISTRATION_TOKEN is required"))
	}
	if cfg.AgentAddress == "" {
		errs = append(errs, errors.New("AOP_AGENT_ADDRESS is required (full URL the controller uses to reach this agent)"))
	}

	if cfg.AgentName == "" {
		h, err := os.Hostname()
		if err != nil {
			cfg.AgentName = "agent"
		} else {
			cfg.AgentName = h
		}
	}

	if v := os.Getenv("AOP_CAPACITY"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n <= 0 {
			errs = append(errs, errors.New("AOP_CAPACITY must be a positive integer"))
		} else {
			cfg.Capacity = n
		}
	}

	if v := os.Getenv("AOP_HEARTBEAT_INTERVAL"); v != "" {
		d, err := time.ParseDuration(v)
		if err != nil {
			errs = append(errs, fmt.Errorf("AOP_HEARTBEAT_INTERVAL invalid: %w", err))
		} else {
			cfg.HeartbeatInterval = d
		}
	}

	if v := os.Getenv("AOP_WORK_DIR"); v != "" {
		cfg.WorkDir = v
	}
	if v := os.Getenv("AOP_PORT"); v != "" {
		cfg.Port = v
	}
	if v := os.Getenv("AOP_LOG_LEVEL"); v != "" {
		cfg.LogLevel = v
	}
	if v := os.Getenv("AOP_LOG_FORMAT"); v != "" {
		cfg.LogFormat = v
	}

	if v := os.Getenv("AOP_LABELS"); v != "" {
		for _, pair := range strings.Split(v, ",") {
			kv := strings.SplitN(strings.TrimSpace(pair), "=", 2)
			if len(kv) == 2 {
				cfg.Labels[strings.TrimSpace(kv[0])] = strings.TrimSpace(kv[1])
			}
		}
	}

	if len(errs) > 0 {
		return nil, errors.Join(errs...)
	}
	return cfg, nil
}
