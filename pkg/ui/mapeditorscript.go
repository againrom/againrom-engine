package ui

import (
	"fmt"
	"image"
)

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
	if key == "drop" {
		e.leave(leaveDrag | leaveReport)
		e.armPick(pickDrop)
		return
	}
	if len(e.report.goals) > 0 {
		e.leave(leaveDrag | leaveReport)
		return
	}
	e.leave(leaveDrag | leavePick | leavePrompt)
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
	e.openReport(rows, fmt.Sprintf("Script arms: %d", len(goals)), goals)
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
	e.armPick(pickGoal)
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
