package game

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"os"
	"strconv"
	"strings"
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/mapedit"
	"againrom/pkg/ui"
)

func parameterMap(t *testing.T) []byte {
	o := synth.ALMOptions{Width: 40, Height: 40,
		Type7Payload: goalScript([]scriptNode{
			{label: "objects", opcode: 1, id: 1, tags: []uint32{4, 2, 3, 9, 8}, values: []uint32{10, 20, 1, 1, 11}},
			{label: "set", opcode: 3, id: 2, tags: []uint32{1, 1}, values: []uint32{5, 1}},
			{label: "increment", opcode: 8, id: 3, tags: []uint32{1}, values: []uint32{7}},
		}, []scriptNode{
			{label: "Funds", opcode: 0x10002, id: 1, tags: []uint32{1}, values: []uint32{9}},
			{label: "read", opcode: 19, id: 2, tags: []uint32{1}, values: []uint32{0}},
			{label: "Unused", opcode: 0x10002, id: 3, tags: []uint32{1}},
		},
			[]goalTrigger{{name: "Rule", actions: []uint32{1}}}),
		Units:   []synth.ALMUnit{{X: 12 * 256, Y: 14 * 256, ClassID: 40}, {X: 15 * 256, Y: 14 * 256, ClassID: 40}},
		Objects: []synth.ALMObject{{X: 8 * 256, Y: 8 * 256, Kind: 1, Field12: 1}, {X: 10 * 256, Y: 8 * 256, Kind: 1, Field12: 2}}}
	m := built(t, o, nil, nil)
	rec, err := m.NewPlayerRecord()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := m.Place(mapedit.KindPlayer, rec); err != nil {
		t.Fatal(err)
	}
	return m.Bytes()
}

func parameterEditor(t *testing.T) *ui.MapEditor {
	e, _ := openGoalMap(t, parameterMap(t))
	x := objectEdits(t, e).inspector
	x.recordOptions = map[string][]ui.InspectionChoice{"item": {{Value: 0xe23, Label: "3619: Tooth"}, {Value: 0xe24, Label: "3620: Ring"}, {Value: 1, Label: "1: Sword"}}}
	d, err := objectEdits(t, e).project(objectEdits(t, e).model)
	if err != nil {
		t.Fatal(err)
	}
	// Reopen the same synthetic source with its host catalog.
	e = x.Editor()
	if err := e.Open(d.Source); err != nil {
		t.Fatal(err)
	}
	e.Layout(1280, 800)
	closeMapPanel(t, e)
	e.Select(trigRecord(t, e, "Action", 0), false)
	return e
}

func TestScriptParameterChoicesAndTypedBytes(t *testing.T) {
	for _, tc := range []struct {
		kind        string
		tag         uint32
		slot, count int
		value       int64
		name        string
	}{
		{"unit", 4, 0, 1002, 11, "Unit 11: class 40"},
		{"group", 2, 1, 2, 21, "Group 21 (units: 1)"},
		{"player", 3, 2, 2, 2, "Player 2: New Player 2"},
		{"structure", 9, 3, 2, 2, "Structure 2: kind 1"},
		{"item", 8, 4, 3, 12, "12: Ring"},
	} {
		t.Run(tc.kind, func(t *testing.T) {
			e := parameterEditor(t)
			targets := e.Document().Script.Targets[tc.tag]
			if len(targets) != tc.count {
				t.Fatalf("choices %d want %d: %v", len(targets), tc.count, targets)
			}
			key := fmt.Sprintf("value%d", tc.slot)
			var host []ui.InspectionChoice
			for _, f := range e.Document().Edits.Fields("Action", 0) {
				if f.Key == key {
					host = f.Choices
				}
			}
			if len(host) != len(targets) {
				t.Fatalf("E choices %d != targets %d", len(host), len(targets))
			}
			found := false
			for _, c := range host {
				if c.Value == tc.value {
					found = true
					if c.Label != tc.name {
						t.Fatalf("choice %q want whole line %q", c.Label, tc.name)
					}
				}
			}
			if !found {
				t.Fatal("missing choice", tc.value)
			}
			for i, c := range host {
				if c.Value != int64(targets[i].Value) || c.Label != targets[i].ChoiceText() {
					t.Fatal("different builders", c, targets[i])
				}
			}
			before := bytes.Clone(e.Document().SourceBytes)
			trigChoose(t, e, 2+tc.slot, tc.name)
			if !bytes.Equal(before, e.Document().SourceBytes) {
				t.Fatal("pick committed before Apply")
			}
			r, ok := e.PopupButtonRect(0)
			trigClick(t, e, r, ok, "Apply")
			picked := bytes.Clone(e.Document().SourceBytes)
			if bytes.Equal(before, picked) {
				t.Fatal("pick wrote nothing")
			}
			if err := e.Key("undo"); err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(before, e.Document().SourceBytes) {
				t.Fatal("Undo bytes")
			}
			scriptWindowSet(t, e, "Action", 0, key, strconv.FormatInt(tc.value, 10))
			if !bytes.Equal(picked, e.Document().SourceBytes) {
				t.Fatal("pick bytes differ from E prompt")
			}
		})
	}
}

func TestScriptParameterSummaryUsesRecordNames(t *testing.T) {
	e := parameterEditor(t)
	e.Select(trigRecord(t, e, "Trigger", 0), false)
	want := "Unit 10 class 40 (12,14), Group 20 (units: 1), Player 1: Self, Structure 1 kind 1 (8,8), Item 11 Tooth"
	for _, line := range e.PopupLines() {
		if line == want {
			return
		}
	}
	t.Fatalf("missing whole summary %q:\n%s", want, trigLines(e))
}

func TestScriptNewVariableSkipsConditionSlots(t *testing.T) {
	for _, count := range []int{3, 99, 100} {
		t.Run(strconv.Itoa(count), func(t *testing.T) {
			conditions := make([]scriptNode, count)
			for i := range conditions {
				conditions[i] = scriptNode{label: "Constant", opcode: 0x10002, id: uint32(i + 1), tags: []uint32{1}}
			}
			actions := []scriptNode{{opcode: 3, id: 1, tags: []uint32{1, 1}, values: []uint32{0, 1}}}
			e, _ := openGoalMap(t, built(t, synth.ALMOptions{Width: 40, Height: 40, Type7Payload: goalScript(actions, conditions, nil)}, nil, nil).Bytes())
			found := false
			for _, target := range e.Document().Script.Targets[ui.ScriptVariableParameter] {
				if target.Record != -1 {
					continue
				}
				found = true
				if count == 100 || target.Value != uint32(count) {
					t.Fatalf("New variable register %d, condition slots %d", target.Value, count)
				}
			}
			if count < 100 && !found {
				t.Fatal("free register omitted")
			}
		})
	}
}

func TestScriptParameterOpenBackStack(t *testing.T) {
	e := parameterEditor(t)
	e.Select(trigRecord(t, e, "Trigger", 0), false)
	r, ok := e.RowButtonRect(11, 0, false)
	trigClick(t, e, r, ok, "Edit")
	if e.Document().Records[e.Selected()].Kind != "Action" {
		t.Fatal("Edit did not open node")
	}
	lines := e.PopupLines()
	at := -1
	for i, line := range lines {
		if strings.HasPrefix(line, "Open Unit") {
			at = i
			break
		}
	}
	if at < 0 {
		t.Fatal("no Open Unit link", lines)
	}
	r, ok = e.PopupRowRect(at)
	trigClick(t, e, r, ok, "Open unit")
	if e.Document().Records[e.Selected()].Kind != "Unit" {
		t.Fatal("no unit")
	}
	r, ok = e.PopupButtonRect(2)
	trigClick(t, e, r, ok, "Back to node")
	if e.Document().Records[e.Selected()].Kind != "Action" {
		t.Fatal("no node after Back")
	}
	r, ok = e.PopupButtonRect(2)
	trigClick(t, e, r, ok, "Back to trigger")
	if e.Document().Records[e.Selected()].Kind != "Trigger" {
		t.Fatal("no trigger after Back")
	}
}

func TestScriptParameterOpenKindsAndCancel(t *testing.T) {
	for _, tc := range []struct {
		label, kind string
		field       int
	}{{"Unit", "Unit", 2}, {"Group", "", 3}, {"Player", "Player", 4}, {"Structure", "Structure", 5}} {
		t.Run(tc.label, func(t *testing.T) {
			e := parameterEditor(t)
			before := bytes.Clone(e.Document().SourceBytes)
			row := -1
			for i, line := range e.PopupLines() {
				if line == "Open "+tc.label {
					row = i
					break
				}
			}
			if row < 0 {
				t.Fatal("missing Open", tc.label, e.PopupLines())
			}
			r, ok := e.PopupRowRect(row)
			trigClick(t, e, r, ok, "Open")
			if tc.kind == "" {
				if e.Selected() >= 0 || len(e.GroupMembers()) != 1 {
					t.Fatal("group popup", e.Selected(), e.GroupMembers())
				}
			} else if e.Document().Records[e.Selected()].Kind != tc.kind {
				t.Fatal("wrong record")
			}
			back := 2
			if tc.kind == "" {
				back = 0
			}
			r, ok = e.PopupButtonRect(back)
			trigClick(t, e, r, ok, "Back")
			if e.Document().Records[e.Selected()].Kind != "Action" || !bytes.Equal(before, e.Document().SourceBytes) {
				t.Fatal("Back changed map or lost node")
			}
		})
	}
}

func TestScriptParameterUnknownValueAndHeroHaveNoOpen(t *testing.T) {
	e := parameterEditor(t)
	s := objectEdits(t, e)
	d, err := s.SetField("Action", 0, "value0", 10003)
	if err != nil {
		t.Fatal(err)
	}
	path := e.Document().Source
	if err := os.WriteFile(path, d.SourceBytes, 0o600); err != nil {
		t.Fatal(err)
	}
	e = s.inspector.Editor()
	if err := e.Open(path); err != nil {
		t.Fatal(err)
	}
	if e.PanelOpen() {
		closeMapPanel(t, e)
	}
	e.Select(trigRecord(t, e, "Action", 0), false)
	for _, line := range e.PopupLines() {
		if line == "Open Unit" || line == "Open Item" {
			t.Fatal("non-record has Open", line)
		}
	}
	if !strings.Contains(trigLines(e), "hero 3") {
		t.Fatal("hero name", trigLines(e))
	}
	s = objectEdits(t, e)
	x := s.inspector
	x.recordOptions["item"] = nil
	if err := e.Open(path); err != nil {
		t.Fatal(err)
	}
	if e.PanelOpen() {
		closeMapPanel(t, e)
	}
	e.Select(trigRecord(t, e, "Action", 0), false)
	before := bytes.Clone(e.Document().SourceBytes)
	chooseParameterValue(t, e, 6, 11, nil)
	r, ok := e.PopupButtonRect(0)
	trigClick(t, e, r, ok, "Apply")
	if !bytes.Equal(before, e.Document().SourceBytes) {
		t.Fatal("unknown current value was rewritten")
	}
}

func TestScriptParameterPickCancelAndDirtyNavigation(t *testing.T) {
	e := parameterEditor(t)
	before := bytes.Clone(e.Document().SourceBytes)
	trigChoose(t, e, 2, "Unit 11:")
	for i, line := range e.PopupLines() {
		if line == "Open Unit" {
			r, ok := e.PopupRowRect(i)
			trigClick(t, e, r, ok, "Open with dirty node")
			break
		}
	}
	if e.Document().Records[e.Selected()].Kind != "Action" || !strings.Contains(trigLines(e), "Apply") {
		t.Fatal("dirty navigation lost node", trigLines(e))
	}
	r, ok := e.PopupButtonRect(1)
	trigClick(t, e, r, ok, "Cancel")
	if !bytes.Equal(before, e.Document().SourceBytes) || e.Dirty() {
		t.Fatal("Cancel wrote bytes")
	}
	e.Select(trigRecord(t, e, "Action", 0), false)
	if !strings.Contains(trigLines(e), "Unit 10:") {
		t.Fatal("Cancel retained pick", trigLines(e))
	}
}

func TestScriptVariableChoicesAndTypedBytes(t *testing.T) {
	e := parameterEditor(t)
	e.Select(trigRecord(t, e, "Action", 1), false)
	targets := e.Document().Script.Targets[ui.ScriptVariableParameter]
	if len(targets) != 4 || targets[0].Name != "Funds" || targets[1].Name != "Variable 5" || targets[2].Name != "Variable 7" || targets[3].Value != 3 || targets[3].Name != "New variable" {
		t.Fatalf("variable choices %v", targets)
	}
	var host []ui.InspectionChoice
	for _, f := range e.Document().Edits.Fields("Action", 1) {
		if f.Key == "value0" {
			host = f.Choices
		}
	}
	if len(host) != len(targets) {
		t.Fatal("E variable choices", host)
	}
	for i, c := range host {
		if c.Value != int64(targets[i].Value) || c.Label != targets[i].ChoiceText() {
			t.Fatal("E differs", host, targets)
		}
	}
	before := bytes.Clone(e.Document().SourceBytes)
	e.Select(trigRecord(t, e, "Action", 1), false)
	trigChoose(t, e, 2, "Funds")
	if !bytes.Equal(before, e.Document().SourceBytes) {
		t.Fatal("pick wrote before Apply")
	}
	r, ok := e.PopupButtonRect(0)
	trigClick(t, e, r, ok, "Apply variable")
	picked := bytes.Clone(e.Document().SourceBytes)
	if bytes.Equal(picked, before) {
		t.Fatal("variable pick wrote nothing")
	}
	if err := e.Key("undo"); err != nil || !bytes.Equal(before, e.Document().SourceBytes) {
		t.Fatal("variable Undo", err)
	}
	scriptWindowSet(t, e, "Action", 1, "value0", "0")
	if !bytes.Equal(picked, e.Document().SourceBytes) {
		t.Fatal("variable E bytes differ")
	}
}

func TestScriptNewVariableAndOpenStack(t *testing.T) {
	e := parameterEditor(t)
	e.Select(trigRecord(t, e, "Action", 1), false)
	before := bytes.Clone(e.Document().SourceBytes)
	trigChoose(t, e, 2, "New variable")
	if !bytes.Equal(before, e.Document().SourceBytes) {
		t.Fatal("New variable wrote before Apply")
	}
	r, ok := e.PopupButtonRect(1)
	trigClick(t, e, r, ok, "Cancel new variable")
	if !bytes.Equal(before, e.Document().SourceBytes) {
		t.Fatal("Cancel new variable wrote")
	}
	e.Select(trigRecord(t, e, "Action", 1), false)
	trigChoose(t, e, 2, "Funds")
	r, ok = e.PopupButtonRect(0)
	trigClick(t, e, r, ok, "Apply variable")
	for i, line := range e.PopupLines() {
		if line == "Open Variable" {
			r, ok := e.PopupRowRect(i)
			trigClick(t, e, r, ok, "Open variable")
			break
		}
	}
	if e.Selected() >= -1 {
		t.Fatal("not the Scripts variable popup", e.Selected())
	}
	for i, line := range e.PopupLines() {
		if strings.Contains(line, "Funds") {
			r, ok := e.PopupRowRect(i)
			trigClick(t, e, r, ok, "Open Constant")
			break
		}
	}
	if e.Selected() < 0 || e.Document().Records[e.Selected()].Kind != "Condition" {
		t.Fatal("no Constant node")
	}
	r, ok = e.PopupButtonRect(2)
	trigClick(t, e, r, ok, "Back to variable")
	if e.Selected() >= -1 {
		t.Fatal("lost variable on Back")
	}
	r, ok = e.PopupButtonRect(1)
	trigClick(t, e, r, ok, "Back to action")
	if e.Selected() != trigRecord(t, e, "Action", 1) {
		t.Fatal("lost action on Back")
	}
	trigChoose(t, e, 2, "New variable")
	r, ok = e.PopupButtonRect(0)
	trigClick(t, e, r, ok, "Apply new variable")
	if e.Document().Script.Actions[1].Values[0] != 3 {
		t.Fatal("wrong new register")
	}
	if len(e.Document().Script.Conditions) != 3 {
		t.Fatal("new register added a Constant")
	}
}

func TestScriptVariablePickerUsesHostRegisterSlot(t *testing.T) {
	e := parameterEditor(t)
	s := objectEdits(t, e)
	d, err := s.SetFields("Action", 1, []ui.InspectionFieldValue{{Key: "type0", Value: 4}, {Key: "value0", Value: 10}, {Key: "type1", Value: 7}, {Key: "value1", Value: 5}})
	if err != nil {
		t.Fatal(err)
	}
	path := e.Document().Source
	if err := os.WriteFile(path, d.SourceBytes, 0o600); err != nil {
		t.Fatal(err)
	}
	e = s.inspector.Editor()
	if err := e.Open(path); err != nil {
		t.Fatal(err)
	}
	e.Layout(1280, 800)
	if e.PanelOpen() {
		closeMapPanel(t, e)
	}
	e.Select(trigRecord(t, e, "Action", 1), false)
	trigChoose(t, e, 3, "Funds")
	r, ok := e.PopupButtonRect(0)
	trigClick(t, e, r, ok, "Apply register slot")
	n := e.Document().Script.Actions[1]
	if n.VariableRegister != 0 || n.Values[0] != 10 || n.Values[1] != 0 {
		t.Fatal("picker did not follow host register reader", n)
	}
}

func TestScriptUnknownTagDoesNotBecomeVariable(t *testing.T) {
	b := parameterMap(t)
	binary.LittleEndian.PutUint32(b[payloadOffsets(t, b)[7]+4+0x74:], 10)
	binary.LittleEndian.PutUint32(b[payloadOffsets(t, b)[7]+4+0x4c:], 5)
	for _, host := range []bool{true, false} {
		t.Run(strconv.FormatBool(host), func(t *testing.T) {
			e, _ := openGoalMap(t, b)
			if !host {
				e.Document().Script.Catalog.ParameterKind = nil
			}
			e.Select(trigRecord(t, e, "Action", 0), false)
			r, ok := e.FieldRect(2)
			trigClick(t, e, r, ok, "unknown parameter")
			if choices := e.DropdownChoices(); len(choices) != 0 {
				t.Fatal("unknown ALM tag became a variable picker", choices)
			}
			for _, line := range e.PopupLines() {
				if line == "Open Kind 10" {
					t.Fatal("unknown ALM tag has a variable link")
				}
			}
			r, ok = e.PopupButtonRect(0)
			trigClick(t, e, r, ok, "unchanged Apply")
			if !bytes.Equal(b, e.Document().SourceBytes) {
				t.Fatal("unknown tag bytes changed")
			}
			e.Select(trigRecord(t, e, "Trigger", 0), false)
			if lines := trigLines(e); strings.Contains(lines, "Variable 5") || !strings.Contains(lines, "Kind 10 5") {
				t.Fatal("unknown ALM tag gained meaning in the trigger summary", lines)
			}
		})
	}
}
