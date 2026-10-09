package ui

import (
	"image"
	"image/color"
	"slices"
	"testing"
)

// The item popup (0151, defect 5). EVERY FIXTURE HERE IS SYNTHETIC (AGENTS.md
// rule 2): messageViewer (messageline_test.go) supplies the lit, fonted viewer, and
// itemPopupViewer below widens it with the one selected entity wornBox and
// packBar both require (inventoryEligible).

// itemPopupViewer is a lit, fonted viewer holding s as its subject, with s's
// own id selected and present — wornBox's and packBar's own gate
// (inventoryEligible) — the same setup
// TestThePackBarsHitTestNamesTheElementUnderTheCursor (inventory_test.go)
// builds by hand, factored out here for this file's own repeated use.
func itemPopupViewer(t *testing.T, s InventorySubject) *Viewer {
	t.Helper()
	v := messageViewer(t)
	layoutViewport(v, 1200, wornBoxFixtureH)
	v.SetEntities([]MapEntity{{ID: s.ID, Cell: image.Pt(1, 1)}})
	v.sel = selection{s.ID}
	v.SetInventorySubject(s)
	return v
}

// cellCenter is the window pixel at the centre of r.
func cellCenter(r image.Rectangle) (int, int) {
	return (r.Min.X + r.Max.X) / 2, (r.Min.Y + r.Max.Y) / 2
}

// hoveredItemInfoAt over slot 0's own cell answers that slot's SlotInfo, and
// over an occupied slot with no SlotInfo entry — an empty (nil) one — or
// with no picture at all, answers false: an empty entry is the only signal
// this package has that a cell holds nothing to show a popup over.
func TestHoveredItemInfoAtNamesTheWornCellUnderTheCursor(t *testing.T) {
	var s InventorySubject
	s.ID = 7
	s.Slots[0] = solidPic(4, 4, color.RGBA{A: 0xff})
	s.SlotInfo[0] = []string{"Sword", "Damage 5 + 3"}
	v := itemPopupViewer(t, s)

	box, ok := v.wornBox()
	if !ok {
		t.Fatal("setup: no worn box drawn")
	}
	x, y := cellCenter(wornSlotRects()[0].Add(box.Min))

	lines, ok := v.hoveredItemInfoAt(x, y)
	want := []string{"Sword", "Damage 5 + 3"}
	if !ok || !slices.Equal(lines, want) {
		t.Errorf("hoveredItemInfoAt(slot 0) = %v, %v, want %v, true", lines, ok, want)
	}
}

// The UI carries the presenter's effective staff interval byte-for-byte. It
// neither substitutes the weapon row's physical 0-0 nor reformats the live
// release range before drawing the popup.
func TestHoveredItemInfoAtKeepsAStaffSpellDamageInterval(t *testing.T) {
	var s InventorySubject
	s.ID = 7
	s.Slots[0] = solidPic(4, 4, color.RGBA{A: 0xff})
	s.SlotInfo[0] = []string{"Wood Staff", "Magic", "Casts Fire Arrow", "Damage 5-10", "Range 5"}
	v := itemPopupViewer(t, s)

	box, ok := v.wornBox()
	if !ok {
		t.Fatal("setup: no worn box drawn")
	}
	x, y := cellCenter(wornSlotRects()[0].Add(box.Min))
	got, ok := v.hoveredItemInfoAt(x, y)
	if !ok || !slices.Equal(got, s.SlotInfo[0]) {
		t.Fatalf("hovered staff tooltip = %v, %v; want %v, true", got, ok, s.SlotInfo[0])
	}
	if slices.Contains(got, "Damage 0-0") {
		t.Fatalf("hovered staff tooltip regressed to physical damage: %v", got)
	}
}

func TestHoveredItemInfoAtIsFalseOverAnEmptyWornCell(t *testing.T) {
	var s InventorySubject
	s.ID = 7
	v := itemPopupViewer(t, s)

	box, ok := v.wornBox()
	if !ok {
		t.Fatal("setup: no worn box drawn")
	}
	x, y := cellCenter(wornSlotRects()[0].Add(box.Min))

	if lines, ok := v.hoveredItemInfoAt(x, y); ok {
		t.Errorf("hoveredItemInfoAt over an empty slot = %v, true; want false", lines)
	}
}

// The pack bar's own half of the same test, over PackInfo instead of
// SlotInfo — inventoryPackCellAt's own geometry, reused rather than a
// second hit test.
func TestHoveredItemInfoAtNamesThePackCellUnderTheCursor(t *testing.T) {
	pic := solidPic(4, 4, color.RGBA{A: 0xff})
	s := InventorySubject{
		ID: 7, Pack: []*image.RGBA{pic}, PackCount: []uint32{1},
		PackInfo: [][]string{{"Arrow", "Damage 1 + 1"}},
	}
	v := itemPopupViewer(t, s)

	bar, cols, ok := v.packBar()
	if !ok {
		t.Fatal("setup: no pack bar drawn")
	}
	x, y := cellCenter(packCellRects(bar, cols)[0])

	lines, ok := v.hoveredItemInfoAt(x, y)
	want := []string{"Arrow", "Damage 1 + 1"}
	if !ok || !slices.Equal(lines, want) {
		t.Errorf("hoveredItemInfoAt(pack 0) = %v, %v, want %v, true", lines, ok, want)
	}
}

// itemPopupPresent answers false before any cursor has ever been observed —
// hasCursor's own gate, the same one the readout carries (readout.go).
func TestItemPopupPresentIsFalseWithNoCursorObserved(t *testing.T) {
	var s InventorySubject
	s.ID = 7
	s.SlotInfo[0] = []string{"Sword"}
	v := itemPopupViewer(t, s)

	if _, _, ok := v.itemPopupPresent(); ok {
		t.Error("itemPopupPresent answered true before any cursor was observed")
	}
}

// itemPopupPresent draws a picture above the cursor once the
// cursor stands over a cell carrying lines — step's own field, cursorX/
// cursorY/hasCursor, set the one way a test can set it (viewer.go).
func TestItemPopupPresentDrawsOverAHoveredWornCell(t *testing.T) {
	var s InventorySubject
	s.ID = 7
	s.SlotInfo[0] = []string{"Sword", "Damage 5 + 3"}
	v := itemPopupViewer(t, s)

	box, ok := v.wornBox()
	if !ok {
		t.Fatal("setup: no worn box drawn")
	}
	x, y := cellCenter(wornSlotRects()[0].Add(box.Min))
	v.step(Input{CursorX: x, CursorY: y}, commandFrozen)

	pic, at, ok := v.itemPopupPresent()
	if !ok || pic == nil {
		t.Fatal("itemPopupPresent answered false over a slot carrying info")
	}
	if at.X != x || at.Y+pic.Bounds().Dy() != y {
		t.Errorf("popup at %v size%v, want lower-left at cursor (%d,%d)", at, pic.Bounds().Size(), x, y)
	}
}

// The item popup is the shared hover box (MENU-128): its interior is the
// hover fill and its outer bevel row the light gold.
func TestItemPopupIsTheHoverBox(t *testing.T) {
	pic := composeHoverBox([]string{"Sword"}, messageFont(), nil)
	if pic == nil {
		t.Fatal("composeHoverBox returned nil")
	}
	if got := pic.RGBAAt(3, 1); got != hoverLight {
		t.Errorf("bevel pixel = %v, want %v", got, hoverLight)
	}
	if got := pic.RGBAAt(3, 3); got != hoverFill {
		t.Errorf("interior pixel = %v, want %v", got, hoverFill)
	}
}

// A cursor over no occupied cell at all draws nothing, whatever SlotInfo the
// subject otherwise carries — the same "an empty entry is an empty cell"
// rule proven above, at the presentation layer.
func TestItemPopupPresentIsFalseOverAnUnoccupiedCell(t *testing.T) {
	var s InventorySubject
	s.ID = 7
	s.SlotInfo[0] = []string{"Sword"}
	v := itemPopupViewer(t, s)

	box, ok := v.wornBox()
	if !ok {
		t.Fatal("setup: no worn box drawn")
	}
	// Slot index 1 (the SECOND cell) carries no SlotInfo entry.
	x, y := cellCenter(wornSlotRects()[1].Add(box.Min))
	v.step(Input{CursorX: x, CursorY: y}, commandFrozen)

	if _, _, ok := v.itemPopupPresent(); ok {
		t.Error("itemPopupPresent answered true over a slot with no SlotInfo entry")
	}
}
