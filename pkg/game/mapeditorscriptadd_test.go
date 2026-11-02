package game

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/base"
	"againrom/pkg/render/text"
	"againrom/pkg/ui"
	"againrom/pkg/words"
)

// scriptAddMap holds four units, a check on unit 10 compared with a
// constant, a check on unit 11 no pair names, a unit-to-unit check no item
// reads and no pair names, a trigger "Full" whose three pairs are taken and
// a trigger "Half" with its first pair taken.
func scriptAddMap(t *testing.T) []byte {
	t.Helper()
	conditions := []scriptNode{
		{label: "dead", opcode: 5, id: 1, values: []uint32{10}, tags: []uint32{kUnit}},
		{label: "zero", opcode: 0x10002, id: 2, values: []uint32{0}, tags: []uint32{kInt}},
		{label: "lone", opcode: 5, id: 3, values: []uint32{11}, tags: []uint32{kUnit}},
		{label: "odd", opcode: 7, id: 4, values: []uint32{10, 11}, tags: []uint32{kUnit, kUnit}},
	}
	actions := []scriptNode{{label: "win", opcode: 4, id: 1}}
	o := synth.ALMOptions{Width: 40, Height: 40, Type7Payload: goalScript(actions, conditions, []goalTrigger{
		{name: "Full", left: 1, right: 2, actions: []uint32{1}, raw: map[int]uint32{0x88: 1, 0x8c: 2, 0x90: 1, 0x94: 2}},
		{name: "Half", left: 1, right: 2, actions: []uint32{1}},
	})}
	for i := 0; i < 4; i++ {
		o.Units = append(o.Units, synth.ALMUnit{X: uint32(12+3*i)*256 + 128, Y: 14*256 + 128, ClassID: 40, ClassSubID: 2})
	}
	return built(t, o, nil, nil).Bytes()
}

// scriptAddLang is one editor language: the profile the host reads its
// words in and the font selector that encodes them.
type scriptAddLang struct {
	name    string
	profile base.Profile
	sel     int
}

var scriptAddLangs = []scriptAddLang{
	{"en", base.Profile{Language: "english"}, 0},
	{"ru", base.Profile{Language: "russian"}, text.SelectorConverting},
}

// scriptAddEditor opens b at 1920x1080 in l's words and answers the
// editor, its host and word, which gives an editor word in the font's bytes.
func scriptAddEditor(t *testing.T, l scriptAddLang, b []byte) (*ui.MapEditor, *MapInspector, func(id string, a ...any) string) {
	t.Helper()
	x := heroLabelInspector(t, l.profile, l.sel)
	e := openHeroLabelMap(t, x, b)
	e.Layout(1920, 1080)
	closeMapPanel(t, e)
	book := words.For(l.profile.Language)
	return e, x, func(id string, a ...any) string { return x.inspectionUTF8(fmt.Sprintf(book.Text(id), a...)) }
}

// addWalk drives the open Add window to its end: each step given in values
// is typed, every other step keeps its default. It answers the keys of the
// steps asked, in order, and stops at stop when that step comes.
func addWalk(t *testing.T, e *ui.MapEditor, values map[string]string, stop string) []string {
	t.Helper()
	var keys []string
	for i := 0; i < 32; i++ {
		key, _, ok := e.AddStep()
		if !ok || key == stop {
			return keys
		}
		keys = append(keys, key)
		if v, ok := values[key]; ok {
			_ = e.Key("clear")
			e.TextInput(v)
		}
		if err := e.Key("enter"); err != nil {
			t.Fatal(err)
		}
		if next, note, ok := e.AddStep(); ok && note != "" {
			t.Fatalf("the Add window refused step %s (now %s): %s", key, next, note)
		}
	}
	t.Fatal("the Add window did not end")
	return nil
}

// openScriptAdd opens the Add window of the Scripts list.
func openScriptAdd(t *testing.T, e *ui.MapEditor) {
	t.Helper()
	if err := e.Key("scripts"); err != nil {
		t.Fatal(err)
	}
	if err := e.Key("player"); err != nil {
		t.Fatal(err)
	}
	if key, _, ok := e.AddStep(); !ok || key != "kind" {
		t.Fatalf("the Scripts list's Add opened %q %v", key, ok)
	}
}

// addCheck adds condition item into trigger (-1 a new one) through the
// Conditions Add, answering the steps it asked.
func addCheck(t *testing.T, e *ui.MapEditor, item, trigger int) []string {
	t.Helper()
	openScriptAdd(t, e)
	return addWalk(t, e, map[string]string{"kind": "1", "check": strconv.Itoa(item), "trigger": strconv.Itoa(trigger)}, "")
}

// wizardCheck adds a condition row of the item labelled label as row key
// of trigger's wizard, with the row's defaults, and applies it.
func wizardCheck(t *testing.T, e *ui.MapEditor, trigger int, key, label string) {
	t.Helper()
	e.Select(trigRecord(t, e, "Trigger", trigger), true)
	wizPress(t, e, "", 0)
	trigChoose(t, e, wizField(t, e, key+".item"), label)
	wizApply(t, e)
}

// checkKinds are condition items covering each parameter kind a condition
// takes: the item's index in the wizard's table, its label word and the
// parameter and comparison steps the Add window asks.
var checkKinds = []struct {
	kind  string
	item  int
	word  string
	steps []string
}{
	{"unit", 4, "editor.wizard.c.unitdead", []string{"p0"}},
	{"player", 6, "editor.wizard.c.playercount", []string{"p0", "p1", "cmp"}},
	{"cell", 15, "editor.wizard.c.sackgone", []string{"p0"}},
	{"variable", 21, "editor.wizard.c.variable", []string{"p0", "p1", "cmp"}},
	{"item", 17, "editor.wizard.c.hasitem", []string{"p0", "p1"}},
	{"group", 5, "editor.wizard.c.groupcount", []string{"p0", "p1", "cmp"}},
	{"number", 11, "editor.wizard.c.health", []string{"p0", "p1", "cmp"}},
	{"structure", 20, "editor.wizard.c.structure", []string{"p0", "p1", "cmp"}},
	{"area", 0, "editor.wizard.c.heroarea", []string{"p0", "p1"}},
}

// The Conditions Add offers the wizard's condition items in the editor's
// words, EN and RU, asks the item's parameters, and writes the item's form
// exactly as a wizard row of that item writes it: into an existing trigger
// the map bytes equal the wizard's, the check in the trigger's first free
// pair; into a new trigger the nodes and the pair are the wizard row's, the
// new trigger reads back with the row as that item, and no constant 0 node
// is written alone.
func TestScriptsAddConditionWritesTheWizardRowOfEachParameterKind(t *testing.T) {
	b := scriptAddMap(t)
	for _, l := range scriptAddLangs {
		for _, k := range checkKinds {
			name := l.name + " " + k.kind
			x, _, w := scriptAddEditor(t, l, b)
			wizardCheck(t, x, 1, "c1", w(k.word))
			wizard := wizScript(t, x.Document().SourceBytes)

			y, _, _ := scriptAddEditor(t, l, b)
			openScriptAdd(t, y)
			addWalk(t, y, map[string]string{"kind": "1"}, "check")
			if lines := y.AddListLines(); !slices.Contains(lines, w(k.word)) {
				t.Fatalf("%s: the item list lacks %q: %q", name, w(k.word), lines)
			}
			keys := addWalk(t, y, map[string]string{"check": strconv.Itoa(k.item), "trigger": "1"}, "")
			if want := append(append([]string{"check"}, k.steps...), "trigger"); !slices.Equal(keys, want) {
				t.Fatalf("%s: the Add window asked %q, want %q", name, keys, want)
			}
			if !bytes.Equal(y.Document().SourceBytes, x.Document().SourceBytes) {
				got := wizScript(t, y.Document().SourceBytes)
				t.Fatalf("%s: Add wrote %+v / %+v, the wizard %+v / %+v", name, got.Triggers[1], got.Conditions[4:], wizard.Triggers[1], wizard.Conditions[4:])
			}
			if tr := wizard.Triggers[1]; tr.Left[1] == 0 || tr.Left[0] != 1 || tr.Left[2] != 0 {
				t.Fatalf("%s: the check is not in the first free pair: %+v", name, tr)
			}

			z, _, _ := scriptAddEditor(t, l, b)
			addCheck(t, z, k.item, -1)
			sc := wizScript(t, z.Document().SourceBytes)
			if len(sc.Triggers) != 3 {
				t.Fatalf("%s: %d triggers after a check into a new trigger", name, len(sc.Triggers))
			}
			tr, want := sc.Triggers[2], wizard.Triggers[1]
			if !reflect.DeepEqual(sc.Conditions, wizard.Conditions) || tr.Left[0] != want.Left[1] || tr.Right[0] != want.Right[1] || tr.Cmp[0] != want.Cmp[1] {
				t.Fatalf("%s: the new trigger %+v over %+v, the wizard row %+v over %+v", name, tr, sc.Conditions[4:], want, wizard.Conditions[4:])
			}
			for _, n := range sc.Conditions[4:] {
				if n.Opcode == 0x10002 && n.Value[0] == 0 && n.ID != tr.Right[0] {
					t.Fatalf("%s: a constant 0 node no pair names was written: %+v", name, n)
				}
			}
			z.Select(trigRecord(t, z, "Trigger", 2), true)
			row := w("editor.wizard.condition") + " " + w(k.word)
			if _, kept := z.WizardRowCounts(); kept != 0 || !slices.Contains(z.PopupLines(), row) {
				t.Fatalf("%s: the new trigger does not read back as %q:\n%s", name, row, trigLines(z))
			}
		}
	}
}

// A trigger with no free pair is refused at the trigger step in the
// editor's words, EN and RU; the window stays, and leaving it changes no
// byte of the map.
func TestScriptsAddConditionRefusesAFullTrigger(t *testing.T) {
	b := scriptAddMap(t)
	for _, l := range scriptAddLangs {
		e, _, w := scriptAddEditor(t, l, b)
		openScriptAdd(t, e)
		addWalk(t, e, map[string]string{"kind": "1", "check": "4"}, "trigger")
		_ = e.Key("clear")
		e.TextInput("0")
		_ = e.Key("enter")
		key, note, ok := e.AddStep()
		if !ok || key != "trigger" || note != w("editor.area.full", 1) {
			t.Fatalf("%s: a full trigger answered %q %q %v, want %q", l.name, key, note, ok, w("editor.area.full", 1))
		}
		r, ok := e.PromptButtonRect(2)
		trigClick(t, e, r, ok, "Cancel")
		if _, _, open := e.AddStep(); open || e.Dirty() || !bytes.Equal(e.Document().SourceBytes, b) {
			t.Fatalf("%s: the refusal changed the map or left the window open", l.name)
		}
	}
}

// A condition no pair names offers Add to trigger in its popup, EN and RU.
// The check the wizard reads as an item enters as that item's row, the
// unit-to-unit check no item reads enters as Details enters a pair (== a
// new constant) and the trigger shows its Details. Either way the pair
// names the condition's ID, every node the map held and every other
// trigger stay as they were, and a full trigger is refused.
func TestScriptsListedConditionGoesIntoATrigger(t *testing.T) {
	b := scriptAddMap(t)
	before := wizScript(t, b)
	for _, l := range scriptAddLangs {
		for _, c := range []struct {
			index   int
			id      uint32
			details bool
		}{{2, 3, false}, {3, 4, true}} {
			for _, trigger := range []int{-1, 1, 0} {
				name := fmt.Sprintf("%s condition %d into trigger %d", l.name, c.id, trigger)
				e, _, w := scriptAddEditor(t, l, b)
				e.Select(trigRecord(t, e, "Condition", c.index), false)
				buttons := e.PopupButtons()
				at := slices.Index(buttons, w("editor.scripts.totrigger"))
				if at < 0 {
					t.Fatalf("%s: the popup offers %q", name, buttons)
				}
				r, ok := e.PopupButtonRect(at)
				trigClick(t, e, r, ok, "Add to trigger")
				if key, _, ok := e.AddStep(); !ok || key != "trigger" {
					t.Fatalf("%s: Add to trigger asked %q %v", name, key, ok)
				}
				_ = e.Key("clear")
				e.TextInput(strconv.Itoa(trigger))
				_ = e.Key("enter")
				if trigger == 0 {
					if _, note, _ := e.AddStep(); note != w("editor.area.full", 1) || !bytes.Equal(e.Document().SourceBytes, b) {
						t.Fatalf("%s: a full trigger answered %q", name, note)
					}
					continue
				}
				if _, note, ok := e.AddStep(); ok {
					t.Fatalf("%s: the window stayed: %s", name, note)
				}
				sc := wizScript(t, e.Document().SourceBytes)
				index, pair := trigger, 1
				if trigger < 0 {
					index, pair = 2, 0
				}
				tr := sc.Triggers[index]
				if tr.Left[pair] != c.id || tr.Right[pair] == 0 || tr.Right[pair] == c.id {
					t.Fatalf("%s: the pair is %d against %d", name, tr.Left[pair], tr.Right[pair])
				}
				if !reflect.DeepEqual(sc.Conditions[:4], before.Conditions) || !reflect.DeepEqual(sc.Actions, before.Actions) || len(sc.Conditions) != 5 {
					t.Fatalf("%s: the nodes moved: %+v", name, sc.Conditions)
				}
				for i, old := range before.Triggers {
					if i == index {
						old.Left[pair], old.Right[pair], old.Cmp[pair] = tr.Left[pair], tr.Right[pair], tr.Cmp[pair]
					}
					if !reflect.DeepEqual(sc.Triggers[i], old) {
						t.Fatalf("%s: trigger %d moved: %+v, was %+v", name, i, sc.Triggers[i], old)
					}
				}
				if trigger < 0 && (tr.Acts != [4]uint32{} || tr.Left[1] != 0 || tr.Left[2] != 0) {
					t.Fatalf("%s: the new trigger holds more than the pair: %+v", name, tr)
				}
				if c.details {
					if right := wizNode(t, sc.Conditions, tr.Right[pair]); tr.Cmp[pair] != 0 || right.Opcode != 0x10002 || e.WizardShown() {
						t.Fatalf("%s: not the Details pair: cmp %d against %+v, wizard %v", name, tr.Cmp[pair], right, e.WizardShown())
					}
					continue
				}
				e.Select(trigRecord(t, e, "Trigger", index), true)
				row := w("editor.wizard.condition") + " " + w("editor.wizard.c.unitdead")
				if _, kept := e.WizardRowCounts(); kept != 0 || !slices.Contains(e.PopupLines(), row) {
					t.Fatalf("%s: the trigger does not read the check as %q:\n%s", name, row, trigLines(e))
				}
			}
		}
	}
}

// variableTarget is the register the host offers as the new variable.
func variableTarget(t *testing.T, e *ui.MapEditor) uint32 {
	t.Helper()
	for _, c := range e.Document().Script.Targets[ui.ScriptVariableParameter] {
		if c.Record == -1 {
			return c.Value
		}
	}
	t.Fatal("the host offers no new variable")
	return 0
}

// newVariable adds a variable labelled label through the Scripts list's Add.
func newVariable(t *testing.T, e *ui.MapEditor, label string) {
	t.Helper()
	openScriptAdd(t, e)
	if keys := addWalk(t, e, map[string]string{"kind": "4", "label": label}, ""); !slices.Equal(keys, []string{"kind", "label"}) {
		t.Fatalf("New variable asked %q", keys)
	}
}

// treeHas reports a member row of the tree's Scripts branch whose text is
// text.
func treeHas(e *ui.MapEditor, text string) bool {
	showTree(e)
	lines := e.TreeLines()
	for n := 0; ; n++ {
		row := e.TreeMemberRow("scripts", n)
		if row < 0 {
			return false
		}
		if row < len(lines) && strings.TrimSpace(lines[row]) == text {
			return true
		}
	}
}

// New variable asks a label and takes the register the host's allocator
// offers; the variable is listed at once, before any node addresses it, and
// its window says no trigger uses it. The label shows in the list, in the
// wizard's and Details' variable choices and in a wizard row's text; a new
// condition after it takes another register, and so does a second
// variable. Label... edits the label, Clear label clears it, and the row
// then reads Variable N. EN and RU.
func TestScriptsNewVariableIsALabelledRegister(t *testing.T) {
	b := scriptAddMap(t)
	for _, l := range scriptAddLangs {
		e, _, w := scriptAddEditor(t, l, b)
		r0 := variableTarget(t, e)
		newVariable(t, e, "Gold")
		if got := e.Document().Script.Labels[r0]; got != "Gold" || !bytes.Equal(e.Document().SourceBytes, b) || !e.Dirty() {
			t.Fatalf("%s: register %d labelled %q; the map bytes moved %v; dirty %v", l.name, r0, got, !bytes.Equal(e.Document().SourceBytes, b), e.Dirty())
		}
		title := w("editor.scripts.variable", r0) + "  Gold"
		if lines := e.PopupLines(); e.Selected() > -2 || !slices.Contains(lines, w("editor.labels.unused")) {
			t.Fatalf("%s: the new variable's window shows %q (selected %d)", l.name, lines, e.Selected())
		}
		if !treeHas(e, title) {
			t.Fatalf("%s: the list lacks %q: %q", l.name, title, e.TreeLines())
		}

		// The wizard's variable choice names it; the row then shows it.
		e.Select(trigRecord(t, e, "Trigger", 1), true)
		wizPress(t, e, "", 0)
		trigChoose(t, e, wizField(t, e, "c1.item"), w("editor.wizard.c.variable"))
		if got := fieldChoices(t, e, "c1.p0"); !slices.ContainsFunc(got, func(c string) bool { return strings.Contains(c, "Gold") }) {
			t.Fatalf("%s: the wizard's variable choices are %q", l.name, got)
		}
		trigChoose(t, e, wizField(t, e, "c1.p0"), "Gold")
		wizApply(t, e)
		sc := wizScript(t, e.Document().SourceBytes)
		check := wizNode(t, sc.Conditions, sc.Triggers[1].Left[1])
		if check.Opcode != 19 || check.Value[0] != r0 {
			t.Fatalf("%s: the condition reads %+v, want register %d", l.name, check, r0)
		}
		for i := range sc.Conditions {
			if uint32(i) == r0 {
				t.Fatalf("%s: a new condition took the variable's register %d", l.name, r0)
			}
		}
		e.Select(trigRecord(t, e, "Trigger", 1), true)
		if !strings.Contains(trigLines(e), "Gold") {
			t.Fatalf("%s: the wizard row does not show the label:\n%s", l.name, trigLines(e))
		}

		// Details' variable choice names it.
		d := e.Document().Script
		var node ui.InspectionScriptNode
		for _, n := range d.Conditions {
			if n.ID == check.ID {
				node = n
			}
		}
		e.Select(node.Record, false)
		r, ok := e.FieldRect(parameterField(node, 0))
		trigClick(t, e, r, ok, "variable field")
		if got := e.DropdownChoices(); !slices.ContainsFunc(got, func(c string) bool { return strings.Contains(c, "Gold") }) {
			t.Fatalf("%s: Details' variable choices are %q", l.name, got)
		}
		_ = e.Key("escape")
		_ = e.Key("escape")

		// A second variable takes another register.
		r1 := variableTarget(t, e)
		newVariable(t, e, "Silver")
		if r1 == r0 || e.Document().Script.Labels[r1] != "Silver" || int(r1) < len(e.Document().Script.Conditions) {
			t.Fatalf("%s: the second variable took register %d (first %d)", l.name, r1, r0)
		}

		// The variable window edits and clears the label.
		if !treeHas(e, title) {
			t.Fatalf("%s: the list lacks %q", l.name, title)
		}
		openVariable(t, e, title)
		if buttons := e.PopupButtons(); len(buttons) < 3 || buttons[0] != w("editor.labels.edit") || buttons[1] != w("editor.labels.clear") {
			t.Fatalf("%s: the variable window's buttons are %q", l.name, buttons)
		}
		if slices.Contains(e.PopupLines(), w("editor.labels.unused")) {
			t.Fatalf("%s: an addressed variable says no trigger uses it", l.name)
		}
		r, ok = e.PopupButtonRect(0)
		trigClick(t, e, r, ok, "Label...")
		if keys := addWalk(t, e, map[string]string{"label": "Treasury"}, ""); !slices.Equal(keys, []string{"label"}) {
			t.Fatalf("%s: Label... asked %q", l.name, keys)
		}
		if e.Document().Script.Labels[r0] != "Treasury" {
			t.Fatalf("%s: the label reads %q", l.name, e.Document().Script.Labels[r0])
		}
		renamed := w("editor.scripts.variable", r0) + "  Treasury"
		openVariable(t, e, renamed)
		r, ok = e.PopupButtonRect(1)
		trigClick(t, e, r, ok, "Clear label")
		if _, labelled := e.Document().Script.Labels[r0]; labelled || !treeHas(e, w("editor.scripts.variable", r0)) {
			t.Fatalf("%s: Clear label left %q: %q", l.name, e.Document().Script.Labels[r0], e.TreeLines())
		}
	}
}

// openVariable opens the Scripts tree's row whose text is title.
func openVariable(t *testing.T, e *ui.MapEditor, title string) {
	t.Helper()
	for n := 0; ; n++ {
		r, ok := treeMemberRect(e, "scripts", n)
		if !ok {
			t.Fatalf("no Scripts row %q: %q", title, e.TreeLines())
		}
		row := e.TreeMemberRow("scripts", n)
		if lines := e.TreeLines(); row < len(lines) && strings.TrimSpace(lines[row]) == title {
			trigClick(t, e, r, true, title)
			if e.Selected() > -2 {
				t.Fatalf("row %q selected %d", title, e.Selected())
			}
			return
		}
	}
}

// The labels go to one file beside the map, written on save and read on
// open: it holds each label by register, the map bytes are those of a save
// without labels, and a map with no label writes no file. A label on a
// register no node addresses below the check count loads without error and
// is dropped on the next save; a malformed file opens the map with a note in
// the editor's words, EN and RU, and no labels, and a save leaves it as it
// was.
func TestScriptsVariableLabelsFileBesideTheMap(t *testing.T) {
	b := scriptAddMap(t)
	for _, l := range scriptAddLangs {
		dir := t.TempDir()
		e, x, _ := scriptAddEditor(t, l, b)
		r0 := variableTarget(t, e)
		newVariable(t, e, "Gold")
		target := filepath.Join(dir, "labelled.alm")
		if _, err := e.Document().Edits.SaveAs(target); err != nil {
			t.Fatal(err)
		}
		saved, err := os.ReadFile(target)
		if err != nil || !bytes.Equal(saved, b) {
			t.Fatalf("%s: the labelled save's map bytes differ from the map: %v", l.name, err)
		}
		labels, err := os.ReadFile(filepath.Join(dir, "labelled.variables"))
		if want := fmt.Sprintf("{\n  \"variables\": [\n    {\n      \"register\": %d,\n      \"label\": \"Gold\"\n    }\n  ]\n}\n", r0); err != nil || string(labels) != want {
			t.Fatalf("%s: the label file holds %q (%v), want %q", l.name, labels, err, want)
		}

		plain, _, _ := scriptAddEditor(t, l, b)
		bare := filepath.Join(dir, "plain.alm")
		if _, err := plain.Document().Edits.SaveAs(bare); err != nil {
			t.Fatal(err)
		}
		if got, _ := os.ReadFile(bare); !bytes.Equal(got, saved) {
			t.Fatalf("%s: a save without labels writes other map bytes", l.name)
		}
		if _, err := os.Stat(filepath.Join(dir, "plain.variables")); !os.IsNotExist(err) {
			t.Fatalf("%s: a map with no label wrote a label file: %v", l.name, err)
		}

		reopened := x.Editor()
		if err := reopened.Open(target); err != nil {
			t.Fatal(err)
		}
		if got := reopened.Document().Script.Labels[r0]; got != "Gold" || reopened.Message() != "" {
			t.Fatalf("%s: the reopened map labels register %d %q, note %q", l.name, r0, got, reopened.Message())
		}

		// Register 2 is the constant's check register, which no node
		// addresses: a stale label.
		stale := filepath.Join(dir, "stale.alm")
		writeFile(t, stale, b)
		writeFile(t, filepath.Join(dir, "stale.variables"), []byte(fmt.Sprintf(`{"variables":[{"register":2,"label":"Old"},{"register":%d,"label":"Gold"}]}`, r0)))
		s := x.Editor()
		if err := s.Open(stale); err != nil || s.Message() != "" {
			t.Fatalf("%s: a stale label did not open cleanly: %v %q", l.name, err, s.Message())
		}
		if _, ok := s.Document().Script.Labels[2]; ok || s.Document().Script.Labels[r0] != "Gold" {
			t.Fatalf("%s: the stale map reads labels %v", l.name, s.Document().Script.Labels)
		}
		if _, err := s.Document().Edits.(ui.InspectionInPlace).SaveInPlace(); err != nil {
			t.Fatal(err)
		}
		if got, _ := os.ReadFile(filepath.Join(dir, "stale.variables")); string(got) != string(labels) {
			t.Fatalf("%s: the save kept the stale label: %q", l.name, got)
		}

		book := words.For(l.profile.Language)
		bad := filepath.Join(dir, "bad.alm")
		writeFile(t, bad, b)
		writeFile(t, filepath.Join(dir, "bad.variables"), []byte("{bad"))
		m := x.Editor()
		if err := m.Open(bad); err != nil {
			t.Fatal(err)
		}
		lead, _, _ := strings.Cut(book.Text("editor.labels.unread"), "%v")
		if !strings.HasPrefix(m.Message(), lead) || len(m.Document().Script.Labels) != 0 {
			t.Fatalf("%s: a malformed file opened with note %q and labels %v, want the note %q", l.name, m.Message(), m.Document().Script.Labels, lead)
		}
		if _, err := m.Document().Edits.(ui.InspectionInPlace).SaveInPlace(); err != nil {
			t.Fatal(err)
		}
		if got, _ := os.ReadFile(filepath.Join(dir, "bad.variables")); string(got) != "{bad" {
			t.Fatalf("%s: the save replaced the malformed file with %q", l.name, got)
		}
		if got, _ := os.ReadFile(bad); !bytes.Equal(got, b) {
			t.Fatalf("%s: the save moved the map's bytes", l.name)
		}
	}
}

// The label file's writer replaces only the map's own file: a save in place
// with no label removes it, a save under a new name leaves a file already
// there and, with labels to write, says so; a directory in the way is never
// replaced; empty labels (a file the open could not read) leave it as it is.
func TestScriptsVariableLabelsFilePlacement(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "map.alm")
	file := filepath.Join(dir, "map.variables")
	labels := encodeMapLabels(map[uint32]string{7: "Gold"})
	for _, c := range []struct {
		name      string
		labels    []byte
		overwrite bool
		kept      bool
		refused   bool
	}{
		{"new name, no label", nil, false, true, false},
		{"new name, labels", labels, false, true, true},
		{"empty labels", []byte{}, true, true, false},
		{"in place, labels", labels, true, false, false},
		{"in place, no label", nil, true, false, false},
	} {
		writeFile(t, file, []byte("old"))
		err := placeMapLabels(target, c.labels, c.overwrite, "en")
		got, rerr := os.ReadFile(file)
		kept := rerr == nil && string(got) == "old"
		if kept != c.kept || errors.As(err, new(labelsUnwritten)) != c.refused || err != nil && !c.refused {
			t.Errorf("%s: kept %v, error %v", c.name, kept, err)
		}
		if c.name == "in place, labels" && string(got) != string(labels) {
			t.Errorf("%s: wrote %q", c.name, got)
		}
		if c.name == "in place, no label" && !os.IsNotExist(rerr) {
			t.Errorf("%s: the file stayed: %v", c.name, rerr)
		}
	}
	if err := os.Remove(file); err != nil && !os.IsNotExist(err) {
		t.Fatal(err)
	}
	if err := os.Mkdir(file, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := placeMapLabels(target, labels, true, "en"); !errors.As(err, new(labelsUnwritten)) {
		t.Fatalf("a directory in the way answered %v", err)
	}
	if fi, err := os.Stat(file); err != nil || !fi.IsDir() {
		t.Fatalf("the directory in the way was replaced: %v", err)
	}
}

// The label file's reader refuses what a mapmaker's hand or another tool
// could leave: unknown fields, trailing data, a register of 100 or more,
// one register twice, and a blank or control-character label; it keeps
// labels of up to 64 characters, Cyrillic ones included, and reads no file
// as no labels.
func TestScriptsVariableLabelsFileReader(t *testing.T) {
	dir := t.TempDir()
	x := heroLabelInspector(t, base.Profile{Language: "english"}, 0)
	source := filepath.Join(dir, "map.alm")
	if got, err := x.mapLabels(source); got != nil || err != nil {
		t.Fatalf("no file reads %v %v", got, err)
	}
	for _, c := range []struct {
		body string
		ok   bool
	}{
		{`{"variables":[{"register":99,"label":"Золото"}]}`, true},
		{`{"variables":[{"register":5,"label":"` + strings.Repeat("я", 64) + `"}]}`, true},
		{`{"variables":[{"register":5,"label":"` + strings.Repeat("я", 65) + `"}]}`, false},
		{`{"variables":[{"register":100,"label":"x"}]}`, false},
		{`{"variables":[{"register":5,"label":"x"},{"register":5,"label":"y"}]}`, false},
		{`{"variables":[{"register":5,"label":" "}]}`, false},
		{`{"variables":[{"register":5,"label":"a\u0007"}]}`, false},
		{`{"variables":[{"register":5,"label":"x","start":1}]}`, false},
		{`{"variables":[]} {}`, false},
	} {
		writeFile(t, filepath.Join(dir, "map.variables"), []byte(c.body))
		got, err := x.mapLabels(source)
		if (err == nil) != c.ok || c.ok && len(got) != 1 {
			t.Errorf("%s: read %v %v", c.body, got, err)
		}
	}
}
