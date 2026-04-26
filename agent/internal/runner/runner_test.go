package runner

import (
	"sync"
	"testing"
	"time"

	"github.com/donnie-ellis/aop/pkg/types"
)

func TestLogBatcher_FlushSendsLines(t *testing.T) {
	var mu sync.Mutex
	var received []types.JobLogLine
	done := make(chan struct{})

	b := NewLogBatcher(10, time.Hour, func(lines []types.JobLogLine) {
		mu.Lock()
		received = append(received, lines...)
		mu.Unlock()
		close(done)
	})

	b.Add(types.JobLogLine{Seq: 1, Line: "line1", Stream: "stdout"})
	b.Add(types.JobLogLine{Seq: 2, Line: "line2", Stream: "stderr"})
	b.Flush()

	// Flush() closes the internal done channel but doesn't block — wait for the
	// goroutine to actually call the flush func before asserting.
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("flush did not complete within 1s")
	}

	mu.Lock()
	defer mu.Unlock()
	if len(received) != 2 {
		t.Fatalf("expected 2 lines, got %d", len(received))
	}
	if received[0].Line != "line1" {
		t.Errorf("line 0: got %q, want line1", received[0].Line)
	}
	if received[1].Line != "line2" {
		t.Errorf("line 1: got %q, want line2", received[1].Line)
	}
}

func TestLogBatcher_TimerFlush(t *testing.T) {
	var mu sync.Mutex
	var received []types.JobLogLine
	flushed := make(chan struct{}, 1)

	b := NewLogBatcher(100, 50*time.Millisecond, func(lines []types.JobLogLine) {
		mu.Lock()
		received = append(received, lines...)
		mu.Unlock()
		select {
		case flushed <- struct{}{}:
		default:
		}
	})
	defer b.Flush()

	b.Add(types.JobLogLine{Seq: 1, Line: "tick", Stream: "stdout"})

	select {
	case <-flushed:
	case <-time.After(500 * time.Millisecond):
		t.Fatal("timer flush did not fire within 500ms")
	}

	mu.Lock()
	defer mu.Unlock()
	if len(received) == 0 {
		t.Error("timer flush should have sent lines")
	}
}

func TestLogBatcher_EmptyFlushIsNoop(t *testing.T) {
	called := false
	b := NewLogBatcher(10, time.Hour, func(lines []types.JobLogLine) {
		called = true
	})
	b.Flush()

	if called {
		t.Error("flush on empty batcher should not invoke the flush func")
	}
}

func TestLogBatcher_PreservesOrder(t *testing.T) {
	var mu sync.Mutex
	var received []types.JobLogLine
	done := make(chan struct{})

	b := NewLogBatcher(100, time.Hour, func(lines []types.JobLogLine) {
		mu.Lock()
		received = append(received, lines...)
		mu.Unlock()
		close(done)
	})

	for i := 1; i <= 5; i++ {
		b.Add(types.JobLogLine{Seq: i, Line: "line", Stream: "stdout"})
	}
	b.Flush()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("flush did not complete within 1s")
	}

	mu.Lock()
	defer mu.Unlock()
	for i, line := range received {
		if line.Seq != i+1 {
			t.Errorf("out of order at index %d: got seq %d, want %d", i, line.Seq, i+1)
		}
	}
}

func TestLogBatcher_MultipleTimerFlushes(t *testing.T) {
	var mu sync.Mutex
	callCount := 0

	b := NewLogBatcher(100, 50*time.Millisecond, func(lines []types.JobLogLine) {
		mu.Lock()
		callCount++
		mu.Unlock()
	})
	defer b.Flush()

	// Add lines in two separate windows.
	b.Add(types.JobLogLine{Seq: 1, Line: "first", Stream: "stdout"})
	time.Sleep(100 * time.Millisecond)
	b.Add(types.JobLogLine{Seq: 2, Line: "second", Stream: "stdout"})
	time.Sleep(100 * time.Millisecond)

	mu.Lock()
	n := callCount
	mu.Unlock()

	if n < 1 {
		t.Errorf("expected at least 1 flush call, got %d", n)
	}
}
