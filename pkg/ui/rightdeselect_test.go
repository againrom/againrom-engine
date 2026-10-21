package ui

import (
	"slices"
	"testing"
)

func rightDeselectCast(t *testing.T) (*App, *Viewer) {
	t.Helper()
	a, v := castCursorApp(t)
	pressPanelKey(t, a, "space")
	chooseSpell(t, v, castTarget)
	pressPanelKey(t, a, "space")
	pressPanelKey(t, a, "c")
	if !v.castOnce || !v.hudShown(hudPanelBook) {
		t.Fatalf("setup: C arm once %v book %v", v.castOnce, v.hudShown(hudPanelBook))
	}
	return a, v
}

func TestRightClickDeselectsAfterBookChosenSpell(t *testing.T) {
	a, v := castCursorApp(t)
	pressPanelKey(t, a, "space")
	chooseSpell(t, v, castTarget)
	x, y := cellPoint(v, castCells.ground[0], castCells.ground[1])
	rightUpAt(v, x, y)
	if len(v.sel) != 0 || !v.hudShown(hudPanelBook) || v.selectedSpell != castTarget {
		t.Fatalf("before a cast: sel %v book %v spell %d", v.sel, v.hudShown(hudPanelBook), v.selectedSpell)
	}
	a, v = castCursorApp(t)
	pressPanelKey(t, a, "space")
	chooseSpell(t, v, castTarget)
	ox, oy := cellPoint(v, castCells.other[0], castCells.other[1])
	if ords, _ := tapAt(v, ox, oy); len(ords) != 1 {
		t.Fatalf("setup: cast issued %+v", ords)
	}
	rightUpAt(v, x, y)
	if len(v.sel) != 0 || !v.hudShown(hudPanelBook) || v.selectedSpell != castTarget {
		t.Fatalf("after a cast: sel %v book %v spell %d", v.sel, v.hudShown(hudPanelBook), v.selectedSpell)
	}
}

func TestRightClickUndoesTheOneShotCastArm(t *testing.T) {
	_, v := rightDeselectCast(t)
	x, y := cellPoint(v, castCells.ground[0], castCells.ground[1])
	rightUpAt(v, x, y)
	if v.spellArmed || v.castOnce || v.hudShown(hudPanelBook) || !slices.Equal(v.sel, selection{panelCasterID}) {
		t.Fatalf("C arm: armed %v once %v book %v sel %v", v.spellArmed, v.castOnce, v.hudShown(hudPanelBook), v.sel)
	}
}

func TestRightClickDeselectsAfterACastFromC(t *testing.T) {
	_, v := rightDeselectCast(t)
	ox, oy := cellPoint(v, castCells.other[0], castCells.other[1])
	if ords, _ := tapAt(v, ox, oy); len(ords) != 1 {
		t.Fatalf("setup: cast issued %+v", ords)
	}
	if v.castOnce || v.hudShown(hudPanelBook) {
		t.Fatalf("setup: cast not spent, once %v book %v", v.castOnce, v.hudShown(hudPanelBook))
	}
	x, y := cellPoint(v, castCells.ground[0], castCells.ground[1])
	rightUpAt(v, x, y)
	if len(v.sel) != 0 {
		t.Fatalf("selection after the C cast and a right click: %v", v.sel)
	}
}

func TestRightClickDeselectsAfterLoadWithStoredSpell(t *testing.T) {
	_, v, _ := quickSpellApp(t)
	s := v.SaveApplication()
	s.PressedSpell, s.SpellBookOpen = 17, false
	if err := v.RestoreSaveApplication(s); err != nil {
		t.Fatal(err)
	}
	if v.selectedSpell != 17 || len(v.sel) != 1 {
		t.Fatalf("setup: spell %d sel %v", v.selectedSpell, v.sel)
	}
	v.command(appInput{SecondaryReleased: true, CursorX: 10, CursorY: 10})
	if len(v.sel) != 0 {
		t.Fatalf("selection after load and a right click: %v", v.sel)
	}
}

func TestRightClickStillAbsorbedByAttackAimedAndItemCast(t *testing.T) {
	for name, arm := range map[string]func(*Viewer){
		"attack":   func(v *Viewer) { v.armed = true },
		"aimed":    func(v *Viewer) { v.aimed = commandMove },
		"itemcast": func(v *Viewer) { v.itemCast = &itemCastSelection{} },
		"key arm":  func(v *Viewer) { v.selectedSpell = 17; v.armSpell(); v.spellNeedsBook = false },
	} {
		t.Run(name, func(t *testing.T) {
			_, v, _ := quickSpellApp(t)
			arm(v)
			v.cancelMapCommand()
			if len(v.sel) != 1 || v.armed || v.aimed != commandNone || v.itemCast != nil || v.spellArmed {
				t.Fatalf("mode not cancelled with the selection kept: sel %v", v.sel)
			}
		})
	}
}

func quickArmedViewer(t *testing.T) *Viewer {
	t.Helper()
	_, v, slots := quickSpellApp(t)
	*slots = [4]uint32{17}
	if v.hudShown(hudPanelBook) {
		v.toggleHudPanel(hudPanelBook)
	}
	v.quickSpell(0, false, -1, -1)
	if !v.spellArmed || !v.castOnce || v.hudShown(hudPanelBook) {
		t.Fatalf("setup: F arm armed %v once %v book %v", v.spellArmed, v.castOnce, v.hudShown(hudPanelBook))
	}
	return v
}

func TestRightClickCancelsAQuickSpellArmBeforeItsCast(t *testing.T) {
	v := quickArmedViewer(t)
	v.cancelMapCommand()
	if v.spellArmed || v.castOnce || len(v.sel) != 1 {
		t.Fatalf("F arm: armed %v once %v sel %v", v.spellArmed, v.castOnce, v.sel)
	}
}

func TestRightClickDeselectsAfterAQuickSpellCast(t *testing.T) {
	v := quickArmedViewer(t)
	v.endCastOnce()
	if v.hudShown(hudPanelBook) {
		t.Fatal("the spend opened the book")
	}
	v.cancelMapCommand()
	if len(v.sel) != 0 {
		t.Fatalf("selection after the F cast and a right click: %v", v.sel)
	}
}
