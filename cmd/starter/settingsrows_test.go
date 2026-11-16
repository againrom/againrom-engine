package main

import (
	"testing"

	"againrom/pkg/launch"
)

// TestSettingsRowsKeepTheRowPositions pins the row of every line of the
// settings view and the two headings' offsets.
func TestSettingsRowsKeepTheRowPositions(t *testing.T) {
	want := map[string]int{
		launch.Sound.Key: 0, launch.Volume.Key: 1, launch.Movies.Key: 2, launch.Video.Key: 3,
		launch.Markers.Key: 4, launch.Saves.Key: 5, launch.Window.Key: 6, launch.CursorInWindow.Key: 7,
		launch.Mission.Key: 9, launch.Picker.Key: 10, launch.Chicken.Key: 11, launch.SmoothRegen.Key: 12, launch.Skill.Key: 13,
		launch.ShopStockMultiplier.Key: 14, launch.Extra.Key: 15,
		launch.WindowFilter.Key: 16,
	}
	seen := map[string]bool{}
	for i, r := range settingsRows {
		switch r.kind {
		case rowSetting:
			if row, ok := want[r.key]; !ok || row != i || seen[r.key] {
				t.Errorf("row %d draws %q; want row %d once", i, r.key, want[r.key])
			}
			seen[r.key] = true
		case rowHeading:
			switch i {
			case 8:
				if got := 66 + 24*i + r.dy; got != 262 || r.head != cHeadAdvanced {
					t.Errorf("advanced heading at %d, want 262", got)
				}
			case 17:
				if got := 66 + 24*i + r.dy; got != 476 || r.head != cHeadStarter {
					t.Errorf("starter heading at %d, want 476", got)
				}
			default:
				t.Errorf("heading at row %d, want rows 8 and 17", i)
			}
		case rowCloseOnPlay:
			if i != 18 {
				t.Errorf("close-after-play at row %d, want 18", i)
			}
		case rowGameProgram:
			if i != 19 {
				t.Errorf("game program at row %d, want 19", i)
			}
		case rowAcceptUnmarked:
			if i != 20 {
				t.Errorf("load unmarked at row %d, want 20", i)
			}
		}
	}
	if len(seen) != len(want) || len(settingsRows) != 21 {
		t.Errorf("%d setting rows and %d lines; want %d and 21", len(seen), len(settingsRows), len(want))
	}
	for language, label := range map[string]string{
		"english": "Keep the mouse in the game window",
		"russian": "Держать мышь в окне игры",
	} {
		a := rigWithLanguage(t, language).app()
		openSettings(t, a)
		items := a.layout()
		found := false
		for i, it := range items {
			if it.text == label {
				found = true
				if textWidth(it.text) > it.rect.Dx() {
					t.Errorf("%s: label %q is clipped", language, label)
				}
			}
			if it.act.kind == aNone {
				continue
			}
			for _, other := range items[i+1:] {
				if other.act.kind != aNone && it.rect.Overlaps(other.rect) {
					t.Errorf("%s: controls %q and %q overlap", language, it.text, other.text)
				}
			}
		}
		if !found {
			t.Errorf("%s: settings lack %q", language, label)
		}
		box, lines := a.outputBox()
		if box.Min.Y < a.actionsY()+72 || 4+18*lines > box.Dy() {
			t.Errorf("%s: Check output overlaps the status or clips its lines", language)
		}
	}
}
