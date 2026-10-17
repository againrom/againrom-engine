package game

import (
	"bytes"
	"image"
	"strings"
	"testing"
	"time"

	"againrom/internal/synth"
	"againrom/pkg/formats/alm"
	"againrom/pkg/ui"
)

// triggerMap is a 40x40 map of four units (unit ids 10..13, groups 20..23)
// and a script of two triggers: "Near" fires when unit 10 comes within 2 of
// cell 5,6 and shows message 7; "Far" fires once player 1 is within 4 of
// cell 20,20 and wins and casts from 1,2 at 30,31.
func triggerMap(t *testing.T) []byte {
	t.Helper()
	conditions := []scriptNode{
		{label: "near", opcode: 3, id: 1, values: []uint32{10, 5, 6, 2}, tags: []uint32{4, 5, 6, 1}},
		{label: "one", opcode: 0x10002, id: 2, values: []uint32{1}, tags: []uint32{1}},
		{label: "far", opcode: 15, id: 3, values: []uint32{1, 20, 20}, tags: []uint32{3, 5, 6}},
		{label: "four", opcode: 0x10002, id: 4, values: []uint32{4}, tags: []uint32{1}},
	}
	actions := []scriptNode{
		{label: "talk", opcode: 2, id: 1, values: []uint32{7}, tags: []uint32{1}},
		{label: "win", opcode: 4, id: 2},
		{label: "cast", opcode: 21, id: 3, values: []uint32{1, 2, 30, 31, 3, 0}, tags: []uint32{5, 6, 5, 6, 1, 1}},
		{label: "drop", opcode: 0x10002, id: 4, values: []uint32{16, 16}, tags: []uint32{5, 6}},
	}
	o := synth.ALMOptions{Width: 40, Height: 40, Type7Payload: goalScript(actions, conditions, []goalTrigger{
		{name: "Near", left: 1, right: 2, actions: []uint32{1}},
		{name: "Far", left: 3, right: 4, actions: []uint32{2, 3}},
	})}
	for i := 0; i < 4; i++ {
		o.Units = append(o.Units, synth.ALMUnit{X: uint32(12+3*i)*256 + 128, Y: 14*256 + 128, ClassID: 40, ClassSubID: 2})
	}
	return built(t, o, nil, nil).Bytes()
}

// trigClick presses and releases at the centre of r.
func trigClick(t *testing.T, e *ui.MapEditor, r image.Rectangle, ok bool, what string) {
	t.Helper()
	if !ok {
		t.Fatalf("%s is not shown", what)
	}
	p := image.Pt((r.Min.X+r.Max.X)/2, (r.Min.Y+r.Max.Y)/2)
	now := time.Unix(int64(p.X*1000+p.Y), 0)
	e.Pointer(ui.Input{CursorX: p.X, CursorY: p.Y, PrimaryDown: true}, now)
	e.Pointer(ui.Input{CursorX: p.X, CursorY: p.Y}, now)
}

// trigRecord is the document record of the script record kind index.
func trigRecord(t *testing.T, e *ui.MapEditor, kind string, index int) int {
	t.Helper()
	if r := recordOf(e.Document(), kind, index); r >= 0 {
		return r
	}
	t.Fatalf("no %s %d", kind, index)
	return -1
}

// trigChoose opens field n's drop-down and chooses the row whose text holds
// want.
func trigChoose(t *testing.T, e *ui.MapEditor, n int, want string) {
	t.Helper()
	r, ok := e.FieldRect(n)
	trigClick(t, e, r, ok, "field")
	choices := e.DropdownChoices()
	for row, c := range choices {
		if strings.Contains(c, want) {
			for k := 0; k < 30; k++ {
				if r, ok := e.DropdownRowRect(row); ok {
					trigClick(t, e, r, ok, "choice "+want)
					return
				}
				e.ScrollDropdown(1)
			}
		}
	}
	t.Fatalf("field %d offers no %q among %q", n, want, choices)
}

func trigLines(e *ui.MapEditor) string { return strings.Join(e.PopupLines(), "\n") }

func TestTriggerPopupReadsAsTheRule(t *testing.T) {
	e, _ := openGoalMap(t, triggerMap(t))
	e.Select(trigRecord(t, e, "Trigger", 1), true)
	got := trigLines(e)
	for _, want := range []string{
		"Name Far", "Fire once Yes: fires once", "Conditions: all must hold",
		"Pair 1 #3 Distance from cell to player's nearest unit", "Player 1:",
		"Cell 20,20", "Comparison equals", "Compared with #4 Number", "Value 4",
		"Actions: run in order", "Action 1 #2 Win", "Action 2 #3 Cast spell from cell at cell",
		"Cell 1,2, Target cell 30,31, Spell 3, Power 0",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("the popup lacks %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, "Pair 2") || strings.Contains(got, "Action 3") {
		t.Errorf("empty pairs or action slots are shown:\n%s", got)
	}
}

// Each popup edit writes, on Apply, the bytes the host's own field and
// record calls write for the same change, as one Undo step.
func TestTriggerPopupApplyWritesTheFieldCallBytes(t *testing.T) {
	b := triggerMap(t)
	e, _ := openGoalMap(t, b)
	c, _ := openGoalMap(t, b)
	set := c.Document().Edits.(ui.InspectionFieldSet)
	author := c.Document().Edits.(ui.InspectionScriptAuthor)
	e.Select(trigRecord(t, e, "Trigger", 1), true)
	apply := func(want func() *ui.InspectionDocument) {
		t.Helper()
		before := e.Document().SourceBytes
		r, ok := e.PopupButtonRect(0)
		trigClick(t, e, r, ok, "Apply")
		got := e.Document().SourceBytes
		w := want()
		if bytes.Equal(got, before) || !bytes.Equal(got, w.SourceBytes) {
			t.Fatalf("the popup wrote other bytes than the field calls: %v\n%s", bytes.Equal(got, before), trigLines(e))
		}
		if err := e.Key("undo"); err != nil || !bytes.Equal(e.Document().SourceBytes, before) {
			t.Fatal("one Undo did not restore the bytes", err)
		}
		if err := e.Key("redo"); err != nil || !bytes.Equal(e.Document().SourceBytes, got) {
			t.Fatal("Redo did not restore the edit", err)
		}
	}
	fields := func(values ...ui.InspectionFieldValue) func() *ui.InspectionDocument {
		return func() *ui.InspectionDocument {
			d, err := set.SetFields("Trigger", 1, values)
			if err != nil || d == nil {
				t.Fatal("SetFields", err)
			}
			return d
		}
	}

	// A comparison.
	trigChoose(t, e, 3, "is at most")
	apply(fields(ui.InspectionFieldValue{Key: "cmp0", Value: 5}))

	// The once switch.
	r, ok := e.FieldRect(1)
	trigClick(t, e, r, ok, "once")
	apply(fields(ui.InspectionFieldValue{Key: "once", Value: 0}))

	// An action removed.
	r, ok = e.RowButtonRect(11, 1, false)
	trigClick(t, e, r, ok, "Remove action 1")
	apply(fields(ui.InspectionFieldValue{Key: "action0", Value: 0}))

	// A pair added from existing conditions.
	r, ok = e.RowButtonRect(-1, 0, false)
	trigClick(t, e, r, ok, "Add condition pair")
	trigChoose(t, e, 5, "#1 Unit near cell")
	trigChoose(t, e, 7, "#2 Number")
	apply(fields(ui.InspectionFieldValue{Key: "left1", Value: 1}, ui.InspectionFieldValue{Key: "right1", Value: 2}))

	// The first pair removed: the second moves up.
	r, ok = e.RowButtonRect(3, 0, false)
	trigClick(t, e, r, ok, "Remove pair 1")
	apply(fields(ui.InspectionFieldValue{Key: "left0", Value: 1}, ui.InspectionFieldValue{Key: "cmp0", Value: 0}, ui.InspectionFieldValue{Key: "right0", Value: 2},
		ui.InspectionFieldValue{Key: "left1", Value: 0}, ui.InspectionFieldValue{Key: "right1", Value: 0}))

	// An action added as a new node.
	r, ok = e.RowButtonRect(-1, 0, true)
	trigClick(t, e, r, ok, "Add action")
	trigChoose(t, e, 11, "New action")
	apply(func() *ui.InspectionDocument {
		if _, err := author.AddScript("Action"); err != nil {
			t.Fatal(err)
		}
		return fields(ui.InspectionFieldValue{Key: "action0", Value: 5})()
	})

	// A new trigger, then its deletion.
	before := e.Document().SourceBytes
	r, ok = e.PopupButtonRect(2)
	trigClick(t, e, r, ok, "New")
	d, err := author.AddScript("Trigger")
	if err != nil || !bytes.Equal(e.Document().SourceBytes, d.SourceBytes) {
		t.Fatal("New wrote other bytes than AddScript", err)
	}
	if !strings.Contains(trigLines(e), "Name Trigger") || !strings.Contains(trigLines(e), "never runs") {
		t.Fatalf("the new trigger's popup:\n%s", trigLines(e))
	}
	r, ok = e.PopupButtonRect(3)
	trigClick(t, e, r, ok, "Delete")
	d, err = c.Document().Edits.Delete("Trigger", 2)
	if err != nil || !bytes.Equal(e.Document().SourceBytes, d.SourceBytes) || !bytes.Equal(d.SourceBytes, before) {
		t.Fatal("Delete wrote other bytes than the record call", err)
	}
}

func TestTriggerPopupRefusesAHalfPairWithItsReason(t *testing.T) {
	b := triggerMap(t)
	e, _ := openGoalMap(t, b)
	e.Select(trigRecord(t, e, "Trigger", 0), true)
	r, ok := e.RowButtonRect(-1, 0, false)
	trigClick(t, e, r, ok, "Add condition pair")
	trigChoose(t, e, 5, "#3 Distance")
	r, ok = e.PopupButtonRect(0)
	trigClick(t, e, r, ok, "Apply")
	if !strings.Contains(trigLines(e), "Pair 2 needs a condition on both sides.") || !bytes.Equal(e.Document().SourceBytes, b) || e.Document().Edits.CanUndo() {
		t.Fatalf("a half pair was not refused:\n%s", trigLines(e))
	}
	r, ok = e.PopupButtonRect(1)
	trigClick(t, e, r, ok, "Cancel")
	if strings.Contains(trigLines(e), "Pair 2") {
		t.Fatalf("Cancel kept the added pair:\n%s", trigLines(e))
	}
}

// A condition opens from its trigger, edits each parameter by its control
// and writes the bytes of the host's field calls; Back returns.
func TestNodePopupEditsByMeaningAndWritesTheFieldCallBytes(t *testing.T) {
	b := triggerMap(t)
	e, _ := openGoalMap(t, b)
	c, _ := openGoalMap(t, b)
	e.Select(trigRecord(t, e, "Trigger", 0), true)
	r, ok := e.RowButtonRect(2, 0, false)
	trigClick(t, e, r, ok, "Edit pair 1 left")
	got := trigLines(e)
	for _, want := range []string{"Operation", "Unit near cell", "Parameters", "Unit 10", "Cell 5,6", "Radius 2"} {
		if !strings.Contains(got, want) {
			t.Fatalf("the condition popup lacks %q:\n%s", want, got)
		}
	}
	// The unit through its drop-down, the cell by a pick on the map and the
	// radius by typing.
	trigChoose(t, e, 2, "13")
	r, ok = e.RowButtonRect(3, 0, false)
	trigClick(t, e, r, ok, "On map")
	cell := e.CellPoint(image.Pt(8, 9))
	trigClick(t, e, image.Rect(cell.X-1, cell.Y-1, cell.X+1, cell.Y+1), true, "cell 8,9")
	r, ok = e.FieldRect(4)
	trigClick(t, e, r, ok, "radius")
	_ = e.Key("backspace")
	e.TextInput("6")
	r, ok = e.PopupButtonRect(0)
	trigClick(t, e, r, ok, "Apply")
	want, err := c.Document().Edits.(ui.InspectionFieldSet).SetFields("Condition", 0, []ui.InspectionFieldValue{
		{Key: "value0", Value: 13}, {Key: "value1", Value: 8}, {Key: "value2", Value: 9}, {Key: "value3", Value: 6}})
	if err != nil || !bytes.Equal(e.Document().SourceBytes, want.SourceBytes) {
		t.Fatalf("the node popup wrote other bytes than SetFields: %v\n%s", err, trigLines(e))
	}

	// A new operation brings its parameters: unit alive keeps the unit.
	trigChoose(t, e, 1, "Unit alive")
	if got := trigLines(e); strings.Contains(got, "Radius") || !strings.Contains(got, "Unit 13") {
		t.Fatalf("the operation's parameters:\n%s", got)
	}
	r, ok = e.PopupButtonRect(0)
	trigClick(t, e, r, ok, "Apply")
	want, err = c.Document().Edits.(ui.InspectionFieldSet).SetFields("Condition", 0, []ui.InspectionFieldValue{
		{Key: "type1"}, {Key: "type2"}, {Key: "type3"}, {Key: "opcode", Value: 5}})
	if err != nil || !bytes.Equal(e.Document().SourceBytes, want.SourceBytes) {
		t.Fatalf("the new operation wrote other bytes than SetFields: %v\n%s", err, trigLines(e))
	}
	m, _ := alm.Open(e.Document().SourceBytes)
	sc, _ := m.Script()
	if n := sc.Conditions[0]; n.Opcode != 5 || n.Type != [10]uint32{4} || n.Value[0] != 13 {
		t.Fatalf("condition 0 reads %+v", n)
	}
	r, ok = e.PopupButtonRect(2)
	trigClick(t, e, r, ok, "Back")
	if !strings.Contains(trigLines(e), "Name Near") || !strings.Contains(trigLines(e), "#1 Unit alive") {
		t.Fatalf("Back did not return to the trigger:\n%s", trigLines(e))
	}
}

func TestTriggerAreasShowSelectAndHide(t *testing.T) {
	e, _ := openGoalMap(t, triggerMap(t))
	near, far := e.TriggerAreas(0), e.TriggerAreas(1)
	if len(near) != 1 || near[0].Rect != image.Rect(3, 4, 8, 9) || near[0].Shape != ui.AreaSquare {
		t.Fatalf("trigger 0 areas %+v", near)
	}
	// Far: player 1 within 4 of 20,20 (equals 4) and the cast's two cells.
	if len(far) != 3 || far[0].Rect != image.Rect(16, 16, 25, 25) || far[1].Rect != image.Rect(1, 2, 2, 3) || far[2].Rect != image.Rect(30, 31, 31, 32) {
		t.Fatalf("trigger 1 areas %+v", far)
	}
	e.Fit()
	r, ok := e.AreaLabelRect(1)
	trigClick(t, e, r, ok, "the Far label")
	if v := e.Document().Records[e.Selected()]; v.Kind != "Trigger" || v.Index != 1 {
		t.Fatalf("the label selected %+v", v)
	}
	p := e.CellPoint(image.Pt(5, 6))
	trigClick(t, e, image.Rect(p.X, p.Y, p.X+1, p.Y+1), true, "inside Near")
	if v := e.Document().Records[e.Selected()]; v.Kind != "Trigger" || v.Index != 0 {
		t.Fatalf("a click in the area selected %+v", v)
	}
	e.Select(trigRecord(t, e, "Trigger", 1), true)
	r, ok = e.BarRect("view")
	trigClick(t, e, r, ok, "View")
	r, ok = e.MenuRowRect(4)
	trigClick(t, e, r, ok, "Trigger areas")
	if _, ok := e.AreaLabelRect(1); ok {
		t.Fatal("the View switch did not hide the areas")
	}
}
