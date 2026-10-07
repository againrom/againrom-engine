package game

import (
	"bytes"
	"image"
	"reflect"
	"testing"
	"time"

	"againrom/pkg/mapload"
	"againrom/pkg/render/terrain"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// cadenceTicks is how many ticks every leg runs. It is the length of
// cadenceStream's own table, so no leg reads past the stream's end and every
// compared index has an entry behind it.
//
// It is THIRTEEN CELLS' worth rather than thirteen ticks: this fixture's units
// take the constructor's default speed and pay a crossing per cell, and every
// order below is written to arrive. Scaled by the diagonal cost, which is the
// larger of the two, so a leg made of diagonal cells arrives inside the run as
// well as a straight one.
const cadenceTicks = 13 * diagonalCellTicks

// The two rates the legs are paced at, and their periods — hand-computed as
// 1000000/rate truncated, never read back off a clock.
const (
	cadenceSlowRate     = 16
	cadenceSlowPeriodUS = 62_500
	cadenceFastRate     = 256
	cadenceFastPeriodUS = 3_906
)

// cadenceStream is the ONE command stream every leg is advanced on: entity 0 and
// entity 1 given scripted targets at ticks 0, 4 and 8 over the fixture, and
// nothing at any other tick, the three CELLS apart rather than four ticks apart
// for the reason cadenceTicks is scaled. Every target stands inside the
// fixture's interior,
// so what a leg's walk answers to is the stream and not the ring.
//
// Entities 2 and 3 are named NOWHERE here, which is what lets the orders below
// join the stream by simple concatenation: the driver cuts scripted commands
// naming a commanded entity, and there are none to cut.
//
// Each entry is its own slice, freshly made per call, so no leg can write into
// another's schedule.
func cadenceStream() [][]sim.Command {
	out := make([][]sim.Command, cadenceTicks)
	out[0] = []sim.Command{{Entity: 0, X: 31, Y: 23}, {Entity: 1, X: 25, Y: 28}}
	out[4*diagonalCellTicks] = []sim.Command{{Entity: 0, X: 21, Y: 23}, {Entity: 1, X: 25, Y: 20}}
	out[8*diagonalCellTicks] = []sim.Command{{Entity: 0, X: 31, Y: 28}}
	return out
}

// cadenceIssue is one order, keyed by the TICK INDEX it is issued before rather
// than by an instant: at is the world tick the next step starts from, so the
// order joins that step's stream on every leg however that leg was paced.
type cadenceIssue struct {
	at     int
	entity uint32
	x, y   int
}

// cadenceIssues are the orders every leg issues, in issue order at each index.
// Both name entities cadenceStream names nowhere, and both targets are within
// reach of the ticks that remain — entity 2 needs seven cells of the eleven
// cells' worth left to it and entity 3 five of the seven — so each ARRIVES, and
// "the order moved something" is read off a cell rather than off a target
// field.
var cadenceIssues = []cadenceIssue{
	{at: 2 * diagonalCellTicks, entity: 2, x: 29, y: 27},
	{at: 6 * diagonalCellTicks, entity: 3, x: 26, y: 28},
}

// cadenceLeg is one driven leg: the driver, the instant its next call is made
// at, and the period it is paced by.
type cadenceLeg struct {
	name string
	mw   *mapWorld
	now  time.Time
	// step drives this leg exactly one tick and reports how many it ran, so
	// each leg carries its own ELAPSED SCHEDULE and not merely its own rate.
	step func(l *cadenceLeg) int
}

// TestOneCommandStreamReachesOneDigestAtEveryRateAndStopSchedule — 0041
// SC-8 (AC-8): one command stream advanced at 16 ticks a second, at 256, and
// through a stop-and-resume schedule reaches the same canonical byte form
// and the same digest AT EVERY TICK INDEX, and reaches a headless run's.
//
// The comparison is made at every index and not at the end, because a rate that
// applied the stream at the wrong tick arrives at the same place eventually;
// only the index-by-index comparison says WHEN each command was applied.
func TestOneCommandStreamReachesOneDigestAtEveryRateAndStopSchedule(t *testing.T) {
	base := time.Unix(1_700_000_000, 0)

	// The periods this file's whole schedule is computed from, asserted once so
	// every literal below has a stated provenance.
	if got := terrain.RatePeriod(cadenceSlowRate); got != cadenceSlowPeriodUS {
		t.Fatalf("rate %d is %d us, not the %d this run is written over", cadenceSlowRate, got, cadenceSlowPeriodUS)
	}
	if got := terrain.RatePeriod(cadenceFastRate); got != cadenceFastPeriodUS {
		t.Fatalf("rate %d is %d us, not the %d this run is written over", cadenceFastRate, got, cadenceFastPeriodUS)
	}

	m := worldFixtureMap()
	open := func(periodUS int) *mapWorld {
		mw := newMapWorld(mapload.FromALM(m), cadenceStream(), nil, worldFixtureViewer(t, m))
		mw.setCadence(periodUS, false)
		if n, applied := mw.paceTo(base); n != 0 || applied != 0 {
			t.Fatalf("a baseline call ran %d ticks and applied %d orders, want 0 and 0", n, applied)
		}
		return mw
	}

	us := func(n int) time.Duration { return time.Duration(n) * time.Microsecond }

	legs := []*cadenceLeg{
		{
			name: "slow (16/s, one call a tick)",
			mw:   open(cadenceSlowPeriodUS),
			now:  base,
			step: func(l *cadenceLeg) int {
				l.now = l.now.Add(us(cadenceSlowPeriodUS))
				n, _ := l.mw.paceTo(l.now)
				return n
			},
		},
		{
			name: "fast (256/s, two half-period calls a tick)",
			mw:   open(cadenceFastPeriodUS),
			now:  base,
			step: func(l *cadenceLeg) int {
				total := 0
				for _, part := range [2]int{1953, cadenceFastPeriodUS - 1953} {
					l.now = l.now.Add(us(part))
					n, _ := l.mw.paceTo(l.now)
					total += n
				}
				return total
			},
		},
		{
			name: "paused (16/s, stopped before every tick, then three uneven calls)",
			mw:   open(cadenceSlowPeriodUS),
			now:  base,
			step: func(l *cadenceLeg) int {
				// A stop of a different length before every tick, so no two ticks
				// of this leg are paced alike either.
				l.mw.setCadence(cadenceSlowPeriodUS, true)
				for i := 0; i < 1+int(l.mw.world.Tick())%5; i++ {
					l.now = l.now.Add(us(16_667))
					if n, applied := l.mw.paceTo(l.now); n != 0 || applied != 0 {
						t.Fatalf("a stopped frame ran %d ticks and applied %d orders, want 0 and 0", n, applied)
					}
				}
				l.mw.setCadence(cadenceSlowPeriodUS, false)

				total := 0
				for _, part := range [3]int{20_833, 20_833, cadenceSlowPeriodUS - 41_666} {
					l.now = l.now.Add(us(part))
					n, _ := l.mw.paceTo(l.now)
					total += n
				}
				return total
			},
		},
	}

	// The headless leg: the same map with no driver, no viewer, no queue and no
	// clock, advanced on the stream this file wrote out.
	headless := mapload.FromALM(m)
	stream := cadenceStream()
	untouched := headless.Hash()

	for _, l := range legs {
		if got, want := l.mw.world.Hash(), untouched; got != want {
			t.Fatalf("the %s leg starts at %#x and the headless run at %#x — every leg must begin equal",
				l.name, got, want)
		}
	}

	digests := make([]uint64, 0, cadenceTicks)
	for tk := 0; tk < cadenceTicks; tk++ {
		// The orders due before this step, issued through the shipped seam on
		// every driven leg and CONCATENATED onto the headless stream, in the same
		// issue order.
		cmds := append([]sim.Command(nil), stream[tk]...)
		var issued []sim.Command
		for _, o := range cadenceIssues {
			if o.at != tk {
				continue
			}
			for _, l := range legs {
				l.mw.enqueue(o.entity, o.x, o.y)
			}
			issued = append(issued, sim.Command{Entity: sim.EntityID(o.entity), X: int32(o.x), Y: int32(o.y)})
		}
		// THE SCRIPT'S OWN COMMANDS ARE NOT ORDERS and are not tagged: a
		// scheduled move-to is a scripted command, and only what leaves through
		// the front-end's seam is a group order. So the split is made here and
		// the issued half alone goes through the boundary's second copy.
		cmds = append(cmds, grouped(issued)...)

		for _, l := range legs {
			if n := l.step(l); n != 1 {
				t.Fatalf("the %s leg ran %d ticks reaching index %d, want 1 — this run compares one tick "+
					"per index", l.name, n, tk+1)
			}
			if got, want := l.mw.world.Tick(), uint64(tk+1); got != want {
				t.Fatalf("the %s leg stands at tick %d, want %d", l.name, got, want)
			}
		}
		sim.Step(headless, cmds)

		wantBytes := marshalWorld(t, headless)
		wantHash := headless.Hash()
		for _, l := range legs {
			if got := marshalWorld(t, l.mw.world); !bytes.Equal(got, wantBytes) {
				t.Fatalf("at tick %d the %s leg's canonical byte form differs from the headless run's — a "+
					"rate, a period, an accumulator, a baseline or a stop reached a world field (FR-8, P-2)",
					tk+1, l.name)
			}
			if got := l.mw.world.Hash(); got != wantHash {
				t.Fatalf("at tick %d the %s leg's digest %#x != the headless %#x (FR-8, P-1)",
					tk+1, l.name, got, wantHash)
			}
		}
		digests = append(digests, wantHash)
	}

	// Non-vacuity, three ways. The digest must actually have moved along the run,
	// the ordered entities must actually have been moved by their orders, and the
	// queues must actually have drained.
	moved := 0
	for i := 1; i < len(digests); i++ {
		if digests[i] != digests[i-1] {
			moved++
		}
	}
	if moved < cadenceTicks/2 {
		t.Errorf("the digest changed at only %d of %d indices, so the agreements above are largely "+
			"agreements between worlds that did nothing", moved, cadenceTicks-1)
	}
	if digests[len(digests)-1] == untouched {
		t.Error("the run ends on a never-stepped world's digest")
	}

	cells := entityCells(headless)
	for _, o := range cadenceIssues {
		if got, want := cells[o.entity], image.Pt(o.x, o.y); got != want {
			t.Errorf("entity %d stands on %v after the run, want the ordered cell %v — an order that moved "+
				"nothing leaves every comparison above an agreement about a world no order reached",
				o.entity, got, want)
		}
	}
	if !someEntityHasMoved(worldFixtureCells, cells) {
		t.Error("no entity left its start cell over the whole run")
	}
	for _, l := range legs {
		if got := len(l.mw.pending); got != 0 {
			t.Errorf("the %s leg holds %d orders after the run, want 0", l.name, got)
		}
	}

	if l0, l1 := legs[0].mw.clock.Period(), legs[1].mw.clock.Period(); l0 == l1 {
		t.Errorf("the slow and fast legs both ended on period %d us, so they were never paced differently", l0)
	}
	if legs[2].mw.stopped {
		t.Error("the paused leg ended stopped, so its last tick was not driven through a resume")
	}
}

func TestPacingReachesNoWorldFieldAndNoSeamState(t *testing.T) {
	base := time.Unix(1_700_000_000, 0)
	m := worldFixtureMap()
	mw := newMapWorld(mapload.FromALM(m), cadenceStream(), nil, worldFixtureViewer(t, m))
	mw.paceTo(base)

	// A few ticks first, so the state held still below is one the world walked
	// to rather than the one the constructor built.
	for i := 0; i < 5; i++ {
		mw.tick()
	}
	if !someEntityHasMoved(worldFixtureCells, entityCells(mw.world)) {
		t.Fatalf("no entity moved in the five ticks before the comparison: %v", entityCells(mw.world))
	}

	beforeTick := mw.world.Tick()
	beforeBounds := mw.world.Bounds()
	beforeBytes := marshalWorld(t, mw.world)
	beforeHash := mw.world.Hash()
	beforeDraws := mw.entityDraws()

	// EVERY RUNG of the cadence ladder, and the stop set and cleared at each of
	// them, with no advance anywhere behind any of it. Walked rung by rung
	// rather than by doubling a rate: the ladder the keys actually move is what
	// the driver has to be invariant under, and nine of its rungs are the
	// game's own speeds, whose truncated periods a doubling never reaches.
	periods := make(map[int]bool)
	for rung := terrain.CadenceRungMin; rung <= terrain.CadenceRungMax; rung++ {
		periodUS := terrain.CadencePeriod(rung)
		for _, stopped := range []bool{true, false} {
			mw.setCadence(periodUS, stopped)
			periods[mw.clock.Period()] = true

			if got := mw.world.Tick(); got != beforeTick {
				t.Fatalf("%d us stopped=%v moved the world's tick %d -> %d", periodUS, stopped, beforeTick, got)
			}
			if got := mw.world.Bounds(); got != beforeBounds {
				t.Fatalf("%d us stopped=%v moved the world's bounds", periodUS, stopped)
			}
			if got := marshalWorld(t, mw.world); !bytes.Equal(got, beforeBytes) {
				t.Fatalf("%d us stopped=%v moved the canonical byte form (FR-8, P-2)", periodUS, stopped)
			}
			if got := mw.world.Hash(); got != beforeHash {
				t.Fatalf("%d us stopped=%v moved the digest %#x -> %#x (FR-8, P-2)",
					periodUS, stopped, beforeHash, got)
			}
			if got := mw.entityDraws(); !sameDraws(got, beforeDraws) {
				t.Fatalf("%d us stopped=%v moved what the push carries", periodUS, stopped)
			}
		}
	}

	// The driver's own cadence really did move over all of that, and over the
	// whole ladder rather than part of it.
	if want := terrain.CadenceRungMax - terrain.CadenceRungMin + 1; len(periods) != want {
		t.Errorf("the ladder left the clock holding %d distinct periods, want one per rung (%d) — the "+
			"comparisons above are comparisons over a driver that was never re-rated", len(periods), want)
	}
	if mw.stopped {
		t.Error("the ladder ended stopped; the loop clears it last")
	}
}

// sameDraws compares two pushes field by field. MapEntity stopped being
// comparable when it gained a route, so this is a deep walk — which is
// what the comparison meant all along and is now strictly stronger, since
// two entries agreeing in every scalar but heading down different routes are
// no longer the same push.
func sameDraws(a, b []ui.MapEntity) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if !reflect.DeepEqual(a[i], b[i]) {
			return false
		}
	}
	return true
}
