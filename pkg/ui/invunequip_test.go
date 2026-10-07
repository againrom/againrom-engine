package ui

import (
	"image"
	"image/color"
	"testing"
)

// A double-click on a worn-box cell raises one one-shot unequip request
// (0151, defect 4) — invequip_test.go's own cases, restated over the worn
// box's own four fields (invWornClickCell, invWornClickFrames,
// invUnequipRequest) rather than the pack's three.
//
// EVERY FIXTURE IS itemPopupViewer's OWN (itempopup_test.go): the worn box
// and the pack bar both gate on inventoryEligible, which that helper already
// satisfies, so this file builds no second fixture for the same gate.

// wornCellCenter is the window-pixel point at the centre of worn cell i,
// built the same way packCellCenter (invequip_test.go) builds one for the
// pack: the box's own rectangle and wornSlotRects' own cells.
func wornCellCenter(t *testing.T, v *Viewer, i int) (int, int) {
	t.Helper()
	box, ok := v.wornBox()
	if !ok {
		t.Fatal("setup: the viewer draws no worn box")
	}
	return cellCenter(wornSlotRects()[i].Add(box.Min))
}

// unequipSubject is one InventorySubject with a picture in every worn cell
// this file presses on — inventoryWornSlotAt is pure geometry and asks
// nothing of Slots (its own doc), but a cell with no picture is still real
// geometry for wornSlotRects, so a subject carrying pictures is not load-
// bearing for these cases; it is here so a reader comparing this file
// against invequip_test.go's own packOf-built fixture is not left wondering
// why one box is bare.
func unequipSubject(id uint32) InventorySubject {
	var s InventorySubject
	s.ID = id
	for i := range s.Slots {
		s.Slots[i] = solidPic(4, 4, color.RGBA{A: 0xff})
	}
	return s
}

// TestADoubleClickOnAWornCellWithinTheThresholdRaisesOneRequest is AC-7's
// first case, run on the worn box: two presses on the same cell, spaced as
// far apart as the threshold still allows, raise exactly one request — and
// draining it a second time yields nothing.
func TestADoubleClickOnAWornCellWithinTheThresholdRaisesOneRequest(t *testing.T) {
	v := itemPopupViewer(t, unequipSubject(7))
	cx, cy := wornCellCenter(t, v, 2)

	tapAt(v, cx, cy)
	for i := 0; i < InventoryDoubleClickFrames-2; i++ {
		idleAt(v, cx, cy)
	}
	tapAt(v, cx, cy)

	idx, ok := v.TakeInventoryUnequip()
	if !ok || idx != 2 {
		t.Fatalf("TakeInventoryUnequip = (%d,%v), want (2,true)", idx, ok)
	}
	if idx, ok := v.TakeInventoryUnequip(); ok {
		t.Errorf("a second drain answered (%d,true), want nothing left to drain", idx)
	}
}

// TestADoubleClickOnAWornCellOneFrameBeyondTheThresholdRaisesNone is AC-7's
// boundary, run on the worn box: the same two presses, with one more idle
// frame between them, raise nothing.
func TestADoubleClickOnAWornCellOneFrameBeyondTheThresholdRaisesNone(t *testing.T) {
	v := itemPopupViewer(t, unequipSubject(7))
	cx, cy := wornCellCenter(t, v, 2)

	tapAt(v, cx, cy)
	for i := 0; i < InventoryDoubleClickFrames-1; i++ {
		idleAt(v, cx, cy)
	}
	tapAt(v, cx, cy)

	if idx, ok := v.TakeInventoryUnequip(); ok {
		t.Errorf("TakeInventoryUnequip = (%d,true), want nothing one frame beyond the threshold", idx)
	}
}

// TestTwoPressesOnDifferentWornCellsRaiseNone is AC-7's third case, run on
// the worn box: a press on one cell immediately followed by a press on
// another is two first clicks, never a double-click on either.
func TestTwoPressesOnDifferentWornCellsRaiseNone(t *testing.T) {
	v := itemPopupViewer(t, unequipSubject(7))
	c0x, c0y := wornCellCenter(t, v, 0)
	c1x, c1y := wornCellCenter(t, v, 1)

	tapAt(v, c0x, c0y)
	tapAt(v, c1x, c1y)
	if idx, ok := v.TakeInventoryUnequip(); ok {
		t.Fatalf("two presses on different worn cells raised (%d,true)", idx)
	}

	tapAt(v, c1x, c1y)
	if idx, ok := v.TakeInventoryUnequip(); !ok || idx != 1 {
		t.Errorf("a press on cell 1 right after the mismatch = (%d,%v), want (1,true)", idx, ok)
	}
}

// TestADoubleClickOnAnEmptyWornCellStillRaisesARequest pins the boundary
// named on inventoryWornSlotAt's own doc and on TakeInventoryUnequip's:
// this layer asks no question about what a cell holds, so a double-click on
// a cell carrying no picture raises the same one-shot request as one on an
// occupied cell — sim's own unequip is what refuses an empty slot
// (pkg/sim/equip.go), not this one.
func TestADoubleClickOnAnEmptyWornCellStillRaisesARequest(t *testing.T) {
	var s InventorySubject
	s.ID = 7 // every Slots entry left nil: every cell empty
	v := itemPopupViewer(t, s)
	cx, cy := wornCellCenter(t, v, 5)

	tapAt(v, cx, cy)
	tapAt(v, cx, cy)

	if idx, ok := v.TakeInventoryUnequip(); !ok || idx != 5 {
		t.Errorf("TakeInventoryUnequip = (%d,%v), want (5,true) — an empty cell is still a cell", idx, ok)
	}
}

// TestAWornClickAndAPackClickDoNotShareACount is the new property this
// story adds to the double-click machine: a press on one box must not
// extend or spend the count the OTHER box's own press started (viewer.go's
// own field doc). One first click on a pack cell, one first click on a worn
// cell, then the SECOND pack press within its own window still completes
// that double-click — which it could not if the worn press had spent or
// reset the pack's shared state.
func TestAWornClickAndAPackClickDoNotShareACount(t *testing.T) {
	pics := make([]*image.RGBA, 4)
	for i := range pics {
		pics[i] = solidPic(4, 4, color.RGBA{A: 0xff})
	}
	s := unequipSubject(7)
	s.Pack = pics
	s.PackCount = []uint32{1, 1, 1, 1}
	v := itemPopupViewer(t, s)

	px, py := packCellCenter(t, v, 1)
	wx, wy := wornCellCenter(t, v, 3)

	tapAt(v, px, py) // first click on pack cell 1
	tapAt(v, wx, wy) // an unrelated first click on worn cell 3, same frame's own next tap
	tapAt(v, px, py) // the pack cell's own second click, matching cell 1 again

	idx, ok := v.TakeInventoryEquip()
	if !ok || idx != 1 {
		t.Errorf("TakeInventoryEquip = (%d,%v), want (1,true) — the worn press must not have spent the pack's count", idx, ok)
	}
	if idx, ok := v.TakeInventoryUnequip(); ok {
		t.Errorf("TakeInventoryUnequip = (%d,true), want nothing — the worn cell saw one press, not two", idx)
	}
}
