package ui

import (
	"image"
	"slices"
	"strings"
	"testing"

	"againrom/pkg/render/terrain"
)

// A row whose label or value the popup cuts shows its whole text, wrapped to
// the row, in the hover line under it while the pointer is over the row; a
// row shown whole has no hover line.
func TestPopupHoverLineShowsACutRowWhole(t *testing.T) {
	f := panelFont()
	long := strings.Repeat("brigands gone mad ", 6)
	s := popupSpec{Title: "Trigger 1", At: image.Pt(20, 40), Width: 360, Bounds: image.Rect(0, 0, 1024, 768), LabelWidth: 120,
		Rows: []popupRow{
			{Kind: popupText, Label: "Name", Value: "Short"},
			{Kind: popupSwitch, Label: "Fire once", Value: long},
			{Kind: popupChoice, Label: "Condition", Value: long, Buttons: []string{"Remove"}},
			{Kind: popupEdit, Label: "Label", Value: long},
			{Kind: popupText, Label: long + "label", Value: "1"},
			{Kind: popupText, Label: "Value", Value: long},
		}}
	l := layoutPopup(s, f, f)
	center := func(r image.Rectangle) image.Point { return r.Min.Add(r.Size().Div(2)) }
	for i, rr := range l.Rows {
		row := s.Rows[rr.Index]
		full, cut := rowFull(f, f, row, rr)
		tip, ok := popupTipAt(image.Rect(0, 0, 1024, 768), f, f, s, l, center(rr.Rect), true)
		if i == 0 {
			if cut || ok {
				t.Fatalf("a short row is cut %v or shows a hover line %+v", cut, tip)
			}
			continue
		}
		if !cut || !ok {
			t.Fatalf("row %d (%q) cut %v, hover line %v", i, full, cut, ok)
		}
		if strings.Join(tip.Lines, "") != full || full != strings.TrimSpace(row.Label+" "+row.Value) {
			t.Fatalf("row %d hover line %q, the row's text %q", i, tip.Lines, full)
		}
		for _, line := range tip.Lines {
			if f.Advance(line) > tip.Rect.Dx()-8 {
				t.Fatalf("row %d hover line %q is wider than its box %v", i, line, tip.Rect)
			}
		}
		if tip.Rect.Min.Y != rr.Rect.Max.Y || len(tip.Lines) < 2 && i != 4 {
			t.Fatalf("row %d hover line at %v over %d lines, the row at %v", i, tip.Rect, len(tip.Lines), rr.Rect)
		}
	}
	if _, ok := popupTipAt(image.Rect(0, 0, 1024, 768), f, f, s, l, center(l.Rows[2].Rect), false); ok {
		t.Fatal("a hover line without a pointer")
	}
	// The painter draws the hover line's box over the popup.
	count := func(p image.Point) int {
		n := 0
		widgetRecord = func(c widgetCall) {
			if c.kind == widgetFrame {
				n++
			}
		}
		defer func() { widgetRecord = nil }()
		drawPopup(image.NewRGBA(image.Rect(0, 0, 1024, 768)), f, f, nil, s, l, editorPopupInk, p, true)
		return n
	}
	if over, off := count(center(l.Rows[1].Rect)), count(center(l.Rows[0].Rect)); over != off+1 {
		t.Fatalf("frames with the hover line %d, without %d", over, off)
	}
}

// groupOrderEditor opens a 120 by 120 map whose one trigger runs a Swarm 2
// order of group 5 to cell 96,84, slot for slot as mission 40's "D - Brigands
// gone mad", under a catalog whose group command signatures are the claims'
// (TRIG-GROUPBOUNDS-136).
func groupOrderEditor(t *testing.T) *MapEditor {
	t.Helper()
	sig := func(condition bool, opcode, command uint32) ([10]InspectionScriptSlot, bool) {
		var out [10]InspectionScriptSlot
		if condition || opcode != scriptGroupCommand {
			return out, false
		}
		out[0], out[9] = InspectionScriptSlot{Type: scriptConst, Role: "command"}, InspectionScriptSlot{Type: scriptGroup}
		switch command {
		case 2, 4, 5, 14:
			out[1], out[2] = InspectionScriptSlot{Type: scriptX, Role: "cell"}, InspectionScriptSlot{Type: scriptY}
		case 10:
			out[1] = InspectionScriptSlot{Type: scriptUnit, Role: "unit"}
		case 3, 17, 18:
		default:
			return out, false
		}
		return out, true
	}
	build := func(string) (*InspectionDocument, error) {
		v, err := NewViewer("order", terrain.Grid{Width: 120, Height: 120, Tiles: make([]uint16, 120*120)}, &terrain.Tileset{})
		if err != nil {
			return nil, err
		}
		d := &InspectionDocument{Viewer: v, Source: "order", Width: 120, Height: 120, SourceBytes: []byte("map"), Edits: &editorTextEdits{}}
		d.Records = []InspectionRecord{{Kind: "Trigger", Index: 0, Section: 7}, {Kind: "Action", Index: 0, Section: 7}}
		d.Script = &InspectionScriptModel{
			Triggers: []InspectionTrigger{{Record: 0, Name: "D - Brigands gone mad", Once: 1, Acts: [4]uint32{21}}},
			Actions: []InspectionScriptNode{{Record: 1, ID: 21, Opcode: scriptGroupCommand,
				Types: [10]uint32{7, 5, 6, 9: 2}, Values: [10]uint32{5, 96, 84, 9: 5}}},
			Targets: map[uint32][]InspectionScriptTarget{scriptGroup: {
				{Value: 14, Record: -1, Members: 1, Name: "Group 14 (units: 1)"},
				{Value: 5, Record: -1, Members: 7, Name: "Group 5 (units: 7)"}}},
			Catalog: InspectionScriptCatalog{GroupCommands: []uint32{1, 2, 3, 4, 5, 10, 11, 14, 15, 17, 18}, Signature: sig},
		}
		return d, nil
	}
	e := NewMapEditor(panelFont(), nil, build)
	e.SetBodyFont(panelFont())
	e.Layout(1280, 800)
	if err := e.Open("order"); err != nil {
		t.Fatal(err)
	}
	return e
}

// The wizard reads a group order as its own row: the group from the map's
// groups, the command from the named commands that take a cell, the point a
// cell; a new row of that item says the shipped node slot for slot.
func TestTriggerWizardReadsAndBuildsAGroupOrder(t *testing.T) {
	e := groupOrderEditor(t)
	e.Select(0, false)
	if !e.WizardShown() {
		t.Fatalf("the trigger opens no wizard:\n%s", strings.Join(e.PopupLines(), "\n"))
	}
	lines := e.PopupLines()
	for _, want := range []string{"Action Order a group to a point", "Group Group 5 (units: 7)", "Command Swarm 2", "Point 96,84"} {
		if !slices.Contains(lines, want) {
			t.Fatalf("the wizard lacks %q:\n%s", want, strings.Join(lines, "\n"))
		}
	}
	command := e.WizardField("a0.p1")
	if command < 0 {
		t.Fatal("no command field")
	}
	var names []string
	for _, c := range e.obj.fields[command].Choices {
		names = append(names, c.Label)
	}
	if !slices.Equal(names, []string{"Swarm", "Move", "Swarm 2", "Patrol"}) {
		t.Fatalf("the command choices %q", names)
	}
	order := slices.IndexFunc(wizActions, func(it wizItem) bool { return it.labelWord == "editor.wizard.a.order" })
	row := wizRow{item: order, origItem: -2, vals: e.wizardDefaults(false, order)}
	row.vals[1], row.vals[2], row.vals[3] = "5", "5", "96,84"
	plan, err := e.rowsPlan(WizardTrigger{}, []wizRow{row})
	if err != nil || len(plan.Actions) != 1 {
		t.Fatalf("the plan %+v: %v", plan, err)
	}
	shipped := e.doc.Script.Actions[0]
	if n := plan.Actions[0]; n.Opcode != shipped.Opcode || n.Types != shipped.Types || n.Values != shipped.Values {
		t.Fatalf("the built node %+v, the shipped %+v", n, shipped)
	}
}
