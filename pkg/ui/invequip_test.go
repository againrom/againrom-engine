package ui

import (
	"image"
	"image/color"
	"testing"
)

// A double-click on a pack cell raises one one-shot equip request. Every
// case below reuses openInventoryAt's own fixture (invclick_test.go) — a
// doll box open over a two-entity map whose subject carries four elements
// — because the double-click state this file exercises only ever runs
// inside the swallow branch that fixture is built to land presses in, and
// tapAt (command_test.go) already drives a press-and- release through the
// shipped shell in one frame, which is the same "one press, one call" shape
// command.go's own count is built on.

// packCellCenter is the window-pixel point at the centre of pack bar cell i,
// built from the same two calls inventoryPackCellAt itself makes: packBar's
// own rectangle and packCellRects' own cells. A test that found this point any
// other way would be exercising a second geometry rather than the one the
// production code is built on (inventory.go's own "one geometry" doc on
// inventoryPackCellAt).
func packCellCenter(t *testing.T, v *Viewer, cell int) (int, int) {
	t.Helper()
	bar, cols, ok := v.packBar()
	if !ok {
		t.Fatal("setup: the viewer draws no pack bar")
	}
	r := packCellRects(bar, cols)[cell]
	return (r.Min.X + r.Max.X) / 2, (r.Min.Y + r.Max.Y) / 2
}

// figureCenter is the same for the character pane's figure: real geometry
// inside a box the inventory swallows presses on, that inventoryPackCellAt
// still answers no cell for.
func figureCenter(box image.Rectangle) (int, int) {
	r := characterPaneFigureRect(box)
	return (r.Min.X + r.Max.X) / 2, (r.Min.Y + r.Max.Y) / 2
}

// idleAt is one map-screen frame with the cursor held still and no button
// edge — the gap between two presses. The cursor stays inside the bar
// throughout every case below, so the bar keeps swallowing on every one
// of these frames and invClickFrames counts exactly what command.go's own
// comment says it counts: one per call, with nothing skipped.
func idleAt(v *Viewer, x, y int) {
	v.command(appInput{CursorX: x, CursorY: y})
}

// TestInventoryPackCellAt pins the geometry the double-click test below is
// built on: a point inside a pack cell answers that cell's own element, a
// point on the doll box's figure answers none, and a viewer holding no
// subject answers none from anywhere at all.
//
// CLOSING THE DOLL BOX DOES NOT CLOSE THE BAR, and that is 0140's own
// separation asserted rather than assumed: the bar is permanent furniture and
// the toggle is the doll's (inventory.go's header). Before that story this
// same case read the other way, and a reader who expects the old answer should
// find the reversal here.
func TestInventoryPackCellAt(t *testing.T) {
	v, box := openInventoryAt(t)

	cx, cy := packCellCenter(t, v, 3)
	if cell, ok := v.inventoryPackCellAt(cx, cy); !ok || cell != 3 {
		t.Errorf("inventoryPackCellAt(%d,%d) = (%d,%v), want (3,true)", cx, cy, cell, ok)
	}

	fx, fy := figureCenter(box)
	if cell, ok := v.inventoryPackCellAt(fx, fy); ok {
		t.Errorf("inventoryPackCellAt over the figure box = (%d,true), want no cell", cell)
	}

	v.toggleHudPanel(hudPanelDoll)
	if cell, ok := v.inventoryPackCellAt(cx, cy); !ok || cell != 3 {
		t.Errorf("inventoryPackCellAt with the doll switched off = (%d,%v), want (3,true) — "+
			"each box answers for its own switch and no other", cell, ok)
	}

	bare := inventoryTestApp(t).flow.viewer
	if cell, ok := bare.inventoryPackCellAt(cx, cy); ok {
		t.Errorf("a viewer holding no subject named cell %d, want none", cell)
	}
}

// SESS-INPUT-037's whole forward end strip scrolls the grid and raises no
// double-click state. The chosen point is outside the small triangle cue.
func TestAPressOnAScrollEndStripScrollsAndCountsNoClick(t *testing.T) {
	v, _ := openInventoryAt(t)
	bar, _, ok := v.packBar()
	if !ok {
		t.Fatal("setup: the viewer draws no pack bar")
	}
	// A pack longer than the bar, so the forward button has somewhere to go.
	_, cols, _ := v.packBar()
	pics := make([]*image.RGBA, cols+3)
	for i := range pics {
		pics[i] = solidPic(4, 4, color.RGBA{A: 0xff})
	}
	v.SetInventorySubject(packOf(5, pics...))

	_, fwd := packScrollRects(bar, cols)
	fx, fy := fwd.Max.X-1, fwd.Min.Y+1

	tapAt(v, fx, fy)
	if v.packScroll != 1 {
		t.Errorf("packScroll = %d after one press on the forward button, want 1", v.packScroll)
	}
	if v.invClickFrames != 0 {
		t.Errorf("invClickFrames = %d after a press on a scroll button, want 0 — a button is not a cell",
			v.invClickFrames)
	}
	tapAt(v, fx, fy)
	if v.packScroll != 2 {
		t.Errorf("packScroll = %d after two presses, want 2", v.packScroll)
	}
	if idx, ok := v.TakeInventoryEquip(); ok {
		t.Errorf("two presses on the scroll button raised equip request %d", idx)
	}
}

// TestADoubleClickWithinTheThresholdRaisesOneRequest is AC-7's first case:
// two presses on the same pack cell, spaced as far apart as the threshold
// still allows, raise exactly one request — and draining it a second time
// yields nothing, so a caller that asks twice cannot equip twice off one
// double-click.
func TestADoubleClickWithinTheThresholdRaisesOneRequest(t *testing.T) {
	v, _ := openInventoryAt(t)
	cx, cy := packCellCenter(t, v, 2)

	tapAt(v, cx, cy)
	for i := 0; i < InventoryDoubleClickFrames-2; i++ {
		idleAt(v, cx, cy)
	}
	tapAt(v, cx, cy)

	idx, ok := v.TakeInventoryEquip()
	if !ok || idx != 2 {
		t.Fatalf("TakeInventoryEquip = (%d,%v), want (2,true)", idx, ok)
	}
	if idx, ok := v.TakeInventoryEquip(); ok {
		t.Errorf("a second drain answered (%d,true), want nothing left to drain", idx)
	}
}

// TestADoubleClickOneFrameBeyondTheThresholdRaisesNone is AC-7's boundary:
// the same two presses, with exactly one more idle frame between them than
// the case above, raise nothing at all. Reverting InventoryDoubleClickFrames'
// own count in command.go's decrement (or the constant itself) reddens this
// on the drain, which is how the threshold is witnessed rather than assumed.
func TestADoubleClickOneFrameBeyondTheThresholdRaisesNone(t *testing.T) {
	v, _ := openInventoryAt(t)
	cx, cy := packCellCenter(t, v, 2)

	tapAt(v, cx, cy)
	for i := 0; i < InventoryDoubleClickFrames-1; i++ {
		idleAt(v, cx, cy)
	}
	tapAt(v, cx, cy)

	if idx, ok := v.TakeInventoryEquip(); ok {
		t.Errorf("TakeInventoryEquip = (%d,true), want nothing one frame beyond the threshold", idx)
	}
}

// TestADoubleClicksSecondPressStillEquipsAfterATremorWithinTheCell is
// counterexample E (round-2 adversarial review, fifth pass): the second
// press of a double-click is also a live drag candidate (dragFromPack, every
// pack press is), and TapSlop is a 4-pixel Manhattan threshold accumulated
// over the whole press, held against a cell 48 pixels wide (invCellSize) —
// a hand's own tremor over the SAME cell crosses it while the gesture never
// leaves the cell the double-click matched. Before this fix, command.go's
// release switch tried v.dragActive first: once the tremor armed the drag,
// the release fell into the drag-release resolution (dollBox or ground-drop,
// neither of which the release matches: still over the pack bar, not the
// doll, and still inside inventoryCaptures' own surface) and raised neither
// an equip nor a drop, losing the double-click outright — the shop's own
// dest == origin tremor guard (app.go), never applied to the mission map's
// own pack cell.
func TestADoubleClicksSecondPressStillEquipsAfterATremorWithinTheCell(t *testing.T) {
	v, _ := openInventoryAt(t)
	cx, cy := packCellCenter(t, v, 2)

	tapAt(v, cx, cy)
	idleAt(v, cx, cy)

	dollPress(v, cx, cy)
	dollMove(v, cx+3, cy+2)
	if !v.dragActive || v.dragCandKind != dragFromPack {
		t.Fatalf("setup: dragActive=%v dragCandKind=%v, want an armed pack drag before the release under test", v.dragActive, v.dragCandKind)
	}
	dollRelease(v, cx+1, cy+1)

	idx, ok := v.TakeInventoryEquip()
	if !ok || idx != 2 {
		t.Fatalf("TakeInventoryEquip = (%d,%v), want (2,true): a tremor within the matched cell must still equip", idx, ok)
	}
	if worn, dropIdx, x, y, ok := v.TakeInventoryDrop(); ok {
		t.Errorf("a tremor-only release also raised a ground drop (worn=%v idx=%d x=%d y=%d), want none", worn, dropIdx, x, y)
	}
}

// TestTwoPressesOnDifferentCellsRaiseNone is AC-7's third case: a press on
// one cell immediately followed by a press on another is two first clicks,
// never a double-click on either. The second press is proven to be a live
// first click of its own, and not simply swallowed, by following it with a
// genuine match on its own cell.
func TestTwoPressesOnDifferentCellsRaiseNone(t *testing.T) {
	v, _ := openInventoryAt(t)
	c0x, c0y := packCellCenter(t, v, 0)
	c1x, c1y := packCellCenter(t, v, 1)

	tapAt(v, c0x, c0y)
	tapAt(v, c1x, c1y)
	if idx, ok := v.TakeInventoryEquip(); ok {
		t.Fatalf("two presses on different cells raised (%d,true)", idx)
	}

	tapAt(v, c1x, c1y)
	if idx, ok := v.TakeInventoryEquip(); !ok || idx != 1 {
		t.Errorf("a press on cell 1 right after the mismatch = (%d,%v), want (1,true)", idx, ok)
	}
}

func TestADoubleClickOnTheFigureBoxRaisesNone(t *testing.T) {
	v, box := openInventoryAt(t)
	fx, fy := figureCenter(box)

	tapAt(v, fx, fy)
	tapAt(v, fx, fy)
	if idx, ok := v.TakeInventoryEquip(); ok {
		t.Errorf("a double-click on the figure box raised (%d,true)", idx)
	}
}
