package game

import (
	"bytes"
	"image"
	"reflect"
	"testing"
	"time"

	"againrom/pkg/render/camera"
	"againrom/pkg/render/terrain"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// The hand-built field. Nine by five is wide enough that every unit has several
// ticks of walking ahead of it and small enough to read as a picture:
//
//	. . . . . . . . .      2 starts at (0,0)
//	. . . . . . . . .      14 at (2,2), five cells from the target
//	5 . 14. . . . T .      5 at (0,2), directly behind it
//	. . . . . . . . .      9 at (0,4)
//	9 . . . . . . . .      T is (7,2)
const (
	groupW, groupH             = 9, 5
	groupTargetX, groupTargetY = 7, 2
)

// groupIDs are SPARSE and ascending, so nothing that answered with a slice
// position instead of an id agrees at more than one of them, and groupStart is
// the cell each of them stands on.
var (
	groupIDs   = []uint32{2, 5, 9, 14}
	groupStart = []image.Point{{X: 0, Y: 0}, {X: 0, Y: 2}, {X: 0, Y: 4}, {X: 2, Y: 2}}
)

// groupWorld is that field as a world: no passability grid, so every cell is
// enterable, and no target on any entity.
func groupWorld(t *testing.T) *sim.World {
	t.Helper()
	ents := make([]sim.Entity, len(groupIDs))
	for i := range groupIDs {
		ents[i] = sim.Entity{ID: sim.EntityID(groupIDs[i]),
			X: int32(groupStart[i].X), Y: int32(groupStart[i].Y)}
	}
	w, err := sim.NewWorld(0x30, sim.Bounds{Width: groupW, Height: groupH}, sim.ModeCanonical, nil, ents)
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	return w
}

// groupViewer is a viewer over the same extent, assembled here so this file
// needs no map stream: the driver pushes its entities into one every tick and is
// never handed a nil.
func groupViewer(t *testing.T) *ui.Viewer {
	t.Helper()
	v, err := ui.NewViewer("group", terrain.Grid{
		Width:  groupW,
		Height: groupH,
		Tiles:  make([]uint16, groupW*groupH),
	}, &terrain.Tileset{})
	if err != nil {
		t.Fatalf("NewViewer: %v", err)
	}
	return v
}

// groupDriver is a map screen's world over that field: an EMPTY schedule, so
// nothing but the orders this file issues ever moves anything.
func groupDriver(t *testing.T) *mapWorld {
	t.Helper()
	mw := newMapWorld(groupWorld(t), nil, nil, groupViewer(t))
	if len(mw.sched) != 0 || len(mw.pending) != 0 {
		t.Fatalf("fixture: the driver starts with %d schedule entries and %d pending orders, want 0 and 0",
			len(mw.sched), len(mw.pending))
	}
	return mw
}

// groupCommands is the command stream one left click on the target produces:
// one GROUP move-to per selected unit, all naming that cell and all under ONE
// tag, in ascending entity id — written out here as literals, which is what lets
// the headless leg stand independent of the driver.
//
// One tag over the whole press is the claim: the simulation reads commands of
// this kind sharing a tag inside one advance as ONE order, so a press over a
// selection is one group order and not that many single ones. The tag's VALUE is
// not the driver's and is compared nowhere — a tag reaches no world field, so
// what has to agree is the partition.
func groupCommands() []sim.Command {
	cmds := make([]sim.Command, 0, len(groupIDs))
	for _, id := range groupIDs {
		cmds = append(cmds, sim.Command{Kind: sim.KindGroupMoveTo,
			Entity: sim.EntityID(id), X: groupTargetX, Y: groupTargetY})
	}
	return cmds
}

// enqueueGroup issues that same stream through the seam the front-end holds, one
// call per order, in the order the map arm makes them.
func enqueueGroup(mw *mapWorld) {
	for _, id := range groupIDs {
		mw.enqueue(id, groupTargetX, groupTargetY)
	}
}

// cellsByID reads a world's entities back as id -> cell, through the seam's own
// derivation, so this file holds no second copy of the entity-to-cell convention.
func cellsByID(w *sim.World) map[uint32]image.Point {
	out := map[uint32]image.Point{}
	for _, d := range seamDraws(w, nil) {
		out[d.ID] = d.Cell
	}
	return out
}

// assertNoSharedCell fails unless every entity the world holds stands on a cell
// of its own. It is called after EVERY tick rather than at the end, because two
// units passing through one cell mid-run and parting again would be invisible to
// a check made only at quiescence.
func assertNoSharedCell(t *testing.T, w *sim.World, tick int) {
	t.Helper()
	seen := map[image.Point]uint32{}
	for id, c := range cellsByID(w) {
		if other, dup := seen[c]; dup {
			t.Fatalf("tick %d: entities %d and %d both stand on %v", tick, other, id, c)
		}
		seen[c] = id
	}
}

// groupQuiescent reports whether no entity still holds a target: every one has
// either arrived or given up.
func groupQuiescent(w *sim.World) bool {
	for _, e := range w.Entities() {
		if e.HasTarget {
			return false
		}
	}
	return true
}

// groupMaxTicks bounds the run. Arrival takes single-digit ticks on this field
// and the refusal that follows it is spent after stallLimit further ones, so a
// run that has not settled by here has not settled at all.
const groupMaxTicks = 120

// TestAGroupOrderedToOneCellKeepsItsShape — 0059 replaces the test that stood
// here, and the replacement is total. What used to be asserted was that a group
// sent to one cell ARRIVES ONE UNIT DEEP: one unit on the target, the rest
// abandoning the order where the search refused them, recorded in this file as a
// divergence that was "OURS AND WORSE".
//
// It is worse no longer, and the old assertion is not weakened but WRONG. A group
// order distributes: each member is sent to the ordered cell offset by its own
// displacement from the group's centroid, so four members are sent to four cells,
// none of them contends for another's, and every one of them arrives. The
// fixture and the shared-cell invariant are unchanged; what changed is the
// behaviour the world has.
//
// The four destinations are written out rather than derived here, so a shifted
// centroid or a dropped offset is a wrong number in this file and not a quietly
// different walk.
func TestAGroupOrderedToOneCellKeepsItsShape(t *testing.T) {
	mw := groupDriver(t)
	target := image.Pt(groupTargetX, groupTargetY)

	// The centroid of (0,0) (0,2) (0,4) (2,2) is (1,2), so the offsets are
	// (-1,-2) (-1,0) (-1,2) (+1,0) and the destinations are these.
	want := map[uint32]image.Point{
		2:  {X: 6, Y: 0},
		5:  {X: 6, Y: 2},
		9:  {X: 6, Y: 4},
		14: {X: 8, Y: 2},
	}

	for i, c := range groupStart {
		if c == target {
			t.Fatalf("fixture: entity %d already stands on the target %v", groupIDs[i], target)
		}
	}

	enqueueGroup(mw)
	if got := len(mw.pending); got != len(groupIDs) {
		t.Fatalf("the queue holds %d orders after %d enqueues, want %d", got, len(groupIDs), len(groupIDs))
	}
	if applied := mw.tick(); applied != len(groupIDs) {
		t.Fatalf("the applying advance applied %d orders, want %d", applied, len(groupIDs))
	}
	assertNoSharedCell(t, mw.world, 1)

	for _, e := range mw.world.Entities() {
		if !e.HasTarget {
			t.Errorf("entity %d holds no target at the advance that applied the orders", e.ID)
			continue
		}
		w := want[uint32(e.ID)]
		if e.TargetX != int32(w.X) || e.TargetY != int32(w.Y) {
			t.Errorf("entity %d carries target (%d,%d), want %v — the ordered cell %v offset by its own "+
				"displacement from the group's centroid", e.ID, e.TargetX, e.TargetY, w, target)
		}
		if e.Stall != 0 {
			t.Errorf("entity %d carries stall count %d at the applying advance, want 0", e.ID, e.Stall)
		}
	}

	ticks := 1
	for ; ticks < groupMaxTicks && !groupQuiescent(mw.world); ticks++ {
		mw.tick()
		assertNoSharedCell(t, mw.world, ticks+1)
	}
	if !groupQuiescent(mw.world) {
		t.Fatalf("the world still holds a target after %d ticks", ticks)
	}

	// EVERY member arrives, which is the whole of what the distribution buys: no
	// two of them were ever sent to one cell, so no search was ever refused by a
	// companion standing on the goal.
	cells := cellsByID(mw.world)
	for _, id := range groupIDs {
		if got := cells[id]; got != want[id] {
			t.Errorf("entity %d came to rest on %v, want %v — the formation did not survive the walk",
				id, got, want[id])
		}
	}
	t.Logf("observation: the four members settled after %d ticks, each on its own distributed cell", ticks)
}

// TestAGroupOrdersDigestFollowsAHeadlessRunAtEveryTick — 0030 SC-8 (AC-9,
// AC-10): the driver's world matches, at EVERY tick, a headless run over the
// command stream this file writes out — and two runs of that one script
// agree with each other tick by tick, byte form and digest alike.
//
// The unordered leg is what says the agreements are not agreements between runs
// that did nothing: from the applying tick on, the ordered worlds must differ
// from it.
func TestAGroupOrdersDigestFollowsAHeadlessRunAtEveryTick(t *testing.T) {
	const orderAt = 3

	first := groupDriver(t)
	second := groupDriver(t)
	quiet := groupDriver(t)
	headless := groupWorld(t)

	if a, b, h := first.world.Hash(), second.world.Hash(), headless.Hash(); a != h || b != h {
		t.Fatalf("the legs start at %#x, %#x and %#x — they must begin equal", a, b, h)
	}

	for tk := 0; tk < 40; tk++ {
		var cmds []sim.Command
		if tk == orderAt {
			enqueueGroup(first)
			enqueueGroup(second)
			cmds = groupCommands()
		}

		wantApplied := len(cmds)
		if got := first.tick(); got != wantApplied {
			t.Fatalf("tick %d: the first leg applied %d orders, want %d", tk, got, wantApplied)
		}
		if got := second.tick(); got != wantApplied {
			t.Fatalf("tick %d: the second leg applied %d orders, want %d", tk, got, wantApplied)
		}
		quiet.tick()
		sim.Step(headless, cmds)

		firstBytes := marshalWorld(t, first.world)
		if got := marshalWorld(t, second.world); !bytes.Equal(got, firstBytes) {
			t.Fatalf("after tick %d two runs of one script hold different byte forms", tk)
		}
		if a, b := first.world.Hash(), second.world.Hash(); a != b {
			t.Fatalf("after tick %d two runs of one script hold digests %#x and %#x", tk, a, b)
		}
		if got := marshalWorld(t, headless); !bytes.Equal(got, firstBytes) {
			t.Fatalf("after tick %d the driver's byte form differs from a headless run over the "+
				"stream this file wrote out — the advance did not run on that stream, at that tick", tk)
		}
		if a, h := first.world.Hash(), headless.Hash(); a != h {
			t.Fatalf("after tick %d the driver digest %#x != the headless digest %#x", tk, a, h)
		}
		if tk >= orderAt && first.world.Hash() == quiet.world.Hash() {
			t.Fatalf("after tick %d the ordered legs hold the unordered run's digest %#x, so their "+
				"agreeing says nothing about what the orders did", tk, quiet.world.Hash())
		}
	}
}

// TestAGroupAdvanceIsOnePerPacedCallAtZeroOneAndManyOrders — 0030 SC-8
// (AC-10): the advance a map-screen tick makes is one, whatever the queue
// held — and the count of orders it drained is the queue's length at that
// advance and zero at every later one.
func TestAGroupAdvanceIsOnePerPacedCallAtZeroOneAndManyOrders(t *testing.T) {
	mw := groupDriver(t)
	base := time.Unix(1_700_000_000, 0)
	tickMs := orderTickMs(t)
	at := func(n int) time.Time { return base.Add(time.Duration(n*tickMs) * time.Millisecond) }

	if n, applied := mw.paceTo(base); n != 0 || applied != 0 {
		t.Fatalf("the baseline call ran %d ticks and applied %d orders, want 0 and 0", n, applied)
	}

	for i, tc := range []struct {
		name    string
		enqueue int
	}{
		{"no order pending", 0},
		{"one order pending", 1},
		{"a whole group pending", len(groupIDs)},
	} {
		for k := 0; k < tc.enqueue; k++ {
			mw.enqueue(groupIDs[k], groupTargetX, groupTargetY)
		}
		n, applied := mw.paceTo(at(i + 1))
		if n != 1 {
			t.Errorf("%s: the advance ran %d ticks, want exactly 1", tc.name, n)
		}
		if applied != tc.enqueue {
			t.Errorf("%s: the advance applied %d orders, want %d", tc.name, applied, tc.enqueue)
		}
		if got := len(mw.pending); got != 0 {
			t.Errorf("%s: the queue holds %d orders after the advance, want 0", tc.name, got)
		}
	}
}

func TestAGroupQueueAndTheFrontEndsRectangleAreNotWorldState(t *testing.T) {
	mw := groupDriver(t)
	v := mw.view

	// A few ticks first, so the state held still below is one the world walked
	// to rather than the one it was built with.
	mw.enqueue(groupIDs[0], groupTargetX, groupTargetY)
	for i := 0; i < 4; i++ {
		mw.tick()
	}
	if cellsByID(mw.world)[groupIDs[0]] == groupStart[0] {
		t.Fatalf("no entity moved before the comparison, so the world held below is the one the "+
			"constructor built: %v", cellsByID(mw.world))
	}

	beforeTick := mw.world.Tick()
	beforeBounds := mw.world.Bounds()
	beforeEnts := mw.world.Entities()
	beforeBytes := marshalWorld(t, mw.world)
	beforeHash := mw.world.Hash()

	// The whole apparatus, with no advance anywhere behind it: the camera moves,
	// a screen rectangle is resolved to a band of cells through the camera's own
	// inverse, the ids standing on those cells are read out of the snapshot the
	// push built, and one order per id is queued.
	cam := v.Camera()
	cam.Pan(11, 7)
	cam.ZoomAbout(40, 30, 1.05)

	const half = camera.CellSize / 2
	x0, y0 := cam.WorldToScreen(half, half)
	x1, y1 := cam.WorldToScreen(float64((groupW-1)*camera.CellSize+half),
		float64((groupH-1)*camera.CellSize+half))
	r, ok := cam.ScreenToCellRange(x0, y0, x1, y1)
	if !ok {
		t.Fatalf("the rectangle over the whole field resolved to no cells at all")
	}

	var picked []uint32
	for _, d := range mw.entityDraws() {
		if d.Cell.X < r.Col0 || d.Cell.X >= r.Col1 || d.Cell.Y < r.Row0 || d.Cell.Y >= r.Row1 {
			continue
		}
		picked = append(picked, d.ID)
	}
	if len(picked) < 2 {
		t.Fatalf("the rectangle caught %d units (%v), so a group queued from it would not be a group",
			len(picked), picked)
	}
	for _, id := range picked {
		mw.enqueue(id, groupTargetX, groupTargetY)
	}

	if got := len(mw.pending); got != len(picked) {
		t.Fatalf("the queue holds %d orders after %d enqueues, want %d — the comparison below would "+
			"then be over a front-end that did nothing", got, len(picked), len(picked))
	}
	for _, id := range picked {
		if !mw.commanded[sim.EntityID(id)] {
			t.Fatalf("the commanded set does not hold entity %d, so the front-end state under test "+
				"never moved", id)
		}
	}

	if got := mw.world.Tick(); got != beforeTick {
		t.Errorf("the world's tick moved %d -> %d with no advance behind it", beforeTick, got)
	}
	if got := mw.world.Bounds(); got != beforeBounds {
		t.Errorf("the world's bounds moved %+v -> %+v with no advance behind it", beforeBounds, got)
	}
	if got := mw.world.Entities(); !reflect.DeepEqual(got, beforeEnts) {
		t.Errorf("a world field moved with no advance behind it:\n got %+v\nwant %+v", got, beforeEnts)
	}
	if got := marshalWorld(t, mw.world); !bytes.Equal(got, beforeBytes) {
		t.Errorf("the world's byte form moved with a group's orders pending and no advance behind them")
	}
	if got := mw.world.Hash(); got != beforeHash {
		t.Errorf("the world's digest moved %#x -> %#x with a group's orders pending and no advance "+
			"behind them", beforeHash, got)
	}
}

// grouped is a SECOND IMPLEMENTATION of the click boundary mapWorld.enqueue
// derives, and it exists so that a headless mirror of a front-end run is built
// from the rule rather than from the driver. It takes orders in issue order and
// tags them: an order joins the group being assembled when that group has a
// member, every member names the same cell, and none of them names this entity;
// anything else opens a new one.
//
// The tag VALUES need not match the driver's and are compared nowhere: a tag
// reaches no world field, so what has to agree is the PARTITION, and this
// function and enqueue partition one stream the same way.
func grouped(cmds []sim.Command) []sim.Command {
	out := make([]sim.Command, 0, len(cmds))
	var tag uint32
	for _, c := range cmds {
		c.Kind = sim.KindGroupMoveTo
		joins := false
		for _, p := range out {
			if p.Group != tag {
				continue
			}
			if p.X != c.X || p.Y != c.Y || p.Entity == c.Entity {
				joins = false
				break
			}
			joins = true
		}
		if !joins {
			tag++
		}
		c.Group = tag
		out = append(out, c)
	}
	return out
}

// TestOnePressIsOneGroupOrderAndTwoPressesAreTwo pins the click boundary the
// queue derives, which is the one piece of this seam that is not a single
// statement.
//
// A press emits one order per selected unit, all naming the pressed cell, each
// unit once — so those share a tag. A SECOND press does not, however it differs:
// a different cell, or the same cell with a unit already in the group. Both are
// exercised, because the rule has two clauses and a build that dropped either
// would merge two presses into one order and silently take the later one's
// destination away.
func TestOnePressIsOneGroupOrderAndTwoPressesAreTwo(t *testing.T) {
	mw := groupDriver(t)

	// One press over the whole selection.
	enqueueGroup(mw)
	first := mw.pending[0].Group
	for _, c := range mw.pending {
		if c.Kind != sim.KindGroupMoveTo {
			t.Fatalf("the queue holds a command of kind %d, want the group move-to", c.Kind)
		}
		if c.Group != first {
			t.Errorf("entity %d carries tag %d and the press opened %d — one press is one order",
				c.Entity, c.Group, first)
		}
	}

	// A second press on a DIFFERENT cell, over the same selection.
	n := len(mw.pending)
	for _, id := range groupIDs {
		mw.enqueue(id, groupTargetX-3, groupTargetY)
	}
	second := mw.pending[n].Group
	if second == first {
		t.Fatalf("a press on another cell carries the first press's tag %d — the two would be one order", first)
	}
	for _, c := range mw.pending[n:] {
		if c.Group != second {
			t.Errorf("entity %d carries tag %d and the second press opened %d", c.Entity, c.Group, second)
		}
	}

	// A third press on the SECOND press's cell, over one unit it already holds:
	// the same cell, so only the repeated entity can end the group.
	n = len(mw.pending)
	mw.enqueue(groupIDs[0], groupTargetX-3, groupTargetY)
	if got := mw.pending[n].Group; got == second {
		t.Errorf("a press repeating an entity already in the group carries its tag %d — a unit ordered "+
			"twice to one cell is two orders, not one group naming it twice", got)
	}
}
