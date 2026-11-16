package main

import (
	"testing"

	"againrom/pkg/launch"
)

// TestSettingsRowsKeepTheRowPositions pins the row of every line of the
// settings view and the two headings' offsets: the starter draws each line at
// 66+24*row, and a heading at 238 and 428.
func TestSettingsRowsKeepTheRowPositions(t *testing.T) {
	want := map[string]int{
		launch.Sound.Key: 0, launch.Volume.Key: 1, launch.Movies.Key: 2, launch.Video.Key: 3,
		launch.Markers.Key: 4, launch.Saves.Key: 5, launch.Window.Key: 6, launch.Mission.Key: 8,
		launch.Picker.Key: 9, launch.Chicken.Key: 10, launch.SmoothRegen.Key: 11, launch.Skill.Key: 12,
		launch.ShopStockMultiplier.Key: 13, launch.Extra.Key: 14,
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
			case 7:
				if got := 66 + 24*i + r.dy; got != 238 || r.head != cHeadAdvanced {
					t.Errorf("advanced heading at %d, want 238", got)
				}
			case 15:
				if got := 66 + 24*i + r.dy; got != 428 || r.head != cHeadStarter {
					t.Errorf("starter heading at %d, want 428", got)
				}
			default:
				t.Errorf("heading at row %d, want rows 7 and 15", i)
			}
		case rowCloseOnPlay:
			if i != 16 {
				t.Errorf("close-after-play at row %d, want 16", i)
			}
		case rowGameProgram:
			if i != 17 {
				t.Errorf("game program at row %d, want 17", i)
			}
		case rowAcceptUnmarked:
			if i != 18 {
				t.Errorf("load unmarked at row %d, want 18", i)
			}
		}
	}
	if len(seen) != len(want) || len(settingsRows) != 19 {
		t.Errorf("%d setting rows and %d lines; want %d and 19", len(seen), len(settingsRows), len(want))
	}
}
