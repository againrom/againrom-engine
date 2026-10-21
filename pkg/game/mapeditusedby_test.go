package game

import (
	"bytes"
	"reflect"
	"strings"
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/mapedit"
	"againrom/pkg/ui"
)

func TestObjectUsedByProductionPopup(t *testing.T) {
	e := parameterEditor(t)
	e.Select(trigRecord(t, e, "Unit", 0), false)
	if got := strings.Join(e.PopupLines(), "\n"); !strings.Contains(got, "Used by") || !strings.Contains(got, "Rule") || !strings.Contains(got, "objects") {
		t.Fatalf("unit popup lacks its trigger and node: %s", got)
	}
}

func usedByEditor(t *testing.T) *ui.MapEditor {
	t.Helper()
	o := synth.ALMOptions{Width: 40, Height: 40,
		Units:   []synth.ALMUnit{{X: 12 * 256, Y: 14 * 256, ClassID: 40}, {X: 15 * 256, Y: 14 * 256, ClassID: 40}},
		Objects: []synth.ALMObject{{X: 8 * 256, Y: 8 * 256, Kind: 1, Field12: 1}, {X: 10 * 256, Y: 8 * 256, Kind: 1, Field12: 2}},
		Type7Payload: goalScript([]scriptNode{
			{label: "Act on objects", opcode: 1, id: 21, tags: []uint32{4, 9, 2, 3}, values: []uint32{10, 1, 20, 1}},
			{label: "Unattached", opcode: 1, id: 22, tags: []uint32{4, 3}, values: []uint32{10, 1}},
		}, []scriptNode{
			{label: "Check objects", opcode: 1, id: 11, tags: []uint32{4, 9, 2, 3, 4, 4, 2, 9, 3, 8}, values: []uint32{10, 1, 20, 1, 10, 10001, 0, 999, 99, 11}},
			{label: "One", opcode: 0x10002, id: 12, tags: []uint32{1}, values: []uint32{1}},
		}, []goalTrigger{
			{name: "First", left: 11, right: 12, actions: []uint32{21, 21}},
			{name: "Second", left: 11, right: 12, actions: []uint32{21}},
		})}
	m := built(t, o, nil, nil)
	rec, err := m.NewPlayerRecord()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := m.Place(mapedit.KindPlayer, rec); err != nil {
		t.Fatal(err)
	}
	e, _ := openGoalMap(t, m.Bytes())
	e.Layout(1280, 800)
	return e
}

func selectUsedByObject(t *testing.T, e *ui.MapEditor, kind string) {
	t.Helper()
	if kind != "Group" {
		e.Select(trigRecord(t, e, kind, 0), false)
		return
	}
	e.Select(trigRecord(t, e, "Action", 0), false)
	for i, text := range e.PopupLines() {
		if text == "Open Group" {
			r, ok := e.PopupRowRect(i)
			trigClick(t, e, r, ok, "Open Group")
			return
		}
	}
	t.Fatal("no Open Group")
}

func TestScriptUsedByKinds(t *testing.T) {
	for _, tc := range []struct {
		kind       string
		tag, value uint32
		slots      []int
	}{
		{"Unit", 4, 10, []int{0, 4, 0, 0}},
		{"Structure", 9, 1, []int{1, 1}},
		{"Group", 2, 20, []int{2, 2}},
		{"Player", 3, 1, []int{3, 3, 1}},
	} {
		t.Run(tc.kind, func(t *testing.T) {
			e := usedByEditor(t)
			m := e.Document().Script
			target := m.OpenTarget(tc.tag, tc.value)
			if target == nil || len(target.Uses) != len(tc.slots) {
				t.Fatalf("%s reverse index: %+v", tc.kind, target)
			}
			for i, use := range target.Uses {
				wantNode := m.Conditions[0].Record
				if tc.kind == "Unit" && i == 2 || tc.kind != "Unit" && i == 1 {
					wantNode = m.Actions[0].Record
				} else if tc.kind == "Unit" && i == 3 || tc.kind == "Player" && i == 2 {
					wantNode = m.Actions[1].Record
				}
				var wantTriggers []int
				if wantNode != m.Actions[1].Record {
					wantTriggers = []int{m.Triggers[0].Record, m.Triggers[1].Record}
				}
				if use.Node != wantNode || use.Slot != tc.slots[i] || !reflect.DeepEqual(use.Triggers, wantTriggers) {
					t.Fatalf("use %d: %+v want node%d slot%d triggers%v", i, use, wantNode, tc.slots[i], wantTriggers)
				}
			}
			before := bytes.Clone(e.Document().SourceBytes)
			selectUsedByObject(t, e, tc.kind)
			lines := e.PopupLines()
			first, second, orphan := -1, -1, -1
			for i, text := range lines {
				if strings.HasPrefix(text, "Trigger 1  First:") {
					first = i
				}
				if strings.HasPrefix(text, "Trigger 2  Second:") {
					second = i
				}
				if strings.HasPrefix(text, "No trigger:") {
					orphan = i
				}
			}
			if first < 0 || second <= first || strings.Count(strings.Join(lines, "\n"), "Trigger 1  First:") != 1 {
				t.Fatal("trigger rows or order", lines)
			}
			if (tc.kind == "Unit" || tc.kind == "Player") != (orphan > second) {
				t.Fatal("unattached node rows", lines)
			}
			for _, name := range []string{"Check objects", "Act on objects"} {
				if !strings.Contains(lines[first], name) {
					t.Fatal("node missing", lines[first])
				}
			}
			for _, row := range []int{second, first} {
				selectUsedByObject(t, e, tc.kind)
				r, ok := e.PopupRowRect(row)
				for k := 0; !ok && k < 30; k++ {
					e.ScrollDetails(1)
					r, ok = e.PopupRowRect(row)
				}
				trigClick(t, e, r, ok, "Used by trigger")
				want := m.Triggers[0].Record
				if row == second {
					want = m.Triggers[1].Record
				}
				if e.Selected() != want {
					t.Fatal("wrong link target", e.Selected(), want)
				}
			}
			if !bytes.Equal(before, e.Document().SourceBytes) {
				t.Fatal("navigation wrote ALM")
			}
		})
	}
}

func TestScriptUsedByEmptyAndUnopenable(t *testing.T) {
	e := usedByEditor(t)
	for _, kind := range []string{"Unit", "Structure", "Player"} {
		e.Select(trigRecord(t, e, kind, 1), false)
		if !strings.Contains(trigLines(e), "Used by: none") {
			t.Fatal(kind, e.PopupLines())
		}
	}
	m := e.Document().Script
	for _, tc := range [][2]uint32{{4, 10001}, {4, 999}, {9, 999}, {3, 99}, {2, 0}, {8, 11}} {
		if target := m.OpenTarget(tc[0], tc[1]); target != nil {
			t.Fatal("unexpected Open", tc, target)
		}
	}
}

func TestScriptUsedByBackPreservesDraft(t *testing.T) {
	e := usedByEditor(t)
	selectUsedByObject(t, e, "Unit")
	before := bytes.Clone(e.Document().SourceBytes)
	field := -1
	for i, f := range e.Document().Edits.Fields("Unit", 0) {
		if len(f.Choices) == 0 && !f.Text {
			field = i
			break
		}
	}
	if field < 0 {
		t.Fatal("no numeric unit field")
	}
	r, ok := e.FieldRect(field)
	trigClick(t, e, r, ok, "unit field")
	for k := 0; k < 12; k++ {
		_ = e.Key("backspace")
	}
	e.TextInput("17")
	var draft string
	for _, line := range e.PopupLines() {
		if strings.Contains(line, "17") {
			draft = line
			break
		}
	}
	if draft == "" {
		t.Fatal("draft not entered", e.PopupLines())
	}
	for i, line := range e.PopupLines() {
		if strings.HasPrefix(line, "Trigger 1  First:") {
			r, ok := e.PopupRowRect(i)
			for k := 0; !ok && k < 30; k++ {
				e.ScrollDetails(1)
				r, ok = e.PopupRowRect(i)
			}
			trigClick(t, e, r, ok, "Used by")
			break
		}
	}
	if e.Selected() != e.Document().Script.Triggers[0].Record {
		t.Fatal("draft prevented navigation", e.PopupLines())
	}
	r, ok = e.PopupButtonRect(5) // after the wizard's Apply, Cancel, New, Delete and Details
	trigClick(t, e, r, ok, "Back")
	if e.Document().Records[e.Selected()].Kind != "Unit" || !strings.Contains(trigLines(e), draft) {
		t.Fatal("draft lost", draft, e.PopupLines())
	}
	if !bytes.Equal(before, e.Document().SourceBytes) || e.Dirty() {
		t.Fatal("draft wrote ALM")
	}
}

func TestScriptUsedByRebuildAfterApplyUndoRedo(t *testing.T) {
	e := usedByEditor(t)
	e.Select(trigRecord(t, e, "Action", 0), false)
	trigChoose(t, e, 2, "Unit 11:")
	r, ok := e.PopupButtonRect(0)
	trigClick(t, e, r, ok, "Apply")
	check := func(old, next int) {
		t.Helper()
		m := e.Document().Script
		if len(m.OpenTarget(4, 10).Uses) != old || len(m.OpenTarget(4, 11).Uses) != next {
			t.Fatalf("reverse index not rebuilt: old=%+v next=%+v", m.OpenTarget(4, 10).Uses, m.OpenTarget(4, 11).Uses)
		}
	}
	check(3, 1)
	if err := e.Key("undo"); err != nil {
		t.Fatal(err)
	}
	check(4, 0)
	if err := e.Key("redo"); err != nil {
		t.Fatal(err)
	}
	check(3, 1)
}
