package sim

import (
	"encoding/binary"
	"testing"
)

func cellPlaneSlotWorld(t *testing.T, ents []Entity, sacks []Sack) *World {
	t.Helper()
	b := Bounds{Width: 16, Height: 16}
	n := int(b.Width * b.Height)
	cost := make([]byte, n)
	for i := range cost {
		cost[i] = 4
	}
	w, err := NewTerrainWorld(1, b, ModeCanonical, Terrain{Block: make([]byte, n), Cost: cost, Height: make([]byte, n)}, ents, nil)
	if err != nil {
		t.Fatal(err)
	}
	w.sacks = sacks
	if err := w.ConstructSavedCellPlanes(CellPlaneStart{Base: &SavedCellPlanes{Costs: [11]byte{255, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10}}}); err != nil {
		t.Fatal(err)
	}
	return w
}

// A body that reaches the terminal stage leaves its cell slots
// (HERO-DEATH-026), the native form (key 0) included, so the World that saves
// and the World that loads the save hold one slot. Loss control: a living air
// actor on the same cell keeps its slot and its Dynamic bit.
func TestTerminalBodyReleasesItsNativeCellSlot(t *testing.T) {
	w := cellPlaneSlotWorld(t, []Entity{
		{ID: 1, X: 3, Y: 3, HP: 10, MaxHP: 10, Domain: DomainGround},
		{ID: 2, X: 3, Y: 3, HP: 10, MaxHP: 10, Domain: DomainAir},
	}, nil)
	if c := w.motionCell(0x0303); c == nil || c.Ground != (SavedActorSlot{Entity: 1, Bound: true}) {
		t.Fatalf("start slot %+v", c)
	}
	w.entities[0].HP, w.entities[0].Decay = -10, DecayBones
	w.ReconcileTerminalActorRegistry(nil)
	c := w.motionCell(0x0303)
	if c == nil || c.Ground != (SavedActorSlot{}) || w.savedCellPlanes.Dynamic[0x0303]&0x40 != 0 {
		t.Fatalf("terminal body kept its slot: %+v Dynamic %#x", c, w.savedCellPlanes.Dynamic[0x0303])
	}
	if c.Air != (SavedActorSlot{Entity: 2, Bound: true}) || w.savedCellPlanes.Dynamic[0x0303]&0x80 == 0 {
		t.Fatalf("the living air actor lost its slot: %+v Dynamic %#x", c.Air, w.savedCellPlanes.Dynamic[0x0303])
	}
}

// A Sack on the map at mission start and a Sack placed in play enter their
// cell record (SAV-SACKENTRY-590) under the key the cell names; the pick-up
// path removes it (SAV-SACKREMOVE-591). Loss control: a World without cell
// planes holds no record for a placed Sack.
func TestPlacedSackEntersItsCellRecord(t *testing.T) {
	w := cellPlaneSlotWorld(t, []Entity{{ID: 1, X: 9, Y: 9, HP: 10, MaxHP: 10, Domain: DomainGround}},
		[]Sack{makeSack(5, 5, 7, nil)})
	for _, key := range []uint16{0x0505, 0x0606} {
		if key == 0x0606 {
			w.pourSack(6, 6, 3, nil)
		}
		c := w.motionCell(key)
		if c == nil || binary.LittleEndian.Uint32(c.Payload[16:]) != sackCellKey(key) ||
			w.savedCellPlanes.Static[key]&0x20 == 0 || w.savedCellPlanes.Dynamic[key]&0x20 == 0 {
			t.Fatalf("Sack cell %04x record %+v Static %#x", key, c, w.savedCellPlanes.Static[key])
		}
	}
	if _, issue := w.removeSavedSackCell(0x0606); issue != "" || w.motionCell(0x0606) != nil || w.savedCellPlanes.Static[0x0606]&0x20 != 0 {
		t.Fatalf("Sack removal kept the record: %q", issue)
	}
	w.savedCellPlanes = nil
	w.pourSack(7, 7, 1, nil)
	if w.motionCell(0x0707) != nil {
		t.Fatal("a World without planes recorded a Sack cell")
	}
}

// An engine SAV written without the structure carrier flag still lists its
// Buildings, and its cell records name them: a World with cell planes keeps
// the imported roster. Loss control: an empty roster holds no carrier.
func TestPlanesWorldKeepsAnImportedRosterWithoutTheCarrierFlag(t *testing.T) {
	for _, roster := range []int{1, 0} {
		w := cellPlaneSlotWorld(t, []Entity{{ID: 1, X: 9, Y: 9, HP: 10, MaxHP: 10, Domain: DomainGround}}, nil)
		p := w.CurrentPolicy()
		p.StructureCarrier = false
		w.hasSavedStructures = true
		w.savedStructures = make([]SavedStructure, roster)
		if err := w.restoreCurrentPolicy(p); err != nil {
			t.Fatal(err)
		}
		if w.hasSavedStructures != (roster != 0) {
			t.Fatalf("roster %d: carrier %v", roster, w.hasSavedStructures)
		}
	}
}

// A LOAD keeps a first-game World's cell planes, whose bytes carry no magic
// wall; the restore marks each standing wall cloud's cells in the native grid
// as it did when it dropped the planes. Loss control: the cell outside the
// cloud stays unmarked.
func TestPlanesWorldRestoreMarksStandingWallClouds(t *testing.T) {
	w := cellPlaneSlotWorld(t, []Entity{{ID: 1, X: 9, Y: 9, HP: 10, MaxHP: 10, Domain: DomainGround}}, nil)
	w.effects = append(w.effects, cellEffect{Key: 0x0505, Spell: 19, Remaining: 200, Mode: areaModeCloud, Cells: []uint16{0x0505}})
	p := w.CurrentPolicy()
	if err := w.restoreCurrentPolicy(p); err != nil {
		t.Fatal(err)
	}
	if w.savedCellPlanes == nil || w.grid[5*16+5]&blockMagicWall == 0 || w.grid[6*16+6]&blockMagicWall != 0 {
		t.Fatalf("wall cell grid %#x, open cell %#x", w.grid[5*16+5], w.grid[6*16+6])
	}
}

// A loaded document can bind a native actor's slot on a cell the actor does
// not stand on. The LOAD's registry reconciliation moves the slot to the
// actor's footprint in the native form. Loss control: an actor sharing a
// cell whose slot another holds (a refused entry) gains no slot.
func TestNativeActorStaleSlotMovesToItsFootprint(t *testing.T) {
	w := cellPlaneSlotWorld(t, []Entity{
		{ID: 1, X: 3, Y: 3, HP: 10, MaxHP: 10, Domain: DomainGround},
		{ID: 2, X: 3, Y: 3, HP: 10, MaxHP: 10, Domain: DomainGround},
	}, nil)
	w.entities[0].X = 8
	w.ReconcileNativeActorRegistry(map[EntityID]uint32{1: 77})
	if c := w.motionCell(0x0308); c == nil || c.Ground != (SavedActorSlot{Entity: 1, Bound: true}) || w.savedCellPlanes.Dynamic[0x0308]&0x60 != 0x60 {
		t.Fatalf("footprint slot %+v", c)
	}
	if c := w.motionCell(0x0303); c != nil && c.Ground != (SavedActorSlot{}) {
		t.Fatalf("stale slot kept %+v, and actor 2 gained it", c.Ground)
	}
}

// A script's removal and placement (instants 16 to 18) leave and enter the
// actor's cells, so its slot follows it (DIV-3099). Loss control: the cell it
// left holds no slot of it.
func TestMapPresenceMovesTheActorsCellSlot(t *testing.T) {
	w := cellPlaneSlotWorld(t, []Entity{{ID: 1, X: 3, Y: 3, HP: 10, MaxHP: 10, Domain: DomainGround}}, nil)
	if err := w.HeadlessPlace(1, 9, 9); err != nil {
		t.Fatal(err)
	}
	e, _ := w.Entity(1)
	key := uint16(e.Y)<<8 | uint16(e.X)
	if c := w.motionCell(key); c == nil || c.Ground != (SavedActorSlot{Entity: 1, Bound: true}) || w.savedCellPlanes.Dynamic[key]&0x40 == 0 {
		t.Fatalf("placed actor at %d,%d holds no slot: %+v", e.X, e.Y, c)
	}
	if c := w.motionCell(0x0303); c != nil && c.Ground != (SavedActorSlot{}) || w.savedCellPlanes.Dynamic[0x0303]&0x40 != 0 {
		t.Fatalf("the cell left behind keeps a slot: %+v", c)
	}
}
