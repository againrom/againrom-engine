package ui

import (
	"fmt"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

func TestSecondMissionBookShortcutsUseAllNineFunctionKeys(t *testing.T) {
	for slot := 0; slot < 9; slot++ {
		t.Run(fmt.Sprint(slot+4), func(t *testing.T) {
			a, v, slots := quickSpellApp(t)
			a.SetMissionKeys("pickup-p")
			v.selectedSpell = 89
			if err := a.HeadlessKey(fmt.Sprintf("ctrl-f%d", slot+4)); err != nil {
				t.Fatal(err)
			}
			if len(slots) != 9 || slots[slot] != 89 {
				t.Fatalf("Ctrl+F%d stored %v, want slot %d = 89", slot+4, slots, slot)
			}
			v.selectedSpell = 0
			if err := a.HeadlessKey(fmt.Sprintf("f%d", slot+4)); err != nil {
				t.Fatal(err)
			}
			if v.selectedSpell != 89 || v.GridOverlay() || v.FPSShown() {
				t.Fatalf("F%d did not recall only its spell", slot+4)
			}
		})
	}
}

func TestSecondMissionControlLettersDoNotIssuePlainOrders(t *testing.T) {
	if MissionShortcutCount("") != 4 || MissionShortcutCount("pickup-p") != 9 {
		t.Fatal("shortcut admission does not follow the mission profile")
	}
	for _, key := range []struct {
		name string
		key  ebiten.Key
		set  func(appInput) bool
	}{
		{"g", ebiten.KeyG, func(in appInput) bool { return in.Guard }},
		{"d", ebiten.KeyD, func(in appInput) bool { return in.Defend }},
		{"c", ebiten.KeyC, func(in appInput) bool { return in.Cast }},
		{"s", ebiten.KeyS, func(in appInput) bool { return in.March }},
		{"t", ebiten.KeyT, func(in appInput) bool { return in.StandGround }},
		{"r", ebiten.KeyR, func(in appInput) bool { return in.PlayerRetreat }},
		{"m", ebiten.KeyM, func(in appInput) bool { return in.Move }},
	} {
		for _, profile := range []string{"", "pickup-p"} {
			for _, ctrl := range []bool{false, true} {
				var in appInput
				in.readMissionKeys(missionKeyProfiles[profile], onlyKey(key.key), ctrl)
				want := profile == "" || !ctrl
				if got := key.set(in); got != want {
					t.Errorf("%q Ctrl=%v %s = %v, want %v", profile, ctrl, key.name, got, want)
				}
			}
		}
	}
}

func TestMissionLetterActionsKeepTheFirstGameAndGateControlInTheSecond(t *testing.T) {
	for _, profile := range []string{"", "pickup-p"} {
		for _, key := range []string{"g", "d", "c", "s", "t", "r", "m"} {
			for _, ctrl := range []bool{false, true} {
				t.Run(fmt.Sprintf("%s/%s/%v", profile, key, ctrl), func(t *testing.T) {
					a, v, _ := quickSpellApp(t)
					a.SetMissionKeys(profile)
					v.selectedSpell = 89
					calls := 0
					a.flow.stance = func(uint32, StanceOrder) { calls++ }
					v.SetPlayerRetreatSink(func([]uint32) { calls++ })
					name := key
					if ctrl {
						name = "ctrl-" + key
					}
					if err := a.HeadlessKey(name); err != nil {
						t.Fatal(err)
					}
					acted := calls != 0 || v.aimed != commandNone || v.spellModeLive()
					if want := profile == "" || !ctrl; acted != want {
						t.Fatalf("key %s: calls=%d mode=%d, want acted=%v", name, calls, v.missionMode(), want)
					}
				})
			}
		}
	}
}

func TestSecondMissionShortcutModifiersKeepTheDebugUnlock(t *testing.T) {
	for _, key := range []string{"f4", "f10", "f11"} {
		for _, unlocked := range []bool{false, true} {
			a, v, slots := quickSpellApp(t)
			a.SetMissionKeys("pickup-p")
			a.SetCheatPrivilege(func() bool { return unlocked })
			v.selectedSpell = 89
			beforeGrid, beforeReadout := v.GridOverlay(), v.readoutHidden
			if err := a.HeadlessKey("ctrl-shift-" + key); err != nil {
				t.Fatal(err)
			}
			if *slots != [9]uint32{} {
				t.Fatal("debug chord assigned a spell")
			}
			changed := v.FogRevealed() || beforeGrid != v.GridOverlay() || beforeReadout != v.readoutHidden
			if changed != unlocked {
				t.Fatalf("%s unlocked=%v changed=%v", key, unlocked, changed)
			}
			if err := a.HeadlessKey("shift-" + key); err != nil {
				t.Fatal(err)
			}
			if *slots != [9]uint32{} {
				t.Fatal("unknown Shift slot action assigned a spell")
			}
		}
	}
}

func TestMissionQuickSaveAndLoadKeysFollowTheProfile(t *testing.T) {
	for _, tc := range []struct {
		profile            string
		key                ebiten.Key
		shift, ctrl        bool
		wantSave, wantLoad bool
	}{
		{"pickup-p", ebiten.KeyF4, true, false, false, false},
		{"pickup-p", ebiten.KeyF4, false, false, false, false},
		{"pickup-p", ebiten.KeyF4, true, true, false, false},
		{"pickup-p", ebiten.KeyF9, true, false, false, false},
		{"", ebiten.KeyF4, false, false, true, false},
		{"", ebiten.KeyF4, true, false, false, false},
		{"", ebiten.KeyF9, false, false, false, true},
		{"", ebiten.KeyF9, true, false, false, false},
	} {
		var in appInput
		in.Viewer.Shift = tc.shift
		in.readMissionKeys(missionKeyProfiles[tc.profile], onlyKey(tc.key), tc.ctrl)
		if in.QuickSave != tc.wantSave || in.QuickLoad != tc.wantLoad {
			t.Errorf("%q key %v shift=%v ctrl=%v: save=%v load=%v, want %v %v", tc.profile, tc.key, tc.shift, tc.ctrl, in.QuickSave, in.QuickLoad, tc.wantSave, tc.wantLoad)
		}
	}
}
