package ui

// The autocast toggle, the rotating dashed border and the spellbook popup
// (0154 AC-10, AC-11, AC-12). Driven through the same real map screen with no
// game data behind it that spellbook_test.go's own click cases use.

import (
	"image"
	"image/color"
	"os"
	"regexp"
	"testing"
	"time"
)

// acBook is sbBook with the second entry already on autocast, which is the
// state the far side pushes once a toggle has landed.
func acBook(auto uint32) []SpellEntry {
	book := []SpellEntry{
		{ID: 1, Name: "Fire Arrow", Info: []string{"Fire Arrow", "Mana 3", "Range 7", "Fire", "Damage 4-8"}},
		{ID: 6, Name: "Heal", Info: []string{"Heal", "Mana 10", "Range 6", "Life", "Heals 10-20"}},
	}
	for i := range book {
		book[i].Autocast = book[i].ID == auto
	}
	return book
}

// acApp is spellbookTestApp with a selected unit, a book and a recording
// autocast sink already in place.
func acApp(t *testing.T, book []SpellEntry) (*App, *[][2]uint32) {
	t.Helper()
	a := spellbookTestApp(t)
	mapAtScaleOne(a)
	v := a.flow.viewer
	v.SetEntities(sbEntities())
	v.sel = selection{sbUnitID}
	v.SetSpellbook(sbUnitID, book)
	var sent [][2]uint32
	v.SetAutocastSink(func(entity, spell uint32) {
		sent = append(sent, [2]uint32{entity, spell})
	})
	return a, &sent
}

// ---------------------------------------------------------------- AC-10

func TestRightClickOnASpellCellSetsAndClearsAutocastThroughTheSink(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	for _, tc := range []struct {
		name string
		auto uint32
		want uint32
	}{
		{"activate", 0, 1},
		{"deactivate", 1, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a, sent := acApp(t, acBook(tc.auto))
			mapAtScaleOne(a)
			mapAtScaleOne(a)
			v := a.flow.viewer
			x, y := sbEntryPoint(t, v, v.spellbook, 0)
			a.step(appInput{CursorX: x, CursorY: y, SecondaryPressed: true}, now)
			if len(*sent) != 1 || (*sent)[0] != [2]uint32{sbUnitID, tc.want} {
				t.Fatalf("right-click sent %v, want unit %d spell %d", *sent, sbUnitID, tc.want)
			}
			if v.selectedSpell != 0 {
				t.Errorf("right-click also selected manual spell %d", v.selectedSpell)
			}
		})
	}
}

func TestOnlyARightClickOnAKnownSpellCellTogglesAutocast(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	for _, tc := range []struct {
		name  string
		input func(*testing.T, *Viewer) appInput
	}{
		{"left button", func(t *testing.T, v *Viewer) appInput {
			x, y := sbEntryPoint(t, v, v.spellbook, 0)
			return appInput{CursorX: x, CursorY: y, PrimaryPressed: true}
		}},
		{"empty cell", func(t *testing.T, v *Viewer) appInput {
			bar, cols, ok := v.spellbookBar()
			if !ok {
				t.Fatal("no spellbook")
			}
			cells := bookCellRects(bar, cols)
			p := cells[len(cells)-1].Min.Add(image.Pt(2, 2))
			return appInput{CursorX: p.X, CursorY: p.Y, SecondaryPressed: true}
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a, sent := acApp(t, acBook(0))
			a.step(tc.input(t, a.flow.viewer), now)
			if len(*sent) != 0 {
				t.Fatalf("gesture sent %v, want no autocast command", *sent)
			}
		})
	}
}

// TestCtrlAWithAUnitAndASpellSelectedSetsClearsAndReplacesTheAutocast is AC-10
// whole, driven through the app's own step so the KEY and not the method is
// what is measured.
func TestCtrlAWithAUnitAndASpellSelectedSetsClearsAndReplacesTheAutocast(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)

	t.Run("it sets the selected spell", func(t *testing.T) {
		a, sent := acApp(t, acBook(0))
		mapAtScaleOne(a)
		v := a.flow.viewer
		x, y := sbEntryPoint(t, v, v.spellbook, 0)
		sbClick(v, x, y)

		a.step(appInput{Autocast: true}, now)

		if len(*sent) != 1 || (*sent)[0] != [2]uint32{sbUnitID, 1} {
			t.Errorf("the sink received %v, want one call naming unit %d and spell 1", *sent, sbUnitID)
		}
	})

	t.Run("it clears a spell already on autocast", func(t *testing.T) {
		// The book arrives with spell 1 already on autocast, which is what
		// the far side pushes after the first toggle landed.
		a, sent := acApp(t, acBook(1))
		v := a.flow.viewer
		x, y := sbEntryPoint(t, v, v.spellbook, 0)
		sbClick(v, x, y)

		a.step(appInput{Autocast: true}, now)

		if len(*sent) != 1 || (*sent)[0] != [2]uint32{sbUnitID, 0} {
			t.Errorf("the sink received %v, want one call clearing unit %d", *sent, sbUnitID)
		}
	})

	t.Run("it replaces the setting when a different spell is selected", func(t *testing.T) {
		a, sent := acApp(t, acBook(1))
		v := a.flow.viewer
		x, y := sbEntryPoint(t, v, v.spellbook, 1)
		sbClick(v, x, y)

		a.step(appInput{Autocast: true}, now)

		if len(*sent) != 1 || (*sent)[0] != [2]uint32{sbUnitID, 6} {
			t.Errorf("the sink received %v, want one call naming spell 6", *sent)
		}
	})

	t.Run("with no spell selected it changes nothing", func(t *testing.T) {
		a, sent := acApp(t, acBook(0))

		a.step(appInput{Autocast: true}, now)

		if len(*sent) != 0 {
			t.Errorf("the sink received %v with no spell selected, want nothing", *sent)
		}
	})

	t.Run("over a unit the local player does not own it changes nothing", func(t *testing.T) {
		a, sent := acApp(t, acBook(0))
		v := a.flow.viewer
		// The selected unit is owned by nobody, so naming a local owner
		// makes the arming gate refuse it — canArmAttack's own rule.
		v.SetLocalOwner(9)
		x, y := sbEntryPoint(t, v, v.spellbook, 0)
		sbClick(v, x, y)

		a.step(appInput{Autocast: true}, now)

		if len(*sent) != 0 {
			t.Errorf("the sink received %v over an unowned unit, want nothing", *sent)
		}
	})
}

func TestTheAutocastKeyIsNotAlsoAPan(t *testing.T) {
	src, err := os.ReadFile("viewer.go")
	if err != nil {
		t.Fatalf("reading this package's own viewer.go: %v", err)
	}
	body := string(src)
	for _, key := range []string{"KeyA", "KeyD", "KeyW", "KeyS"} {
		// \b after the key name excludes ebiten.KeyArrowUp/Down/Left/Right,
		// which share the "ebiten.KeyA" prefix with KeyA but are a different
		// identifier.
		re := regexp.MustCompile(`ebiten\.` + key + `\b`)
		if re.MatchString(body) {
			t.Errorf("viewer.go names ebiten.%s; readInput's own camera pan must read the arrow keys alone since story 1028", key)
		}
	}
}

// ---------------------------------------------------------------- AC-11

// TestOnlyAnAutocastingCellCarriesTheDashedBorderAndItsDashesTravel is AC-11's
// two halves: the mark is on the autocasting cell and on no other, and two
// frames one ambient step apart draw the dashes in different places.
func TestOnlyAnAutocastingCellCarriesTheDashedBorderAndItsDashesTravel(t *testing.T) {
	a := spellbookTestApp(t)
	v := a.flow.viewer
	v.SetEntities(sbEntities())
	v.sel = selection{sbUnitID}

	v.SetSpellbook(sbUnitID, acBook(6))
	bar, cols, ok := v.spellbookBar()
	if !ok {
		t.Fatal("setup: the bar stands nowhere with a book set")
	}
	cells := bookCellRects(bar, cols)

	plain := composeSpellBar(v.font, acBook(0), 0, cols, bar, 0)
	marked := composeSpellBar(v.font, acBook(6), 0, cols, bar, 0)

	// Entry 1 is the autocasting one, so its cell must differ; entry 0 must
	// not — which is what says the mark is per-cell and not per-bar.
	if sameCell(plain, marked, cells[0].Sub(bar.Min)) == false {
		t.Error("the cell of a spell that is not on autocast changed when another one was put on it")
	}
	if sameCell(plain, marked, cells[1].Sub(bar.Min)) {
		t.Error("the autocasting cell is identical to the same cell without the setting")
	}

	// And the dashes travel: one ambient step moves them.
	moved := composeSpellBar(v.font, acBook(6), 0, cols, bar, 1)
	if sameCell(marked, moved, cells[1].Sub(bar.Min)) {
		t.Error("two frames one ambient step apart draw the dashes in the same place")
	}
}

func TestTheDashedBorderIsDashedAndNotSolid(t *testing.T) {
	box := image.Rect(0, 0, 36, 36)
	img := image.NewRGBA(box)
	drawAutocastBorder(img, box, 0, color.RGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff})

	on, off := 0, 0
	for x := box.Min.X; x < box.Max.X; x++ {
		if img.RGBAAt(x, box.Min.Y).A != 0 {
			on++
		} else {
			off++
		}
	}
	if on == 0 || off == 0 {
		t.Errorf("the top edge carries %d drawn and %d clear pixel(s); a dash needs both", on, off)
	}
}

// sameCell reports whether two compositions agree over one cell's rectangle.
func sameCell(a, b *image.RGBA, box image.Rectangle) bool {
	for y := box.Min.Y; y < box.Max.Y; y++ {
		for x := box.Min.X; x < box.Max.X; x++ {
			if a.RGBAAt(x, y) != b.RGBAAt(x, y) {
				return false
			}
		}
	}
	return true
}

// ---------------------------------------------------------------- AC-12

// TestTheSpellPopupStatesTheHoveredSpellAndNothingElse is AC-12: a hovered cell
// yields its own lines, an empty cell yields none, and the hover issues no
// order and swallows no press.
func TestTheSpellPopupStatesTheHoveredSpellAndNothingElse(t *testing.T) {
	a := spellbookTestApp(t)
	v := a.flow.viewer
	v.SetEntities(sbEntities())
	v.sel = selection{sbUnitID}
	book := acBook(6)
	v.SetSpellbook(sbUnitID, book)

	x, y := sbEntryPoint(t, v, book, 1)
	v.cursorX, v.cursorY, v.hasCursor = x, y, true

	pic, at, ok := v.spellPopupPresent()
	if !ok || pic == nil {
		t.Fatal("no popup over a cell whose entry carries lines")
	}
	if at.X < 0 || at.Y < 0 || at.X+pic.Bounds().Dx() > v.frameW || at.Y+pic.Bounds().Dy() > v.frameH {
		t.Errorf("the popup at %v sized %v runs off a %dx%d frame",
			at, pic.Bounds().Size(), v.frameW, v.frameH)
	}

	// A CELL PAST THE END OF THE BOOK, which is most of this bar: no entry,
	// so no lines, so no popup.
	bar, cols, _ := v.spellbookBar()
	empty := bookCellRects(bar, cols)[len(book)]
	v.cursorX, v.cursorY = (empty.Min.X+empty.Max.X)/2, (empty.Min.Y+empty.Max.Y)/2
	if _, _, ok := v.spellPopupPresent(); ok {
		t.Error("a cell past the end of the book drew a popup")
	}

	// AN ENTRY CARRYING NO LINES draws none either, which is the state a
	// front end that was never given any pushes.
	bare := []SpellEntry{{ID: 1, Name: "Fire Arrow"}}
	v.SetSpellbook(sbUnitID, bare)
	bx, by := sbEntryPoint(t, v, bare, 0)
	v.cursorX, v.cursorY = bx, by
	if _, _, ok := v.spellPopupPresent(); ok {
		t.Error("an entry carrying no lines drew a popup")
	}

	// AND THE HOVER ISSUES NOTHING. A frame with the cursor over the bar and
	// no button at all produces no order, which is what "it gates nothing and
	// issues nothing" means for this box.
	v.SetSpellbook(sbUnitID, book)
	if ords, ok := v.command(appInput{CursorX: x, CursorY: y}); ok || len(ords) != 0 {
		t.Errorf("a hover over the bar produced %v (ok=%v), want nothing", ords, ok)
	}
}

func TestOnlyAMarkedUnitContributesASpellEffectPass(t *testing.T) {
	a := spellbookTestApp(t)
	v := a.flow.viewer
	v.SetUnits(true, nil)

	unmarked := []MapEntity{
		{ID: 1, Life: LifeAlive, Cell: image.Pt(1, 1)},
		{ID: 2, Life: LifeAlive, Cell: image.Pt(2, 2)},
	}
	v.SetEntities(unmarked)
	if got := v.spellEffectPasses(); len(got) != 0 {
		t.Errorf("a frame with no mark contributes %d pass(es), want none", len(got))
	}

	marked := append([]MapEntity(nil), unmarked...)
	marked[1].SpellFX, marked[1].SpellFXSchool = 4, 1 // fire
	v.SetEntities(marked)

	passes := v.spellEffectPasses()
	if len(passes) != 1 {
		t.Fatalf("one marked unit contributes %d pass(es), want 1", len(passes))
	}
	if passes[0].Color != SpellSchoolColors[1] {
		t.Errorf("the pass is drawn in %v, want the fire entry %v", passes[0].Color, SpellSchoolColors[1])
	}
	if len(passes[0].Rects) == 0 {
		t.Error("the pass carries no rectangle, so nothing would be drawn")
	}
}

// TestASchoolOutsideThePaletteStillDrawsAMark is SpellSchoolColors' own rule: a
// value this package cannot interpret takes entry 0 rather than dropping the
// mark, because a spell that landed and showed nothing is the worse failure.
func TestASchoolOutsideThePaletteStillDrawsAMark(t *testing.T) {
	a := spellbookTestApp(t)
	v := a.flow.viewer
	v.SetUnits(true, nil)
	v.SetEntities([]MapEntity{
		{ID: 1, Life: LifeAlive, Cell: image.Pt(1, 1), SpellFX: 2, SpellFXSchool: 99},
	})

	passes := v.spellEffectPasses()
	if len(passes) != 1 {
		t.Fatalf("a mark naming school 99 contributes %d pass(es), want 1", len(passes))
	}
	if passes[0].Color != SpellSchoolColors[0] {
		t.Errorf("the pass is drawn in %v, want the no-school entry %v",
			passes[0].Color, SpellSchoolColors[0])
	}
}
