// Package reconciler implements the controller's main dispatch loop.
// It polls for pending jobs, selects agents, builds payloads, and dispatches.
// It also detects stuck jobs and transitions them to failed.
package reconciler

import (
	"context"
	"fmt"
	"time"

	"github.com/donnie-ellis/aop/pkg/transport"
	"github.com/donnie-ellis/aop/pkg/types"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// jobStore is the subset of store.Store methods the reconciler needs.
type jobStore interface {
	GetPendingJobs(ctx context.Context) ([]types.Job, error)
	GetStuckDispatchedJobs(ctx context.Context, timeout time.Duration) ([]types.Job, error)
	GetStuckRunningJobs(ctx context.Context, timeout time.Duration) ([]types.Job, error)
	TransitionJob(ctx context.Context, jobID uuid.UUID, from, to types.JobStatus, agentID *uuid.UUID, reason string) (bool, error)
	GetAvailableAgents(ctx context.Context, ttl time.Duration) ([]types.Agent, error)
	GetJobTemplate(ctx context.Context, id uuid.UUID) (*types.JobTemplate, error)
	GetProject(ctx context.Context, id uuid.UUID) (*types.Project, error)
	GetInventoryHosts(ctx context.Context, projectID uuid.UUID) ([]types.InventoryHost, error)
	GetCredentialWithSecret(ctx context.Context, id uuid.UUID) (*types.Credential, error)
}

// secretsProvider decrypts a credential into its raw fields.
type secretsProvider interface {
	Resolve(credentialID uuid.UUID) (*types.CredentialSecret, error)
}

// Config bundles everything the reconciler needs from the outer config.
type Config struct {
	APIURL          string
	HeartbeatTTL    time.Duration
	DispatchTimeout time.Duration
	RunningTimeout  time.Duration
}

// Reconciler runs the dispatch loop.
type Reconciler struct {
	store    jobStore
	secrets  secretsProvider
	selector transport.AgentSelector
	dispatch transport.AgentTransport
	cfg      Config
	log      zerolog.Logger
}

func New(
	store jobStore,
	secrets secretsProvider,
	selector transport.AgentSelector,
	dispatch transport.AgentTransport,
	cfg Config,
	log zerolog.Logger,
) *Reconciler {
	return &Reconciler{
		store:    store,
		secrets:  secrets,
		selector: selector,
		dispatch: dispatch,
		cfg:      cfg,
		log:      log,
	}
}

// Run blocks, running one reconcile tick every interval until ctx is cancelled.
func (r *Reconciler) Run(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			r.tick(ctx)
		}
	}
}

func (r *Reconciler) tick(ctx context.Context) {
	r.dispatchPending(ctx)
	r.failStuck(ctx)
}

// dispatchPending picks up pending jobs and dispatches them to agents.
func (r *Reconciler) dispatchPending(ctx context.Context) {
	jobs, err := r.store.GetPendingJobs(ctx)
	if err != nil {
		r.log.Error().Err(err).Msg("get pending jobs")
		return
	}
	if len(jobs) == 0 {
		return
	}

	agents, err := r.store.GetAvailableAgents(ctx, r.cfg.HeartbeatTTL)
	if err != nil {
		r.log.Error().Err(err).Msg("get available agents")
		return
	}

	for _, job := range jobs {
		if err := r.dispatchOne(ctx, job, agents); err != nil {
			r.log.Error().Err(err).Str("job_id", job.ID.String()).Msg("dispatch failed")
			// Fail the job so it doesn't get stuck in pending.
			reason := fmt.Sprintf("dispatch error: %v", err)
			if _, terr := r.store.TransitionJob(ctx, job.ID, types.JobStatusPending, types.JobStatusFailed, nil, reason); terr != nil {
				r.log.Error().Err(terr).Str("job_id", job.ID.String()).Msg("fail pending job")
			}
		}
	}
}

// dispatchOne handles the full dispatch flow for a single job.
func (r *Reconciler) dispatchOne(ctx context.Context, job types.Job, agents []types.Agent) error {
	agent, err := r.selector.Select(ctx, agents)
	if err != nil {
		return fmt.Errorf("agent selection: %w", err)
	}

	payload, err := r.buildPayload(ctx, job, agent.ID)
	if err != nil {
		return fmt.Errorf("build payload: %w", err)
	}

	// Mark dispatched before calling the agent so we don't double-dispatch on
	// a slow loop tick.
	ok, err := r.store.TransitionJob(ctx, job.ID, types.JobStatusPending, types.JobStatusDispatched, &agent.ID, "")
	if err != nil {
		return fmt.Errorf("mark dispatched: %w", err)
	}
	if !ok {
		// Another controller picked it up — skip.
		return nil
	}

	if err := r.dispatch.Dispatch(ctx, agent.ID, payload); err != nil {
		// Agent rejected — mark failed. dispatchOne is responsible for the
		// terminal transition once the claim succeeded; don't bubble the error
		// back to dispatchPending so it doesn't attempt a second transition.
		reason := fmt.Sprintf("agent dispatch error: %v", err)
		if _, terr := r.store.TransitionJob(ctx, job.ID, types.JobStatusDispatched, types.JobStatusFailed, nil, reason); terr != nil {
			r.log.Error().Err(terr).Str("job_id", job.ID.String()).Msg("revert dispatch failure")
			return terr
		}
		r.log.Warn().Err(err).Str("job_id", job.ID.String()).Msg("dispatch failed, job marked failed")
		return nil
	}

	r.log.Info().
		Str("job_id", job.ID.String()).
		Str("agent_id", agent.ID.String()).
		Str("agent", agent.Name).
		Msg("job dispatched")
	return nil
}

// buildPayload assembles a JobPayload from DB records and decrypted credentials.
func (r *Reconciler) buildPayload(ctx context.Context, job types.Job, agentID uuid.UUID) (types.JobPayload, error) {
	tmpl, err := r.store.GetJobTemplate(ctx, job.TemplateID)
	if err != nil {
		return types.JobPayload{}, fmt.Errorf("get template: %w", err)
	}

	project, err := r.store.GetProject(ctx, tmpl.ProjectID)
	if err != nil {
		return types.JobPayload{}, fmt.Errorf("get project: %w", err)
	}

	hosts, err := r.store.GetInventoryHosts(ctx, tmpl.ProjectID)
	if err != nil {
		return types.JobPayload{}, fmt.Errorf("get inventory: %w", err)
	}

	// Merge extra vars: template defaults are the base; job vars override.
	extraVars := make(map[string]any, len(tmpl.DefaultExtraVars)+len(job.ExtraVars))
	for k, v := range tmpl.DefaultExtraVars {
		extraVars[k] = v
	}
	for k, v := range job.ExtraVars {
		extraVars[k] = v
	}

	// Decrypt credentials.
	var creds []types.CredentialSecret
	if tmpl.CredentialID != nil {
		secret, err := r.secrets.Resolve(*tmpl.CredentialID)
		if err != nil {
			return types.JobPayload{}, fmt.Errorf("resolve credential: %w", err)
		}
		creds = append(creds, *secret)
	}

	return types.JobPayload{
		JobID:       job.ID,
		TemplateID:  job.TemplateID,
		Playbook:    tmpl.Playbook,
		Inventory:   hosts,
		ExtraVars:   extraVars,
		Credentials: creds,
		CallbackURL: r.cfg.APIURL,
		RepoURL:     project.RepoURL,
		RepoBranch:  project.Branch,
	}, nil
}

// failStuck marks dispatched and running jobs that haven't progressed as failed.
func (r *Reconciler) failStuck(ctx context.Context) {
	r.failStuckJobs(ctx, types.JobStatusDispatched, r.cfg.DispatchTimeout, "dispatch timeout: agent never reported running")
	r.failStuckJobs(ctx, types.JobStatusRunning, r.cfg.RunningTimeout, "running timeout: agent stopped reporting")
}

func (r *Reconciler) failStuckJobs(ctx context.Context, status types.JobStatus, timeout time.Duration, reason string) {
	var jobs []types.Job
	var err error
	switch status {
	case types.JobStatusDispatched:
		jobs, err = r.store.GetStuckDispatchedJobs(ctx, timeout)
	case types.JobStatusRunning:
		jobs, err = r.store.GetStuckRunningJobs(ctx, timeout)
	}
	if err != nil {
		r.log.Error().Err(err).Str("status", string(status)).Msg("get stuck jobs")
		return
	}
	for _, job := range jobs {
		ok, err := r.store.TransitionJob(ctx, job.ID, status, types.JobStatusFailed, nil, reason)
		if err != nil {
			r.log.Error().Err(err).Str("job_id", job.ID.String()).Msg("fail stuck job")
			continue
		}
		if ok {
			r.log.Warn().Str("job_id", job.ID.String()).Str("from", string(status)).Msg("stuck job marked failed")
		}
	}
}
