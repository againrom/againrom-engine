package video

import (
	"testing"
	"time"
)

// The first poll arms the deadline and is not ready; the clock then has to
// reach it. Ready re-arms by lateness: within one interval after the deadline
// from now, up to one further interval from the deadline, later from now.
func TestTimerPacerPollSequence(t *testing.T) {
	p := newTimerPacer(1000)
	const t0 = 5_000_000
	if p.poll(t0) {
		t.Fatal("first poll must not be ready")
	}
	if p.deadline != t0+1000 {
		t.Fatalf("deadline = %d, want %d", p.deadline, t0+1000)
	}
	if p.poll(t0 + 999) {
		t.Fatal("ready before the deadline")
	}
	if !p.poll(t0 + 1000) {
		t.Fatal("not ready at the deadline")
	}
	if p.deadline != t0+2000 {
		t.Fatalf("on-time re-arm deadline = %d, want %d", p.deadline, t0+2000)
	}
	// One interval or more late, less than two: deadline plus interval.
	if !p.poll(t0 + 3500) {
		t.Fatal("late poll not ready")
	}
	if p.deadline != t0+3000 {
		t.Fatalf("late re-arm deadline = %d, want %d", p.deadline, t0+3000)
	}
	// Two or more intervals late: from now.
	if !p.poll(t0 + 6000) {
		t.Fatal("very late poll not ready")
	}
	if p.deadline != t0+7000 {
		t.Fatalf("very late re-arm deadline = %d, want %d", p.deadline, t0+7000)
	}
}

func TestTimerPacerZeroIntervalWaitsForNothing(t *testing.T) {
	p := newTimerPacer(0)
	if p.poll(100) {
		t.Fatal("first poll must arm")
	}
	if p.remaining(100) != 0 || !p.poll(100) {
		t.Fatal("zero interval must be ready on the second poll at the same clock")
	}
}

// A 32-bit deadline that wraps below the clock is in the past, so a huge
// positive header value does not wait (VIDEO-073).
func TestTimerPacerWrappedDeadlineIsReady(t *testing.T) {
	p := newTimerPacer(4294967196)
	const now = 1 << 30
	p.poll(now)
	if p.remaining(now) != 0 || !p.poll(now) {
		t.Fatal("wrapped deadline should be ready at once")
	}
}

func TestWaitFrameHoldsForTheInterval(t *testing.T) {
	stop := make(chan struct{})
	c := pacerClock{start: time.Now(), stop: stop}
	p := newTimerPacer(2000) // 20 ms
	begin := time.Now()
	if !waitFrame(p, c) {
		t.Fatal("stopped")
	}
	if d := time.Since(begin); d < 19*time.Millisecond || d > 200*time.Millisecond {
		t.Fatalf("first wait took %v, want about 20ms", d)
	}
}

func TestWaitFrameInterruptedByStop(t *testing.T) {
	stop := make(chan struct{})
	c := pacerClock{start: time.Now(), stop: stop}
	p := newTimerPacer(1 << 30) // about 107 seconds
	done := make(chan bool)
	go func() { done <- waitFrame(p, c) }()
	time.Sleep(20 * time.Millisecond)
	close(stop)
	select {
	case ok := <-done:
		if ok {
			t.Fatal("wait reported ready after stop")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("stop did not interrupt the wait")
	}
}
