package sim

import (
	"slices"
	"testing"
)

func runtimeIDWorld(t *testing.T) *World {
	t.Helper()
	return dropGoldWorld(t, 500, nil, Entity{ID: 1, X: 10, Y: 20, Owner: SelfSlot}, Entity{ID: 2, X: 30, Y: 20, Owner: SelfSlot})
}

func runtimeIDOf(t *testing.T, w *World, id EntityID) uint32 {
	t.Helper()
	e, ok := w.Entity(id)
	if !ok {
		t.Fatal("missing actor", id)
	}
	return e.SourceBinding.RuntimeID
}

// Insertion order decides the ids; a Sack constructor takes the next lowest
// clear bit and its row keeps it.
func TestRuntimeIDAllocatorGivesTheLowestClearBitInInsertionOrder(t *testing.T) {
	w := runtimeIDWorld(t)
	if got := w.NextRuntimeID(); got != 1 {
		t.Fatalf("empty allocator gives %d, want 1", got)
	}
	w.InsertActorRuntimeIDs([]EntityID{2, 1, 2})
	if runtimeIDOf(t, w, 2) != 1 || runtimeIDOf(t, w, 1) != 2 {
		t.Fatalf("insertion ids %d %d, want 1 2", runtimeIDOf(t, w, 2), runtimeIDOf(t, w, 1))
	}
	Step(w, []Command{DropGold(SelfSlot, 100, CellPoint{X: 11, Y: 21})})
	if got := w.SackRuntimeIDs(); len(got) != 1 || got[0] != 3 {
		t.Fatalf("Sack ids %v, want [3]", got)
	}
	if got := w.RuntimeIDMarks(); !slices.Equal(got, []uint16{0, 1, 2, 3}) || w.NextRuntimeID() != 4 {
		t.Fatalf("marks %v next %d", got, w.NextRuntimeID())
	}
	back := roundTripRuntimeIDs(t, w)
	if got := back.SackRuntimeIDs(); len(got) != 1 || got[0] != 3 || back.NextRuntimeID() != 4 {
		t.Fatalf("LOAD of the byte form lost the Sack id: %v", got)
	}
}

// A map's Sacks take the ids after its actors', in Sacks order, and keep them
// through the byte form.
func TestRuntimeIDMapSacksFollowTheActors(t *testing.T) {
	w := dropGoldWorld(t, 0, []Sack{{X: 12, Y: 22, Gold: 5}, {X: 3, Y: 4, Gold: 7}}, Entity{ID: 1, X: 10, Y: 20, Owner: SelfSlot})
	if got := w.SackRuntimeIDs(); !slices.Equal(got, []uint32{0, 0}) {
		t.Fatalf("map Sacks before mission entry %v, want none", got)
	}
	w.InsertActorRuntimeIDs([]EntityID{1})
	w.InsertSackRuntimeIDs()
	w.InsertSackRuntimeIDs()
	if got := w.SackRuntimeIDs(); runtimeIDOf(t, w, 1) != 1 || !slices.Equal(got, []uint32{2, 3}) {
		t.Fatalf("actor %d, Sacks %v, want 1 and [2 3]", runtimeIDOf(t, w, 1), got)
	}
	if back := roundTripRuntimeIDs(t, w); !slices.Equal(back.SackRuntimeIDs(), []uint32{2, 3}) || back.NextRuntimeID() != 4 {
		t.Fatal("byte form lost the map Sack ids", back.SackRuntimeIDs())
	}
}

func roundTripRuntimeIDs(t *testing.T, w *World) *World {
	t.Helper()
	form := mustMarshal(t, w)
	var back World
	if err := back.UnmarshalBinary(form); err != nil {
		t.Fatal(err)
	}
	if back.Hash() != w.Hash() {
		t.Fatal("byte form round trip changed the World")
	}
	return &back
}

// SAV-1371: bit 0, each saved id in archive order, Items and Effects included,
// each Sack's constructor first taking the lowest clear bit. Bits no object
// carries stay marked and travel in the byte form.
func TestRuntimeIDLoadReplayMarksSavedIDsAndSackConstructorBits(t *testing.T) {
	w := runtimeIDWorld(t)
	w.entities[0].SourceBinding.RuntimeID = 1
	w.entities[1].SourceBinding.RuntimeID = 5
	plain := mustMarshal(t, w)
	if plain[0] != formatVersion {
		t.Fatalf("a World without orphan bits writes version %d", plain[0])
	}
	w.RestoreRuntimeIDs([]RuntimeIDLoad{{Saved: 1}, {Saved: 3}, {Saved: 9, Sack: true}, {Saved: 0x10004}, {Saved: 0x10000}, {}, {Saved: 5}})
	if got := w.RuntimeIDMarks(); !slices.Equal(got, []uint16{0, 1, 2, 3, 4, 5, 9}) {
		t.Fatalf("marks %v", got)
	}
	if !slices.Equal(w.runtimeOrphans, []uint16{2, 3, 4, 9}) || w.NextRuntimeID() != 6 {
		t.Fatalf("orphans %v next %d", w.runtimeOrphans, w.NextRuntimeID())
	}
	back := roundTripRuntimeIDs(t, w)
	if !slices.Equal(back.runtimeOrphans, w.runtimeOrphans) || back.NextRuntimeID() != 6 {
		t.Fatalf("byte form lost the orphan bits: %v", back.runtimeOrphans)
	}
	w.RestoreRuntimeIDs(nil)
	if w.runtimeOrphans != nil || w.NextRuntimeID() != 2 {
		t.Fatalf("an empty replay keeps %v", w.runtimeOrphans)
	}
}

// An orphan bit an object carries, or an unordered orphan list, is no form
// this package writes.
func TestRuntimeIDFaultRefusesCarriedOrUnorderedOrphans(t *testing.T) {
	w := runtimeIDWorld(t)
	w.entities[0].SourceBinding.RuntimeID = 1
	for _, orphans := range [][]uint16{{1}, {4, 3}, {0}, {3, 3}} {
		w.runtimeOrphans = orphans
		if _, err := w.MarshalBinary(); err == nil {
			t.Fatalf("orphans %v accepted", orphans)
		}
	}
}

func TestRuntimeIDHeldCorpseKeepsTheDeadManagerIDWhenPlaced(t *testing.T) {
	w := deadWorld(t)
	if err := w.ImportOriginalDeadActors([]OriginalDeadActor{deadInput(1, 3, -40)}); err != nil {
		t.Fatal(err)
	}
	w.InsertActorRuntimeIDs([]EntityID{1})
	if err := w.HeadlessPlace(1, 6, 6); err != nil {
		t.Fatal(err)
	}
	if got := runtimeIDOf(t, w, 1); got != 0 {
		t.Fatalf("placed corpse took allocator id %d", got)
	}
	if d := w.OriginalDeadActors(); len(d) != 1 || d[0].Current.RuntimeID != 31 {
		t.Fatalf("dead manager id %+v, want 31", d)
	}
	if got := w.RuntimeIDMarks(); !slices.Equal(got, []uint16{0, 31}) || w.NextRuntimeID() != 1 {
		t.Fatalf("marks %v next %d", got, w.NextRuntimeID())
	}
}

func TestRuntimeIDColdLoadOfASackPlantedBeforeAnOlderSackHoldsTheConstructorBit(t *testing.T) {
	for _, tc := range []struct {
		last    int32
		orphans []uint16
	}{{40, []uint16{6}}, {6, nil}} {
		w := dropGoldWorld(t, 500, []Sack{{X: 3, Y: 4, Gold: 5}, {X: 12, Y: tc.last, Gold: 7}}, Entity{ID: 1, X: 10, Y: 20, Owner: SelfSlot}, Entity{ID: 2, X: 30, Y: 20, Owner: SelfSlot})
		w.InsertActorRuntimeIDs([]EntityID{1, 2})
		w.InsertSackRuntimeIDs()
		Step(w, []Command{DropGold(SelfSlot, 100, CellPoint{X: 11, Y: 21})})
		records := []RuntimeIDLoad{{Saved: runtimeIDOf(t, w, 1)}, {Saved: runtimeIDOf(t, w, 2)}}
		for _, id := range w.SackRuntimeIDs() {
			records = append(records, RuntimeIDLoad{Saved: id, Sack: true})
		}
		if len(records) != 5 {
			t.Fatalf("older Sack at row %d left Sacks %v", tc.last, w.SackRuntimeIDs())
		}
		cold := roundTripRuntimeIDs(t, w)
		cold.RestoreRuntimeIDs(records)
		if !slices.Equal(cold.runtimeOrphans, tc.orphans) || !slices.Equal(cold.SackRuntimeIDs(), w.SackRuntimeIDs()) {
			t.Fatalf("older Sack at row %d: Sacks %v, cold orphans %v, want %v", tc.last, w.SackRuntimeIDs(), cold.runtimeOrphans, tc.orphans)
		}
		if (cold.Hash() == w.Hash()) != (tc.orphans == nil) || (cold.NextRuntimeID() == w.NextRuntimeID()) != (tc.orphans == nil) {
			t.Fatalf("older Sack at row %d: continued next %d, cold next %d", tc.last, w.NextRuntimeID(), cold.NextRuntimeID())
		}
		marks := cold.RuntimeIDMarks()
		adopted := make([]RuntimeIDLoad, len(marks))
		for i, id := range marks {
			adopted[i] = RuntimeIDLoad{Saved: uint32(id)}
		}
		w.RestoreRuntimeIDs(adopted)
		if cold.Hash() != w.Hash() || cold.NextRuntimeID() != w.NextRuntimeID() {
			t.Fatalf("older Sack at row %d: the loaded allocator does not make the continued World equal", tc.last)
		}
	}
}
