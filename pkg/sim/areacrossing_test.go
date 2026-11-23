package sim

import (
	"reflect"
	"testing"
)

// The tick on which a transit's cell changes, and the cells that change
// releases and enters.

type strideEvent struct {
	tick     int32
	from, to cell
}

func strideEvents(s NativeStride) []strideEvent {
	var out []strideEvent
	widest := max(int32(s.StepX), -int32(s.StepX), int32(s.StepY), -int32(s.StepY))
	total := (256 + widest - 1) / widest
	for tick := int32(0); tick < total; tick++ {
		if from, to, crossed := strideCrossing(s, tick, total); crossed {
			out = append(out, strideEvent{tick, from, to})
		}
	}
	return out
}

// The expected ticks are worked by hand from the position arithmetic: a transit
// starts at fraction 0x80, tick t adds step t+1, and the cell byte changes when
// the sixteen-bit sum leaves its 256-wide cell. A positive step reaches 0x100
// at step ceil(128/s); a negative one borrows at step floor(128/s)+1.
func TestStrideCrossingTickFollowsTheSignOfTheStep(t *testing.T) {
	t.Parallel()
	c := func(x, y int32) cell { return cell{x, y} }
	for _, tc := range []struct {
		name string
		s    NativeStride
		want []strideEvent
	}{
		{"east 16: 128+8*16 = 256", NativeStride{FromX: 3, FromY: 3, ToX: 4, ToY: 3, StepX: 16}, []strideEvent{{7, c(3, 3), c(4, 3)}}},
		{"west 16: 128-8*16 = 0 stays, the ninth step borrows", NativeStride{FromX: 3, FromY: 3, ToX: 2, ToY: 3, StepX: -16}, []strideEvent{{8, c(3, 3), c(2, 3)}}},
		{"east 63: three steps", NativeStride{FromX: 3, FromY: 3, ToX: 4, ToY: 3, StepX: 63}, []strideEvent{{2, c(3, 3), c(4, 3)}}},
		{"north 5: 128-26*5 = -2", NativeStride{FromX: 3, FromY: 3, ToX: 3, ToY: 2, StepY: -5}, []strideEvent{{25, c(3, 3), c(3, 2)}}},
		{"south 5: 128+26*5 = 258", NativeStride{FromX: 3, FromY: 3, ToX: 3, ToY: 4, StepY: 5}, []strideEvent{{25, c(3, 3), c(3, 4)}}},
		{"diagonal 11 both ways positive: one corner", NativeStride{FromX: 3, FromY: 3, ToX: 4, ToY: 4, StepX: 11, StepY: 11}, []strideEvent{{11, c(3, 3), c(4, 4)}}},
		{"diagonal 16 east and north: x on step 8, y on step 9", NativeStride{FromX: 3, FromY: 3, ToX: 4, ToY: 2, StepX: 16, StepY: -16},
			[]strideEvent{{7, c(3, 3), c(4, 3)}, {8, c(4, 3), c(4, 2)}}},
	} {
		if got := strideEvents(tc.s); !reflect.DeepEqual(got, tc.want) {
			t.Errorf("%s: events %v, want %v", tc.name, got, tc.want)
		}
	}
}

// Octants 1 and 5 subtract one from x on the call that leaves both sub bytes
// 0 (MOVE-174, R2-ENGINE-1213), so both axes change cell on one call: octant 1
// on step 9 and octant 5 on step 8 at stride 16 (MOVE-200), in both games.
func TestStrideCrossingTakesTheOctantCorrection(t *testing.T) {
	t.Parallel()
	c := func(x, y int32) cell { return cell{x, y} }
	ne := NativeStride{FromX: 3, FromY: 3, ToX: 4, ToY: 2, StepX: 16, StepY: -16, Direction: 1}
	sw := NativeStride{FromX: 3, FromY: 3, ToX: 2, ToY: 4, StepX: -16, StepY: 16, Direction: 5}
	for _, tc := range []struct {
		name string
		s    NativeStride
		want []strideEvent
	}{
		{"octant 1", ne, []strideEvent{{8, c(3, 3), c(4, 2)}}},
		{"octant 5", sw, []strideEvent{{7, c(3, 3), c(2, 4)}}},
	} {
		if got := strideEvents(tc.s); !reflect.DeepEqual(got, tc.want) {
			t.Errorf("%s: events %v, want %v", tc.name, got, tc.want)
		}
	}
}

// A Speed-16 diagonal strides 11 per axis for 24 calls (MOVE-200). In octant
// 1 at stride 32 (128/32 = 4) the fourth call leaves both sub bytes 0 and x
// takes one off; the position keeps it until the snap to the destination
// centre on the last call. A rate-1 diagonal strides 0 in both games, 256 calls.
func TestStrideFinePointFollowsTheStepLaw(t *testing.T) {
	t.Parallel()
	e := Entity{X: 4, Y: 2}
	e.retainStride(cell{3, 3}, 16)
	if e.Stride.StepX != 11 || e.Stride.StepY != -11 || e.Stride.Direction != 1 || transitOf(16, true) != 24 {
		t.Fatalf("speed 16 diagonal stride %+v, %d calls", e.Stride, transitOf(16, true))
	}
	if x, y := e.Stride.finePoint(23, 24); x != 3*256+128+23*11 || y != 3*256+128-23*11 {
		t.Fatalf("call 23 at (%d,%d)", x, y)
	}
	if x, y := e.Stride.finePoint(24, 24); x != 4*256+128 || y != 2*256+128 {
		t.Fatalf("call 24 at (%d,%d), want the destination centre", x, y)
	}
	s := NativeStride{FromX: 3, FromY: 3, ToX: 4, ToY: 2, StepX: 32, StepY: -32, Direction: 1}
	for _, tc := range []struct {
		elapsed int32
		x       int32
	}{{3, 3*256 + 128 + 96}, {4, 4*256 - 1}, {5, 4*256 + 31}} {
		if x, _ := s.finePoint(tc.elapsed, 8); x != tc.x {
			t.Errorf("call %d: x %d, want %d", tc.elapsed, x, tc.x)
		}
	}
	slow := Entity{X: 4, Y: 4, HP: 1, MaxHP: 1, TransitTotal: uint16(transitOf(1, true)), Transit: 255}
	slow.retainStride(cell{3, 3}, 1)
	if slow.Stride.StepX != 0 || slow.Stride.StepY != 0 || slow.TransitTotal != 256 || strideFault(slow) != nil {
		t.Errorf("rate-1 diagonal: stride %+v, %d calls, fault %v", slow.Stride, slow.TransitTotal, strideFault(slow))
	}
	old := slow
	old.Stride.StepX, old.Stride.StepY = 1, 1
	if strideFault(old) != nil {
		t.Error("the earlier build's rate-1 diagonal stride of 1 is refused")
	}
}

// A mover of footprint two releases and occupies every cell of both footprints,
// not the two anchors; a cell outside both keeps its decayed byte.
func TestACrossingRecomputesTheWholeFootprint(t *testing.T) {
	t.Parallel()
	b := Bounds{Width: 12, Height: 8}
	cost := make([]byte, int(b.Width*b.Height))
	for i := range cost {
		cost[i] = 8
	}
	w, err := NewTerrainWorld(1, b, ModeCanonical, Terrain{Cost: cost, StartSlots: true},
		[]Entity{{ID: 1, X: 1, Y: 1, Speed: 16, TokenSize: 2}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	oldOnly, newOnly, outside := cell{1, 2}, cell{3, 2}, cell{8, 6}
	var keys []uint16
	for _, c := range []cell{oldOnly, newOnly, outside} {
		keys = append(keys, cellKey(c.x, c.y))
	}
	w.effects = []cellEffect{{Key: keys[0], Spell: 3, Remaining: 200, Mode: areaModeCloud, Cells: keys}}
	w.syncAreaCosts()
	for _, c := range []cell{oldOnly, newOnly, outside} {
		acReads(w, c, 2)
	}
	Step(w, []Command{acMove(1, 2, 1)})
	e := w.Entities()[0]
	if e.X != 2 || e.Y != 1 || !e.Stride.Present {
		t.Fatalf("the mover stands at (%d,%d), stride present %v", e.X, e.Y, e.Stride.Present)
	}
	stored := func(c cell) uint8 { a, _, _ := w.areaStored(cellKey(c.x, c.y), 8); return a }
	cross := acCrossingTick(t, e.Stride)
	for i := 1; i <= cross; i++ {
		if i == cross {
			if stored(oldOnly) != 2 || stored(newOnly) != 2 {
				t.Fatalf("before the crossing tick the footprint cells store %d and %d, want the decayed 2", stored(oldOnly), stored(newOnly))
			}
		}
		Step(w, nil)
	}
	if stored(oldOnly) != 32 || stored(newOnly) != 32 {
		t.Errorf("after the crossing the far footprint cells store %d and %d, want the recomputed 32", stored(oldOnly), stored(newOnly))
	}
	if stored(outside) != 2 {
		t.Errorf("a cell outside both footprints stores %d, want its decayed 2", stored(outside))
	}
}

// On a world with a saved cell plane a layer event's cost write waits for the
// end of the tick: the movers read the cell as the last tick left it.
func TestSavedPlaneLayerCostIsWrittenAfterTheActors(t *testing.T) {
	t.Parallel()
	w := acPlaneWorld(t)
	w.savedCellPlanes.Cost[0x1111], w.savedCellPlanes.CostKnown[0x1111] = 8, 1
	w.openCostWindow()
	if w.costWindow == nil {
		t.Fatal("no window on a saved plane world with cell records")
	}
	key := uint16(0x1111)
	w.addAreaEffect(cellEffect{Key: key, Spell: 3, Remaining: 200, Mode: areaModeCloud, Cells: []uint16{key}})
	if got := w.savedCellPlanes.Cost[key]; got != 8 {
		t.Errorf("the plane holds %d while the actors move, want the baseline 8", got)
	}
	if w.layeredPlaneCell(key) {
		t.Error("a cell first layered this tick reads as layered to this tick's movers")
	}
	// The cell that loses its layer keeps reading as layered until the end.
	w.clearSavedAreaLayer(0x1010, 0)
	if got := w.savedCellPlanes.Cost[0x1010]; got != 32 {
		t.Errorf("the expiring cell holds %d while the actors move, want 32", got)
	}
	if !w.layeredPlaneCell(0x1010) {
		t.Error("a cell whose layer ends this tick reads as bare to this tick's movers")
	}
	w.settleCostWindow()
	if got := w.savedCellPlanes.Cost[key]; got != 32 {
		t.Errorf("after the tick the new layer's cell holds %d, want 32", got)
	}
	if got := w.savedCellPlanes.Cost[0x1010]; got != 8 {
		t.Errorf("after the tick the expired cell holds %d, want its baseline 8", got)
	}
	if w.costWindow != nil || !w.layeredPlaneCell(key) || w.layeredPlaneCell(0x1010) {
		t.Error("the window is not closed or the flags did not follow the writes")
	}
}

// A layer whose effect ends in a tick is still a layer to that tick's movers:
// the area-effect tick that removes it runs after every actor. The mover's
// read finds the byte a previous read decayed, so a world that removed the
// layer first would price the step at the baseline.
func TestSavedPlaneLayerEndsAfterTheActorsOfItsTick(t *testing.T) {
	t.Parallel()
	const target = uint16(0x1111)
	build := func(layer bool, remaining uint16) *World {
		w := acPlaneWorldWith(t, []Entity{{ID: 1, X: 0x10, Y: 0x11, Speed: 16}})
		for _, k := range []uint16{0x1011, target} {
			w.savedCellPlanes.Cost[k], w.savedCellPlanes.CostKnown[k] = 8, 1
		}
		if layer {
			w.addAreaEffect(cellEffect{Key: target, Spell: 3, Remaining: remaining, Mode: areaModeCloud, Cells: []uint16{target}})
			// A first read leaves the baseline stored; the mover's read finds a quarter of it.
			if got := w.readCost(cell{x: 0x11, y: 0x11}, true); got != 8 {
				t.Fatalf("the first read answers %d, want the baseline 8", got)
			}
		}
		return w
	}
	transit := func(w *World) uint16 {
		Step(w, []Command{acMove(1, 0x11, 0x11)})
		e := w.Entities()[0]
		if e.X != 0x11 || e.Y != 0x11 {
			t.Fatalf("the mover stands at (%d,%d)", e.X, e.Y)
		}
		return e.TransitTotal
	}
	ends, stands, bare := build(true, 0), build(true, 200), build(false, 0)
	tEnds, tStands, tBare := transit(ends), transit(stands), transit(bare)
	if tEnds != tStands {
		t.Errorf("a layer ending this tick gives transit %d, one that stands gives %d", tEnds, tStands)
	}
	if tEnds == tBare {
		t.Errorf("transit %d equals bare ground's, so the witness does not see the layer", tEnds)
	}
	if got := ends.savedCellPlanes.Cost[target]; got != 8 {
		t.Errorf("after the tick the expired cell holds %d, want its baseline 8", got)
	}
	if ends.layeredPlaneCell(target) {
		t.Error("the expired cell is still layered after the tick")
	}
}

func TestASavedPlaneCrossingRecomputesTheEnteredLayeredCell(t *testing.T) {
	t.Parallel()
	const from, target = uint16(0x1110), uint16(0x1111)
	w := acPlaneWorldWith(t, []Entity{{ID: 1, X: 0x10, Y: 0x11, Speed: 16}})
	for _, k := range []uint16{from, target} {
		w.savedCellPlanes.Cost[k], w.savedCellPlanes.CostKnown[k] = 8, 1
	}
	w.occupyMissionStartCells()
	w.addAreaEffect(cellEffect{Key: target, Spell: 3, Remaining: 200, Mode: areaModeCloud, Cells: []uint16{target}})
	if c := w.motionCell(from); c == nil || c.Ground != (SavedActorSlot{Entity: 1, Bound: true}) {
		t.Fatalf("the mover holds no slot where it stands: %+v", c)
	}
	if got := w.savedCellPlanes.Cost[target]; got != 32 {
		t.Fatalf("the layered cell holds %d, want 32", got)
	}
	slot := func() SavedActorSlot {
		if c := w.motionCell(target); c != nil {
			return c.Ground
		}
		return SavedActorSlot{}
	}
	Step(w, []Command{acMove(1, 0x11, 0x11)})
	crossing := acCrossingTick(t, w.Entities()[0].Stride)
	for i := 0; i < crossing; i++ {
		if got := w.savedCellPlanes.Cost[target]; got != 8 || slot() != (SavedActorSlot{}) {
			t.Fatalf("tick %d of the transit: the entered cell holds %d and slot %+v, want the decayed 8 and none", i, got, slot())
		}
		Step(w, nil)
	}
	if got := w.savedCellPlanes.Cost[target]; got != 32 || slot() != (SavedActorSlot{Entity: 1, Bound: true}) {
		t.Errorf("after the crossing the entered cell holds %d and slot %+v, want the recomputed 32 and the mover", got, slot())
	}
}
