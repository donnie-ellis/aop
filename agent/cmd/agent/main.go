package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/donnie-ellis/aop/agent/internal/apiclient"
	"github.com/donnie-ellis/aop/agent/internal/config"
	"github.com/donnie-ellis/aop/agent/internal/server"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal().Err(err).Msg("config error")
	}

	logger := buildLogger(cfg)

	if err := os.MkdirAll(cfg.WorkDir, 0755); err != nil {
		logger.Fatal().Err(err).Str("path", cfg.WorkDir).Msg("create work dir")
	}

	client := apiclient.New(cfg.APIURL, logger)

	regCtx, regCancel := context.WithTimeout(context.Background(), 30*time.Second)
	if err := client.Register(regCtx, cfg.RegistrationToken, cfg.AgentName, cfg.AgentAddress, cfg.Labels, cfg.Capacity); err != nil {
		regCancel()
		logger.Fatal().Err(err).Msg("registration failed")
	}
	regCancel()

	srv := server.New(cfg.Port, cfg.WorkDir, cfg.Capacity, client, logger)

	// Heartbeat loop.
	go func() {
		ticker := time.NewTicker(cfg.HeartbeatInterval)
		defer ticker.Stop()
		for range ticker.C {
			hbCtx, hbCancel := context.WithTimeout(context.Background(), 10*time.Second)
			if err := client.Heartbeat(hbCtx, srv.RunningJobs()); err != nil {
				logger.Warn().Err(err).Msg("heartbeat failed")
			}
			hbCancel()
		}
	}()

	// Dispatch listener.
	go func() {
		if err := srv.Start(); err != nil && err != http.ErrServerClosed {
			logger.Fatal().Err(err).Msg("dispatch server error")
		}
	}()

	logger.Info().
		Str("address", cfg.AgentAddress).
		Str("api", cfg.APIURL).
		Int("capacity", cfg.Capacity).
		Msg("agent ready")

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info().Msg("shutting down")
	shutCtx, shutCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutCancel()
	if err := srv.Shutdown(shutCtx); err != nil {
		logger.Error().Err(err).Msg("shutdown error")
	}
}

func buildLogger(cfg *config.Config) zerolog.Logger {
	level, err := zerolog.ParseLevel(cfg.LogLevel)
	if err != nil {
		level = zerolog.InfoLevel
	}
	zerolog.SetGlobalLevel(level)

	if cfg.LogFormat == "console" {
		return zerolog.New(zerolog.ConsoleWriter{Out: os.Stderr}).With().Timestamp().Logger()
	}
	return zerolog.New(os.Stderr).With().Timestamp().Logger()
}
