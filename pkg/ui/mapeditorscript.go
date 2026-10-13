package ui

import (
	"fmt"
	"image"
)

// The rail's help lines. Each line must fit the rail width unclipped;
// EditorHelpLines lists every line the window can show so a test can measure
// them with the installed font.
var (
	helpKeys      = []string{"Drag: move / C: clone / Del", "E: fields / S: sack / R: report", "D: drop / G: arms / T, H: brush"}
	helpPickClone = []string{"Clone: click a cell", "Esc: cancel"}
	helpPickSack  = []string{"Sack: click a cell", "Esc: cancel"}
	helpPickDrop  = []string{"Drop cell: click a cell", "Esc: cancel"}
	helpPickGoal  = []string{"Target: click a unit or", "structure / Esc: cancel"}
	helpTerrain   = []string{"T: off / Drag: paint / [ ]: size"}
	helpHeight    = []string{"H: off / Drag: shape / [ ]: size"}
	helpGoals     = []string{"Enter: pick a target", "Up, Down: select / G: close"}
)

// EditorHelpLines is every help line the editor window shows.
func EditorHelpLines() []string {
	var out []string
	for _, rows := range [][]string{helpKeys, helpPickClone, helpPickSack, helpPickDrop, helpPickGoal, helpGoals, helpTerrain, helpHeight} {
		out = append(out, rows...)
	}
	return out
}

// helpLines is the help rows for the window's state: the keys, or the
// instruction of the armed pick.
func (e *MapEditor) helpLines() []string {
	if e.brushActive() {
		if e.brush.mode == brushHeight {
			return helpHeight
		}
		return helpTerrain
	}
	switch e.pick {
	case "clone":
		return helpPickClone
	case "sack":
		return helpPickSack
	case "drop":
		return helpPickDrop
	case "goal":
		return helpPickGoal
	}
	if len(e.report.goals) > 0 {
		return helpGoals
	}
	return helpKeys
}

func (e *MapEditor) script() (InspectionScript, bool) {
	if !e.editable() {
		return nil, false
	}
	s, ok := e.doc.Edits.(InspectionScript)
	return s, ok
}

// scriptKey handles D, which arms the drop cell on the next clicked cell, and
// G, which opens the script panel or closes it.
func (e *MapEditor) scriptKey(key string) {
	s, ok := e.script()
	if !ok || e.catalog {
		return
	}
	e.cancelRecordDrag()
	if key == "drop" {
		e.closeReport()
		e.pick, e.message = "drop", ""
		return
	}
	if len(e.report.goals) > 0 {
		e.closeReport()
		return
	}
	e.pick, e.fieldMode, e.pathMode = "", false, false
	e.openGoals(s, 0)
}

// openGoals shows the script's arms in the report panel's list, with row sel
// selected. A row addresses the script record it edits and scrolls the map to
// the cell its target stands on.
func (e *MapEditor) openGoals(s InspectionScript, sel int) {
	goals := s.Goals()
	rows := make([]InspectionFinding, len(goals))
	for i, g := range goals {
		rows[i] = InspectionFinding{Rule: g.Arm, Kind: g.Kind, Index: g.Index, Cell: g.Cell, HasCell: g.HasCell, Text: g.Text}
	}
	e.openReport(rows, fmt.Sprintf("Script arms: %d", len(goals)))
	e.report.goals = goals
	if len(goals) > 0 {
		e.selectFinding(max(0, min(sel, len(goals)-1)))
	}
}

// goalLabel is a script arm as one list row.
func goalLabel(g InspectionGoal) string { return g.Arm + ": " + g.Text }

// dropAt moves the drop cell onto cell.
func (e *MapEditor) dropAt(cell image.Point) {
	s, ok := e.script()
	if !ok {
		return
	}
	next, err := s.SetDropCell(cell)
	_ = e.adoptEdit(next, err)
}

// armGoal arms the pick of the selected script arm's target.
func (e *MapEditor) armGoal() {
	if len(e.report.goals) == 0 || e.report.sel >= len(e.report.goals) {
		return
	}
	e.pick, e.message = "goal", ""
}

// goalAt gives the selected script arm the unit or structure under p as its
// target, then shows the panel again with the arm still selected.
func (e *MapEditor) goalAt(p image.Point) {
	s, ok := e.script()
	if !ok || e.report.sel >= len(e.report.goals) {
		return
	}
	sel := e.report.sel
	g := e.report.goals[sel]
	for _, id := range e.hitsAt(p) {
		r := e.doc.Records[id]
		if r.Kind != g.Pick {
			continue
		}
		next, err := s.SetGoalTarget(sel, r.Kind, r.Index)
		if e.adoptEdit(next, err) == nil && next != nil {
			e.openGoals(s, sel)
		}
		return
	}
	e.message = "Click a " + map[string]string{"Unit": "unit", "Structure": "structure"}[g.Pick] + " on the map."
}

// dropMarkers are the cells the script drops the party on.
func (e *MapEditor) dropMarkers() []image.Point {
	if e.doc == nil {
		return nil
	}
	return e.doc.Drops
}
