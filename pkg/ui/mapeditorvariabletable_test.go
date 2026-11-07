package ui

import (
	"fmt"
	"image"
	"slices"
	"strings"
	"testing"

	"againrom/pkg/render/text"
)

// variableTableEditor is the menu map with a trigger "Count" (record 13)
// whose pair compares the check on register 30 with the constant 7 (is at
// least) and whose action adds 1 to register 31, and the label "Spare" on
// register 32, which no node addresses.
func variableTableEditor(t *testing.T, lang string) *MapEditor {
	t.Helper()
	e, _, _ := menuEditor(t, lang)
	for _, r := range []InspectionRecord{{Kind: "Condition", Index: 0}, {Kind: "Condition", Index: 1}, {Kind: "Action", Index: 1}, {Kind: "Trigger", Index: 1}} {
		r.Key = uint64(len(e.doc.Records) + 1)
		e.doc.Records = append(e.doc.Records, r)
	}
	s := e.doc.Script
	s.Conditions = []InspectionScriptNode{
		{Record: 10, ID: 1, Opcode: 19, Types: [10]uint32{scriptInt}, Values: [10]uint32{30}, AddressesVariable: true, VariableRegister: 30},
		{Record: 11, ID: 2, Opcode: 0x10002, Types: [10]uint32{scriptInt}, Values: [10]uint32{7}},
	}
	s.Actions = append(s.Actions, InspectionScriptNode{Record: 12, ID: 2, Opcode: 8, Types: [10]uint32{scriptInt}, Values: [10]uint32{31}, AddressesVariable: true, VariableRegister: 31})
	s.Triggers = append(s.Triggers, InspectionTrigger{Record: 13, Name: "Count", Left: []uint32{1, 0, 0}, Right: []uint32{2, 0, 0}, Cmp: []uint32{4, 0, 0}, Acts: []uint32{2, 0, 0, 0}})
	s.Labels = map[uint32]string{32: "Spare"}
	return e
}

// variableTableWant are the editor's expected table rows, cells parted by a
// tab.
func variableTableWant(e *MapEditor) []string {
	trigger := fmt.Sprintf(e.word("editor.trigger.title"), 2) + "  Count"
	return []string{
		"30\t7\t" + e.word("editor.cmp.4") + "\t" + trigger,
		"31\t1\t" + e.opName(false, 8) + "\t" + trigger,
		"32  Spare\t\t\t",
	}
}

// The Window menu's Variables table row opens the table in the list popup
// and carries the open mark while it shows; a second press closes it. The
// table's title and four column heads are the words in EN and RU, each head
// laid out over its column above the list, and its rows are the variables
// in register order, one per use, a variable no trigger uses on a row with
// no trigger. Its registers are the Variables list's.
func TestVariablesTableOpensFromTheWindowMenuWithItsColumns(t *testing.T) {
	for _, c := range []struct {
		lang  string
		title string
		heads []string
	}{
		{"", "Variables table", []string{"Variable", "Value", "Comparison or action", "Trigger"}},
		{"russian", "Таблица переменных", []string{"Номер переменной", "Значение", "=, >, < и так далее", "Название триггера"}},
	} {
		t.Run(fmt.Sprintf("%q", c.lang), func(t *testing.T) {
			e := variableTableEditor(t, c.lang)
			if label, ok := menuLabel(e, "window", "variables"); !ok || label != markWord(false, c.title) {
				t.Fatalf("the Window menu's row reads %q %v", label, ok)
			}
			openMenuRow(t, e, "window", "variables")
			if e.listKind() != "variables" || e.listTitle("variables") != c.title {
				t.Fatalf("the row opened %q titled %q", e.listKind(), e.listTitle(e.listKind()))
			}
			if label, _ := menuLabel(e, "window", "variables"); label != markWord(true, c.title) {
				t.Fatalf("the open table's row reads %q", label)
			}
			assertMenuMarks(t, e, "table open")
			s, ok := e.listSpec()
			if !ok {
				t.Fatal("no list spec")
			}
			var heads []string
			for _, col := range s.ListColumns {
				heads = append(heads, col.Head)
			}
			if !slices.Equal(heads, c.heads) {
				t.Fatalf("heads %q, want %q", heads, c.heads)
			}
			l := layoutPopup(s, e.font, e.bodyFont())
			if len(l.ListHeads) != 4 || len(l.List.Columns) != 4 {
				t.Fatalf("%d heads and %d columns laid out", len(l.ListHeads), len(l.List.Columns))
			}
			for k, r := range l.ListHeads {
				if got := strings.Join(l.ListHeadLines[k], " "); got != c.heads[k] {
					t.Errorf("head %d wraps to %q", k, l.ListHeadLines[k])
				}
				if !r.In(l.Rect) || r.Max.Y > l.List.Rect.Min.Y || k > 0 && r.Min.X < l.ListHeads[k-1].Max.X {
					t.Errorf("head %d at %v, list at %v, popup at %v", k, r, l.List.Rect, l.Rect)
				}
			}
			if got, want := e.ListLines(), variableTableWant(e); !slices.Equal(got, want) {
				t.Fatalf("rows\n%q\nwant\n%q", got, want)
			}
			var listed []uint32
			for _, v := range e.scriptVariables() {
				listed = append(listed, v.register)
				if row := fmt.Sprintf(e.word("editor.scripts.variable"), v.register); !slices.ContainsFunc(scriptsTreeRows(e), func(r PickerRow) bool { return strings.HasPrefix(strings.TrimSpace(r.Text), row) }) {
					t.Errorf("the Variables list has no row %q", row)
				}
			}
			var shown []uint32
			for _, line := range e.ListLines() {
				var r uint32
				fmt.Sscan(line, &r)
				if len(shown) == 0 || shown[len(shown)-1] != r {
					shown = append(shown, r)
				}
			}
			if !slices.Equal(shown, listed) || !slices.Equal(listed, []uint32{30, 31, 32}) {
				t.Fatalf("the table names registers %v, the Variables list %v", shown, listed)
			}
			openMenuRow(t, e, "window", "variables")
			if e.listKind() != "" || e.listShown || e.filter != 0 {
				t.Fatalf("the second press left %q, filter %d", e.listKind(), e.filter)
			}
			assertMenuMarks(t, e, "table closed")
		})
	}
}

// A press on a table row selects its trigger as a Scripts list row does,
// with the right panel docked and floating; a press on the row of a
// variable no trigger uses selects that variable. The table follows an
// edit on the next frame, moves by its title and keeps its corner when it
// is closed and opened again.
func TestVariablesTableRowSelectsItsTriggerDockedAndFloating(t *testing.T) {
	for _, float := range []bool{false, true} {
		t.Run(fmt.Sprintf("float=%v", float), func(t *testing.T) {
			e := variableTableEditor(t, "")
			if float {
				e.undockPanel(panelRight)
			}
			if e.panelFloats(panelRight) != float || !e.panelOn(panelRight) {
				t.Fatalf("right panel floats %v, shown %v", e.panelFloats(panelRight), e.panelOn(panelRight))
			}
			openMenuRow(t, e, "window", "variables")
			e.places[popList].at = image.Pt(e.mapArea().Min.X+8, e.mapArea().Min.Y+8)
			before, after := dragTitle(t, e, image.Pt(40, 30))
			if after.Min != before.Min.Add(image.Pt(40, 30)) {
				t.Fatalf("the title moved from %v to %v", before, after)
			}
			openMenuRow(t, e, "window", "variables")
			openMenuRow(t, e, "window", "variables")
			if again, _ := e.TitleRect(); again.Min != after.Min {
				t.Fatalf("the reopened table stands at %v, was %v", again, after)
			}
			r, ok := e.ListRowRect(1)
			if !ok {
				t.Fatal("row 1 is not shown")
			}
			modesClick(e, center(r))
			if e.selected != 13 || e.listKind() != "variables" {
				t.Fatalf("row 1 selected %d, the list shows %q", e.selected, e.listKind())
			}
			// The Scripts list's route, chooseListed, on a second editor
			// shows the same trigger window.
			f := variableTableEditor(t, "")
			if float {
				f.undockPanel(panelRight)
			}
			openMenuRow(t, f, "window", "variables")
			f.chooseListed(13)
			if got, want := e.PopupLines(), f.PopupLines(); len(got) == 0 || !slices.Equal(got, want) || e.objectInPanel() != f.objectInPanel() {
				t.Fatalf("the row shows\n%q\nthe Scripts route\n%q", got, want)
			}
			r, _ = e.ListRowRect(2)
			modesClick(e, center(r))
			if v, ok := e.scriptVariableRow(e.selected); !ok || v.register != 32 {
				t.Fatalf("the Spare row selected %d", e.selected)
			}
			e.doc.Script.Triggers[1].Cmp[0] = 1
			if got := e.ListLines(); got[0] != "30\t7\t"+e.word("editor.cmp.1")+"\t"+fmt.Sprintf(e.word("editor.trigger.title"), 2)+"  Count" {
				t.Fatalf("the table did not follow the edit: %q", got)
			}
		})
	}
}

// The table paints each cell inside its column, every row's four cells and
// the four heads, and a rule between two columns.
func TestVariablesTablePaintsEachCellInItsColumn(t *testing.T) {
	e := variableTableEditor(t, "")
	openMenuRow(t, e, "window", "variables")
	s, _ := e.listSpec()
	l := layoutPopup(s, e.font, e.bodyFont())
	dst := image.NewRGBA(image.Rect(0, 0, e.window.X, e.window.Y))
	calls := text.Record(func() { drawPopup(dst, e.font, e.bodyFont(), nil, s, l, editorPopupInk, image.Pt(-1, -1), false) })
	x, width := l.List.Rect.Min.X+listTextX, l.List.Rect.Dx()-2*listTextX
	stops := columnStops(l.List.Columns, width)
	cells := map[[2]int]bool{}
	for _, c := range calls {
		for k, h := range l.ListHeads {
			if c.Y >= h.Min.Y && c.Y < h.Max.Y && c.X >= h.Min.X && c.X < h.Max.X {
				cells[[2]int{-1, k}] = true
			}
		}
		row, ok := l.List.RowAt(image.Pt(x, c.Y))
		if !ok || c.Y < l.List.Rect.Min.Y {
			continue
		}
		k := 0
		for k+1 < len(stops)-1 && c.X >= x+stops[k+1] {
			k++
		}
		if c.Clip.Max.X > x+stops[k+1] || c.Clip.Min.X < x+stops[k] {
			t.Fatalf("row %d column %d glyph clipped to %v, the column runs %d..%d", row, k, c.Clip, x+stops[k], x+stops[k+1])
		}
		cells[[2]int{row, k}] = true
	}
	for _, want := range [][2]int{{-1, 0}, {-1, 1}, {-1, 2}, {-1, 3}, {0, 0}, {0, 1}, {0, 2}, {0, 3}, {1, 0}, {1, 1}, {1, 2}, {1, 3}, {2, 0}} {
		if !cells[want] {
			t.Errorf("nothing painted in row %d column %d (row -1 the heads)", want[0], want[1])
		}
	}
	if cells[[2]int{2, 1}] || cells[[2]int{2, 3}] {
		t.Error("the unused variable's row paints a value or a trigger")
	}
	for _, stop := range stops[1:4] {
		if got := dst.RGBAAt(x+stop-listColumnGap/2, l.List.Rect.Max.Y-2); got != listRuleLight {
			t.Errorf("no rule at %d: %v", x+stop-listColumnGap/2, got)
		}
	}
}

// A map whose scripts use no variable opens an empty table with the note
// that says so.
func TestVariablesTableOfAMapWithNoVariableSaysSo(t *testing.T) {
	e, _, _ := menuEditor(t, "")
	openMenuRow(t, e, "window", "variables")
	s, ok := e.listSpec()
	if !ok || e.listKind() != "variables" || len(e.ListLines()) != 0 || len(s.Notes) == 0 || s.Notes[0] != "The map's scripts use no variable." {
		t.Fatalf("the empty table shows %q, notes %q", e.ListLines(), s.Notes)
	}
}

// columnStops parts a width by its shares and ends at the width.
func TestListColumnStopsPartTheWidth(t *testing.T) {
	if got := columnStops([]int{1, 1, 2}, 400); !slices.Equal(got, []int{0, 100, 200, 400}) {
		t.Fatalf("stops %v", got)
	}
}

// A check on the right side of a pair shows the comparison from its own
// side. The pair "7 cmp variable 30" reads as "variable 30 mirror(cmp) 7" for
// every code, and the pair "variable 30 is less than variable 31" gives
// variable 31 the comparison "is greater than", as the editor's own reading
// of a pair does.
func TestVariablesTableReadsAPairFromTheCheckSide(t *testing.T) {
	e, _, _ := menuEditor(t, "")
	s := e.doc.Script
	s.Conditions = []InspectionScriptNode{
		{Record: 10, ID: 1, Opcode: 19, Types: [10]uint32{scriptInt}, Values: [10]uint32{30}, AddressesVariable: true, VariableRegister: 30},
		{Record: 11, ID: 2, Opcode: 0x10002, Types: [10]uint32{scriptInt}, Values: [10]uint32{7}},
		{Record: 12, ID: 3, Opcode: 19, Types: [10]uint32{scriptInt}, Values: [10]uint32{31}, AddressesVariable: true, VariableRegister: 31},
	}
	for _, r := range []InspectionRecord{{Kind: "Condition", Index: 0}, {Kind: "Condition", Index: 1}, {Kind: "Condition", Index: 2}} {
		r.Key = uint64(len(e.doc.Records) + 1)
		e.doc.Records = append(e.doc.Records, r)
	}
	// The code that holds for the variable when the constant is on the left
	// of a pair with code c: equal and not equal stand, greater than and less
	// than exchange, at least and at most exchange.
	holds := [6]int64{0, 1, 3, 2, 5, 4}
	var want []string
	for c := uint32(0); c < 6; c++ {
		s.Triggers = append(s.Triggers, InspectionTrigger{Record: 13 + int(c), Name: "T", Left: []uint32{2, 0, 0}, Right: []uint32{1, 0, 0}, Cmp: []uint32{c, 0, 0}, Acts: make([]uint32, 4)})
		r := InspectionRecord{Kind: "Trigger", Index: int(c), Key: uint64(len(e.doc.Records) + 1)}
		e.doc.Records = append(e.doc.Records, r)
		want = append(want, "30\t7\t"+e.cmpWord(holds[c]))
	}
	s.Triggers = append(s.Triggers, InspectionTrigger{Record: 19, Name: "T", Left: []uint32{1, 0, 0}, Right: []uint32{3, 0, 0}, Cmp: []uint32{3, 0, 0}, Acts: make([]uint32, 4)})
	e.doc.Records = append(e.doc.Records, InspectionRecord{Kind: "Trigger", Index: 6, Key: uint64(len(e.doc.Records) + 1)})
	n1, _ := e.nodeByID(true, 1)
	n3, _ := e.nodeByID(true, 3)
	want = append(want,
		"30\t"+e.nodeTitle(true, n3)+"\t"+e.cmpWord(3),
		"31\t"+e.nodeTitle(true, n1)+"\t"+e.cmpWord(2))
	var got []string
	for _, row := range e.variableTableRows() {
		got = append(got, strings.Join(row.cells[:3], "\t"))
	}
	if !slices.Equal(got, want) {
		t.Fatalf("rows\n got %q\nwant %q", got, want)
	}
}
