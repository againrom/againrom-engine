package game

import (
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// The chapter grant has no live mission actor. Its constructed source Book
// must bind to the current member and survive ordinary town export.
func TestReleaseOriginalSaveWritesAGraftedCompanionsSourceSpellbook(t *testing.T) {
	f := releaseFront(t)
	party := f.ChargenParty(ui.ChargenResult{Name: "Graft book", Choices: []int{1, 0, 0}, Stats: []int{31, 27, 24, 29}})
	app := f.App("graft book")
	if err := app.OpenMission(f.MissionOpenerWith(20, party)); err != nil {
		t.Fatalf("open mission 20: %v", err)
	}
	// LiveCompleteCampaign is the production headless-finish path
	// (FinishMissionWithRoster): completing mission 20 is the chapter
	// transition addChapterCompanions grants npc:22 on, on every campaign
	// that reaches it, not only an older save lineage. It also opens f.Town,
	// which Snapshot below requires.
	world := f.live.world
	if _, _, err := f.LiveCompleteCampaign(); err != nil {
		t.Fatalf("LiveCompleteCampaign: %v", err)
	}
	// Weapon.Name is "the literal this weapon was resolved from" (data.Weapon's
	// own doc comment), not a canonical label: chargen's own starting-weapon
	// literal and the baseline's table-driven reverse composition from the
	// identical Code are free to differ in spelling. A live session that saved
	// and reloaded even once before this point would already carry the
	// composed spelling; this fixture never does, so it is aligned here by
	// hand, exactly as
	// TestReleaseMissionCityReturnSkipsCheckReturnForAGraftedMemberWithNoLiveCarry
	// does, to keep this pre-existing quirk from obscuring the book fix this
	// test isolates.
	for i := range f.Carried {
		if !f.Carried[i].StartingHero || f.Carried[i].Weapon == nil {
			continue
		}
		shapes, materials, weapons := f.tableWeapons()
		if canonical, err := data.WeaponFromCode(f.Carried[i].Weapon.Code, shapes, materials, weapons); err == nil {
			f.Carried[i].Weapon.Name = canonical.Name
		}
	}
	// Bind the completed grant through the production return seam.
	if err := f.retainMissionCity(f.townInstall(), 20, world); err != nil {
		t.Fatalf("retainMissionCity: %v", err)
	}
	var companionKnownSpells uint32
	var companionFound bool
	for _, p := range f.Carried {
		if p.CompanionNPC != 22 {
			continue
		}
		companionFound = true
		companionKnownSpells = p.KnownSpells
		if p.Carry == nil || p.Book.State == sim.BookLegacy || !p.SpellbookRestored {
			t.Fatal("the current grant did not bind its constructed source book")
		}
		if p.KnownSpells == 0 {
			t.Fatal("companion 22 must know at least one spell to reproduce the legacy-book refusal")
		}
	}
	if !companionFound {
		t.Fatal("mission 20 completion did not graft companion 22 into f.Carried")
	}
	snap, label, err := f.Snapshot(false)
	if err != nil {
		t.Fatalf("Snapshot: %v", err)
	}
	out, err := f.ExportOriginalSave(snap, label)
	if err != nil {
		t.Fatalf("ExportOriginalSave: %v, want a written original-compatible town save", err)
	}
	if len(out) == 0 {
		t.Fatal("ExportOriginalSave returned no bytes")
	}
	written, err := sav.Open(out)
	if err != nil {
		t.Fatalf("sav.Open(exported): %v", err)
	}
	provenance, err := written.CityProvenance()
	if err != nil {
		t.Fatalf("CityProvenance(exported): %v", err)
	}
	var found bool
	for _, character := range provenance.Roster() {
		if character.KnownSpells != companionKnownSpells {
			continue
		}
		found = true
		if !character.HasSpellbook {
			t.Fatal("exported companion lost HasSpellbook")
		}
		if len(character.Spells) == 0 {
			t.Fatal("exported companion lost her spell instances")
		}
	}
	if !found {
		t.Fatal("exported roster has no character with the companion's KnownSpells")
	}
}
