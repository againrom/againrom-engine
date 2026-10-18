package ui

import (
	"reflect"
	"strings"
	"testing"
)

func scriptsListEditor(t *testing.T) *MapEditor {
	t.Helper()
	e, _ := newModesEditor(t, "full")
	e.doc.Records = []InspectionRecord{
		{Kind: "Action", Index: 0}, {Kind: "Condition", Index: 0},
		{Kind: "Trigger", Index: 0}, {Kind: "Condition", Index: 1},
		{Kind: "Condition", Index: 2}, {Kind: "Action", Index: 1},
	}
	e.doc.Script = &InspectionScriptModel{
		Actions: []InspectionScriptNode{
			{Record: 0, ID: 10, Opcode: 3, Types: [10]uint32{scriptUnit, scriptConst}, Values: [10]uint32{77, 1}, AddressesVariable: true, VariableRegister: 1},
			{Record: 5, ID: 11, Opcode: 4, Label: "Unused action"},
		},
		Conditions: []InspectionScriptNode{
			{Record: 1, ID: 20, Opcode: 5},
			{Record: 3, ID: 21, Opcode: 0x10002, Label: "Counter"},
			{Record: 4, ID: 22, Opcode: 0x10002, Label: "Unused condition"},
		},
		Triggers: []InspectionTrigger{{Record: 2, Name: "Rule", Left: [3]uint32{20}, Right: [3]uint32{21}, Acts: [4]uint32{10}}},
	}
	return e
}

func TestScriptsListHeadingsAndRecordOrder(t *testing.T) {
	e := scriptsListEditor(t)
	_ = e.Key("scripts")
	rows, _ := e.listRows("records")
	want := []string{"Triggers", "Trigger 1  Rule", "Variables", "Variable 1  Counter", "Conditions", "Condition 3  Unused condition", "Actions", "Action 2  Unused action"}
	var got []string
	for _, r := range rows {
		got = append(got, strings.TrimSpace(r.Text))
	}
	for section, name := range []string{"Triggers", "Variables", "Conditions", "Actions"} {
		t.Run(name, func(t *testing.T) {
			at := 2 * section
			if at+1 >= len(rows) || rows[at].Choosable || !rows[at+1].Choosable || got[at] != want[at] || got[at+1] != want[at+1] {
				t.Fatalf("heading and record missing or misplaced: %q", got)
			}
		})
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("rows %q, want %q", got, want)
	}
	if !reflect.DeepEqual(e.rows(), []int{2, -2, 4, 5}) {
		t.Fatal("record order", e.rows())
	}
}

func TestScriptsListEmptyHeadingsSayNone(t *testing.T) {
	e := scriptsListEditor(t)
	e.doc.Records, e.doc.Script = nil, &InspectionScriptModel{}
	_ = e.Key("scripts")
	rows, _ := e.listRows("records")
	want := []PickerRow{{Text: "Triggers: none"}, {Text: "Variables: none"}, {Text: "Conditions: none"}, {Text: "Actions: none"}}
	if !reflect.DeepEqual(rows, want) {
		t.Fatal("empty headings", rows)
	}
}

func TestScriptsListFreeRegister(t *testing.T) {
	e := scriptsListEditor(t)
	e.doc.Script.Actions[0].Values[1] = 67
	e.doc.Script.Actions[0].VariableRegister = 67
	_ = e.Key("scripts")
	rows, _ := e.listRows("records")
	for _, row := range rows {
		if row.Text == "Variable 67" && row.Choosable {
			return
		}
	}
	t.Fatalf("addressed free register missing: %+v", rows)
}

func TestScriptsListVariableLinksAndSharedRegister(t *testing.T) {
	e := scriptsListEditor(t)
	m := e.doc.Script
	m.Conditions[0].AddressesVariable, m.Conditions[0].VariableRegister = true, 67
	m.Actions[1].AddressesVariable, m.Actions[1].VariableRegister = true, 0
	m.Triggers[0].Acts[1] = 11
	_ = e.Key("scripts")
	rows, _ := e.listRows("records")
	for i, want := range []string{"Variable 0 (shared with Condition 1)", "Variable 1  Counter", "Variable 67"} {
		if rows[i+3].Text != want {
			t.Fatalf("row %d = %q, want %q", i+3, rows[i+3].Text, want)
		}
	}
	for _, tc := range []struct {
		id     int
		labels []string
		links  []int
	}{
		{-2, []string{"Changed by triggers", "Trigger 1  Rule", "Read by triggers: none"}, []int{0, -3, 0}},
		{-3, []string{"Changed by triggers", "Trigger 1  Rule", "Read by triggers: none", "Condition 2  Counter"}, []int{0, -3, 0, -4}},
		{-4, []string{"Changed by triggers: none", "Read by triggers", "Trigger 1  Rule"}, []int{0, 0, -3}},
	} {
		e.Select(tc.id, false)
		s, links, ok := e.objectSpec()
		if !ok || !reflect.DeepEqual(links, tc.links) || len(s.Buttons) != 1 || len(s.Rows) != len(tc.labels) || e.objectHasFields() {
			t.Fatalf("variable %d popup: %+v links=%v", tc.id, s, links)
		}
		for i, row := range s.Rows {
			if row.Label != tc.labels[i] || links[i] != 0 && row.Kind != popupLink {
				t.Fatalf("row %d: %+v", i, row)
			}
			if links[i] < 0 {
				e.Select(tc.id, false)
				layout := layoutPopup(s, e.font, e.bodyFont())
				r := layout.Rows[i].Rect
				e.objectPress(r.Min.Add(r.Size().Div(2)))
				if e.Selected() != -links[i]-1 {
					t.Fatal("link did not open target", links[i], e.Selected())
				}
				if _, _, ok := e.objectSpec(); !ok {
					t.Fatal("link target has no popup")
				}
			}
		}
	}
}

func TestScriptsListNoticeAndOverriddenRecords(t *testing.T) {
	e := scriptsListEditor(t)
	e.doc.Script, e.doc.Records = nil, nil
	e.doc.ScriptNotice = "Section 7 absent; no authored script"
	_ = e.Key("scripts")
	s, ok := e.listSpec()
	if !ok || !strings.Contains(strings.Join(s.Notes, " "), e.doc.ScriptNotice) {
		t.Fatal("empty script reason missing", s.Notes)
	}
	e = scriptsListEditor(t)
	e.doc.Records = append(e.doc.Records, InspectionRecord{Kind: "Action", Index: 1}, InspectionRecord{Kind: "Condition", Index: 0}, InspectionRecord{Kind: "Trigger", Index: 0})
	_ = e.Key("scripts")
	if len(e.rows()) != 7 || e.ListRowCount() != 11 {
		t.Fatal("overridden records lost their generic rows", e.rows())
	}
}

func TestScriptsListRowsKeepTheirPopupAndSelection(t *testing.T) {
	e := scriptsListEditor(t)
	_ = e.Key("scripts")
	for n, id := range []int{2, -2, 4, 5} {
		e.Select(id, true)
		before, _, ok := e.objectSpec()
		if !ok {
			t.Fatal("no direct popup", id)
		}
		e.listChoose(2*n+1, true)
		after, _, ok := e.objectSpec()
		if !ok || e.Selected() != id || before.Title != after.Title || !reflect.DeepEqual(before.Rows, after.Rows) || !reflect.DeepEqual(before.Buttons, after.Buttons) {
			t.Fatal("list row changed the record popup", id)
		}
	}
	e.Select(2, false)
	if e.selectedTrigger() != 0 {
		t.Fatal("trigger row lost the area selection")
	}
	e.listChoose(0, true)
	if e.Selected() != 2 {
		t.Fatal("heading selected a record")
	}
	for _, id := range []int{-2, 4, 5} {
		_ = e.Key("down")
		if e.Selected() != id {
			t.Fatalf("Down selected %d, want %d", e.Selected(), id)
		}
	}
	_ = e.Key("up")
	if e.Selected() != 4 {
		t.Fatal("Up did not skip the heading")
	}
}

func TestScriptsListVariableRegisterAndLabel(t *testing.T) {
	for _, opcode := range []uint32{3, 8, 19} {
		e := scriptsListEditor(t)
		e.doc.Script.Actions[0].Opcode = 4
		e.doc.Script.Actions[0].AddressesVariable = false
		if opcode == 19 {
			e.doc.Script.Conditions[0].Opcode = 19
			e.doc.Script.Conditions[0].Types = [10]uint32{scriptInt}
			e.doc.Script.Conditions[0].Values = [10]uint32{1}
			e.doc.Script.Conditions[0].AddressesVariable = true
			e.doc.Script.Conditions[0].VariableRegister = 1
		} else {
			e.doc.Script.Actions[0].Opcode = opcode
			e.doc.Script.Actions[0].AddressesVariable = true
		}
		e.doc.Script.Conditions[1].Label = ""
		_ = e.Key("scripts")
		rows, _ := e.listRows("records")
		if strings.TrimSpace(rows[3].Text) != "Variable 1" {
			t.Fatalf("opcode %d: unnamed variable not named by register: %+v", opcode, rows)
		}
	}
}

func TestScriptsBarAndF2OpenOneListAndAddWizard(t *testing.T) {
	for _, bar := range []bool{false, true} {
		e := scriptsListEditor(t)
		if bar {
			e.barAct("scripts")
		} else {
			_ = e.Key("scripts")
		}
		if e.filter != editorScriptsFilter || e.listKind() != "records" || e.Selected() != 2 {
			t.Fatal("entry did not open Scripts", bar, e.filter, e.Selected())
		}
		for _, b := range e.barEntries() {
			if b.key == "triggers" {
				t.Fatal("old bar entry remains")
			}
		}
		_ = e.Key("player")
		if addKindNow(e) != "Script" {
			t.Fatal("A did not open the script wizard")
		}
	}
}

func TestScriptsRawSectionsAndEveryNodeRemainReachable(t *testing.T) {
	e := scriptsListEditor(t)
	e.doc.Records = append(e.doc.Records,
		InspectionRecord{Kind: "Section", Section: 7, Index: 10, Label: "Raw script"},
		InspectionRecord{Kind: "Section", Section: 7, Index: 11, Label: "Duplicate script"})
	found := false
	for _, item := range e.menuItems("view") {
		if item.key == "records" {
			found = true
			e.menuAct(item.key)
		}
	}
	if !found || e.filter != 0 || !e.listShown || len(e.rows()) != len(e.doc.Records) {
		t.Fatal("View has no route to every physical record")
	}
	for id := range e.doc.Records {
		e.listChoose(id, true)
		if e.Selected() != id {
			t.Fatal("All records lost record", id)
		}
		if _, _, ok := e.objectSpec(); !ok {
			t.Fatal("All records did not open popup", id)
		}
	}
}
