package game

import (
	"bytes"
	"reflect"
	"testing"
	"time"

	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

// The cadence every case below is driven at: our rate model's 16 ticks a second,
// so 1000000/16 = 62500 us a tick — hand-computed, never read back off a clock —
// and a 60 fps front-end's own 16667 us between frames. It is deliberately a
// period NO rung of the cadence ladder produces, so nothing below can be passing
// only because the driver was handed a cadence the ladder happens to name.
const (
	stoppedPeriodUS = 62_500
	stoppedFrameUS  = 16_667
)

// stoppedOrders are the orders issued while the world is stopped, in ISSUE
// ORDER. Entity 0 is named TWICE, to different cells, which is the whole
// discriminating power of the list: Step applies a slice in order with the last
// write winning, so a driver that drained the queue backwards sends entity 0 to
// (25,23) instead of (31,28) and misses the digest at the very first tick.
//
// Every target is inside the fixture's INTERIOR — not merely inside its extent —
// and none is a cell another listed entity is sent to, so neither the ring nor
// contention decides where anything ends up.
var stoppedOrders = []struct {
	entity uint32
	x, y   int
}{
	{entity: 0, x: 25, y: 23},
	{entity: 2, x: 29, y: 27},
	{entity: 0, x: 31, y: 28},
}

// TestOrdersIssuedWhileStoppedReachTheFirstTickAfterIt — 0041 SC-4's order
// clause (AC-4): orders issued while the stop is set are held, none of them
// reaching a world; the FIRST tick after the stop clears applies exactly
// those orders, in issue order; and no later tick applies any of them again.
//
// THE HEADLESS LEG IS THE DISCRIMINATOR. Two driven runs agreeing would agree
// just as happily over a queue dropped on the floor, so the resumed world is
// compared against one sim.Step over the command slice this file wrote out — and
// against an untouched world, which is what says that slice did anything at all.
func TestOrdersIssuedWhileStoppedReachTheFirstTickAfterIt(t *testing.T) {
	base := time.Unix(1_700_000_000, 0)
	period := time.Duration(stoppedPeriodUS) * time.Microsecond

	m := worldFixtureMap()
	mw := newMapWorld(mapload.FromALM(m), nil, nil, worldFixtureViewer(t, m))
	if len(mw.sched) != 0 {
		t.Fatalf("fixture: the script holds %d entries, want an empty one", len(mw.sched))
	}
	mw.setCadence(stoppedPeriodUS, false)
	if n, applied := mw.paceTo(base); n != 0 || applied != 0 {
		t.Fatalf("the baseline call ran %d ticks and applied %d orders, want 0 and 0", n, applied)
	}

	beforeBytes, beforeHash := marshalWorld(t, mw.world), mw.world.Hash()

	mw.setCadence(stoppedPeriodUS, true)
	now := base

	// One order per stopped frame, each through the shipped seam and one call
	// each — the very spelling the front-end reaches this world by.
	for i, o := range stoppedOrders {
		mw.enqueue(o.entity, o.x, o.y)
		now = now.Add(time.Duration(stoppedFrameUS) * time.Microsecond)
		if n, applied := mw.paceTo(now); n != 0 || applied != 0 {
			t.Fatalf("the frame after order %d ran %d ticks and applied %d, want 0 and 0", i+1, n, applied)
		}
		if got := len(mw.pending); got != i+1 {
			t.Fatalf("the queue holds %d orders after %d enqueues, want %d", got, i+1, i+1)
		}
		if !bytes.Equal(marshalWorld(t, mw.world), beforeBytes) || mw.world.Hash() != beforeHash {
			t.Fatalf("order %d moved the world with no tick behind it (0028 FR-9, P-1)", i+1)
		}
	}

	// Many further stopped frames: nothing drains and nothing is lost.
	for i := 0; i < 200; i++ {
		now = now.Add(time.Duration(stoppedFrameUS) * time.Microsecond)
		if n, applied := mw.paceTo(now); n != 0 || applied != 0 {
			t.Fatalf("stopped frame %d ran %d ticks and applied %d, want 0 and 0", i+1, n, applied)
		}
	}
	if got, want := len(mw.pending), len(stoppedOrders); got != want {
		t.Fatalf("the queue holds %d orders after a stopped span, want the %d issued into it", got, want)
	}

	// The resume: one period of elapsed time, one tick, and that tick applies
	// every order issued while the stop was set.
	mw.setCadence(stoppedPeriodUS, false)
	now = now.Add(period)
	n, applied := mw.paceTo(now)
	if n != 1 {
		t.Fatalf("the first advance after the stop ran %d ticks, want 1", n)
	}
	if want := len(stoppedOrders); applied != want {
		t.Fatalf("the first tick after the stop applied %d orders, want the %d issued while it was set "+
			"(FR-4: none lost)", applied, want)
	}
	if got := len(mw.pending); got != 0 {
		t.Errorf("the queue holds %d orders after the drain, want 0", got)
	}

	// The headless side: ONE step over exactly those commands, in the order this
	// file lists them.
	cmds := make([]sim.Command, 0, len(stoppedOrders))
	for _, o := range stoppedOrders {
		cmds = append(cmds, sim.Command{Entity: sim.EntityID(o.entity), X: int32(o.x), Y: int32(o.y)})
	}
	// The front-end issues a GROUP order per press, tagged by the boundary the
	// queue derives; the mirror tags the same stream through its own copy of it.
	cmds = grouped(cmds)
	headless := mapload.FromALM(m)
	untouched := headless.Hash()
	sim.Step(headless, cmds)

	if got, want := marshalWorld(t, mw.world), marshalWorld(t, headless); !bytes.Equal(got, want) {
		t.Errorf("the resumed world's byte form differs from a headless step over the orders in issue order " +
			"— they were applied in some other order, or some other tick applied them (FR-4)")
	}
	if got, want := mw.world.Hash(), headless.Hash(); got != want {
		t.Errorf("the resumed digest %#x != the headless digest %#x (FR-4)", got, want)
	}
	if headless.Hash() == untouched {
		t.Fatal("the headless step reached a never-stepped world's digest, so the comparison discriminates nothing")
	}

	// ...and no later tick applies any of them again.
	for i := 0; i < 4; i++ {
		now = now.Add(period)
		n, applied := mw.paceTo(now)
		if n != 1 || applied != 0 {
			t.Errorf("tick %d after the drain ran %d ticks and applied %d orders, want 1 and 0 (FR-4: none "+
				"applied twice)", i+2, n, applied)
		}
	}
}

// TestNoEntityFrameChangesWhileTheWorldIsStopped — 0041 SC-4's frame
// clause (AC-4): while the stop is set, no drawn entity's frame changes.
//
// IT IS ASSERTED ON THE PUSHED ENTITIES and not on the scene clock behind them:
// entityDraws is the very derivation push hands the viewer, so what is compared
// is what a frame would draw. The bundle is the resolving one, so an entity has
// a frame to hold at all — under no bundle every entity crosses art-less and
// this whole comparison would be a comparison of nils.
//
// THE RUNNING LEG IS WHAT MAKES IT DISCRIMINATE. Entity 2's class is the
// idle-cycle one, whose sub-frame alternates once per scene tick, so an unstopped
// span of the same length moves a FRAME and not merely a cell. Without that leg,
// "no frame changed" would pass over a bundle that resolved nothing.
func TestNoEntityFrameChangesWhileTheWorldIsStopped(t *testing.T) {
	base := time.Unix(1_700_000_000, 0)
	period := time.Duration(stoppedPeriodUS) * time.Microsecond

	m := worldFixtureMap()
	opened := func() *mapWorld {
		mw := newMapWorld(mapload.FromALM(m), scriptedLaps(), worldFixtureUnitSet(), worldFixtureViewer(t, m))
		mw.setCadence(stoppedPeriodUS, false)
		if n, _ := mw.paceTo(base); n != 0 {
			t.Fatalf("the baseline call ran ticks")
		}
		return mw
	}
	stopped, running := opened(), opened()

	// Six ticks first, so the frames held still below are frames the seam
	// SELECTED rather than the ones a freshly opened screen happens to start on.
	now := base
	for i := 0; i < 6; i++ {
		now = now.Add(period)
		if n, _ := stopped.paceTo(now); n != 1 {
			t.Fatalf("warm-up call %d ran %d ticks, want 1", i+1, n)
		}
		running.paceTo(now)
	}

	held := stopped.entityDraws()
	if len(held) != len(worldFixtureCells) {
		t.Fatalf("the push carries %d entities, want %d", len(held), len(worldFixtureCells))
	}
	resolved := 0
	for _, d := range held {
		if d.Frame != nil {
			resolved++
		}
	}
	if resolved != len(held) {
		t.Fatalf("only %d of %d pushed entities resolved to a frame; a frozen nil is not evidence of a "+
			"frozen frame", resolved, len(held))
	}

	stopped.setCadence(stoppedPeriodUS, true)
	stoppedAt := now
	const frames = 300
	for i := 1; i <= frames; i++ {
		now = stoppedAt.Add(time.Duration(i*stoppedFrameUS) * time.Microsecond)
		if n, _ := stopped.paceTo(now); n != 0 {
			t.Fatalf("stopped frame %d ran %d ticks, want 0", i, n)
		}
		running.paceTo(now)
		if got := stopped.entityDraws(); !reflect.DeepEqual(got, held) {
			t.Fatalf("stopped frame %d changed what the push carries:\n got %+v\nwant %+v (FR-4)", i, got, held)
		}
	}

	// The unstopped leg, over exactly those frames, moved a FRAME — so "no frame
	// changed" is a statement about the stop and not about the fixture.
	moved := running.entityDraws()
	if len(moved) != len(held) {
		t.Fatalf("the unstopped leg carries %d entities, want %d", len(moved), len(held))
	}
	frameMoved := false
	for i := range moved {
		if moved[i].Frame != held[i].Frame {
			frameMoved = true
		}
	}
	if !frameMoved {
		t.Fatalf("no entity's frame moved on the unstopped leg over %d frames either, so the comparison "+
			"above discriminates nothing", frames)
	}

	// Clearing the stop puts frames back in motion, which is what says they were
	// held rather than exhausted.
	// Three ticks: the idle class's sub-frame alternates every scene tick, so one
	// would do, and three leaves no room for a coincidence about parity.
	stopped.setCadence(stoppedPeriodUS, false)
	for i := 0; i < 3; i++ {
		now = now.Add(period)
		if n, _ := stopped.paceTo(now); n != 1 {
			t.Fatalf("the resumed call ran %d ticks, want 1", n)
		}
	}
	if reflect.DeepEqual(stopped.entityDraws(), held) {
		t.Errorf("the pushed entities are unchanged after the resume, so the stop was not what held them")
	}
}
