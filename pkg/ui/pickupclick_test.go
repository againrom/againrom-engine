package ui

import (
	"image"
	"testing"
)

// The hero and the sack. THE TWO ARE NOT ON THE SAME CELL and are not adjacent:
// the defect this file witnesses is a click on a DISTANT sack, which is the
// case the owner reported against his own build of round 3 -- the cursor
// changed and the click did nothing at all, neither walking nor taking.
const (
	pcHeroID, pcHeroCol, pcHeroRow = 21, 3, 3
	pcSackCol, pcSackRow           = 9, 7
)

// pcEntities is one player character and nothing else, because `pickupGate`
// (`AI-CURSOR-242`) requires exactly one selected object and that object's own
// player-character flag.
func pcEntities() []MapEntity {
	return []MapEntity{{
		ID: pcHeroID, Cell: image.Pt(pcHeroCol, pcHeroRow), Life: LifeAlive,
		HP: 100, MaxHP: 100, PlayerCharacter: true,
	}}
}

// pcOnMap parks an App on the map with that hero selected and one sack on the
// ground, and returns the window point of the sack's own cell.
func pcOnMap(t *testing.T) (*App, *Viewer, *mapSeam, int, int) {
	t.Helper()
	a, v, seam := atOnMap(t)
	v.SetEntities(pcEntities())
	v.SetSacks([]MapSack{{Cell: image.Pt(pcSackCol, pcSackRow)}})
	atPress(a, v, pcHeroCol, pcHeroRow)
	if id, ok := v.SelectedUnit(); !ok || id != pcHeroID {
		t.Fatalf("setup: SelectedUnit() = %v,%v, want %d,true", id, ok, pcHeroID)
	}
	x, y := cellPoint(v, pcSackCol, pcSackRow)
	return a, v, seam, x, y
}

// TestTheMapPickupCursorAppearsOverADistantSack is the promise half of the
// defect: the cursor `AI-CURSOR-242` gates carries NO proximity term, so it
// appears over any sack the hover mask reports, at any distance. Read whole,
// that claim's `[EBP-0x68]` is exactly one selected `CUnit`, that object's own
// `+0x18c` bit `0x1`, and then either the hover mask equal to `0x40` or the hit
// object being the selection itself with `0x40` set. There is no distance term
// to drop, so the cursor is not the side of this defect that was wrong.
func TestTheMapPickupCursorAppearsOverADistantSack(t *testing.T) {
	_, v, _, x, y := pcOnMap(t)
	v.cursorX, v.cursorY, v.hasCursor = x, y, true
	if got := v.mapCursorName(); got != "pickup" {
		t.Fatalf("mapCursorName() over a sack %d cells away = %q, want pickup",
			pcSackCol-pcHeroCol, got)
	}
}

// TestAClickUnderThePickupCursorReachesTheSeamWithTheUnitAndTheSacksCell is the
// performance half, and it is the assertion round 3 did not have.
//
// `AI-CLICK-050` gives a click under `pickup` order-space opcode `0x21`, and
// `ITEM-PICK-016` reads that arm whole: it names an ARBITRARY cell taken from
// the command's own `cmd+0x0a`/`cmd+0x0c`, and it walks the ORDERED ACTOR
// there. So the seam must receive the unit the cursor's gate named and the
// sack's own cell.
//
// THROUGH ROUND 3 IT RECEIVED NEITHER. MapGrab took no arguments at all, so the
// far side answered "which character" for the inventory window's subject at
// that subject's own cell -- which for a distant sack is a cell with no sack on
// it, and the click was silent. `DIV-292` was the row and this is its close.
func TestAClickUnderThePickupCursorReachesTheSeamWithTheUnitAndTheSacksCell(t *testing.T) {
	a, v, seam, x, y := pcOnMap(t)
	v.cursorX, v.cursorY, v.hasCursor = x, y, true
	if got := v.mapCursorName(); got != "pickup" {
		t.Fatalf("setup: mapCursorName() = %q, want pickup", got)
	}
	before := len(seam.orders)

	atTapPoint(a, x, y)

	if len(seam.grabs) != 1 {
		t.Fatalf("MapGrab calls = %d (%+v), want exactly 1", len(seam.grabs), seam.grabs)
	}
	want := grabbed{entity: pcHeroID, x: pcSackCol, y: pcSackRow, aimed: true}
	if seam.grabs[0] != want {
		t.Errorf("MapGrab received %+v, want %+v", seam.grabs[0], want)
	}
	// The order is ONE order and not a move beside it: `AI-CLICK-050` gives the
	// arm one opcode, and the walk is `0x21`'s own, issued by the far side.
	if got := len(seam.orders) - before; got != 0 {
		t.Errorf("MapOrder calls = %d, want 0 -- the walk is the pick-up order's own, not a second order from this tier", got)
	}
}

// TestAClickUnderThePickupCursorPastTheMapExtentOrdersNothing is the extent
// test every other cell-naming arm of `decide` already applies, asked of this
// one. A press outside what groundAt resolves names no cell, so there is no
// cell to hand the seam and nothing is issued.
func TestAClickUnderThePickupCursorPastTheMapExtentOrdersNothing(t *testing.T) {
	a, v, seam, _, _ := pcOnMap(t)
	// A point on the map surface whose cell lies past the grid: the camera is
	// at the origin and the fixture grid is 60x60, so a window point far below
	// the last row resolves to no cell.
	x, y := cellPoint(v, pcSackCol, pcSackRow)
	g := gesture{tap: true, x: x, y: y, cursor: "pickup"}
	nowhere := func(px, py float64) (col, row int, inside bool) { return 0, 0, false }
	_, _, ords, ok := decide(selection{pcHeroID}, pcEntities(), nowhere, v.entityPickRect, 0, g)
	if ok || len(ords) != 0 {
		t.Fatalf("decide with no resolvable cell = %+v,%v, want no orders, false", ords, ok)
	}
	_ = a
	_ = seam
}
