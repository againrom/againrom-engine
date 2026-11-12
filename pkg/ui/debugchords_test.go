package ui

import (
	"slices"
	"strings"
	"testing"
)

func TestDebugChordsAreBoundOnlyToTheirChords(t *testing.T) {
	got := bindingSource(t)
	for _, tc := range []struct {
		field string
		key   string
	}{
		{"Kill", "ebiten.KeyK"},
		{"Chip", "ebiten.KeySemicolon"},
		{"Reveal", "ebiten.KeyF4"},
	} {
		expr, ok := got[tc.field]
		if !ok {
			t.Fatalf("readAppInput names no %s binding", tc.field)
		}
		if !strings.Contains(expr, tc.key) || !strings.Contains(expr, "shiftHeld") ||
			!strings.Contains(expr, "ctrlHeld()") || strings.Contains(expr, "!ctrlHeld()") {
			t.Errorf("%s is bound to %q, want %s with Ctrl and Shift held", tc.field, expr, tc.key)
		}
		if tc.field != "Reveal" && strings.Contains(expr, "ebiten.KeyL") {
			t.Errorf("%s still reads L: %q", tc.field, expr)
		}
	}
	if load := got["Load"]; !strings.Contains(load, "ebiten.KeyL") || strings.Contains(load, "Semicolon") {
		t.Errorf("Load is bound to %q, want bare L", load)
	}
}

func TestDebugChordsActOnlyAfterTheUnlock(t *testing.T) {
	for _, profile := range []string{"", "pickup-p"} {
		t.Run("profile "+profile, func(t *testing.T) {
			a, v, seam := affOnMap(t)
			a.SetMissionKeys(profile)
			v.sel = selection{affLoID, affHiID}
			x, y := cellPoint(v, affEmptyCol, affEmptyRow)
			press := func() {
				in := affFrame(x, y)
				in.Kill, in.Chip, in.Reveal = true, true, true
				a.step(in, affAt)
			}

			a.SetCheatPrivilege(nil)
			press()
			if len(seam.blows) != 0 || v.FogRevealed() {
				t.Fatalf("locked: blows %v reveal %v, want none", seam.blows, v.FogRevealed())
			}
			a.SetCheatPrivilege(func() bool { return false })
			press()
			if len(seam.blows) != 0 || v.FogRevealed() {
				t.Fatalf("privilege false: blows %v reveal %v, want none", seam.blows, v.FogRevealed())
			}

			a.SetCheatPrivilege(func() bool { return true })
			press()
			want := []struck{{affLoID, true}, {affHiID, true}, {affLoID, false}, {affHiID, false}}
			if !slices.Equal(seam.blows, want) || !v.FogRevealed() {
				t.Fatalf("unlocked: blows %v reveal %v, want %v and reveal on", seam.blows, v.FogRevealed(), want)
			}
		})
	}
}

func TestGridAndReadoutActOnlyAfterTheUnlock(t *testing.T) {
	for _, profile := range []string{"", "pickup-p"} {
		t.Run("profile "+profile, func(t *testing.T) {
			a, v, _ := affOnMap(t)
			a.SetMissionKeys(profile)
			x, y := cellPoint(v, affEmptyCol, affEmptyRow)
			press := func() {
				in := affFrame(x, y)
				in.Grid, in.Readout = true, true
				a.step(in, affAt)
			}
			grid0, readout0 := v.GridOverlay(), v.ReadoutShown()

			for _, unlock := range []func() bool{nil, func() bool { return false }} {
				a.SetCheatPrivilege(unlock)
				press()
				if v.GridOverlay() != grid0 || v.ReadoutShown() != readout0 {
					t.Fatalf("locked: grid %v readout %v, want %v and %v", v.GridOverlay(), v.ReadoutShown(), grid0, readout0)
				}
			}

			a.SetCheatPrivilege(func() bool { return true })
			press()
			if v.GridOverlay() == grid0 || v.ReadoutShown() == readout0 {
				t.Fatalf("unlocked: grid %v readout %v, want both flipped from %v and %v", v.GridOverlay(), v.ReadoutShown(), grid0, readout0)
			}
		})
	}
}

func TestBareLStillOpensLoadOnTheMenu(t *testing.T) {
	a := newTestApp(t, appRows(1), nil)
	a.SetCheatPrivilege(nil)
	a.flow.screen = ScreenMenu
	if err := a.HeadlessKey("l"); err != nil {
		t.Fatal(err)
	}
	if a.Screen() != ScreenLoad {
		t.Fatalf("screen after L on the menu = %v, want ScreenLoad", a.Screen())
	}
}
