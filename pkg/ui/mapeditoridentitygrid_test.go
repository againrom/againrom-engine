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

// identityGridShown is the Units grid's switch captions and chosen state,
// whether each switch button is drawn pressed, and the grid's heading text
// and cell order as laid out at width 220.
func identityGridShown(t *testing.T, e *MapEditor) (tabs []string, active int, pressed []bool, heads []string, order []int) {
	t.Helper()
	s := mustListSpec(t, e)
	l := layoutPopup(s, e.font, e.bodyFont())
	calls := recordWidgets(t, func() { e.ChromeImage() })
	for _, tab := range l.Tabs {
		down := false
		for _, c := range calls {
			if b, ok := c.state.(pushButton); ok && c.kind == widgetPushButton && c.rect == tab {
				down = b.Pressed
			}
		}
		pressed = append(pressed, down)
	}
	s.Width, s.Size = 220, image.Point{}
	l = layoutPopup(s, e.font, e.bodyFont())
	for _, rect := range l.GridCells {
		part, index := l.hit(center(rect))
		if part != popupGridPart {
			t.Fatal("picture does not take a press")
		}
		order = append(order, index)
	}
	for i, head := range l.GridRowHeads {
		if head.Dx() != l.Grid.Dx() || head.Dy() != s.Grid.side() {
			t.Errorf("heading %d = %v, want a full grid row", i, head)
		}
		if part, _ := l.hit(center(head)); part == popupGridPart {
			t.Fatal("a heading takes a cell press")
		}
	}
	return s.Tabs, s.ActiveTab, pressed, l.GridHeadText, order
}

// identityGridSwitch presses the Units grid's switch button n.
func identityGridSwitch(t *testing.T, e *MapEditor, n int) {
	t.Helper()
	l := layoutPopup(mustListSpec(t, e), e.font, e.bodyFont())
	if n >= len(l.Tabs) {
		t.Fatalf("the Units grid shows %d switch buttons, want Creatures and People", len(l.Tabs))
	}
	chromeClick(e, center(l.Tabs[n]))
}

func TestMapEditorIdentityGridSwitchShowsOneBlock(t *testing.T) {
	e := identityGridEditor(t)
	tabs, active, pressed, heads, order := identityGridShown(t, e)
	if !slices.Equal(tabs, []string{"Creatures", "People"}) || active != 0 {
		t.Fatalf("switch = %q at %d, want Creatures and People with Creatures chosen", tabs, active)
	}
	if !slices.Equal(pressed, []bool{true, false}) {
		t.Fatalf("switch pressed = %v, want Creatures alone", pressed)
	}
	if !slices.Equal(heads, []string{"Goblins"}) || !slices.Equal(order, []int{2, 0}) {
		t.Fatalf("Creatures shows heads %q and cells %v, want Goblins over cells 2, 0", heads, order)
	}
	identityGridSwitch(t, e, 1)
	_, active, pressed, heads, order = identityGridShown(t, e)
	if active != 1 || !slices.Equal(pressed, []bool{false, true}) {
		t.Fatalf("after pressing People the switch is at %d, pressed %v", active, pressed)
	}
	if !slices.Equal(heads, []string{"NPC"}) || !slices.Equal(order, []int{1, 3}) {
		t.Fatalf("People shows heads %q and cells %v, want NPC over cells 1, 3", heads, order)
	}
	chromeClick(e, barPoint(t, e, "units"))
	if _, ok := e.listSpec(); ok {
		t.Fatal("the Units bar button did not close the grid")
	}
	chromeClick(e, barPoint(t, e, "units"))
	if _, active, _, heads, _ = identityGridShown(t, e); active != 1 || !slices.Equal(heads, []string{"NPC"}) {
		t.Fatalf("reopened grid is at %d with heads %q, want People kept", active, heads)
	}
	identityGridSwitch(t, e, 0)
	if _, active, _, heads, _ = identityGridShown(t, e); active != 0 || !slices.Equal(heads, []string{"Goblins"}) {
		t.Fatalf("pressing Creatures gives %d with heads %q", active, heads)
	}
}

func TestMapEditorIdentityGridWheelStaysInTheChosenBlock(t *testing.T) {
	e := identityGridEditor(t)
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
	s := mustListSpec(t, e)
	s.Width, s.Size = 220, image.Point{}
	l := layoutPopup(s, e.font, e.bodyFont())
	if len(l.GridCells) < 2 || l.GridCells[0].Min.Y >= l.GridCells[len(l.GridCells)-1].Min.Y {
		t.Fatal("a group wider than the window does not wrap")
	}
	s.Size = image.Pt(260, 280)
	l = layoutPopup(s, e.font, e.bodyFont())
	wantRows := 1 + (len(values)+l.GridCols-1)/l.GridCols
	if wantRows <= l.GridRows {
		t.Fatalf("fixture fits: %d rows in %d", wantRows, l.GridRows)
	}
	e.placeGridWheel(l, 10000)
	if got := *e.placeTop(); got != wantRows-l.GridRows {
		t.Fatalf("Creatures wheel top = %d, want %d: its heading and wrap rows, no People rows", got, wantRows-l.GridRows)
	}
	s.Grid, _, _ = e.placeGridSpec()
	l = layoutPopup(s, e.font, e.bodyFont())
	part, n := l.hit(center(l.GridCells[len(l.GridCells)-1]))
	if part != popupGridPart || n != 22 {
		t.Fatalf("last scrolled cell = %v / %d, want the last creature, cell 22", part, n)
	}
	identityGridSwitch(t, e, 1)
	if got := *e.placeTop(); got != 0 {
		t.Fatalf("People starts at row %d, want the top", got)
	}
	s = mustListSpec(t, e)
	s.Size = image.Pt(260, 280)
	l = layoutPopup(s, e.font, e.bodyFont())
	peopleRows := 1 + (2+l.GridCols-1)/l.GridCols
	e.placeGridWheel(l, 10000)
	if got := *e.placeTop(); got != max(0, peopleRows-l.GridRows) {
		t.Fatalf("People wheel top = %d, want %d", got, max(0, peopleRows-l.GridRows))
	}
	e.placeGridWheel(l, -10000)
	if got := *e.placeTop(); got != 0 {
		t.Fatalf("People wheel up top = %d", got)
	}
	e.placeGridWheel(l, 10000)
	identityGridSwitch(t, e, 0)
	if got := *e.placeTop(); got != 0 {
		t.Fatalf("Creatures starts at row %d after the switch, want the top", got)
	}
}

func TestMapEditorIdentityGridPressesChooseEachBlocksEdges(t *testing.T) {
	for _, c := range []struct{ state, n, hidden int }{{0, 2, 1}, {0, 0, 3}, {1, 1, 2}, {1, 3, 0}} {
		t.Run(fmt.Sprint(c.state, "/", c.n), func(t *testing.T) {
			e := identityGridEditor(t)
			if c.state != 0 {
				identityGridSwitch(t, e, c.state)
			}
			if _, ok := e.FindControl(ControlQuery{Role: ControlPlaceGridCell, N: c.hidden}); ok {
				t.Fatalf("cell %d of the other block is shown", c.hidden)
			}
			n := c.n
			r, ok := e.FindControl(ControlQuery{Role: ControlPlaceGridCell, N: n})
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
	if s := mustListSpec(t, e); len(s.Tabs) != 0 {
		t.Fatalf("Structures palette shows a switch %q", s.Tabs)
	}
	objectBrushOn(t, e, false)
	g, _, _ = e.placeGridSpec()
	want.Cells = []popupGridCell{{Tip: "2: Kind 2"}}
	if !reflect.DeepEqual(g, want) {
		t.Fatalf("object brush spec changed: %+v", g)
	}
	if s := mustListSpec(t, e); len(s.Tabs) != 0 {
		t.Fatalf("object brush shows a switch %q", s.Tabs)
	}
}

// previewByKeyHost answers each preview key with its own picture, so a test
// tells the cell's picture from the hover tip's.
type previewByKeyHost struct{ *identityGridHost }

func (h *previewByKeyHost) NewRecordPreview(k RecordKind, key string, v int64) *image.RGBA {
	side := map[string]int{"identity-grid": 8, "identity": 20}[key]
	if side == 0 {
		return nil
	}
	return image.NewRGBA(image.Rect(0, 0, side, side))
}

// A unit identity cell keeps its small sprite and its hover tip shows the
// choice's own picture at full size.
func TestMapEditorIdentityGridTipShowsTheChoicePicture(t *testing.T) {
	e := identityGridEditor(t)
	host := &previewByKeyHost{e.doc.Edits.(*identityGridHost)}
	host.self, e.doc.Edits = host, host
	e.place = placeGrid{}
	g, _, _ := e.placeGridSpec()
	for n, c := range g.Cells {
		if c.Icon == nil || c.Icon.Bounds().Dx() != 8 {
			t.Fatalf("cell %d icon is not the grid sprite: %v", n, c.Icon)
		}
		if c.TipPicture == nil || c.TipPicture.Bounds().Dx() != 20 {
			t.Fatalf("cell %d tip picture is not the choice picture: %v", n, c.TipPicture)
		}
	}
}

// Block captions wider than their share of the grid's switch take a line
// each, so every caption stays inside its own button.
func TestMapEditorIdentityGridSwitchKeepsLongCaptionsInsideTheirButtons(t *testing.T) {
	e := identityGridEditor(t)
	host := e.doc.Edits.(*identityGridHost)
	var step InspectionNewStep
	if err := json.Unmarshal([]byte(`{"Choices":[{"Value":1,"Label":"a"},{"Value":2,"Label":"b"}],"Value":1,"Groups":[{"Block":"WWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWW","Head":"H","Values":[1]},{"Block":"MMMMMMMMMMMMMMMMMMMMMMMMMMMMMMMMMMMMMMMMMMMMMMMMMMMMMMMMMMMMMMMMMMMMMM","Head":"N","Values":[2]}]}`), &step); err != nil {
		t.Fatal(err)
	}
	host.step = &step
	e.place = placeGrid{}
	l := layoutPopup(mustListSpec(t, e), e.font, e.bodyFont())
	if len(l.Tabs) != 2 {
		t.Fatalf("switch buttons = %d, want 2", len(l.Tabs))
	}
	for i, tab := range l.Tabs {
		caption := []string{"WWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWW", "MMMMMMMMMMMMMMMMMMMMMMMMMMMMMMMMMMMMMMMMMMMMMMMMMMMMMMMMMMMMMMMMMMMMMM"}[i]
		if w := e.bodyFont().Advance(caption); w > tab.Dx() {
			t.Errorf("caption %d is %d wide inside a %d wide button", i, w, tab.Dx())
		}
	}
}
