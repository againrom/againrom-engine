package game

import (
	"reflect"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
)

func setCurrentHumanSight(t *testing.T, f *FrontEnd, want uint16) uint16 {
	t.Helper()
	for i := range f.Carried {
		member := &f.Carried[i]
		if member.ID != "hero" {
			continue
		}
		retained, ok := member.OriginalHumanState()
		if !ok || member.Carry == nil || member.Carry.LiveLoad == nil || member.Carry.LiveLoad.Inventory.Source.Class != 2 {
			t.Fatal("loaded town hero has no retained and current Human")
		}
		if member.Carry.LiveLoad.Inventory.Source.Sight != retained.Sight || want == retained.Sight {
			t.Fatal("Human sight control does not distinguish loaded and current values", retained.Sight, member.Carry.LiveLoad.Inventory.Source.Sight, want)
		}
		member.Carry.LiveLoad.Inventory.Source.Sight = want
		stale, ok := member.OriginalHumanState()
		if !ok || stale.Sight != retained.Sight {
			t.Fatal("current sight edit changed the retained Human or invalidated its guard", ok, stale.Sight, retained.Sight)
		}
		current, _, ok := currentCityHuman(*member)
		if !ok || current.Sight != want {
			t.Fatal("current Human does not hold the sight edit", ok, current.Sight, want)
		}
		return retained.Sight
	}
	t.Fatal("loaded town has no hero")
	return 0
}

func assertSavedHumanSight(t *testing.T, raw []byte, want uint16) {
	t.Helper()
	file, err := sav.Open(raw)
	if err != nil {
		t.Fatal(err)
	}
	city, err := file.CityProvenance()
	if err != nil {
		t.Fatal(err)
	}
	heroes := 0
	for _, member := range city.Roster() {
		if !member.Hero {
			continue
		}
		heroes++
		human, err := city.Human(member.Identity)
		if err != nil {
			t.Fatal(err)
		}
		if human.Fields.Sight != want {
			t.Fatal("town SAVE wrote the loaded Human sight", human.Fields.Sight, want)
		}
	}
	if heroes != 1 {
		t.Fatal("saved town has no unique hero", heroes)
	}
}

func assertReloadedHumanSight(t *testing.T, f *FrontEnd, want uint16) {
	t.Helper()
	member := trainingPartyMember(t, f, "hero")
	current, _, ok := currentCityHuman(member)
	if !ok || current.Sight != want {
		t.Fatal("town LOAD lost current Human sight", ok, current.Sight, want)
	}
	if member.OriginalHuman == nil || member.OriginalHuman.State.Sight != want {
		t.Fatal("town LOAD did not bind the written Human sight", member.OriginalHuman)
	}
}

func TestTownSAVWritesCurrentHumanSight(t *testing.T) {
	for _, want := range []uint16{1844, 0} {
		f := currentTrainingCity(t)
		if old := setCurrentHumanSight(t, f, want); old != 1587 {
			t.Fatal("fixture loaded Human sight changed", old)
		}
		before := mapload.CloneParty(f.Carried)
		raw := currentTownSave(t, f)
		if !reflect.DeepEqual(before, f.Carried) {
			t.Fatal("town SAVE changed the current party")
		}
		assertSavedHumanSight(t, raw, want)
		fresh := &FrontEnd{InstallResources: f.InstallResources}
		if _, town, err := fresh.RestoreOriginal(raw); err != nil || !town {
			t.Fatal("town LOAD", town, err)
		}
		assertReloadedHumanSight(t, fresh, want)
	}
}
