package game

import (
	"bytes"
	"testing"

	"againrom/pkg/ui"
)

// R2-ENGINE-343: TALK counts without the section.
func TestReleaseSecondTownLoadWithoutInnSection(t *testing.T) {
	secondGameRoot(t)
	dir := secondMissionSaveDirectory(t)
	f, app := secondTownNew(t, true, false)
	raw := secondTownNamedSave(t, f, app, dir, "First town")
	secondTownShape(t, raw)

	withSection, a := secondTownCold(t, dir, "First town.sav")
	if err := a.HeadlessActivate("TALK 517"); err != nil {
		t.Fatal(err)
	}
	if _, open := withSection.TownScreen().(ui.TownDialogueScreen).TownDialogue(); !open {
		t.Fatal("control: the installed section shows no conversation")
	}
	control := withSection.Town.second.bank

	without := func(f *FrontEnd) {
		fs := f.Archives.Containers.Fork()
		fs.Overlay(mainPrefix+"text/town.txt", func() ([]byte, error) {
			return []byte("#other\r\nunrelated\r\n"), nil
		})
		own := *f.Archives
		own.Containers = fs
		f.Archives = &own
	}
	cold, b := secondTownCold(t, dir, "First town.sav", without)
	if got, _ := cold.Archives.Containers.ReadFile(mainPrefix + "text/town.txt"); bytes.Contains(got, []byte("npc517talk10")) {
		t.Fatal("overlay left the section in place")
	}
	before := cold.Town.second.bank
	if before == control {
		t.Fatal("TALK has not yet changed the bank, the control proves nothing")
	}
	if err := b.HeadlessActivate("TALK 517"); err != nil {
		t.Fatal(err)
	}
	if _, open := cold.TownScreen().(ui.TownDialogueScreen).TownDialogue(); open {
		t.Fatal("a missing section opened a conversation")
	}
	if cold.Town.second.bank != control {
		t.Fatal("TALK without the section did not record the talk as the control did")
	}
}
