package ui

import (
	"errors"
	"reflect"
	"slices"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

func pressOnMap(a *App, c MissionKeyChord) appInput {
	in := a.headlessIdleInput()
	in.layout = a.activePlayerKeys()
	in.Viewer.Ctrl, in.Viewer.Shift, in.Viewer.Alt = c.Ctrl, c.Shift, c.Alt
	in.readMissionKeys(a.activeMissionKeys(), onlyKey(c.Key), c.Ctrl)
	return in
}

func onMapWithKeys(t *testing.T, profile string) (*App, *[]PlayerMissionKeys) {
	t.Helper()
	a, _ := mkOnMap(t)
	a.SetMissionKeys(profile)
	var written []PlayerMissionKeys
	a.SetPlayerMissionKeys(PlayerMissionKeys{}, func(k PlayerMissionKeys) error {
		written = append(written, k)
		return nil
	})
	return a, &written
}

func TestReboundMissionKeyFiresAndTheDefaultKeyNoLonger(t *testing.T) {
	for _, profile := range []string{"", "pickup-p"} {
		a, written := onMapWithKeys(t, profile)
		k := MissionKeyChord{Key: ebiten.KeyK}
		if err := a.bindMissionKey("inventory", k); err != nil {
			t.Fatal(err)
		}
		if !pressOnMap(a, k).Inventory {
			t.Fatalf("%q: K bound to the inventory does not open it", profile)
		}
		for _, old := range []ebiten.Key{ebiten.KeyI, ebiten.KeyBackquote} {
			if pressOnMap(a, MissionKeyChord{Key: old}).Inventory {
				t.Fatalf("%q: default key %v still opens the inventory", profile, old)
			}
		}
		shifted := MissionKeyChord{Key: ebiten.KeyF5, Shift: true, Alt: true}
		if err := a.bindMissionKey(shortcutSlotName(profile, 0), shifted); err != nil {
			t.Fatal(err)
		}
		if in := pressOnMap(a, shifted); !in.QuickSpell[0] {
			t.Fatalf("%q: Shift+Alt+F5 does not cast spell 1", profile)
		}
		if in := pressOnMap(a, MissionKeyChord{Key: ebiten.KeyF5, Shift: true}); in.QuickSpell[0] {
			t.Fatalf("%q: Shift+F5 without Alt still casts spell 1", profile)
		}
		if len(*written) != 2 || (*written)[1][shortcutSlotName(profile, 0)] != shifted {
			t.Fatalf("%q: stored layouts %v", profile, *written)
		}
	}
}

func TestStolenMissionKeyLeavesTheOldActionUnassigned(t *testing.T) {
	a, _ := onMapWithKeys(t, "")
	e := MissionKeyChord{Key: ebiten.KeyE}
	if err := a.bindMissionKey("patrol", e); err != nil {
		t.Fatal(err)
	}
	if got := a.player.keys["select-all"]; !got.None {
		t.Fatalf("select-all after E moved to patrol = %+v, want not assigned", got)
	}
	if in := pressOnMap(a, e); !in.Patrol || in.SelectAll {
		t.Fatalf("E: patrol %v select-all %v, want true false", in.Patrol, in.SelectAll)
	}
	for _, c := range []MissionKeyChord{{Key: ebiten.KeyE, Ctrl: true}, {Key: ebiten.KeyE, Shift: true}} {
		if pressOnMap(a, c).SelectAll {
			t.Fatalf("%v still selects all", c)
		}
	}
	if pressOnMap(a, MissionKeyChord{Key: ebiten.KeyP}).Patrol {
		t.Fatal("P still patrols after patrol moved to E")
	}
}

func TestMissionKeyDefaultsRestoreTheTable(t *testing.T) {
	for _, profile := range []string{"", "pickup-p"} {
		a, written := onMapWithKeys(t, profile)
		p := missionKeyProfiles[profile]
		for name, c := range map[string]MissionKeyChord{"inventory": {Key: ebiten.KeyK}, "group-3": {Key: ebiten.KeyY, Ctrl: true},
			"highlight": {Key: ebiten.KeyV}, "attack": {Key: ebiten.KeyE}} {
			if err := a.bindMissionKey(name, c); err != nil {
				t.Fatal(name, err)
			}
		}
		if reflect.DeepEqual(missionKeyRows(p, a.player.keys), missionBindings(p)) {
			t.Fatal("the rebound layout reads as the table")
		}
		if err := a.restoreMissionKeys(); err != nil {
			t.Fatal(err)
		}
		if got := missionKeyRows(p, a.player.keys); !reflect.DeepEqual(got, missionBindings(p)) {
			t.Fatalf("%q: rows after Defaults differ from the table", profile)
		}
		if last := (*written)[len(*written)-1]; len(last) != 0 || len(last.Entries(profile)) != 0 {
			t.Fatalf("%q: Defaults stored %v", profile, last)
		}
	}
}

func TestFixedMissionKeysCannotBeCaptured(t *testing.T) {
	for _, profile := range []string{"", "pickup-p"} {
		a, written := onMapWithKeys(t, profile)
		for _, c := range []MissionKeyChord{
			{Key: ebiten.KeyEscape}, {Key: ebiten.KeyF1}, {Key: ebiten.KeyF1, Ctrl: true},
			{Key: ebiten.KeyK, Ctrl: true, Shift: true}, {Key: ebiten.KeyF10, Ctrl: true, Shift: true},
			{Key: ebiten.KeySemicolon, Ctrl: true, Shift: true}, {Key: ebiten.KeyF4, Ctrl: true, Shift: true},
			{Key: ebiten.KeyEnter}, {Key: ebiten.KeyC, Alt: true}, {Key: ebiten.KeyShiftLeft}, {None: true},
			{Key: ebiten.KeyArrowDown}, {Key: ebiten.KeyTab}, {Key: ebiten.KeyPageUp}, {Key: ebiten.KeyPageDown, Shift: true},
		} {
			if err := a.bindMissionKey("book", c); !errors.Is(err, errMissionKeyFixed) {
				t.Fatalf("%q: binding %v answered %v, want the fixed refusal", profile, c, err)
			}
		}
		if err := a.bindMissionKey("kill", MissionKeyChord{Key: ebiten.KeyY}); !errors.Is(err, errMissionKeyUnknown) {
			t.Fatalf("%q: a debug chord's action was rebound: %v", profile, err)
		}
		if err := a.bindMissionKey("help", MissionKeyChord{Key: ebiten.KeyY}); !errors.Is(err, errMissionKeyUnknown) {
			t.Fatalf("%q: F1's action was rebound: %v", profile, err)
		}
		if len(*written) != 0 || len(a.player.keys) != 0 {
			t.Fatalf("%q: a refused capture changed the layout: %v", profile, *written)
		}
	}
}

func TestMissionKeyLayoutListsEachGamesActionsAndReadsBack(t *testing.T) {
	first, second := MissionKeyActions(""), MissionKeyActions("pickup-p")
	for _, tc := range []struct {
		profile string
		names   []string
		has     []string
		lacks   []string
	}{
		{"", first, []string{"patrol", "spell-4", "group-0", "pause"}, []string{"pickup", "support", "help", "kill"}},
		{"pickup-p", second, []string{"pickup", "support", "spell-9"}, []string{"patrol", "help", "chip", "reveal"}},
	} {
		for _, n := range tc.has {
			if !slices.Contains(tc.names, n) {
				t.Fatalf("%q lists no %q: %v", tc.profile, n, tc.names)
			}
		}
		for _, n := range tc.lacks {
			if slices.Contains(tc.names, n) {
				t.Fatalf("%q lists %q", tc.profile, n)
			}
		}
		for i, n := range tc.names {
			if n == "" || slices.Index(tc.names, n) != i {
				t.Fatalf("%q: action %d named %q twice or not at all", tc.profile, i, n)
			}
		}
	}
	k := PlayerMissionKeys{"book": {Key: ebiten.KeyNumpad7, Ctrl: true, Alt: true}, "worn": {None: true}}
	entries := k.Entries("pickup-p")
	if entries["SecondGameKeys.book"] != "Ctrl+Alt+Numpad7" || entries["SecondGameKeys.worn"] != "none" {
		t.Fatalf("entries %v", entries)
	}
	entries["SecondGameKeys.kill"], entries["SecondGameKeys.doll"] = "Y", "Ctrl+Shift+K"
	// Numpad 7 is also a group key; the profile read takes it from the group.
	want := PlayerMissionKeys{"book": k["book"], "worn": k["worn"], "group-7": {Key: ebiten.KeyDigit7}}
	if back := PlayerMissionKeysFrom("pickup-p", entries); !reflect.DeepEqual(back, want) {
		t.Fatalf("read back %v, want %v", back, want)
	}
	if back := PlayerMissionKeysFrom("", entries); len(back) != 0 {
		t.Fatalf("the first game read the second game's keys: %v", back)
	}
}

func TestMissionKeyLayoutStaysOffTheTown(t *testing.T) {
	a, _ := onMapWithKeys(t, "")
	if err := a.bindMissionKey("inventory", MissionKeyChord{Key: ebiten.KeyK}); err != nil {
		t.Fatal(err)
	}
	a.flow.screen = ScreenMenu
	if a.activePlayerKeys() != nil {
		t.Fatal("the player's mission keys apply off the map")
	}
}
