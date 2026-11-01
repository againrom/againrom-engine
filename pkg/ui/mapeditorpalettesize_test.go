package ui

import (
	"bytes"
	"fmt"
	"image"
	"testing"

	"againrom/pkg/words"
)

var paletteSizes = []image.Point{{640, 480}, {1024, 768}, {1280, 800}, {1920, 1080}}

var paletteLangs = []string{"en", "ru"}

// paletteEditorIn is paletteEditor in language lang, with the profile bytes
// every save gives appended to saved.
func paletteEditorIn(t *testing.T, w, h int, lang string, profile []byte, saved *[][]byte) *MapEditor {
	t.Helper()
	e := paletteEditor(t, w, h)
	e.SetWords(words.For(lang))
	var save func([]byte) error
	if saved != nil {
		save = func(b []byte) error { *saved = append(*saved, bytes.Clone(b)); return nil }
	}
	e.SetPanelProfile(profile, nil, save)
	e.Layout(w, h)
	e.places[popList].placed = false
	return e
}

// closePalette closes the shown palette window by its bar entry.
func closePalette(t *testing.T, e *MapEditor, key string) {
	t.Helper()
	chromeClick(e, center(layoutPopup(mustListSpec(t, e), e.font, e.bodyFont()).Close))
	if _, ok := e.listSpec(); ok {
		t.Fatalf("the %s window did not close", key)
	}
}

// gripDrag drags the front popup's bottom-right grip by d.
func gripDrag(t *testing.T, e *MapEditor, d image.Point) image.Rectangle {
	t.Helper()
	_, after := edgeDrag(t, e, func(r image.Rectangle) image.Point { return image.Pt(r.Max.X-3, r.Max.Y-3) }, d)
	return after
}

func paletteRect(t *testing.T, e *MapEditor, key string) image.Rectangle {
	t.Helper()
	_, l := paletteGrid(t, e, key)
	return l.Rect
}

// The Units and Structures windows open at the list popup's own width, or the
// least that shows six columns where that is more, and no wider; every cell
// is reached by scrolling.
func TestMapEditorPaletteOpensAtANormalWidth(t *testing.T) {
	for _, lang := range paletteLangs {
		for _, sz := range paletteSizes {
			e := paletteEditorIn(t, sz.X, sz.Y, lang, nil, nil)
			for _, key := range []string{"units", "structures"} {
				s, l := paletteGrid(t, e, key)
				g := s.Grid
				six := 2*popupPad + gridLeastCols*g.side() + widgetSpriteSize + 2
				room := e.mapArea()
				most := min(max(s.Width, min(six, room.Dx())), e.popupBounds().Dx())
				name := fmt.Sprintf("%s %s %v", lang, key, sz)
				if l.Rect.Dx() != most {
					t.Errorf("%s: width %d, want %d (own %d, six columns %d)", name, l.Rect.Dx(), most, s.Width, six)
				}
				if oneLine := 2*popupPad + len(g.Cells)*g.side() + widgetSpriteSize + 2; most < oneLine && l.Rect.Dx() >= oneLine {
					t.Errorf("%s: width %d is the one-line width", name, l.Rect.Dx())
				}
				if l.Rect.Dx() >= six && l.GridCols < gridLeastCols {
					t.Errorf("%s: %d columns at width %d", name, l.GridCols, l.Rect.Dx())
				}
				t.Logf("%s: width %d, %d columns, %d rows, bounds %d", name, l.Rect.Dx(), l.GridCols, l.GridRows, e.popupBounds().Dx())
				seen := map[int]bool{}
				for top := 0; top <= l.GridRows+len(g.Cells); top++ {
					e.place.top = top
					ll := layoutPopup(mustListSpec(t, e), e.font, e.bodyFont())
					for i := range ll.GridCells {
						seen[ll.GridFirst+i] = true
					}
				}
				e.place.top = 0
				if len(seen) != len(g.Cells) {
					t.Errorf("%s: scrolling reaches %d of %d cells", name, len(seen), len(g.Cells))
				}
			}
		}
	}
}

// A size dragged on the grip is kept across close and reopen, for each window
// on its own.
func TestMapEditorPaletteKeepsItsSizeAcrossReopen(t *testing.T) {
	for _, lang := range paletteLangs {
		for _, sz := range paletteSizes {
			e := paletteEditorIn(t, sz.X, sz.Y, lang, nil, nil)
			name := fmt.Sprintf("%s %v", lang, sz)
			def := map[string]image.Rectangle{}
			got := map[string]image.Rectangle{}
			for i, key := range []string{"units", "structures"} {
				def[key] = paletteRect(t, e, key)
				got[key] = gripDrag(t, e, image.Pt(-def[key].Dx()/(5+i), -def[key].Dy()/(4+i)))
				if got[key].Size() == def[key].Size() {
					t.Fatalf("%s %s: the drag left the size %v", name, key, got[key].Size())
				}
			}
			if got["units"].Size() == got["structures"].Size() {
				t.Fatalf("%s: both windows dragged to %v", name, got["units"].Size())
			}
			for _, key := range []string{"units", "structures", "units"} {
				closePalette(t, e, key)
				if r := paletteRect(t, e, key); r.Size() != got[key].Size() {
					t.Errorf("%s %s: reopened at %v, want %v", name, key, r.Size(), got[key].Size())
				}
			}
		}
	}
}

// The profile bytes written after a resize give a new editor the same size;
// a profile without the entry, with a broken one, or of the panels only reads
// as the default.
func TestMapEditorPaletteSizeLivesInTheProfile(t *testing.T) {
	for _, lang := range paletteLangs {
		for _, sz := range paletteSizes {
			var saved [][]byte
			e := paletteEditorIn(t, sz.X, sz.Y, lang, nil, &saved)
			name := fmt.Sprintf("%s %v", lang, sz)
			def := map[string]image.Rectangle{}
			got := map[string]image.Rectangle{}
			for i, key := range []string{"units", "structures"} {
				def[key] = paletteRect(t, e, key)
				got[key] = gripDrag(t, e, image.Pt(-def[key].Dx()/(5+i), -def[key].Dy()/(4+i)))
			}
			if len(saved) == 0 {
				t.Fatalf("%s: no profile written", name)
			}
			profile := saved[len(saved)-1]
			n := paletteEditorIn(t, sz.X, sz.Y, lang, profile, nil)
			for _, key := range []string{"units", "structures"} {
				if r := paletteRect(t, n, key); r.Size() != got[key].Size() {
					t.Errorf("%s %s: new editor opens at %v, want %v (profile %s)", name, key, r.Size(), got[key].Size(), profile)
				}
			}
			for _, c := range []struct{ what, data string }{
				{"panels only", `{"left":{"minimized":false,"width":260},"right":{"minimized":false,"width":520,"view":"plain"}}`},
				{"no windows", `{"left":{"minimized":false,"width":260}}`},
				{"windows not an object", `{"left":{"minimized":false,"width":260},"windows":5}`},
				{"entry not an object", `{"windows":{"units":"wide","structures":[1,2]}}`},
				{"entry without a height", `{"windows":{"units":{"width":300},"structures":{"width":300,"height":-4}}}`},
				{"not json", `{"windows":`},
			} {
				n := paletteEditorIn(t, sz.X, sz.Y, lang, []byte(c.data), nil)
				for _, key := range []string{"units", "structures"} {
					if r := paletteRect(t, n, key); r.Size() != def[key].Size() {
						t.Errorf("%s %s, %s: opens at %v, want the default %v", name, key, c.what, r.Size(), def[key].Size())
					}
				}
			}
		}
	}
}

// A broken windows entry costs the panels nothing.
func TestMapEditorBrokenWindowEntryKeepsThePanelState(t *testing.T) {
	data := []byte(`{"left":{"minimized":true,"width":300},"right":{"minimized":false,"width":600,"view":"technical"},"windows":5}`)
	e := paletteEditor(t, 1280, 800)
	e.SetPanelProfile(data, nil, nil)
	if s := e.panels.state[panelLeft]; s.Restored || s.Width != 300 {
		t.Fatalf("left panel %+v", s)
	}
	if s := e.panels.state[panelRight]; !s.Restored || s.Width != 600 || e.props.plain {
		t.Fatalf("right panel %+v, plain %v", s, e.props.plain)
	}
}
