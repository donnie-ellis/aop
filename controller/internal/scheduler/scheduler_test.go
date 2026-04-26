package scheduler

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/donnie-ellis/aop/pkg/types"
	"github.com/google/uuid"
	"github.com/robfig/cron/v3"
	"github.com/rs/zerolog"
)

// ---------------------------------------------------------------------------
// Mock store
// ---------------------------------------------------------------------------

type mockStore struct {
	mu        sync.Mutex
	schedules []types.Schedule
	created   []types.Job
	createErr error
}

func (m *mockStore) GetActiveSchedules(_ context.Context) ([]types.Schedule, error) {
	return m.schedules, nil
}
func (m *mockStore) UpdateScheduleLastRun(_ context.Context, _ uuid.UUID, _ *time.Time) error {
	return nil
}
func (m *mockStore) CreateJob(_ context.Context, templateID uuid.UUID, extraVars map[string]any) (*types.Job, error) {
	if m.createErr != nil {
		return nil, m.createErr
	}
	j := &types.Job{ID: uuid.New(), TemplateID: templateID, Status: types.JobStatusPending}
	m.mu.Lock()
	m.created = append(m.created, *j)
	m.mu.Unlock()
	return j, nil
}

func nop() zerolog.Logger { return zerolog.Nop() }

// ---------------------------------------------------------------------------
// register error cases
// ---------------------------------------------------------------------------

func TestRegister_InvalidTimezone(t *testing.T) {
	s := New(&mockStore{}, nop())
	s.cron = cron.New()

	sc := types.Schedule{
		ID:         uuid.New(),
		TemplateID: uuid.New(),
		CronExpr:   "* * * * *",
		Timezone:   "Not/A/Timezone",
	}
	if err := s.register(sc); err == nil {
		t.Fatal("expected error for invalid timezone")
	}
}

func TestRegister_InvalidCronExpr(t *testing.T) {
	s := New(&mockStore{}, nop())
	s.cron = cron.New()

	sc := types.Schedule{
		ID:         uuid.New(),
		TemplateID: uuid.New(),
		CronExpr:   "not a cron expression",
		Timezone:   "UTC",
	}
	if err := s.register(sc); err == nil {
		t.Fatal("expected error for invalid cron expression")
	}
}

func TestRegister_ValidSchedule(t *testing.T) {
	s := New(&mockStore{}, nop())
	s.cron = cron.New()

	sc := types.Schedule{
		ID:         uuid.New(),
		TemplateID: uuid.New(),
		CronExpr:   "0 9 * * 1", // every Monday at 09:00
		Timezone:   "America/New_York",
	}
	if err := s.register(sc); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := s.entryIDs[sc.ID]; !ok {
		t.Error("entry should be registered in entryIDs map")
	}
}

// ---------------------------------------------------------------------------
// sync: register / deregister behaviour
// ---------------------------------------------------------------------------

func TestSync_RegistersNewSchedules(t *testing.T) {
	sc := types.Schedule{
		ID:         uuid.New(),
		TemplateID: uuid.New(),
		CronExpr:   "0 * * * *",
		Timezone:   "UTC",
		Enabled:    true,
	}
	store := &mockStore{schedules: []types.Schedule{sc}}
	s := New(store, nop())
	s.cron = cron.New()

	s.sync(context.Background())

	if _, ok := s.entryIDs[sc.ID]; !ok {
		t.Error("schedule should be registered after sync")
	}
}

func TestSync_DoesNotDuplicateExisting(t *testing.T) {
	sc := types.Schedule{
		ID:         uuid.New(),
		TemplateID: uuid.New(),
		CronExpr:   "0 * * * *",
		Timezone:   "UTC",
		Enabled:    true,
	}
	store := &mockStore{schedules: []types.Schedule{sc}}
	s := New(store, nop())
	s.cron = cron.New()

	s.sync(context.Background())
	s.sync(context.Background()) // second call — must not double-register

	if len(s.entryIDs) != 1 {
		t.Errorf("expected 1 entry, got %d", len(s.entryIDs))
	}
}

func TestSync_RemovesDeletedSchedules(t *testing.T) {
	sc := types.Schedule{
		ID:         uuid.New(),
		TemplateID: uuid.New(),
		CronExpr:   "0 * * * *",
		Timezone:   "UTC",
		Enabled:    true,
	}
	store := &mockStore{schedules: []types.Schedule{sc}}
	s := New(store, nop())
	s.cron = cron.New()

	s.sync(context.Background())

	// Remove the schedule from DB (simulate disable/delete).
	store.schedules = nil
	s.sync(context.Background())

	if len(s.entryIDs) != 0 {
		t.Errorf("expected entry removed after schedule gone from DB; got %d entries", len(s.entryIDs))
	}
}

// ---------------------------------------------------------------------------
// Fired job creation
// ---------------------------------------------------------------------------

func TestScheduleFires_CreatesJob(t *testing.T) {
	templateID := uuid.New()
	store := &mockStore{}
	s := New(store, nop())
	s.cron = cron.New()

	sc := types.Schedule{
		ID:         uuid.New(),
		TemplateID: templateID,
		CronExpr:   "0 * * * *", // top of every hour
		Timezone:   "UTC",
	}
	if err := s.register(sc); err != nil {
		t.Fatalf("register: %v", err)
	}

	// Directly invoke the registered entry's job function rather than waiting
	// for the cron ticker, which would require a real-time wait.
	entries := s.cron.Entries()
	if len(entries) == 0 {
		t.Fatal("no cron entries registered")
	}
	entries[0].Job.Run()

	store.mu.Lock()
	defer store.mu.Unlock()
	if len(store.created) == 0 {
		t.Fatal("expected job to be created after running the cron entry")
	}
	if store.created[0].TemplateID != templateID {
		t.Errorf("template_id: got %v, want %v", store.created[0].TemplateID, templateID)
	}
}
