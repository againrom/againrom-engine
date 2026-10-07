package ui

import (
	"image"
	"image/color"
	"testing"
)

// The doll figure's own interactivity (1005, "the interactive doll", round
// 1): the per-pixel hit test, the hover popup, the tap-to-unequip gesture
// and the drag machine between the pack bar and the doll. Every fixture
// below is synthetic (AGENTS.md rule 2) and built on openInventoryAt's own
// two-entity map (invclick_test.go), widened with a hand-built figure and
// mask so this file addresses the doll box by the SAME geometry
// dollFigureSlotAt itself reads, never a second guess at it.

// dollMaskClearY is how far down the figure crop this file's masks start. 36
// clears it with two rows to spare, so every case below presses on a pixel
// the corners do not share.
const dollMaskClearY = 36

// dollMaskFixture is a 4-wide SlotMask whose marked block is the four rows at
// dollMaskClearY, small enough to address every case in this file by an exact
// mask pixel: the left half names slot 1 (mask value 1, dollFigureSlotAt's own
// zero-based 0), the top-right quadrant names slot 2 (mask value 2, zero-based
// 1), and the bottom-right quadrant names no slot at all — the base body or
// the ground around the figure, on SlotMask.At's own "zero is absent" rule.
func dollMaskFixture() *SlotMask {
	return dollMaskAt(dollMaskClearY, []uint8{
		1, 1, 2, 2,
		1, 1, 2, 2,
		1, 1, 0, 0,
		1, 1, 0, 0,
	})
}

// dollMaskAt is a 4-wide SlotMask whose rows are blank down to y and then the
// block given, so every fixture in this file names the same pixels.
func dollMaskAt(y int, block []uint8) *SlotMask {
	m := &SlotMask{W: 4, H: y + len(block)/4, Slot: make([]uint8, 4*(y+len(block)/4))}
	copy(m.Slot[y*4:], block)
	return m
}

// openDollAt is openInventoryAt's own fixture, widened with a composed
// figure and its mask: dollFigureSlotAt answers false for every point over a
// subject with no mask at all (its own doc), so a file exercising the mask
// needs a subject that carries one.
func openDollAt(t *testing.T) (*Viewer, image.Rectangle) {
	t.Helper()
	v, box := openInventoryAt(t)
	sub := v.invSubject
	sub.Figure = solidPic(4, 4, color.RGBA{R: 0xff, A: 0xff})
	sub.SlotMask = dollMaskFixture()
	sub.SlotInfo[0] = []string{"slot one"}
	sub.SlotInfo[1] = []string{"slot two"}
	v.SetInventorySubject(sub)
	return v, box
}

// dollMaskPoint is the window pixel standing over the marked block's own
// coordinate (mx, my) — the character pane's figure origin (its own top-left
// corner, two rows down) plus dollMaskClearY, restated here rather than read
// back off the function under test, packCellCenter's own convention one box
// over.
func dollMaskPoint(box image.Rectangle, mx, my int) (int, int) {
	return box.Min.X + mx, box.Min.Y + 2 + dollMaskClearY + my
}

// dollPress, dollMove and dollRelease drive one frame each of a left
// gesture through the SHIPPED camera step, marqueePress's own shape: the
// accumulator, the cursor and the "is a button held" latch dragIntent
// writes are the ones this file's drag cases are judged by, and command.go
// itself never touches any of the three.
func dollPress(v *Viewer, x, y int) {
	v.step(Input{PrimaryDown: true, CursorX: x, CursorY: y}, commandFrozen)
	v.command(appInput{PrimaryPressed: true, CursorX: x, CursorY: y})
}

func dollMove(v *Viewer, x, y int) {
	v.step(Input{PrimaryDown: true, CursorX: x, CursorY: y}, commandFrozen)
	v.command(appInput{CursorX: x, CursorY: y})
}

func dollRelease(v *Viewer, x, y int) {
	v.step(Input{CursorX: x, CursorY: y}, commandFrozen)
	v.command(appInput{PrimaryReleased: true, CursorX: x, CursorY: y})
}

// TestDollFigureSlotAtReadsTheMask is item 1's own hit test: a point over
// slot 1's own mask pixels answers 0 (zero-based), a point over slot 2's
// answers 1, and a point over the unmarked quadrant answers no slot at all.
func TestDollFigureSlotAtReadsTheMask(t *testing.T) {
	v, box := openDollAt(t)

	x1, y1 := dollMaskPoint(box, 0, 0)
	if n, ok := v.dollFigureSlotAt(x1, y1); !ok || n != 0 {
		t.Errorf("dollFigureSlotAt over slot 1's own pixels = (%d,%v), want (0,true)", n, ok)
	}

	x2, y2 := dollMaskPoint(box, 2, 0)
	if n, ok := v.dollFigureSlotAt(x2, y2); !ok || n != 1 {
		t.Errorf("dollFigureSlotAt over slot 2's own pixels = (%d,%v), want (1,true)", n, ok)
	}

	xb, yb := dollMaskPoint(box, 2, 2)
	if n, ok := v.dollFigureSlotAt(xb, yb); ok {
		t.Errorf("dollFigureSlotAt over the unmarked quadrant = (%d,true), want no slot", n)
	}

	v.sel = selection{6}
	if n, ok := v.dollFigureSlotAt(x1, y1); ok {
		t.Errorf("dollFigureSlotAt with the selection moved to a different unit = (%d,true), want none", n)
	}
}

func TestDollFigureSlotAtAnswersUnderAMultiUnitSelectionLedByTheSubject(t *testing.T) {
	v, box := openDollAt(t)
	v.sel = selection{5, 6}

	x1, y1 := dollMaskPoint(box, 0, 0)
	if n, ok := v.dollFigureSlotAt(x1, y1); !ok || n != 0 {
		t.Fatalf("dollFigureSlotAt over slot 1 under a two-unit selection led by the subject = (%d,%v), want (0,true)", n, ok)
	}

	lines, ok := v.hoveredItemInfoAt(x1, y1)
	if !ok || len(lines) != 1 || lines[0] != "slot one" {
		t.Errorf("hoveredItemInfoAt over slot 1 under the same selection = (%v,%v), want ([slot one],true)", lines, ok)
	}

	// THE SELECTION LED BY A DIFFERENT UNIT STILL ANSWERS NOTHING: entity 6
	// carries no composed figure of its own, so dollSubject's arm 1 does not
	// apply and the fix must not have widened the gate into "any selection
	// containing the subject" by mistake.
	v.sel = selection{6, 5}
	if n, ok := v.dollFigureSlotAt(x1, y1); ok {
		t.Errorf("dollFigureSlotAt under a selection NOT led by the subject = (%d,true), want none — entity 6 draws no composed figure", n)
	}

	// THE PRESS ITSELF, DRIVEN THROUGH THE SHIPPED GESTURE (command.go's own
	// drag machine), under the leading-subject selection restored: a tap on
	// the doll's own slot-1 pixels still raises the one-shot unequip request,
	// exactly as TestPressOnADollSlotWithoutSlopUnequips already proves for a
	// single-unit selection.
	v.sel = selection{5, 6}
	tapAt(v, x1, y1)
	if idx, ok := v.TakeInventoryDollUnequip(); !ok || idx != 0 {
		t.Errorf("TakeInventoryDollUnequip under a two-unit selection = (%d,%v), want (0,true)", idx, ok)
	}
}

// TestHoveredItemInfoAtAsksTheDollBox is item 2: hoveredItemInfoAt shows
// SlotInfo's own line for a point over an occupied slot's mask pixels, and
// nothing for a point over the figure's own unmarked ground — itempopup.go's
// own comment on why the doll is asked second, between the worn box and the
// pack bar.
func TestHoveredItemInfoAtAsksTheDollBox(t *testing.T) {
	v, box := openDollAt(t)

	x1, y1 := dollMaskPoint(box, 0, 0)
	lines, ok := v.hoveredItemInfoAt(x1, y1)
	if !ok || len(lines) != 1 || lines[0] != "slot one" {
		t.Errorf("hoveredItemInfoAt over slot 1 = (%v,%v), want ([slot one],true)", lines, ok)
	}

	xb, yb := dollMaskPoint(box, 2, 2)
	if lines, ok := v.hoveredItemInfoAt(xb, yb); ok {
		t.Errorf("hoveredItemInfoAt over the unmarked ground = (%v,true), want nothing", lines)
	}
}

// TestPressOnADollSlotWithoutSlopUnequips is item 3: a press released on the
// SAME slot it began on, with no travel at all — tapAt's own "one frame,
// zero pixels" shape — raises TakeInventoryDollUnequip's own one-shot
// request, zero-based exactly as dollFigureSlotAt itself answers.
func TestPressOnADollSlotWithoutSlopUnequips(t *testing.T) {
	v, box := openDollAt(t)
	x1, y1 := dollMaskPoint(box, 0, 0)

	tapAt(v, x1, y1)

	idx, ok := v.TakeInventoryDollUnequip()
	if !ok || idx != 0 {
		t.Fatalf("TakeInventoryDollUnequip = (%d,%v), want (0,true)", idx, ok)
	}
	if idx, ok := v.TakeInventoryEquip(); ok {
		t.Errorf("a tap on the doll also raised TakeInventoryEquip(%d), want none", idx)
	}
	if v.dragActive {
		t.Error("dragActive is true after a tap that never crossed TapSlop")
	}
}

// TestDollTapStillUnequipsAfterATremorWithinTheSlot is counterexample F
// (round-2 adversarial review, fifth pass): the same tremor hole as
// counterexample E, on the doll's own single-tap gesture rather than the
// pack's double-click, and pre-existing on master rather than introduced by
// this pass — undisclosed until now because the shop's own dest == origin
// tremor guard (app.go) was never compared against this file's own doll
// case. dollMaskFixture's own slots are 2 real pixels wide, too small to
// wobble inside past TapSlop without leaving the mask entirely, so this test
// builds its own larger single-slot mask (20x20, one slot, centred in the
// figure area) to give a press room to move and still land back on the same
// slot's own pixels.
func TestDollTapStillUnequipsAfterATremorWithinTheSlot(t *testing.T) {
	v, box := openDollAt(t)
	mask := &SlotMask{W: 20, H: dollMaskClearY + 20, Slot: make([]uint8, 20*(dollMaskClearY+20))}
	for i := dollMaskClearY * 20; i < len(mask.Slot); i++ {
		mask.Slot[i] = 1
	}
	sub := v.invSubject
	sub.SlotMask = mask
	v.SetInventorySubject(sub)

	x1, y1 := box.Min.X+10, box.Min.Y+2+dollMaskClearY+10

	dollPress(v, x1, y1)
	dollMove(v, x1+3, y1+2)
	if !v.dragActive || v.dragCandKind != dragFromDoll {
		t.Fatalf("setup: dragActive=%v dragCandKind=%v, want an armed doll drag before the release under test", v.dragActive, v.dragCandKind)
	}
	// THE PRODUCTION PUSH, refreshDollDrag's own act (pkg/game/world.go):
	// slot 1 (one-based, the only slot this fixture has) suppressed, with
	// its own code cleared from the mask — suppressedFigureMask is all
	// zero, so dollFigureSlotAt reading it answers no slot anywhere on the
	// figure for as long as this suppression stands, exactly as it would in
	// the running game.
	suppressedFigureMask := &SlotMask{W: 20, H: dollMaskClearY + 20, Slot: make([]uint8, 20*(dollMaskClearY+20))}
	suppressedFigurePic := solidPic(20, 20, color.RGBA{A: 0xff})
	v.SetDollSuppressedFigure(sub.ID, 1, suppressedFigurePic, suppressedFigureMask)
	if slot, ok := v.dollFigureSlotAt(x1, y1); ok {
		t.Fatalf("setup: dollFigureSlotAt(origin) = (%d,true) AFTER pushing the suppression, want no slot: "+
			"the suppressed mask was not actually installed", slot)
	}

	dollRelease(v, x1+1, y1+1)

	idx, ok := v.TakeInventoryDollUnequip()
	if !ok || idx != 0 {
		t.Fatalf("TakeInventoryDollUnequip = (%d,%v), want (0,true): a tremor within the same slot must still unequip", idx, ok)
	}
	if worn, dropIdx, x, y, ok := v.TakeInventoryDrop(); ok {
		t.Errorf("a tremor-only release also raised a ground drop (worn=%v idx=%d x=%d y=%d), want none", worn, dropIdx, x, y)
	}
}

// TestDragPackCellToDollEquips is item 4's first direction: a press on a
// pack cell, carried past TapSlop and released anywhere inside the doll
// box, raises the SAME one-shot request a double-click on that cell already
// does (TakeInventoryEquip) — equipFromPack's existing path, not a second
// one.
func TestDragPackCellToDollEquips(t *testing.T) {
	v, box := openDollAt(t)
	cx, cy := packCellCenter(t, v, 1)
	rx, ry := (box.Min.X+box.Max.X)/2, (box.Min.Y+box.Max.Y)/2

	dollPress(v, cx, cy)
	dollMove(v, cx+3*TapSlop, cy)
	if !v.dragActive {
		t.Fatal("setup: dragActive is false after travelling well past TapSlop")
	}
	dollRelease(v, rx, ry)

	idx, ok := v.TakeInventoryEquip()
	if !ok || idx != 1 {
		t.Fatalf("TakeInventoryEquip = (%d,%v), want (1,true)", idx, ok)
	}
	if v.dragActive {
		t.Error("dragActive is still true after the release")
	}
}

// TestDragDollSlotToPackUnequips is item 4's other direction: a press on a
// doll slot, carried past TapSlop and released anywhere inside the pack
// bar, raises TakeInventoryDollUnequip — the same request a tap-in-place
// raises, drained by pkg/game's unequipFromDoll.
func TestDragDollSlotToPackUnequips(t *testing.T) {
	v, box := openDollAt(t)
	sx, sy := dollMaskPoint(box, 0, 0)
	px, py := packCellCenter(t, v, 2)

	dollPress(v, sx, sy)
	dollMove(v, sx+3*TapSlop, sy)
	if !v.dragActive {
		t.Fatal("setup: dragActive is false after travelling well past TapSlop")
	}
	dollRelease(v, px, py)

	idx, ok := v.TakeInventoryDollUnequip()
	if !ok || idx != 0 {
		t.Fatalf("TakeInventoryDollUnequip = (%d,%v), want (0,true)", idx, ok)
	}
}

// TestDragFromPackReleasedOutsideEveryBoxDropsToTheGround is round 2's own
// closing case for item 4's pack origin (`ITEM-DROP-008`, `DIV-088`): a
// release on neither the doll box, the worn box nor the pack bar —
// inventoryCaptures' own single surface — raises the ground-drop request
// instead of returning silently to origin, which is what this exact
// gesture did before round 2 existed (TestDragReleasedElsewhereReturnsToOrigin,
// round 1). worn is false and idx is the pack element the drag carried,
// dragCandIdx's own CarriedStacks ordering.
func TestDragFromPackReleasedOutsideEveryBoxDropsToTheGround(t *testing.T) {
	v, _ := openDollAt(t)
	cx, cy := packCellCenter(t, v, 1)

	dollPress(v, cx, cy)
	dollMove(v, cx+3*TapSlop, cy)
	dollRelease(v, 0, 0) // window origin: outside every inventory box

	worn, idx, _, _, ok := v.TakeInventoryDrop()
	if !ok || worn || idx != 1 {
		t.Fatalf("TakeInventoryDrop = (worn=%v,idx=%d,ok=%v), want (worn=false,idx=1,ok=true)", worn, idx, ok)
	}
	if idx, ok := v.TakeInventoryEquip(); ok {
		t.Errorf("the same release also raised TakeInventoryEquip(%d), want none", idx)
	}
	if v.dragActive || v.dragCandKind != dragNone {
		t.Errorf("dragActive=%v dragCandKind=%v after release, want the gesture fully cleared", v.dragActive, v.dragCandKind)
	}
}

// TestDragFromDollReleasedOutsideEveryBoxDropsToTheGround is the same
// closing case for item 4's doll origin: worn is true and idx is the
// equipment slot the drag lifted, dollFigureSlotAt's own zero-based
// numbering.
func TestDragFromDollReleasedOutsideEveryBoxDropsToTheGround(t *testing.T) {
	v, _ := openDollAt(t)
	sx, sy := dollMaskPoint(figureBoxFor(t, v), 2, 0) // slot 2's own pixels

	dollPress(v, sx, sy)
	dollMove(v, sx+3*TapSlop, sy)
	dollRelease(v, 0, 0)

	worn, idx, _, _, ok := v.TakeInventoryDrop()
	if !ok || !worn || idx != 1 {
		t.Fatalf("TakeInventoryDrop = (worn=%v,idx=%d,ok=%v), want (worn=true,idx=1,ok=true)", worn, idx, ok)
	}
	if idx, ok := v.TakeInventoryDollUnequip(); ok {
		t.Errorf("the same release also raised TakeInventoryDollUnequip(%d), want none", idx)
	}
}

// TestGroundDropRefusesTheSpellbookStripDuringAnArmedDrag is round-2's own
// tenth-pass adversarial review, "the two shipped witnesses cannot see the
// production state" — an unproven observation the review named and this
// test settles: measured directly, before this pass's fix, a release over
// the spellbook strip during an armed pack-origin drag raised
// TakeInventoryDrop = (worn=false idx=1 x=1 y=25 ok=true), a real cell and
// not the off-map sentinel, planting a sack there.
//
// THE MECHANISM: the inventory's own swallowed block (v.invGrab, set the
// moment this drag's origin press landed on the pack bar) stays true for the
// whole gesture and RETURNS before command ever reaches spellbookCaptures'
// own handling further down — the ground-drop arm's own gate,
// `!v.inventoryCaptures(x, y)` alone, does not know the spellbook strip
// exists, so a release over it answered "not captured" and was treated as
// open ground. groundSurfaceCaptures (inventory.go) widens the gate to ask
// every HUD surface, not only inventoryCaptures' own three boxes.
//
// THE SETUP ASSERTION IS THE MEASUREMENT'S OWN CONTROL: it confirms the
// release point is inside the spellbook bar and OUTSIDE inventoryCaptures —
// the exact combination the old gate could not tell apart from open ground.
func TestGroundDropRefusesTheSpellbookStripDuringAnArmedDrag(t *testing.T) {
	v, _ := openDollAt(t)
	v.SetFont(panelFont())
	v.SetSpellbook(5, []SpellEntry{{ID: 1, Name: "Fire Arrow"}})

	bar, cols, ok := v.spellbookBar()
	if !ok {
		t.Fatal("setup: spellbookBar answered false with a font and a book set")
	}
	cells := bookCellRects(bar, cols)
	if len(cells) == 0 {
		t.Fatal("setup: the spellbook bar has no cells")
	}
	r := cells[0]
	sx, sy := (r.Min.X+r.Max.X)/2, (r.Min.Y+r.Max.Y)/2
	if v.inventoryCaptures(sx, sy) {
		t.Fatalf("setup: inventoryCaptures(%d,%d) = true over the spellbook cell, "+
			"want false — this case proves nothing unless the point sits OUTSIDE the doll box, "+
			"the worn box and the pack bar", sx, sy)
	}

	cx, cy := packCellCenter(t, v, 1)
	dollPress(v, cx, cy)
	dollMove(v, sx, sy)
	if !v.dragActive {
		t.Fatal("setup: the drag did not arm before the release under test")
	}
	dollRelease(v, sx, sy)

	if worn, idx, x, y, ok := v.TakeInventoryDrop(); ok {
		t.Errorf("TakeInventoryDrop = (worn=%v idx=%d x=%d y=%d ok=true) for a release over the "+
			"spellbook strip, want ok=false: this HUD surface is not the ground", worn, idx, x, y)
	}
	if idx, ok := v.TakeInventoryEquip(); ok {
		t.Errorf("the same release also raised TakeInventoryEquip(%d), want none", idx)
	}
}

// TestGroundDropRefusesTheMinimapDuringAnArmedDrag is the same defect's
// minimap arm, groundSurfaceCaptures' own third disjunct beside the
// spellbook strip and the unit panel — minimapCaptures answers true for the
// whole of the minimap's own box, not only its drawn terrain, so a release
// anywhere inside that box during an armed drag is refused, matching
// minimapGrab's own reservation of the same rectangle two switches below in
// command.go.
func TestGroundDropRefusesTheMinimapDuringAnArmedDrag(t *testing.T) {
	v, _ := openDollAt(t)
	g, ok := v.minimapGeometry()
	if !ok {
		t.Fatal("setup: minimapGeometry answered false over a real grid")
	}
	mx, my := (g.Box.Min.X+g.Box.Max.X)/2, (g.Box.Min.Y+g.Box.Max.Y)/2
	if v.inventoryCaptures(mx, my) {
		t.Fatalf("setup: inventoryCaptures(%d,%d) = true over the minimap box, want false", mx, my)
	}

	cx, cy := packCellCenter(t, v, 1)
	dollPress(v, cx, cy)
	dollMove(v, mx, my)
	if !v.dragActive {
		t.Fatal("setup: the drag did not arm before the release under test")
	}
	dollRelease(v, mx, my)

	if worn, idx, x, y, ok := v.TakeInventoryDrop(); ok {
		t.Errorf("TakeInventoryDrop = (worn=%v idx=%d x=%d y=%d ok=true) for a release over the "+
			"minimap, want ok=false", worn, idx, x, y)
	}
}

// figureBoxFor is openDollAt's own doll box, re-fetched for a Viewer already
// opened, so a caller that only kept the Viewer can still address the mask
// by dollMaskPoint's own geometry.
func figureBoxFor(t *testing.T, v *Viewer) image.Rectangle {
	t.Helper()
	box, ok := v.dollBox()
	if !ok {
		t.Fatal("setup: dollBox answered false")
	}
	return box
}

// TestDragReleasedOnTheWornBoxStillReturnsToOrigin is the ground drop's own
// boundary: the worn box is part of inventoryCaptures' single surface, so a
// release there is neither an equip/unequip destination nor "outside every
// box" — the request raised is none of the three, exactly as before round 2.
func TestDragReleasedOnTheWornBoxStillReturnsToOrigin(t *testing.T) {
	v, _ := openDollAt(t)
	cx, cy := packCellCenter(t, v, 0)
	wornBox, ok := v.wornBox()
	if !ok {
		t.Fatal("setup: wornBox answered false")
	}
	wx, wy := (wornBox.Min.X+wornBox.Max.X)/2, (wornBox.Min.Y+wornBox.Max.Y)/2

	dollPress(v, cx, cy)
	dollMove(v, cx+3*TapSlop, cy)
	dollRelease(v, wx, wy)

	if worn, idx, _, _, ok := v.TakeInventoryDrop(); ok {
		t.Errorf("TakeInventoryDrop = (worn=%v,idx=%d,true), want nothing — the worn box is an inventory box", worn, idx)
	}
	if idx, ok := v.TakeInventoryEquip(); ok {
		t.Errorf("TakeInventoryEquip = (%d,true), want nothing", idx)
	}
}

// TestDollSubjectSubstitutesTheSuppressedFigure is item 5: while a drag
// holds one of the doll's own slots, the box draws a DIFFERENT composition
// — that slot's own equipment cleared — pushed by pkg/game's refreshDollDrag
// through SetDollSuppressedFigure, and dollSubject is the one place that
// picture is chosen over the ordinary one (its own header comment). The
// returned suppressSlot is part of dollSource's cache key (1005): reverting
// dollSubject to return figure alone, with no suppressSlot field at all,
// would leave the pane's own rebuild guard unable to tell a suppressed
// composition from an ordinary one at the same pointer.
func TestDollSubjectSubstitutesTheSuppressedFigure(t *testing.T) {
	v, _ := openDollAt(t)
	ordinary := v.invSubject.Figure

	src, ok := v.dollSubject()
	if !ok || src.figure != ordinary || src.suppressSlot != 0 {
		t.Fatalf("dollSubject before any drag = %+v, want the ordinary figure and suppressSlot 0", src)
	}

	suppressed := solidPic(4, 4, color.RGBA{G: 0xff, A: 0xff})
	v.SetDollSuppressedFigure(v.invSubject.ID, 3, suppressed, nil)

	src, ok = v.dollSubject()
	if !ok || src.figure != suppressed || src.suppressSlot != 3 {
		t.Errorf("dollSubject while slot 3 is suppressed = %+v, want the suppressed figure and suppressSlot 3", src)
	}

	v.SetDollSuppressedFigure(v.invSubject.ID, 0, nil, nil)
	src, ok = v.dollSubject()
	if !ok || src.figure != ordinary || src.suppressSlot != 0 {
		t.Errorf("dollSubject after the drag ends = %+v, want the ordinary figure restored", src)
	}
}

// dollSuppressedMaskFixture is dollMaskFixture's own shape with slot 1's
// left-half pixels cleared to no slot at all — the mask a drag lifting
// slot 1 off the doll would push alongside the recomposed figure (1005
// round-1 adversarial review): slot 2's own top-right quadrant is left
// exactly as dollMaskFixture states it, since lifting slot 1 does not touch
// slot 2's own layer.
func dollSuppressedMaskFixture() *SlotMask {
	return dollMaskAt(dollMaskClearY, []uint8{
		0, 0, 2, 2,
		0, 0, 2, 2,
		0, 0, 0, 0,
		0, 0, 0, 0,
	})
}

// TestHoverDuringADollDragReadsTheSuppressedMask is the counterexample the
// round-1 adversarial review raised against the UI/HUD aspect: while a drag
// holds slot 1 off the doll, dollFigureSlotAt and hoveredItemInfoAt must
// answer from the SUPPRESSED mask SetDollSuppressedFigure was pushed
// alongside the suppressed figure — the mask of the picture actually drawn —
// not from invSubject.SlotMask, which still describes the figure the doll
// box is no longer showing. Slot 2's own pixels, untouched by the
// suppression, answer exactly as before.
func TestHoverDuringADollDragReadsTheSuppressedMask(t *testing.T) {
	v, box := openDollAt(t)
	x1, y1 := dollMaskPoint(box, 0, 0)

	if lines, ok := v.hoveredItemInfoAt(x1, y1); !ok || len(lines) != 1 || lines[0] != "slot one" {
		t.Fatalf("setup: hoveredItemInfoAt before any drag = (%v,%v), want ([slot one],true)", lines, ok)
	}

	suppressedPic := solidPic(4, 4, color.RGBA{B: 0xff, A: 0xff})
	v.SetDollSuppressedFigure(v.invSubject.ID, 1, suppressedPic, dollSuppressedMaskFixture())

	if n, ok := v.dollFigureSlotAt(x1, y1); ok {
		t.Errorf("dollFigureSlotAt over the lifted slot's own pixels while suppressed = (%d,true), want no slot", n)
	}
	if lines, ok := v.hoveredItemInfoAt(x1, y1); ok {
		t.Errorf("hoveredItemInfoAt over the lifted slot while suppressed = (%v,true), want nothing — the suppressed figure's own bare pixel", lines)
	}

	x2, y2 := dollMaskPoint(box, 2, 0)
	if n, ok := v.dollFigureSlotAt(x2, y2); !ok || n != 1 {
		t.Errorf("dollFigureSlotAt over slot 2 while slot 1 is suppressed = (%d,%v), want (1,true) — untouched by the suppression", n, ok)
	}
}

// TestHoverDuringADollDragNamesTheExposedSlot covers the one mask case the
// test above never presents: a pixel whose correct suppressed answer is a
// DIFFERENT, non-zero slot — the layer the lift uncovers.
//
// Both fixtures above store the same value at every pixel where either is
// non-zero, so the only differences they can express are "slot 1 goes bare"
// and "slot 2 is untouched". Two mutations of dollFigureSlotAt's suppressed
// branch were run against them at the round-1 landing: reading
// invSubject.SlotMask instead of dollSuppressMask, and returning no slot
// whenever a suppression is active. Both redden the test above, so the branch
// itself is already witnessed. What no assertion above can fail on is a
// suppressed mask built without the layers UNDER the lifted one — every pixel
// such a mask names, the ordinary mask names identically.
//
// That is the case DIV-085's stated property turns on once a doll carries
// stacked layers: a hit test and a drawn pixel may never disagree, and after
// the lift the drawn pixel belongs to the layer below. The mask below is what
// composeInventorySubject produces when the dragged slot's own code is
// cleared over a stack; pkg/game's
// TestClearingTheTopSlotExposesTheLowerSlotInBothPictureAndMask builds the
// same case from the compositor rather than by hand. Slot 1 owns the pixel in
// the ordinary mask and slot 7 owns it here, so reading the wrong mask names
// slot 1 and only reading the suppressed one names slot 7.
func TestHoverDuringADollDragNamesTheExposedSlot(t *testing.T) {
	v, box := openDollAt(t)
	x, y := dollMaskPoint(box, 0, 2)

	if n, ok := v.dollFigureSlotAt(x, y); !ok || n != 0 {
		t.Fatalf("setup: dollFigureSlotAt before any drag = (%d,%v), want (0,true) — slot 1 owns this pixel", n, ok)
	}

	exposed := dollMaskAt(dollMaskClearY, []uint8{
		0, 0, 2, 2,
		0, 0, 2, 2,
		7, 7, 0, 0,
		7, 7, 0, 0,
	})
	v.SetDollSuppressedFigure(v.invSubject.ID, 1, solidPic(4, 4, color.RGBA{B: 0xff, A: 0xff}), exposed)

	if n, ok := v.dollFigureSlotAt(x, y); !ok || n != 6 {
		t.Errorf("dollFigureSlotAt over the exposed layer while slot 1 is suppressed = (%d,%v), want (6,true) — slot 7, the layer the lift uncovered", n, ok)
	}
}

// TestDragFromInventoryOverTheMapDoesNotStartBoxSelect is item 6's named
// case: a gesture that began on a pack cell and is carried well past
// TapSlop, off both inventory boxes and onto the map itself, builds no
// box-select outline — v.invGrab's own latch, which keeps the WHOLE gesture
// inside the swallow branch in command.go once it began there, so decide()
// (the map's own box-select build) is never reached for it. marqueeRects
// (marquee_test.go) reads the same overlay pass the map screen actually
// draws from.
func TestDragFromInventoryOverTheMapDoesNotStartBoxSelect(t *testing.T) {
	v, _ := openDollAt(t)
	if !v.commandMode {
		t.Fatal("setup: viewer is not in command mode — box-select could never fire regardless")
	}
	cx, cy := packCellCenter(t, v, 0)

	dollPress(v, cx, cy)
	dollMove(v, cx+3*TapSlop, cy+3*TapSlop) // well past the slop and off both boxes
	if got := marqueeRects(v); got != nil {
		t.Errorf("a drag begun on a pack cell built a box-select outline %+v over the map, want none", got)
	}
	dollRelease(v, cx+3*TapSlop, cy+3*TapSlop)
	if got := marqueeRects(v); got != nil {
		t.Errorf("after release the outline still stands: %+v", got)
	}
}

// TestReleasingAnArmedDollDragOverTheHudToggleBarCancelsIt is counterexample
// 5 (round-2 adversarial review, third pass): the HUD toggle bar's own early
// return in command.go (hudToggleCaptures) sits ahead of the drag machine's
// release switch, so a release inside the bar used to leave dragActive,
// dragCandKind, dragIcon and invGrab exactly as armed as they were
// mid-press. A LATER, unrelated release anywhere off the inventory then read
// the stale dragCandKind == dragFromDoll and resolved as a ground drop at
// that later release's own cell (raiseGroundDrop) — a cell the player
// never aimed the original drag at. raises neither request").
func TestReleasingAnArmedDollDragOverTheHudToggleBarCancelsIt(t *testing.T) {
	v, box := openDollAt(t)
	bar, ok := v.hudToggleBar()
	if !ok {
		t.Fatal("setup: no control panel in the shipped window size")
	}
	if bar.Overlaps(box) {
		t.Fatal("setup: the fixture's own HUD toggle bar overlaps the doll box, so this test cannot isolate the two surfaces")
	}
	x1, y1 := dollMaskPoint(box, 0, 0) // slot 1's own mask pixel
	bx, by := (bar.Min.X+bar.Max.X)/2, (bar.Min.Y+bar.Max.Y)/2

	dollPress(v, x1, y1)
	// ARM THE DRAG OFF THE DOLL, past TapSlop, but still short of the HUD
	// bar: command()'s own hudToggleCaptures early return sits ahead of the
	// arm check (v.dragMoved >= TapSlop) in the same function, so a move
	// frame landing INSIDE the bar would never reach the arm check at all.
	// A real gesture crosses TapSlop somewhere over the map or another
	// inventory box first and only reaches the bar afterward; this fixture
	// states that same order.
	dollMove(v, x1+3*TapSlop, y1)
	if !v.dragActive || v.dragCandKind != dragFromDoll {
		t.Fatalf("setup: dragActive=%v dragCandKind=%v, want an armed doll drag before the release under test", v.dragActive, v.dragCandKind)
	}
	dollRelease(v, bx, by)

	if v.dragActive {
		t.Error("dragActive still true after a release inside the HUD toggle bar")
	}
	if v.dragCandKind != dragNone {
		t.Errorf("dragCandKind = %v, want dragNone after a release inside the HUD toggle bar", v.dragCandKind)
	}
	if v.dragIcon != nil {
		t.Error("dragIcon still set after a release inside the HUD toggle bar")
	}
	if v.invGrab {
		t.Error("invGrab still true after a release inside the HUD toggle bar")
	}
	if _, ok := v.TakeInventoryDollUnequip(); ok {
		t.Error("a release inside the HUD toggle bar raised an unequip request")
	}
	if _, _, _, _, ok := v.TakeInventoryDrop(); ok {
		t.Error("a release inside the HUD toggle bar raised a ground drop request")
	}

	// THE OBSERVABLE CONSEQUENCE: a second, wholly unrelated press and
	// release far off every inventory box and off the HUD bar — the stale
	// state, left standing before this fix, resolved THIS release as a
	// ground drop of the original drag's own origin slot.
	dollPress(v, 5, 5)
	dollRelease(v, 5, 5)
	if worn, idx, x, y, ok := v.TakeInventoryDrop(); ok {
		t.Errorf("an unrelated later release raised a ground drop (worn=%v idx=%d x=%d y=%d), want none: the cancelled drag must not survive to a later gesture", worn, idx, x, y)
	}
}

// TestInventoryCapturesTheReservedGroundOnlyWhileADollDragIsInFlight is
// counterexample 6 (round-2 adversarial review, third pass) and
// counterexample C (fifth pass), the same reserved ground with its contract
// corrected: under a selection led by the subject but holding more than one
// entity, inventoryEligible is false, so wornBox and packBar both answer
// "not drawn" while dollBox — dollSubject's own wider rule — still
// stands. The reserved ground stands in for the invisible pack bar ONLY
// WHILE A DOLL-ORIGIN DRAG IS IN FLIGHT, never merely because this selection
// state holds: a plain press or release at that same screen position with no
// drag under way is an ordinary map gesture and must reach the map's own
// dispatch. Before the fifth pass's fix, inventoryCaptures reserved the
// ground for the SELECTION STATE alone, so a plain click aimed at a map cell
// the pack bar happens to stand over on every other selection was silently
// swallowed and never became a move order.
func TestInventoryCapturesTheReservedGroundOnlyWhileADollDragIsInFlight(t *testing.T) {
	v, box := openDollAt(t)
	v.sel = selection{5, 6} // multi-unit, subject (5) still first

	if _, ok := v.dollBox(); !ok {
		t.Fatal("setup: dollBox answered false under a subject-led multi-selection")
	}
	if _, ok := v.wornBox(); ok {
		t.Fatal("setup: wornBox answered true under a multi-selection — inventoryEligible did not narrow it, so this case proves nothing")
	}
	if _, _, ok := v.packBar(); ok {
		t.Fatal("setup: packBar answered true under a multi-selection — inventoryEligible did not narrow it, so this case proves nothing")
	}
	bar, ok := v.packBarArea()
	if !ok {
		t.Fatal("setup: packBarArea answered false — the pack switch itself must still be on for this case")
	}
	if bar.Overlaps(box) {
		t.Fatal("setup: the fixture's own pack bar area overlaps the doll box")
	}
	px, py := (bar.Min.X+bar.Max.X)/2, (bar.Min.Y+bar.Max.Y)/2

	// COUNTEREXAMPLE C: with no drag armed at all, the reserved ground is
	// map ground — a plain press-and-release at this exact point must not be
	// swallowed by the inventory window.
	if v.inventoryCaptures(px, py) {
		t.Fatalf("inventoryCaptures(%d,%d) = true with no doll drag in flight, want false: this is a plain map gesture, not a drag release", px, py)
	}

	// COUNTEREXAMPLE 6, restated: once a doll-origin drag IS in flight, the
	// same point is reserved, so a release there must not resolve as a
	// ground drop — the item returns to its origin exactly as a release back
	// inside a drawn pack bar already would.
	x1, y1 := dollMaskPoint(box, 0, 0)
	dollPress(v, x1, y1)
	dollMove(v, x1+3*TapSlop, y1)
	if !v.dragActive || v.dragCandKind != dragFromDoll {
		t.Fatalf("setup: dragActive=%v dragCandKind=%v, want an armed doll drag before the release under test", v.dragActive, v.dragCandKind)
	}
	if !v.inventoryCaptures(px, py) {
		t.Errorf("inventoryCaptures(%d,%d) = false over the pack bar's own reserved ground with a doll drag in flight, want true", px, py)
	}
	dollRelease(v, px, py)
	if worn, idx, x, y, ok := v.TakeInventoryDrop(); ok {
		t.Errorf("a release over the pack bar's own reserved ground raised a ground drop (worn=%v idx=%d x=%d y=%d), want none", worn, idx, x, y)
	}
	// COUNTEREXAMPLE D: the drag must resolve as the unequip it plainly is,
	// not merely fail to raise a ground drop — command.go's dragFromDoll
	// release arm used to ask the narrow packBar (false here) rather than
	// packBarArea, so the gesture was recognised (no ground drop) but never
	// completed (no unequip either): the item simply snapped back with
	// nothing to show for the release.
	if _, ok := v.TakeInventoryDollUnequip(); !ok {
		t.Error("a doll drag released over the reserved pack ground under a multi-selection raised no unequip request")
	}
}

func TestInventoryDoesNotCaptureInvisibleAreasWithoutASelection(t *testing.T) {
	v, _ := openDollAt(t)
	v.sel = nil
	bar, ok := v.packBarArea()
	if !ok {
		t.Fatal("setup: pack switch did not provide its geometric area")
	}
	px, py := (bar.Min.X+bar.Max.X)/2, (bar.Min.Y+bar.Max.Y)/2
	if v.inventoryCaptures(px, py) {
		t.Fatalf("inventoryCaptures(%d,%d) = true with no selection, want false: an empty pack bar leaves the map ground clickable", px, py)
	}
}

// TestAFallbackOnlySlotOneIsTakenOffLikeAnyOther is the owner's own
// directive and it replaces the refusal this test asserted for four passes.
// WeaponFallback means the doll's mask paints slot 1 from PartyMember.Weapon
// with nothing in the live entity's equipment array behind it; the press arm
// used to refuse to arm anything there, so the mission doll drew a weapon,
// named it on hover and did nothing when pressed, while the SHOP's doll took
// the same drawn weapon off. The gesture is now armed here and answered
// across the seam by pkg/game's materializeFallbackWeapon.
//
// A TAP AND A DRAG ARE BOTH ASSERTED, because they leave this function by two
// different arms: the tap through the release's originSame branch, the drag
// through dragActive.
func TestAFallbackOnlySlotOneIsTakenOffLikeAnyOther(t *testing.T) {
	v, box := openDollAt(t)
	sub := v.invSubject
	sub.WeaponFallback = true
	v.SetInventorySubject(sub)

	x1, y1 := dollMaskPoint(box, 0, 0) // slot 1's own mask pixel
	dollPress(v, x1, y1)
	dollRelease(v, x1, y1)
	if idx, ok := v.TakeInventoryDollUnequip(); !ok || idx != 0 {
		t.Errorf("a tap on a fallback-only slot 1 raised unequip (%d,%v), want (0,true) -- the zero-based slot", idx, ok)
	}

	dollPress(v, x1, y1)
	dollMove(v, x1+3*TapSlop, y1) // well past TapSlop
	if !v.dragActive || v.dragCandKind != dragFromDoll {
		t.Errorf("dragActive=%v dragCandKind=%v after a press+move on a fallback-only slot 1, want an armed doll drag", v.dragActive, v.dragCandKind)
	}
	bar, ok := v.packBarArea()
	if !ok {
		t.Fatal("setup: the pack bar has no area to release over")
	}
	bx, by := (bar.Min.X+bar.Max.X)/2, (bar.Min.Y+bar.Max.Y)/2
	dollMove(v, bx, by)
	dollRelease(v, bx, by)
	if idx, ok := v.TakeInventoryDollUnequip(); !ok || idx != 0 {
		t.Errorf("a fallback-only slot 1 dragged to the pack raised unequip (%d,%v), want (0,true) -- the zero-based slot", idx, ok)
	}

	// HOVER IS UNAFFECTED: the mask names the slot exactly as it did while the
	// arm above was a refusal.
	if n, ok := v.dollFigureSlotAt(x1, y1); !ok || n != 0 {
		t.Errorf("dollFigureSlotAt over a fallback-only slot 1 = (%d,%v), want (0,true): the hit test itself is not gated by WeaponFallback", n, ok)
	}

	// A DIFFERENT SLOT STILL ARMS under the same WeaponFallback (which only
	// ever names slot 1): slot 2's own mask pixel is unaffected.
	x2, y2 := dollMaskPoint(box, 2, 0) // slot 2's own mask pixel (top-right quadrant)
	dollPress(v, x2, y2)
	dollMove(v, x2+3*TapSlop, y2)
	if !v.dragActive || v.dragCandKind != dragFromDoll {
		t.Errorf("dragActive=%v dragCandKind=%v after a press+move on slot 2 while slot 1's WeaponFallback is set, want an armed doll drag: WeaponFallback refuses only slot 1", v.dragActive, v.dragCandKind)
	}
	dollRelease(v, x2+3*TapSlop, y2)
}
