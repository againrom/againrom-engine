package game

import (
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

func originalFacingMission(t *testing.T) *Mission {
	t.Helper()
	w, err := sim.NewWorld(1, sim.Bounds{Width: 8, Height: 8}, sim.ModeCanonical, nil, []sim.Entity{
		{ID: 1, X: 1, Y: 1, HP: 10, MaxHP: 10, MapUnitID: 71},
		{ID: 2, X: 2, Y: 1, HP: 10, MaxHP: 10},
		{ID: 3, X: 3, Y: 1, HP: 10, MaxHP: 10},
	})
	if err != nil {
		t.Fatal(err)
	}
	return &Mission{World: w, Start: mapload.Start{IDs: []sim.EntityID{2, 3}}}
}

func TestOriginalFacingPartyUsesSourceOffsetsNotMapIDOrFileOrder(t *testing.T) {
	ms := originalFacingMission(t)
	source := []sav.Actor{
		{Off: 100, RuntimeID: 1, HP: 10, Facing: 171},
		{Off: 200, RuntimeID: 2, HP: 10, Facing: 64},
		{Off: 300, RuntimeID: 3, HP: 10, Facing: 255, MapUnitID: 71},
		{Off: 400, RuntimeID: 4, HP: 10, Facing: 128, MapUnitID: 99},
		{Off: 500, HP: 0, Facing: 160, MapUnitID: 71},
	}
	if err := applyOriginalFacings(ms, source, []int{200, 100}); err != nil {
		t.Fatal(err)
	}
	for i, e := range ms.World.Entities() {
		if want := []uint8{255, 64, 171}[i]; e.Facing != want || e.DesiredFacing != want {
			t.Fatalf("entity%d facing %d/%d want%d", e.ID, e.Facing, e.DesiredFacing, want)
		}
	}
}

func TestOriginalFacingBadLateBindingLeavesWorldUnchanged(t *testing.T) {
	for _, offsets := range [][]int{{100}, {100, 999}, {100, 100}} {
		ms := originalFacingMission(t)
		before := ms.World.Hash()
		if err := applyOriginalFacings(ms, []sav.Actor{{Off: 100, RuntimeID: 1, HP: 10, Facing: 160}}, offsets); err == nil || before != ms.World.Hash() {
			t.Fatalf("invalid offsets %v: %v", offsets, err)
		}
	}
	for _, duplicateTarget := range []bool{false, true} {
		ms := originalFacingMission(t)
		ms.Start.IDs = nil
		source := []sav.Actor{{Off: 100, RuntimeID: 1, HP: 10, Facing: 160, MapUnitID: 71}}
		if duplicateTarget {
			entities := ms.World.Entities()
			entities[1].MapUnitID = 71
			var err error
			ms.World, err = sim.NewWorld(1, ms.World.Bounds(), sim.ModeCanonical, nil, entities)
			if err != nil {
				t.Fatal(err)
			}
		} else {
			source = append(source, sav.Actor{Off: 200, RuntimeID: 2, HP: 10, MapUnitID: 71})
		}
		before := ms.World.Hash()
		if err := applyOriginalFacings(ms, source, nil); err == nil || before != ms.World.Hash() {
			t.Fatalf("ambiguous join changed world: %v", err)
		}
	}
}
