package sim

import (
	"bytes"
	"testing"
)

// slotWorlds builds the two kinds of World over one layered cell: key is held
// by actor 1, actor 2 stands beside it. stored reads the cell's cost byte as
// each kind keeps it.
func slotWorlds(t *testing.T) (key uint16, worlds map[string]*World, stored func(*World) uint8) {
	t.Helper()
	key = cellKey(2, 1)
	ents := []Entity{{ID: 1, X: 2, Y: 1, HP: 10, MaxHP: 10}, {ID: 2, X: 1, Y: 1, HP: 10, MaxHP: 10}}
	plain, err := NewTerrainWorld(1, srBounds, ModeCanonical, Terrain{Cost: srPlane(8), StartSlots: true}, ents, nil)
	if err != nil {
		t.Fatal(err)
	}
	plain.effects = []cellEffect{{Key: key, Spell: 3, Remaining: 200, Mode: areaModeCloud, Cells: []uint16{key}}}
	plain.syncAreaCosts()
	p, err := NewTerrainWorld(1, srBounds, ModeCanonical, Terrain{Cost: srPlane(8), Base: &SavedCellPlanes{Costs: [11]byte{255, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10}}}, ents, nil)
	if err != nil {
		t.Fatal(err)
	}
	p.effects = []cellEffect{{Key: key, Spell: 3, Remaining: 200, Mode: areaModeCloud, Cells: []uint16{key}}}
	p.recomputeSavedCell(key)
	stored = func(w *World) uint8 {
		if w.savedCellPlanes != nil {
			return w.savedCellPlanes.Cost[key]
		}
		b, _, _ := w.areaStored(key, 8)
		return b
	}
	return key, map[string]*World{"without planes": plain, "with planes": p}, stored
}

// A second walker entering a layered cell another actor holds is refused its
// slot, so the cell keeps the byte the reads decayed (MOVE-088); its departure
// clears the held slot whoever holds it and recomputes (SAV-CELLLEAVE-584).
// One builder answers both kinds of World alike.
func TestCrossingKeepsADecayedCellWhileASlotIsHeld(t *testing.T) {
	key, worlds, stored := slotWorlds(t)
	for name, w := range worlds {
		t.Run(name, func(t *testing.T) {
			if c := w.motionCell(key); c == nil || c.Ground != (SavedActorSlot{Entity: 1, Bound: true}) {
				t.Fatalf("mission start left the slot %+v", c)
			}
			fresh := stored(w)
			acReads(w, cell{x: 2, y: 1}, 1)
			decayed := stored(w)
			if decayed >= fresh {
				t.Fatalf("the read left %d from %d", decayed, fresh)
			}
			w.recomputeCrossing(w.entities[1], cell{x: 1, y: 1}, cell{x: 2, y: 1})
			if got := stored(w); got != decayed {
				t.Errorf("entering a held cell left %d, want the decayed %d", got, decayed)
			}
			if c := w.motionCell(key); c == nil || c.Ground.Entity != 1 {
				t.Errorf("the entry took the held slot: %+v", c)
			}
			w.recomputeCrossing(w.entities[1], cell{x: 2, y: 1}, cell{x: 3, y: 1})
			if got := stored(w); got != fresh {
				t.Errorf("the departure left %d, want the recomputed %d", got, fresh)
			}
			if c := w.motionCell(key); c != nil && c.Ground != (SavedActorSlot{}) {
				t.Errorf("the departure left the slot %+v", c.Ground)
			}
			if err := w.cellRecordFault(); err != nil {
				t.Fatal(err)
			}
		})
	}
}

// A World without planes keeps its slot table through its byte form: the
// loaded World answers the next crossing as the continuing one does.
func TestSlotTableSurvivesTheByteForm(t *testing.T) {
	key, worlds, stored := slotWorlds(t)
	w := worlds["without planes"]
	acReads(w, cell{x: 2, y: 1}, 1)
	w.recomputeCrossing(w.entities[1], cell{x: 1, y: 1}, cell{x: 2, y: 1})
	loaded := acLoaded(t, w)
	if !bytes.Equal(mustMarshal(t, w), mustMarshal(t, loaded)) {
		t.Fatal("the loaded form differs from the continuing one")
	}
	c := loaded.motionCell(key)
	if c == nil || c.Ground != (SavedActorSlot{Entity: 1, Bound: true}) || loaded.motionCell(cellKey(1, 1)) != nil {
		t.Fatalf("the loaded slot table holds %+v", c)
	}
	loaded.ResetLoadedAreaCosts()
	for _, x := range []*World{w, loaded} {
		x.recomputeCrossing(x.entities[1], cell{x: 2, y: 1}, cell{x: 3, y: 1})
	}
	if a, b := stored(w), stored(loaded); a != b {
		t.Errorf("the departure left %d continuing and %d loaded", a, b)
	}
}

// A script's removal frees an actor's cell slots and its return takes them
// again, in a World with planes and in one without alike.
func TestPresenceRemovalAndReturnMoveTheSlots(t *testing.T) {
	key, worlds, stored := slotWorlds(t)
	for name, w := range worlds {
		t.Run(name, func(t *testing.T) {
			held := SavedActorSlot{Entity: 1, Bound: true}
			if c := w.motionCell(key); c == nil || c.Ground != held {
				t.Fatalf("mission start left the slot %+v", c)
			}
			fresh := stored(w)
			acReads(w, cell{x: 2, y: 1}, 1)
			if !w.takeOffMap(0) {
				t.Fatal("the removal did not happen")
			}
			if c := w.motionCell(key); c != nil && c.Ground != (SavedActorSlot{}) {
				t.Errorf("the removal left the slot %+v", c.Ground)
			}
			if got := stored(w); got != fresh {
				t.Errorf("the removal left the byte %d, want the recomputed %d", got, fresh)
			}
			if !w.placeAt(0, 2, 1) {
				t.Fatal("the return did not place")
			}
			if c := w.motionCell(key); c == nil || c.Ground.Entity != 1 || !c.Ground.Bound {
				t.Errorf("the return took no slot: %+v", c)
			}
		})
	}
}
