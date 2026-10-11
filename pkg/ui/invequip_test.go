package ui

import (
	"image"
	"image/color"
	"testing"
)

func packCellCenter(t *testing.T, v *Viewer, cell int) (int, int) {
	t.Helper()
	bar, cols, ok := v.packBar()
	if !ok {
		t.Fatal("setup: the viewer draws no pack bar")
	}
	r := packCellRects(bar, cols)[cell]
	return (r.Min.X + r.Max.X) / 2, (r.Min.Y + r.Max.Y) / 2
}
func figureCenter(box image.Rectangle) (int, int) {
	r := characterPaneFigureRect(box)
	return (r.Min.X + r.Max.X) / 2, (r.Min.Y + r.Max.Y) / 2
}
func idleAt(v *Viewer, x, y int) {
	v.command(appInput{CursorX: x, CursorY: y})
}
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
func TestAPressOnAScrollEndStripScrollsAndCountsNoClick(t *testing.T) {
	v, _ := openInventoryAt(t)
	bar, _, ok := v.packBar()
	if !ok {
		t.Fatal("setup: the viewer draws no pack bar")
	}
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
	if v.invClickCell != 0 {
		t.Errorf("pack cell identity = %d after a scroll press, want none", v.invClickCell)
	}
	tapAt(v, fx, fy)
	if v.packScroll != 2 {
		t.Errorf("packScroll = %d after two presses, want 2", v.packScroll)
	}
	if idx, ok := v.TakeInventoryEquip(); ok {
		t.Errorf("two presses on the scroll button raised equip request %d", idx)
	}
}
func TestPackClassifiedDoublePressRaisesOneRequest(t *testing.T) {
	v, _ := openInventoryAt(t)
	cx, cy := packCellCenter(t, v, 2)

	tapAt(v, cx, cy)
	classifiedInventoryTap(v, cx, cy)

	idx, ok := v.TakeInventoryEquip()
	if !ok || idx != 2 {
		t.Fatalf("TakeInventoryEquip = (%d,%v), want (2,true)", idx, ok)
	}
	if idx, ok := v.TakeInventoryEquip(); ok {
		t.Errorf("a second drain answered (%d,true), want nothing left to drain", idx)
	}
}
func TestPackUnpairedPressesRaiseNone(t *testing.T) {
	v, _ := openInventoryAt(t)
	cx, cy := packCellCenter(t, v, 2)

	tapAt(v, cx, cy)
	tapAt(v, cx, cy)

	if idx, ok := v.TakeInventoryEquip(); ok {
		t.Errorf("TakeInventoryEquip = (%d,true), want nothing without a classified pair", idx)
	}
}
func TestADoubleClicksSecondPressStillEquipsAfterATremorWithinTheCell(t *testing.T) {
	v, _ := openInventoryAt(t)
	cx, cy := packCellCenter(t, v, 2)

	tapAt(v, cx, cy)
	idleAt(v, cx, cy)

	v.step(Input{PrimaryDown: true, CursorX: cx, CursorY: cy}, commandFrozen)
	v.command(appInput{PrimaryPressed: true, PrimaryDouble: true, CursorX: cx, CursorY: cy})
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
func TestTwoPressesOnDifferentCellsRaiseNone(t *testing.T) {
	v, _ := openInventoryAt(t)
	c0x, c0y := packCellCenter(t, v, 0)
	c1x, c1y := packCellCenter(t, v, 1)

	tapAt(v, c0x, c0y)
	tapAt(v, c1x, c1y)
	if idx, ok := v.TakeInventoryEquip(); ok {
		t.Fatalf("two presses on different cells raised (%d,true)", idx)
	}

	classifiedInventoryTap(v, c1x, c1y)
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

func classifiedInventoryTap(v *Viewer, x, y int) {
	v.command(appInput{PrimaryPressed: true, PrimaryReleased: true, PrimaryDouble: true, CursorX: x, CursorY: y})
}
