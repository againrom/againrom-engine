package ui

import (
	"encoding/json"
	"fmt"
	"image"
	"reflect"
	"slices"
	"strings"
	"testing"
)

type identityGridHost struct {
	*modesFull
	step *InspectionNewStep
}

func (h *identityGridHost) NewRecordStep(r InspectionNewRecord, key string) InspectionNewStep {
	if r.Kind != KindUnit || key != "identity" {
		return h.modesFull.NewRecordStep(r, key)
	}
	if h.step != nil {
		return *h.step
	}
	var step InspectionNewStep
	err := json.Unmarshal([]byte(`{"Choices":[{"Value":1,"Label":"last creature"},{"Value":2,"Label":"first person"},{"Value":3,"Label":"first creature"},{"Value":4,"Label":"last person"}],"Value":1,"Groups":[{"Block":"Creatures","Head":"Goblins","Values":[3,1]},{"Block":"People","Head":"NPC","Values":[2,4]}]}`), &step)
	if err != nil {
		panic(err)
	}
	return step
}

func identityGridEditor(t *testing.T) *MapEditor {
	t.Helper()
	e, _ := newModesEditor(t, "full")
	host := &identityGridHost{modesFull: e.doc.Edits.(*modesFull)}
	host.self, e.doc.Edits = host, host
	e.place = placeGrid{}
	e.Layout(1280, 800)
	e.places[popList].placed = false
	chromeClick(e, barPoint(t, e, "units"))
	return e
}

func TestMapEditorIdentityGridHeadsRowsAndWrap(t *testing.T) {
	e := identityGridEditor(t)
	s := mustListSpec(t, e)
	s.Width, s.Size = 220, image.Point{}
	l := layoutPopup(s, e.font, e.bodyFont())
	if len(l.GridRowHeads) != 4 {
		t.Fatalf("grid shows %d heading rows, want Creatures, Goblins, People, NPC", len(l.GridRowHeads))
	}
	if !slices.Equal(l.GridHeadText, []string{"Creatures", "Goblins", "People", "NPC"}) {
		t.Fatalf("heading text = %q", l.GridHeadText)
	}
	for i, head := range l.GridRowHeads {
		if head.Dx() != l.Grid.Dx() || head.Dy() != s.Grid.side() {
			t.Errorf("heading %d = %v, want a full grid row", i, head)
		}
		if part, _ := l.hit(center(head)); part == popupGridPart {
			t.Fatal("a heading takes a cell press")
		}
	}
	var order []int
	for _, rect := range l.GridCells {
		part, index := l.hit(center(rect))
		if part != popupGridPart {
			t.Fatal("picture does not take a press")
		}
		order = append(order, index)
	}
	if !slices.Equal(order, []int{2, 0, 1, 3}) {
		t.Fatalf("visual cell order = %v, want the groups' original choice indices", order)
	}
	step := e.doc.Edits.(*identityGridHost).NewRecordStep(InspectionNewRecord{Kind: KindUnit}, "identity")
	for i := 5; i < 24; i++ {
		step.Choices = append(step.Choices, InspectionChoice{Value: int64(i), Label: fmt.Sprint(i)})
	}
	data, _ := json.Marshal(step)
	var fixture map[string]any
	_ = json.Unmarshal(data, &fixture)
	values := []int{3, 1}
	for i := 5; i < 24; i++ {
		values = append(values, i)
	}
	fixture["Groups"] = []any{map[string]any{"Block": "Creatures", "Head": "Goblins", "Values": values}, map[string]any{"Block": "People", "Head": "NPC", "Values": []int{2, 4}}}
	data, _ = json.Marshal(fixture)
	_ = json.Unmarshal(data, &step)
	e.doc.Edits.(*identityGridHost).step = &step
	e.place = placeGrid{}
	s = mustListSpec(t, e)
	s.Width, s.Size = 220, image.Point{}
	l = layoutPopup(s, e.font, e.bodyFont())
	if len(l.GridCells) < 2 || l.GridCells[0].Min.Y >= l.GridCells[len(l.GridCells)-1].Min.Y {
		t.Fatal("a group wider than the window does not wrap")
	}
	s.Size = image.Pt(260, 280)
	l = layoutPopup(s, e.font, e.bodyFont())
	wantRows := 4 + (len(values)+l.GridCols-1)/l.GridCols + (2+l.GridCols-1)/l.GridCols
	e.placeGridWheel(l, 10000)
	if got := *e.placeTop(); got != wantRows-l.GridRows {
		t.Fatalf("wheel top = %d, want %d including heading and wrap rows", got, wantRows-l.GridRows)
	}
	s.Grid, _, _ = e.placeGridSpec()
	l = layoutPopup(s, e.font, e.bodyFont())
	part, n := l.hit(center(l.GridCells[len(l.GridCells)-1]))
	if part != popupGridPart || n != 3 {
		t.Fatalf("last scrolled cell = %v / %d, want last person", part, n)
	}
}

func TestMapEditorIdentityGridPressesChooseEachBlocksEdges(t *testing.T) {
	for _, n := range []int{2, 0, 1, 3} {
		t.Run(fmt.Sprint(n), func(t *testing.T) {
			e := identityGridEditor(t)
			r, ok := e.PlaceGridCellRect(n)
			if !ok {
				t.Fatal("cell is not visible", n)
			}
			g, _, _ := e.placeGridSpec()
			if tip, _, ok := e.gridTipAt(center(r)); !ok || tip != g.Cells[n].Tip {
				t.Fatalf("cell %d hint = %q", n, tip)
			}
			chromeClick(e, center(r))
			run, value, placing := e.placing("identity")
			if !placing || run.kind != KindUnit || value != int64(n+1) || e.pick != pickAddCell {
				t.Fatalf("press %d selected %d, placing=%v, pick=%q", n, value, placing, e.pick)
			}
			g, note, _ := e.placeGridSpec()
			if !g.Cells[n].Pressed || !strings.Contains(note, g.Cells[n].Tip) {
				t.Fatal("pressed cell or placing note lost the chosen identity")
			}
			chromeClick(e, center(r))
			if _, _, placing = e.placing("identity"); placing {
				t.Fatal("second press did not end placing")
			}
			chromeClick(e, center(r))
			_ = e.Key("escape")
			if e.pick == pickAddCell {
				t.Fatal("Esc did not end the placement pick")
			}
		})
	}
}

func TestMapEditorStructureGridSpecStaysUngrouped(t *testing.T) {
	e, _ := placeableHostEditor(t, paletteKinds, paletteObjects...)
	chromeClick(e, barPoint(t, e, "structures"))
	g, note, ok := e.placeGridSpec()
	want := placeGridShell(KindStructure, 0)
	want.Cells = []popupGridCell{{Tip: "1: Kind 1"}, {Tip: "3: Kind 3"}}
	if !ok || note != e.word("editor.place.hint") || !reflect.DeepEqual(g, want) {
		t.Fatalf("Structures spec changed: %+v, %q", g, note)
	}
	objectBrushOn(t, e, false)
	g, _, _ = e.placeGridSpec()
	want.Cells = []popupGridCell{{Tip: "2: Kind 2"}}
	if !reflect.DeepEqual(g, want) {
		t.Fatalf("object brush spec changed: %+v", g)
	}
}
