//go:build release

package game

import (
	"strings"
	"testing"

	"againrom/internal/firstgame"
	"againrom/pkg/base"
	"againrom/pkg/ui"
)

// TestReleaseFirstGameMultiplayerMapChoiceCountsInstalledMaps checks the first
// game's multiplayer map choice page against an installed root (SESS-100:
// type-0 +0x70 at least 2; scanned directory and shipped count Medium: EN 10,
// RU 5 of its 6 root maps).
func TestReleaseFirstGameMultiplayerMapChoiceCountsInstalledMaps(t *testing.T) {
	firstgame.SkipOnSecondGame(t)
	f := releaseFront(t)
	app := f.App("first map choice")
	app.Layout(640, 480)
	failed := f.OpenMultiplayerMapChoice(app, ui.ScreenMenu, nil)
	if failed != 0 {
		t.Fatalf("failed to decode %d installed map(s)", failed)
	}
	rows := app.HeadlessRows()
	want, ok := map[string]int{base.ROM1EN: 10, base.ROM1RU: 5}[f.Archives.Base.Profile.ID]
	if !ok {
		t.Fatalf("unexpected first-game profile %s", f.Archives.Base.Profile.ID)
	}
	if len(rows) != want {
		t.Fatalf("%s: listed %d multiplayer maps, want %d", f.Archives.Base.Profile.ID, len(rows), want)
	}
	for _, r := range rows {
		if r.Word70 < 2 {
			t.Errorf("row %q has Word70=%d, want >= 2", r.Text, r.Word70)
		}
	}
	if pix, _, err := app.HeadlessPicture(ui.HeadlessRequest{Part: ui.HeadlessPartFrame}); err != nil || pix == nil {
		t.Fatalf("map choice frame: %v", err)
	}
	if err := app.HeadlessKey("escape"); err != nil {
		t.Fatal(err)
	}
	if app.Screen() != ui.ScreenMenu {
		t.Fatalf("Escape left map choice for %v", app.Screen())
	}
}

// TestReleaseSecondGameMultiplayerMapChoiceCountsInstalledMaps checks the
// second game's multiplayer map choice page against an installed root
// (R2-ASSET-403: EN 13, RU 37; R2-ENGINE-1842: level from 0x88 per row).
func TestReleaseSecondGameMultiplayerMapChoiceCountsInstalledMaps(t *testing.T) {
	f := secondGameFront(t)
	app := f.App("second map choice")
	app.Layout(640, 480)
	failed := f.OpenMultiplayerMapChoice(app, ui.ScreenMenu, nil)
	if failed != 0 {
		t.Fatalf("failed to decode %d installed map(s)", failed)
	}
	rows := app.HeadlessRows()
	want, ok := map[string]int{base.ROM2EN: 13, base.ROM2RU: 37}[f.Archives.Base.Profile.ID]
	if !ok {
		t.Fatalf("unexpected second-game profile %s", f.Archives.Base.Profile.ID)
	}
	if len(rows) != want {
		t.Fatalf("%s: listed %d multiplayer maps, want %d", f.Archives.Base.Profile.ID, len(rows), want)
	}
	levelPrefix := strings.Split(f.Words.Engine.Text("mapchoice.level"), "%d")[0]
	for _, r := range rows {
		if !strings.Contains(r.Text, levelPrefix) {
			t.Errorf("row %q does not show level (%q)", r.Text, levelPrefix)
		}
	}
	if pix, _, err := app.HeadlessPicture(ui.HeadlessRequest{Part: ui.HeadlessPartFrame}); err != nil || pix == nil {
		t.Fatalf("map choice frame: %v", err)
	}
	if err := app.HeadlessKey("escape"); err != nil {
		t.Fatal(err)
	}
	if app.Screen() != ui.ScreenMenu {
		t.Fatalf("Escape left map choice for %v", app.Screen())
	}
}
