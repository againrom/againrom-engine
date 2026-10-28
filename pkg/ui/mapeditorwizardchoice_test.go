package ui

import (
	"slices"
	"strings"
	"testing"

	"againrom/pkg/render/terrain"
)

// wizardWrites records each trigger the wizard's Apply writes and writes
// nothing.
type wizardWrites struct {
	*editorTextEdits
	plans []WizardTrigger
}

func (w *wizardWrites) WriteTrigger(t WizardTrigger) (*InspectionDocument, error) {
	w.plans = append(w.plans, t)
	return nil, nil
}

// wizardChoiceEditor opens a 64 by 64 map whose one trigger holds a ready
// condition (Always: constant 1 equal to constant 1), a condition no item
// reads, a ready action (Win) and an action no item reads.
func wizardChoiceEditor(t *testing.T) (*MapEditor, *wizardWrites) {
	t.Helper()
	host := &wizardWrites{editorTextEdits: &editorTextEdits{}}
	build := func(string) (*InspectionDocument, error) {
		v, err := NewViewer("wizard", terrain.Grid{Width: 64, Height: 64, Tiles: make([]uint16, 64*64)}, &terrain.Tileset{})
		if err != nil {
			return nil, err
		}
		d := &InspectionDocument{Viewer: v, Source: "wizard", Width: 64, Height: 64, SourceBytes: []byte("map"), Edits: host}
		d.Records = []InspectionRecord{{Kind: "Trigger", Index: 0, Section: 7},
			{Kind: "Condition", Index: 0, Section: 7}, {Kind: "Condition", Index: 1, Section: 7}, {Kind: "Condition", Index: 2, Section: 7},
			{Kind: "Action", Index: 0, Section: 7}, {Kind: "Action", Index: 1, Section: 7}}
		d.Script = &InspectionScriptModel{
			Triggers: []InspectionTrigger{{Record: 0, Name: "Start", Once: 1,
				Left: [3]uint32{1, 3}, Right: [3]uint32{2, 2}, Acts: [4]uint32{11, 12}}},
			Conditions: []InspectionScriptNode{
				{Record: 1, ID: 1, Opcode: scriptConstantOp, Types: [10]uint32{scriptInt}, Values: [10]uint32{1}},
				{Record: 2, ID: 2, Opcode: scriptConstantOp, Types: [10]uint32{scriptInt}, Values: [10]uint32{1}},
				{Record: 3, ID: 3, Opcode: 99}},
			Actions: []InspectionScriptNode{{Record: 4, ID: 11, Opcode: 4}, {Record: 5, ID: 12, Opcode: 99}},
		}
		return d, nil
	}
	e := NewMapEditor(panelFont(), nil, build)
	e.SetBodyFont(panelFont())
	e.Layout(1280, 800)
	if err := e.Open("wizard"); err != nil {
		t.Fatal(err)
	}
	e.Select(0, false)
	if !e.WizardShown() {
		t.Fatalf("the trigger opens no wizard:\n%s", strings.Join(e.PopupLines(), "\n"))
	}
	return e, host
}

// wizardPress presses the popup row that holds field f, or the row whose
// own button is act when f is -1.
func wizardPress(t *testing.T, e *MapEditor, f int, act string) {
	t.Helper()
	if _, _, ok := e.objectSpec(); !ok {
		t.Fatal("the wizard builds no popup")
	}
	if f >= 0 {
		n := e.fieldRow(f)
		if n < 0 {
			t.Fatalf("no row shows field %q", e.obj.fields[f].Key)
		}
		e.objectRowPress(popupRowLine, n)
		return
	}
	for n, acts := range e.obj.rowActs {
		if k := slices.Index(acts, act); k >= 0 {
			e.scriptRowButton(n, k)
			return
		}
	}
	t.Fatalf("no row has %q", act)
}

// wizardChoose opens the drop-down of wizard field key and chooses the
// choice whose value is value, as a click on it does.
func wizardChoose(t *testing.T, e *MapEditor, key string, value int64) {
	t.Helper()
	f := e.WizardField(key)
	if f < 0 {
		t.Fatalf("no field %q", key)
	}
	e.obj.open = -1
	wizardPress(t, e, f, "")
	row := slices.IndexFunc(e.obj.fields[f].Choices, func(c InspectionChoice) bool { return c.Value == value })
	if e.obj.open != f || row < 0 {
		t.Fatalf("field %q: drop-down %d open, choice %d not among %+v", key, e.obj.open, value, e.obj.fields[f].Choices)
	}
	e.chooseObjectChoice(row)
}

// Every choice of a trigger row's item list can be chosen: on a row read
// as a ready item, on a row kept as it stands ("Other", the list's last
// choice), and on a row just added, for conditions and actions. After each
// choice the window shows it and the trigger Apply would write is that
// item's node, or for "Other" the kept nodes; a row chosen back takes the
// texts it was read with, and Apply writes the kept rows unchanged.
func TestTriggerWizardEveryItemChoiceHolds(t *testing.T) {
	e, host := wizardChoiceEditor(t)
	wizardPress(t, e, -1, actAddPair)
	wizardPress(t, e, -1, actAddAct)
	if rows, kept := e.WizardRowCounts(); rows != 6 || kept != 2 {
		t.Fatalf("rows %d, kept %d; want 6 and 2", rows, kept)
	}
	type rowCase struct {
		key        string
		condition  bool
		plan       int
		orig       int64
		keptIDs    [2]uint32
		otherShown bool
	}
	cases := []rowCase{
		{"c0.item", true, 0, int64(len(wizConditions) - 1), [2]uint32{1, 2}, false},
		{"c1.item", true, 1, -1, [2]uint32{3, 2}, true},
		{"c2.item", true, 2, 0, [2]uint32{}, false},
		{"a0.item", false, 0, 1, [2]uint32{11}, false},
		{"a1.item", false, 1, -1, [2]uint32{12}, true},
		{"a2.item", false, 2, 0, [2]uint32{}, false},
	}
	for _, c := range cases {
		items := wizItems(c.condition)
		f := e.WizardField(c.key)
		var values []int64
		for _, ch := range e.obj.fields[f].Choices {
			values = append(values, ch.Value)
		}
		r := -1
		for k := range e.obj.wiz.rows {
			if e.obj.wiz.wizardKey(k, "item") == c.key {
				r = k
			}
		}
		orig := append([]string(nil), e.obj.wiz.rows[r].vals...)
		for _, value := range append(values, c.orig) {
			if e.obj.wiz.shut(r, int(value)) {
				continue
			}
			wizardChoose(t, e, c.key, value)
			f = e.WizardField(c.key)
			if got := e.obj.fields[f].Choices[slices.IndexFunc(e.obj.fields[f].Choices, func(ch InspectionChoice) bool { return ch.Value == value })].Label; choiceText(e.obj.fields[f], e.obj.buf[f]) != got {
				t.Fatalf("%s: chose %q, the window shows %q", c.key, got, choiceText(e.obj.fields[f], e.obj.buf[f]))
			}
			v, _ := e.scriptPopup()
			plan, err := e.wizardPlan(v)
			if err != nil {
				t.Fatalf("%s: choice %d plans no trigger: %v", c.key, value, err)
			}
			if c.condition {
				p := plan.Pairs[c.plan]
				if value < 0 && (!p.Left.Keep || !p.Right.Keep || p.Left.ID != c.keptIDs[0] || p.Right.ID != c.keptIDs[1]) {
					t.Fatalf("%s: Other writes %+v; want nodes %v kept", c.key, p, c.keptIDs)
				}
				if value >= 0 && !p.Left.Keep && p.Left.Opcode != items[value].opcode {
					t.Fatalf("%s: choice %d writes operation %d; want %d", c.key, value, p.Left.Opcode, items[value].opcode)
				}
			} else {
				n := plan.Actions[c.plan]
				if value < 0 && (!n.Keep || n.ID != c.keptIDs[0]) {
					t.Fatalf("%s: Other writes %+v; want node %d kept", c.key, n, c.keptIDs[0])
				}
				if value >= 0 && !n.Keep && n.Opcode != items[value].opcode {
					t.Fatalf("%s: choice %d writes operation %d; want %d", c.key, value, n.Opcode, items[value].opcode)
				}
			}
		}
		if got := e.obj.wiz.rows[r].vals; !slices.Equal(got, orig) {
			t.Fatalf("%s: chosen back, the row holds %q; it was read with %q", c.key, got, orig)
		}
		want := len(items)
		if c.otherShown {
			want++
		}
		if len(values) != want || slices.Contains(values, -1) != c.otherShown {
			t.Fatalf("%s: choices %v; want the %d ready items, Other %v", c.key, values, len(items), c.otherShown)
		}
	}
	e.obj.open = -1
	v, _ := e.scriptPopup()
	e.wizardButton(v, 0)
	if len(host.plans) != 1 {
		t.Fatalf("Apply wrote %d triggers; want 1 (note %q)", len(host.plans), e.obj.note)
	}
	plan := host.plans[0]
	if len(plan.Pairs) != 3 || len(plan.Actions) != 3 {
		t.Fatalf("Apply wrote %d pairs and %d actions; want 3 and 3", len(plan.Pairs), len(plan.Actions))
	}
	for k, ids := range [][2]uint32{{1, 2}, {3, 2}} {
		if p := plan.Pairs[k]; !p.Left.Keep || !p.Right.Keep || p.Left.ID != ids[0] || p.Right.ID != ids[1] || p.Cmp != 0 {
			t.Fatalf("pair %d written as %+v; want nodes %v kept", k, p, ids)
		}
	}
	for k, id := range []uint32{11, 12} {
		if n := plan.Actions[k]; !n.Keep || n.ID != id {
			t.Fatalf("action %d written as %+v; want node %d kept", k, n, id)
		}
	}
	if p, n := plan.Pairs[2], plan.Actions[2]; p.Left.Keep || p.Left.Opcode != wizConditions[0].opcode || n.Keep || n.Opcode != wizActions[0].opcode {
		t.Fatalf("the added rows written as %+v and %+v; want the first ready items", p, n)
	}
}
