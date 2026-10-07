package sim

import (
	"encoding/binary"
	"testing"
)

// The mutable movement-cost byte under area layers. Expected values are
// computed here with explicit eight-bit shifts and checked against the
// quotients the claim's table states for the shipped cost range 6..16.

func acMove(id EntityID, x, y int32) Command { return MoveTo(id, CellPoint{X: x, Y: y}) }

// acWorld is a world over srBounds with a uniform cost plane and the named
// cells covered by one cloud per listed spell.
func acWorld(t *testing.T, cost byte, ents []Entity, covered []uint16, spells ...uint16) *World {
	t.Helper()
	w := srWorld(t, srPlane(cost), nil, ents)
	for _, s := range spells {
		w.effects = append(w.effects, cellEffect{Key: covered[0], Spell: s, Remaining: 200, Mode: areaModeCloud, Cells: covered})
	}
	w.syncAreaCosts()
	return w
}

func acReads(w *World, c cell, n int) []uint8 {
	out := make([]uint8, n)
	for i := range out {
		out[i] = w.readCost(c, true)
	}
	return out
}

func TestAreaCostDecaysOnEveryRead(t *testing.T) {
	t.Parallel()
	at := cell{x: 2, y: 1}
	key := cellKey(2, 1)
	for c := byte(6); c <= 16; c++ {
		one := acWorld(t, c, nil, []uint16{key}, 3)
		got := acReads(one, at, 4)
		stored := uint8(c << 2)
		for i, g := range got {
			stored >>= 2
			if g != stored {
				t.Errorf("one layer, cost %d, read %d = %d, want %d", c, i+1, g, stored)
			}
		}
		if got[0] != c {
			t.Errorf("one layer, cost %d: read 1 = %d, want the baseline", c, got[0])
		}
		if got[1] != c>>2 {
			t.Errorf("one layer, cost %d: read 2 = %d, want %d", c, got[1], c>>2)
		}
		if c == 16 && got[2] != 1 || c != 16 && got[2] != 0 {
			t.Errorf("one layer, cost %d: read 3 = %d", c, got[2])
		}

		two := acWorld(t, c, nil, []uint16{key}, 3, 12)
		got = acReads(two, at, 5)
		stored = uint8(c<<2) << 2
		for i, g := range got {
			stored >>= 2
			if g != stored {
				t.Errorf("two layers, cost %d, read %d = %d, want %d", c, i+1, g, stored)
			}
		}
		if want := uint8(c<<4) >> 2; got[0] != want {
			t.Errorf("two layers, cost %d: read 1 = %d, want %d in eight bits", c, got[0], want)
		}
		if c == 16 && got[0] != 0 {
			t.Errorf("two layers at cost 16: read 1 = %d, want 0", got[0])
		}
	}
}

func TestAreaCostThreeLayersTruncateToZeroOrOneTwentyEight(t *testing.T) {
	t.Parallel()
	key := cellKey(2, 1)
	for c, want := range map[byte]uint8{6: 128, 8: 0, 12: 0, 16: 0} {
		w := acWorld(t, c, nil, []uint16{key}, 3, 12, 7)
		if got, _, _ := w.areaStored(key, c); got != want {
			t.Errorf("three layers at cost %d store %d, want %d", c, got, want)
		}
	}
}

func TestAreaCostRecomputeRestoresTheStoredByte(t *testing.T) {
	t.Parallel()
	at := cell{x: 2, y: 1}
	w := acWorld(t, 12, nil, []uint16{cellKey(2, 1)}, 3)
	acReads(w, at, 3)
	if got := w.readCost(at, false); got != 0 {
		t.Fatalf("after three reads the next read answers %d, want 0", got)
	}
	w.recomputeAreaCosts(cell{x: 1, y: 1}, at)
	if got := acReads(w, at, 2); got[0] != 12 || got[1] != 3 {
		t.Errorf("after a recompute the reads answer %v, want [12 3]", got)
	}
	if a, _, _ := w.areaStored(cellKey(2, 1), 12); a != 3 {
		t.Errorf("stored byte %d, want 3", a)
	}
}

func TestAreaCostUnchangedLayersKeepTheirDecayAtTheAreaPass(t *testing.T) {
	t.Parallel()
	at := cell{x: 2, y: 1}
	key := cellKey(2, 1)
	w := acWorld(t, 16, nil, []uint16{key}, 3)
	acReads(w, at, 2)
	w.syncAreaCosts()
	if got := w.readCost(at, false); got != 1 {
		t.Errorf("an area pass over an unchanged layer set answers %d, want the decayed 1", got)
	}
	w.effects = append(w.effects, cellEffect{Spell: 12, Remaining: 9, Mode: areaModeCloud, Cells: []uint16{key}})
	w.syncAreaCosts()
	if a, m, _ := w.areaStored(key, 16); m != 1|1<<4 || a != 0 {
		t.Errorf("a second layer stores %d under mask %b, want a recompute to 0 under 10001", a, m)
	}
	w.effects = nil
	w.syncAreaCosts()
	if _, _, ok := w.areaStored(key, 16); ok || w.readCost(at, true) != 16 {
		t.Error("a cell with no layer left did not return to its baseline")
	}
}

// Actor updates run before the area pass: a layer first present during a tick
// is read as no layer by every actor of that tick.
func TestAreaLayerIsAppliedAfterTheActorsOfItsTick(t *testing.T) {
	t.Parallel()
	ents := []Entity{{ID: 1, X: 1, Y: 1, Speed: 16}}
	key := cellKey(2, 1)
	build := func(layers bool) *World {
		w := srWorld(t, srPlane(8), nil, ents)
		if layers {
			w.effects = []cellEffect{
				{Key: key, Spell: 3, Remaining: 200, Mode: areaModeCloud, Cells: []uint16{key}},
				{Key: key, Spell: 12, Remaining: 200, Mode: areaModeCloud, Cells: []uint16{key}},
			}
		}
		return w
	}
	bare, layered := build(false), build(false)
	Step(bare, nil)
	Step(layered, nil)
	layered.effects = build(true).effects
	Step(bare, []Command{acMove(1, 2, 1)})
	Step(layered, []Command{acMove(1, 2, 1)})
	if a, b := bare.Entities()[0].TransitTotal, layered.Entities()[0].TransitTotal; a != b {
		t.Fatalf("a layer present for its first tick changed the transit from %d to %d", a, b)
	}
	// One tick later the layers are applied: two layers store 128 and a read
	// divides it to 32, so the transit is longer than over bare ground.
	later := build(false)
	Step(later, nil)
	later.effects = build(true).effects
	Step(later, nil)
	Step(later, []Command{acMove(1, 2, 1)})
	if a, b := bare.Entities()[0].TransitTotal, later.Entities()[0].TransitTotal; b <= a {
		t.Errorf("two applied layers left the transit at %d, bare ground gives %d", b, a)
	}
}

// Two actors reading one layered cell before its recompute: the second read
// divides the byte the first one left.
func TestASecondActorReadsTheDecayedByte(t *testing.T) {
	t.Parallel()
	key := cellKey(2, 1)
	run := func(layer bool) (first, second uint16) {
		ents := []Entity{{ID: 1, X: 2, Y: 1, Speed: 16}, {ID: 2, X: 1, Y: 1, Speed: 16}}
		w := srWorld(t, srPlane(8), nil, ents)
		if layer {
			w.effects = []cellEffect{{Key: key, Spell: 3, Remaining: 200, Mode: areaModeCloud, Cells: []uint16{key}}}
			w.syncAreaCosts()
		}
		Step(w, []Command{acMove(1, 3, 1), acMove(2, 2, 1)})
		es := w.Entities()
		if es[0].X != 3 || es[1].X != 2 {
			t.Fatalf("movers at (%d,%d) and (%d,%d)", es[0].X, es[0].Y, es[1].X, es[1].Y)
		}
		return es[0].TransitTotal, es[1].TransitTotal
	}
	f0, s0 := run(false)
	f1, s1 := run(true)
	if f0 != f1 {
		t.Errorf("the first reader's transit changed from %d to %d under one layer", f0, f1)
	}
	if s1 >= s0 {
		t.Errorf("the second reader's transit is %d under the layer and %d without it, want shorter", s1, s0)
	}
}

// acCrossingTick is the transit tick on which the position's cell changes,
// counted from the starting tick (tick 0), by adding the signed step to each
// axis's sixteen-bit position once per tick.
func acCrossingTick(t *testing.T, s NativeStride) int {
	t.Helper()
	x, y := (s.FromX<<8)|0x80, (s.FromY<<8)|0x80
	for tick := 0; tick < 300; tick++ {
		nx, ny := x+int32(s.StepX), y+int32(s.StepY)
		if nx>>8 != x>>8 || ny>>8 != y>>8 {
			return tick
		}
		x, y = nx, ny
	}
	t.Fatalf("stride %+v never changes cell", s)
	return 0
}

func TestACrossingRecomputesBothCells(t *testing.T) {
	t.Parallel()
	key := cellKey(2, 1)
	w := srWorld(t, srPlane(8), nil, []Entity{{ID: 1, X: 1, Y: 1, Speed: 16}})
	w.effects = []cellEffect{{Key: key, Spell: 3, Remaining: 200, Mode: areaModeCloud, Cells: []uint16{key}}}
	w.syncAreaCosts()
	acReads(w, cell{x: 2, y: 1}, 2)
	Step(w, []Command{acMove(1, 2, 1)})
	// The step read the cell a third time, so it holds the byte after three reads.
	if a, _, _ := w.areaStored(key, 8); a != 0 {
		t.Fatalf("stored %d after the transit start, want the decayed 0", a)
	}
	crossing := acCrossingTick(t, w.Entities()[0].Stride)
	for i := 1; i < crossing; i++ {
		Step(w, nil)
		if a, _, _ := w.areaStored(key, 8); a != 0 {
			t.Fatalf("tick %d of the transit: stored %d, recompute is due at tick %d", i, a, crossing)
		}
	}
	Step(w, nil)
	if a, _, _ := w.areaStored(key, 8); a != 32 {
		t.Errorf("at the crossing tick the cell stores %d, want the recomputed 32", a)
	}
}

func TestAReadOnlyRateQueryNeverDecays(t *testing.T) {
	t.Parallel()
	key := cellKey(2, 1)
	w := srWorld(t, srPlane(16), nil, []Entity{{ID: 1, X: 1, Y: 1, Speed: 16}})
	w.effects = []cellEffect{{Key: key, Spell: 3, Remaining: 200, Mode: areaModeCloud, Cells: []uint16{key}}}
	w.syncAreaCosts()
	before := acMarshal(t, w)
	r1, _, _, _ := w.StepRate(1, 2, 1)
	r2, _, _, _ := w.StepRate(1, 2, 1)
	if r1 != r2 {
		t.Errorf("two queries answered %d and %d", r1, r2)
	}
	if string(acMarshal(t, w)) != string(before) {
		t.Error("a rate query changed the world")
	}
}

// A decayed byte of zero reaches the rate law's mean and the route search.
// The mean substitutes eight for a zero mean; the search prices a free step.
// Neither divides by the byte.
func TestAZeroCostByteFaultsNowhere(t *testing.T) {
	t.Parallel()
	key := cellKey(2, 1)
	w := srWorld(t, srPlane(16), nil, []Entity{{ID: 1, X: 1, Y: 1, Speed: 16}})
	w.effects = []cellEffect{{Key: key, Spell: 3, Remaining: 200, Mode: areaModeCloud, Cells: []uint16{key}}, {Key: key, Spell: 12, Remaining: 200, Mode: areaModeCloud, Cells: []uint16{key}}}
	w.syncAreaCosts()
	if got := w.readCost(cell{x: 2, y: 1}, false); got != 0 {
		t.Fatalf("two layers at cost 16 read %d, want 0", got)
	}
	if got := w.costAt(cell{x: 2, y: 1}); got != 0 {
		t.Fatalf("the route plane holds %d, want the truncated 0", got)
	}
	if rate, transit, _, ok := w.StepRate(1, 2, 1); !ok || rate < rateFloor || transit < 1 {
		t.Errorf("rate %d transit %d ok %v over a zero byte", rate, transit, ok)
	}
	Step(w, []Command{acMove(1, 4, 1)})
	if e := w.Entities()[0]; e.X != 2 {
		t.Errorf("the mover stands at x=%d, want it across the zero-cost cell", e.X)
	}
}

func TestAreaCostBinaryFormRoundTripsAndOmitsFreshCells(t *testing.T) {
	t.Parallel()
	key := cellKey(2, 1)
	w := srWorld(t, srPlane(8), nil, []Entity{{ID: 1, X: 1, Y: 1, Speed: 16}})
	w.effects = []cellEffect{{Key: key, Spell: 3, Remaining: 200, Mode: areaModeCloud, Cells: []uint16{key}}}
	w.syncAreaCosts()
	plain := acMarshal(t, w)
	if plain[0] == areaCostFormVersion {
		t.Fatal("an undecayed layered cell wrote an area cost trailer")
	}
	acReads(w, cell{x: 2, y: 1}, 2)
	form := acMarshal(t, w)
	if form[0] != areaCostFormVersion {
		t.Fatalf("a decayed cell wrote form %d", form[0])
	}
	var back World
	if err := back.UnmarshalBinary(form); err != nil {
		t.Fatal(err)
	}
	if got := back.readCost(cell{x: 2, y: 1}, false); got != 0 {
		t.Errorf("the loaded world reads %d, want the decayed 0", got)
	}
	if again := acMarshal(t, &back); string(again) != string(form) {
		t.Error("the loaded world marshals differently")
	}
	// A world loaded without the trailer recomputes the layered cell: it
	// stores 32 and its first read answers the baseline 8.
	var fresh World
	if err := fresh.UnmarshalBinary(plain); err != nil {
		t.Fatal(err)
	}
	if got := fresh.readCost(cell{x: 2, y: 1}, false); got != 8 {
		t.Errorf("a world without a trailer reads %d, want the baseline 8", got)
	}
}

func acMarshal(t *testing.T, w *World) []byte {
	t.Helper()
	b, err := w.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// A world with a saved cell plane keeps the byte in the plane itself: the read
// divides and stores it there, a cell without the record's static bit is left
// alone, and a recompute puts the multiplied byte back.
func TestAreaCostOnASavedCellPlane(t *testing.T) {
	t.Parallel()
	w := acPlaneWorld(t)
	layered, bare := cell{x: 0x10, y: 0x10}, cell{x: 0x20 - 1, y: 0x1f}
	w.motionCell(0x1010).Payload[0] = 8
	w.recomputeSavedCell(0x1010)
	if got := w.savedCellPlanes.Cost[0x1010]; got != 32 {
		t.Fatalf("recomputed plane byte %d, want 32", got)
	}
	if got := w.searchCostAt(0, layered); got != 32 {
		t.Errorf("the route search sees %d, want the multiplied 32", got)
	}
	if got := acReads(w, layered, 3); got[0] != 8 || got[1] != 2 || got[2] != 0 {
		t.Errorf("reads %v, want [8 2 0]", got)
	}
	if got := w.savedCellPlanes.Cost[0x1010]; got != 0 {
		t.Errorf("plane byte %d after three reads, want 0", got)
	}
	if q := w.readCost(bare, true); q != w.costAt(bare) {
		t.Errorf("a cell with no record divided to %d", q)
	}
	w.recomputeAreaCosts(layered)
	if got := w.savedCellPlanes.Cost[0x1010]; got != 32 {
		t.Errorf("plane byte %d after a recompute, want 32", got)
	}
}

// A cost plane is not saved, so a loaded layered cell holds the byte terrain
// ingest gave it. ResetLoadedAreaCosts puts that byte back in every layered
// cell and leaves every other plane byte as it stands.
func TestRewriteImportedLayerCostsRewritesTheLayeredCell(t *testing.T) {
	t.Parallel()
	w := acPlaneWorld(t)
	w.motionCell(0x1010).Payload[0] = 8
	w.savedCellPlanes.Cost[0x1313], w.savedCellPlanes.CostKnown[0x1313] = 11, 1
	w.savedCellPlanes.Cost[0x1010] = 8
	w.RewriteImportedLayerCosts()
	if got := w.savedCellPlanes.Cost[0x1010]; got != 32 {
		t.Errorf("a layered cell holds %d after the rewrite, want the recomputed 32", got)
	}
	if got := w.savedCellPlanes.Cost[0x1313]; got != 11 {
		t.Errorf("a cell with no layer changed to %d", got)
	}
}

// A saved-plane world's form carries the cost plane, so SAVE then cold LOAD
// keeps a layered cell's decayed byte, and a mover's first read of it is the
// continuing world's. The reset rewrites nothing. A rewrite after the load, as
// an import makes, restores the full byte and changes the read.
func TestSavedPlaneFormKeepsADecayedLayeredCellAcrossLoad(t *testing.T) {
	t.Parallel()
	w := acPlaneWorld(t)
	w.motionCell(0x1010).Payload[0] = 8
	layered := cell{x: 0x10, y: 0x10}
	w.savedCellPlanes.Cost[0x1010], w.savedCellPlanes.CostKnown[0x1010] = 32, 1
	if got := w.readCost(layered, true); got != 8 {
		t.Fatalf("the first read answers %d, want the baseline 8", got)
	}
	decayed := w.savedCellPlanes.Cost[0x1010]
	if decayed == 32 {
		t.Fatal("the read left the full byte, so the witness is blind")
	}
	loaded := acLoaded(t, w)
	loaded.ResetLoadedAreaCosts()
	if got := loaded.savedCellPlanes.Cost[0x1010]; got != decayed {
		t.Errorf("the loaded cell holds %d, the continuing world %d", got, decayed)
	}
	if c, l := w.readCost(layered, true), loaded.readCost(layered, true); c != l {
		t.Errorf("the next read answers %d continuing and %d loaded", c, l)
	}
	// Loss control: rewriting the loaded plane restores the full byte.
	bare := acLoaded(t, w)
	bare.RewriteImportedLayerCosts()
	if got := bare.savedCellPlanes.Cost[0x1010]; got != 32 {
		t.Errorf("the rewritten control holds %d, want 32", got)
	}
}

// A native world loaded from its form lists every layered cell at its baseline.
func TestResetLoadedAreaCostsOnANativeWorldRestoresTheContinuingByte(t *testing.T) {
	t.Parallel()
	key := cellKey(2, 1)
	at := cell{x: 2, y: 1}
	for _, reads := range []int{0, 1} {
		w := acWorld(t, 8, nil, []uint16{key}, 3)
		acReads(w, at, reads)
		loaded := acLoaded(t, w)
		bare := acLoaded(t, w)
		loaded.ResetLoadedAreaCosts()
		if !loaded.areaCostLive {
			t.Errorf("%d prior reads: the reset world holds no entries for the first tick", reads)
		}
		if bare.areaCostLive {
			t.Errorf("%d prior reads: a bare restore claims a first area pass", reads)
		}
		want, _, _ := w.areaStored(key, 8)
		if got, _, ok := loaded.areaStored(key, 8); !ok || got != want {
			t.Errorf("%d prior reads: the loaded entry stores %d (layered %v), the continuing world %d", reads, got, ok, want)
		}
		if c, l := w.readCost(at, true), loaded.readCost(at, true); c != l {
			t.Errorf("%d prior reads: the continuing first read answers %d, the loaded one %d", reads, c, l)
		}
	}
}

// acPlaneWorld is a world over a 32 by 32 map whose saved cell plane holds one
// layered cell, 0x1010, with a baseline of 8 and a Wall of Fire layer, and one
// bare cell, 0x1313, so a plane read has both to tell apart.
func acPlaneWorld(t *testing.T) *World { return acPlaneWorldWith(t, nil) }

func acPlaneWorldWith(t *testing.T, ents []Entity) *World {
	t.Helper()
	w := mustWorld(t, 7, Bounds{32, 32}, ents)
	w.SetSavedCellRecords([]SavedCellRecord{{Cell: 0x1010, LayerCount: 1, SpellEffects: [6]uint32{17}}})
	cells := make([]SavedActorCell, 1)
	cells[0].Cell, cells[0].Payload[0], cells[0].Payload[2] = 0x1010, 8, 1
	binary.LittleEndian.PutUint32(cells[0].Payload[20:], 17)
	p := &SavedCellPlanes{}
	p.Costs[0], p.Costs[5] = 255, 8
	p.Static[0x1010], p.Dynamic[0x1010] = 0x20, 0x20
	if err := w.ImportOriginalActorMotions(nil, cells, nil); err != nil {
		t.Fatal(err)
	}
	if err := w.ImportOriginalCellPlanes(p); err != nil {
		t.Fatal(err)
	}
	if issue := w.recomputeSavedCell(0x1010); issue != "" {
		t.Fatal(issue)
	}
	w.refreshSavedPlaneBlocks()
	return w
}
