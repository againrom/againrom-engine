package game

import (
	"crypto/sha256"
	"fmt"
	"os"
	"testing"

	"againrom/pkg/sim"
)

func TestReleaseLegacyBatPaletteAfterLoad(t *testing.T) {
	path := os.Getenv("AGAINROM_LEGACY_BAT_SAV")
	if path == "" {
		t.Skip("no AGAINROM_LEGACY_BAT_SAV: exact legacy mission SAV is required")
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := fmt.Sprintf("%X", sha256.Sum256(raw)); got != "AEEBFDEA1183B11A3F38C6CFA487092FF9038F5C8C8623B9F0E9ADA4E85B80A2" {
		t.Fatalf("legacy bat SAV hash=%s", got)
	}
	f := loadRoodMission(t, raw)
	byID := make(map[sim.EntityID]sim.Entity)
	for _, e := range f.live.world.Entities() {
		byID[e.ID] = e
	}
	state := f.live.mission.state.savedDocument
	if state == nil || state.Document == nil {
		t.Fatal("legacy SAV lost its exact actor document")
	}
	checked := 0
	for _, binding := range state.Actors {
		if binding.Retired || binding.ObjectIndex == 0 || int(binding.ObjectIndex) > len(state.Document.Objects) {
			continue
		}
		e, live := byID[binding.EntityID]
		record := &state.Document.Objects[binding.ObjectIndex-1]
		if !live || record.Class != "Unit" {
			continue
		}
		class := f.live.units.Classes[e.Class]
		if class == nil || len(class.Tiers) == 0 {
			continue
		}
		face, err := savedStructureValue(record, "U4B")
		if err != nil || face == 0 {
			t.Fatalf("tiered actor %d has no saved face: %v", e.ID, err)
		}
		checked++
		if got := f.live.tiers[e.ID]; got != int(face) {
			t.Fatalf("tiered actor %d mapID %d draws tier %d, saved face %d", e.ID, e.MapUnitID, got, face)
		}
	}
	if checked < 100 {
		t.Fatalf("only %d tiered saved actors checked", checked)
	}
	for _, mapID := range []uint16{1098, 1099} {
		var id sim.EntityID
		found := false
		for _, e := range f.live.world.Entities() {
			if e.MapUnitID == mapID {
				if found || e.TypeID != 70 {
					t.Fatalf("bat MapUnitID %d has ambiguous or wrong actors", mapID)
				}
				id, found = e.ID, true
			}
		}
		if !found || f.live.tiers[id] != 4 {
			t.Fatalf("bat MapUnitID %d runtime ID %d tier=%d, want 4", mapID, id, f.live.tiers[id])
		}
		class := f.live.units.Classes[70]
		if class == nil || len(class.Tiers) < 4 || len(class.Tiers[3]) == 0 {
			t.Fatal("bat tier-4 art unavailable")
		}
		var drawn bool
		for _, d := range f.live.entityDraws() {
			if d.ID != uint32(id) {
				continue
			}
			drawn = true
			if d.Frame == nil || d.Frame.Palette != class.Tiers[3][0].Palette || d.Frame.Palette == class.Frames[0].Palette {
				t.Fatalf("bat MapUnitID %d drew the base palette instead of tier 4", mapID)
			}
		}
		if !drawn {
			t.Fatalf("bat MapUnitID %d is not drawn", mapID)
		}
	}
}
