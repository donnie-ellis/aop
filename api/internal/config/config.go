package config

import (
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"time"
)

type Config struct {
	DBUrl         string
	EncryptionKey []byte        // 32 bytes decoded from AOP_ENCRYPTION_KEY hex
	JWTSecret     []byte        // AOP_JWT_SECRET
	JWTExpiry     time.Duration // AOP_JWT_EXPIRY, default 8h
	Port          string        // AOP_PORT, default 8080
	LogLevel      string        // AOP_LOG_LEVEL, default info
	LogFormat     string        // AOP_LOG_FORMAT, default json
	HeartbeatTTL  time.Duration // AOP_AGENT_HEARTBEAT_TTL, default 30s
	RegToken      string        // AOP_REGISTRATION_TOKEN
}

func Load() (*Config, error) {
	cfg := &Config{
		DBUrl:        os.Getenv("AOP_DB_URL"),
		JWTExpiry:    8 * time.Hour,
		Port:         "8080",
		LogLevel:     "info",
		LogFormat:    "json",
		HeartbeatTTL: 30 * time.Second,
		RegToken:     os.Getenv("AOP_REGISTRATION_TOKEN"),
	}

	var errs []error

	if cfg.DBUrl == "" {
		errs = append(errs, errors.New("AOP_DB_URL is required"))
	}

	keyHex := os.Getenv("AOP_ENCRYPTION_KEY")
	if keyHex == "" {
		errs = append(errs, errors.New("AOP_ENCRYPTION_KEY is required"))
	} else {
		key, err := hex.DecodeString(keyHex)
		if err != nil || len(key) != 32 {
			errs = append(errs, errors.New("AOP_ENCRYPTION_KEY must be a 64-character hex string (32 bytes)"))
		} else {
			cfg.EncryptionKey = key
		}
	}

	jwtSecret := os.Getenv("AOP_JWT_SECRET")
	if jwtSecret == "" {
		errs = append(errs, errors.New("AOP_JWT_SECRET is required"))
	} else {
		cfg.JWTSecret = []byte(jwtSecret)
	}

	if cfg.RegToken == "" {
		errs = append(errs, errors.New("AOP_REGISTRATION_TOKEN is required"))
	}

	if v := os.Getenv("AOP_JWT_EXPIRY"); v != "" {
		d, err := time.ParseDuration(v)
		if err != nil {
			errs = append(errs, fmt.Errorf("AOP_JWT_EXPIRY invalid: %w", err))
		} else {
			cfg.JWTExpiry = d
		}
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
	if v := os.Getenv("AOP_AGENT_HEARTBEAT_TTL"); v != "" {
		d, err := time.ParseDuration(v)
		if err != nil {
			errs = append(errs, fmt.Errorf("AOP_AGENT_HEARTBEAT_TTL invalid: %w", err))
		} else {
			cfg.HeartbeatTTL = d
		}
	}

	if len(errs) > 0 {
		return nil, errors.Join(errs...)
	}
	return cfg, nil
}
