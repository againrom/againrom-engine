package game

import (
	"bytes"
	"image"
	"testing"
	"time"

	"againrom/pkg/mapload"
	"againrom/pkg/render/terrain"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// drawnTicks is how many ticks every leg runs. It is the length of drawnStream's
// own table, so no leg reads past the stream's end and every compared index has
// an entry behind it.
//
// It is THIRTEEN CELLS' worth rather than thirteen ticks, scaled by what a
// diagonal cell costs this fixture's units: a mover now crosses a cell over many
// ticks, and every order below is written to arrive inside the run.
const drawnTicks = 13 * diagonalCellTicks

// The rate every leg is paced at, and its period — hand-computed as 1000000/rate
// truncated, never read back off a clock. It divides by five exactly, which is
// what lets a leg place four frames inside a tick at whole microseconds.
const (
	drawnRate     = 16
	drawnPeriodUS = 62_500
)

// drawnStream is the ONE command stream every leg is advanced on: entity 0 and
// entity 1 given scripted targets over the fixture, and nothing at any other
// tick. Every target stands inside the fixture's interior, so what a leg's walk
// answers to is the stream and not the ring.
//
// Somebody is MID-STRIDE at nearly every index: entity 0 crosses east along row
// 23 and turns back, entity 1 walks north along column 25 and back. An entity
// standing still all run would make "the displacement moved no world field" a
// statement about a displacement that was always zero. Neither walk crosses the
// other's line, so no contention decides where either one ends up.
//
// ENTITY 2 IS NAMED NOWHERE, here or below, and that is deliberate: it is the
// idle-cycle class, so it is the at-rest entity whose frame cadence the ledger
// reads.
//
// IT ALSO CARRIES A DEATH (SC-8), and it carries one because the death clock
// is state the DRAWING side gained: a run in which nothing dies leaves that
// memory empty and every comparison here an agreement about a world whose
// new front-end state was never written. Entity 1 takes a blow it survives,
// then one that leaves it at exactly zero, then a kill — the three life
// states in one run, and the clock stamped in the middle of it. Entity 0
// walks throughout, so somebody is still mid-stride after entity 1 stops.
//
// Each entry is its own slice, freshly made per call, so no leg can write into
// another's schedule.
func drawnStream() [][]sim.Command {
	out := make([][]sim.Command, drawnTicks)
	out[0] = []sim.Command{{Entity: 0, X: 31, Y: 23}, {Entity: 1, X: 25, Y: 14}}
	out[3*diagonalCellTicks] = []sim.Command{{Kind: sim.KindDamage, Entity: 1, X: 60}}
	out[5*diagonalCellTicks] = []sim.Command{{Entity: 0, X: 21, Y: 23}, {Entity: 1, X: 25, Y: 20}}
	// 48 and not 40, since 0109. Entity 1 comes off this fixture's map with a
	// health regeneration period of 100 against a maximum of 100, so a
	// qualifying full tick returns 100*2*100/100 = 200 hundredths — two whole
	// points — and four of them fall between the blow above and this one
	// (sub-ticks 140, 204, 268 and 332 of the run, the ones whose index is 12
	// modulo 64). It stands at 48 rather than 40 when this blow lands, and the
	// blow is what leaves it at EXACTLY zero: that is the state the life-state
	// non-vacuity check at the foot of this file requires to be crossed, and it
	// is the whole reason this number is written out rather than shared with the
	// 60 above. What moved is the fixture, not the seam — regeneration is
	// another story's clock, and this file's subject is the drawing.
	out[9*diagonalCellTicks] = []sim.Command{{Kind: sim.KindDamage, Entity: 1, X: 48}}
	out[11*diagonalCellTicks] = []sim.Command{{Kind: sim.KindKill, Entity: 1}}
	return out
}

// drawnLife is the three life states transcribed HERE, off the world's own two
// integers, and never a call into the seam's own classifier — which is one of
// the things this file exists to pin.
func drawnLife(e sim.Entity) uint8 {
	switch {
	case e.HP < 0:
		return ui.LifeDead
	case e.HP == 0 && e.MaxHP > 0:
		return ui.LifeDowned
	default:
		return ui.LifeAlive
	}
}

// drawnIssue is one order, keyed by the TICK INDEX it is issued before rather
// than by an instant: at is the world tick the next step starts from, so the
// order joins that step's stream on every leg however that leg was driven.
type drawnIssue struct {
	at     int
	entity uint32
	x, y   int
}

// drawnIssues are the orders every leg issues. Both name entity 3, which
// drawnStream names nowhere, so the driver's exclusion removes nothing; and each
// target is within reach of the ticks that remain, so each ARRIVES and "the
// order moved something" is read off a cell rather than off a target field. The
// second supersedes the first, which is why only the last one's cell is asserted
// at the end of the run.
var drawnIssues = []drawnIssue{
	{at: 1 * diagonalCellTicks, entity: 3, x: 31, y: 29},
	{at: 6 * diagonalCellTicks, entity: 3, x: 28, y: 29},
}

// drawnLeg is one driven leg: the driver, the instant its next front-end call is
// made at, and how one tick's worth of front-end calls is made.
//
// frames, phases and mid are the DRAWING LEDGER, and they are what keeps the
// drawn leg from being the undrawn one with extra calls: how many front-end
// frames it made, which distinct phase pairs its renderer was handed, and how
// many of those frames were drawn with somebody mid-stride. A leg that drew once
// a tick, or always at the same remainder, or only ever over units standing
// still, is a leg over which "whatever was drawn between the ticks" says nothing.
type drawnLeg struct {
	name string
	mw   *mapWorld
	now  time.Time
	// tick drives this leg through exactly one world tick, however many
	// front-end calls that takes, and reports how many ticks fired.
	tick func(l *drawnLeg) int

	frames int
	phases map[[2]int]bool
	mid    int
}

// draw is ONE front-end frame of a leg: the paced call the front-end makes, and
// then the read a renderer makes of what that call pushed — the snapshot it
// draws from and the phase it draws it at.
//
// The read goes through the same two surfaces the window tier draws from, and
// its results are RECORDED rather than discarded: the whole point of this leg is
// that the reading happens on every frame, and a call whose result is dropped is
// a call a compiler is free to notice.
func (l *drawnLeg) draw(elapsed time.Duration) int {
	l.now = l.now.Add(elapsed)
	n, _ := l.mw.paceTo(l.now)

	l.frames++
	e, p := l.mw.view.Phase()
	l.phases[[2]int{e, p}] = true
	for _, d := range l.mw.entityDraws() {
		if d.Step != (image.Point{}) {
			l.mid++
			break
		}
	}
	return n
}

// TestOneCommandStreamReachesOneDigestWhateverIsDrawnBetweenItsTicks —
// 0047 SC-7 (AC-8): one command stream advanced over a driver whose renderer
// is fed at every tick reaches the same canonical byte form and the same
// digest AT EVERY TICK INDEX as a headless run over that same stream.
//
// The comparison is made at every index and not at the end, because a driver
// that applied the stream at the wrong tick arrives at the same place
// eventually; only the index-by-index comparison says WHEN each command was
// applied.
func TestOneCommandStreamReachesOneDigestWhateverIsDrawnBetweenItsTicks(t *testing.T) {
	base := time.Unix(1_700_000_000, 0)

	// The period this file's whole frame schedule is computed from, asserted
	// once so every literal below has a stated provenance.
	if got := terrain.RatePeriod(drawnRate); got != drawnPeriodUS {
		t.Fatalf("rate %d is %d us, not the %d this run is written over", drawnRate, got, drawnPeriodUS)
	}

	m := worldFixtureMap()
	us := func(n int) time.Duration { return time.Duration(n) * time.Microsecond }
	open := func() *mapWorld {
		mw := newMapWorld(mapload.FromALM(m), drawnStream(), worldFixtureUnitSet(), worldFixtureViewer(t, m))
		mw.setCadence(drawnPeriodUS, false)
		if n, applied := mw.paceTo(base); n != 0 || applied != 0 {
			t.Fatalf("a baseline call ran %d ticks and applied %d orders, want 0 and 0", n, applied)
		}
		return mw
	}

	// THE TWO LEGS DIFFER IN EXACTLY WHAT AC-8 NAMES and in nothing else: the
	// same map, the same rate and the same stream, run with drawing driven at
	// many remainders and again with none. So a difference between them can only
	// be the drawing.
	//
	// The drawn leg makes five front-end calls a tick at a fifth of the period
	// each, four of which fire no tick at all: those four are the frames that
	// exist only because a picture is drawn BETWEEN two advances, and each is
	// handed its own remainder — 12500, 25000, 37500, 50000, then 0 as the fifth
	// crosses.
	legs := []*drawnLeg{
		{
			name:   "one call a tick, nothing drawn between",
			mw:     open(),
			now:    base,
			phases: map[[2]int]bool{},
			tick: func(l *drawnLeg) int {
				l.now = l.now.Add(us(drawnPeriodUS))
				n, _ := l.mw.paceTo(l.now)
				return n
			},
		},
		{
			name:   "drawn at many remainders (five frames a tick)",
			mw:     open(),
			now:    base,
			phases: map[[2]int]bool{},
			tick: func(l *drawnLeg) int {
				total := 0
				for i := 0; i < 5; i++ {
					total += l.draw(us(drawnPeriodUS / 5))
				}
				return total
			},
		},
	}

	// The headless leg: the same map with no driver, no viewer, no renderer and
	// no clock, advanced on the stream this file wrote out.
	headless := mapload.FromALM(m)
	stream := drawnStream()
	untouched := headless.Hash()

	for _, l := range legs {
		if got, want := l.mw.world.Hash(), untouched; got != want {
			t.Fatalf("the %q leg starts at %#x and the headless run at %#x — every leg must begin equal",
				l.name, got, want)
		}
	}

	// The push ledger, read off every driven leg at every tick: whether any
	// entity ever resolved to art, and every distinct frame the AT-REST
	// idle-cycle entity held.
	resolved := make(map[string]int, len(legs))
	lives := map[uint8]int{}
	idleFrames := make(map[string]map[*terrain.StaticFrame]bool, len(legs))
	for _, l := range legs {
		idleFrames[l.name] = map[*terrain.StaticFrame]bool{}
	}

	digests := make([]uint64, 0, drawnTicks)
	for tk := 0; tk < drawnTicks; tk++ {
		// The orders due before this step, issued through the shipped seam on
		// every driven leg and CONCATENATED onto the headless stream, in the
		// same issue order.
		cmds := append([]sim.Command(nil), stream[tk]...)
		var issued []sim.Command
		for _, o := range drawnIssues {
			if o.at != tk {
				continue
			}
			for _, l := range legs {
				l.mw.enqueue(o.entity, o.x, o.y)
			}
			issued = append(issued, sim.Command{Entity: sim.EntityID(o.entity), X: int32(o.x), Y: int32(o.y)})
		}
		// THE SCRIPT'S OWN COMMANDS ARE NOT ORDERS and are not tagged: only what
		// leaves through the front-end's seam is a group order.
		cmds = append(cmds, grouped(issued)...)

		for _, l := range legs {
			if n := l.tick(l); n != 1 {
				t.Fatalf("the %q leg ran %d ticks reaching index %d, want 1 — this run compares one tick "+
					"per index", l.name, n, tk+1)
			}
			if got, want := l.mw.world.Tick(), uint64(tk+1); got != want {
				t.Fatalf("the %q leg stands at tick %d, want %d", l.name, got, want)
			}
		}
		sim.Step(headless, cmds)

		wantBytes := marshalWorld(t, headless)
		wantHash := headless.Hash()
		for _, l := range legs {
			if got := marshalWorld(t, l.mw.world); !bytes.Equal(got, wantBytes) {
				t.Fatalf("at tick %d the %q leg's canonical byte form differs from the headless run's — "+
					"something the front-end holds reached a world field (FR-7, P-1)", tk+1, l.name)
			}
			if got := l.mw.world.Hash(); got != wantHash {
				t.Fatalf("at tick %d the %q leg's digest %#x != the headless %#x (FR-7, P-1)",
					tk+1, l.name, got, wantHash)
			}
			assertPushSurvives(t, l, tk+1, resolved, idleFrames[l.name], lives)
		}
		digests = append(digests, wantHash)
	}

	// Non-vacuity, four ways. The digest must actually have moved along the run,
	// the ordered entities must actually have been moved by their orders, the
	// queues must actually have drained, and the renderer must actually have
	// been fed something that resolves and moves.
	moved := 0
	for i := 1; i < len(digests); i++ {
		if digests[i] != digests[i-1] {
			moved++
		}
	}
	if moved < drawnTicks/2 {
		t.Errorf("the digest changed at only %d of %d indices, so the agreements above are largely "+
			"agreements between worlds that did nothing", moved, drawnTicks-1)
	}
	if digests[len(digests)-1] == untouched {
		t.Error("the run ends on a never-stepped world's digest")
	}

	cells := entityCells(headless)
	last := map[uint32]drawnIssue{}
	for _, o := range drawnIssues {
		last[o.entity] = o
	}
	for e, o := range last {
		if got, want := cells[e], image.Pt(o.x, o.y); got != want {
			t.Errorf("entity %d stands on %v after the run, want the ordered cell %v — an order that moved "+
				"nothing leaves every comparison above an agreement about a world no order reached",
				e, got, want)
		}
	}
	if !someEntityHasMoved(worldFixtureCells, cells) {
		t.Error("no entity left its start cell over the whole run")
	}
	for _, l := range legs {
		if got := len(l.mw.pending); got != 0 {
			t.Errorf("the %q leg holds %d orders after the run, want 0", l.name, got)
		}
	}
	for _, l := range legs {
		if got := resolved[l.name]; got != drawnTicks {
			t.Errorf("the %q leg's push carried art at %d of %d ticks, want all of them — a renderer "+
				"handed nothing satisfies every comparison above trivially", l.name, got, drawnTicks)
		}
		if got := len(idleFrames[l.name]); got < 2 {
			t.Errorf("the %q leg's at-rest entity held %d distinct frame(s) over the run, want at least 2 "+
				"— the frame cadence of an entity at rest must survive this story", l.name, got)
		}
	}

	// AC-8's own non-vacuity: the drawn leg must actually have been DRAWN, at
	// MANY remainders, over units that were actually mid-stride. Without it the
	// two legs above could have been one leg run twice.
	drawn := legs[len(legs)-1]
	if got, want := drawn.frames, 5*drawnTicks; got != want {
		t.Errorf("the %q leg made %d front-end frames over %d ticks, want %d", drawn.name, got, drawnTicks, want)
	}
	if got := len(drawn.phases); got < 5 {
		t.Errorf("the %q leg's renderer was handed %d distinct phase(s); it must be drawn at many "+
			"remainders for AC-8 to be about drawing at all", drawn.name, got)
	}
	if got := drawn.mid; got < drawn.frames/2 {
		t.Errorf("only %d of the %q leg's %d frames were drawn with somebody mid-stride, so most of the "+
			"drawing compared above carried no displacement", got, drawn.name, drawn.frames)
	}
	if legs[0].frames != 0 {
		t.Errorf("the undrawn leg made %d front-end frame reads, want none", legs[0].frames)
	}

	// The death's own non-vacuity (SC-8). All three life states must have
	// crossed the seam over the run, and the DEATH CLOCK — the one piece of
	// front-end state this story adds — must actually have been written on
	// every driven leg, or the digest agreements above are agreements about a
	// memory that stayed empty.
	for _, want := range []uint8{ui.LifeAlive, ui.LifeDowned, ui.LifeDead} {
		if lives[want] == 0 {
			t.Errorf("no entity crossed the seam in life state %d over the run — the stream is "+
				"meant to carry all three", want)
		}
	}
	for _, l := range legs {
		if got := len(l.mw.died); got != 1 {
			t.Errorf("the %q leg's death clock holds %d entries, want exactly 1 — one entity stops "+
				"being alive in this run and it is stamped once", l.name, got)
		}
	}
	// And the clock is the front end's alone: the headless run has no such
	// memory, and its byte form and digest are what every leg above matched.
	if got := len(marshalWorld(t, headless)); got == 0 {
		t.Fatal("the headless world marshalled to nothing")
	}
}

// TestAStoppedWorldIsDrawnFromOneSnapshotAtOnePhaseInEveryFrame — 0047
// SC-6's stopped arm and SC-7's repeatability clause (AC-7): over a world
// stopped MID-STRIDE, however many frames are drawn, the two things a frame
// is drawn from do not move — the snapshot and the phase — and neither
// does the world behind them.
//
// THAT IS THE WHOLE OF WHAT THIS TIER CAN SAY, and it is said rather than
// overstated. The geometry is the window tier's, and that it is a function of
// those two inputs alone is measured there; what is measured here is that the
// inputs do not move, which is the half a stop can break.
func TestAStoppedWorldIsDrawnFromOneSnapshotAtOnePhaseInEveryFrame(t *testing.T) {
	base := time.Unix(1_700_000_000, 0)
	m := worldFixtureMap()
	v := worldFixtureViewer(t, m)
	mw := newMapWorld(mapload.FromALM(m), drawnStream(), worldFixtureUnitSet(), v)
	mw.setCadence(drawnPeriodUS, false)

	us := func(n int) time.Duration { return time.Duration(n) * time.Microsecond }
	now := base
	mw.paceTo(now)

	// Run until the world is caught mid-stride with the frame standing part-way
	// through a tick: a stop taken on a tick boundary, or over units that have
	// not moved, holds a picture with no displacement in it to hold.
	for i := 0; i < 12; i++ {
		now = now.Add(us(drawnPeriodUS / 5))
		mw.paceTo(now)
	}
	movers := 0
	for _, d := range mw.entityDraws() {
		if d.Step != (image.Point{}) {
			movers++
		}
	}
	if e, _ := v.Phase(); movers == 0 || e == 0 {
		t.Fatalf("the run reached %d mover(s) at remainder %d; this criterion is stated over a unit caught "+
			"mid-stride part-way through a tick", movers, e)
	}

	mw.setCadence(drawnPeriodUS, true)
	wantDraws := mw.entityDraws()
	wantE, wantP := v.Phase()
	wantBytes := marshalWorld(t, mw.world)
	wantHash := mw.world.Hash()
	wantTick := mw.world.Tick()

	for i := 0; i < 20; i++ {
		now = now.Add(us(drawnPeriodUS / 3))
		if n, applied := mw.paceTo(now); n != 0 || applied != 0 {
			t.Fatalf("stopped frame %d ran %d ticks and applied %d orders, want 0 and 0", i, n, applied)
		}
		if e, p := v.Phase(); e != wantE || p != wantP {
			t.Fatalf("stopped frame %d is drawn at phase (%d, %d), want the held (%d, %d) — a stopped "+
				"world's picture must not move (FR-6, P-2)", i, e, p, wantE, wantP)
		}
		if got := mw.entityDraws(); !sameDraws(got, wantDraws) {
			t.Fatalf("stopped frame %d is drawn from a different snapshot:\n %+v\n %+v", i, got, wantDraws)
		}
		if got := mw.world.Tick(); got != wantTick {
			t.Fatalf("stopped frame %d left the world at tick %d, want %d", i, got, wantTick)
		}
		if got := marshalWorld(t, mw.world); !bytes.Equal(got, wantBytes) {
			t.Fatalf("stopped frame %d moved the canonical byte form (FR-7, P-1)", i)
		}
		if got := mw.world.Hash(); got != wantHash {
			t.Fatalf("stopped frame %d moved the digest %#x -> %#x (FR-7, P-1)", i, wantHash, got)
		}
	}

	// And the stop really was holding something back: the next frame after it
	// clears is drawn at a different phase, out of a world that advanced.
	mw.setCadence(drawnPeriodUS, false)
	now = now.Add(us(drawnPeriodUS + drawnPeriodUS/5))
	if n, _ := mw.paceTo(now); n == 0 {
		t.Fatal("the frame after the resume fired no tick, so the comparisons above are comparisons over a " +
			"driver that was never going to advance")
	}
	if e, _ := v.Phase(); e == wantE {
		t.Errorf("the frame after the resume is still drawn at remainder %d", e)
	}
}

// assertPushSurvives reads the driven leg's snapshot at one tick and pins the
// four things about it that must come through this story unchanged: the
// snapshot's order, its art resolution, the life byte and health pair it carries
// across, and — through the caller's ledger — the frame cadence of an entity at
// rest.
//
// It reads the seam's own derivation rather than the viewer, because what the
// viewer holds is that derivation's own slice: the push adopted its twin this
// very tick.
func assertPushSurvives(t *testing.T, l *drawnLeg, tick int, resolved map[string]int, idle map[*terrain.StaticFrame]bool, lives map[uint8]int) {
	t.Helper()

	draws := l.mw.entityDraws()
	ents := l.mw.world.Entities()
	if len(draws) != len(ents) {
		t.Fatalf("at tick %d the %q leg pushes %d entries for %d entities", tick, l.name, len(draws), len(ents))
	}
	if got, want := l.mw.view.EntityMarkers(), len(ents); got != want {
		t.Fatalf("at tick %d the %q leg's viewer holds %d entries, want %d — the driven side must carry a "+
			"renderer for these comparisons to witness anything", tick, l.name, got, want)
	}

	all := true
	for i, d := range draws {
		e := ents[i]
		if d.ID != uint32(e.ID) {
			t.Fatalf("at tick %d entry %d carries id %d, want %d — the snapshot is in the world's own order",
				tick, i, d.ID, e.ID)
		}
		if d.Cell != (image.Point{X: int(e.X), Y: int(e.Y)}) {
			t.Fatalf("at tick %d entity %d is pushed on %v and stands on (%d,%d)", tick, e.ID, d.Cell, e.X, e.Y)
		}
		// The stream carries two blows and a kill, so the life byte is not a
		// constant here: it is compared against this file's own transcription
		// of the three states, read off the world's two integers.
		if d.Life != drawnLife(e) || d.HP != int(e.HP) || d.MaxHP != int(e.MaxHP) {
			t.Fatalf("at tick %d entity %d crosses as life %d %d/%d, want %d %d/%d",
				tick, e.ID, d.Life, d.HP, d.MaxHP, drawnLife(e), e.HP, e.MaxHP)
		}
		lives[d.Life]++
		if d.Art == nil || d.Frame == nil {
			all = false
		}
	}
	if all {
		resolved[l.name]++
	}
	// Entity 2 is the idle-cycle class, and nothing in this file's stream or its
	// orders ever names it, so its frame moving is the scene clock and nothing
	// else.
	if got, want := draws[2].Cell, worldFixtureCells[2]; got != want {
		t.Fatalf("at tick %d the at-rest entity stands on %v, want %v — it must be at rest for the "+
			"cadence below to be an at-rest cadence", tick, got, want)
	}
	if draws[2].Frame != nil {
		idle[draws[2].Frame] = true
	}
}
