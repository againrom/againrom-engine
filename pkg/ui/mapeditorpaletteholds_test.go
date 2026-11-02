package ui

import (
	"fmt"
	"image"
	"slices"
	"strings"
	"testing"

	"againrom/pkg/words"
)

// placeableHost offers the structure kinds of kinds, each in the palette the
// host's data gives it, and the object classes of objects.
type placeableHost struct {
	*modesFull
	kinds   []int64
	palette map[int64]Palette
	objects []BrushObject
}

func (h *placeableHost) NewRecordStep(r InspectionNewRecord, key string) InspectionNewStep {
	if r.Kind != KindStructure || key != "kind" {
		return h.modesFull.NewRecordStep(r, key)
	}
	s := InspectionNewStep{Value: h.kinds[0], Palettes: h.palette}
	for _, k := range h.kinds {
		s.Choices = append(s.Choices, InspectionChoice{Value: k, Label: fmt.Sprintf("%d: Kind %d", k, k)})
	}
	return s
}

func (h *placeableHost) BrushObjects() []BrushObject { return h.objects }

// placeableHostEditor is the full modes editor at 1920x1080 over a
// placeableHost.
func placeableHostEditor(t *testing.T, kinds map[int64]Palette, objects ...BrushObject) (*MapEditor, *modesCalls) {
	t.Helper()
	e, calls := newModesEditor(t, "full")
	host := &placeableHost{modesFull: e.doc.Edits.(*modesFull), palette: kinds, objects: append([]BrushObject{{Code: 0, Name: "No object"}}, objects...)}
	for k := range kinds {
		host.kinds = append(host.kinds, k)
	}
	slices.Sort(host.kinds)
	host.self, e.doc.Edits = host, host
	e.place = placeGrid{}
	e.Layout(1920, 1080)
	return e, calls
}

// The fixture's records: structure kind 1 a tower that can be selected,
// kind 2 a fence that only decorates, kind 3 a gate the data does not
// settle, which the host gives no palette; object class 5 a stone and 6 a
// bush that decorate, 6 with no palette from the host, and 7 a pine.
var (
	paletteKinds   = map[int64]Palette{1: PaletteStructures, 2: PaletteObjects, 3: PaletteOwn}
	paletteObjects = []BrushObject{{Code: 5, Name: "Stone", Palette: PaletteObjects}, {Code: 6, Name: "Bush", Palette: PaletteOwn}, {Code: 7, Name: "Pine", Palette: PaletteTrees}}
)

// gridTips are the tips of the shown list popup's grid cells.
func gridTips(t *testing.T, e *MapEditor) []string {
	t.Helper()
	g, _, ok := e.placeGridSpec()
	if !ok {
		return nil
	}
	var out []string
	for _, c := range g.Cells {
		out = append(out, c.Tip)
	}
	return out
}

// objectBrushOn turns the object brush on, in Forest mode when forest is set,
// by its key and its Forest or One kind button.
func objectBrushOn(t *testing.T, e *MapEditor, forest bool) {
	t.Helper()
	if e.listKind() != "brush" || e.brush.mode != brushObjects {
		if err := e.Key("objects"); err != nil {
			t.Fatal(err)
		}
	}
	k := 0
	if forest {
		k = 1
	}
	r, ok := e.ListButtonRect(2, k)
	if !ok {
		t.Fatal("the object brush shows no Forest row")
	}
	chromeClick(e, center(r))
	if e.forestOn() != forest {
		t.Fatalf("the Forest row's button %d left Forest %v", k, e.forestOn())
	}
}

// Each placeable record lists in exactly one palette, as the host's data
// gives it: Structures holds the tower and the unsettled gate, Objects the
// fence, the stone and the bush, Trees the pine. The windows show the same:
// the Structures grid, the object brush's grid of decoration kinds and its
// rows, and the Forest grid.
func TestMapEditorEachPlaceableRecordListsInOnePalette(t *testing.T) {
	e, _ := placeableHostEditor(t, paletteKinds, paletteObjects...)
	for _, c := range []struct {
		p       Palette
		kinds   []int64
		objects []int
	}{
		{PaletteStructures, []int64{1, 3}, nil},
		{PaletteObjects, []int64{2}, []int{5, 6}},
		{PaletteTrees, nil, []int{7}},
	} {
		kinds, objects := e.PaletteHolds(c.p)
		if !slices.Equal(kinds, c.kinds) || !slices.Equal(objects, c.objects) {
			t.Errorf("palette %d holds kinds %v and objects %v, want %v and %v", c.p, kinds, objects, c.kinds, c.objects)
		}
	}

	chromeClick(e, barPoint(t, e, "structures"))
	if tips := gridTips(t, e); !slices.Equal(tips, []string{"1: Kind 1", "3: Kind 3"}) {
		t.Fatalf("the Structures grid shows %q", tips)
	}
	objectBrushOn(t, e, false)
	if tips := gridTips(t, e); !slices.Equal(tips, []string{"2: Kind 2"}) {
		t.Fatalf("the Objects grid shows %q", tips)
	}
	var names []string
	for _, o := range e.brushObjectRows() {
		names = append(names, o.Name)
	}
	if !slices.Equal(names, []string{"No object", "Stone", "Bush"}) {
		t.Fatalf("the Objects rows are %q", names)
	}
	objectBrushOn(t, e, true)
	names = nil
	for _, o := range e.forestKinds() {
		names = append(names, o.Name)
	}
	if g := mustListSpec(t, e).Grid; !slices.Equal(names, []string{"Pine"}) || g == nil || len(g.Cells) != 1 {
		t.Fatalf("the Forest grid holds %q (%+v)", names, g)
	}
	if gridTips(t, e) != nil {
		t.Fatal("Forest shows the decoration kinds' grid")
	}
}

// A structure whose kind lists in the Objects palette is still a structure
// record: the bar's search finds it, and the tree lists it under the
// Structures branch, the one its record kind names.
func TestMapEditorDecorationStructureIsFoundByTheSearchAndTheTree(t *testing.T) {
	e, _ := placeableHostEditor(t, map[int64]Palette{1: PaletteObjects, 2: PaletteObjects}, paletteObjects...)
	if kinds, _ := e.PaletteHolds(PaletteStructures); len(kinds) != 0 {
		t.Fatalf("the Structures palette holds %v", kinds)
	}
	hall := slices.IndexFunc(e.doc.Records, func(r InspectionRecord) bool { return r.Kind == KindStructure && r.Label == "hall" })
	e.search.text = "hall"
	if hits := e.searchHits(); !slices.ContainsFunc(hits, func(m treeMember) bool { return m.row.record == hall }) {
		t.Fatalf("the search for hall finds %+v", hits)
	}
	c, _ := categoryByKey("structures")
	if members := e.treeMatches(c, "hall"); !slices.ContainsFunc(members, func(m treeMember) bool { return m.row.record == hall }) {
		t.Fatalf("the tree's Structures branch keeps %+v for hall", members)
	}
}

// listTexts are the shown list popup's row labels.
func listTexts(t *testing.T, e *MapEditor) string {
	t.Helper()
	var out []string
	for _, r := range mustListSpec(t, e).Rows {
		out = append(out, r.Label)
	}
	return strings.Join(out, "\n")
}

// A palette with an empty category says so, in English and Russian: the
// Structures palette with no structure kind, the Objects palette with no
// decoration class or no decoration kind, Forest with no tree. A category
// that holds a record says nothing.
func TestMapEditorEmptyPaletteSaysSoInPlainWords(t *testing.T) {
	for _, lang := range []string{"", "ru"} {
		book := words.For(lang)
		say := func(id string) string { return book.Text(id) }
		for _, c := range []struct {
			name    string
			kinds   map[int64]Palette
			objects []BrushObject
			says    map[string][]string // bar entry or brush mode, notes
		}{
			{"decoration only", map[int64]Palette{2: PaletteObjects}, paletteObjects[:1], map[string][]string{
				"structures": {say("editor.palette.nostructures")}, "objects": nil, "forest": {say("editor.palette.notrees")}}},
			{"structures and trees only", map[int64]Palette{1: PaletteStructures}, paletteObjects[2:], map[string][]string{
				"structures": nil, "objects": {say("editor.palette.noobjects"), say("editor.palette.nodecoration")}, "forest": nil}},
		} {
			e, _ := placeableHostEditor(t, c.kinds, c.objects...)
			e.SetWords(book)
			for _, where := range []string{"structures", "objects", "forest"} {
				switch where {
				case "structures":
					chromeClick(e, barPoint(t, e, "structures"))
				default:
					objectBrushOn(t, e, where == "forest")
				}
				got := listTexts(t, e)
				for _, id := range []string{"editor.palette.nostructures", "editor.palette.noobjects", "editor.palette.nodecoration", "editor.palette.notrees"} {
					if strings.Contains(got, say(id)) != slices.Contains(c.says[where], say(id)) {
						t.Errorf("%s %q, %s: shows %q is %v:\n%s", lang, c.name, where, say(id), strings.Contains(got, say(id)), got)
					}
				}
				_ = e.Key("escape")
			}
		}
	}
}

// A press on the Objects grid's cell places the decoration kind through the
// one Add window: the copy it places on a cell is the host's same Add as the
// Structure key's window with that kind and cell. Esc goes back to the
// object brush, before a copy and after one.
func TestMapEditorObjectsGridPlacesTheAddWindowsRecord(t *testing.T) {
	cell := image.Pt(25, 20)
	var byGrid string
	for _, copies := range []int{0, 1} {
		e, calls := placeableHostEditor(t, paletteKinds, paletteObjects...)
		objectBrushOn(t, e, false)
		r, ok := e.PlaceGridCellRect(0)
		if !ok {
			t.Fatal("the Objects grid shows no cell")
		}
		chromeClick(e, center(r))
		run, ok := e.addRunning()
		if !ok || !run.repeat || run.kind != KindStructure || e.pick != pickAddCell {
			t.Fatalf("the grid press opened no placing Add window: %+v pick %q", run, e.pick)
		}
		if copies > 0 {
			e.Fit()
			modesClick(e, e.CellPoint(cell))
			if len(calls.lines) != 1 {
				t.Fatalf("the click placed %v", calls.lines)
			}
			byGrid = calls.lines[0]
		}
		_ = e.Key("escape") // disarms the cell pick
		if _, open := e.addRunning(); !open || e.pick != pickNone {
			t.Fatalf("after %d copies the first Esc left the pick %q", copies, e.pick)
		}
		_ = e.Key("escape")
		if _, open := e.addRunning(); open || e.listKind() != "brush" || e.brush.mode != brushObjects || e.forestOn() {
			t.Fatalf("after %d copies Esc left the Add window %v, %q, brush mode %d, Forest %v", copies, open, e.listKind(), e.brush.mode, e.forestOn())
		}
	}

	w, wcalls := placeableHostEditor(t, paletteKinds, paletteObjects...)
	_ = w.Key("structure")
	if addKindNow(w) != "Structure" {
		t.Fatal("the Structure key opened no Add window")
	}
	w.prompt.path = "2"
	addAdvance(t, w, 2) // kind, owner
	w.prompt.path = fmt.Sprintf("%d,%d", cell.X, cell.Y)
	_ = w.Key("enter")
	if len(wcalls.lines) != 1 || wcalls.lines[0] != byGrid {
		t.Fatalf("the Add window placed %v, the Objects grid %q", wcalls.lines, byGrid)
	}
}
