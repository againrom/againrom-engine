package ui

import (
	"fmt"
	"image"
	"strconv"
	"testing"
	"time"

	"againrom/pkg/words"
)

type placementLayoutHost struct {
	*modesFull
	templates InspectionNewStep
	power     InspectionNewStep
}

func (h *placementLayoutHost) NewRecordStep(r InspectionNewRecord, key string) InspectionNewStep {
	if key == "template" {
		return h.templates
	}
	if key == "power" && h.power.Bounds != nil {
		return h.power
	}
	if key != "kind" && key != "item" && key != "spell" {
		return h.modesFull.NewRecordStep(r, key)
	}
	offer := InspectionNewStep{Value: 1}
	for i := 1; i <= 250; i++ {
		v := int64(i)
		offer.Choices = append(offer.Choices, InspectionChoice{Value: v, Label: fmt.Sprintf("Choice %d", i)})
		if key == "kind" {
			if offer.Palettes == nil {
				offer.Palettes = map[int64]Palette{}
			}
			offer.Palettes[v] = []Palette{PaletteStructures, PaletteObjects}[i%2]
		}
		if key == "item" {
			if (i-1)%10 == 0 {
				offer.Groups = append(offer.Groups, InspectionChoiceGroup{Block: fmt.Sprintf("Folder %d", (i+9)/10), Head: "Items"})
			}
			group := &offer.Groups[len(offer.Groups)-1]
			group.Values = append(group.Values, v)
		}
	}
	return offer
}

func newPlacementLayoutEditor(t *testing.T, lang string) *MapEditor {
	e, _ := newModesEditor(t, "full")
	book := words.For(lang)
	h := &placementLayoutHost{modesFull: e.doc.Edits.(*modesFull), templates: InspectionNewStep{
		Value: -30,
		Choices: []InspectionChoice{
			{Value: -30, Label: book.Text("editor.cast.where.cell")},
			{Value: -70, Label: book.Text("editor.cast.where.unit")},
			{Value: -80, Label: book.Text("editor.cast.where.structure")},
		},
		Templates: map[int64]InspectionSpellTemplate{
			-30: {Constructor: -30}, -70: {Constructor: -70, Target: KindUnit}, -80: {Constructor: -80, Target: KindStructure},
		},
	}}
	h.self, e.doc.Edits = h, h
	e.SetWords(book)
	e.Layout(1920, 1080)
	return e
}

func TestChangedPlacementPalettesFitTheEditorTarget(t *testing.T) {
	for _, lang := range []string{"en", "ru"} {
		for _, key := range []string{"structures", "objects", "sacks", "cellspells"} {
			t.Run(lang+"/"+key, func(t *testing.T) {
				e := newPlacementLayoutEditor(t, lang)
				s, l := paletteGrid(t, e, key)
				if l.GridBar.Empty() {
					t.Fatal("the fixture needs more than one page")
				}
				holdsWindowProperties(t, e, popList, s, image.Pt(1920, 1080))
			})
		}
	}
}

func TestSackPaletteTicksItemsInOneGridWithFolderHeads(t *testing.T) {
	e := newPlacementLayoutEditor(t, "en")
	s, l := paletteGrid(t, e, "sacks")
	if len(s.Tabs) != 0 || len(s.Grid.Groups) != 25 {
		t.Fatalf("the sack grid has %d tabs and %d folder heads", len(s.Tabs), len(s.Grid.Groups))
	}
	for i, group := range s.Grid.Groups {
		if group.Head != fmt.Sprintf("Folder %d", i+1) || len(group.Cells) != 10 {
			t.Fatalf("folder %d = %+v", i+1, group)
		}
	}
	for _, n := range []int{0, 1} {
		r, ok := e.FindControl(ControlQuery{Role: ControlPlaceGridCell, N: n})
		if !ok {
			t.Fatalf("item %d has no cell in %+v", n+1, l.Grid)
		}
		chromeClick(e, center(r))
	}
	s = mustListSpec(t, e)
	if !s.Grid.Cells[0].Ticked || !s.Grid.Cells[1].Ticked || !e.addPicking() {
		t.Fatal("both items should be ticked for placement")
	}
	got := e.addPlacement(image.Pt(16, 16))
	if len(got.Values) != 2 || got.Values[0].Value != 1 || got.Values[1].Value != 2 {
		t.Fatalf("next sack holds %+v", got.Values)
	}
}

func TestSpellPaletteShowsNumbersAndHoverNamesWithoutBookIcons(t *testing.T) {
	e := newPlacementLayoutEditor(t, "en")
	s, l := paletteGrid(t, e, "cellspells")
	for _, id := range []int{11, 17, 27, 28} {
		cell := s.Grid.Cells[id-1]
		if cell.Icon != nil || cell.Label != strconv.Itoa(id) || cell.Tip != fmt.Sprintf("Choice %d", id) {
			t.Fatalf("spell %d shows %+v", id, cell)
		}
		q := ControlQuery{Role: ControlPlaceGridCell, N: id - 1}
		at, ok := e.FindControl(q)
		if !ok {
			e.placeGridWheel(l, (id-1)/l.GridCols-l.GridTop-l.GridRows+1)
			l = layoutPopup(mustListSpec(t, e), e.font, e.bodyFont())
			at, ok = e.FindControl(q)
		}
		tip, shown := e.tipAt(center(at))
		if !ok || !shown || len(tip.Lines) != 1 || tip.Lines[0] != cell.Tip {
			t.Fatalf("spell %d tip: %+v, %v", id, tip, shown)
		}
	}
}

func TestSpellPaletteUsesRussianRecordTerms(t *testing.T) {
	e := newPlacementLayoutEditor(t, "ru")
	e.SetHostTextEncoder(func(s string) string { return "font:" + s })
	h := e.doc.Edits.(*placementLayoutHost)
	for i := range h.templates.Choices {
		c := &h.templates.Choices[i]
		c.Label = e.editorHostText(c.Label)
	}
	paletteGrid(t, e, "cellspells")
	for _, want := range []string{"Отряд: его заклинание", "Постройка: она колдует"} {
		r, ok := e.FindControl(ControlQuery{Role: ControlListButton, N: 0, K: 1})
		if !ok {
			t.Fatal("the Where row has no next button")
		}
		chromeClick(e, center(r))
		s := mustListSpec(t, e)
		if s.Rows[0].Value != e.editorHostText(want) {
			t.Fatalf("Where = %q, want %q", s.Rows[0].Value, want)
		}
	}
}

// cellCountHost offers 33 unit identities, and as many structure kinds and
// tree kinds as the shipped first-game data does: 66 and 75.
type cellCountHost struct {
	*modesFull
	units, structures, trees int
}

func (h *cellCountHost) NewRecordStep(r InspectionNewRecord, key string) InspectionNewStep {
	n := 0
	switch {
	case r.Kind == "Unit" && key == "identity":
		n = h.units
	case r.Kind == "Structure" && key == "kind":
		n = h.structures
	default:
		return h.modesFull.NewRecordStep(r, key)
	}
	s := InspectionNewStep{Value: 1}
	for v := 1; v <= n; v++ {
		s.Choices = append(s.Choices, InspectionChoice{Value: int64(v), Label: fmt.Sprintf("%d: Kind %d", v, v)})
	}
	return s
}

func (h *cellCountHost) BrushObjects() []BrushObject {
	out := []BrushObject{{Code: 0, Name: "No object"}}
	for c := 1; c <= h.trees; c++ {
		out = append(out, BrushObject{Code: c, Name: fmt.Sprintf("Tree %d", c), Palette: PaletteTrees})
	}
	return out
}

func paletteEditor(t *testing.T, w, h int) *MapEditor {
	t.Helper()
	e, _ := newModesEditor(t, "full")
	host := &cellCountHost{modesFull: e.doc.Edits.(*modesFull), units: 33, structures: 66, trees: 75}
	host.self, e.doc.Edits = host, host
	e.Layout(w, h)
	e.places[popList].placed = false // the palette opens at its default place
	return e
}

// paletteGrid opens bar entry key and lays its window out.
func paletteGrid(t *testing.T, e *MapEditor, key string) (popupSpec, popupLayout) {
	t.Helper()
	chromeClick(e, barPoint(t, e, key))
	s := mustListSpec(t, e)
	if s.Grid == nil {
		t.Fatalf("%s shows no grid", key)
	}
	return s, layoutPopup(s, e.font, e.bodyFont())
}

// A palette opens at a normal width and grows down to the room the map
// gives, scrolling the rest. At 1280x800 the Units window shows every cell
// in its rows; Structures' large cells need more rows than the room holds,
// so that window takes the room down to the map's bottom edge.
func TestMapEditorPalettesShowEveryCellTheRoomHolds(t *testing.T) {
	e := paletteEditor(t, 1280, 800)
	room := e.mapArea()
	s, l := paletteGrid(t, e, "units")
	if !l.Rect.In(e.popupBounds()) || l.Rect.Dx() > room.Dx() || l.Rect.Dx() > 400 {
		t.Fatalf("Units window %v leaves the room %v or opens wider than 400", l.Rect, room)
	}
	if l.GridCols < gridLeastCols || len(l.GridCells) != len(s.Grid.Cells) || !l.GridBar.Empty() {
		t.Fatalf("Units shows %d of %d cells in %d columns, scroll bar %v", len(l.GridCells), len(s.Grid.Cells), l.GridCols, l.GridBar)
	}

	s, l = paletteGrid(t, e, "structures")
	if len(l.GridCells) == len(s.Grid.Cells) {
		t.Fatalf("Structures shows all %d cells at 1280x800; this test expects the room to fall short", len(l.GridCells))
	}
	if l.GridBar.Empty() || room.Max.Y-l.Rect.Max.Y >= s.Grid.side() {
		t.Fatalf("Structures window %v does not take the room %v with a scroll bar (%v)", l.Rect, room, l.GridBar)
	}
}

// Loss control: at 800x600 the Units grid still shows every cell, the
// Structures grid cannot; its scroll bar remains and the wheel over the
// grid scrolls to the last row.
func TestMapEditorPaletteScrollsWhereTheRoomFallsShort(t *testing.T) {
	e := paletteEditor(t, 800, 600)
	s, l := paletteGrid(t, e, "units")
	if len(l.GridCells) != len(s.Grid.Cells) || !l.GridBar.Empty() {
		t.Fatalf("Units at 800x600 shows %d of %d cells, scroll bar %v", len(l.GridCells), len(s.Grid.Cells), l.GridBar)
	}
	s, l = paletteGrid(t, e, "structures")
	if len(l.GridCells) >= len(s.Grid.Cells) || l.GridBar.Empty() {
		t.Fatalf("Structures at 800x600 shows %d of %d cells, scroll bar %v", len(l.GridCells), len(s.Grid.Cells), l.GridBar)
	}
	p := center(l.Grid)
	for range 40 {
		e.Pointer(Input{CursorX: p.X, CursorY: p.Y, WheelY: -1}, time.Unix(1, 0))
	}
	l = layoutPopup(mustListSpec(t, e), e.font, e.bodyFont())
	if l.GridFirst == 0 || l.GridFirst+len(l.GridCells) != len(s.Grid.Cells) {
		t.Fatalf("after the wheel the grid shows cells %d..%d of %d", l.GridFirst, l.GridFirst+len(l.GridCells), len(s.Grid.Cells))
	}
}

// A size the border drag set is kept: the palette scrolls in it, and
// leaving for another tab and coming back shows it at that size again.
func TestMapEditorPaletteKeepsItsDraggedSize(t *testing.T) {
	e := paletteEditor(t, 1280, 800)
	_, l := paletteGrid(t, e, "units")
	e.drag, e.dragEdge = popList, edgeBottom
	e.resizePopup(image.Pt(0, -l.Rect.Dy()/2))
	e.drag, e.dragEdge = popNone, 0
	want, dragged := e.places[popList].size, layoutPopup(mustListSpec(t, e), e.font, e.bodyFont()).Rect.Size()
	if want == (image.Point{}) || dragged.Y >= l.Rect.Dy() {
		t.Fatalf("the drag set size %v, window %v from %v", want, dragged, l.Rect.Size())
	}
	check := func(when string) {
		t.Helper()
		l := layoutPopup(mustListSpec(t, e), e.font, e.bodyFont())
		if e.places[popList].size != want || l.Rect.Size() != dragged || l.GridBar.Empty() {
			t.Fatalf("%s: window %v, size %v, want %v at size %v with a scroll bar (%v)", when, l.Rect.Size(), e.places[popList].size, dragged, want, l.GridBar)
		}
	}
	check("after the drag")
	paletteGrid(t, e, "structures")
	paletteGrid(t, e, "units")
	check("back on Units")
}

// A window that shows a list keeps its list rows: the Trees brush window's
// Forest mode shows its grid alone with at least its least rows, and its One
// kind mode's list is as long as the list window gives it.
func TestMapEditorBrushWindowKeepsItsListRows(t *testing.T) {
	e := paletteEditor(t, 1280, 800)
	s, l := paletteGrid(t, e, "trees")
	if s.List != nil || l.GridRows < gridLeastRows {
		t.Fatalf("the Forest window shows a list (%v) or %d grid rows, want at least %d", s.List != nil, l.GridRows, gridLeastRows)
	}
	e.brushRowButton(0, 0)
	s = mustListSpec(t, e)
	l = layoutPopup(s, e.font, e.bodyFont())
	if s.List == nil {
		t.Fatal("the One kind window shows no list")
	}
	pitch := fontHeight(e.bodyFont(), 15) + listPitchExtra
	avail := e.mapArea().Dy() - 4*(fontHeight(e.bodyFont(), 15)+popupRowExtra) - 40
	if want := max(3, min(18, avail/pitch)); s.ListRows != want || l.ListRows != want {
		t.Fatalf("the list shows %d rows (spec %d), want %d", l.ListRows, s.ListRows, want)
	}
}
