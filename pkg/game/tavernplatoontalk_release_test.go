package game

import (
	"slices"
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/ui"
)

// TestReleasePlatoonTalkCellShowsItsUnit opens the chapter-110 tavern and
// selects the talk cell of npc2, the Ballista's mercenary section. The left
// panel states the Ballista row's statistics and draws a figure below them,
// as the owner saw in the original (DIV-2778, TAVERN-TALKSTATS-017).
func TestReleasePlatoonTalkCellShowsItsUnit(t *testing.T) {
	f := releaseFront(t)
	const chapter, npc = 110, 2
	town := NewTown(f.Campaign.Value())
	for _, prior := range f.Campaign.Value().Main {
		if prior >= chapter {
			break
		}
		town.Won(prior)
	}
	if town.Chapter() != chapter {
		t.Fatalf("the town reached chapter %d, want %d", town.Chapter(), chapter)
	}
	f.Town = town
	f.Carried = f.ChargenParty(ui.ChargenResult{Name: "Platoon talk", Choices: []int{1, 1, 3}, Stats: []int{31, 27, 24, 29}})
	f.arriveInTown()
	if !slices.Contains(f.Campaign.Value().Chapters[chapter].InnNPC, npc) {
		t.Fatalf("chapter %d lists no talk cell for npc%d", chapter, npc)
	}
	units := f.Table.Units
	var want data.UnitDef
	found := false
	for i := 1; i < units.Len(); i++ {
		if units.EntryName(i) == "Ballista" {
			def, err := data.NewUnitDef("Ballista", units.EntryParams(i))
			if err != nil {
				t.Fatal(err)
			}
			want, found = def, true
		}
	}
	if !found {
		t.Fatal("no Ballista row in the Units table")
	}
	s := f.TownScreen().(*townScreen)
	s.room = roomTavern
	cell := -1
	for i, c := range s.tavernSurfaceCells() {
		if c.Semantic == "NPC 2" {
			cell = i
		}
	}
	if cell < 0 {
		t.Fatal("the tavern lists no npc2 cell")
	}
	s.TownSurfaceClick(ui.TownSurfaceControl{Kind: ui.TownSurfaceControlCell, Index: cell}, false)
	got := s.TownSurface().Candidate
	if !got.HasSubject {
		t.Fatal("npc2's cell shows no statistics")
	}
	sub := got.Subject
	if sub.MaxHP != int(want.HealthMax) || sub.Char.Body != int(want.Body) || sub.Char.Reaction != int(want.Reaction) ||
		sub.Combat.Defence != int(want.Defence) || sub.Combat.Absorption != int(want.Absorption) || sub.Combat.ToHit != int(want.ToHit) {
		t.Fatalf("npc2's panel = health %d body %d reaction %d defence %d absorption %d to-hit %d, want the Ballista row %+v",
			sub.MaxHP, sub.Char.Body, sub.Char.Reaction, sub.Combat.Defence, sub.Combat.Absorption, sub.Combat.ToHit, want)
	}
	if got.Figure == nil {
		t.Fatal("npc2's panel draws no figure")
	}
	t.Logf("chapter %d npc%d: %q health %d/%d defence %d absorption %d", chapter, npc, sub.Name, sub.HP, sub.MaxHP,
		sub.Combat.Defence, sub.Combat.Absorption)
}
