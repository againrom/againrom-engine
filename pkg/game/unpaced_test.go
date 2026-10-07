package game

import (
	"bytes"
	"testing"
	"time"

	"againrom/pkg/render/terrain"
)

func newUnpacedWorld(t *testing.T) *mapWorld {
	t.Helper()
	m := worldFixtureMap()
	return mustOpenMapWorld(t, m, nil, worldFixtureUnitSet(), worldFixtureViewer(t, m))
}

func TestUnpacedOwnerLoopRunsExactlyOneTickPerCallWithoutADeadline(t *testing.T) {
	period := terrain.SpeedIndexPeriod(terrain.DefaultSpeedIndex)
	base := time.Unix(1_700_000_000, 0)
	mw := newUnpacedWorld(t)
	direct := newUnpacedWorld(t)

	beforeBytes := marshalWorld(t, mw.world)
	beforeHash := mw.world.Hash()
	mw.setCadenceMode(period, false, true, false)
	if !bytes.Equal(marshalWorld(t, mw.world), beforeBytes) || mw.world.Hash() != beforeHash {
		t.Fatal("selecting the driver arm changed canonical world bytes or hash")
	}

	// The first call, the same timestamp, a backwards timestamp and a day-long
	// stall each run one tick. A deadline loop would answer 0, 0, 0 and a
	// catch-up bound respectively, so every member discriminates the arm.
	instants := []time.Time{base, base, base.Add(-time.Hour), base.Add(24 * time.Hour)}
	for i, now := range instants {
		n, _ := mw.paceTo(now)
		if n != 1 {
			t.Fatalf("unpaced call %d ran %d ticks, want exactly 1", i+1, n)
		}
		direct.tick()
	}
	if got, want := mw.world.Hash(), direct.world.Hash(); got != want {
		t.Fatalf("four unpaced calls reached hash %#x, four direct ticks reached %#x", got, want)
	}
	if mw.world.Tick() != 4 || mw.scene != 4 {
		t.Fatalf("four calls left world tick %d scene %d, want 4 and 4", mw.world.Tick(), mw.scene)
	}
}

func TestUnpacedStopRunsNothingAndResumesOneCallAtATime(t *testing.T) {
	period := terrain.SpeedIndexPeriod(terrain.DefaultSpeedIndex)
	base := time.Unix(1_700_000_000, 0)
	mw := newUnpacedWorld(t)
	mw.setCadenceMode(period, true, true, false)

	beforeBytes := marshalWorld(t, mw.world)
	beforeHash := mw.world.Hash()
	for i, now := range []time.Time{base, base.Add(24 * time.Hour)} {
		if n, applied := mw.paceTo(now); n != 0 || applied != 0 {
			t.Fatalf("stopped unpaced call %d ran %d ticks and applied %d orders", i+1, n, applied)
		}
	}
	if !bytes.Equal(marshalWorld(t, mw.world), beforeBytes) || mw.world.Hash() != beforeHash {
		t.Fatal("stopped unpaced calls changed canonical world bytes or hash")
	}

	mw.setCadenceMode(period, false, true, false)
	if n, _ := mw.paceTo(base.Add(24 * time.Hour)); n != 1 {
		t.Fatalf("first eligible unpaced call after stop ran %d ticks, want 1", n)
	}
}

func TestRestoringPacedModeResetsPhaseAndBaselineWithoutRewinding(t *testing.T) {
	period := terrain.SpeedIndexPeriod(terrain.DefaultSpeedIndex)
	base := time.Unix(1_700_000_000, 0)
	mw := newUnpacedWorld(t)

	// Seed a real paced half-period before selecting unpaced. The clear command
	// must discard this phase as well as the time spent in the other arm.
	mw.paceTo(base)
	mw.paceTo(base.Add(time.Duration(period/2) * time.Microsecond))
	if got := mw.clock.Remainder(); got != period/2 {
		t.Fatalf("setup remainder = %d, want %d", got, period/2)
	}
	mw.setCadenceMode(period, false, true, false)
	far := base.Add(24 * time.Hour)
	if n, _ := mw.paceTo(far); n != 1 {
		t.Fatalf("unpaced setup call ran %d ticks, want 1", n)
	}
	count := mw.clock.Count()
	tick := mw.world.Tick()

	mw.setCadenceMode(period, false, false, true)
	if mw.unpaced || !mw.last.IsZero() || mw.clock.Remainder() != 0 {
		t.Fatalf("restored mode has unpaced=%v last=%v remainder=%d, want false/zero/zero",
			mw.unpaced, mw.last, mw.clock.Remainder())
	}
	if mw.clock.Count() != count || mw.world.Tick() != tick || mw.clock.Period() != period {
		t.Fatalf("restore rewound or re-rated state: clock count %d/%d world tick %d/%d period %d/%d",
			mw.clock.Count(), count, mw.world.Tick(), tick, mw.clock.Period(), period)
	}

	// The restore frame takes a baseline. Less than one full period after it is
	// still no tick; the boundary itself is exactly one, never a backlog.
	if n, _ := mw.paceTo(far); n != 0 {
		t.Fatalf("restore baseline ran %d ticks, want 0", n)
	}
	if n, _ := mw.paceTo(far.Add(time.Duration(period-1) * time.Microsecond)); n != 0 {
		t.Fatalf("restored period-1 ran %d ticks, want 0", n)
	}
	if n, _ := mw.paceTo(far.Add(time.Duration(period) * time.Microsecond)); n != 1 {
		t.Fatalf("one restored period ran %d ticks, want 1", n)
	}

	// The clear command's reset is an action, not only a transition. Pressing
	// minus while already paced still clears a newly accumulated half phase.
	half := far.Add(time.Duration(period+period/2) * time.Microsecond)
	if n, _ := mw.paceTo(half); n != 0 {
		t.Fatalf("paced half-period ran %d ticks, want 0", n)
	}
	count = mw.clock.Count()
	tick = mw.world.Tick()
	mw.setCadenceMode(period, false, false, true)
	if !mw.last.IsZero() || mw.clock.Remainder() != 0 {
		t.Fatalf("paced clear left last=%v remainder=%d, want zero/zero", mw.last, mw.clock.Remainder())
	}
	if mw.clock.Count() != count || mw.world.Tick() != tick {
		t.Fatalf("paced clear rewound count %d/%d or world tick %d/%d", mw.clock.Count(), count, mw.world.Tick(), tick)
	}
}
