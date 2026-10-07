package sim

import (
	"encoding/binary"
	"reflect"
	"testing"
)

// A size-n mover stores its footprint's top-left cell, and the centre every
// reader takes is that corner plus (n-1)*128 per axis. The expected cells are
// worked here from explicit 8.8 coordinates.
func TestFleeCellMeasuresAWideMoverAtItsFootprintCentre(t *testing.T) {
	t.Parallel()
	hostile := withdrawalFighter(2, 3, 24, 20, 100)
	for _, tc := range []struct {
		name         string
		size         uint8
		wantX, wantY int32
	}{
		// Centre 20*256+128 = 5248; hostile centre 6272; dx = -1024, dy = 0
		// becomes +1; three cells west give 4480 and 5248, cell (17,20).
		{"size 1", 1, 17, 20},
		// Centre 5376 on both axes; dx = -896, dy = 128; x moves to 4608
		// (cell 18) and y by 768*128/896 = 109 to 5485 (cell 21).
		{"size 2", 2, 18, 21},
		// Centre 5504 on both axes; dx = -768, dy = 256; x moves to 4736
		// (cell 18) and y by 768*256/768 = 256 to 5760 (cell 22).
		{"size 3", 3, 18, 22},
	} {
		self := withdrawalFighter(1, 2, 20, 20, 10)
		self.TokenSize = tc.size
		w := engWorld(t, engRel(t, [3]uint32{2, 3, 1}), self, hostile)
		x, y, ok := w.fleeCell(0, []int{1})
		if !ok || x != tc.wantX || y != tc.wantY {
			t.Errorf("%s: flees to (%d,%d,%t), want (%d,%d,true)", tc.name, x, y, ok, tc.wantX, tc.wantY)
		}
	}
}

// A wide hostile is read at its centre too.
func TestFleeCellMeasuresAWideHostileAtItsFootprintCentre(t *testing.T) {
	t.Parallel()
	self := withdrawalFighter(1, 2, 20, 20, 10)
	hostile := withdrawalFighter(2, 3, 22, 18, 100)
	hostile.TokenSize = 3
	w := engWorld(t, engRel(t, [3]uint32{2, 3, 1}), self, hostile)
	// Hostile centre (22*256+128+256, 18*256+128+256) = (6016, 4992); self
	// (5248, 5248). dx = -768, dy = 256, so x is the major axis: x moves
	// -768 to 4480 (cell 17) and y by 768*256/768 = 256 to 5504 (cell 21).
	x, y, ok := w.fleeCell(0, []int{1})
	if !ok || x != 17 || y != 21 {
		t.Fatalf("flees to (%d,%d,%t), want (17,21,true)", x, y, ok)
	}
}

// crossingFixture is one imported mover of footprint one halfway to a cell
// boundary, with its saved route, cell records and blocks.
func crossingFixture(t *testing.T, fineX, fineY uint8, stepX, stepY int8, direction, elapsed, total uint16) (*World, SavedActorMotion, []SavedActorCell, []SavedActorBlock) {
	t.Helper()
	w := mustWorld(t, 7, Bounds{32, 32}, []Entity{{ID: 7, Owner: 1, X: 15, Y: 16, HP: 100, MaxHP: 100, Speed: 1000, Facing: 64}})
	o := SavedActorOrder{Entity: 7, State: 0xb}
	o.Raw[8], o.Raw[9], o.Raw[10], o.Raw[11] = 1, 3, 20, 16
	g := SavedGroup{ID: 1, Selector: 1, Members: []SavedGroupMember{{Archive: 1, Entity: 7, Bound: true}}}
	g.AI[0x20], g.AI[0x45] = 4, 1
	if err := w.ImportSavedGroups([]SavedGroup{g}, []SavedActorOrder{o}); err != nil {
		t.Fatal(err)
	}
	m := SavedActorMotion{Entity: 7, Position: SavedActorPosition{Cell: 0x100f, PackedCell: 0x100f, FineX: fineX, FineY: fineY, Residue: 0x9234, TerrainKey: 0x31415926}, StaticRoute: []uint16{0x1010, 0x1011}, DynamicRoute: []uint16{0x1010}}
	m.ActorAction = 77
	m.Mover[0], m.Mover[1], m.Mover[10], m.Mover[33] = 64, 64, 100, 0xcd
	for _, at := range []int{6, 0x74, 0x76, 0x80, 0xa6} {
		binary.LittleEndian.PutUint16(m.Mover[at:], 0x1010)
	}
	binary.LittleEndian.PutUint16(m.Mover[0x70:], 0x100f)
	binary.LittleEndian.PutUint16(m.Mover[0xa8:], 32)
	binary.LittleEndian.PutUint16(m.Mover[0xaa:], total)
	binary.LittleEndian.PutUint16(m.Mover[0xac:], elapsed)
	binary.LittleEndian.PutUint16(m.Mover[0xae:], direction)
	m.Mover[0xb0], m.Mover[0xb1] = byte(stepX), byte(stepY)
	cells := []SavedActorCell{{Cell: 0x100f, Ground: SavedActorSlot{Key: 0x1004, Entity: 7, Bound: true}}, {Cell: 0x1010}}
	binary.LittleEndian.PutUint32(cells[0].Payload[4:], 0x1004)
	cells[0].Payload[17], cells[1].Payload[17] = 0x92, 0x93
	blocks := []SavedActorBlock{{Cell: 0x100f, Dyn: 0x43, Static: 7}, {Cell: 0x1010, Dyn: 0x45, Static: 9}}
	return w, m, cells, blocks
}

// refusedEntryFixture is a mover of footprint two crossing from (15,16) to
// (16,16) with a saved cell plane. The cell record at (15,17) holds no slot,
// so the release stops there and leaves (16,17) held by the mover itself; the
// occupy then refuses (16,17), and the cell records at (17,16) and (17,17)
// are the ones the loop may or may not reach.
func refusedEntryFixture(t *testing.T) *World {
	t.Helper()
	w, m, _, _ := crossingFixture(t, 224, 128, 32, 0, 2, 3, 8)
	w.entities[0].TokenSize = 2
	w.spells = []SpellRule{{ID: 13, School: 3, DamageMin: 20, DamageMax: 20, Damaging: true, TargetsUnit: true}}
	held := SavedActorSlot{Key: 0x1004, Entity: 7, Bound: true}
	mk := func(key uint16, slot SavedActorSlot) SavedActorCell {
		c := SavedActorCell{Cell: key, Ground: slot}
		c.Payload[0] = 7
		binary.LittleEndian.PutUint32(c.Payload[4:], slot.Key)
		return c
	}
	cells := []SavedActorCell{
		mk(0x100f, held), mk(0x1010, held), mk(0x110f, SavedActorSlot{}), mk(0x1110, held),
		mk(0x1011, SavedActorSlot{}), mk(0x1111, SavedActorSlot{}),
	}
	bs := []SavedActorBlock{{Cell: 0x100f, Dyn: 0x43, Static: 7}, {Cell: 0x1010, Dyn: 0x45, Static: 9}}
	if err := w.ImportOriginalActorMotions([]SavedActorMotion{m}, cells, bs); err != nil {
		t.Fatal(err)
	}
	p := new(SavedCellPlanes)
	p.Costs[0], p.Costs[5] = 255, 6
	for _, k := range []uint16{0x100f, 0x1010, 0x110f, 0x1110, 0x1011, 0x1111} {
		p.Cost[k], p.CostKnown[k], p.Static[k] = 33, 1, 0x20
	}
	if err := w.ImportOriginalCellPlanes(p); err != nil {
		t.Fatal(err)
	}
	// Trigger tails on the refused cell and on the cell after it.
	w.writeCellTail(0x1110, [6]byte{13, 1, 9, 10, 11, 12})
	w.writeCellTail(0x1111, [6]byte{13, 1, 5, 6, 7, 8})
	return w
}

// The refused entry of a crossing: the release ends at the first cell with no
// held slot and leaves the later cells as they were; the occupy builds the
// trigger caster of each cell it enters before testing the slot, a taken slot
// ends it with no store and no recompute, and the cells after it are neither
// entered nor given a caster. The step itself goes on and the crossing leaves
// no coverage issue beyond the speed callback every crossing leaves.
func TestRefusedEntryEndsTheOccupyLoopAndTheCrossingGoesOn(t *testing.T) {
	t.Parallel()
	w := refusedEntryFixture(t)
	Step(w, nil)
	e := w.Entities()[0]
	if e.X != 16 || e.Y != 16 {
		t.Fatalf("the mover stands at (%d,%d), want its crossing's cell (16,16)", e.X, e.Y)
	}
	if got := w.motionFor(7).Issue; got != "original boundary speed callback is not executed" {
		t.Errorf("the crossing's issue is %q, want the speed callback alone", got)
	}
	slot := func(key uint16) SavedActorSlot {
		c := w.motionCell(key)
		if c == nil {
			return SavedActorSlot{}
		}
		return c.Ground
	}
	held := SavedActorSlot{Key: 0x1004, Entity: 7, Bound: true}
	if got := slot(0x1010); got != held {
		t.Errorf("(16,16) holds %+v, want the mover after release and occupy", got)
	}
	if got := slot(0x1011); got != held {
		t.Errorf("(17,16) holds %+v, want the mover: the loop reached it before the refusal", got)
	}
	if got := slot(0x110f); got != (SavedActorSlot{}) {
		t.Errorf("(15,17) holds %+v, want the empty slot it had", got)
	}
	if got := slot(0x1110); got != held {
		t.Errorf("(16,17) holds %+v: the release ended at (15,17) before it, and the occupy found it taken", got)
	}
	if got := w.savedCellPlanes.Cost[0x1110]; got != 33 {
		t.Errorf("the refused cell's cost is %d, want its untouched 33: a refusal skips the recompute", got)
	}
	if got := w.savedCellPlanes.Cost[0x1111]; got != 33 || w.motionCell(0x1111).Ground.Key != 0 {
		t.Errorf("(17,17) cost %d slot %+v, want it neither entered nor recomputed", got, w.motionCell(0x1111).Ground)
	}
	// One caster: the refused cell's. The cell after it was never entered.
	want := []ScriptCast{{FromX: 9, FromY: 10, Spell: 13, Power: 1, Target: 7, AtUnit: true}}
	if got := w.ScriptCasts(); !reflect.DeepEqual(got, want) {
		t.Errorf("pending trigger casts %+v, want %+v", got, want)
	}
	// The same state continues alike after SAVE and a cold LOAD.
	var cold World
	if err := cold.UnmarshalBinary(mustMarshal(t, w)); err != nil {
		t.Fatal(err)
	}
	for tick := 0; tick < 4; tick++ {
		Step(w, nil)
		Step(&cold, nil)
		if w.Hash() != cold.Hash() {
			t.Fatalf("tick %d after the crossing: the cold world differs", tick+1)
		}
	}
}

// A mover felled in transit lies on the cell its stride committed, and its
// teardown recomputes that cell. The cell it left keeps the byte a read
// decayed: the body is not stored there.
func TestTeardownRecomputesTheBodyCellOfAMoverFelledInTransit(t *testing.T) {
	t.Parallel()
	b := Bounds{Width: 12, Height: 8}
	cost := make([]byte, int(b.Width*b.Height))
	for i := range cost {
		cost[i] = 8
	}
	w, err := NewTerrainWorld(1, b, ModeCanonical, Terrain{Cost: cost},
		[]Entity{{ID: 1, X: 1, Y: 1, Speed: 16, HP: 20, MaxHP: 20, DyingTime: 1}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	left, body := cell{1, 1}, cell{2, 1}
	keys := []uint16{cellKey(left.x, left.y), cellKey(body.x, body.y)}
	w.effects = []cellEffect{{Key: keys[0], Spell: 3, Remaining: 2000, Mode: areaModeCloud, Cells: keys}}
	w.syncAreaCosts()
	stored := func(c cell) uint8 { a, _, _ := w.areaStored(cellKey(c.x, c.y), 8); return a }
	Step(w, []Command{acMove(1, 2, 1)})
	if e := w.Entities()[0]; !e.Stride.Present || e.Transit == 0 || e.X != 2 {
		t.Fatalf("no transit to cut short: %+v", e)
	}
	leftBefore := stored(left)
	if leftBefore == 32 || stored(body) == 32 {
		t.Fatalf("before the kill the cells store %d and %d, want decayed bytes", leftBefore, stored(body))
	}
	Step(w, []Command{TerminalKill(1)})
	for i := 0; i < 3 && stored(body) != 32; i++ {
		Step(w, nil)
	}
	if got := stored(body); got != 32 {
		t.Errorf("the body's cell stores %d after the teardown, want the recomputed 32", got)
	}
	if got := stored(left); got != leftBefore {
		t.Errorf("the cell the mover left stores %d, want its decayed %d", got, leftBefore)
	}
}

// The teardown of a mover at rest recomputes every cell of its footprint.
func TestTeardownRecomputesTheFootprintOfAMoverAtRest(t *testing.T) {
	t.Parallel()
	b := Bounds{Width: 12, Height: 8}
	cost := make([]byte, int(b.Width*b.Height))
	for i := range cost {
		cost[i] = 8
	}
	w, err := NewTerrainWorld(1, b, ModeCanonical, Terrain{Cost: cost},
		[]Entity{{ID: 1, X: 2, Y: 1, HP: 20, MaxHP: 20, DyingTime: 1, TokenSize: 2}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	inside, outside := []cell{{2, 1}, {3, 1}, {2, 2}, {3, 2}}, cell{4, 1}
	var keys []uint16
	for _, c := range append([]cell{outside}, inside...) {
		keys = append(keys, cellKey(c.x, c.y))
	}
	w.effects = []cellEffect{{Key: keys[0], Spell: 3, Remaining: 2000, Mode: areaModeCloud, Cells: keys}}
	w.syncAreaCosts()
	for _, c := range append([]cell{outside}, inside...) {
		acReads(w, c, 2)
	}
	stored := func(c cell) uint8 { a, _, _ := w.areaStored(cellKey(c.x, c.y), 8); return a }
	Step(w, []Command{TerminalKill(1)})
	for i := 0; i < 3 && stored(inside[0]) == 2; i++ {
		Step(w, nil)
	}
	for _, c := range inside {
		if got := stored(c); got != 32 {
			t.Errorf("footprint cell %v stores %d after the teardown, want the recomputed 32", c, got)
		}
	}
	if got := stored(outside); got != 2 {
		t.Errorf("the cell beside the footprint stores %d, want its decayed 2", got)
	}
}

// A script cast of Prismatic Spray is made by a temporary caster that has no
// owner and no group, so no candidate group stands behind it and the primary is
// hit alone, whoever else stands in sight of the party.
func TestSprayCastByATemporaryCasterHitsThePrimaryAlone(t *testing.T) {
	t.Parallel()
	foes := func() []Entity {
		return []Entity{prismaticFoeAt(2, 8, 8), prismaticFoeAt(3, 2, 1), prismaticFoeAt(4, 3, 3)}
	}
	w := prismaticWorld(t, 100, foes()...)
	w.castAtUnit(unitNode(5, 5, 14, 100, 2))
	Step(w, nil)
	var got []EntityID
	for _, d := range w.deliveries {
		got = append(got, d.Target)
	}
	if want := []EntityID{2}; !reflect.DeepEqual(got, want) {
		t.Fatalf("the script's spray queued %v, want the primary alone %v", got, want)
	}
	// Control: the same spray cast by the party's mage takes the nearest foes.
	_, ids := prismaticCast(t, prismaticWorld(t, 100, foes()...), 2)
	if want := []EntityID{2, 3, 4}; !reflect.DeepEqual(ids, want) {
		t.Fatalf("a mage's spray queued %v, want %v", ids, want)
	}
}
