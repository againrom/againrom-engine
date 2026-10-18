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

func (e *MapEditor) openGoals(s InspectionScript, sel int) {
	goals := s.Goals()
	if drops, ok := s.(interface{ DropGoals() []InspectionGoal }); ok {
		goals = append(goals, drops.DropGoals()...)
	}
	rows := make([]InspectionFinding, len(goals))
	for i, g := range goals {
		rows[i] = InspectionFinding{Rule: g.Arm, Kind: g.Kind, Index: g.Index, Cell: g.Cell, HasCell: g.HasCell, Text: g.Text}
	}
	e.openReport(rows, fmt.Sprintf(e.word("editor.arms.count"), len(goals)), goals)
	if len(goals) > 0 {
		e.selectFinding(max(0, min(sel, len(goals)-1)))
	}
}

// goalLabel is a script arm as one list row.
func goalLabel(g InspectionGoal) string { return g.Arm + ": " + g.Text }

func (e *MapEditor) goalText(g InspectionGoal) string {
	arm := map[string]string{"Win": e.word("editor.arms.Win"), "Lose": e.word("editor.arms.Lose"), "Protect": e.word("editor.arms.Protect")}[g.Arm]
	if g.Arm == "Drop" {
		return fmt.Sprintf(e.word("editor.arms.dropcell"), g.Cell.X, g.Cell.Y)
	}
	if g.OpenKind != "" {
		format := e.word("editor.arms.Condition")
		if g.OpenKind == "Trigger" {
			format = e.word("editor.arms.Trigger")
		}
		label := fmt.Sprintf(format, g.OpenIndex+1)
		if g.OpenKind == "Trigger" && e.doc.Script != nil && g.OpenIndex < len(e.doc.Script.Triggers) {
			label += "  " + e.editorHostText(e.doc.Script.Triggers[g.OpenIndex].Name)
		}
		if g.HasCell {
			label += fmt.Sprintf(e.word("editor.arms.at"), g.Cell.X, g.Cell.Y)
		}
		return arm + ": " + label
	}
	key := map[string]string{
		"add: the death of a unit wins":                       e.word("editor.arms.add.winunit"),
		"add: a group wiped out wins":                         e.word("editor.arms.add.wingroup"),
		"add: a structure destroyed wins":                     e.word("editor.arms.add.winstructure"),
		"add: the death of a party unit loses":                e.word("editor.arms.add.loseunit"),
		"add a check that loses the mission when a unit dies": e.word("editor.arms.add.protect"),
	}[g.Text]
	if key != "" {
		return arm + ": " + key
	}
	return goalLabel(g)
}

func (e *MapEditor) goalButton(l popupLayout, row int) image.Rectangle {
	top, count := e.model.Visible()
	if row < top || row >= top+count || e.goalRecord(row) < 0 {
		return image.Rectangle{}
	}
	r := l.List.Row(row - top)
	return image.Rect(l.Aside.Min.X, r.Min.Y, l.Aside.Max.X, r.Max.Y)
}

func (e *MapEditor) drawGoalLinks(pic *image.RGBA, s popupSpec) {
	if e.listKind() != "report" || len(e.report.goals) == 0 {
		return
	}
	l := layoutPopup(s, e.font, e.bodyFont())
	for row := range e.report.goals {
		if r := e.goalButton(l, row); !r.Empty() {
			drawPushButton(pic, e.bodyFont(), pushButton{Rect: r, Label: e.word("editor.button.open"), Literal: true,
				LabelRaise: labelRaise(e.bodyFont()), Hover: e.pointerOK && e.pointer.In(r)})
		}
	}
}

// GoalOpenRect is the visible arm or drop row's Open button.
func (e *MapEditor) GoalOpenRect(row int) (image.Rectangle, bool) {
	if e.listKind() != "report" || len(e.report.goals) == 0 {
		return image.Rectangle{}, false
	}
	s, _ := e.listSpec()
	r := e.goalButton(layoutPopup(s, e.font, e.bodyFont()), row)
	return r, !r.Empty()
}

func (e *MapEditor) goalRecord(row int) int {
	if row < 0 || row >= len(e.report.goals) {
		return -1
	}
	g := e.report.goals[row]
	for i, r := range e.doc.Records {
		if g.OpenKind != "" && r.Kind == g.OpenKind && r.Index == g.OpenIndex {
			return i
		}
	}
	return -1
}

func (e *MapEditor) openGoal(row int) {
	record := e.goalRecord(row)
	if record < 0 || !e.objectMayLeave() {
		return
	}
	if row != e.report.sel || len(e.objectBack) > 0 {
		e.selectFinding(row)
	}
	p := e.objectPlace()
	p.goals, p.goalRow, p.hidden = true, row, e.objectHidden
	e.listSpec()
	p.listTop, _ = e.model.Visible()
	e.openObjectRecord(record, false)
	if len(e.objectBack) > 0 {
		e.objectBack[len(e.objectBack)-1] = p
	}
}

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
	if e.report.goals[e.report.sel].Pick == "" {
		e.openGoal(e.report.sel)
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
