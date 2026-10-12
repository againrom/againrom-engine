package sim

import (
	"encoding/binary"
	"testing"
)

// Native actors occupy cell slots at mission start and through crossings on a
// World with planes: a taken slot refuses a second entry, a release clears
// whatever the slot holds, and a native actor's slot keeps the native form
// (key 0) after a LOAD bound a key to it.
func TestNativeCellSlots(t *testing.T) {
	b := Bounds{Width: 16, Height: 16}
	n := int(b.Width * b.Height)
	cost, height := make([]byte, n), make([]byte, n)
	for i := range cost {
		cost[i] = 4
	}
	ents := []Entity{
		{ID: 1, X: 3, Y: 3, HP: 10, MaxHP: 10, Domain: DomainGround},
		{ID: 2, X: 3, Y: 3, HP: 10, MaxHP: 10, Domain: DomainGround},
		{ID: 3, X: 3, Y: 3, HP: 10, MaxHP: 10, Domain: DomainAir},
		{ID: 4, X: 8, Y: 8, HP: 10, MaxHP: 10, Domain: DomainGround, OffMap: true},
	}
	w, err := NewTerrainWorld(1, b, ModeCanonical, Terrain{Block: make([]byte, n), Cost: cost, Height: height}, ents, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := w.ConstructSavedCellPlanes(CellPlaneStart{Base: &SavedCellPlanes{Costs: [11]byte{255, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10}}}); err != nil {
		t.Fatal(err)
	}
	native := SavedActorSlot{Entity: 1, Bound: true}
	c := w.motionCell(0x0303)
	if c == nil || c.Ground != native || c.Air != (SavedActorSlot{Entity: 3, Bound: true}) {
		t.Fatalf("mission start slots = %+v", c)
	}
	if p := w.savedCellPlanes; p.Static[0x0303]&0x20 == 0 || p.Dynamic[0x0303]&0xc0 != 0xc0 || w.motionCell(0x0808) != nil {
		t.Fatalf("mission start planes Static %#x Dynamic %#x", p.Static[0x0303], p.Dynamic[0x0303])
	}
	if err := w.cellRecordFault(); err != nil {
		t.Fatal(err)
	}

	w.recomputeCrossing(w.entities[0], cell{3, 3}, cell{4, 3})
	if c := w.motionCell(0x0303); c == nil || c.Ground != (SavedActorSlot{}) {
		t.Fatalf("release left %+v", c)
	}
	if c := w.motionCell(0x0304); c == nil || c.Ground != native {
		t.Fatalf("occupation left %+v", c)
	}
	// Actor 2 never held a slot; its release at (4,3) clears actor 1's slot
	// (MOVE-088), and the emptied record goes.
	w.recomputeCrossing(w.entities[1], cell{4, 3}, cell{5, 3})
	if w.motionCell(0x0304) != nil || w.savedCellPlanes.Static[0x0304]&0x20 != 0 {
		t.Fatalf("a release of another actor's slot kept the record")
	}
	// A LOAD binds the document key 77 to actor 2; the registry gives the
	// slot the native form again, which the document writer keys at SAVE.
	if c := w.motionCell(0x0305); c == nil || c.Ground != (SavedActorSlot{Entity: 2, Bound: true}) {
		t.Fatalf("occupation of (5,3) left %+v", c)
	} else {
		c.Ground.Key = 77
		binary.LittleEndian.PutUint32(c.Payload[4:], 77)
	}
	w.entities[1].X = 5 // recomputeCrossing moves slots, not the actor
	w.ReconcileNativeActorRegistry(map[EntityID]uint32{2: 77})
	if c := w.motionCell(0x0305); c == nil || c.Ground != (SavedActorSlot{Entity: 2, Bound: true}) || binary.LittleEndian.Uint32(c.Payload[4:]) != 0 {
		t.Fatalf("native slot form left %+v", c)
	}
	if c := w.motionCell(0x0303); c.Air.Key != 0 || !c.Air.Bound {
		t.Fatalf("an actor without a key gained one: %+v", c.Air)
	}
	w.recomputeCrossing(w.entities[2], cell{3, 3}, cell{3, 4})
	if w.motionCell(0x0303) != nil || w.motionCell(0x0403) == nil {
		t.Fatalf("the air crossing did not move its slot")
	}
	if err := w.cellRecordFault(); err != nil {
		t.Fatal(err)
	}
}
