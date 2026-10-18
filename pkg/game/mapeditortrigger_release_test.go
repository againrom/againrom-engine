package game

import (
	"bytes"
	"fmt"
	"image"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"

	"againrom/pkg/formats/alm"
	"againrom/pkg/mapload"
	"againrom/pkg/ui"
)

// installedMapBytes are the bytes of every installed map, by catalogue key.
func installedMapBytes(t *testing.T, x *MapInspector) (keys []string, maps [][]byte) {
	t.Helper()
	keys = installedMapKeys(x)
	for _, key := range keys {
		var b []byte
		var err error
		if name, ok := strings.CutPrefix(key, "loose/"); ok {
			b, err = x.archives.Loose.ReadFile(name)
		} else {
			b, err = x.archives.Containers.ReadFile(key)
		}
		if err != nil {
			t.Fatal(key, err)
		}
		maps = append(maps, b)
	}
	return keys, maps
}

// editableSlot is the popup field and the slot of a node's first number,
// constant or cell parameter, false when it has none; a group command's
// command is a drop-down.
func editableSlot(condition bool, n ui.InspectionScriptNode) (field, slot int, ok bool) {
	field = 2 // its label and operation come first
	for s, t := range n.Types {
		if t == 0 {
			continue
		}
		if !condition && n.Opcode == 6 && s == 0 {
			field++
			continue
		}
		if cell := t == 5 && s+1 < len(n.Types) && n.Types[s+1] == 6; cell || t == 1 || t == 7 {
			return field, s, true
		}
		field++
	}
	return 0, 0, false
}

// trigPairField is the trigger popup's field of side (left, comparison,
// right) of pair p: the name and once switch come first.
func trigPairField(p, side int) int { return 2 + 3*p + side }

// scriptNodeOf is the one node of a family with id.
func scriptNodeOf(sc *ui.InspectionScriptModel, condition bool, id uint32) (ui.InspectionScriptNode, bool) {
	nodes := sc.Actions
	if condition {
		nodes = sc.Conditions
	}
	var found ui.InspectionScriptNode
	n := 0
	for _, node := range nodes {
		if node.ID == id {
			found, n = node, n+1
		}
	}
	return found, n == 1 && id != 0
}

// bytesTrigger is a trigger the bytes proof can drive: its first pair's left
// condition and one of its actions have a typed parameter, and a pair slot is
// free.
func bytesTrigger(sc *ui.InspectionScriptModel) (int, bool) {
	for i, tr := range sc.Triggers {
		left, ok := scriptNodeOf(sc, true, tr.Left[0])
		if !ok || tr.Right[0] == 0 || tr.Left[2] != 0 {
			continue
		}
		if _, _, ok := editableSlot(true, left); !ok {
			continue
		}
		for _, a := range tr.Acts {
			if act, ok := scriptNodeOf(sc, false, a); ok && act.Opcode < 0x10000 {
				if _, _, ok := editableSlot(false, act); ok {
					return i, true
				}
			}
		}
	}
	return 0, false
}

// TestReleaseMapEditorTriggerPopupWritesTheFieldCallBytes drives the trigger
// and node popups on installed maps: each Apply writes the bytes the host's
// own field and record calls write for the same change, one Undo restores the
// bytes before it and Redo the bytes after it.
func TestReleaseMapEditorTriggerPopupWritesTheFieldCallBytes(t *testing.T) {
	x := installedInspector(t)
	keys := installedMapKeys(x)
	open := func(key string) *ui.MapEditor {
		e := x.Editor()
		e.Layout(1280, 800)
		if err := e.Open(key); err != nil {
			t.Fatal(err)
		}
		return e
	}
	driven := 0
	for _, key := range keys {
		if driven == 3 {
			break
		}
		e := open(key)
		if e.Document().Script == nil {
			continue
		}
		ti, ok := bytesTrigger(e.Document().Script)
		if !ok {
			continue
		}
		driven++
		t.Run(key, func(t *testing.T) { driveTriggerBytes(t, e, open(key), ti) })
	}
	if driven < 3 {
		t.Fatalf("only %d installed maps have a trigger the proof can drive", driven)
	}
}

func driveTriggerBytes(t *testing.T, e, c *ui.MapEditor, ti int) {
	set := c.Document().Edits.(ui.InspectionFieldSet)
	author := c.Document().Edits.(*mapInspectionEdits)
	trigger := func() ui.InspectionTrigger { return e.Document().Script.Triggers[ti] }
	trigIndex := e.Document().Records[trigger().Record].Index
	selectTrigger := func() { e.Select(e.Document().Script.Triggers[ti].Record, true) }
	check := func(what string, before []byte, want *ui.InspectionDocument) {
		t.Helper()
		got := e.Document().SourceBytes
		if bytes.Equal(got, before) || !bytes.Equal(got, want.SourceBytes) {
			t.Fatalf("%s: the popup wrote other bytes than the field calls (unchanged %v)\n%s", what, bytes.Equal(got, before), trigLines(e))
		}
		if err := e.Key("undo"); err != nil || !bytes.Equal(e.Document().SourceBytes, before) {
			t.Fatalf("%s: one Undo did not restore the bytes: %v", what, err)
		}
		if err := e.Key("redo"); err != nil || !bytes.Equal(e.Document().SourceBytes, got) {
			t.Fatalf("%s: Redo did not restore the edit: %v", what, err)
		}
	}
	apply := func(what string, want func() *ui.InspectionDocument) {
		t.Helper()
		before := e.Document().SourceBytes
		r, ok := e.PopupButtonRect(0)
		trigClick(t, e, r, ok, "Apply")
		check(what, before, want())
	}
	fields := func(kind string, index int, values ...ui.InspectionFieldValue) *ui.InspectionDocument {
		t.Helper()
		d, err := set.SetFields(kind, index, values)
		if err != nil || d == nil {
			t.Fatal("SetFields", err)
		}
		return d
	}
	value := func(key string, v int64) ui.InspectionFieldValue { return ui.InspectionFieldValue{Key: key, Value: v} }
	chooseRow := func(field, row int) string {
		t.Helper()
		r, ok := e.FieldRect(field)
		trigClick(t, e, r, ok, fmt.Sprintf("field %d", field))
		choices := e.DropdownChoices()
		if row < 0 {
			row += len(choices)
		}
		if row < 0 || row >= len(choices) {
			t.Fatalf("field %d offers %d choices, not row %d", field, len(choices), row)
		}
		for k := 0; k < 200; k++ {
			if r, ok := e.DropdownRowRect(row); ok {
				trigClick(t, e, r, ok, "a choice")
				return choices[row]
			}
			e.ScrollDropdown(1)
		}
		t.Fatalf("row %d of field %d never shows", row, field)
		return ""
	}
	nodeID := func(text string) int64 {
		t.Helper()
		f := strings.Fields(strings.TrimPrefix(text, "#"))
		id, err := strconv.ParseInt(f[0], 10, 64)
		if len(f) == 0 || err != nil {
			t.Fatalf("choice %q names no node", text)
		}
		return id
	}
	typeInto := func(field int, text string) {
		t.Helper()
		r, ok := e.FieldRect(field)
		trigClick(t, e, r, ok, fmt.Sprintf("field %d", field))
		for k := 0; k < 16; k++ {
			_ = e.Key("backspace")
		}
		e.TextInput(text)
	}
	// editNode opens the node behind a trigger row, types a new value into
	// its first typed parameter and checks the bytes against SetFields.
	editNode := func(what string, row int, condition bool, id uint32) {
		t.Helper()
		n, _ := scriptNodeOf(e.Document().Script, condition, id)
		field, s, _ := editableSlot(condition, n)
		r, ok := e.RowButtonRect(row, 0, false)
		trigClick(t, e, r, ok, "Edit "+what)
		kind := "Action"
		if condition {
			kind = "Condition"
		}
		index := e.Document().Records[n.Record].Index
		var want []ui.InspectionFieldValue
		if n.Types[s] == 5 {
			cx, cy := int64(n.Values[s]), int64(n.Values[s+1])
			if cx > 0 {
				cx--
			} else {
				cx++
			}
			typeInto(field, fmt.Sprintf("%d,%d", cx, cy))
			want = []ui.InspectionFieldValue{value(fmt.Sprintf("value%d", s), cx), value(fmt.Sprintf("value%d", s+1), cy)}
		} else {
			v := int64(int32(n.Values[s])) + 1
			typeInto(field, strconv.FormatInt(v, 10))
			want = []ui.InspectionFieldValue{value(fmt.Sprintf("value%d", s), v)}
		}
		apply(what, func() *ui.InspectionDocument { return fields(kind, index, want...) })
		r, ok = e.PopupButtonRect(2)
		trigClick(t, e, r, ok, "Back")
		if v := e.Document().Records[e.Selected()]; v.Kind != "Trigger" {
			t.Fatalf("Back after %s selected %+v", what, v)
		}
	}

	selectTrigger()
	// A condition parameter, then an action parameter.
	editNode("the first pair's left condition", 2, true, trigger().Left[0])
	for a, id := range trigger().Acts {
		if n, ok := scriptNodeOf(e.Document().Script, false, id); ok && n.Opcode < 0x10000 {
			if _, _, ok := editableSlot(false, n); ok {
				editNode(fmt.Sprintf("action %d", a+1), 11+a, false, id)
				break
			}
		}
	}

	// A comparison.
	cmp := 5
	if trigger().Cmp[0] == 5 {
		cmp = 0
	}
	chooseRow(3, cmp)
	apply("comparison", func() *ui.InspectionDocument { return fields("Trigger", trigIndex, value("cmp0", int64(cmp))) })

	// Fire once.
	once := int64(1 - trigger().Once)
	if trigger().Once > 1 {
		once = 1
	}
	r, ok := e.FieldRect(1)
	trigClick(t, e, r, ok, "Fire once")
	apply("fire once", func() *ui.InspectionDocument { return fields("Trigger", trigIndex, value("once", once)) })

	// A pair added from existing conditions.
	p := 1
	if trigger().Left[1] != 0 {
		p = 2
	}
	r, ok = e.RowButtonRect(-1, 0, false)
	trigClick(t, e, r, ok, "Add condition pair")
	left := nodeID(chooseRow(trigPairField(p, 0), 0))
	right := nodeID(chooseRow(trigPairField(p, 2), 0))
	apply("add pair", func() *ui.InspectionDocument {
		return fields("Trigger", trigIndex, value(fmt.Sprintf("left%d", p), left), value(fmt.Sprintf("right%d", p), right))
	})

	// The first pair removed: the later pairs move up and the last used one
	// empties.
	tr := trigger()
	r, ok = e.RowButtonRect(trigPairField(0, 1), 0, false)
	trigClick(t, e, r, ok, "Remove pair 1")
	var want []ui.InspectionFieldValue
	for q := 0; q < p; q++ {
		want = append(want, value(fmt.Sprintf("left%d", q), int64(tr.Left[q+1])), value(fmt.Sprintf("cmp%d", q), int64(tr.Cmp[q+1])), value(fmt.Sprintf("right%d", q), int64(tr.Right[q+1])))
	}
	want = append(want, value(fmt.Sprintf("left%d", p), 0), value(fmt.Sprintf("right%d", p), 0))
	apply("remove pair", func() *ui.InspectionDocument { return fields("Trigger", trigIndex, want...) })

	// An action removed, then a new action added in the first free slot.
	a := 0
	for trigger().Acts[a] == 0 {
		a++
	}
	r, ok = e.RowButtonRect(11+a, 1, false)
	trigClick(t, e, r, ok, "Remove an action")
	apply("remove action", func() *ui.InspectionDocument {
		return fields("Trigger", trigIndex, value(fmt.Sprintf("action%d", a), 0))
	})
	free := 0
	for trigger().Acts[free] != 0 {
		free++
	}
	r, ok = e.RowButtonRect(-1, 0, true)
	trigClick(t, e, r, ok, "Add action")
	chooseRow(11+free, -1)
	apply("add action", func() *ui.InspectionDocument {
		d, err := author.AddScript("Action")
		if err != nil {
			t.Fatal(err)
		}
		id := d.Script.Actions[len(d.Script.Actions)-1].ID
		return fields("Trigger", trigIndex, value(fmt.Sprintf("action%d", free), int64(id)))
	})

	// A new trigger, then its deletion.
	before := e.Document().SourceBytes
	n := len(e.Document().Script.Triggers)
	r, ok = e.PopupButtonRect(2)
	trigClick(t, e, r, ok, "New")
	addSteps(t, e, nil)
	d, err := author.AddScript("Trigger")
	if err != nil {
		t.Fatal(err)
	}
	check("new trigger", before, d)
	if len(e.Document().Script.Triggers) != n+1 {
		t.Fatalf("New left %d triggers, want %d", len(e.Document().Script.Triggers), n+1)
	}
	e.Select(e.Document().Script.Triggers[n].Record, true)
	before = e.Document().SourceBytes
	r, ok = e.PopupButtonRect(3)
	trigClick(t, e, r, ok, "Delete")
	d, err = c.Document().Edits.Delete("Trigger", n)
	if err != nil {
		t.Fatal(err)
	}
	check("delete trigger", before, d)
}

// TestReleaseMapEditorTriggerAreaCensus lists over every installed map how
// many triggers draw an area on the map, which places are drawn and which
// are not with the reason; AGAINROM_TRIGGER_AREAS names a file to write it
// to. Every drawn area lies inside its map, and the editor's model carries
// the decoder's areas unchanged.
func TestReleaseMapEditorTriggerAreaCensus(t *testing.T) {
	x := installedInspector(t)
	keys, maps := installedMapBytes(t, x)
	type tally struct{ maps, triggers, drawing, areas, notes int }
	var all tally
	shapes := map[string]int{}
	reasons := map[string]int{}
	var lines []string
	for i, b := range maps {
		m, err := alm.Open(b)
		if err != nil {
			t.Fatal(keys[i], err)
		}
		sc, err := m.Script()
		if err != nil || len(sc.Triggers) == 0 {
			continue
		}
		areas, notes := mapload.ScriptAreas(m, sc)
		drawing := map[int]bool{}
		for _, a := range areas {
			drawing[a.Trigger] = true
			kind := "action"
			if a.Condition {
				kind = "check"
			}
			shapes[fmt.Sprintf("%s %d: %s", kind, a.Opcode, []string{"cell", "box", "square"}[a.Shape])]++
			if a.Rect.Empty() || !a.Rect.In(image.Rect(0, 0, int(m.Width), int(m.Height))) {
				t.Errorf("%s: trigger %d draws %v outside its %dx%d map", keys[i], a.Trigger, a.Rect, m.Width, m.Height)
			}
		}
		for _, n := range notes {
			if !n.Drawn {
				kind := "action"
				if n.Condition {
					kind = "check"
				}
				reasons[fmt.Sprintf("%s %d: %s", kind, n.Opcode, n.Reason)]++
			}
		}
		all.maps++
		all.triggers += len(sc.Triggers)
		all.drawing += len(drawing)
		all.areas += len(areas)
		lines = append(lines, fmt.Sprintf("| %s | %d | %d | %d |", keys[i], len(sc.Triggers), len(drawing), len(areas)))
		if len(lines) == 1 {
			e := x.Editor()
			e.Layout(1280, 800)
			if err := e.Open(keys[i]); err != nil {
				t.Fatal(err)
			}
			if got := len(e.Document().Script.Areas); got != len(areas) {
				t.Errorf("%s: the editor carries %d areas, the decoder finds %d", keys[i], got, len(areas))
			}
		}
	}
	var out []string
	out = append(out, fmt.Sprintf("maps with triggers %d, triggers %d, triggers drawing an area %d, areas %d", all.maps, all.triggers, all.drawing, all.areas))
	var keysShape []string
	for k := range shapes {
		keysShape = append(keysShape, k)
	}
	sort.Strings(keysShape)
	out = append(out, "", "| place drawn | areas |", "|---|---|")
	for _, k := range keysShape {
		out = append(out, fmt.Sprintf("| %s | %d |", k, shapes[k]))
	}
	var keysReason []string
	for k := range reasons {
		keysReason = append(keysReason, k)
	}
	sort.Strings(keysReason)
	out = append(out, "", "| place not drawn | count |", "|---|---|")
	for _, k := range keysReason {
		out = append(out, fmt.Sprintf("| %s | %d |", k, reasons[k]))
	}
	out = append(out, "", "| map | triggers | drawing | areas |", "|---|---|---|---|")
	out = append(out, lines...)
	t.Log(out[0])
	if all.drawing == 0 {
		t.Fatal("no installed trigger draws an area")
	}
	if path := os.Getenv("AGAINROM_TRIGGER_AREAS"); path != "" {
		if err := os.WriteFile(path, []byte(strings.Join(out, "\n")+"\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

// TestReleaseMapEditorTriggerFrames writes the trigger editor's states,
// rendered headless from the install, into AGAINROM_EDITOR_FRAMES: the
// trigger list, a trigger's popup, an action with its operation's drop-down
// open, a cell pick, and the areas with one trigger highlighted. In each the
// installed fonts keep every glyph.
func TestReleaseMapEditorTriggerFrames(t *testing.T) {
	x := installedInspector(t)
	dir := os.Getenv("AGAINROM_EDITOR_FRAMES")
	keys := installedMapKeys(x)
	open := func(key string) *ui.MapEditor {
		e := x.Editor()
		e.Layout(1280, 800)
		if err := e.Open(key); err != nil {
			t.Fatal(err)
		}
		return e
	}
	// The first map with a trigger drawing an area, whose first action has a
	// cell parameter somewhere among its triggers.
	key, ti, cellAct := "", -1, -1
	for _, k := range keys {
		e := open(k)
		sc := e.Document().Script
		if sc == nil {
			continue
		}
		for i, tr := range sc.Triggers {
			if len(e.TriggerAreas(i)) == 0 {
				continue
			}
			for a, id := range tr.Acts {
				if n, ok := scriptNodeOf(sc, false, id); ok && n.Opcode < 0x10000 {
					if _, s, ok := editableSlot(false, n); ok && n.Types[s] == 5 {
						key, ti, cellAct = k, i, a
						break
					}
				}
			}
			if key != "" {
				break
			}
		}
		if key != "" {
			break
		}
	}
	if key == "" {
		t.Fatal("no installed trigger draws an area and moves a cell")
	}
	t.Logf("%s trigger %d action %d", key, ti+1, cellAct+1)
	shot := func(e *ui.MapEditor, name string, cover image.Rectangle) {
		t.Helper()
		editorShot(t, e, dir, name, cover)
	}

	e := open(key)
	r, ok := e.BarRect("scripts")
	trigClick(t, e, r, ok, "the Scripts entry")
	_ = e.Key("escape")
	shot(e, "trigger-list", image.Rectangle{})

	e = open(key)
	e.Select(e.Document().Script.Triggers[ti].Record, true)
	shot(e, "trigger-popup", image.Rectangle{})

	if err := e.Key("escape"); err != nil {
		t.Fatal(err)
	}
	if _, ok := e.AreaLabelRect(ti); !ok {
		t.Fatal("the highlighted trigger has no label on the map")
	}
	shot(e, "trigger-areas", image.Rectangle{})

	e = open(key)
	e.Select(e.Document().Script.Triggers[ti].Record, true)
	r, ok = e.RowButtonRect(11+cellAct, 0, false)
	trigClick(t, e, r, ok, "Edit the action")
	r, ok = e.FieldRect(1)
	trigClick(t, e, r, ok, "the operation")
	shot(e, "action-dropdown", image.Rect(r.Min.X, r.Max.Y, 1280, 800))
	for _, c := range e.DropdownChoices() {
		if strings.Contains(c, "?") {
			t.Errorf("the operation choice %q lost letters to the font's encoding", c)
		}
	}

	e = open(key)
	e.Select(e.Document().Script.Triggers[ti].Record, true)
	r, ok = e.RowButtonRect(11+cellAct, 0, false)
	trigClick(t, e, r, ok, "Edit the action")
	n, _ := scriptNodeOf(e.Document().Script, false, e.Document().Script.Triggers[ti].Acts[cellAct])
	field, _, _ := editableSlot(false, n)
	r, ok = e.RowButtonRect(field, 0, false)
	trigClick(t, e, r, ok, "On map")
	shot(e, "cell-pick", image.Rectangle{})
}

// editorShot draws the editor headless, checks that the installed fonts keep
// every glyph outside cover, and writes the frame into dir when it is set.
func editorShot(t *testing.T, e *ui.MapEditor, dir, name string, cover image.Rectangle) {
	t.Helper()
	if dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	writeEditorFrame(t, e, filepath.Join(dir, name+".png"), dir != "", cover)
}
