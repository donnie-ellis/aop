package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/donnie-ellis/aop/api/internal/auth"
	"github.com/donnie-ellis/aop/api/internal/config"
	"github.com/donnie-ellis/aop/api/internal/handler"
	"github.com/donnie-ellis/aop/api/internal/secrets"
	"github.com/donnie-ellis/aop/api/internal/server"
	"github.com/donnie-ellis/aop/api/internal/store"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal().Err(err).Msg("config error")
	}

	logger := buildLogger(cfg)

	pool, err := store.Connect(context.Background(), cfg.DBUrl)
	if err != nil {
		logger.Fatal().Err(err).Msg("db connect")
	}
	defer pool.Close()

	st := store.New(pool)
	sp := secrets.NewProvider(st, cfg.EncryptionKey)
	reg := auth.NewStaticTokenValidator(cfg.RegToken)
	h := handler.New(st, sp, cfg, reg, logger)
	srv := server.New(cfg, h, st, logger)

	go func() {
		if err := srv.Start(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Fatal().Err(err).Msg("server error")
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		logger.Error().Err(err).Msg("shutdown error")
	}
	logger.Info().Msg("server stopped")
}

func buildLogger(cfg *config.Config) zerolog.Logger {
	level, err := zerolog.ParseLevel(cfg.LogLevel)
	if err != nil {
		level = zerolog.InfoLevel
	}
	zerolog.SetGlobalLevel(level)

	if cfg.LogFormat == "console" {
		return zerolog.New(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339}).
			With().Timestamp().Logger()
	}
	return zerolog.New(os.Stderr).With().Timestamp().Logger()
}
