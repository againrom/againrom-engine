package ui

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"testing"

	"againrom/pkg/words"
)

// spellsHost is a mod host that gives the spells page's fields as the game's
// host shapes them: the chosen spell's numbers, the unfolded globals, a
// catalog of 32 spells with four of the other game, and a book of 24 cells
// with one empty.
type spellsHost struct {
	*modesWorkspace
	lang string
}

// spellsCatalog and spellsBookCells are the fixture's catalog and book sizes.
const spellsCatalog, spellsBookCells = 32, 24

func (h spellsHost) WorkspaceFields(scope string) []InspectionField {
	if scope != ModSpellsScope {
		return h.modesWorkspace.WorkspaceFields(scope)
	}
	b := words.For(h.lang)
	out := []InspectionField{{Key: "chosen", ReadOnly: true, Text: true, Lines: []InspectionLine{{Text: "Spell: Fire Arrow"}}}}
	for _, key := range []string{"mana", "range", "radius", "duration", "areaduration", "damage", "cell"} {
		out = append(out, InspectionField{Key: "spell." + key, Label: b.Text("editor.spells." + key), Text: true, Inherit: true, Inherited: "12"})
	}
	out = append(out, InspectionField{Key: "spell.off", Label: b.Text("editor.spells.off"), Value: 1,
		Choices: []InspectionChoice{{Value: 1, Label: "no"}, {Value: 2, Label: "off"}}})
	out = append(out, InspectionField{Key: "globals", Label: b.Text("editor.spells.all"), Action: true, Column: 1})
	for _, key := range []string{"damage", "heal", "mana", "range", "radius", "duration", "booksize"} {
		out = append(out, InspectionField{Key: "global." + key, Label: b.Text("editor.spells." + key), Text: true, Inherit: true, Inherited: "1", Column: 1})
	}
	grid := InspectionField{Key: "catalog", Label: b.Text("editor.spells.catalog"), Grid: true, Small: true, Marks: map[int64]string{}}
	for i := range spellsCatalog {
		v := int64(i + 1)
		grid.Choices = append(grid.Choices, InspectionChoice{Value: v, Label: fmt.Sprintf("Spell number %d", v)})
		grid.Marks[v] = "SN"
		if i >= spellsCatalog-4 {
			grid.Shut = append(grid.Shut, v)
		} else if i < spellsBookCells-1 {
			grid.Ticked = append(grid.Ticked, v)
		}
	}
	book := &InspectionBook{Grow: "The book holds 24 cells", Remove: "remove", Drops: []InspectionDrop{
		{From: DropBook, To: DropBook, Word: "move"},
		{From: DropGrid, To: DropBook, Word: "add"},
		{From: DropBook, To: DropGrid, Word: "remove"},
	}}
	for i := range spellsBookCells {
		cell := SpellEntry{ID: uint32(i + 1), Name: fmt.Sprintf("Spell number %d", i+1)}
		if i == spellsBookCells-1 {
			cell = SpellEntry{}
		}
		book.Cells = append(book.Cells, cell)
		book.Tips = append(book.Tips, cell.Name)
	}
	return append(out, grid, InspectionField{Key: "book", Label: b.Text("editor.spells.book"), Book: book})
}

// A book with the game's art is the game's book painter's picture, pixel for
// pixel, and each cell's box is the painter's cell, bare over the picture.
func TestSpellsPageBookIsTheGamesBookPainter(t *testing.T) {
	e, _ := newModesEditor(t, "workspace")
	green, hidden := color.RGBA{G: 200, A: 255}, color.RGBA{R: 90, A: 255}
	art := &BottomHUDArt{Book: solidPic(480, 85, green), UnknownSpell: solidPic(36, 36, hidden)}
	book := &InspectionBook{Art: art, Chosen: 3}
	for i := range spellsBookCells {
		book.Cells = append(book.Cells, SpellEntry{ID: uint32(i + 1), Name: fmt.Sprintf("Spell number %d", i+1)})
	}
	book.Cells[5] = SpellEntry{}
	c, _ := e.modBook(book, 900)
	if c.Picture == nil {
		t.Fatal("the book has no picture")
	}
	bar := image.Rect(0, 0, c.Picture.Rect.Dx(), c.Picture.Rect.Dy())
	want := composeOriginalSpellBar(art, book.Cells, 3, bar, 0)
	if !bytes.Equal(c.Picture.Pix, want.Pix) {
		t.Fatal("the book's picture is not the game's book painter's")
	}
	cells := bookCellRects(bar, bookColumns)
	for i, box := range c.Boxes[:spellsBookCells] {
		if !box.Bare || box.Area != cells[i].Add(image.Pt(1, 1)) || box.Empty != (i == 5) {
			t.Fatalf("box %d: bare %v, area %v, empty %v; the painter's cell %v", i, box.Bare, box.Area, box.Empty, cells[i])
		}
	}
}

// At 1920x1080 in both languages the spells page lies inside the screen,
// every catalog cell and book cell lies inside the window, a press at a
// cell's centre reaches that cell, an other-game cell takes no press, and
// every label is drawn whole.
func TestSpellsPageFitsTheScreenAndEveryCellTakesItsPress(t *testing.T) {
	screen := image.Rect(0, 0, 1920, 1080)
	for _, lang := range []string{"en", "ru"} {
		e := modesWordsEditor(t, "workspace", lang, screen.Max)
		e.doc.Edits = spellsHost{e.doc.Edits.(*modesWorkspace), lang}
		if err := e.Key("package"); err != nil {
			t.Fatal(err)
		}
		tabs, _ := e.modTabs()
		for i := range tabs {
			if e.modPage == modPageSpells {
				break
			}
			e.turnModPage(i)
		}
		if e.modPage != modPageSpells || e.modFieldScope() != ModSpellsScope {
			t.Fatalf("%s: page %d scope %q, tabs %v", lang, e.modPage, e.modFieldScope(), tabs)
		}
		s, _, ok := e.objectSpec()
		if !ok {
			t.Fatalf("%s: no window", lang)
		}
		l := layoutPopup(s, e.font, e.bodyFont())
		if !l.Rect.In(screen) {
			t.Errorf("%s: window %v outside the screen", lang, l.Rect)
		}
		if l.GridFirst != 0 || len(l.GridCells) != spellsCatalog {
			t.Fatalf("%s: catalog shows %d cells from %d, want all %d", lang, len(l.GridCells), l.GridFirst, spellsCatalog)
		}
		for i, c := range l.GridCells {
			if !c.In(l.Rect) {
				t.Errorf("%s: catalog cell %d %v outside the window %v", lang, i, c, l.Rect)
			}
			mid := image.Pt((c.Min.X+c.Max.X)/2, (c.Min.Y+c.Max.Y)/2)
			part, n := l.hit(mid)
			if shut := i >= spellsCatalog-4; shut {
				if part == popupGridPart {
					t.Errorf("%s: the other game's cell %d takes a press", lang, i)
				}
				continue
			}
			if part != popupGridPart || n != i {
				t.Errorf("%s: a press on catalog cell %d reaches %v %d", lang, i, part, n)
			}
		}
		if len(l.CanvasBoxes) < spellsBookCells {
			t.Fatalf("%s: the book shows %d boxes, want %d cells", lang, len(l.CanvasBoxes), spellsBookCells)
		}
		if !l.Canvas.In(l.Rect) {
			t.Errorf("%s: book %v outside the window %v", lang, l.Canvas, l.Rect)
		}
		for i, box := range l.CanvasBoxes {
			if !box.rect.In(l.Canvas) {
				t.Errorf("%s: book box %d %v outside the book %v", lang, i, box.rect, l.Canvas)
			}
			if i >= spellsBookCells {
				continue
			}
			mid := image.Pt((box.rect.Min.X+box.rect.Max.X)/2, (box.rect.Min.Y+box.rect.Max.Y)/2)
			if part, n := l.hit(mid); part != popupCanvasBox || n != i {
				t.Errorf("%s: a press on book cell %d reaches %v %d", lang, i, part, n)
			}
		}
		for _, rr := range l.Rows {
			row := s.Rows[rr.Index]
			if row.Label == "" || row.Kind == popupHeading {
				continue
			}
			if !rr.Rect.In(l.Rect) {
				t.Errorf("%s: row %q %v outside the window", lang, row.Label, rr.Rect)
			}
			if editorClip(e.bodyFont(), row.Label, rr.Label.Dx()-2) != row.Label {
				t.Errorf("%s: label %q is cut to %d pixels", lang, row.Label, rr.Label.Dx()-2)
			}
		}
	}
}
