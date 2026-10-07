package game

import (
	"os"
	"path/filepath"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
)

func townPlacementRoundTrip(t *testing.T, f *FrontEnd) ([]byte, *FrontEnd) {
	t.Helper()
	dir := t.TempDir()
	save, _, _ := f.SaveSeams(SaveStore{Dir: dir}, OriginalStore{}, nil)
	name, err := save(false)
	if err != nil {
		t.Fatal("ordinary town SAVE", err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil || len(entries) != 1 || entries[0].Name() != name || filepath.Ext(name) != ".sav" {
		t.Fatal("ordinary town SAVE wrote", entries, err)
	}
	raw, err := os.ReadFile(filepath.Join(dir, name))
	if err != nil {
		t.Fatal(err)
	}
	loaded := releaseFront(t)
	openLocalTownSAV(t, loaded, dir, name)
	return raw, loaded
}

func townPlacementCharacter(t *testing.T, raw []byte, name string) sav.Character {
	t.Helper()
	file, err := sav.Open(raw)
	if err != nil {
		t.Fatal(err)
	}
	party, err := file.Party()
	if err != nil {
		t.Fatal(err)
	}
	return rawCharacterNamed(t, party, name)
}

func TestReleaseTownSAVEWritesCurrentPlacementCell(t *testing.T) {
	_, source := groundCorpusFile(t, "2026-08-15/game0010.sav", "89cfca4c14e2b0bafd1fe28911badf246e213d83398874699947008739e8c5d4")
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "source.sav"), source, 0o600); err != nil {
		t.Fatal(err)
	}
	f := releaseFront(t)
	openLocalTownSAV(t, f, dir, "source.sav")
	hero := f.Carried[0]
	var baseline *mapload.Saved
	for _, b := range f.originalCity.bindings {
		if b.partyID == hero.ID {
			baseline = b.baseline.Saved
		}
	}
	if baseline == nil {
		t.Fatal("hero has no bound source placement")
	}
	document := townPlacementCharacter(t, source, hero.Name)
	if hero.Saved == nil || hero.Saved.Cell != (mapload.Cell{}) {
		t.Fatalf("town LOAD did not clear the previous mission cell: %+v", hero.Saved)
	}
	base, unmutatedLoad := townPlacementRoundTrip(t, f)
	unmutated := townPlacementCharacter(t, base, hero.Name)
	if unmutated.Cell != 0 {
		t.Fatalf("unmutated SAVE cell %d,%d, current 0,0", unmutated.Col(), unmutated.Row())
	}
	if got := trainingPartyMember(t, unmutatedLoad, hero.ID).Saved; got == nil || got.Cell != (mapload.Cell{}) {
		t.Fatalf("unmutated cold LOAD placement %+v, want (0,0)", got)
	}
	want := mapload.Cell{X: int32(document.Col()) + 4, Y: int32(document.Row()) + 8}
	saved := *baseline
	saved.Cell = want
	f.Carried[0].Saved = &saved
	raw, loaded := townPlacementRoundTrip(t, f)
	written := townPlacementCharacter(t, raw, hero.Name)
	if int32(written.Col()) != want.X || int32(written.Row()) != want.Y {
		t.Fatalf("SAV cell %d,%d, current %+v, document %d,%d", written.Col(), written.Row(), want, document.Col(), document.Row())
	}
	if written.FineX != document.FineX || written.FineY != document.FineY {
		t.Fatal("fine position left the document's bytes", written.FineX, written.FineY)
	}
	if got := trainingPartyMember(t, loaded, hero.ID).Saved; got == nil || got.Cell != want || got.HP != saved.HP || got.MaxHP != saved.MaxHP {
		t.Fatalf("loaded placement %+v, current %+v", got, saved)
	}
	t.Logf("%s: document cell %d,%d; cleared current cell 0,0 written and loaded; moved current cell %d,%d written and loaded", hero.Name, document.Col(), document.Row(), want.X, want.Y)
}
