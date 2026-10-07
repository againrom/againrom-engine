package sim

import (
	"bytes"
	"testing"
)

func TestLegacyUnitCapacityRepairsOnlyUnboundNativeZero(t *testing.T) {
	entities := []Entity{
		{ID: 1, X: 1, Y: 1, MaxHP: 20, HP: 17, Load: 42, Speed: 9},
		{ID: 2, X: 2, Y: 1, MaxHP: 20, HP: 0},
		{ID: 3, X: 3, Y: 1, Humanoid: true},
		{ID: 4, X: 4, Y: 1, Capacity: 271},
		{ID: 5, X: 5, Y: 1, Capacity: -3},
		{ID: 6, X: 6, Y: 1, ActorLoad: ActorLoad{Present: true}},
		{ID: 7, X: 1, Y: 2},
		{ID: 8, X: 2, Y: 2, SourceBinding: SourceBinding{Class: 1, ArchiveIndex: 1},
			ActorLoad: ActorLoad{Present: true, Source: SourceActor{Class: 1}}},
		{ID: 9, X: 3, Y: 2, SourceBinding: SourceBinding{Class: GeneratedUnitBinding, Identity: 9, RuntimeID: 9},
			ActorLoad: ActorLoad{Present: true, Source: SourceActor{Class: 1}}},
	}
	makeWorld := func() *World {
		t.Helper()
		w, err := NewWorld(41, Bounds{Width: 8, Height: 8}, ModeCanonical, nil, entities)
		if err != nil {
			t.Fatal(err)
		}
		if !w.SetHumanMovement(7, 6, 12) || !w.entities[6].HumanMovement.Present {
			t.Fatal("retained movement control was not installed")
		}
		return w
	}
	w := makeWorld()
	entities[0].Capacity, entities[1].Capacity = 300, 300
	want := makeWorld()
	wantBytes, err := want.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	if got := RepairLegacyUnitCapacity(w, 300); got != 2 {
		t.Fatalf("repaired %d actors, want 2", got)
	}
	gotBytes, err := w.MarshalBinary()
	if err != nil || !bytes.Equal(gotBytes, wantBytes) {
		t.Fatalf("repair changed fields beyond the two native capacities: %v", err)
	}
	if got := RepairLegacyUnitCapacity(w, 300); got != 0 || w.Hash() != want.Hash() {
		t.Fatal("second repair changed the world", got)
	}
}

func TestLegacyUnitCapacityKeepsCurrentRegistryAndInvalidDefault(t *testing.T) {
	for _, capacity := range []int32{0, -1, 300} {
		w, err := NewWorld(2, Bounds{Width: 2, Height: 2}, ModeCanonical, nil, []Entity{{ID: 1}})
		if err != nil {
			t.Fatal(err)
		}
		if capacity == 300 {
			if err := w.ImportSavedObjects(&SavedObjects{Version: SavedObjectsVersion, NextID: 1}, nil); err != nil {
				t.Fatal(err)
			}
		}
		before := w.Hash()
		if got := RepairLegacyUnitCapacity(w, capacity); got != 0 || w.Hash() != before {
			t.Fatalf("capacity %d changed protected world: %d", capacity, got)
		}
	}
}
