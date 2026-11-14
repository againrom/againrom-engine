package ui

import (
	"reflect"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

func TestQuickSaveAndLoadKeysAreF2AndF3InBothGames(t *testing.T) {
	for _, profile := range []string{"", "pickup-p"} {
		for _, tc := range []struct {
			name        string
			key         ebiten.Key
			ctrl, shift bool
			quickSave   bool
			quickLoad   bool
			saveWindow  bool
			loadWindow  bool
		}{
			{"F2", ebiten.KeyF2, false, false, true, false, false, false},
			{"Shift+F2", ebiten.KeyF2, false, true, true, false, false, false},
			{"Ctrl+F2", ebiten.KeyF2, true, false, false, false, true, false},
			{"F3", ebiten.KeyF3, false, false, false, true, false, false},
			{"Shift+F3", ebiten.KeyF3, false, true, false, true, false, false},
			{"Ctrl+F3", ebiten.KeyF3, true, false, false, false, false, true},
		} {
			var in appInput
			in.Viewer.Ctrl, in.Viewer.Shift = tc.ctrl, tc.shift
			in.readMissionKeys(missionKeyProfiles[profile], onlyKey(tc.key), tc.ctrl)
			got := [4]bool{in.QuickSave, in.QuickLoad, in.SaveGame, in.LoadGame}
			want := [4]bool{tc.quickSave, tc.quickLoad, tc.saveWindow, tc.loadWindow}
			if got != want {
				t.Errorf("profile %q %s: quick save, quick load, save window, load window = %v, want %v", profile, tc.name, got, want)
			}
		}
	}
}

func TestShiftedQuickKeysNoLongerSaveOrLoad(t *testing.T) {
	for _, tc := range []struct {
		profile string
		key     ebiten.Key
	}{
		{"pickup-p", ebiten.KeyF4},
		{"pickup-p", ebiten.KeyF9},
		{"", ebiten.KeyF9},
	} {
		var in appInput
		in.Viewer.Shift = true
		in.readMissionKeys(missionKeyProfiles[tc.profile], onlyKey(tc.key), false)
		if want := (appInput{Viewer: Input{Shift: true}}); !reflect.DeepEqual(in, want) {
			t.Errorf("profile %q Shift+%v set %+v, want nothing", tc.profile, tc.key, in)
		}
	}
}

func TestFirstGameKeepsF4AndF9ForQuickSaveAndLoad(t *testing.T) {
	for _, tc := range []struct {
		key        ebiten.Key
		save, load bool
	}{
		{ebiten.KeyF4, true, false},
		{ebiten.KeyF9, false, true},
	} {
		var in appInput
		in.readMissionKeys(missionKeyProfiles[""], onlyKey(tc.key), false)
		if in.QuickSave != tc.save || in.QuickLoad != tc.load || in.SaveGame || in.LoadGame {
			t.Errorf("first game %v: save=%v load=%v, want %v %v", tc.key, in.QuickSave, in.QuickLoad, tc.save, tc.load)
		}
	}
}
