package game

import (
	"fmt"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/ui"
	"againrom/pkg/words"
)

// variableTableMap holds two triggers. "Alpha" compares the check on
// register 20 with the constant 10 (is greater than) in its first pair and
// with the constant 3 (does not equal) in its second, and sets register 21
// to 42. "Beta" compares the check on register 21 with 10 (equals), adds 1
// to register 22 and wins. Register 21 is used in both triggers.
func variableTableMap(t *testing.T) []byte {
	t.Helper()
	conditions := []scriptNode{
		{label: "score", opcode: 19, id: 1, values: []uint32{20}, tags: []uint32{kInt}},
		{label: "ten", opcode: 0x10002, id: 2, values: []uint32{10}, tags: []uint32{kInt}},
		{label: "three", opcode: 0x10002, id: 3, values: []uint32{3}, tags: []uint32{kInt}},
		{label: "gold", opcode: 19, id: 4, values: []uint32{21}, tags: []uint32{kInt}},
	}
	actions := []scriptNode{
		{label: "set", opcode: 3, id: 1, values: []uint32{21, 42}, tags: []uint32{kInt, kInt}},
		{label: "count", opcode: 8, id: 2, values: []uint32{22}, tags: []uint32{kInt}},
		{label: "win", opcode: 4, id: 3},
	}
	o := synth.ALMOptions{Width: 40, Height: 40, Type7Payload: goalScript(actions, conditions, []goalTrigger{
		{name: "Alpha", left: 1, right: 2, cmp: 2, actions: []uint32{1}, raw: map[int]uint32{0x88: 1, 0x8c: 3, 0xac: 1}},
		{name: "Beta", left: 4, right: 2, actions: []uint32{2, 3}},
	})}
	for i := 0; i < 2; i++ {
		o.Units = append(o.Units, synth.ALMUnit{X: uint32(12+3*i)*256 + 128, Y: 14*256 + 128, ClassID: 40, ClassSubID: 2})
	}
	return built(t, o, nil, nil).Bytes()
}

// variableTableEditor opens b with the label file that names register 40,
// which no node addresses, "Spare", at 1920x1080 in l's words. It answers
// the editor, the conversion of UTF-8 to the font's bytes, and word.
func variableTableEditor(t *testing.T, l scriptAddLang, b []byte) (*ui.MapEditor, func(string) string, func(id string, a ...any) string) {
	t.Helper()
	x := heroLabelInspector(t, l.profile, l.sel)
	dir := t.TempDir()
	path := filepath.Join(dir, "table.alm")
	writeFile(t, path, b)
	writeFile(t, filepath.Join(dir, "table.variables"), []byte(`{"variables":[{"register":40,"label":"Spare"}]}`))
	e := x.Editor()
	if err := e.Open(path); err != nil || e.Message() != "" {
		t.Fatalf("open: %v %q", err, e.Message())
	}
	e.Layout(1920, 1080)
	closeMapPanel(t, e)
	book := words.For(l.profile.Language)
	return e, x.inspectionUTF8, func(id string, a ...any) string { return x.inspectionUTF8(fmt.Sprintf(book.Text(id), a...)) }
}

// variableTableToggle presses the Window menu's Variables table row.
func variableTableToggle(t *testing.T, e *ui.MapEditor) {
	t.Helper()
	r, ok := e.FindControl(ui.ControlQuery{Role: ui.ControlBar, Key: "window"})
	trigClick(t, e, r, ok, "the Window menu")
	n, ok := e.MenuRow("window", "variables")
	if !ok {
		t.Fatal("the Window menu has no Variables table row")
	}
	r, ok = e.FindControl(ui.ControlQuery{Role: ui.ControlMenuRow, N: n})
	trigClick(t, e, r, ok, "the Variables table row")
}

// variableTableRows are the expected rows of the table, cells parted by a
// tab, in l's words: EN and RU spelled out, so a word that changes in the
// words file fails here.
func variableTableRows(l scriptAddLang, enc func(string) string) []string {
	cells := map[string][]string{
		"en": {"is greater than", "does not equal", "Set variable", "equals", "Add 1 to variable", "Trigger 1  Alpha", "Trigger 2  Beta"},
		"ru": {"больше", "не равно", "Задать переменную", "равно", "Прибавить 1 к переменной", "Триггер 1  Alpha", "Триггер 2  Beta"},
	}[l.name]
	for i := range cells {
		cells[i] = enc(cells[i])
	}
	return []string{
		tableRow("20", "10", cells[0], cells[5]),
		tableRow("20", "3", cells[1], cells[5]),
		tableRow("21", "42", cells[2], cells[5]),
		tableRow("21", "10", cells[3], cells[6]),
		tableRow("22", "1", cells[4], cells[6]),
		tableRow("40  Spare", "", "", ""),
	}
}

// tableRow is a table row's text: its cells parted by a tab.
func tableRow(cells ...string) string { return strings.Join(cells, "\t") }

// variableListRegisters are the registers the Scripts tree's Variables
// section names, in its order.
func variableListRegisters(e *ui.MapEditor, w func(string, ...any) string) []string {
	prefix := strings.TrimSuffix(w("editor.scripts.variable", 0), "0")
	showTree(e)
	lines := e.TreeLines()
	var out []string
	for n := 0; ; n++ {
		row := e.TreeMemberRow("scripts", n)
		if row < 0 {
			return out
		}
		if text := strings.TrimSpace(lines[row]); strings.HasPrefix(text, prefix) {
			out = append(out, strings.Fields(strings.TrimPrefix(text, prefix))[0])
		}
	}
}

// tableRegisters are the registers of the table's rows, each once, in row
// order.
func tableRegisters(lines []string) []string {
	var out []string
	for _, line := range lines {
		if r := strings.Fields(line)[0]; len(out) == 0 || out[len(out)-1] != r {
			out = append(out, r)
		}
	}
	return out
}

// The Window menu's Variables table lists every variable the map's scripts
// use, one row per use in register, trigger and slot order: a variable
// compared with a number in two pairs with two comparisons, one set and one
// added to by an action, one used in two triggers, and a labelled variable
// no trigger uses on a row with no trigger. The words are EN and RU. A
// press on a row selects its trigger as the Scripts list does, the unused
// variable's row selects the variable; the table names the Variables list's
// registers and follows a wizard Apply, a trigger's Delete and their undo.
func TestVariablesTableListsEachUseAndFollowsEdits(t *testing.T) {
	b := variableTableMap(t)
	for _, l := range scriptAddLangs {
		t.Run(l.name, func(t *testing.T) {
			e, enc, w := variableTableEditor(t, l, b)
			variableTableToggle(t, e)
			want := variableTableRows(l, enc)
			if got := e.ListLines(); !slices.Equal(got, want) {
				t.Fatalf("rows\n%q\nwant\n%q", got, want)
			}
			if got, list := tableRegisters(e.ListLines()), variableListRegisters(e, w); !slices.Equal(got, list) || !slices.Equal(got, []string{"20", "21", "22", "40"}) {
				t.Fatalf("the table names registers %v, the Variables list %v", got, list)
			}

			for _, c := range []struct {
				row    int
				record int
			}{{3, trigRecord(t, e, "Trigger", 1)}, {0, trigRecord(t, e, "Trigger", 0)}, {5, -5}} {
				r, ok := e.FindControl(ui.ControlQuery{Role: ui.ControlListRow, N: c.row})
				trigClick(t, e, r, ok, "table row "+strconv.Itoa(c.row))
				if e.Selected() != c.record {
					t.Fatalf("row %d selected %d, want %d", c.row, e.Selected(), c.record)
				}
			}

			// A use added in the wizard: Beta adds 1 to Spare.
			e.Select(trigRecord(t, e, "Trigger", 1), true)
			wizPress(t, e, "+action", 0)
			last := -1
			for n := 0; e.WizardField(fmt.Sprintf("a%d.item", n)) >= 0; n++ {
				last = n
			}
			trigChoose(t, e, wizField(t, e, fmt.Sprintf("a%d.item", last)), w("editor.wizard.a.addvar"))
			trigChoose(t, e, wizField(t, e, fmt.Sprintf("a%d.p0", last)), "Spare")
			wizApply(t, e)
			added := slices.Clone(want)
			added[5] = tableRow("40  Spare", "1", w("editor.action.op8"), w("editor.trigger.title", 2)+"  Beta")
			if got := e.ListLines(); !slices.Equal(got, added) {
				t.Fatalf("after the wizard's Apply\n%q\nwant\n%q", got, added)
			}
			if err := e.Key("undo"); err != nil {
				t.Fatal(err)
			}
			if got := e.ListLines(); !slices.Equal(got, want) {
				t.Fatalf("after undo\n%q\nwant\n%q", got, want)
			}

			// Alpha deleted: its uses leave the table, Beta becomes
			// trigger 1, and register 20, which Alpha's check still
			// addresses, keeps a row with no trigger.
			e.Select(trigRecord(t, e, "Trigger", 0), false)
			if err := e.Key("delete"); err != nil {
				t.Fatal(err)
			}
			beta := w("editor.trigger.title", 1) + "  Beta"
			deleted := []string{
				tableRow("20", "", "", ""),
				tableRow("21", "10", w("editor.cmp.0"), beta),
				tableRow("22", "1", w("editor.action.op8"), beta),
				tableRow("40  Spare", "", "", ""),
			}
			if got := e.ListLines(); !slices.Equal(got, deleted) {
				t.Fatalf("after Delete\n%q\nwant\n%q", got, deleted)
			}
			if got, list := tableRegisters(e.ListLines()), variableListRegisters(e, w); !slices.Equal(got, list) {
				t.Fatalf("after Delete the table names %v, the Variables list %v", got, list)
			}
			if err := e.Key("undo"); err != nil {
				t.Fatal(err)
			}
			if got := e.ListLines(); !slices.Equal(got, want) {
				t.Fatalf("after the second undo\n%q\nwant\n%q", got, want)
			}
			if !slices.Contains(e.OpenWindows(), "list") {
				t.Fatalf("the table closed: %q", e.OpenWindows())
			}
		})
	}
}
