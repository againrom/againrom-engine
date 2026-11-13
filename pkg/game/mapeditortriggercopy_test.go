package game

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"image"
	"os"
	"reflect"
	"slices"
	"strings"
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/formats/alm"
	"againrom/pkg/mapedit"
	"againrom/pkg/mapload"
	"againrom/pkg/mod"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
	"againrom/pkg/words"
)

// copyMap is a 32x32 map of one unit whose script sets variable 50 to 7 at
// once ("Set": constant 1 against constant 1) and holds a later trigger that
// never fires ("Never": constant 0 against constant 1, running Win).
func copyMap(t *testing.T) []byte {
	t.Helper()
	actions := []scriptNode{{label: "set", opcode: 3, id: 1, values: []uint32{50, 7}, tags: []uint32{1, 1}}, {label: "win", opcode: 4, id: 2}}
	conditions := []scriptNode{
		{label: "one", opcode: mapload.ScriptConstant, id: 1, values: []uint32{1}, tags: []uint32{1}},
		{label: "one too", opcode: mapload.ScriptConstant, id: 2, values: []uint32{1}, tags: []uint32{1}},
		{label: "zero", opcode: mapload.ScriptConstant, id: 3, values: []uint32{0}, tags: []uint32{1}},
	}
	return copyWordsMap(t, actions, conditions, []goalTrigger{
		{name: "Set", left: 1, right: 2, actions: []uint32{1}},
		{name: "Never", left: 3, right: 2, actions: []uint32{2}},
	})
}

// copyPress opens trigger index's window and presses its Copy, the sixth of
// its buttons, which stand apart inside the window.
func copyPress(t *testing.T, e *ui.MapEditor, index int) {
	t.Helper()
	e.Select(trigRecord(t, e, "Trigger", index), true)
	var shown []image.Rectangle
	for n := 0; ; n++ {
		r, ok := e.FindControl(ui.ControlQuery{Role: ui.ControlPopupButton, N: n})
		if !ok {
			break
		}
		if !r.In(image.Rect(0, 0, 1280, 800)) || slices.ContainsFunc(shown, r.Overlaps) {
			t.Fatalf("button %d at %v after %v", n, r, shown)
		}
		shown = append(shown, r)
	}
	if len(shown) < 6 {
		t.Fatalf("the trigger window shows %d buttons", len(shown))
	}
	trigClick(t, e, shown[5], true, "Copy")
}

// copyRecords is every record of kind k in map b, byte for byte.
func copyRecords(t *testing.T, b []byte, k mapedit.Kind) [][]byte {
	t.Helper()
	m, err := mapedit.New(b)
	if err != nil {
		t.Fatal(err)
	}
	var out [][]byte
	for i := range m.Count(k) {
		r, err := m.Record(k, i)
		if err != nil {
			t.Fatal(err)
		}
		out = append(out, r)
	}
	return out
}

// copyLatches is the step of the first script pass at which each of the
// first n trigger positions of map b latched in a headless mission, -1 for
// one that did not within 512 steps.
func copyLatches(t *testing.T, b []byte, n int) []int {
	t.Helper()
	m, err := alm.Open(b)
	if err != nil {
		t.Fatal(err)
	}
	program, rep, err := mapload.CompileScript(m, mapload.ScriptRefs{Structures: mapload.ScriptStructures(m)})
	if err != nil || len(rep.OmittedChecks)+len(rep.OmittedActions) != 0 {
		t.Fatalf("the script compiles with %+v: %v", rep, err)
	}
	w, err := sim.NewControlledScriptWorld(mapload.FromALM(m), program)
	if err != nil {
		t.Fatal(err)
	}
	at := make([]int, n)
	for i := range at {
		at[i] = -1
	}
	for step := 1; step <= 512; step++ {
		sim.Step(w, nil)
		for i := range at {
			if at[i] < 0 && w.ScriptLatched(int32(i)) {
				at[i] = step
			}
		}
	}
	return at
}

// sameCopied checks that each slot of copied names a new node equal to the
// node of the same slot of orig but for its ID, the same node for one ID, and
// no node at an empty slot.
func sameCopied(t *testing.T, before, after []alm.ScriptNode, orig, copied []uint32, to map[uint32]uint32) {
	t.Helper()
	if len(orig) != len(copied) {
		t.Fatalf("%d slots copied as %d", len(orig), len(copied))
	}
	for i, id := range orig {
		c := copied[i]
		if id == 0 || c == 0 {
			if id != c {
				t.Fatalf("slot %d: %d copied as %d", i, id, c)
			}
			continue
		}
		if prev, ok := to[id]; ok && prev != c || slices.ContainsFunc(before, func(n alm.ScriptNode) bool { return n.ID == c }) {
			t.Fatalf("slot %d: node %d copied as %d, a node of the original's or another copy", i, id, c)
		}
		to[id] = c
		n := wizNode(t, after, c)
		n.ID = id
		if !reflect.DeepEqual(n, wizNode(t, before, id)) {
			t.Fatalf("slot %d: node %d copied as %+v", i, id, n)
		}
	}
}

// The window's Copy places a trigger right after the original with the
// original's fields, nodes of its own and the name with the copy mark, and
// selects it; the later trigger moves one index on with its bytes; undo is
// one step back to the bytes before and redo makes the same copy; the Tools
// menu row and the command copy_trigger write the same bytes.
func TestTriggerCopyStandsAfterTheOriginalWithNodesOfItsOwn(t *testing.T) {
	e, _ := openGoalMap(t, copyMap(t))
	start := bytes.Clone(e.Document().SourceBytes)
	before := wizScript(t, start)
	copyPress(t, e, 0)
	got := bytes.Clone(e.Document().SourceBytes)
	after := wizScript(t, got)
	if len(after.Triggers) != 3 {
		t.Fatalf("%d triggers after the copy: %q", len(after.Triggers), e.Message())
	}
	if !reflect.DeepEqual(after.Triggers[0], before.Triggers[0]) || !reflect.DeepEqual(after.Triggers[2], before.Triggers[1]) {
		t.Fatal("the original or the later trigger reads otherwise")
	}
	triggers, was := copyRecords(t, got, mapedit.KindTrigger), copyRecords(t, start, mapedit.KindTrigger)
	if !bytes.Equal(triggers[0], was[0]) || !bytes.Equal(triggers[2], was[1]) {
		t.Fatal("the original's or the later trigger's bytes changed")
	}
	for _, k := range []mapedit.Kind{mapedit.KindCondition, mapedit.KindAction} {
		now, old := copyRecords(t, got, k), copyRecords(t, start, k)
		if len(now) <= len(old) || !reflect.DeepEqual(now[:len(old)], old) {
			t.Fatalf("%v: the nodes before the copy changed or none was added", k)
		}
	}
	orig, copied := after.Triggers[0], after.Triggers[1]
	if copied.Name != "Set (copy)" || !reflect.DeepEqual(copied.Cmp, orig.Cmp) || copied.Once != orig.Once || !reflect.DeepEqual(copied.Group, orig.Group) {
		t.Fatalf("the copy reads %+v, the original %+v", copied, orig)
	}
	conds := map[uint32]uint32{}
	sameCopied(t, before.Conditions, after.Conditions, orig.Left, copied.Left, conds)
	sameCopied(t, before.Conditions, after.Conditions, orig.Right, copied.Right, conds)
	sameCopied(t, before.Actions, after.Actions, orig.Acts, copied.Acts, map[uint32]uint32{})
	if e.Selected() != trigRecord(t, e, "Trigger", 1) {
		t.Fatalf("record %d is selected, not the copy", e.Selected())
	}
	_ = e.Key("undo")
	if !bytes.Equal(e.Document().SourceBytes, start) {
		t.Fatal("one undo does not give the bytes before the copy")
	}
	_ = e.Key("redo")
	if !bytes.Equal(e.Document().SourceBytes, got) {
		t.Fatal("redo makes another copy")
	}
	_ = e.Key("undo")
	e.Select(trigRecord(t, e, "Trigger", 0), false)
	if err := e.Key("copytrigger"); err != nil || !bytes.Equal(e.Document().SourceBytes, got) {
		t.Fatalf("the Tools menu's Copy trigger wrote other bytes: %v %q", err, e.Message())
	}
	m, _ := openGoalMap(t, copyMap(t))
	_, must := editorMCP(t, m)
	if r := commandReport(t, must("copy_trigger", map[string]any{"index": 0})); r["changed"] != true {
		t.Fatalf("copy_trigger reports %v", r)
	}
	if !bytes.Equal(m.Document().SourceBytes, got) {
		t.Fatal("the command and the window's Copy wrote different bytes")
	}
}

// The copy fires in a headless mission on the same pass as the original;
// once its own nodes are edited, the original's nodes and record keep their
// bytes and the original alone fires; the saved map reopens to the same
// bytes and the mod check finds nothing the map before the copy lacked.
func TestTriggerCopyFiresAsTheOriginalAndEditsAlone(t *testing.T) {
	e, path := openGoalMap(t, copyMap(t))
	start := bytes.Clone(e.Document().SourceBytes)
	_, must := editorMCP(t, e)
	must("copy_trigger", map[string]any{"index": 0})
	copied := bytes.Clone(e.Document().SourceBytes)
	if at := copyLatches(t, copied, 3); at[0] < 0 || at[1] != at[0] || at[2] >= 0 {
		t.Fatalf("latched at steps %v, want the original and the copy on one pass and Never not", at)
	}
	sc := wizScript(t, copied)
	left := slices.IndexFunc(sc.Conditions, func(n alm.ScriptNode) bool { return n.ID == sc.Triggers[1].Left[0] })
	act := slices.IndexFunc(sc.Actions, func(n alm.ScriptNode) bool { return n.ID == sc.Triggers[1].Acts[0] })
	if left < 3 || act < 2 {
		t.Fatalf("the copy's nodes are condition %d and action %d, nodes of the original's", left, act)
	}
	must("set_field", map[string]any{"kind": "Condition", "index": left, "key": "value0", "value": 0})
	must("set_field", map[string]any{"kind": "Action", "index": act, "key": "value1", "value": 9})
	edited := bytes.Clone(e.Document().SourceBytes)
	for _, k := range []mapedit.Kind{mapedit.KindCondition, mapedit.KindAction, mapedit.KindTrigger} {
		now, was := copyRecords(t, edited, k), copyRecords(t, start, k)
		if k == mapedit.KindTrigger {
			now, was = now[:1], was[:1]
		}
		if !reflect.DeepEqual(now[:len(was)], was) {
			t.Fatalf("%v: editing the copy changed the original's bytes", k)
		}
	}
	if at := copyLatches(t, edited, 3); at[0] < 0 || at[1] >= 0 {
		t.Fatalf("latched at steps %v, want the original alone", at)
	}
	if _, err := e.Document().Edits.SaveAs(path + ".out.alm"); err != nil {
		t.Fatal(err)
	}
	saved, err := os.ReadFile(path + ".out.alm")
	if err != nil || !bytes.Equal(saved, edited) {
		t.Fatalf("the saved map differs: %v", err)
	}
	reopened, _ := openGoalMap(t, saved)
	if !bytes.Equal(reopened.Document().SourceBytes, saved) || !reflect.DeepEqual(wizScript(t, saved), wizScript(t, edited)) {
		t.Fatal("the reopened map reads otherwise")
	}
	x := &MapInspector{}
	errs := func(b []byte) []string {
		var out []string
		for _, r := range x.CheckModMap(mod.Mission{Map: "copy.alm", MapBytes: b}, "mod/x") {
			if r.Error {
				out = append(out, r.Rule)
			}
		}
		return out
	}
	if a, b := errs(start), errs(saved); !reflect.DeepEqual(a, b) {
		t.Fatalf("the mod check refuses %v after the copy, %v before", b, a)
	}
}

// On an extended map the copy keeps every pair, comparison and group of a
// six-pair trigger and stands right after it.
func TestTriggerCopyKeepsAnExtendedTriggersPairs(t *testing.T) {
	e, _ := openGoalMap(t, extendedScriptMap(t, nil, 1))
	before := wizScript(t, e.Document().SourceBytes)
	_, must := editorMCP(t, e)
	must("copy_trigger", map[string]any{"index": 1})
	after := wizScript(t, e.Document().SourceBytes)
	if len(after.Triggers) != len(before.Triggers)+1 {
		t.Fatalf("%d triggers after the copy, %d before", len(after.Triggers), len(before.Triggers))
	}
	orig, copied := before.Triggers[1], after.Triggers[2]
	if !reflect.DeepEqual(after.Triggers[1], orig) || !reflect.DeepEqual(after.Triggers[3], before.Triggers[2]) {
		t.Fatal("the original or the later trigger reads otherwise")
	}
	if copied.Name != orig.Name+" (copy)" || !reflect.DeepEqual(copied.Cmp, orig.Cmp) || !reflect.DeepEqual(copied.Group, orig.Group) || copied.Once != orig.Once {
		t.Fatalf("the copy reads %+v, the original %+v", copied, orig)
	}
	conds := map[uint32]uint32{}
	sameCopied(t, before.Conditions, after.Conditions, orig.Left, copied.Left, conds)
	sameCopied(t, before.Conditions, after.Conditions, orig.Right, copied.Right, conds)
	sameCopied(t, before.Actions, after.Actions, orig.Acts, copied.Acts, map[uint32]uint32{})
}

// copyWordsMap is a 32x32 map of one unit whose script holds actions,
// conditions and triggers as given.
func copyWordsMap(t *testing.T, actions, conditions []scriptNode, triggers []goalTrigger) []byte {
	t.Helper()
	o := synth.ALMOptions{Width: 32, Height: 32, Type7Payload: goalScript(actions, conditions, triggers),
		Units: []synth.ALMUnit{{X: 12*256 + 128, Y: 14*256 + 128, ClassID: 40, ClassSubID: 2}}}
	return built(t, o, nil, nil).Bytes()
}

// Slots the game does not use can hold words outside the comparison alphabet
// and the ID cap: an empty pair coded 0xffffffff, 0x2020001 or 0x308, a pair
// of IDs 0xffffffff no node carries, a used pair coded 0xffffffff. Each such
// trigger copies with those words as they are and its nodes renamed, and one
// undo gives the bytes before the copy.
func TestTriggerCopyKeepsWordsOutsideTheAlphabet(t *testing.T) {
	const none = 0xffffffff
	actions := []scriptNode{{label: "set", opcode: 3, id: 1, values: []uint32{50, 7}, tags: []uint32{1, 1}}}
	conditions := []scriptNode{
		{label: "one", opcode: mapload.ScriptConstant, id: 1, values: []uint32{1}, tags: []uint32{1}},
		{label: "one too", opcode: mapload.ScriptConstant, id: 2, values: []uint32{1}, tags: []uint32{1}},
	}
	triggers := []goalTrigger{
		{name: "Empty pair coded ffffffff", raw: map[int]uint32{0xac: none}},
		{name: "Empty pair coded 2020001", raw: map[int]uint32{0xb0: 0x2020001}},
		{name: "Empty pair coded 308", raw: map[int]uint32{0xac: 0x308}},
		{name: "Pair of ffffffff", raw: map[int]uint32{0x90: none, 0x94: none}},
		{name: "Used pair coded ffffffff", cmp: none},
	}
	for i := range triggers {
		triggers[i].left, triggers[i].right, triggers[i].actions = 1, 2, []uint32{1}
	}
	b := copyWordsMap(t, actions, conditions, triggers)
	for i, tr := range triggers {
		e, _ := openGoalMap(t, b)
		start := bytes.Clone(e.Document().SourceBytes)
		e.Select(trigRecord(t, e, "Trigger", i), true)
		if err := e.Key("copytrigger"); err != nil {
			t.Fatal(err)
		}
		was, now := copyRecords(t, start, mapedit.KindTrigger), copyRecords(t, e.Document().SourceBytes, mapedit.KindTrigger)
		if len(now) != len(was)+1 {
			t.Fatalf("%s: no copy: %q", tr.name, e.Message())
		}
		want := bytes.Clone(was[i])
		copy(want, tr.name+" (copy)")
		// The first pair's conditions 1 and 2 and action 1 get the next free IDs.
		for at, id := range map[int]uint32{0x80: 3, 0x84: 4, 0x98: 2} {
			binary.LittleEndian.PutUint32(want[at:], id)
		}
		if !bytes.Equal(now[i+1], want) || !bytes.Equal(now[i], was[i]) {
			t.Fatalf("%s: the copy is % x, want % x", tr.name, now[i+1], want)
		}
		if _ = e.Key("undo"); !bytes.Equal(e.Document().SourceBytes, start) {
			t.Fatalf("%s: one undo does not give the bytes before the copy", tr.name)
		}
	}
}

// A trigger whose node no new node can carry is refused in the word book's
// words of the game's language, EN and RU, and nothing is written: a node at
// the ID cap, a label that fills its field, an ID two nodes carry.
func TestTriggerCopyRefusesInTheWordBook(t *testing.T) {
	ru, err := words.Load(words.Engine(), "ru")
	if err != nil {
		t.Fatal(err)
	}
	win := []scriptNode{{label: "win", opcode: 4, id: 1}}
	consts := func() []scriptNode {
		return []scriptNode{
			{label: "one", opcode: mapload.ScriptConstant, id: 1, values: []uint32{1}, tags: []uint32{1}},
			{label: "two", opcode: mapload.ScriptConstant, id: 2, values: []uint32{2}, tags: []uint32{1}},
		}
	}
	pair := []goalTrigger{{name: "T", left: 1, right: 2, actions: []uint32{1}}}
	capped, long, shared := consts(), consts(), consts()
	capped[1].id = 5000
	long[0].label = strings.Repeat("L", alm.ScriptLabelLen)
	shared = append(shared, scriptNode{label: "two again", opcode: mapload.ScriptConstant, id: 2, values: []uint32{9}, tags: []uint32{1}})
	cases := []struct {
		key  string
		id   uint32
		acts []scriptNode
		cond []scriptNode
		trig []goalTrigger
	}{
		{"editor.trigger.copyidsfull", 5000, win, capped, []goalTrigger{{name: "T", left: 1, right: 5000, actions: []uint32{1}}}},
		{"editor.trigger.copylongcondition", 1, win, long, pair},
		{"editor.trigger.copysharedcondition", 2, win, shared, pair},
		{"editor.trigger.copysharedaction", 1, append(win, scriptNode{label: "win again", opcode: 4, id: 1}), consts(), pair},
	}
	for _, c := range cases {
		b := copyWordsMap(t, c.acts, c.cond, c.trig)
		e, _ := openGoalMap(t, b)
		e.Select(trigRecord(t, e, "Trigger", 0), true)
		if err := e.Key("copytrigger"); err != nil {
			t.Fatal(err)
		}
		if want := fmt.Sprintf(words.For("").Text(c.key), c.id); e.Message() != want || !bytes.Equal(e.Document().SourceBytes, b) {
			t.Fatalf("%s: the copy says %q, want %q", c.key, e.Message(), want)
		}
		m, err := mapedit.New(b)
		if err != nil {
			t.Fatal(err)
		}
		_, err = m.CopyTrigger(0, "T (copy)")
		got, want := copyRefusal(err, ru.Text), fmt.Sprintf(ru.Text(c.key), c.id)
		if got == nil || got.Error() != want || want == fmt.Sprintf(words.For("").Text(c.key), c.id) {
			t.Fatalf("%s: in RU the copy says %v, want %q", c.key, got, want)
		}
	}
}

// The name with the copy mark fits the record's name field in either
// language's words, cutting the original's name from its end; a name that
// already carries the mark keeps one mark.
func TestTriggerCopyNameFitsTheNameField(t *testing.T) {
	for _, format := range []string{"%s (copy)", "%s (копия)"} {
		if got := copyName("Set", format); got != strings.Replace(format, "%s", "Set", 1) {
			t.Fatalf("%q", got)
		}
		if marked := strings.Replace(format, "%s", "Set", 1); copyName(marked, format) != marked {
			t.Fatalf("%q copies as %q", marked, copyName(marked, format))
		}
		got := copyName(strings.Repeat("Я", 63), format)
		b, err := alm.ScriptLabelText.Encode(got)
		if err != nil || len(b) >= alm.ScriptLabelLen || !strings.HasSuffix(got, strings.TrimPrefix(format, "%s")) || !strings.HasPrefix(got, "ЯЯЯ") {
			t.Fatalf("%q encodes to %d bytes: %v", got, len(b), err)
		}
	}
}
