package ui

import (
	"image"
	"slices"
	"strings"
	"testing"
	"time"

	"againrom/pkg/render/terrain"
)

// The action names of the "on the map" buttons, as the popups list them.
const (
	testPickParam = "pickparam"
	testPickArea  = "pickarea"
)

// The cells of the map the tests click.
var (
	cellNamedUnit   = image.Pt(10, 10) // unit 101, group 3
	cellPlainUnit   = image.Pt(12, 10) // a unit no script can name, in no group
	cellStructure   = image.Pt(14, 10) // structure 201
	cellEmpty       = image.Pt(40, 8)
	cellBigBox      = image.Pt(20, 20) // only the 20..26 box covers it
	cellNestedBox   = image.Pt(23, 23) // the 20..26 box and the 22..23 box cover it
	cellSquareAreaC = image.Pt(35, 35) // the square of radius 3
)

// paramPickEditor opens a 64 by 64 map whose one trigger has no rows, with a
// condition node holding a unit, a structure and a group parameter, three
// placed records and three trigger areas. host.plans records what Apply
// writes.
func paramPickEditor(t *testing.T) (*MapEditor, *wizardWrites) {
	t.Helper()
	host := &wizardWrites{editorTextEdits: &editorTextEdits{}}
	build := func(string) (*InspectionDocument, error) {
		v, err := NewViewer("params", terrain.Grid{Width: 64, Height: 64, Tiles: make([]uint16, 64*64)}, &terrain.Tileset{})
		if err != nil {
			return nil, err
		}
		d := &InspectionDocument{Viewer: v, Source: "params", Width: 64, Height: 64, SourceBytes: []byte("map"), Edits: host}
		d.Records = []InspectionRecord{
			{Kind: "Trigger", Index: 0, Section: 7},
			{Kind: "Condition", Index: 0, Section: 7},
			{Kind: "Unit", Index: 0, Spatial: true, Cell: cellNamedUnit, Size: image.Pt(1, 1), Player: 1, Group: 3},
			{Kind: "Unit", Index: 1, Spatial: true, Cell: cellPlainUnit, Size: image.Pt(1, 1), Player: 1},
			{Kind: "Structure", Index: 0, Spatial: true, Cell: cellStructure, Size: image.Pt(1, 1)},
		}
		d.Script = &InspectionScriptModel{
			Triggers: []InspectionTrigger{{Record: 0, Name: "Start"}},
			Conditions: []InspectionScriptNode{{Record: 1, ID: 1, Opcode: 99,
				Types:  [10]uint32{scriptUnit, scriptStructure, scriptGroup},
				Values: [10]uint32{7, 8, 9}}},
			Targets: map[uint32][]InspectionScriptTarget{
				scriptUnit:      {{Value: 101, Record: 2, Name: "Archer"}},
				scriptStructure: {{Value: 201, Record: 4, Name: "Tower"}},
				scriptGroup:     {{Value: 3, Record: -1, Members: 1}},
			},
			Areas: []InspectionArea{
				{Trigger: 0, Node: 1, Shape: AreaBox, Rect: image.Rect(20, 20, 27, 27), Slots: [4]int{-1, -1, -1, -1}},
				{Trigger: 0, Node: 1, Shape: AreaBox, Rect: image.Rect(22, 22, 24, 24), Slots: [4]int{-1, -1, -1, -1}},
				{Trigger: 0, Node: 1, Shape: AreaSquare, Rect: image.Rect(32, 32, 39, 39), Centre: cellSquareAreaC, Radius: 3, Slots: [4]int{-1, -1, -1, -1}},
			},
		}
		return d, nil
	}
	e := NewMapEditor(panelFont(), nil, build)
	e.SetBodyFont(panelFont())
	e.Layout(1280, 800)
	if err := e.Open("params"); err != nil {
		t.Fatal(err)
	}
	// The popups stand in the window's bottom-right corner, clear of the
	// cells the clicks aim at.
	e.objectAt, e.objPlaced = image.Pt(e.window.X, e.window.Y), true
	e.Fit()
	return e, host
}

// armParam presses the "on the map" button act of the popup row that holds
// the field key.
func armParam(t *testing.T, e *MapEditor, key, act string) {
	t.Helper()
	if _, _, ok := e.objectSpec(); !ok {
		t.Fatal("no popup")
	}
	for n, f := range e.obj.rowField {
		if e.obj.fields[f].Key != key {
			continue
		}
		k := slices.Index(e.obj.rowActs[n], act)
		if k < 0 {
			t.Fatalf("field %q: the row offers %q, not %q", key, e.obj.rowActs[n], act)
		}
		e.scriptRowButton(n, k)
		return
	}
	t.Fatalf("no row shows field %q", key)
}

// fieldText is the text of the field key.
func fieldText(t *testing.T, e *MapEditor, key string) string {
	t.Helper()
	for i, f := range e.obj.fields {
		if f.Key == key {
			return e.obj.buf[i]
		}
	}
	t.Fatalf("no field %q", key)
	return ""
}

// wizardParamEditor is the editor with the wizard open and one condition and
// one action row added.
func wizardParamEditor(t *testing.T) *MapEditor {
	t.Helper()
	e, _ := paramPickEditor(t)
	e.Select(0, false)
	if !e.WizardShown() {
		t.Fatalf("the trigger opens no wizard:\n%s", strings.Join(e.PopupLines(), "\n"))
	}
	wizardPress(t, e, -1, actAddPair)
	wizardPress(t, e, -1, actAddAct)
	return e
}

func clickCell(e *MapEditor, cell image.Point) { modesClick(e, e.CellPoint(cell)) }

// pickCase runs one parameter kind: the button arms the pick and the status
// line names it, a click on a record or area that does not fit leaves the
// field unchanged and the pick armed with the status line saying so, Esc
// cancels, and a click that fits fills the field.
func pickCase(t *testing.T, e *MapEditor, key, act string, bad image.Point, good image.Point, want string) {
	t.Helper()
	before := fieldText(t, e, key)
	armParam(t, e, key, act)
	if e.pick == pickNone {
		t.Fatalf("%s: the button armed no pick", key)
	}
	if armed := e.statusText(); !strings.Contains(armed, "Esc cancels") || strings.Contains(armed, "No ") {
		t.Fatalf("%s: the armed status line does not name the pick: %q", key, armed)
	}
	clickCell(e, bad)
	if e.pick == pickNone || fieldText(t, e, key) != before {
		t.Fatalf("%s: a click on nothing that fits left pick %q, field %q (was %q)", key, e.pick, fieldText(t, e, key), before)
	}
	if missed := e.statusText(); !strings.Contains(missed, "Esc cancels") || !strings.Contains(missed, "No ") {
		t.Fatalf("%s: the status line does not say the click found nothing: %q", key, missed)
	}
	_ = e.Key("escape")
	if e.pick != pickNone || fieldText(t, e, key) != before {
		t.Fatalf("%s: Esc left pick %q, field %q", key, e.pick, fieldText(t, e, key))
	}
	armParam(t, e, key, act)
	clickCell(e, good)
	if e.pick != pickNone {
		t.Fatalf("%s: the pick stays armed after a click that fits", key)
	}
	if got := fieldText(t, e, key); got != want {
		t.Fatalf("%s: the click filled %q, want %q", key, got, want)
	}
}

func TestTriggerWizardParametersTakeAClickOnTheMap(t *testing.T) {
	e := wizardParamEditor(t)
	// Hero enters area: a unit and a box.
	pickCase(t, e, "c0.p0", testPickParam, cellStructure, cellNamedUnit, "101")
	if f := e.WizardField("c0.p0"); !slices.ContainsFunc(e.obj.fields[f].Choices, func(c InspectionChoice) bool { return c.Value == 101 }) {
		t.Fatalf("the unit field's choices %+v lack the picked unit", e.obj.fields[f].Choices)
	}
	pickCase(t, e, "c0.p0", testPickParam, cellPlainUnit, cellNamedUnit, "101")
	pickCase(t, e, "c0.p1", testPickArea, cellEmpty, cellBigBox, "20,20,26,26")
	// The smallest of the areas under the pointer is picked, as a click selects it.
	pickCase(t, e, "c0.p1", testPickArea, cellEmpty, cellNestedBox, "22,22,23,23")
	// A square area is a box too.
	pickCase(t, e, "c0.p1", testPickArea, cellEmpty, cellSquareAreaC, "32,32,38,38")

	// Give group: a group through one of its units.
	wizardChoose(t, e, "a0.item", 5)
	pickCase(t, e, "a0.p0", testPickParam, cellPlainUnit, cellNamedUnit, "3")
	wizardChoose(t, e, "a0.item", 4)
	pickCase(t, e, "a0.p0", testPickParam, cellStructure, cellNamedUnit, "101")

	// A player near a point: the point and its radius from a square area.
	wizardChoose(t, e, "c0.item", 1)
	before := fieldText(t, e, "c0.p1")
	pickCase(t, e, "c0.p1", testPickArea, cellBigBox, cellSquareAreaC, "35,35")
	if got := fieldText(t, e, "c0.p2"); got != "3" {
		t.Fatalf("the radius field holds %q after the square area is picked, want 3 (the point was %q)", got, before)
	}
}

func TestTriggerWizardAreaPickFollowsTheViewMenu(t *testing.T) {
	e := wizardParamEditor(t)
	e.view.hideAreas = true
	armParam(t, e, "c0.p1", testPickArea)
	before := fieldText(t, e, "c0.p1")
	clickCell(e, cellBigBox)
	if e.pick == pickNone || fieldText(t, e, "c0.p1") != before {
		t.Fatal("a hidden area was picked")
	}
}

func TestTriggerNodeParametersTakeAClickOnTheMap(t *testing.T) {
	e, _ := paramPickEditor(t)
	e.Select(1, false)
	if e.WizardShown() {
		t.Fatal("a node opens the wizard")
	}
	pickCase(t, e, "value0", testPickParam, cellStructure, cellNamedUnit, "101")
	pickCase(t, e, "value0", testPickParam, cellPlainUnit, cellNamedUnit, "101")
	pickCase(t, e, "value1", testPickParam, cellNamedUnit, cellStructure, "201")
	pickCase(t, e, "value2", testPickParam, cellPlainUnit, cellNamedUnit, "3")
	pickCase(t, e, "value2", testPickParam, cellStructure, cellNamedUnit, "3")
	// The pick is a draft until Apply, as every field of the popup is.
	if !e.objectDirty() {
		t.Fatal("the popup is not dirty after picks")
	}
	e.cancelPopup()
	e.Select(1, false)
	if got := fieldText(t, e, "value0"); got != "7" {
		t.Fatalf("Cancel kept the picked unit: %q", got)
	}
}

// The popup's other fields offer no map pick: a number, a player, a cell
// keeps its own.
func TestTriggerParameterButtonsAreOnlyForMapRecords(t *testing.T) {
	e, _ := paramPickEditor(t)
	e.Select(1, false)
	if _, _, ok := e.objectSpec(); !ok {
		t.Fatal("no popup")
	}
	var picks int
	for n, f := range e.obj.rowField {
		for _, a := range e.obj.rowActs[n] {
			if a == testPickParam || a == testPickArea {
				picks++
				if k := e.obj.fields[f].Key; k != "value0" && k != "value1" && k != "value2" {
					t.Fatalf("field %q offers a map pick", k)
				}
			}
		}
	}
	if picks != 3 {
		t.Fatalf("%d map picks, want 3", picks)
	}
}

func TestAreaPickHighlightsTheAreaUnderThePointer(t *testing.T) {
	e := wizardParamEditor(t)
	armParam(t, e, "c0.p1", testPickArea)
	p := e.CellPoint(cellNestedBox)
	e.Pointer(Input{CursorX: p.X, CursorY: p.Y}, time.Time{})
	f, ok := e.placement()
	if !ok || len(f.Cells) != 1 || f.Cells[0].Rect != image.Rect(22, 22, 24, 24) {
		t.Fatalf("placement %+v %v; want the nested area highlighted", f, ok)
	}
}
