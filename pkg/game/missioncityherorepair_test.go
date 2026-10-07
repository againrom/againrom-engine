package game

import (
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/formats/sav"
	"againrom/pkg/ui"
)

// repairGeneratedHeroDefRow is the export-time counterpart to the
// mapload.ConstructActorBasis fix: that fix keeps a FRESH session's own
// generated hero object correct from construction on, but a document already
// frozen by an earlier session (an imported save, or any other route this
// package does not control) can still carry the wrong row for the hero's own
// object. A deliberately wrong T0C proves this repair overrides whatever was
// frozen, rather than only leaving an already-correct value untouched.
func TestRepairGeneratedHeroDefRowCorrectsAFrozenWrongRow(t *testing.T) {
	f := releaseFront(t)
	party := f.ChargenParty(ui.ChargenResult{Name: "DefRow repair", Choices: []int{1, 0, 3}, Stats: []int{31, 27, 24, 29}})
	app := f.App("generated hero def row repair")
	if err := app.OpenMission(f.MissionOpenerWith(20, party)); err != nil {
		t.Fatalf("open mission 20: %v", err)
	}
	heroIndex := -1
	for i, p := range party {
		if p.StartingHero {
			heroIndex = i
		}
	}
	if heroIndex < 0 {
		t.Fatal("no starting hero in this fixture's party")
	}
	member := party[heroIndex]
	id := f.live.mission.ids[heroIndex]
	fig := data.FigureDir(member.FigureDir)
	_, wantRow, ok := data.ChargenBase(f.Humans, fig.Mage(), fig.Female())
	if !ok {
		t.Fatal("chargen base row unavailable for this install")
	}
	wrongRow := wantRow + 1
	if wrongRow > 255 {
		wrongRow = wantRow - 1
	}
	doc := &sav.DocumentData{Objects: []sav.DocumentRecordData{
		{Class: "Human", Values: []sav.DocumentValueData{{Name: "T0C", Value: uint32(wrongRow)}}},
	}}
	repairGeneratedHeroDefRow(doc, 1, member, f.live.world, id, f.Table)
	if got := doc.Objects[0].Values[0].Value; got != uint32(wantRow) {
		t.Fatalf("T0C after repair = %d, want chargen row %d (deliberately frozen wrong value was %d)", got, wantRow, wrongRow)
	}
}
