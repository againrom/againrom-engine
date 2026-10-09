package game

import (
	"crypto/sha256"
	"fmt"
	"os"
	"slices"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

func TestReleaseChargenHeroTypeHasBothAxes(t *testing.T) {
	for _, tc := range []struct {
		name         string
		mage, female bool
		want         int32
	}{
		{"man fighter", false, false, 0x21},
		{"woman fighter", false, true, 0x22},
		{"man mage", true, false, 0x23},
		{"woman mage", true, true, 0x24},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := releaseFront(t)
			mage, female := 0, 0
			if tc.mage {
				mage = 1
			}
			if tc.female {
				female = 1
			}
			party := f.ChargenParty(ui.ChargenResult{Name: "Hero", Choices: []int{female, mage, 0}, Stats: []int{25, 25, 25, 25}})
			if err := f.App("hero type").OpenMission(f.MissionOpenerWith(10, party)); err != nil {
				t.Fatal(err)
			}
			id := f.live.mission.state.Start.IDs[0]
			e := releaseEntity(t, &mapWorld{world: f.live.mission.state.World}, id)
			if e.TypeID != tc.want {
				t.Fatalf("hero type=%#x, want %#x", e.TypeID, tc.want)
			}
		})
	}
}

func TestReleaseRoodHeroTypeFreshAndContaminated(t *testing.T) {
	path := os.Getenv("AGAINROM_ROOD_SAV")
	if path == "" {
		t.Skip("no AGAINROM_ROOD_SAV")
	}
	source, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	t.Run("fresh", func(t *testing.T) {
		f := releaseFront(t)
		if err := f.App("Rood fresh type").OpenMission(f.MissionOpener(140)); err != nil {
			t.Fatal(err)
		}
		var rood sim.Entity
		for _, e := range f.live.mission.state.World.Entities() {
			if e.MapUnitID == 443 {
				rood = e
			}
		}
		if rood.MapUnitID != 443 {
			t.Fatal("fresh Rood map actor absent")
		}
		if rood.TypeID != 0x23 {
			t.Fatalf("fresh Rood type=%#x, want male mage 0x23", rood.TypeID)
		}
		first := saveRoodMission(t, f)
		written, err := sav.DecodeDocumentData(first)
		if err != nil {
			t.Fatal(err)
		}
		var found bool
		for i, r := range written.Objects {
			if r.Class != "Human" {
				continue
			}
			unit, _ := savedStructureValue(&r, "T08")
			if unit != 443 {
				continue
			}
			found = true
			typeID, _ := savedStructureValue(&r, "T0E")
			flags, _ := savedStructureValue(&r, "U4C")
			health, _ := savedStructureValue(&r, "Health")
			stage, _ := savedStructureValue(&r, "Stage")
			if typeID != 0x23 || flags&4 == 0 || health != 0 || stage != 1 || slices.Contains(written.DeadActors, uint16(i+1)) {
				t.Fatalf("fresh Rood SAV type=%#x flags=%#x HP=%d stage=%d dead-root=%v", typeID, flags, health, stage, slices.Contains(written.DeadActors, uint16(i+1)))
			}
		}
		if !found {
			t.Fatal("fresh Rood SAV record absent")
		}
		cold := loadRoodMission(t, first)
		for i := 0; i < 256; i++ {
			cold.live.tick()
		}
		coldFound := false
		for _, e := range cold.live.mission.state.World.Entities() {
			if e.MapUnitID == 443 {
				coldFound = true
				if e.TypeID != 0x23 || e.HP != 0 || e.Decay != sim.DecayFallen {
					t.Fatalf("cold Rood type=%#x HP=%d decay=%d", e.TypeID, e.HP, e.Decay)
				}
			}
		}
		if !coldFound {
			t.Fatal("cold Rood map actor absent")
		}
	})
	t.Run("contaminated", func(t *testing.T) {
		input := alterSAV(t, source, func(doc *sav.DocumentData) bool {
			object, _ := roodDocumentActor(t, *doc)
			savedObjectSetValue(&doc.Objects[object-1], "T0E", 0x21)
			doc.DeadActors = append(doc.DeadActors, object)
			return true
		})
		f := loadRoodMission(t, input)
		rood := worldEntityByRuntimeID(t, f, 258)
		if rood.TypeID != 0x23 || rood.SourceBinding.TypeID != 0x23 || rood.ActorLoad.Source.TypeID != 0x23 {
			t.Fatalf("loaded Rood type entity=%#x binding=%#x basis=%#x, want 0x23", rood.TypeID, rood.SourceBinding.TypeID, rood.ActorLoad.Source.TypeID)
		}
		first := saveRoodMission(t, f)
		written, err := sav.DecodeDocumentData(first)
		if err != nil {
			t.Fatal(err)
		}
		requireRoodPlacement(t, written, 0, 1, false)
		object, _ := roodDocumentActor(t, written)
		if typeID, _ := savedStructureValue(&written.Objects[object-1], "T0E"); typeID != 0x23 {
			t.Fatalf("re-saved Rood type=%#x, want 0x23", typeID)
		}
		cold := loadRoodMission(t, first)
		for i := 0; i < 256; i++ {
			cold.live.tick()
		}
		rood = worldEntityByRuntimeID(t, cold, 258)
		if rood.HP != 0 || rood.Decay != sim.DecayFallen || rood.TypeID != 0x23 {
			t.Fatal("cold Rood state", rood.HP, rood.Decay, rood.TypeID)
		}
	})
	t.Run("owner_engine_save", func(t *testing.T) {
		badPath := os.Getenv("AGAINROM_ROOD_BAD_SAV")
		if badPath == "" {
			t.Skip("no AGAINROM_ROOD_BAD_SAV")
		}
		bad, err := os.ReadFile(badPath)
		if err != nil {
			t.Fatal(err)
		}
		if got := fmt.Sprintf("%X", sha256.Sum256(bad)); got != "5B3EB30C1ED3F32BE5AB847ECC95E5A04040D6BEFDA85AC40FC40A1E985E1058" {
			t.Fatalf("owner engine SAV hash=%s", got)
		}
		f := loadRoodMission(t, bad)
		first := saveRoodMission(t, f)
		doc, err := sav.DecodeDocumentData(first)
		if err != nil {
			t.Fatal(err)
		}
		var object uint16
		for i := range doc.Objects {
			r := &doc.Objects[i]
			if r.Class != "Human" {
				continue
			}
			// The party Rood's RuntimeID is the one the writer gives him
			// (DIV-2504); his map placement names him.
			unit, _ := savedStructureValue(r, "T08")
			runtime, _ := savedStructureValue(r, "RuntimeID")
			if unit != 443 || runtime == 0 {
				continue
			}
			object = uint16(i + 1)
			for name, want := range map[string]uint32{"T0E": 0x23, "U4C": 6, "Health": 0, "Stage": 1} {
				got, err := savedStructureValue(r, name)
				if err != nil || got != want {
					t.Fatalf("re-saved owner Rood %s=%#x err=%v, want %#x", name, got, err, want)
				}
			}
		}
		if object == 0 || slices.Contains(doc.DeadActors, object) {
			t.Fatalf("re-saved owner Rood object=%d dead-root=%v", object, slices.Contains(doc.DeadActors, object))
		}
		cold := loadRoodMission(t, first)
		for i := 0; i < 256; i++ {
			cold.live.tick()
		}
		var found bool
		for _, rood := range cold.live.mission.state.World.Entities() {
			if rood.MapUnitID != 443 {
				continue
			}
			found = true
			if rood.TypeID != 0x23 || rood.HP != 0 || rood.Decay != sim.DecayFallen {
				t.Fatal("owner cold Rood state", rood.TypeID, rood.HP, rood.Decay)
			}
		}
		if !found {
			t.Fatal("owner cold Rood map actor absent")
		}
	})
}
