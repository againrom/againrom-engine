package ui

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"slices"
	"testing"
)

// brushWindowsHost is the palette fixture (cellCountHost) with pictures: each
// terrain a flat swatch of its own colour, each object class a flat picture
// its code names, and two decoration objects beside the 75 trees.
type brushWindowsHost struct{ *cellCountHost }

// brushWindowsSwatch is the colour of terrain i's swatch.
func brushWindowsSwatch(i int) color.RGBA { return color.RGBA{uint8(40 + 60*i), 160, 30, 255} }

func brushWindowsFlat(w, h int, c color.RGBA) *image.RGBA {
	pic := image.NewRGBA(image.Rect(0, 0, w, h))
	draw.Draw(pic, pic.Bounds(), image.NewUniform(c), image.Point{}, draw.Src)
	return pic
}

func (h *brushWindowsHost) BrushTerrains() []BrushTerrain {
	var out []BrushTerrain
	for i, n := range []string{"Grass", "Mountain", "Water"} {
		out = append(out, BrushTerrain{Class: i, Name: n, Swatch: brushWindowsFlat(64, 64, brushWindowsSwatch(i))})
	}
	return out
}

func (h *brushWindowsHost) BrushObjects() []BrushObject {
	out := append(h.cellCountHost.BrushObjects(), BrushObject{Code: 101, Name: "Stone", Palette: PaletteObjects}, BrushObject{Code: 102, Name: "Bush", Palette: PaletteObjects})
	return out
}

func (h *brushWindowsHost) NewRecordPreview(_ RecordKind, key string, value int64) *image.RGBA {
	if key != "object" || value <= 0 {
		return nil
	}
	return brushWindowsFlat(40, 50, color.RGBA{uint8(value), 90, 200, 255})
}

// brushWindowsEditor is the palette editor at 1920x1080 in lang over a
// brushWindowsHost.
func brushWindowsEditor(t *testing.T, lang string) *MapEditor {
	t.Helper()
	e := paletteEditorIn(t, 1920, 1080, lang, nil, nil)
	h := &brushWindowsHost{e.doc.Edits.(*cellCountHost)}
	h.self, e.doc.Edits = h, h
	return e
}

// brushWindowCases open the windows of the bar's Structures, Objects and
// Trees entries and of the Height, Terrain and Bridge brushes, each by its
// own route. Sacks opens the Add window (TestMapEditorSacksEntryOpensTheAddWindow).
var brushWindowCases = []struct {
	name string
	open func(t *testing.T, e *MapEditor)
}{
	{"structures", func(t *testing.T, e *MapEditor) { chromeClick(e, barPoint(t, e, "structures")) }},
	{"objects", func(t *testing.T, e *MapEditor) { chromeClick(e, barPoint(t, e, "objects")) }},
	{"trees forest", func(t *testing.T, e *MapEditor) { chromeClick(e, barPoint(t, e, "trees")) }},
	{"trees one kind", func(t *testing.T, e *MapEditor) {
		chromeClick(e, barPoint(t, e, "trees"))
		r, ok := e.FindControl(ControlQuery{Role: ControlListButton, N: 0, K: 0})
		if !ok {
			t.Fatal("the Trees window shows no One kind button")
		}
		chromeClick(e, center(r))
	}},
	{"height", func(t *testing.T, e *MapEditor) { brushKey(t, e, "height", brushHeight) }},
	{"terrain", func(t *testing.T, e *MapEditor) { brushKey(t, e, "terrain", brushTerrain) }},
	{"bridge", func(t *testing.T, e *MapEditor) { brushKey(t, e, "bridge", brushBridge) }},
}

func brushKey(t *testing.T, e *MapEditor, key string, mode brushMode) {
	t.Helper()
	if err := e.Key(key); err != nil {
		t.Fatal(err)
	}
	if e.brush.mode != mode || e.listKind() != "brush" {
		t.Fatalf("the %s key left brush mode %d, window %q", key, e.brush.mode, e.listKind())
	}
}

// Each window of the bar's Structures, Objects and Trees entries and of the
// Height, Terrain and Bridge brushes holds the layout properties at
// 1920x1080 in English and Russian: it stands inside the screen, every row
// label, value, button caption, grid head and list row shows whole, and a
// press at the middle of each row, row button, list row, switch, button
// and grid cell reaches that part of the window through the editor's own
// press routing.
func TestMapEditorBrushWindowsHoldTheLayoutProperties(t *testing.T) {
	size := image.Pt(1920, 1080)
	for _, lang := range paletteLangs {
		for _, c := range brushWindowCases {
			t.Run(lang+"/"+c.name, func(t *testing.T) {
				e := brushWindowsEditor(t, lang)
				c.open(t, e)
				s := mustListSpec(t, e)
				holdsWindowProperties(t, e, popList, s, size)
				body := e.bodyFont()
				l := layoutPopup(s, e.font, body)
				reach := func(r image.Rectangle, want popupPart, what string) {
					t.Helper()
					if !r.In(l.Rect) {
						t.Errorf("%s %v lies outside the window %v", what, r, l.Rect)
					}
					if part, _ := l.hit(center(r)); part != want {
						t.Errorf("a press on %s %v lands on part %d, want %d", what, r, part, want)
					}
					if got := e.chromeHit(center(r)); got != popList {
						t.Errorf("a press on %s %v reaches window %d", what, r, got)
					}
				}
				for _, rr := range l.Rows {
					row := s.Rows[rr.Index]
					if row.Value != "" && !rr.Control.Empty() && body.Advance(row.Value) > rr.Control.Dx()-12 {
						t.Errorf("row %d value %q does not fit its %d wide control", rr.Index, row.Value, rr.Control.Dx())
					}
					for k, b := range rr.Buttons {
						if k < len(row.Buttons) && body.Advance(row.Buttons[k])+8 > b.Dx() {
							t.Errorf("row %d button %q does not fit its %d wide box", rr.Index, row.Buttons[k], b.Dx())
						}
						reach(b, popupRowButton, fmt.Sprintf("row %d button %d", rr.Index, k))
					}
				}
				for k, r := range l.GridRowHeads {
					head := s.Grid.RowHeads[l.GridTop+k]
					if len(l.GridHeadText) > 0 {
						head = l.GridHeadText[k]
					}
					if body.Advance(head)+2 > r.Dx() {
						t.Errorf("grid head %q does not fit its %d wide box", head, r.Dx())
					}
				}
				if l.HasList {
					for i := 0; i < l.ListRows && i < s.List.Len(); i++ {
						reach(l.List.Row(i), popupListRow, fmt.Sprintf("list row %d", i))
					}
					if !l.Aside.Empty() && !l.Aside.In(l.Rect) {
						t.Errorf("the picture column %v lies outside the window %v", l.Aside, l.Rect)
					}
					rows, text := s.List.Rows(), s.ListText
					var cut []string
					s.ListText = func(row, width int) string {
						got := text(row, width)
						if row >= 0 && row < len(rows) && got != rows[row].Text {
							cut = append(cut, rows[row].Text)
						}
						return got
					}
					e.drawWindow(image.NewRGBA(image.Rectangle{Max: size}), s)
					if len(cut) > 0 {
						t.Errorf("the painter cuts the list rows %q", cut)
					}
				}
				if !l.SwitchButton.Empty() {
					reach(l.SwitchButton, popupSwitchButton, "the switch")
				}
			})
		}
	}
}

// The palette and brush windows paint nothing after the popup painter: the
// list window's draw leaves the same pixels as drawPopup alone, and the
// picture beside the Objects, One kind and Terrain lists is drawPopup's own.
func TestMapEditorBrushWindowsPaintNothingAfterThePopup(t *testing.T) {
	for _, c := range brushWindowCases {
		t.Run(c.name, func(t *testing.T) {
			e := brushWindowsEditor(t, "en")
			c.open(t, e)
			if e.brush.mode == brushObjects && !e.forestOn() || e.brush.mode == brushTerrain {
				_ = e.Key("down") // a row with a picture
			}
			s := mustListSpec(t, e)
			pictured := c.name == "objects" || c.name == "trees one kind" || c.name == "terrain"
			if (s.ListPicture != nil) != pictured {
				t.Fatalf("the window shows a picture beside its list %v, want %v", s.ListPicture != nil, pictured)
			}
			whole := image.NewRGBA(image.Rect(0, 0, 1920, 1080))
			e.drawListWindow(whole, s)
			alone := image.NewRGBA(whole.Rect)
			press := e.buttonPress
			drawPopup(alone, e.font, e.bodyFont(), nil, s, layoutPopup(s, e.font, e.bodyFont()), editorPopupInk, e.pointer, e.pointerOK, &press)
			if !bytes.Equal(whole.Pix, alone.Pix) {
				t.Fatal("the window's draw paints past the popup painter")
			}
			if p := s.ListPicture; p != nil {
				box := layoutPopup(s, e.font, e.bodyFont()).Aside
				want := p.RGBAAt(p.Rect.Dx()/2, p.Rect.Dy()/2)
				if got := alone.RGBAAt(center(box).X, center(box).Y); got != want {
					t.Fatalf("drawPopup draws %v in the picture column, want the picture's %v", got, want)
				}
			}
		})
	}
}

// The Objects window has no Tool row and no Forest; the Trees
// entry opens Forest, its first row the tool row.
func TestMapEditorObjectsWindowLeavesForestToTrees(t *testing.T) {
	for _, lang := range paletteLangs {
		e := brushWindowsEditor(t, lang)
		tool := e.word("editor.brush.tool")
		chromeClick(e, barPoint(t, e, "objects"))
		for _, r := range mustListSpec(t, e).Rows {
			if r.Label == tool {
				t.Errorf("%s: the Objects window shows the tool row %q", lang, r.Label)
			}
		}
		if e.forestOn() || e.brush.mode != brushObjects {
			t.Errorf("%s: the Objects entry left Forest %v, brush mode %d", lang, e.forestOn(), e.brush.mode)
		}
		chromeClick(e, barPoint(t, e, "trees"))
		if rows := mustListSpec(t, e).Rows; !e.forestOn() || len(rows) == 0 || rows[0].Label != tool {
			t.Errorf("%s: the Trees entry left Forest %v with rows %+v", lang, e.forestOn(), rows)
		}
	}
}

// Object and structure names stay the install's registry names in
// both languages: the Structures grid's tips, the Objects and the One kind
// lists show the host's names as it gives them.
func TestMapEditorPlaceNamesStayTheRegistryNames(t *testing.T) {
	for _, lang := range paletteLangs {
		e := brushWindowsEditor(t, lang)
		chromeClick(e, barPoint(t, e, "structures"))
		if tips := gridTips(t, e); len(tips) != 66 || tips[0] != "1: Kind 1" || tips[65] != "66: Kind 66" {
			t.Errorf("%s: the Structures grid tips are %d, first %q", lang, len(tips), tips[:min(1, len(tips))])
		}
		chromeClick(e, barPoint(t, e, "objects"))
		if rows := e.brushRows(); len(rows) != 3 || !slices.Equal(rows[1:], []string{"Stone", "Bush"}) {
			t.Errorf("%s: the Objects list is %q", lang, rows)
		}
		chromeClick(e, barPoint(t, e, "trees"))
		e.brushRowButton(0, 0)
		if rows := e.brushRows(); len(rows) != 76 || rows[1] != "Tree 1" || rows[75] != "Tree 75" {
			t.Errorf("%s: the One kind list holds %d rows", lang, len(rows))
		}
	}
}

// The Sacks entry opens the Add window for a sack at once, with no
// palette between; its second press closes it.
func TestMapEditorSacksEntryOpensTheAddWindow(t *testing.T) {
	for _, lang := range paletteLangs {
		e := brushWindowsEditor(t, lang)
		chromeClick(e, barPoint(t, e, "sacks"))
		if _, ok := e.listSpec(); ok || addKindNow(e) != string(KindSack) || e.barWindow() != "sacks" {
			t.Fatalf("%s: the Sacks entry shows a palette %v, Add window %q, bar window %q", lang, ok, addKindNow(e), e.barWindow())
		}
		chromeClick(e, barPoint(t, e, "sacks"))
		if addKindNow(e) != "" || e.barWindow() != "" {
			t.Fatalf("%s: the second press left the Add window %q", lang, addKindNow(e))
		}
	}
}

// Forest ticks kinds in its grid and shows no list; One kind shows
// the list and no grid.
func TestMapEditorForestTicksInTheGridOnly(t *testing.T) {
	for _, lang := range paletteLangs {
		e := brushWindowsEditor(t, lang)
		chromeClick(e, barPoint(t, e, "trees"))
		s := mustListSpec(t, e)
		if s.List != nil || s.Grid == nil {
			t.Fatalf("%s: the Forest window shows a list %v, a grid %v", lang, s.List != nil, s.Grid != nil)
		}
		r, ok := e.FindControl(ControlQuery{Role: ControlPlaceGridCell, N: 0})
		if !ok {
			t.Fatalf("%s: the Forest grid shows no cell", lang)
		}
		chromeClick(e, center(r))
		if got := e.forestTicked(); len(got) != 1 {
			t.Fatalf("%s: a grid press ticked %v", lang, got)
		}
		e.brushRowButton(0, 0)
		if s := mustListSpec(t, e); s.List == nil || s.Grid != nil {
			t.Fatalf("%s: the One kind window shows a list %v, a grid %v", lang, s.List != nil, s.Grid != nil)
		}
	}
}

// The Forest grid's cell is 80 px, the Structures grid's: its side
// in the spec, and the same box in the layout.
func TestMapEditorForestGridCellIsEightyPixels(t *testing.T) {
	for _, lang := range paletteLangs {
		e := brushWindowsEditor(t, lang)
		_, structures := paletteGrid(t, e, "structures")
		s, forest := paletteGrid(t, e, "trees")
		if s.Grid.side() != 80 || len(forest.GridCells) == 0 || forest.GridCells[0].Size() != structures.GridCells[0].Size() {
			t.Fatalf("%s: the Forest cell side is %d, its box %v, the Structures box %v", lang, s.Grid.side(), forest.GridCells[0].Size(), structures.GridCells[0].Size())
		}
	}
}

// The Height window keeps no picture beside its list.
func TestMapEditorHeightShowsNoPicture(t *testing.T) {
	for _, lang := range paletteLangs {
		e := brushWindowsEditor(t, lang)
		brushKey(t, e, "height", brushHeight)
		s := mustListSpec(t, e)
		if l := layoutPopup(s, e.font, e.bodyFont()); s.ListAside != 0 || s.ListPicture != nil || !l.Aside.Empty() {
			t.Fatalf("%s: the Height window keeps a picture column %d (%v)", lang, s.ListAside, l.Aside)
		}
	}
}

// The Bridge window is titled by its name with its how-to in the title's
// (?), and its Shape row reads Rectangle with no button; it shows no list.
func TestMapEditorBridgeWindowShowsItsRectangle(t *testing.T) {
	for _, lang := range paletteLangs {
		e := brushWindowsEditor(t, lang)
		brushKey(t, e, "bridge", brushBridge)
		s := mustListSpec(t, e)
		if s.Title != e.word("editor.brush.bridge") || s.TitleHelp != e.word("editor.brush.bridge.help") || s.List != nil {
			t.Fatalf("%s: the Bridge window reads %q (?) %q, list %v", lang, s.Title, s.TitleHelp, s.List != nil)
		}
		if len(s.Rows) != 2 || s.Rows[1].Value != e.word("editor.brush.rectangle") || len(s.Rows[1].Buttons) != 0 {
			t.Fatalf("%s: the Bridge rows are %+v", lang, s.Rows)
		}
	}
}

// The Terrain window shows the chosen ground's swatch as a large
// sample beside its list, through the list's picture, and it follows the
// choice.
func TestMapEditorTerrainShowsTheChosenGround(t *testing.T) {
	for _, lang := range paletteLangs {
		e := brushWindowsEditor(t, lang)
		brushKey(t, e, "terrain", brushTerrain)
		for sel := 0; sel < 3; sel++ {
			s := mustListSpec(t, e)
			l := layoutPopup(s, e.font, e.bodyFont())
			if s.ListPicture == nil || l.Aside.Dx() < brushPreviewW {
				t.Fatalf("%s: terrain %d shows no sample (%d wide)", lang, sel, l.Aside.Dx())
			}
			pic := image.NewRGBA(image.Rect(0, 0, 1920, 1080))
			e.drawListWindow(pic, s)
			if got := pic.RGBAAt(center(l.Aside).X, center(l.Aside).Y); got != brushWindowsSwatch(sel) {
				t.Fatalf("%s: terrain %d's sample draws %v, want %v", lang, sel, got, brushWindowsSwatch(sel))
			}
			_ = e.Key("down")
		}
	}
}
