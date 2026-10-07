package game

import (
	"path/filepath"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/ui"
)

// A town SAV's player-list dword is one past its Player count, as in the
// loaded original town it was imported from.
func TestReleaseCitySAVPlayerListFieldIsOnePastTheCount(t *testing.T) {
	f := releaseFront(t)
	sourcePath, source := groundCorpusFile(t, "2026-08-15/game0010.sav", "89cfca4c14e2b0bafd1fe28911badf246e213d83398874699947008739e8c5d4")
	app := f.App("city player list")
	save, list, load := f.SaveSeams(SaveStore{Dir: t.TempDir()}, OriginalStore{Dir: filepath.Dir(sourcePath)}, nil)
	app.SetSaveSeams(save, list, load)
	if err := headlessOpenLoad(app); err != nil {
		t.Fatal(err)
	}
	label := ""
	for _, e := range list() {
		if e.Name == filepath.Base(sourcePath) {
			label = e.Label
		}
	}
	if err := app.HeadlessActivate(label); err != nil || app.Screen() != ui.ScreenTown {
		t.Fatal("open the original town", err, app.Screen())
	}
	loaded, err := sav.DecodeDocumentData(source)
	if err != nil {
		t.Fatal(err)
	}
	written := saleDocument(t, f, "player list")
	if len(loaded.Players) != 1 || loaded.Head.PlayerListField != 2 {
		t.Fatalf("source town has %d Players and player-list dword %d, want 1 and 2", len(loaded.Players), loaded.Head.PlayerListField)
	}
	if got := written.Head.PlayerListField; got != uint32(len(written.Players)+1) || got != loaded.Head.PlayerListField {
		t.Fatalf("written town has %d Players and player-list dword %d, want one past the count, as loaded (%d)", len(written.Players), got, loaded.Head.PlayerListField)
	}
}
