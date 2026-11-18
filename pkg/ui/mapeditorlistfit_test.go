package ui

import (
	"image"
	"testing"
	"time"
)

// listFitWidths are the widths the Players window and each brush window
// have at 1920x1080, by language and window.
var listFitWidths = map[string]int{
	"en/structures": 400, "en/objects": 248, "en/trees forest": 518, "en/trees one kind": 248,
	"en/height": 234, "en/terrain": 259, "en/bridge": 218, "en/players": 608,
	"ru/structures": 400, "ru/objects": 326, "ru/trees forest": 518, "ru/trees one kind": 296,
	"ru/height": 274, "ru/terrain": 274, "ru/bridge": 250, "ru/players": 648,
}

// The Players window and each brush window keep the width the list-window
// fit gives them, in English and Russian at 1920x1080.
func TestBrushAndPlayersWindowsKeepTheirWidth(t *testing.T) {
	size := image.Pt(1920, 1080)
	for _, lang := range paletteLangs {
		for _, c := range brushWindowCases {
			t.Run(lang+"/"+c.name, func(t *testing.T) {
				e := brushWindowsEditor(t, lang)
				c.open(t, e)
				s := mustListSpec(t, e)
				checkListFitWidth(t, lang+"/"+c.name, s.Width)
			})
		}
		t.Run(lang+"/players", func(t *testing.T) {
			e := playersEditor(t, lang, size, ninePlayers)
			if err := e.Key("players"); err != nil {
				t.Fatal(err)
			}
			checkListFitWidth(t, lang+"/players", mustListSpec(t, e).Width)
		})
	}
}

func checkListFitWidth(t *testing.T, name string, got int) {
	t.Helper()
	if want, ok := listFitWidths[name]; !ok || got != want {
		t.Errorf("%s: width %d, want %d (kept: %v)", name, got, want, ok)
	}
}

// The mockup gives a flow grid's row head the width the editor's grids give
// it (sizeGrid): the head's text and 8.
func TestDescriptionGridHeadWidthIsTheEditorsRule(t *testing.T) {
	for _, lang := range []string{"en", "ru"} {
		font := wordsFont(lang)
		m, err := newEditorMockup([]byte(listDesc), lang, image.Pt(800, 600), font, font, nil, mockupPerson{picture: redPicture})
		if err != nil {
			t.Fatal(err)
		}
		g := m.plain.Grid
		if g == nil || len(g.RowHeads) != 1 {
			t.Fatalf("%s: grid %+v", lang, g)
		}
		want := *g
		sizeGrid(&want, 0, font, func(s string) string { return s }, image.Pt(800, 600))
		if g.HeadW != want.HeadW {
			t.Errorf("%s: the mockup's Mode head is %d wide, the editor's rule gives %d", lang, g.HeadW, want.HeadW)
		}
	}
}

// A wheel over a brush window whose list is hidden (the Trees window's
// Forest mode) leaves the list where it is; over a visible list it still
// moves the highlight.
func TestBrushWheelLeavesAHiddenListAlone(t *testing.T) {
	for _, c := range []struct {
		name string
		move bool
	}{{"trees one kind", false}, {"objects", true}} {
		var open func(*testing.T, *MapEditor)
		for _, w := range brushWindowCases {
			if w.name == c.name {
				open = w.open
			}
		}
		e := brushWindowsEditor(t, "en")
		open(t, e)
		mustListSpec(t, e)
		if e.model == nil {
			t.Fatalf("%s: no list model", c.name)
		}
		if !c.move {
			// Forest mode: the One kind list is hidden but kept.
			r, ok := e.FindControl(ControlQuery{Role: ControlListButton, N: 0, K: 1})
			if !ok {
				t.Fatal("the Trees window shows no Forest button")
			}
			chromeClick(e, center(r))
			mustListSpec(t, e)
			if !e.forestOn() || e.model == nil {
				t.Fatalf("forest on %v, list kept %v", e.forestOn(), e.model != nil)
			}
		}
		l := layoutPopup(mustListSpec(t, e), e.font, e.bodyFont())
		at := l.Rect.Min.Add(image.Pt(6, 6))
		moved := false
		for _, dy := range []float64{-1, -1, -1} {
			before := e.model.Selection()
			e.Pointer(Input{CursorX: at.X, CursorY: at.Y, WheelY: dy}, time.Unix(1, 0))
			e.ChromeImage()
			moved = moved || e.model.Selection() != before
		}
		if moved != c.move {
			t.Errorf("%s: the wheel moved the list: %v, want %v", c.name, moved, c.move)
		}
	}
}

// The pictures the brush windows read through the place icon cache go with
// the document: a map shown next reads its own.
func TestPlaceIconsLeaveWithTheDocument(t *testing.T) {
	e := brushWindowsEditor(t, "en")
	if e.placeTipPicture("brush", "object", 1) == nil || len(e.place.icons) == 0 {
		t.Fatal("the host gave no picture to keep")
	}
	e.showDoc(e.doc, nil)
	if len(e.place.icons) != 0 {
		t.Errorf("%d pictures of the map left stay in the cache", len(e.place.icons))
	}
}
