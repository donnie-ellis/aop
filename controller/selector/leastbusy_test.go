package selector

import (
	"context"
	"testing"
	"time"

	"github.com/donnie-ellis/aop/pkg/types"
	"github.com/google/uuid"
)

func TestSelect_Empty(t *testing.T) {
	s := New()
	_, err := s.Select(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error for empty agent list")
	}
}

func TestSelect_Single(t *testing.T) {
	s := New()
	agents := []types.Agent{{ID: uuid.New(), RunningJobs: 2}}
	got, err := s.Select(context.Background(), agents)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ID != agents[0].ID {
		t.Errorf("got %v, want %v", got.ID, agents[0].ID)
	}
}

func TestSelect_PicksLeastBusy(t *testing.T) {
	s := New()
	idBusy := uuid.New()
	idIdle := uuid.New()
	agents := []types.Agent{
		{ID: idBusy, RunningJobs: 3},
		{ID: idIdle, RunningJobs: 0},
		{ID: uuid.New(), RunningJobs: 2},
	}
	got, err := s.Select(context.Background(), agents)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ID != idIdle {
		t.Errorf("got agent %v, want idle agent %v", got.ID, idIdle)
	}
}

func TestSelect_TiebreakByHeartbeat(t *testing.T) {
	s := New()
	older := time.Now().Add(-20 * time.Second)
	newer := time.Now().Add(-5 * time.Second)

	idRecent := uuid.New()
	agents := []types.Agent{
		{ID: uuid.New(), RunningJobs: 1, LastHeartbeatAt: &older},
		{ID: idRecent, RunningJobs: 1, LastHeartbeatAt: &newer},
	}
	got, err := s.Select(context.Background(), agents)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ID != idRecent {
		t.Errorf("tie should be broken by most-recent heartbeat; got %v, want %v", got.ID, idRecent)
	}
}

func TestSelect_TiebreakNilHeartbeat(t *testing.T) {
	s := New()
	hb := time.Now()
	idWithHB := uuid.New()
	agents := []types.Agent{
		{ID: uuid.New(), RunningJobs: 0, LastHeartbeatAt: nil},
		{ID: idWithHB, RunningJobs: 0, LastHeartbeatAt: &hb},
	}
	got, err := s.Select(context.Background(), agents)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ID != idWithHB {
		t.Errorf("agent with heartbeat should win over nil; got %v", got.ID)
	}
}

func TestSelect_AllSameLoad(t *testing.T) {
	s := New()
	agents := []types.Agent{
		{ID: uuid.New(), RunningJobs: 2},
		{ID: uuid.New(), RunningJobs: 2},
		{ID: uuid.New(), RunningJobs: 2},
	}
	got, err := s.Select(context.Background(), agents)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Any agent is acceptable; just verify one was returned.
	found := false
	for _, a := range agents {
		if a.ID == got.ID {
			found = true
			break
		}
	}
	if !found {
		t.Error("returned agent not in input slice")
	}
}
