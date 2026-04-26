package main

import (
	"context"
	"errors"
	"os"
	"os/signal"
	"syscall"

	"github.com/donnie-ellis/aop/controller/internal/config"
	"github.com/donnie-ellis/aop/controller/internal/reconciler"
	"github.com/donnie-ellis/aop/controller/internal/scheduler"
	"github.com/donnie-ellis/aop/controller/internal/store"
	"github.com/donnie-ellis/aop/controller/inventory"
	"github.com/donnie-ellis/aop/controller/selector"
	"github.com/donnie-ellis/aop/controller/transport"
	"github.com/donnie-ellis/aop/pkg/types"
	"github.com/google/uuid"
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
	// SecretsProvider: reuse the store as the credential getter (controller
	// decrypts via AES-GCM using the same key material as the API). In v1 the
	// controller does not have an encryption key — it relies on the store
	// returning plaintext-equivalent data via GetCredentialWithSecret. A full
	// SecretsProvider implementation will be wired here once the controller has
	// AOP_ENCRYPTION_KEY in its config.
	rec := reconciler.New(db, &noopSecrets{}, sel, xport, recCfg, logger)
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

// noopSecrets is a placeholder SecretsProvider that always errors. It will be
// replaced once AOP_ENCRYPTION_KEY is added to the controller config and the
// full postgres secrets provider is wired in.
type noopSecrets struct{}

func (n *noopSecrets) Resolve(credentialID uuid.UUID) (*types.CredentialSecret, error) {
	return nil, errors.New("no secrets provider configured: set AOP_ENCRYPTION_KEY")
}
