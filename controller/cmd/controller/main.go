package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/donnie-ellis/aop/controller/internal/config"
	"github.com/donnie-ellis/aop/controller/internal/reconciler"
	"github.com/donnie-ellis/aop/controller/internal/scheduler"
	"github.com/donnie-ellis/aop/controller/internal/secrets"
	"github.com/donnie-ellis/aop/controller/internal/store"
	"github.com/donnie-ellis/aop/controller/inventory"
	"github.com/donnie-ellis/aop/controller/selector"
	"github.com/donnie-ellis/aop/controller/transport"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal().Err(err).Msg("config error")
	}

	logger := buildLogger(cfg)

	if cfg.EncryptionKey == nil {
		logger.Warn().Msg("AOP_ENCRYPTION_KEY not set — jobs with credentials will fail at dispatch")
	}

	pool, err := pgxpool.New(context.Background(), cfg.DBURL)
	if err != nil {
		logger.Fatal().Err(err).Msg("connect to database")
	}
	defer pool.Close()

	if err := pool.Ping(context.Background()); err != nil {
		logger.Fatal().Err(err).Msg("ping database")
	}
	logger.Info().Msg("database connected")

	db := store.New(pool)
	sel := selector.New()
	xport := transport.New(db)

	recCfg := reconciler.Config{
		APIURL:          cfg.APIURL,
		HeartbeatTTL:    cfg.HeartbeatTTL,
		DispatchTimeout: cfg.DispatchTimeout,
		RunningTimeout:  cfg.RunningTimeout,
	}
	rec := reconciler.New(db, secrets.NewProvider(db, cfg.EncryptionKey), sel, xport, recCfg, logger)
	sched := scheduler.New(db, logger)
	gitSync := inventory.NewGitSync(db, cfg.WorkspaceDir, logger)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go rec.Run(ctx, cfg.ReconcileInterval)
	go sched.Run(ctx, cfg.SyncInterval)
	go gitSync.Run(ctx, cfg.SyncInterval)

	logger.Info().
		Dur("reconcile_interval", cfg.ReconcileInterval).
		Dur("heartbeat_ttl", cfg.HeartbeatTTL).
		Bool("credentials_enabled", cfg.EncryptionKey != nil).
		Msg("controller started")

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info().Msg("shutting down")
	cancel()
}

func buildLogger(cfg *config.Config) zerolog.Logger {
	level, err := zerolog.ParseLevel(cfg.LogLevel)
	if err != nil {
		level = zerolog.InfoLevel
	}
	zerolog.SetGlobalLevel(level)

	if cfg.LogFormat == "pretty" {
		return zerolog.New(zerolog.ConsoleWriter{Out: os.Stderr}).With().Timestamp().Logger()
	}
	return zerolog.New(os.Stderr).With().Timestamp().Logger()
}
