package ui

import (
	"image"
	"image/color"
	"testing"
	"time"
)

// The inventory's own boxes swallow their own clicks (hotfix,
// docs/hotfix/LEDGER.md).
//
// Every case below is stated over the ORDERS AND THE SELECTION rather than over
// the guard's own flag: what the defect looked like to the owner is a press on
// the window selecting whatever stood under it and a left click walking the
// party there, so that is what these assert.
//
// THE SUBJECT UNDER TEST IS THE DOLL BOX SINCE 0140. The one centred window
// became two boxes — the doll in the bottom left and the pack bar along the
// bottom (inventory.go's own header) — and this file follows the box that kept
// the toggle. The bar's own swallow is the same statement in command.go and is
// covered by the double-click cases in invequip_test.go.

// openInventoryAt is a viewer on a map with two entities, a subject naming the
// first, the doll box OPEN, and that box's own rectangle — everything a press
// needs a place to land inside and outside of.
func openInventoryAt(t *testing.T) (*Viewer, image.Rectangle) {
	t.Helper()
	a := inventoryTestApp(t)
	v := a.flow.viewer
	// THE SHIPPED WINDOW'S WIDTH, and not this harness's own 640x480 (0140).
	layoutViewport(v, MenuWindowW, wornBoxFixtureH)
	v.SetEntities([]MapEntity{{ID: 5, Cell: image.Pt(1, 1)}, {ID: 6, Cell: image.Pt(2, 2)}})
	v.sel = selection{5}
	// FOUR CARRIED ELEMENTS, so the pack bar's first four cells name one each:
	// a cell past the end of the pack names nothing (inventory.go), and the
	// double-click cases in invequip_test.go press on cells 0 to 3.
	v.SetInventorySubject(packOf(5,
		solidPic(8, 8, color.RGBA{R: 0xff, A: 0xff}),
		solidPic(8, 8, color.RGBA{G: 0xff, A: 0xff}),
		solidPic(8, 8, color.RGBA{B: 0xff, A: 0xff}),
		solidPic(8, 8, color.RGBA{R: 0xff, G: 0xff, A: 0xff}),
	))
	box, ok := v.dollBox()
	if !ok {
		t.Fatal("setup: dollBox answered false with the subject selected")
	}
	if box.Dx() <= 0 || box.Dy() <= 0 {
		t.Fatalf("setup: doll box %v has no area", box)
	}
	return v, box
}

// TestTheWindowRectIsWhereThePictureIsDrawn pins the one property the hit test
// rests on: the rectangle a press is judged against is the rectangle the
// picture's own BODY occupies, origin and size both.
func TestTheWindowRectIsWhereThePictureIsDrawn(t *testing.T) {
	v, box := openInventoryAt(t)

	// panelPresent refuses without a font, which openInventoryAt does not
	// set: this case is about geometry, so it supplies one rather than
	// moving the fixture.
	v.SetCardFont(panelFont())
	pic, at, ok := v.panelPresent()
	if !ok {
		t.Fatal("panelPresent answered false with the doll box open")
	}
	drawn := image.Rectangle{Min: at, Max: at.Add(pic.Bounds().Size())}
	body := image.Rect(drawn.Min.X+characterPaneSeamW, drawn.Min.Y, drawn.Max.X, drawn.Max.Y)
	if body != box {
		t.Errorf("the picture's own body is drawn on %v and presses are judged against %v", body, box)
	}
}

// TestARightPressOverTheOpenWindowIssuesNoOrder is the owner's own report:
// the click fell through as a move order. Reverting the guard in command
// reddens this on the order count.
func TestARightPressOverTheOpenWindowIssuesNoOrder(t *testing.T) {
	v, box := openInventoryAt(t)
	cx, cy := (box.Min.X+box.Max.X)/2, (box.Min.Y+box.Max.Y)/2

	ords, ok := tapAt(v, cx, cy)
	if ok || len(ords) != 0 {
		t.Errorf("a left click at (%d,%d), inside the box %v, issued %d order(s) (ok=%v)",
			cx, cy, box, len(ords), ok)
	}

	mapX := v.frameW - MissionPanelW - 50
	ords, ok = tapAt(v, mapX, cy)
	if !ok || len(ords) != 1 {
		t.Errorf("a left click on the map view at (%d,%d) issued %d order(s) (ok=%v), want 1",
			mapX, cy, len(ords), ok)
	}
}

// TestATapOverTheOpenWindowLeavesTheSelectionAlone is the other half of the
// report: the click fell through as a selection.
func TestATapOverTheOpenWindowLeavesTheSelectionAlone(t *testing.T) {
	v, box := openInventoryAt(t)
	cx, cy := (box.Min.X+box.Max.X)/2, (box.Min.Y+box.Max.Y)/2

	v.command(appInput{PrimaryPressed: true, CursorX: cx, CursorY: cy})
	v.command(appInput{PrimaryReleased: true, CursorX: cx, CursorY: cy})

	if len(v.sel) != 1 || v.sel[0] != 5 {
		t.Errorf("selection = %v after a tap on the box, want the subject alone", v.sel)
	}
	if v.held {
		t.Error("held is still raised after a release the box swallowed")
	}
	if v.invGrab {
		t.Error("invGrab is still latched after the gesture ended")
	}
}

// TestAGestureBegunOnTheWindowPansNothingWhenItLeavesIt is the drag half. The
// latch, not the point test, is what carries it: the cursor is outside the
// box by the time the camera would move.
func TestAGestureBegunOnTheWindowPansNothingWhenItLeavesIt(t *testing.T) {
	v, box := openInventoryAt(t)
	cx, cy := (box.Min.X+box.Max.X)/2, (box.Min.Y+box.Max.Y)/2
	before := v.cam

	// The press, on the box. step runs first on the map arm, so the anchor
	// is taken before command latches — and the anchor pans zero.
	v.step(Input{PrimaryDown: true, CursorX: cx, CursorY: cy}, time.Unix(1_700_000_000, 0))
	v.command(appInput{PrimaryPressed: true, CursorX: cx, CursorY: cy})
	if !v.invGrab {
		t.Fatal("invGrab was not latched by a press inside the box")
	}

	// The drag, well clear of the box on the far side of the slop.
	out := box.Max.X + 200
	v.step(Input{PrimaryDown: true, CursorX: out, CursorY: cy}, time.Unix(1_700_000_000, 0))
	if v.cam.X != before.X || v.cam.Y != before.Y {
		t.Errorf("the camera moved to (%v,%v) from (%v,%v) on a drag begun on the box",
			v.cam.X, v.cam.Y, before.X, before.Y)
	}
	if v.marqueeScreenRects() != nil {
		t.Error("a gesture the box took drew a selection rectangle")
	}

	// The release, outside the box: it selects nothing either, because the
	// press it would belong to was never delivered.
	v.command(appInput{PrimaryReleased: true, CursorX: out, CursorY: cy})
	if len(v.sel) != 1 || v.sel[0] != 5 {
		t.Errorf("selection = %v after a drag begun on the box, want the subject alone", v.sel)
	}
}

// TestASwitchedOffBoxCapturesNothing keeps the guard from becoming a permanent
// dead zone on the screen: with the doll box switched off, the very same press
// captures nothing as inventory input and orders nothing on the map.
//
// IT NO LONGER ASSERTS THE PRESS REACHES THE MAP: since 1026 the whole
// right-hand column is off the map's own viewport regardless of any switch
// (DIV-212, groundCellAt) — a box turned off leaves that part of the
// column undrawn, not handed to the map underneath, because there is no map
// underneath it any more. The press is refused outright.
func TestASwitchedOffBoxCapturesNothing(t *testing.T) {
	v, box := openInventoryAt(t)
	cx, cy := (box.Min.X+box.Max.X)/2, (box.Min.Y+box.Max.Y)/2

	v.sel = nil

	if v.inventoryCaptures(cx, cy) {
		t.Fatal("a box with nothing selected captured a press")
	}
	ords, ok := tapAt(v, cx, cy)
	if ok || len(ords) != 0 {
		t.Errorf("a left click with the box switched off, still inside the reserved column, "+
			"issued %d order(s) (ok=%v), want 0 — the column is not map ground", len(ords), ok)
	}
}
