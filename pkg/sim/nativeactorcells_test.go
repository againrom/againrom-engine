package sim

import (
	"encoding/binary"
	"reflect"
	"testing"
)

func TestNativeActorRegistryRestoresFootprintWithoutReplacingOriginalMotion(t *testing.T) {
	w := mustWorld(t, 7, Bounds{32, 32}, []Entity{
		{ID: 1, Owner: SelfSlot, X: 12, Y: 12, HP: 20, MaxHP: 20, TokenSize: 2},
		{ID: 2, Owner: SelfSlot, X: 8, Y: 8, HP: 20, MaxHP: 20, Domain: DomainAir},
		{ID: 3, Owner: SelfSlot, X: 16, Y: 16, HP: 20, MaxHP: 20},
		{ID: 4, Owner: SelfSlot, X: 20, Y: 20, HP: 0, MaxHP: 20},
		{ID: 5, Owner: SelfSlot, X: 5, Y: 5, HP: 20, MaxHP: 20},
	})
	w.entities[0].SourceBinding = SourceBinding{Class: 1, ArchiveIndex: 1, Identity: 0x1000}
	w.savedMotion = &savedActorMotionState{Motions: []SavedActorMotion{
		{Entity: 1}, {Entity: 2, Current: true}, {Entity: 3, Current: true, Active: true}, {Entity: 4},
	}}
	binary.LittleEndian.PutUint16(w.savedMotion.Motions[2].Mover[0x80:], 0x0809)
	w.savedCellPlanes = &SavedCellPlanes{}
	for y := 0; y < 32; y++ {
		for x := 0; x < 32; x++ {
			key := y*256 + x
			w.savedCellPlanes.Cost[key], w.savedCellPlanes.CostKnown[key] = 8, 1
		}
	}
	for _, key := range []uint16{0x0808, 0x0809, 0x0908, 0x0909} {
		c := SavedActorCell{Cell: key, Ground: SavedActorSlot{Key: 0x1000, Entity: 1, Bound: true}}
		c.Payload[0], c.Payload[2] = 8, 6
		binary.LittleEndian.PutUint32(c.Payload[4:], 0x1000)
		if key == 0x0808 {
			c.Air = SavedActorSlot{Key: 0x2000, Entity: 2, Bound: true}
			binary.LittleEndian.PutUint32(c.Payload[8:], 0x2000)
			w.savedCellPlanes.Dynamic[key] = 0x80
		}
		w.savedMotion.Cells = append(w.savedMotion.Cells, c)
		w.savedCellRecords = append(w.savedCellRecords, SavedCellRecord{Cell: key, Ground: SavedCellActorSlot{Key: 0x1000, Entity: 1, Bound: true}})
		w.savedCellPlanes.Static[key] = 0x20
		w.savedCellPlanes.Dynamic[key] |= 0x60
	}
	untouched := SavedActorCell{Cell: 0x1413, Ground: SavedActorSlot{Key: 0x4000, Entity: 4, Bound: true}}
	w.savedMotion.Cells = append(w.savedMotion.Cells, untouched)
	original := append([]SavedActorMotion(nil), w.savedMotion.Motions[1:]...)
	w.ReconcileNativeActorRegistry(map[EntityID]uint32{5: 0x5000})
	if w.motionCell(0x0505) != nil {
		t.Fatal("actor identity without a motion carrier acquired an invalid bound slot")
	}
	if !reflect.DeepEqual(original, w.savedMotion.Motions[1:]) || *w.motionCell(untouched.Cell) != untouched {
		t.Fatal("native repair changed original motion or a healable body")
	}
	for _, key := range []uint16{0x0c0c, 0x0c0d, 0x0d0c, 0x0d0d} {
		c := w.motionCell(key)
		if c == nil || c.Ground.Key != 0x1000 || !c.Ground.Bound || c.Ground.Entity != 1 || binary.LittleEndian.Uint32(c.Payload[4:]) != 0x1000 || w.savedCellPlanes.Dynamic[key]&0x60 != 0x60 {
			t.Fatalf("current footprint cell %04x is not occupied", key)
		}
	}
	for _, key := range []uint16{0x0808, 0x0809, 0x0908, 0x0909} {
		c := w.motionCell(key)
		if c.Ground.Key != 0 || binary.LittleEndian.Uint32(c.Payload[4:]) != 0 || c.Payload[2] != 6 {
			t.Fatal("stale footprint retained or unrelated payload changed")
		}
		want := byte(0x20)
		if key == 0x0808 {
			want |= 0x80
		}
		if key == 0x0809 {
			want |= 0x40
		}
		if w.savedCellPlanes.Dynamic[key] != want {
			t.Fatalf("cell %04x flags=%x want=%x", key, w.savedCellPlanes.Dynamic[key], want)
		}
	}
	for _, c := range w.savedCellRecords {
		if c.Ground.Key != 0 {
			t.Fatal("carried cells retained stale ground slot")
		}
	}
	before := w.Hash()
	w.ReconcileNativeActorRegistry(nil)
	if w.Hash() != before {
		t.Fatal("native registry reconciliation is not idempotent")
	}
}

func TestNativeMovementCarriesActorCellRegistry(t *testing.T) {
	for _, domain := range []Domain{DomainGround, DomainAir} {
		w := mustWorld(t, 7, Bounds{16, 16}, []Entity{{ID: 1, Owner: SelfSlot, X: 3, Y: 2, HP: 20, MaxHP: 20, Domain: domain, Speed: 100, RotationSpeed: 255}})
		w.entities[0].SourceBinding = SourceBinding{Class: 1, ArchiveIndex: 1, Identity: 0x1000}
		layer := domain.layer()
		c := SavedActorCell{Cell: 0x0203}
		c.Payload[0] = 8
		*motionSlot(&c, layer) = SavedActorSlot{Key: 0x1000, Entity: 1, Bound: true}
		binary.LittleEndian.PutUint32(c.Payload[4+4*layer:], 0x1000)
		w.savedMotion = &savedActorMotionState{Motions: []SavedActorMotion{{Entity: 1, Position: SavedActorPosition{Cell: c.Cell, PackedCell: c.Cell, FineX: 128, FineY: 128}, Issue: "native order supersedes original route continuation"}}, Cells: []SavedActorCell{c}}
		w.savedCellPlanes = &SavedCellPlanes{}
		for y := 0; y < 16; y++ {
			for x := 0; x < 16; x++ {
				key := y*256 + x
				w.savedCellPlanes.Cost[key], w.savedCellPlanes.CostKnown[key] = 8, 1
			}
		}
		w.savedCellPlanes.Static[c.Cell], w.savedCellPlanes.Dynamic[c.Cell] = 0x20, 0x20|byte(0x40<<layer)
		Step(w, []Command{MoveTo(1, CellPoint{X: 8, Y: 2})})
		for tick := 0; tick < 128; tick++ {
			Step(w, nil)
		}
		e := w.entities[0]
		if e.X != 8 || e.Y != 2 || e.Transit != 0 {
			t.Fatalf("control did not reach destination: %+v", e)
		}
		seen := 0
		for _, c := range w.savedMotion.Cells {
			if motionSlot(&c, layer).Key == 0x1000 {
				seen++
				if c.Cell != 0x0208 {
					t.Fatalf("domain%d actor reached 8,2 but registry retained %04x", domain, c.Cell)
				}
			}
		}
		if seen != 1 || w.savedCellPlanes.Dynamic[0x0208]&(0x40<<layer) == 0 || w.savedCellPlanes.Dynamic[0x0203]&(0x40<<layer) != 0 {
			t.Fatal("current registry or occupancy bits do not follow native movement")
		}
	}
}

func TestNativeRegistryPreservesImportedCellsUntilMovement(t *testing.T) {
	for _, domain := range []Domain{DomainGround, DomainAir} {
		w := mustWorld(t, 7, Bounds{16, 16}, []Entity{{ID: 1, Owner: SelfSlot, X: 3, Y: 2, HP: 20, MaxHP: 20, Domain: domain, Speed: 100, RotationSpeed: 255}})
		w.entities[0].SourceBinding = SourceBinding{Class: 1, ArchiveIndex: 1, Identity: 0x1000}
		layer := domain.layer()
		w.savedMotion = &savedActorMotionState{Motions: []SavedActorMotion{{Entity: 1, Position: SavedActorPosition{Cell: 0x0203, PackedCell: 0x0203, FineX: 128, FineY: 128}, Issue: "native movement continues the imported route"}}}
		w.savedCellPlanes = &SavedCellPlanes{}
		for y := 0; y < 16; y++ {
			for x := 0; x < 16; x++ {
				key := y*256 + x
				w.savedCellPlanes.Cost[key], w.savedCellPlanes.CostKnown[key] = 8, 1
			}
		}
		for _, key := range []uint16{0x0203, 0x0707} {
			c := SavedActorCell{Cell: key}
			c.Payload[0] = 8
			*motionSlot(&c, layer) = SavedActorSlot{Key: 0x1000, Entity: 1, Bound: true}
			binary.LittleEndian.PutUint32(c.Payload[4+4*layer:], 0x1000)
			w.savedMotion.Cells = append(w.savedMotion.Cells, c)
			w.savedCellPlanes.Static[key], w.savedCellPlanes.Dynamic[key] = 0x20, 0x20|byte(0x40<<layer)
		}
		before := w.Hash()
		w.ReconcileNativeActorRegistry(nil)
		if w.Hash() != before {
			t.Fatal("load changed an imported registry whose current footprint is present")
		}
		Step(w, []Command{MoveTo(1, CellPoint{X: 8, Y: 2})})
		for tick := 0; tick < 128; tick++ {
			Step(w, nil)
		}
		e := w.entities[0]
		if e.X != 8 || e.Y != 2 || e.Transit != 0 {
			t.Fatalf("control did not reach destination: %+v", e)
		}
		for _, key := range []uint16{0x0203, 0x0707} {
			if motionSlot(w.motionCell(key), layer).Key != 0 || w.savedCellPlanes.Dynamic[key]&(0x40<<layer) != 0 {
				t.Fatalf("movement retained old registry cell %04x", key)
			}
		}
		if motionSlot(w.motionCell(0x0208), layer).Key != 0x1000 || w.savedCellPlanes.Dynamic[0x0208]&(0x40<<layer) == 0 {
			t.Fatal("movement did not register its current footprint")
		}
	}
}
