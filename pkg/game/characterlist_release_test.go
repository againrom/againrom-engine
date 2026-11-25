package game

import (
	"testing"

	"againrom/pkg/ui"
)

// On the installed game the Multiplayer button opens the character list page
// with the New row alone when no store is set, its frame composes, Escape returns to the menu and Load then
// opens the load page.
func TestReleaseMultiplayerOpensTheCharacterList(t *testing.T) {
	f := releaseFront(t)
	app := f.App("character list")
	app.Layout(640, 480)
	f.ConfigureSaveSeams(app, SaveStore{Dir: t.TempDir()}, OriginalStore{}, nil)
	if err := app.HeadlessActivate("multi player"); err != nil {
		t.Fatal(err)
	}
	if !app.CharacterList() || app.Screen() != ui.ScreenLoad || len(app.HeadlessRows()) != 1 {
		t.Fatalf("Multiplayer opens %v (character list %v) with %d rows", app.Screen(), app.CharacterList(), len(app.HeadlessRows()))
	}
	if pix, _, err := app.HeadlessPicture(ui.HeadlessRequest{Part: ui.HeadlessPartFrame}); err != nil || pix == nil {
		t.Fatalf("character list frame: %v", err)
	}
	if err := app.HeadlessKey("escape"); err != nil {
		t.Fatal(err)
	}
	if app.Screen() != ui.ScreenMenu || app.CharacterList() {
		t.Fatalf("Escape left the character list for %v", app.Screen())
	}
	if err := app.HeadlessKey("load"); err != nil {
		t.Fatal(err)
	}
	if app.Screen() == ui.ScreenLoad && app.CharacterList() {
		t.Fatal("Load opened the character list")
	}
}
