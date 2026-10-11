package ui

import (
	"image"
	"image/color"
	"testing"
)

func wornCellCenter(t *testing.T, v *Viewer, i int) (int, int) {
	t.Helper()
	box, ok := v.wornBox()
	if !ok {
		t.Fatal("setup: the viewer draws no worn box")
	}
	return cellCenter(wornSlotRects()[i].Add(box.Min))
}
func unequipSubject(id uint32) InventorySubject {
	var s InventorySubject
	s.ID = id
	for i := range s.Slots {
		s.Slots[i] = solidPic(4, 4, color.RGBA{A: 0xff})
	}
	return s
}
func TestWornClassifiedDoublePressRaisesOneRequest(t *testing.T) {
	v := itemPopupViewer(t, unequipSubject(7))
	cx, cy := wornCellCenter(t, v, 2)

	tapAt(v, cx, cy)
	classifiedInventoryTap(v, cx, cy)

	idx, ok := v.TakeInventoryUnequip()
	if !ok || idx != 2 {
		t.Fatalf("TakeInventoryUnequip = (%d,%v), want (2,true)", idx, ok)
	}
	if idx, ok := v.TakeInventoryUnequip(); ok {
		t.Errorf("a second drain answered (%d,true), want nothing left to drain", idx)
	}
}
func TestWornUnpairedPressesRaiseNone(t *testing.T) {
	v := itemPopupViewer(t, unequipSubject(7))
	cx, cy := wornCellCenter(t, v, 2)

	tapAt(v, cx, cy)
	tapAt(v, cx, cy)

	if idx, ok := v.TakeInventoryUnequip(); ok {
		t.Errorf("TakeInventoryUnequip = (%d,true), want nothing without a classified pair", idx)
	}
}
func TestTwoPressesOnDifferentWornCellsRaiseNone(t *testing.T) {
	v := itemPopupViewer(t, unequipSubject(7))
	c0x, c0y := wornCellCenter(t, v, 0)
	c1x, c1y := wornCellCenter(t, v, 1)

	tapAt(v, c0x, c0y)
	tapAt(v, c1x, c1y)
	if idx, ok := v.TakeInventoryUnequip(); ok {
		t.Fatalf("two presses on different worn cells raised (%d,true)", idx)
	}

	classifiedInventoryTap(v, c1x, c1y)
	if idx, ok := v.TakeInventoryUnequip(); !ok || idx != 1 {
		t.Errorf("a press on cell 1 right after the mismatch = (%d,%v), want (1,true)", idx, ok)
	}
}
func TestADoubleClickOnAnEmptyWornCellStillRaisesARequest(t *testing.T) {
	var s InventorySubject
	s.ID = 7 // every Slots entry left nil: every cell empty
	v := itemPopupViewer(t, s)
	cx, cy := wornCellCenter(t, v, 5)

	tapAt(v, cx, cy)
	classifiedInventoryTap(v, cx, cy)

	if idx, ok := v.TakeInventoryUnequip(); !ok || idx != 5 {
		t.Errorf("TakeInventoryUnequip = (%d,%v), want (5,true) — an empty cell is still a cell", idx, ok)
	}
}
func TestWornAndPackPressesClearEachOthersCellIdentity(t *testing.T) {
	s := unequipSubject(7)
	s.Pack = []*image.RGBA{solidPic(4, 4, color.RGBA{A: 255}), solidPic(4, 4, color.RGBA{A: 255})}
	s.PackCount = []uint32{1, 1}
	v := itemPopupViewer(t, s)
	px, py := packCellCenter(t, v, 1)
	wx, wy := wornCellCenter(t, v, 3)
	tapAt(v, px, py)
	tapAt(v, wx, wy)
	classifiedInventoryTap(v, px, py)
	if idx, ok := v.TakeInventoryEquip(); ok {
		t.Fatalf("pack press paired across a worn press: %d", idx)
	}
	if idx, ok := v.TakeInventoryUnequip(); ok {
		t.Fatalf("single worn press raised request %d", idx)
	}
	classifiedInventoryTap(v, px, py)
	if idx, ok := v.TakeInventoryEquip(); !ok || idx != 1 {
		t.Fatalf("fresh pack pair = (%d,%v), want (1,true)", idx, ok)
	}
}
