// Package scheduler fires cron schedules and creates job records for each
// schedule that is due to run. Jobs are created in pending status; the
// reconciler picks them up like any other pending job.
package scheduler

import (
	"context"
	"fmt"
	"time"

	"github.com/donnie-ellis/aop/pkg/types"
	"github.com/google/uuid"
	"github.com/robfig/cron/v3"
	"github.com/rs/zerolog"
)

// scheduleStore is the subset of store.Store the scheduler needs.
type scheduleStore interface {
	GetActiveSchedules(ctx context.Context) ([]types.Schedule, error)
	UpdateScheduleLastRun(ctx context.Context, scheduleID uuid.UUID, nextRunAt *time.Time) error
	CreateJob(ctx context.Context, templateID uuid.UUID, extraVars map[string]any) (*types.Job, error)
}

// Scheduler loads cron schedules from the DB and fires job creation on each tick.
// It reloads the schedule list on every sync cycle so changes take effect without
// a process restart.
type Scheduler struct {
	store    scheduleStore
	log      zerolog.Logger
	cron     *cron.Cron
	entryIDs map[uuid.UUID]cron.EntryID // scheduleID → cron entry
}

func New(store scheduleStore, log zerolog.Logger) *Scheduler {
	return &Scheduler{
		store:    store,
		log:      log,
		entryIDs: make(map[uuid.UUID]cron.EntryID),
	}
}

// Run starts the scheduler and blocks until ctx is cancelled. It performs an
// initial load and then re-syncs on every syncInterval.
func (s *Scheduler) Run(ctx context.Context, syncInterval time.Duration) {
	s.cron = cron.New(cron.WithSeconds())
	s.cron.Start()
	defer s.cron.Stop()

	s.sync(ctx)

	ticker := time.NewTicker(syncInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.sync(ctx)
		}
	}
}

// sync reloads schedules from the DB, registers new ones, and removes
// entries for schedules that were deleted or disabled.
func (s *Scheduler) sync(ctx context.Context) {
	schedules, err := s.store.GetActiveSchedules(ctx)
	if err != nil {
		s.log.Error().Err(err).Msg("load schedules")
		return
	}

	// Build set of currently active schedule IDs.
	active := make(map[uuid.UUID]struct{}, len(schedules))
	for _, sc := range schedules {
		active[sc.ID] = struct{}{}
	}

	// Remove cron entries for schedules that are no longer active.
	for id, entryID := range s.entryIDs {
		if _, ok := active[id]; !ok {
			s.cron.Remove(entryID)
			delete(s.entryIDs, id)
			s.log.Info().Str("schedule_id", id.String()).Msg("schedule removed")
		}
	}

	// Register any new schedules.
	for _, sc := range schedules {
		if _, registered := s.entryIDs[sc.ID]; registered {
			continue
		}
		if err := s.register(sc); err != nil {
			s.log.Error().Err(err).Str("schedule_id", sc.ID.String()).Str("name", sc.Name).Msg("register schedule")
		}
	}
}

func (s *Scheduler) register(sc types.Schedule) error {
	loc, err := time.LoadLocation(sc.Timezone)
	if err != nil {
		return fmt.Errorf("load timezone %q: %w", sc.Timezone, err)
	}

	parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
	_, err = parser.Parse(sc.CronExpr)
	if err != nil {
		return fmt.Errorf("invalid cron expression %q: %w", sc.CronExpr, err)
	}

	scheduleID := sc.ID
	templateID := sc.TemplateID
	extraVars := sc.ExtraVars

	entryID, err := s.cron.AddFunc(
		"CRON_TZ="+loc.String()+" "+sc.CronExpr,
		func() {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			job, err := s.store.CreateJob(ctx, templateID, extraVars)
			if err != nil {
				s.log.Error().Err(err).Str("schedule_id", scheduleID.String()).Msg("create scheduled job")
				return
			}

			entries := s.cron.Entries()
			var nextRun *time.Time
			for _, e := range entries {
				if e.ID == s.entryIDs[scheduleID] {
					t := e.Next
					nextRun = &t
					break
				}
			}

			if err := s.store.UpdateScheduleLastRun(ctx, scheduleID, nextRun); err != nil {
				s.log.Error().Err(err).Str("schedule_id", scheduleID.String()).Msg("update schedule last_run_at")
			}

			s.log.Info().
				Str("schedule_id", scheduleID.String()).
				Str("job_id", job.ID.String()).
				Msg("scheduled job created")
		},
	)
	if err != nil {
		return fmt.Errorf("add cron func: %w", err)
	}

	s.entryIDs[sc.ID] = entryID
	s.log.Info().
		Str("schedule_id", sc.ID.String()).
		Str("name", sc.Name).
		Str("cron", sc.CronExpr).
		Msg("schedule registered")
	return nil
}
