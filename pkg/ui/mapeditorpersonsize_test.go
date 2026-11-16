package ui

import (
	"fmt"
	"image"
	"os"
	"path/filepath"
	"testing"

	"againrom/internal/witness"
)

// personCardBandScrolls checks the plain view's card band: in a room lower
// than the band it scrolls with the rows in the window's one scroll, every
// card cell shows inside the rows' viewport at some top and only there, the
// card keeps its size and the band's bottom shows at the last top; in a
// higher room it stands fixed.
func personCardBandScrolls(t *testing.T, e *MapEditor, lang string, low bool) {
	t.Helper()
	s, _, _ := e.objectSpec()
	if got := bandLines(specWindow(s)) > 0; got != low {
		t.Fatalf("the card band scrolls %v, want %v", got, low)
	}
	if !low {
		return
	}
	cells := map[int]bool{}
	for i, row := range s.Rows {
		if row.placed() {
			cells[i] = false
		}
	}
	e.ScrollDetails(-1000)
	var l popupLayout
	for range 200 {
		s, _, _ = e.objectSpec()
		l = layoutPopup(s, e.font, e.bodyFont())
		if l.RowBar.Empty() || l.Card.Size() != cardSize(specWindow(s)) {
			t.Fatalf("top %d: no scroll bar or the card %v lost its size", s.Top, l.Card)
		}
		for _, rr := range l.Rows {
			if s.Rows[rr.Index].placed() {
				if !rr.Rect.In(l.RowsViewport) {
					t.Fatalf("top %d: card cell %d stands outside the viewport", s.Top, rr.Index)
				}
				cells[rr.Index] = true
			}
		}
		before := s.Top
		if dir := os.Getenv("AGAINROM_PERSON_FRAMES"); dir != "" && (before == 0 || before%6 == 0) {
			witness.PNG(t, filepath.Join(dir, fmt.Sprintf("person-plain-%s-640x480-top%d.png", lang, before)), e.ChromeImage())
		}
		e.ScrollDetails(1)
		if s, _, _ = e.objectSpec(); s.Top == before {
			break
		}
	}
	for i, seen := range cells {
		if !seen {
			t.Errorf("card cell %d never shows", i)
		}
	}
	if len(cells) == 0 || max(l.Card.Max.Y, l.CardRest.Max.Y) > l.RowsViewport.Max.Y || l.CardRest.Dy() < e.unitFigureSize().Y {
		t.Errorf("the last top does not show the band's bottom: card %v rest %v viewport %v", l.Card, l.CardRest, l.RowsViewport)
	}
}

// Each view of the person window takes the size its rows need: the plain
// view the least width that holds its rows, card and figure, as wide as the
// room at most, its Kind row shown and changeable; the detailed view as wide
// as its columns need, the room at most, and no tabs; both every row high,
// or as high as the room with the rows scrolled.
func TestEditorPersonWindowTakesTheSizeItsRowsNeed(t *testing.T) {
	for _, lang := range []string{"en", "ru"} {
		for _, size := range []image.Point{{640, 480}, {1920, 1080}} {
			for _, plain := range []bool{true, false} {
				t.Run(fmt.Sprintf("%s/%v/plain=%v", lang, size, plain), func(t *testing.T) {
					e, _ := cardEditor(t, lang, 300, false)
					e.props.plain = plain
					e.Layout(size.X, size.Y)
					s, _, ok := e.objectSpec()
					if !ok || e.PlainView() != plain {
						t.Fatal("no person window in the asked view")
					}
					l := layoutPopup(s, e.font, e.bodyFont())
					if !l.Rect.In(e.popupBounds()) {
						t.Fatal("window leaves its bounds", l.Rect, e.popupBounds())
					}
					if len(s.Tabs) != 0 {
						t.Error("the window has tabs", s.Tabs)
					}
					need := min(max(popupMinWidth(specWindow(s), e.bodyFont()), buttonRowWidth(specWindow(s), e.bodyFont())), e.popupBounds().Dx())
					if plain && l.Rect.Dx() != need {
						t.Errorf("plain view %d wide, its rows need %d", l.Rect.Dx(), need)
					}
					if !plain && l.Rect.Dx() < min(e.unitWindowWidth()-32-e.hostedAside(s)-popupPad, s.Columns*personColumnWidth) {
						t.Errorf("detailed view %d wide, the window allows %d", l.Rect.Dx(), e.unitWindowWidth()-32)
					}
					shown := map[int]popupRowRect{}
					for _, rr := range l.Rows {
						shown[rr.Index] = rr
					}
					scrolled := s.MaxRows > 0 && popupRowCount(specWindow(s)) > s.MaxRows
					if scrolled && l.Rect.Dy() < e.popupRoom().Dy()-16 {
						t.Errorf("rows scroll in a window %d high, the room holds %d", l.Rect.Dy(), e.popupRoom().Dy()-16)
					}
					for i, row := range s.Rows {
						if _, ok := shown[i]; !ok && !scrolled && row.Kind != popupHeading {
							t.Errorf("row %d %q is not shown", i, row.Label)
						}
					}
					if plain {
						rr, ok := shown[e.fieldRow(cardFieldIndex(t, e, "identity"))]
						if !ok || rr.Control.Empty() || !rr.Control.In(l.Rect) || s.Rows[rr.Index].Kind != popupChoice {
							t.Error("the Kind row is not shown as a choice", rr)
						}
						personCardBandScrolls(t, e, lang, size.Y < 600)
					}
				})
			}
		}
	}
}
