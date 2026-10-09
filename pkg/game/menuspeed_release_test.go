package game

import (
	"strings"
	"testing"

	"againrom/pkg/render/terrain"
	"againrom/pkg/ui"
)

// Called by the registered EN/RU pause-menu witness. The pointer uses the
// literal panel coordinates, independently of its hit-test helper.
func releaseMenuSpeedWitness(t *testing.T, f *FrontEnd, app *ui.App) {
	t.Helper()
	app.Layout(640, 480)
	before := f.live.world.Hash()
	rows := app.HeadlessRows()
	if len(rows) < 4 || !strings.HasSuffix(rows[0].Text, ": 5") {
		t.Fatal("options do not show the default speed level", rows)
	}
	// Positive level 6 is the default rung plus one after the pause position.
	for _, edge := range []string{"press", "release"} {
		if err := app.HeadlessPointer(edge, 227, 116); err != nil {
			t.Fatal(err)
		}
	}
	want := terrain.DefaultCadenceRung + 1
	if got, err := (OptionsStore{Path: f.Options.Path}).GameSpeed(); err == nil && got == want || app.MapCadencePreference() == want {
		t.Fatal("the slider reached the stored speed before OK", got, err)
	}
	if app.HeadlessRows()[0].Text == rows[0].Text {
		t.Fatal("the slider did not redraw its level")
	}
	if err := app.HeadlessGameMenuAction("page-return"); err != nil {
		t.Fatal(err)
	}
	if got, err := (OptionsStore{Path: f.Options.Path}).GameSpeed(); err != nil || got != want || app.MapCadencePreference() != want {
		t.Fatal("OK did not persist the speed through a fresh store", got, err)
	}
	if f.live.world.Hash() != before {
		t.Fatal("speed advanced the paused world")
	}
	// Reopening the ordinary frontend applies the same value on its next map.
	fresh := f.App("stored menu speed")
	if fresh.MapCadencePreference() != want {
		t.Fatal("new App lost menu speed")
	}
	for _, action := range []string{"game-options", "speed-down", "page-return", "game-options"} {
		if err := app.HeadlessGameMenuAction(action); err != nil {
			t.Fatal(err)
		}
	}
	if got, _ := f.Options.GameSpeed(); got != terrain.DefaultCadenceRung || f.live.world.Hash() != before {
		t.Fatal("slower failed or advanced the world", got)
	}
}
