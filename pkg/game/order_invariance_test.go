package game

import (
	"bytes"
	"image"
	"reflect"
	"testing"
	"time"

	"againrom/pkg/mapload"
	"againrom/pkg/render/camera"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// invarianceTicks is how many advances every leg runs. It is the length of
// scriptedLaps' own table, so no leg reads past the schedule's end and every
// tick compared below has a scripted entry behind it.
const invarianceTicks = 13 * cellTicks

// sessionOrder is one order the front-end forms: the tick it is issued before,
// the cell the unit is PICKED on, the id the snapshot must mint there, and the
// cell it is sent to.
type sessionOrder struct {
	at     int
	from   image.Point
	entity uint32
	to     image.Point
}

// invarianceOrders are the two orders the ordered legs run.
//
// Neither entity is named by any entry of scriptedLaps, so nothing else moves
// them and the from-cells stay where worldFixtureCells puts them until the order
// arrives. Neither target lies on a cell a scripted walk crosses — entity 0
// works along row 23 and entity 1 along column 25 — so no contention decides
// where an ordered unit ends up, and both arrive with ticks to spare, which is
// what leaves each order visible in the world for the rest of the run rather
// than only at the tick it drained. Both targets stand inside the interior, so
// neither walk answers to the ring.
var invarianceOrders = []sessionOrder{
	{at: 3 * cellTicks, from: image.Pt(22, 27), entity: 2, to: image.Pt(29, 27)},
	{at: 7 * cellTicks, from: image.Pt(31, 24), entity: 3, to: image.Pt(31, 28)},
}

// cellUnderCursor puts the cursor over the centre of one cell, resolves it back
// through the camera's own inverse — ScreenToCell, the shipped pick — and fails
// unless the cell that comes back is the one aimed at.
//
// It is the round trip and not a computation: the column and row this returns
// came out of the camera, so an order built on it is an order the front-end's
// own route produced rather than a literal wearing a pick's clothes.
func cellUnderCursor(t *testing.T, cam *camera.Camera, c image.Point) image.Point {
	t.Helper()
	const half = camera.CellSize / 2
	sx, sy := cam.WorldToScreen(float64(c.X*camera.CellSize+half), float64(c.Y*camera.CellSize+half))
	col, row, inside := cam.ScreenToCell(sx, sy)
	if !inside || col != c.X || row != c.Y {
		t.Fatalf("a cursor over the centre of cell %v resolved to (%d,%d) inside=%v, want %v inside",
			c, col, row, inside, c)
	}
	return image.Pt(col, row)
}

// lowestIDOn is what a pick holds after resolving to cell c: the LOWEST id
// standing there, and whether one stands there at all. The tie is settled over
// ids rather than over slice position, as the front-end's own pick settles it.
func lowestIDOn(ents []ui.MapEntity, c image.Point) (uint32, bool) {
	var best uint32
	hit := false
	for _, e := range ents {
		if e.Cell != c {
			continue
		}
		if !hit || e.ID < best {
			best, hit = e.ID, true
		}
	}
	return best, hit
}

// frontEndFrame is the front-end's own work between two advances, driven through
// every surface of it a headless test can reach: the camera moves, a cursor well
// past the map's edge is resolved and must come back OUTSIDE, and the snapshot
// is read back through the seam's own derivation — the very call the push makes.
// It hands that snapshot to the caller, which is what an order's id then comes
// out of.
//
// None of this may reach a world. That is the whole of what the comparisons
// below assert, and the reason this runs on one leg and not on the other.
func frontEndFrame(t *testing.T, mw *mapWorld, v *ui.Viewer, i int) []ui.MapEntity {
	t.Helper()
	cam := v.Camera()
	cam.Pan(float64(3+i), float64(2+i))
	cam.ZoomAbout(float64(40+i), float64(30+i), 1.05)

	off := float64((worldFixtureW + 3) * camera.CellSize)
	sx, sy := cam.WorldToScreen(off, off)
	if col, row, inside := cam.ScreenToCell(sx, sy); inside {
		t.Fatalf("a cursor %v world pixels past the extent came back inside, on cell (%d,%d)", off, col, row)
	}

	// 0030: the selection a release forms, driven as far as this tier can reach
	// it — a screen rectangle resolved to a band of cells through the camera's
	// own inverse, and the ids the snapshot holds on that band. It reads and it
	// returns; nothing on this path may touch a world, which is the whole of
	// what the comparisons below assert.
	const half = camera.CellSize / 2
	bx0, by0 := cam.WorldToScreen(half, half)
	bx1, by1 := cam.WorldToScreen(float64((worldFixtureW-1)*camera.CellSize+half),
		float64((worldFixtureH-1)*camera.CellSize+half))
	band, ok := cam.ScreenToCellRange(bx0, by0, bx1, by1)
	if !ok {
		t.Fatalf("a rectangle over the whole map resolved to no cells at all")
	}
	if r, ok := cam.ScreenToCellRange(sx, sy, sx+8, sy+8); ok {
		t.Fatalf("a rectangle %v world pixels past the extent came back holding %+v", off, r)
	}

	draws := mw.entityDraws()
	var picked []uint32
	for _, d := range draws {
		if d.Cell.X < band.Col0 || d.Cell.X >= band.Col1 || d.Cell.Y < band.Row0 || d.Cell.Y >= band.Row1 {
			continue
		}
		picked = append(picked, d.ID)
	}
	if len(picked) == 0 {
		t.Fatalf("frame %d: the rectangle over the whole map caught no unit, so the selection this "+
			"frame drives is empty and drives nothing", i)
	}
	return draws
}

func TestASessionsPickAndQueueReachNoWorldBeyondTheOrder(t *testing.T) {
	base := time.Unix(1_700_000_000, 0)
	tickMs := orderTickMs(t)
	at := func(n int) time.Time { return base.Add(time.Duration(n*tickMs) * time.Millisecond) }

	m := worldFixtureMap()

	sv := worldFixtureViewer(t, m)
	session := newMapWorld(mapload.FromALM(m), scriptedLaps(), nil, sv)
	bare := newMapWorld(mapload.FromALM(m), scriptedLaps(), nil, worldFixtureViewer(t, m))
	quiet := newMapWorld(mapload.FromALM(m), scriptedLaps(), nil, worldFixtureViewer(t, m))

	// The headless leg: the same map with no driver, no viewer and no queue.
	headless := mapload.FromALM(m)
	headSched := scriptedLaps()

	// The first paced call only takes the baseline and advances nothing, so every
	// leg is still at tick 0 when the run starts.
	for _, leg := range []struct {
		name string
		mw   *mapWorld
	}{{"session", session}, {"bare", bare}, {"quiet", quiet}} {
		if n, applied := leg.mw.paceTo(base); n != 0 || applied != 0 {
			t.Fatalf("the %s leg's baseline call ran %d ticks and applied %d orders, want 0 and 0",
				leg.name, n, applied)
		}
	}
	if s, b, h := session.world.Hash(), bare.world.Hash(), headless.Hash(); s != h || b != h {
		t.Fatalf("the legs start at %#x, %#x and %#x — they must begin equal", s, b, h)
	}

	startZoom := sv.Camera().Zoom
	startX, startY := sv.Camera().X, sv.Camera().Y

	for tk := 0; tk < invarianceTicks; tk++ {
		ents := frontEndFrame(t, session, sv, tk)

		combined := headSched[tk]
		var issued []sim.Command
		for _, o := range invarianceOrders {
			if o.at != tk {
				continue
			}
			// The front-end's own route to the three scalars: the unit is picked
			// on its cell, its id read out of the snapshot the push builds, and
			// the target resolved by the same inverse.
			id, hit := lowestIDOn(ents, cellUnderCursor(t, sv.Camera(), o.from))
			if !hit {
				t.Fatalf("tick %d: the pick on %v found no unit at all, so the order below names nothing",
					tk, o.from)
			}
			if id != o.entity {
				t.Fatalf("tick %d: the pick on %v yielded id %d, want %d", tk, o.from, id, o.entity)
			}
			to := cellUnderCursor(t, sv.Camera(), o.to)
			session.enqueue(id, to.X, to.Y)

			// The other two legs take the same three scalars as literals.
			bare.enqueue(o.entity, o.to.X, o.to.Y)
			issued = append(issued,
				sim.Command{Entity: sim.EntityID(o.entity), X: int32(o.to.X), Y: int32(o.to.Y)})
		}
		// THE SCRIPT'S OWN COMMANDS ARE NOT ORDERS. Only what leaves through the
		// front-end's seam is a group order, so the issued half alone goes
		// through this file's own copy of the click boundary.
		combined = append(append([]sim.Command(nil), combined...), grouped(issued)...)

		sn, sa := session.paceTo(at(tk + 1))
		bn, ba := bare.paceTo(at(tk + 1))
		quiet.paceTo(at(tk + 1))
		sim.Step(headless, combined)

		if sn != 1 || bn != 1 {
			t.Fatalf("advance %d ran %d ticks on the session leg and %d on the bare one; this run is written "+
				"over one tick per advance", tk+1, sn, bn)
		}
		if sa != ba {
			t.Fatalf("advance %d applied %d orders on the session leg and %d on the bare one — the same three "+
				"scalars issued at the same advance must drain at the same one", tk+1, sa, ba)
		}

		sessionBytes := marshalWorld(t, session.world)
		if got, want := marshalWorld(t, bare.world), sessionBytes; !bytes.Equal(got, want) {
			t.Fatalf("after advance %d the session's byte form differs from the bare run's — the camera, the "+
				"pick, the snapshot read and the queue are front-end state and reach no world field (FR-9, P-2)",
				tk+1)
		}
		if got, want := session.world.Hash(), bare.world.Hash(); got != want {
			t.Fatalf("after advance %d the session digest %#x != the bare digest %#x (FR-9, P-2)", tk+1, got, want)
		}
		if got := marshalWorld(t, headless); !bytes.Equal(got, sessionBytes) {
			t.Fatalf("after advance %d the session's byte form differs from a headless run over the script's "+
				"entry plus the order — the advance did not run on that stream, at that tick (FR-9)", tk+1)
		}
		if got, want := session.world.Hash(), headless.Hash(); got != want {
			t.Fatalf("after advance %d the session digest %#x != the headless digest %#x (FR-9)", tk+1, got, want)
		}

		// Non-vacuity, from the first drain on: the orders must actually be
		// telling in the world, or every agreement above is an agreement between
		// runs that did nothing.
		if tk >= invarianceOrders[0].at && session.world.Hash() == quiet.world.Hash() {
			t.Fatalf("after advance %d the ordered legs hold the unordered run's digest %#x, so their agreeing "+
				"says nothing about what an order did", tk+1, quiet.world.Hash())
		}
	}

	// Each ordered unit stands on the cell it was sent to, and not on the one it
	// started from: an order that named a unit the world moved nowhere would leave
	// every comparison above an agreement about a world no order reached.
	cells := entityCells(session.world)
	for _, o := range invarianceOrders {
		if got := cells[o.entity]; got != o.to {
			t.Errorf("entity %d stands on %v after the run, want the ordered cell %v", o.entity, got, o.to)
		}
		if got := cells[o.entity]; got == worldFixtureCells[o.entity] {
			t.Errorf("entity %d never left its start cell %v, so its order moved nothing", o.entity, got)
		}
	}

	// The front-end state the session drove really did move, or the leg it stands
	// on is one that did nothing either.
	if cam := sv.Camera(); cam.Zoom == startZoom && cam.X == startX && cam.Y == startY {
		t.Errorf("the session's camera is still at zoom %v and (%v,%v) after %d frames of panning and zooming",
			startZoom, startX, startY, invarianceTicks)
	}
	if got := len(session.pending); got != 0 {
		t.Errorf("the session's queue holds %d orders after the run, want 0", got)
	}
}

func TestAPendingOrderAndAPickAreNotWorldState(t *testing.T) {
	m := worldFixtureMap()
	v := worldFixtureViewer(t, m)
	mw := newMapWorld(mapload.FromALM(m), scriptedLaps(), nil, v)

	// A few ticks first, so the state held still below is a state the world
	// walked to rather than the one it was built with.
	for i := 0; i < 5; i++ {
		mw.tick()
	}
	if !someEntityHasMoved(worldFixtureCells, entityCells(mw.world)) {
		t.Fatalf("no entity moved in the five ticks before the comparison, so the world held below is the "+
			"one the constructor built: %v", entityCells(mw.world))
	}

	beforeTick := mw.world.Tick()
	beforeBounds := mw.world.Bounds()
	beforeEnts := mw.world.Entities()
	beforeBytes := marshalWorld(t, mw.world)
	beforeHash := mw.world.Hash()

	// The whole apparatus, with no advance anywhere behind it.
	ents := frontEndFrame(t, mw, v, 0)
	id, hit := lowestIDOn(ents, cellUnderCursor(t, v.Camera(), image.Pt(22, 27)))
	if !hit {
		t.Fatalf("the pick on (22,27) found no unit, so nothing below is ordered")
	}
	to := cellUnderCursor(t, v.Camera(), image.Pt(29, 27))
	mw.enqueue(id, to.X, to.Y)
	mw.enqueue(id, to.X+1, to.Y)
	mw.enqueue(3, 31, 28)

	if got := len(mw.pending); got != 3 {
		t.Fatalf("the queue holds %d orders after three enqueues, want 3 — the comparison below would then be "+
			"over a front-end that did nothing", got)
	}
	if !mw.commanded[sim.EntityID(id)] || !mw.commanded[3] {
		t.Fatalf("the commanded set does not hold both ordered entities, so the front-end state under test " +
			"never moved")
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
		t.Errorf("the world's byte form moved with three orders pending and no advance behind them — a " +
			"pending order is front-end state, never world state (FR-9, P-2)")
	}
	if got := mw.world.Hash(); got != beforeHash {
		t.Errorf("the world's digest moved %#x -> %#x with three orders pending and no advance behind them "+
			"(FR-9, P-2)", beforeHash, got)
	}
}

// marshalWorld is a world's canonical byte form, the form its digest is taken
// over. MarshalBinary's error is checked here rather than discarded at each of
// the call sites above.
func marshalWorld(t *testing.T, w *sim.World) []byte {
	t.Helper()
	b, err := w.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary: %v", err)
	}
	return b
}
