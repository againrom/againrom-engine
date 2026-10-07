package ui

import (
	"image"
	"testing"
	"time"
)

// TestMissionCharacterPaneRightUpCoversEveryPanePixel is the coordinate-free
// half of TOWN-344: the handler receives no child rectangle or slot-map result.
// Exhausting the 160x242 widget makes a seam, corner gate, presentation mode or
// empty body unable to leave a pixel that falls through to the map path.
func TestMissionCharacterPaneRightUpCoversEveryPanePixel(t *testing.T) {
	v := missionPaneViewer(t)
	pane, ok := v.characterPaneRect()
	if !ok {
		t.Fatal("the shipped mission frame has no character pane")
	}
	for y := pane.Min.Y; y < pane.Max.Y; y++ {
		for x := pane.Min.X; x < pane.Max.X; x++ {
			v.sel = selection{11, 22}
			v.armed = true
			v.aimed = commandNone
			v.selectedSpell = 0
			v.cmdOverlayHidden = true
			// A pane right-up posts 0x405 directly. It does not inherit the
			// map widget's marked-drag suppression.
			v.rightPanned = true
			ords, ordered := rightUpAt(v, x, y)
			if ordered || len(ords) != 0 {
				t.Fatalf("pane pixel (%d,%d) issued orders %+v (ok=%v)", x, y, ords, ordered)
			}
			if v.armed || v.aimed != commandNone || v.selectedSpell != 0 {
				t.Fatalf("pane pixel (%d,%d) left a command mode armed", x, y)
			}
			if len(v.sel) != 2 || v.sel[0] != 11 || v.sel[1] != 22 {
				t.Fatalf("pane pixel (%d,%d) changed selection to %v while cancelling a mode", x, y, v.sel)
			}
			if v.cmdOverlayHidden {
				t.Fatalf("pane pixel (%d,%d) did not restore the command overlay", x, y)
			}
		}
	}
}

// TestMissionCharacterPaneRightUpSharesTheMapCancelDecision covers all four
// states of AI-INPUT-127's ordinary 0x405 decision. The three armed rows keep
// selection; the unarmed row clears it.
func TestMissionCharacterPaneRightUpSharesTheMapCancelDecision(t *testing.T) {
	for _, tc := range []struct {
		name         string
		arm          func(*Viewer)
		wantSelected bool
	}{
		{"attack", func(v *Viewer) { v.armed = true }, true},
		{"aimed order", func(v *Viewer) { v.aimed = commandMove }, true},
		{"selected spell", func(v *Viewer) { v.selectedSpell = 17 }, true},
		{"no mode", func(*Viewer) {}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			v := missionPaneViewer(t)
			pane, ok := v.characterPaneRect()
			if !ok {
				t.Fatal("the shipped mission frame has no character pane")
			}
			v.sel = selection{11, 22}
			tc.arm(v)
			p := image.Pt((pane.Min.X+pane.Max.X)/2, (pane.Min.Y+pane.Max.Y)/2)
			if ords, ordered := rightUpAt(v, p.X, p.Y); ordered || len(ords) != 0 {
				t.Fatalf("right-up issued orders %+v (ok=%v)", ords, ordered)
			}
			if v.armed || v.aimed != commandNone || v.selectedSpell != 0 {
				t.Fatalf("right-up left state armed: attack=%v aimed=%d spell=%d",
					v.armed, v.aimed, v.selectedSpell)
			}
			if got := len(v.sel) != 0; got != tc.wantSelected {
				t.Fatalf("selection after cancel = %v; want selected=%v", v.sel, tc.wantSelected)
			}
		})
	}
}

// TestMissionCharacterPaneSecondaryGestureCannotPan proves both ownership
// directions: a down edge on the pane stays out of the map after leaving, and
// an unowned map gesture produces no delta while its cursor crosses the pane.
func TestMissionCharacterPaneSecondaryGestureCannotPan(t *testing.T) {
	frozen := time.Unix(1_700_000_000, 0)

	t.Run("a pane down edge owns the gesture until an outside release", func(t *testing.T) {
		v := stepViewer(t)
		v.commandMode = true
		pane, ok := v.characterPaneRect()
		if !ok {
			t.Fatal("the test frame has no character pane")
		}
		inside := image.Pt((pane.Min.X+pane.Max.X)/2, (pane.Min.Y+pane.Max.Y)/2)
		outside := image.Pt(pane.Min.X-80, inside.Y)
		v.sel = selection{11}
		x0, y0 := v.Camera().X, v.Camera().Y

		v.step(Input{SecondaryDown: true, CursorX: inside.X, CursorY: inside.Y}, frozen)
		v.command(appInput{
			Viewer:           Input{SecondaryDown: true, CursorX: inside.X, CursorY: inside.Y},
			SecondaryPressed: true, CursorX: inside.X, CursorY: inside.Y,
		})
		if !v.paneSecondaryGrab {
			t.Fatal("the pane did not retain its secondary down edge")
		}

		for _, p := range []image.Point{outside, outside.Add(image.Pt(-20, 10))} {
			v.step(Input{SecondaryDown: true, CursorX: p.X, CursorY: p.Y}, frozen)
			v.command(appInput{
				Viewer:  Input{SecondaryDown: true, CursorX: p.X, CursorY: p.Y},
				CursorX: p.X, CursorY: p.Y,
			})
		}
		v.step(Input{CursorX: outside.X, CursorY: outside.Y}, frozen)
		v.command(appInput{SecondaryReleased: true, CursorX: outside.X, CursorY: outside.Y})

		if v.Camera().X != x0 || v.Camera().Y != y0 || v.rightPanned {
			t.Fatalf("pane-owned gesture moved/marked camera: (%v,%v)->(%v,%v), marked=%v",
				x0, y0, v.Camera().X, v.Camera().Y, v.rightPanned)
		}
		if v.paneSecondaryGrab {
			t.Fatal("outside release left pane ownership armed")
		}
		if len(v.sel) != 1 {
			t.Fatalf("outside release on a pane-owned gesture changed selection to %v", v.sel)
		}
		// Ownership ended. A fresh ordinary map right-up reaches the shared
		// cancel and proves no stale swallow survived the release.
		rightUpAt(v, outside.X, outside.Y)
		if len(v.sel) != 0 {
			t.Fatalf("fresh map cancel was swallowed after pane release: selection %v", v.sel)
		}
	})

	t.Run("a map gesture crossing the pane banks no hidden delta", func(t *testing.T) {
		v := stepViewer(t)
		v.commandMode = true
		pane, ok := v.characterPaneRect()
		if !ok {
			t.Fatal("the test frame has no character pane")
		}
		inside := image.Pt((pane.Min.X+pane.Max.X)/2, (pane.Min.Y+pane.Max.Y)/2)
		insideEdge := image.Pt(pane.Max.X-1, inside.Y)
		outside := image.Pt(pane.Min.X-80, inside.Y)
		v.sel = selection{11}
		x0, y0 := v.Camera().X, v.Camera().Y

		v.step(Input{SecondaryDown: true, CursorX: outside.X, CursorY: outside.Y}, frozen)
		v.command(appInput{
			Viewer:           Input{SecondaryDown: true, CursorX: outside.X, CursorY: outside.Y},
			SecondaryPressed: true, CursorX: outside.X, CursorY: outside.Y,
		})
		// The rightmost pane pixel is also inside the window's edge-scroll
		// band. Secondary ownership must suppress both that pointer term and
		// the right-drag delta; the centre then proves no hidden distance was
		// banked behind the pane.
		for _, p := range []image.Point{insideEdge, inside} {
			v.step(Input{SecondaryDown: true, CursorX: p.X, CursorY: p.Y}, frozen)
			v.command(appInput{
				Viewer:  Input{SecondaryDown: true, CursorX: p.X, CursorY: p.Y},
				CursorX: p.X, CursorY: p.Y,
			})
		}
		v.step(Input{CursorX: inside.X, CursorY: inside.Y}, frozen)
		v.command(appInput{SecondaryReleased: true, CursorX: inside.X, CursorY: inside.Y})

		if v.Camera().X != x0 || v.Camera().Y != y0 || v.rightPanned {
			t.Fatalf("crossing the pane moved/marked camera: (%v,%v)->(%v,%v), marked=%v",
				x0, y0, v.Camera().X, v.Camera().Y, v.rightPanned)
		}
		if len(v.sel) != 0 {
			t.Fatalf("right-up inside the pane did not run 0x405: selection %v", v.sel)
		}
	})
}

// TestMissionCharacterPaneRightHoldKeepsBackpackLeftClickLive is the
// both-buttons boundary of TOWN-344: the pane owns the secondary gesture, not
// the whole input frame. A complete primary press/release over rect A must
// still reach the existing Backpack action while right remains physically
// held.
func TestMissionCharacterPaneRightHoldKeepsBackpackLeftClickLive(t *testing.T) {
	v := missionPaneViewer(t)
	v.commandMode = true
	pane, ok := v.characterPaneRect()
	if !ok {
		t.Fatal("the shipped mission frame has no character pane")
	}
	r := CharacterPaneCornerRect(pane, CharacterPaneBackpack)
	p := image.Pt(r.Min.X+1, r.Min.Y+1)
	if !v.hudShown(hudPanelPack) {
		t.Fatal("setup: a fresh viewer has the backpack hidden")
	}

	v.step(Input{SecondaryDown: true, CursorX: p.X, CursorY: p.Y}, commandFrozen)
	v.command(appInput{
		Viewer:           Input{SecondaryDown: true, CursorX: p.X, CursorY: p.Y},
		SecondaryPressed: true, CursorX: p.X, CursorY: p.Y,
	})
	v.step(Input{PrimaryDown: true, SecondaryDown: true, CursorX: p.X, CursorY: p.Y}, commandFrozen)
	v.command(appInput{
		Viewer:         Input{PrimaryDown: true, SecondaryDown: true, CursorX: p.X, CursorY: p.Y},
		PrimaryPressed: true, CursorX: p.X, CursorY: p.Y,
	})
	v.step(Input{SecondaryDown: true, CursorX: p.X, CursorY: p.Y}, commandFrozen)
	v.command(appInput{
		Viewer:          Input{SecondaryDown: true, CursorX: p.X, CursorY: p.Y},
		PrimaryReleased: true, CursorX: p.X, CursorY: p.Y,
	})

	if v.hudShown(hudPanelPack) {
		t.Fatal("a left click on Backpack was swallowed while right was held")
	}
	if !v.paneSecondaryGrab {
		t.Fatal("the left click incorrectly released the pane's still-held secondary gesture")
	}
}

// TestMissionCharacterPaneRightHoldDoesNotStrandPackDragRelease covers the
// other primary route that crosses the pane's secondary owner. The drag is
// reached through a real pack press and TapSlop move; releasing left over the
// pane while right stays down must run the inventory's ordinary cleanup.
func TestMissionCharacterPaneRightHoldDoesNotStrandPackDragRelease(t *testing.T) {
	v, _ := openDollAt(t)
	cx, cy := packCellCenter(t, v, 0)
	dollPress(v, cx, cy)
	dollMove(v, cx+TapSlop, cy)
	if !v.invGrab || !v.dragActive || v.dragCandKind != dragFromPack {
		t.Fatalf("setup: invGrab=%v dragActive=%v kind=%v, want an active pack drag",
			v.invGrab, v.dragActive, v.dragCandKind)
	}

	pane, ok := v.characterPaneRect()
	if !ok {
		t.Fatal("the shipped mission frame has no character pane")
	}
	r := CharacterPaneCornerRect(pane, CharacterPaneBackpack)
	p := image.Pt(r.Min.X+1, r.Min.Y+1)
	v.step(Input{PrimaryDown: true, SecondaryDown: true, CursorX: p.X, CursorY: p.Y}, commandFrozen)
	v.command(appInput{
		Viewer:           Input{PrimaryDown: true, SecondaryDown: true, CursorX: p.X, CursorY: p.Y},
		SecondaryPressed: true, CursorX: p.X, CursorY: p.Y,
	})
	if !v.paneSecondaryGrab {
		t.Fatal("setup: the pane did not own the held secondary gesture")
	}
	v.step(Input{SecondaryDown: true, CursorX: p.X, CursorY: p.Y}, commandFrozen)
	v.command(appInput{
		Viewer:          Input{SecondaryDown: true, CursorX: p.X, CursorY: p.Y},
		PrimaryReleased: true, CursorX: p.X, CursorY: p.Y,
	})

	if v.invGrab || v.dragActive || v.dragCandKind != dragNone || v.dragIcon != nil {
		t.Fatalf("left release under held right left drag state: invGrab=%v active=%v kind=%v icon=%v",
			v.invGrab, v.dragActive, v.dragCandKind, v.dragIcon != nil)
	}
	if !v.paneSecondaryGrab {
		t.Fatal("the inventory release incorrectly released the still-held secondary gesture")
	}
}
