package game

import (
	"testing"

	"againrom/pkg/data"
)

// requirePartyClassesFromEquipment checks that every party member who is not a
// hire carries the class its visible equipment selects (HERO-APPEAR-042), not
// its hero-band TypeID, and that the class has a swing sound.
func requirePartyClassesFromEquipment(t *testing.T, f *FrontEnd, when string) {
	t.Helper()
	checked := 0
	for i, member := range f.live.mission.party {
		if member.Hired() {
			continue
		}
		id := f.live.mission.ids[i]
		slots, ok := f.live.world.Equipped(id)
		if !ok {
			t.Fatalf("%s: %s has no equipment", when, member.Name)
		}
		_, _, want, matched := data.HeroAppearance(f.Bodies, equipmentFromSlots(slots), member.Mage, false)
		if !matched {
			t.Fatalf("%s: %s's equipment selects no class", when, member.Name)
		}
		for _, e := range f.live.world.Entities() {
			if e.ID != id {
				continue
			}
			if e.Class != want {
				t.Fatalf("%s: %s has class %d (TypeID %d), want %d from its equipment", when, member.Name, e.Class, e.TypeID, want)
			}
			if snd, ok := f.SoundClasses[e.Class]; !ok || len(snd.Slots) == 0 || snd.Slots[0] == 0 {
				t.Fatalf("%s: %s's class %d has no swing sound", when, member.Name, e.Class)
			}
			checked++
		}
	}
	if checked < 4 {
		t.Fatalf("%s: checked %d party members, want the four hero-band members", when, checked)
	}
}

// After LOAD of an original mission SAV, which carries no Againrom leaf, each
// party member's class is the one its equipment selects and has a swing
// sound; our SAVE and LOAD keep it.
func TestReleaseOriginalSAVPartyKeepsItsEquipmentClass(t *testing.T) {
	_, raw := groundCorpusFile(t, "2026-09-09/game0076.sav", "ca6f2980986859fb19c7b602a00b92b0e3ae95b1d00adc6c757152d596ed556c")
	f := releaseFront(t)
	open, town, err := f.RestoreOriginal(raw)
	if err != nil || town {
		t.Fatalf("LOAD: town=%t %v", town, err)
	}
	if err := f.App("party class").OpenMission(open); err != nil {
		t.Fatal(err)
	}
	requirePartyClassesFromEquipment(t, f, "original LOAD")

	store := SaveStore{Dir: t.TempDir()}
	save, _, _ := f.SaveSeams(store, OriginalStore{}, nil)
	name, err := save(true)
	if err != nil || !IsOriginal(name) {
		t.Fatalf("mission SAVE = %q: %v", name, err)
	}
	g := releaseFront(t)
	_, _, load := g.SaveSeams(store, OriginalStore{}, nil)
	open, town, err = load(localOriginalSaveToken(name))
	if err != nil || town {
		t.Fatalf("LOAD of our SAV: town=%t %v", town, err)
	}
	if err := g.App("party class reload").OpenMission(open); err != nil {
		t.Fatal(err)
	}
	requirePartyClassesFromEquipment(t, g, "our SAVE and LOAD")
}
