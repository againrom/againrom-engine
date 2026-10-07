package ui

// The spellbook strip's own click behaviour: a click on an entry selects it,
// a second click on the same one clears it, and changing the selected unit
// clears it too. Driven through v.command, inventory's own invclick_test.go
// shape — no window, no engine, a real map screen with no game data behind
// it.

import (
	"image"
	"slices"
	"testing"
	"time"
)

// spellbookTestApp is inventoryTestApp's own shape (inventory_test.go), with
// a font added: the strip needs one to measure and draw its rows, where the
// inventory window's fixed-pixel geometry does not.
func spellbookTestApp(t *testing.T) *App {
	t.Helper()
	a := newTestApp(t, appRows(3), okLoader(t))
	a.flow.screen = ScreenPicker
	a.step(appInput{Enter: true}, time.Unix(1_700_000_000, 0))
	if a.Screen() != ScreenMap {
		t.Fatalf("setup: screen = %v, want ScreenMap", a.Screen())
	}
	a.flow.viewer.SetFont(panelFont())
	return a
}

// The two units a case selects between, and a small two-spell book for the
// first of them.
const (
	sbUnitID  = 5
	sbOtherID = 6
)

func sbEntities() []MapEntity {
	return []MapEntity{
		{ID: sbUnitID, Life: LifeAlive},
		{ID: sbOtherID, Life: LifeAlive},
	}
}

func sbBook() []SpellEntry {
	return []SpellEntry{{ID: 1, Name: "Fire Arrow"}, {ID: 6, Name: "Heal"}}
}

// sbClick is one press-then-release at a window position, the shape a real
// tick delivers a click in and invclick_test.go's own two-call pattern.
func sbClick(v *Viewer, x, y int) {
	v.command(appInput{PrimaryPressed: true, CursorX: x, CursorY: y})
	v.command(appInput{PrimaryReleased: true, CursorX: x, CursorY: y})
}

// sbEntryPoint is a window pixel inside entry i of the book, by the SAME
// geometry the bar is drawn and hit-tested with (spellbookBar, bookCellRects)
// — inventoryPackCellAt's own precedent: a test judges a click against the
// rectangle the code itself computes, never a position guessed independently
// of it.
func sbEntryPoint(t *testing.T, v *Viewer, book []SpellEntry, i int) (int, int) {
	t.Helper()
	bar, cols, ok := v.spellbookBar()
	if !ok {
		t.Fatal("setup: spellbookBar answered false with a book set")
	}
	if i >= len(book) {
		t.Fatalf("setup: entry %d is past the end of a book of %d", i, len(book))
	}
	r := bookCellRects(bar, cols)[i]
	return (r.Min.X + r.Max.X) / 2, (r.Min.Y + r.Max.Y) / 2
}

func TestASpellbookClickSelectsAndASecondClickOnTheSameOneClears(t *testing.T) {
	a := spellbookTestApp(t)
	mapAtScaleOne(a)
	v := a.flow.viewer
	v.SetEntities(sbEntities())
	v.sel = selection{sbUnitID}
	book := sbBook()
	v.SetSpellbook(sbUnitID, book)

	x, y := sbEntryPoint(t, v, book, 0)

	sbClick(v, x, y)
	if v.selectedSpell != book[0].ID {
		t.Fatalf("selectedSpell = %d after a click on entry 0, want %d", v.selectedSpell, book[0].ID)
	}

	sbClick(v, x, y)
	if v.selectedSpell != 0 {
		t.Errorf("selectedSpell = %d after a second click on the same entry, want 0 (cleared)", v.selectedSpell)
	}
}

func TestASpellbookClickOnADifferentEntryReplacesTheSelection(t *testing.T) {
	a := spellbookTestApp(t)
	mapAtScaleOne(a)
	v := a.flow.viewer
	v.SetEntities(sbEntities())
	v.sel = selection{sbUnitID}
	book := sbBook()
	v.SetSpellbook(sbUnitID, book)

	x0, y0 := sbEntryPoint(t, v, book, 0)
	x1, y1 := sbEntryPoint(t, v, book, 1)

	sbClick(v, x0, y0)
	if v.selectedSpell != book[0].ID {
		t.Fatalf("selectedSpell = %d after selecting entry 0, want %d", v.selectedSpell, book[0].ID)
	}
	sbClick(v, x1, y1)
	if v.selectedSpell != book[1].ID {
		t.Errorf("selectedSpell = %d after clicking a different entry, want %d", v.selectedSpell, book[1].ID)
	}
}

func TestChangingTheSelectedUnitClearsTheSpellSelection(t *testing.T) {
	a := spellbookTestApp(t)
	mapAtScaleOne(a)
	v := a.flow.viewer
	v.SetEntities(sbEntities())
	v.sel = selection{sbUnitID}
	book := sbBook()
	v.SetSpellbook(sbUnitID, book)

	x, y := sbEntryPoint(t, v, book, 0)
	sbClick(v, x, y)
	if v.selectedSpell != book[0].ID {
		t.Fatalf("setup: selectedSpell = %d after selecting entry 0, want %d", v.selectedSpell, book[0].ID)
	}

	v.SetSpellbook(sbOtherID, book)
	if v.selectedSpell != 0 {
		t.Errorf("selectedSpell = %d after the pushed book's owner changed, want 0 (cleared)", v.selectedSpell)
	}

	// The SAME owner repeated — the ordinary case, once a frame, for as
	// long as the same unit stays selected — must NOT clear a fresh
	// selection.
	sbClick(v, x, y)
	if v.selectedSpell != book[0].ID {
		t.Fatalf("setup: selectedSpell = %d after re-selecting entry 0, want %d", v.selectedSpell, book[0].ID)
	}
	v.SetSpellbook(sbOtherID, book)
	if v.selectedSpell != book[0].ID {
		t.Errorf("selectedSpell = %d after SetSpellbook repeated the SAME owner, want %d (unchanged)",
			v.selectedSpell, book[0].ID)
	}
}

// The owner keeps a chosen spell ready after casting. Both target kinds use
// ordinary press/release gestures; a same-owner book refresh preserves the mode.
func TestTheSpellSelectionStaysArmedForRepeatedCasts(t *testing.T) {
	a := spellbookTestApp(t)
	mapAtScaleOne(a)
	v := a.flow.viewer

	const casterID, victimID = 5, 9
	v.SetEntities([]MapEntity{
		{ID: casterID, Cell: image.Pt(3, 3), Life: LifeAlive},
		{ID: victimID, Cell: image.Pt(7, 3), Life: LifeAlive},
	})
	v.Camera().X, v.Camera().Y = 0, 0
	v.Camera().Clamp()
	v.sel = selection{casterID}
	book := sbBook()
	v.SetSpellbook(casterID, book)

	x, y := sbEntryPoint(t, v, book, 0)
	sbClick(v, x, y)
	if v.selectedSpell != book[0].ID {
		t.Fatalf("setup: selectedSpell = %d after selecting entry 0, want %d", v.selectedSpell, book[0].ID)
	}

	v.armed = true
	vx, vy := cellPoint(v, 7, 3)
	if v.spellbookCaptures(vx, vy) {
		t.Fatal("setup: the victim's own cell falls inside the spellbook strip — fixture needs to move one of the two")
	}
	ords, ok := tapAt(v, vx, vy)
	if !ok || len(ords) != 1 || ords[0].kind != orderKindCast || ords[0].victim != victimID || ords[0].spell != book[0].ID {
		t.Fatalf("setup: the armed press with a spell selected produced %+v (ok=%v), want one cast order onto %d naming spell %d",
			ords, ok, victimID, book[0].ID)
	}
	if v.selectedSpell != book[0].ID || !v.spellArmed {
		t.Fatalf("after cast: selected spell %d, armed %v; want the same armed spell", v.selectedSpell, v.spellArmed)
	}
	// Same-owner refreshes happen every game frame, including after the
	// simulation applies a spell. Repeated entity and ground clicks need no
	// intervening book click or arming key.
	for _, target := range []image.Point{{7, 3}, {7, 3}} {
		v.SetSpellbook(casterID, book)
		x, y := cellPoint(v, target.X, target.Y)
		if got := v.gestureCursorAt(x, y); got != "cast" {
			t.Fatalf("repeat cursor = %q, want cast", got)
		}
		ords, ok := tapAt(v, x, y)
		if !ok || len(ords) != 1 || ords[0].spell != book[0].ID || ords[0].entity != casterID {
			t.Fatalf("repeat at %v produced %v/%v", target, ords, ok)
		}
	}
	// An area spell repeats on open ground, where a target spell does not.
	book = append(sbBook(), SpellEntry{ID: 2, Name: "Fire Ball", PointTarget: true})
	v.SetSpellbook(casterID, book)
	v.selectedSpell = 2
	for _, target := range []image.Point{{5, 6}, {6, 6}} {
		v.SetSpellbook(casterID, book)
		x, y := cellPoint(v, target.X, target.Y)
		if got := v.gestureCursorAt(x, y); got != "cast" {
			t.Fatalf("area repeat cursor = %q, want cast", got)
		}
		ords, ok := tapAt(v, x, y)
		if !ok || len(ords) != 1 || ords[0].spell != 2 || !ords[0].cell {
			t.Fatalf("area repeat at %v produced %v/%v", target, ords, ok)
		}
	}
	// A right click still cancels casting without deselecting the mage.
	v.command(appInput{SecondaryReleased: true, CursorX: vx, CursorY: vy})
	if v.selectedSpell != 0 || v.spellArmed || !slices.Equal(v.sel, selection{casterID}) {
		t.Fatal("right click failed to cancel only the cast mode")
	}
}

func TestASelectedSpellNeedsNoSecondArmingKey(t *testing.T) {
	a := spellbookTestApp(t)
	mapAtScaleOne(a)
	v := a.flow.viewer

	const casterID, victimID = 5, 9
	v.SetEntities([]MapEntity{
		{ID: casterID, Cell: image.Pt(3, 3), Life: LifeAlive},
		{ID: victimID, Cell: image.Pt(7, 3), Life: LifeAlive},
	})
	v.Camera().X, v.Camera().Y = 0, 0
	v.Camera().Clamp()
	v.sel = selection{casterID}
	book := sbBook()
	v.SetSpellbook(casterID, book)

	vx, vy := cellPoint(v, 7, 3)
	if v.spellbookCaptures(vx, vy) {
		t.Fatal("setup: the victim's own cell falls inside the spellbook strip — fixture needs to move one of the two")
	}

	// THE CONTROL, and it runs first so a fixture that could never issue an
	// order cannot let the case below pass for the wrong reason: with no spell
	// selected and no arm, a tap is the plain move it has always been.
	//
	// IT IS MADE ON EMPTY GROUND rather than on the victim's own cell (story
	// 1034). Unarmed, over a non-hostile actor, the cursor is `select` and the
	// click goes to the selection routine (`AI-CURSOR-226` arm 4,
	// `AI-CLICK-050`), so the same tap would select the victim instead of
	// ordering anything. Empty ground with a unit selected is `move`, which is
	// the arm this control needs. The caster stays selected either way, which is
	// what the case below depends on.
	const ctrlCol, ctrlRow = 5, 6
	cx, cy := cellPoint(v, ctrlCol, ctrlRow)
	ctrl, ok := tapAt(v, cx, cy)
	if !ok || len(ctrl) != 1 || ctrl[0].kind != orderKindMove || ctrl[0].spell != 0 || ctrl[0].x != ctrlCol || ctrl[0].y != ctrlRow {
		t.Fatalf("control: the unarmed tap with no spell selected produced %+v (ok=%v), want one plain move to (%d,%d)",
			ctrl, ok, ctrlCol, ctrlRow)
	}
	if want := (selection{casterID}); !slices.Equal(v.sel, want) {
		t.Fatalf("control: the ordering tap left selection %+v, want %+v", v.sel, want)
	}

	x, y := sbEntryPoint(t, v, book, 0)
	sbClick(v, x, y)
	if v.selectedSpell != book[0].ID {
		t.Fatalf("setup: selectedSpell = %d after selecting entry 0, want %d", v.selectedSpell, book[0].ID)
	}
	if v.armed || v.attackHeld {
		t.Fatalf("setup: selecting a spell raised the attack arm itself (armed=%v held=%v) — this case must run with both DOWN",
			v.armed, v.attackHeld)
	}

	ords, ok2 := tapAt(v, vx, vy)
	if !ok2 || len(ords) != 1 {
		t.Fatalf("the press produced %+v (ok=%v), want exactly one order", ords, ok2)
	}
	if ords[0].kind != orderKindCast || ords[0].entity != casterID || ords[0].victim != victimID || ords[0].spell != book[0].ID {
		t.Errorf("the press produced %+v, want one cast by %d onto %d naming spell %d",
			ords[0], casterID, victimID, book[0].ID)
	}
}

// DIV-326
func TestASelectedUnitWithNoSpellsStillOpensAnEmptyBar(t *testing.T) {
	a := spellbookTestApp(t)
	mapAtScaleOne(a)
	v := a.flow.viewer
	v.SetEntities(sbEntities())
	v.sel = selection{sbUnitID}
	v.SetSpellbook(sbUnitID, nil)

	bar, cols, ok := v.spellbookBar()
	if !ok {
		t.Fatal("spellbookBar answered false for a selected unit with an empty book, want the bar still shown (empty)")
	}
	if bar.Dx() == 0 || bar.Dy() == 0 || cols == 0 {
		t.Fatalf("spellbookBar returned an empty rectangle (%v) or 0 columns for a selected unit with an empty book", bar)
	}

	pix, origin, present := v.spellbookPresent()
	if !present {
		t.Fatal("spellbookPresent answered false for a selected unit with an empty book, want the bar still drawn")
	}
	if origin != bar.Min {
		t.Errorf("spellbookPresent origin = %v, want %v (spellbookBar's own Min)", origin, bar.Min)
	}
	want := composeSpellBar(v.font, nil, 0, cols, bar, int(v.anim.Count()))
	if !imagesEqual(pix, want) {
		t.Error("spellbookPresent's pixels differ from composeSpellBar(nil book) — an empty book must render as ordinary blank bordered cells, nothing else")
	}

	// With no unit selected at all the switched-on bar stands empty
	// (spellbookPresent draws the no-hero line); it names no entry.
	v.ClearSpellbook()
	if _, _, ok := v.spellbookBar(); !ok || v.spellbookHeld {
		t.Error("a cleared book did not stand empty")
	}
}

// Entity id 0 is a real unit: its book stands and answers clicks, and only
// ClearSpellbook takes the bar away.
func TestASpellbookOwnedByEntityZeroStandsAndAnswersClicks(t *testing.T) {
	a := spellbookTestApp(t)
	mapAtScaleOne(a)
	v := a.flow.viewer
	v.SetEntities([]MapEntity{{ID: 0, Life: LifeAlive}})
	v.sel = selection{0}
	book := sbBook()
	v.SetSpellbook(0, book)

	x, y := sbEntryPoint(t, v, book, 0)
	sbClick(v, x, y)
	if v.selectedSpell != book[0].ID {
		t.Fatalf("selectedSpell = %d after a click on entry 0 of entity 0's book, want %d", v.selectedSpell, book[0].ID)
	}
	v.SetSpellbook(0, book)
	if v.selectedSpell != book[0].ID {
		t.Errorf("repeating the same owner cleared the selected spell")
	}
	v.ClearSpellbook()
	if v.spellbookHeld || v.selectedSpell != 0 {
		t.Errorf("ClearSpellbook left the book held (%v) or spell %d selected", v.spellbookHeld, v.selectedSpell)
	}
}

// A book-chosen spell casts only while the book is shown.
func TestClosingTheBookEndsTheBookChosenCastCursor(t *testing.T) {
	a := spellbookTestApp(t)
	mapAtScaleOne(a)
	v := a.flow.viewer

	const casterID, victimID = 5, 9
	v.SetEntities([]MapEntity{
		{ID: casterID, Cell: image.Pt(3, 3), Life: LifeAlive},
		{ID: victimID, Cell: image.Pt(7, 3), Life: LifeAlive},
	})
	v.Camera().X, v.Camera().Y = 0, 0
	v.Camera().Clamp()
	v.sel = selection{casterID}
	book := sbBook()
	v.SetSpellbook(casterID, book)
	x, y := sbEntryPoint(t, v, book, 0)
	sbClick(v, x, y)
	vx, vy := cellPoint(v, 7, 3)
	if got := v.gestureCursorAt(vx, vy); got != "cast" || v.missionMode() != modeCast {
		t.Fatalf("open book: cursor=%q mode=%d, want cast", got, v.missionMode())
	}

	v.toggleHudPanel(hudPanelBook)
	vx, vy = cellPoint(v, 7, 3)
	if got := v.gestureCursorAt(vx, vy); got == "cast" || v.missionMode() == modeCast {
		t.Fatalf("closed book: cursor=%q mode=%d, want the ordinary cursor", got, v.missionMode())
	}
	if _, ok := commandPanelSelected(v); ok {
		t.Fatal("closed book left the command panel Cast cell lit")
	}
	if ords, _ := tapAt(v, vx, vy); len(ords) > 0 && ords[0].kind == orderKindCast {
		t.Fatalf("closed book still issued a cast order %+v", ords)
	}

	v.toggleHudPanel(hudPanelBook)
	vx, vy = cellPoint(v, 7, 3)
	if got := v.gestureCursorAt(vx, vy); got != "cast" {
		t.Fatalf("reopened book: cursor=%q, want cast", got)
	}
}
