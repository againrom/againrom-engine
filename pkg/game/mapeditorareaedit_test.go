package game

import (
	"bytes"
	"encoding/binary"
	"image"
	"strings"
	"testing"
	"time"

	"againrom/internal/synth"
	"againrom/pkg/formats/alm"
	"againrom/pkg/mapedit"
	"againrom/pkg/mapload"
	"againrom/pkg/ui"
)

// areaTrigger is a trigger of areaScript: up to three pairs as left, right
// and comparison, and its actions.
type areaTrigger struct {
	name  string
	pairs [][3]uint32
	acts  []uint32
}

// areaScript is goalScript with every pair and comparison a trigger has.
func areaScript(actions, conditions []scriptNode, triggers []areaTrigger) []byte {
	b := scriptPayload(actions, conditions)
	b = b[:len(b)-4]
	b = binary.LittleEndian.AppendUint32(b, uint32(len(triggers)))
	for _, t := range triggers {
		rec := make([]byte, 184)
		copy(rec, t.name)
		for k, p := range t.pairs {
			binary.LittleEndian.PutUint32(rec[0x80+8*k:], p[0])
			binary.LittleEndian.PutUint32(rec[0x84+8*k:], p[1])
			binary.LittleEndian.PutUint32(rec[0xa8+4*k:], p[2])
		}
		for i, id := range t.acts {
			binary.LittleEndian.PutUint32(rec[0x98+4*i:], id)
		}
		binary.LittleEndian.PutUint32(rec[0xb4:], 1)
		b = append(b, rec...)
	}
	return b
}

// areaEditMap is a 40x40 map whose triggers draw every handle kind:
// T1 "Box" a box 4,5..9,8 with a cast at 12,25 and 14,25 and a move to
// 33,33; T2 "Near" a check 3 square of radius 3 at 20,20 and a check 15
// square at 30,10 whose radius is the constant 4; T3 "Far" the same check 15
// pair and the same cast. The constant 4 is compared by two pairs, the cast
// is run by two triggers. withOne adds an unshared constant 1.
func areaEditMap(t *testing.T, withOne bool) []byte {
	t.Helper()
	one := scriptNode{label: "one", opcode: 0x10002, id: 2, values: []uint32{1}, tags: []uint32{1}}
	if !withOne {
		one.values = []uint32{0}
	}
	conditions := []scriptNode{
		{label: "box", opcode: 2, id: 1, values: []uint32{10001, 4, 5, 9, 8}, tags: []uint32{4, 5, 6, 5, 6}},
		one,
		{label: "near", opcode: 3, id: 3, values: []uint32{10001, 20, 20, 3}, tags: []uint32{4, 5, 6, 1}},
		{label: "far", opcode: 15, id: 4, values: []uint32{1, 30, 10}, tags: []uint32{3, 5, 6}},
		{label: "four", opcode: 0x10002, id: 5, values: []uint32{4}, tags: []uint32{1}},
	}
	actions := []scriptNode{
		{label: "cast", opcode: 21, id: 1, values: []uint32{12, 25, 14, 25, 3, 0}, tags: []uint32{5, 6, 5, 6, 1, 1}},
		{label: "talk", opcode: 2, id: 2, values: []uint32{7}, tags: []uint32{1}},
		{label: "move", opcode: 27, id: 3, values: []uint32{10001, 33, 33}, tags: []uint32{4, 5, 6}},
	}
	o := synth.ALMOptions{Width: 40, Height: 40, Type7Payload: areaScript(actions, conditions, []areaTrigger{
		{name: "Box", pairs: [][3]uint32{{1, 2, 0}}, acts: []uint32{1, 3}},
		{name: "Near", pairs: [][3]uint32{{3, 2, 0}, {4, 5, 0}}, acts: []uint32{2}},
		{name: "Far", pairs: [][3]uint32{{4, 5, 0}}, acts: []uint32{1}},
	})}
	for i := 0; i < 4; i++ {
		o.Units = append(o.Units, synth.ALMUnit{X: uint32(12+3*i)*256 + 128, Y: 14*256 + 128, ClassID: 40, ClassSubID: 2})
	}
	return built(t, o, nil, nil).Bytes()
}

// areaEditors opens the map twice: the editor under test and a copy for the
// host calls its bytes are compared with.
func areaEditors(t *testing.T, b []byte) (e, c *ui.MapEditor) {
	t.Helper()
	e, _ = openGoalMap(t, b)
	c, _ = openGoalMap(t, b)
	return e, c
}

// areaSelect selects trigger n (0 first), closes its popup and fits the map.
func areaSelect(t *testing.T, e *ui.MapEditor, n int) {
	t.Helper()
	e.Select(trigRecord(t, e, "Trigger", n), true)
	_ = e.Key("escape")
	e.Fit()
}

// areaPress presses at from, moves to to and, when release, lets go there.
func areaPress(e *ui.MapEditor, from, to image.Point, release bool) {
	now := time.Unix(100, 0)
	e.Pointer(ui.Input{CursorX: from.X, CursorY: from.Y, PrimaryDown: true}, now)
	mid := from.Add(to).Div(2)
	e.Pointer(ui.Input{CursorX: mid.X, CursorY: mid.Y, PrimaryDown: true}, now)
	e.Pointer(ui.Input{CursorX: to.X, CursorY: to.Y, PrimaryDown: true}, now)
	if release {
		e.Pointer(ui.Input{CursorX: to.X, CursorY: to.Y}, now)
	}
}

// areaOf is trigger n's area of shape covering cell, failing without one.
func areaOf(t *testing.T, e *ui.MapEditor, n, shape int, cell image.Point) ui.InspectionArea {
	t.Helper()
	for _, a := range e.TriggerAreas(n) {
		if a.Shape == shape && cell.In(a.Rect) {
			return a
		}
	}
	t.Fatalf("trigger %d has no area of shape %d at %v: %+v", n+1, shape, cell, e.TriggerAreas(n))
	return ui.InspectionArea{}
}

// The handle kinds: a drag on each rewrites the parameters its area comes
// from with the bytes the host's SetFields writes for the same values, as
// one Undo step, and Redo writes them again.
func TestAreaHandlesRewriteTheirParameters(t *testing.T) {
	for _, c := range []struct {
		name    string
		trigger int
		from    func(*ui.MapEditor, ui.InspectionArea) image.Point
		shape   int
		at, to  image.Point
		kind    string
		index   int
		values  []ui.InspectionFieldValue
	}{
		{"box corner", 0, func(e *ui.MapEditor, a ui.InspectionArea) image.Point { return e.AreaCornerPoint(a.Rect.Max) }, ui.AreaBox, image.Pt(5, 6), image.Pt(12, 11),
			"Condition", 0, []ui.InspectionFieldValue{{Key: "value3", Value: 12}, {Key: "value4", Value: 11}}},
		{"box edge", 0, func(e *ui.MapEditor, a ui.InspectionArea) image.Point {
			return e.AreaCornerPoint(image.Pt(a.Rect.Min.X, 7))
		}, ui.AreaBox, image.Pt(5, 6), image.Pt(2, 7),
			"Condition", 0, []ui.InspectionFieldValue{{Key: "value1", Value: 2}}},
		{"box edge clamped", 0, func(e *ui.MapEditor, a ui.InspectionArea) image.Point {
			return e.AreaCornerPoint(image.Pt(a.Rect.Min.X, 7))
		}, ui.AreaBox, image.Pt(5, 6), image.Pt(15, 7),
			"Condition", 0, []ui.InspectionFieldValue{{Key: "value1", Value: 9}}},
		{"radius", 1, func(e *ui.MapEditor, a ui.InspectionArea) image.Point {
			return e.AreaCornerPoint(image.Pt(a.Rect.Max.X, 20))
		}, ui.AreaSquare, image.Pt(20, 20), image.Pt(26, 20),
			"Condition", 2, []ui.InspectionFieldValue{{Key: "value3", Value: 6}}},
		{"cell", 0, func(e *ui.MapEditor, a ui.InspectionArea) image.Point { return e.CellPoint(a.Rect.Min) }, ui.AreaCell, image.Pt(33, 33), image.Pt(30, 31),
			"Action", 2, []ui.InspectionFieldValue{{Key: "value1", Value: 30}, {Key: "value2", Value: 31}}},
	} {
		t.Run(c.name, func(t *testing.T) {
			b := areaEditMap(t, true)
			e, twin := areaEditors(t, b)
			areaSelect(t, e, c.trigger)
			a := areaOf(t, e, c.trigger, c.shape, c.at)
			areaPress(e, c.from(e, a), e.CellPoint(c.to), true)
			want, err := twin.Document().Edits.(ui.InspectionFieldSet).SetFields(c.kind, c.index, c.values)
			if err != nil {
				t.Fatal(err)
			}
			got := e.Document().SourceBytes
			if bytes.Equal(got, b) || !bytes.Equal(got, want.SourceBytes) {
				t.Fatalf("the drag wrote other bytes than SetFields %+v (unchanged %v); message %q", c.values, bytes.Equal(got, b), e.Message())
			}
			if err := e.Key("undo"); err != nil || !bytes.Equal(e.Document().SourceBytes, b) {
				t.Fatalf("one Undo did not restore the map: %v", err)
			}
			if err := e.Key("redo"); err != nil || !bytes.Equal(e.Document().SourceBytes, got) {
				t.Fatalf("Redo did not write the drag again: %v", err)
			}
		})
	}
}

// A check 15 pair `square < constant 5` holds within radius 4: a rim drag to
// radius 6 writes 7 into the constant, the radius plus the comparison's
// bias, as SetFields writes it.
func TestAreaRimWritesTheConstantWithItsBias(t *testing.T) {
	conditions := []scriptNode{
		{label: "far", opcode: 15, id: 1, values: []uint32{1, 20, 20}, tags: []uint32{3, 5, 6}},
		{label: "five", opcode: 0x10002, id: 2, values: []uint32{5}, tags: []uint32{1}},
	}
	actions := []scriptNode{{label: "talk", opcode: 2, id: 1, values: []uint32{7}, tags: []uint32{1}}}
	o := synth.ALMOptions{Width: 40, Height: 40, Type7Payload: areaScript(actions, conditions, []areaTrigger{
		{name: "Less", pairs: [][3]uint32{{1, 2, 3}}, acts: []uint32{1}},
	})}
	b := built(t, o, nil, nil).Bytes()
	e, twin := areaEditors(t, b)
	areaSelect(t, e, 0)
	a := areaOf(t, e, 0, ui.AreaSquare, image.Pt(20, 20))
	if a.Radius != 4 || a.RadiusBias != 1 {
		t.Fatalf("the square reads radius %d bias %d, want 4 and 1", a.Radius, a.RadiusBias)
	}
	areaPress(e, e.AreaCornerPoint(image.Pt(a.Rect.Max.X, 20)), e.CellPoint(image.Pt(26, 20)), true)
	want, err := twin.Document().Edits.(ui.InspectionFieldSet).SetFields("Condition", 1, []ui.InspectionFieldValue{{Key: "value0", Value: 7}})
	if err != nil || !bytes.Equal(e.Document().SourceBytes, want.SourceBytes) {
		t.Fatalf("the rim drag wrote other bytes than the constant 7: %v; message %q", err, e.Message())
	}
	if err := e.Key("undo"); err != nil || !bytes.Equal(e.Document().SourceBytes, b) {
		t.Fatalf("one Undo did not restore the map: %v", err)
	}
}

// A drag cancelled with Esc, a click on a handle, showing the areas,
// selecting a trigger and the draw tool cancelled with Esc write nothing.
func TestAreaDragCancelledWritesNothing(t *testing.T) {
	b := areaEditMap(t, true)
	e, _ := areaEditors(t, b)
	areaSelect(t, e, 0)
	a := areaOf(t, e, 0, ui.AreaBox, image.Pt(5, 6))
	areaPress(e, e.AreaCornerPoint(a.Rect.Max), e.CellPoint(image.Pt(15, 15)), false)
	if r := e.TriggerAreas(0); len(r) == 0 {
		t.Fatal("the trigger lost its areas")
	}
	_ = e.Key("escape")
	p := e.CellPoint(image.Pt(15, 15))
	e.Pointer(ui.Input{CursorX: p.X, CursorY: p.Y}, time.Unix(100, 0))
	p = e.AreaCornerPoint(a.Rect.Max)
	e.Pointer(ui.Input{CursorX: p.X, CursorY: p.Y, PrimaryDown: true}, time.Unix(200, 0))
	e.Pointer(ui.Input{CursorX: p.X, CursorY: p.Y}, time.Unix(200, 0))
	if !bytes.Equal(e.Document().SourceBytes, b) || e.Dirty() {
		t.Fatalf("a cancelled drag or a click wrote the map (dirty %v)", e.Dirty())
	}
	if err := e.Key("drawarea"); err != nil || !e.AreaDrawArmed() {
		t.Fatalf("L did not arm the draw tool: %v", err)
	}
	if _ = e.Key("escape"); e.AreaDrawArmed() || e.Dirty() {
		t.Fatalf("Esc left the draw tool armed (%v) or wrote the map", e.AreaDrawArmed())
	}
}

// A change to a node another trigger names, or to a constant other pairs
// compare with, waits for the notice: Cancel writes nothing, Change all
// writes the SetFields bytes.
func TestAreaSharedNodeAsksBeforeTheChange(t *testing.T) {
	for _, c := range []struct {
		name    string
		trigger int
		shape   int
		at      image.Point
		from    func(*ui.MapEditor, ui.InspectionArea) image.Point
		to      image.Point
		names   string
		kind    string
		index   int
		values  []ui.InspectionFieldValue
	}{
		{"shared constant", 2, ui.AreaSquare, image.Pt(30, 10), func(e *ui.MapEditor, a ui.InspectionArea) image.Point {
			return e.AreaCornerPoint(image.Pt(a.Rect.Max.X, 10))
		},
			image.Pt(37, 10), "T2, T3", "Condition", 4, []ui.InspectionFieldValue{{Key: "value0", Value: 7}}},
		{"shared node", 2, ui.AreaCell, image.Pt(12, 25), func(e *ui.MapEditor, a ui.InspectionArea) image.Point { return e.CellPoint(a.Rect.Min) },
			image.Pt(10, 28), "T1, T3", "Action", 0, []ui.InspectionFieldValue{{Key: "value0", Value: 10}, {Key: "value1", Value: 28}}},
	} {
		t.Run(c.name, func(t *testing.T) {
			b := areaEditMap(t, true)
			e, twin := areaEditors(t, b)
			areaSelect(t, e, c.trigger)
			a := areaOf(t, e, c.trigger, c.shape, c.at)
			areaPress(e, c.from(e, a), e.CellPoint(c.to), true)
			notes := strings.Join(e.PromptNotes(), "")
			if !strings.Contains(notes, c.names) || !bytes.Equal(e.Document().SourceBytes, b) {
				t.Fatalf("no notice naming %s before the write: %q", c.names, notes)
			}
			r, ok := e.PromptButtonRect(1)
			trigClick(t, e, r, ok, "Cancel")
			if _, open := e.PromptRect(); open || !bytes.Equal(e.Document().SourceBytes, b) || e.Dirty() {
				t.Fatalf("Cancel left the notice open (%v) or wrote the map", open)
			}
			areaSelect(t, e, c.trigger)
			areaPress(e, c.from(e, a), e.CellPoint(c.to), true)
			r, ok = e.PromptButtonRect(0)
			trigClick(t, e, r, ok, "Change all")
			want, err := twin.Document().Edits.(ui.InspectionFieldSet).SetFields(c.kind, c.index, c.values)
			if err != nil || !bytes.Equal(e.Document().SourceBytes, want.SourceBytes) {
				t.Fatalf("Change all wrote other bytes than SetFields: %v; message %q", err, e.Message())
			}
			if err := e.Key("undo"); err != nil || !bytes.Equal(e.Document().SourceBytes, b) {
				t.Fatalf("one Undo did not restore the map: %v", err)
			}
		})
	}
	// An unshared node and an unshared constant ask nothing: the box above.
}

// boxAreaBytes are the bytes of b with a box check over box (inclusive
// corners) and, when newOne, a constant 1 after it, and trigger pair k of
// trigger n, or of a new trigger when n is -1, pointing at them: the
// records placed directly, with no editor call.
func boxAreaBytes(t *testing.T, b []byte, x0, y0, x1, y1 uint32, n, k int, one uint32, newOne bool) []byte {
	t.Helper()
	m, err := mapedit.New(b)
	if err != nil {
		t.Fatal(err)
	}
	id := uint32(6)
	rec, err := mapedit.NewScriptNode(mapload.ScriptOperationName(true, 2), 2, id, mapedit.ScriptParam{Type: 4, Value: 10001},
		mapedit.ScriptParam{Type: 5, Value: x0}, mapedit.ScriptParam{Type: 6, Value: y0}, mapedit.ScriptParam{Type: 5, Value: x1}, mapedit.ScriptParam{Type: 6, Value: y1})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := m.Place(mapedit.KindCondition, rec); err != nil {
		t.Fatal(err)
	}
	if newOne {
		one = id + 1
		rec, err := mapedit.NewScriptNode(mapload.ScriptOperationName(true, mapload.ScriptConstant), mapload.ScriptConstant, one, mapedit.ScriptParam{Type: 1, Value: 1})
		if err != nil {
			t.Fatal(err)
		}
		if _, err := m.Place(mapedit.KindCondition, rec); err != nil {
			t.Fatal(err)
		}
	}
	if n < 0 {
		rec := make([]byte, 184)
		copy(rec, "Trigger")
		binary.LittleEndian.PutUint32(rec[0x80:], id)
		binary.LittleEndian.PutUint32(rec[0x84:], one)
		binary.LittleEndian.PutUint32(rec[0xb4:], 1)
		if _, err := m.Place(mapedit.KindTrigger, rec); err != nil {
			t.Fatal(err)
		}
		return m.Bytes()
	}
	err = m.SetFields(mapedit.KindTrigger, n, mapedit.FieldSet{Key: "left" + string(rune('0'+k)), Value: int64(id)}, mapedit.FieldSet{Key: "right" + string(rune('0'+k)), Value: int64(one)})
	if err != nil {
		t.Fatal(err)
	}
	return m.Bytes()
}

// areaDraw arms the draw tool and drags a box from cell a to cell b.
func areaDraw(t *testing.T, e *ui.MapEditor, a, b image.Point) {
	t.Helper()
	if n := e.Selected(); n >= 0 {
		e.Select(n, false)
		_ = e.Key("escape") // the selected record's popup closes
	}
	e.Fit()
	if err := e.Key("drawarea"); err != nil || !e.AreaDrawArmed() {
		t.Fatalf("L did not arm the draw tool: %v; message %q", err, e.Message())
	}
	areaPress(e, e.CellPoint(a), e.CellPoint(b), true)
	if e.AreaDrawArmed() {
		t.Fatal("the draw tool stayed armed after the box")
	}
	if _, ok := e.AreaTriggerRect(); !ok {
		t.Fatalf("no trigger choice after drawing; message %q", e.Message())
	}
}

// Drawing a box asks which trigger takes it, a new trigger first: Create
// adds the box check and a new trigger with that pair only, or one more
// pair of a chosen trigger; a full trigger is refused and Cancel writes
// nothing. Each Create is one Undo step.
func TestAreaDrawnBoxGoesToTheChosenTrigger(t *testing.T) {
	b := areaEditMap(t, true)
	e, _ := areaEditors(t, b)
	areaDraw(t, e, image.Pt(6, 33), image.Pt(2, 30))
	if notes := strings.Join(e.PromptNotes(), ""); !strings.Contains(notes, "2,30 to 6,33") {
		t.Fatalf("the choice does not name the box: %q", notes)
	}
	r, ok := e.PromptButtonRect(0)
	trigClick(t, e, r, ok, "Create")
	if want := boxAreaBytes(t, b, 2, 30, 6, 33, -1, 0, 2, false); !bytes.Equal(e.Document().SourceBytes, want) {
		t.Fatalf("a new trigger: other bytes than the box check and its trigger placed directly; message %q", e.Message())
	}
	sc := e.Document().Script
	if v := e.Document().Records[e.Selected()]; v.Kind != "Trigger" || v.Index != len(sc.Triggers)-1 {
		t.Fatalf("the new trigger is not selected: %+v", v)
	}
	if got := e.TriggerAreas(len(sc.Triggers) - 1); len(got) != 1 || got[0].Rect != image.Rect(2, 30, 7, 34) {
		t.Fatalf("the new trigger's areas %+v", got)
	}
	if err := e.Key("undo"); err != nil || !bytes.Equal(e.Document().SourceBytes, b) {
		t.Fatalf("one Undo did not remove the new area: %v", err)
	}

	// Attach to T2 through the drop-down: its third pair.
	areaDraw(t, e, image.Pt(2, 30), image.Pt(3, 31))
	r, ok = e.AreaTriggerRect()
	trigClick(t, e, r, ok, "the trigger choice")
	choices := e.DropdownChoices()
	if len(choices) != 4 || choices[0] != "New trigger" || choices[2] != "T2 Near" {
		t.Fatalf("the choices %q", choices)
	}
	r, ok = e.DropdownRowRect(2)
	trigClick(t, e, r, ok, "T2 Near")
	r, ok = e.PromptButtonRect(0)
	trigClick(t, e, r, ok, "Create")
	after := e.Document().SourceBytes
	if want := boxAreaBytes(t, b, 2, 30, 3, 31, 1, 2, 2, false); !bytes.Equal(after, want) {
		t.Fatalf("T2: other bytes than its third pair set directly; message %q", e.Message())
	}

	// T2 is now full: refused in the prompt, and Cancel writes nothing.
	areaDraw(t, e, image.Pt(2, 35), image.Pt(3, 36))
	_ = e.Key("down")
	_ = e.Key("down")
	_ = e.Key("enter")
	if notes := strings.Join(e.PromptNotes(), ""); !strings.Contains(notes, "Trigger 2 already has three condition pairs") {
		t.Fatalf("a full trigger was not refused: %q", notes)
	}
	_ = e.Key("escape")
	if _, open := e.PromptRect(); open || !bytes.Equal(e.Document().SourceBytes, after) {
		t.Fatalf("Cancel left the choice open (%v) or wrote the map", open)
	}
}

// Without an unshared constant 1 the new pair compares the box with a new
// constant 1 placed after it.
func TestAreaDrawnBoxAddsItsConstantWhenNoneIsFree(t *testing.T) {
	b := areaEditMap(t, false)
	e, _ := areaEditors(t, b)
	areaDraw(t, e, image.Pt(2, 30), image.Pt(6, 33))
	_ = e.Key("enter")
	if want := boxAreaBytes(t, b, 2, 30, 6, 33, -1, 0, 0, true); !bytes.Equal(e.Document().SourceBytes, want) {
		t.Fatalf("other bytes than the box, a new constant 1 and the trigger placed directly; message %q", e.Message())
	}
	m, err := alm.Open(e.Document().SourceBytes)
	if err != nil {
		t.Fatal(err)
	}
	sc, err := m.Script()
	if err != nil {
		t.Fatal(err)
	}
	box, one, tr := sc.Conditions[5], sc.Conditions[6], sc.Triggers[len(sc.Triggers)-1]
	if box.Opcode != 2 || box.Value != [10]uint32{10001, 2, 30, 6, 33} || one.Opcode != mapload.ScriptConstant || one.Value[0] != 1 ||
		tr.Left != [3]uint32{box.ID} || tr.Right != [3]uint32{one.ID} || tr.Cmp != [3]uint32{} || tr.Acts != [4]uint32{} || tr.Once != 1 {
		t.Fatalf("decoded: box %+v constant %+v trigger %+v", box, one, tr)
	}
}
