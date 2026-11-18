package ui

import (
	"fmt"
	"image"
	"slices"
	"strings"
	"testing"
	"time"

	"againrom/pkg/render/text"
	"againrom/pkg/render/textfit"
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
			if again, _ := e.FindControl(ControlQuery{Role: ControlTitle}); again.Min != after.Min {
				t.Fatalf("the reopened table stands at %v, was %v", again, after)
			}
			r, ok := e.FindControl(ControlQuery{Role: ControlListRow, N: 1})
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
			r, _ = e.FindControl(ControlQuery{Role: ControlListRow, N: 2})
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

// longTriggerName is a trigger name wider than a 1920 pixel screen.
var longTriggerName = strings.Repeat("Open the northern gate after the scout returns ", 12)

// variableTablePagedEditor is variableTableEditor with the trigger Count
// renamed name and uses more triggers comparing register 30 with 7, enough
// rows to page.
func variableTablePagedEditor(t *testing.T, lang, name string, uses int) *MapEditor {
	t.Helper()
	e := variableTableEditor(t, lang)
	s := e.doc.Script
	s.Triggers[len(s.Triggers)-1].Name = name
	for i := 0; i < uses; i++ {
		record := len(e.doc.Records)
		e.doc.Records = append(e.doc.Records, InspectionRecord{Kind: "Trigger", Index: len(s.Triggers), Key: uint64(record + 1)})
		s.Triggers = append(s.Triggers, InspectionTrigger{Record: record, Name: fmt.Sprintf("Use %d", i), Left: []uint32{1, 0, 0}, Right: []uint32{2, 0, 0}, Cmp: []uint32{0, 0, 0}, Acts: make([]uint32, 4)})
	}
	return e
}

// variableColumnNeeds are the widths the table's columns ask, measured here:
// each head in the title face (its advance, or its ink when wider) and each
// cell of every row in the body face by advance,
// plus the gap between columns.
func variableColumnNeeds(e *MapEditor, s popupSpec) [4]int {
	var need [4]int
	for k, c := range s.ListColumns {
		ink, _ := e.font.Measure(c.Head)
		need[k] = max(e.font.Advance(c.Head), ink)
	}
	for _, line := range e.ListLines() {
		for k, cell := range strings.SplitN(line, "\t", 4) {
			w := e.bodyFont().Advance(cell)
			need[k] = max(need[k], w)
		}
	}
	for k := range need {
		need[k] += listColumnGap
	}
	return need
}

// At 1920x1080 the variables table, with rows enough to page and a trigger
// name wider than the screen, lies inside the screen with every head whole
// on one line; the cut cell ends in the cut mark and shows its whole text in
// the tip after the shared delay through the editor's input; and a press on
// each visible row, through the editor's input, chooses that row's record.
func TestVariablesTableHoldsTheLayoutProperties(t *testing.T) {
	for _, lang := range []string{"", "russian"} {
		t.Run(fmt.Sprintf("%q", lang), func(t *testing.T) {
			e := variableTablePagedEditor(t, lang, longTriggerName, 40)
			openMenuRow(t, e, "window", "variables")
			s, ok := e.listSpec()
			if !ok {
				t.Fatal("no list spec")
			}
			checkWindowProperties(t, e, s, image.Pt(1920, 1080))
			l := layoutPopup(s, e.font, e.bodyFont())
			screen := image.Rect(0, 0, 1920, 1080)
			if s.List.Len() <= l.List.Rows || !l.List.Bar().In(l.Rect) {
				t.Fatalf("%d rows in a list of %d rows with its bar at %v: the list does not page inside %v", s.List.Len(), l.List.Rows, l.List.Bar(), l.Rect)
			}
			for k, r := range l.ListHeads {
				head := s.ListColumns[k].Head
				if w, _ := e.font.Measure(head); len(l.ListHeadLines[k]) != 1 || l.ListHeadLines[k][0] != head || w > r.Dx() || !r.In(l.Rect) {
					t.Errorf("head %d %q laid out as %q, %d wide in %v inside %v", k, head, l.ListHeadLines[k], w, r, l.Rect)
				}
			}
			// The trigger Count is the first two rows' trigger.
			cut := l.List.Cell(0, 3)
			full := strings.SplitN(e.ListLines()[0], "\t", 4)[3]
			if !cut.In(l.Rect) || !l.Rect.In(screen) {
				t.Fatalf("the cut cell %v or the window %v leaves the screen", cut, l.Rect)
			}
			var painted []text.DrawCall
			calls := text.Record(func() { e.chromeCompose(false) })
			for _, c := range calls {
				if c.Y == cut.Min.Y+listTextY && c.X >= cut.Min.X && c.X < cut.Max.X {
					painted = append(painted, c)
				}
			}
			shown := editorFit(e.bodyFont()).Fit(full, cut.Dx())
			if !strings.HasSuffix(shown, textfit.Mark) || len(painted) != len(shown) {
				t.Fatalf("the cut cell paints %d glyphs, want %d for %q", len(painted), len(shown), shown)
			}
			for _, c := range painted[len(painted)-len(textfit.Mark):] {
				if c.Glyph != e.bodyFont().GlyphFor('.') || c.X+c.Glyph.Width > cut.Max.X {
					t.Fatalf("the cut cell does not end in the cut mark inside %v", cut)
				}
			}
			p := center(cut)
			t0 := time.Unix(100, 0)
			e.Pointer(Input{CursorX: p.X, CursorY: p.Y}, t0)
			if got := e.TipShown(); got != "" {
				t.Fatalf("the cut cell's tip shows before the delay: %q", got)
			}
			e.Pointer(Input{CursorX: p.X, CursorY: p.Y}, t0.Add(editorTipDelay*time.Millisecond))
			if got := e.TipShown(); got != full {
				t.Fatalf("the cut cell's tip reads %q, want the whole %q", got, full)
			}
			if _, ok := popupTipAt(screen, e.font, e.bodyFont(), s, l, center(l.List.Cell(0, 0)), true); ok {
				t.Fatal("a whole cell shows a tip")
			}
			rows := e.variableTableRows()
			top, count := e.model.Visible()
			for n := top; n < top+min(count, l.List.Rows); n++ {
				r, ok := e.FindControl(ControlQuery{Role: ControlListRow, N: n})
				if !ok {
					t.Fatalf("row %d is not shown", n)
				}
				s, _ := e.listSpec()
				if part, i := layoutPopup(s, e.font, e.bodyFont()).hit(center(r)); part != popupListRow || i != n-top {
					t.Errorf("hit at row %d gives part %d, index %d", n, part, i)
				}
				modesClick(e, center(r))
				if e.selected != rows[n].record || e.listKind() != "variables" {
					t.Fatalf("a press on row %d selected %d, want %d; the list shows %q", n, e.selected, rows[n].record, e.listKind())
				}
			}
		})
	}
}

// Owner answer 1: the head stays Value («Значение»), and the title's (?)
// says that Value is the number in the trigger's condition or action, not
// the variable's current value, in its tip.
func TestVariablesTableHelpSaysWhatValueIs(t *testing.T) {
	for _, c := range []struct {
		lang, head string
		says       []string
	}{
		{"", "Value", []string{"Value is the number in the trigger's condition or action", "not the variable's current value"}},
		{"russian", "Значение", []string{"Значение: число в условии или действии триггера", "а не текущее значение переменной"}},
	} {
		t.Run(fmt.Sprintf("%q", c.lang), func(t *testing.T) {
			e := variableTableEditor(t, c.lang)
			openMenuRow(t, e, "window", "variables")
			s, _ := e.listSpec()
			if s.ListColumns[1].Head != c.head {
				t.Fatalf("the value head reads %q, want %q", s.ListColumns[1].Head, c.head)
			}
			l := layoutPopup(s, e.font, e.bodyFont())
			tip, ok := popupTipAt(image.Rect(0, 0, 1920, 1080), e.font, e.bodyFont(), s, l, center(l.TitleHelp), true)
			if l.TitleHelp.Empty() || !ok || !tip.Help || tip.Text != e.word("editor.variables.help") {
				t.Fatalf("the title's help mark at %v shows %+v (%v)", l.TitleHelp, tip, ok)
			}
			for _, say := range c.says {
				if !strings.Contains(tip.Text, say) {
					t.Errorf("the help %q does not say %q", tip.Text, say)
				}
			}
		})
	}
}

// Owner answer 2: with a trigger name wider than the screen the first three
// columns take their natural width, the trigger column takes all the room
// left in a window as wide as the screen allows, and its cut text ends in the
// cut mark. With short names every column takes its natural width.
func TestVariablesTableLongTriggerTakesTheRoomLeft(t *testing.T) {
	for _, lang := range []string{"", "russian"} {
		for _, long := range []bool{false, true} {
			t.Run(fmt.Sprintf("%q/long=%v", lang, long), func(t *testing.T) {
				name := "Count"
				if long {
					name = longTriggerName
				}
				e := variableTablePagedEditor(t, lang, name, 0)
				openMenuRow(t, e, "window", "variables")
				s, _ := e.listSpec()
				l := layoutPopup(s, e.font, e.bodyFont())
				need := variableColumnNeeds(e, s)
				row := l.List.Rect.Dx() - 2*listTextX
				for k := 0; k < 3; k++ {
					if got := l.List.Cell(0, k).Dx() + listColumnGap; got != need[k] {
						t.Errorf("column %d is %d wide, its natural width is %d", k, got, need[k])
					}
				}
				last := l.List.Cell(0, 3).Dx() + listColumnGap
				if long {
					if l.Rect.Dx() != e.window.X-16 || last != row-need[0]-need[1]-need[2] || last >= need[3] {
						t.Fatalf("window %d wide (screen %d), trigger column %d of a %d row, natural %v", l.Rect.Dx(), e.window.X, last, row, need)
					}
					full := strings.SplitN(e.ListLines()[0], "\t", 4)[3]
					if shown := editorFit(e.bodyFont()).Fit(full, l.List.Cell(0, 3).Dx()); !strings.HasSuffix(shown, textfit.Mark) {
						t.Fatalf("the cut trigger reads %q", shown)
					}
					return
				}
				if last != need[3] {
					t.Errorf("the trigger column is %d wide, its natural width is %d", last, need[3])
				}
			})
		}
	}
}

// The table's window is as wide as listFit makes it from the columns' need
// by advance, the one text measure of a list window, and the brush and
// Players windows keep their width.
func TestVariablesTableWidthComesFromListFit(t *testing.T) {
	for _, lang := range []string{"", "russian"} {
		t.Run(fmt.Sprintf("%q", lang), func(t *testing.T) {
			e := variableTablePagedEditor(t, lang, "Count", 0)
			openMenuRow(t, e, "window", "variables")
			s, _ := e.listSpec()
			need := variableColumnNeeds(e, s)
			want := 2*popupPad + widgetSpriteSize + s.ListAside + 2*listTextX
			for _, n := range need {
				want += n
			}
			if got := layoutPopup(s, e.font, e.bodyFont()).Rect.Dx(); got != want {
				t.Fatalf("the table window is %d wide, the columns' advance need makes it %d", got, want)
			}
			fit := s
			fit.Width = 0
			e.listFit(&fit, listColumnsFit(&fit, e.font, e.window.X-16))
			if fit.Width != s.Width {
				t.Fatalf("listFit gives the table %d wide, the window spec has %d", fit.Width, s.Width)
			}
		})
	}
}
