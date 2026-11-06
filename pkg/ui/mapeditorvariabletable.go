package ui

import (
	"strconv"
	"strings"
)

// The variables table: the Window menu's list of every variable the map's
// scripts use, one row per use, in four columns: the register and its
// label, the value, the comparison or action, and the trigger. It stands in
// the list popup as the list kind "variables", opened as a records list
// whose filter is editorVariablesFilter. Its rows read scriptVariables, so
// it names the variables the Variables list names; a row chooses its
// trigger, or a variable no trigger uses, by chooseListed, as a Scripts list
// row does.

// editorVariablesFilter is the list popup's filter while it shows the
// variables table.
const editorVariablesFilter = 12

// variableColumns are the table's column heads' words and shares.
var variableColumns = []struct {
	word  string
	share int
}{
	{word: "editor.variables.register", share: 26},
	{word: "editor.variables.value", share: 20},
	{word: "editor.variables.how", share: 24},
	{word: "editor.variables.trigger", share: 30},
}

// variableTableRow is one row of the table: its cells, in column order, and
// the record a press on it chooses.
type variableTableRow struct {
	cells  [4]string
	record int
}

// variablesShown reports that the list popup shows the variables table.
func (e *MapEditor) variablesShown() bool { return e.listKind() == "variables" }

// toggleVariables opens the variables table in the list popup, or closes it
// when it shows.
func (e *MapEditor) toggleVariables() {
	if e.variablesShown() {
		e.closeList()
		e.listRestore = nil
		return
	}
	e.showRecords(editorVariablesFilter)
}

// variableTableRows are the table's rows: each variable scriptVariables
// lists, in register order, one row per use in use order, and one row with
// no trigger for a variable no trigger uses.
func (e *MapEditor) variableTableRows() []variableTableRow {
	var rows []variableTableRow
	for i, v := range e.scriptVariables() {
		name := strconv.FormatUint(uint64(v.register), 10)
		if v.label != "" {
			name += "  " + e.editorHostText(v.label)
		}
		if len(v.uses) == 0 {
			rows = append(rows, variableTableRow{cells: [4]string{name, "", "", ""}, record: -2 - i})
			continue
		}
		for _, u := range v.uses {
			t := e.doc.Script.Triggers[u.trigger]
			value, how := e.variableUseWords(t, u)
			rows = append(rows, variableTableRow{cells: [4]string{name, value, how, e.scriptRowText(t.Record, 0)}, record: t.Record})
		}
	}
	return rows
}

// variableUseWords are a use's value and its comparison or action: for a
// check, what the other side of its pair is (a constant's number, else the
// other check's title) and the pair's comparison read from the check's own
// side (mirrored when the check is on the right); for an action, what it
// writes and its name.
func (e *MapEditor) variableUseWords(t InspectionTrigger, u scriptVariableUse) (string, string) {
	if u.action {
		return e.variableWrite(u.node), e.opName(false, u.node.Opcode)
	}
	other := t.Right[u.slot]
	if u.right {
		other = t.Left[u.slot]
	}
	value := ""
	if c, ok := e.constantValue(other); ok {
		value = strconv.FormatInt(c, 10)
	} else if n, ok := e.nodeByID(true, other); ok {
		value = e.nodeTitle(true, n)
	}
	cmp := t.Cmp[u.slot]
	if u.right && cmp < uint32(len(mirroredCmps)) {
		cmp = mirroredCmps[cmp]
	}
	return value, e.cmpWord(int64(cmp))
}

// variableWrite is the value action n writes into its variable: 1 for the
// action that adds 1, else the parameter after the variable's (TRIG-ACT-004).
func (e *MapEditor) variableWrite(n InspectionScriptNode) string {
	const addOne = 8
	if n.Opcode == addOne {
		return "1"
	}
	v := scriptView{node: &n}
	at := -1
	for s, t := range n.Types {
		switch {
		case at < 0 && e.nodeParameterKind(v, n.Opcode, n.Types, s) == ScriptVariableParameter:
			at = s
		case at >= 0 && t != scriptNone:
			return strconv.FormatInt(slotValue(t, n.Values[s]), 10)
		}
	}
	return ""
}

// columnCells is row text cut to the columns of shares at width: each
// cell fitted to its column, the cells parted by a tab as listBox's Columns
// read them.
func (e *MapEditor) columnCells(text string, shares []int, width int) string {
	stops := columnStops(shares, width)
	cells := strings.SplitN(text, "\t", len(shares))
	for k := range cells {
		cells[k] = editorClip(e.bodyFont(), cells[k], stops[k+1]-stops[k]-listColumnGap)
	}
	return strings.Join(cells, "\t")
}

// variableTableSpec gives list popup s the table's columns, width, row text
// and, with no row, the note that the map uses no variable.
func (e *MapEditor) variableTableSpec(s *popupSpec, m *Picker) {
	for _, c := range variableColumns {
		s.ListColumns = append(s.ListColumns, listColumn{Head: e.word(c.word), Share: c.share})
	}
	shares := listShares(s.ListColumns)
	s.Width = e.popupWide(popList, 680)
	s.ListText = func(row, width int) string {
		rows := m.Rows()
		if row < 0 || row >= len(rows) {
			return ""
		}
		return e.columnCells(rows[row].Text, shares, width)
	}
	if m.Len() == 0 {
		s.Notes = []string{e.word("editor.variables.none")}
	}
}

// variablesChoose is a press on table row n: it chooses the row's record.
func (e *MapEditor) variablesChoose(n int) {
	rows := e.variableTableRows()
	if n < 0 || n >= len(rows) {
		return
	}
	e.model.Select(n)
	e.chooseListed(rows[n].record)
}
