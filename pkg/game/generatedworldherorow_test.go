package game

import (
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// The ordinary hero record uses the chargen row when TypeID also names a generic sibling.
func TestGeneratedMissionHeroDefRowMatchesChargenRow(t *testing.T) {
	f := releaseFront(t)
	party := f.ChargenParty(ui.ChargenResult{Name: "DefRow check", Choices: []int{1, 0, 3}, Stats: []int{31, 27, 24, 29}})
	app := f.App("generated hero def row")
	if err := app.OpenMission(f.MissionOpenerWith(20, party)); err != nil {
		t.Fatal(err)
	}
	var hero sim.Entity
	var found bool
	for _, e := range f.live.world.Entities() {
		if e.Owner == sim.SelfSlot && e.Humanoid {
			hero, found = e, true
			break
		}
	}
	if !found {
		t.Fatal("no hero entity in the generated mission 20 world")
	}
	if hero.SourceBinding.Class != 0 || f.live.mission.state.savedDocument != nil {
		t.Fatal("native mission entry fabricated source provenance")
	}
	var member data.FigureDir
	for _, p := range party {
		if p.StartingHero {
			member = data.FigureDir(p.FigureDir)
		}
	}
	_, wantRow, ok := data.ChargenBase(f.Humans, member.Mage(), member.Female())
	if !ok {
		t.Fatal("chargen base row unavailable for this install")
	}
	genericRow := data.FindHumanByType(f.Humans, hero.TypeID)
	if genericRow == wantRow {
		t.Skip("this install's TypeID search happens to land on the chargen row; no collision to prove against")
	}
	snapshot, label, err := f.Snapshot(true)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := f.ExportCurrentSave(snapshot, label)
	if err != nil {
		t.Fatal(err)
	}
	doc, err := sav.DecodeDocumentData(raw)
	if err != nil {
		t.Fatal(err)
	}
	policy, err := readCurrentActions(&doc)
	if err != nil || policy == nil {
		t.Fatal("missing actor bindings", err)
	}
	var object uint16
	for _, binding := range policy.Bindings {
		if binding.ID == hero.ID && !binding.Structure && !binding.Missing {
			object = binding.Object
		}
	}
	if object == 0 {
		t.Fatal("hero ordinary object is absent")
	}
	row, err := savedStructureValue(&doc.Objects[object-1], "T0C")
	if err != nil || int(row) != wantRow {
		t.Fatalf("hero ordinary TokenRow=%d, want chargen row%d (generic row%d): %v", row, wantRow, genericRow, err)
	}
}
