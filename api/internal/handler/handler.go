package handler

import (
	"github.com/donnie-ellis/aop/api/internal/auth"
	"github.com/donnie-ellis/aop/api/internal/config"
	"github.com/donnie-ellis/aop/api/internal/store"
	"github.com/donnie-ellis/aop/pkg/types"
	"github.com/rs/zerolog"
)

// Handlers holds shared dependencies for all HTTP handlers.
type Handlers struct {
	store   *store.Store
	secrets types.SecretsProvider
	cfg     *config.Config
	reg     auth.RegistrationValidator
	log     zerolog.Logger
}

func New(st *store.Store, secrets types.SecretsProvider, cfg *config.Config, reg auth.RegistrationValidator, log zerolog.Logger) *Handlers {
	return &Handlers{
		store:   st,
		secrets: secrets,
		cfg:     cfg,
		reg:     reg,
		log:     log,
	}
}
