package ui

import (
	"image"
	"reflect"
	"testing"
)

func TestArmOpenAndBackPreservePanel(t *testing.T) {
	for _, arm := range []string{"Win", "Lose", "Protect", "Drop"} {
		t.Run(arm, func(t *testing.T) {
			e := scriptsListEditor(t)
			g := InspectionGoal{Arm: arm, Kind: "Condition", Index: 0, OpenKind: "Trigger", OpenIndex: 0, Pick: "Unit"}
			if arm == "Protect" {
				g.OpenKind, g.OpenIndex = "Condition", 0
			}
			if arm == "Drop" {
				g.Kind, g.OpenKind, g.OpenIndex, g.Pick = "Action", "Action", 1, ""
			}
			rows, goals := make([]InspectionFinding, 30), make([]InspectionGoal, 30)
			for i := range rows {
				rows[i] = InspectionFinding{Kind: g.Kind, Index: g.Index}
				goals[i] = g
			}
			e.openReport(rows, "arms", goals)
			e.selectFinding(25)
			e.listSpec()
			e.model.Scroll(100)
			top, _ := e.model.Visible()
			r, ok := e.GoalOpenRect(25)
			if !ok {
				t.Fatal("Open link missing")
			}
			e.chromePress(popList, r.Min.Add(image.Pt(2, 2)), false)
			want := map[string]int{"Win": 2, "Lose": 2, "Protect": 1, "Drop": 5}[arm]
			if e.selected != want || e.objectHidden || len(e.objectBack) != 1 {
				t.Fatalf("Open = %d hidden=%v stack=%d", e.selected, e.objectHidden, len(e.objectBack))
			}
			e.backObject()
			after, _ := e.model.Visible()
			rowRecord := 1
			if arm == "Drop" {
				rowRecord = 0
			}
			if !e.report.open || e.report.sel != 25 || e.objectHidden || after != top || e.selected != rowRecord {
				t.Fatalf("Back lost panel: sel=%d top=%d want=%d", e.report.sel, after, top)
			}
			_ = e.Key("field")
			if e.selected != want || e.objectHidden {
				t.Fatal("E did not open the same script")
			}
		})
	}
}

func TestArmBackClearsTriggerAreaHighlight(t *testing.T) {
	e := scriptsListEditor(t)
	e.doc.Script.Areas = []InspectionArea{{Trigger: 0, Rect: image.Rect(2, 2, 5, 5)}}
	g := InspectionGoal{Kind: "Condition", Index: 0, OpenKind: "Trigger", OpenIndex: 0}
	e.openReport([]InspectionFinding{{Kind: g.Kind, Index: g.Index}}, "arms", []InspectionGoal{g})
	e.selectFinding(0)
	before := e.areaLines()
	e.openGoal(0)
	if e.selectedTrigger() != 0 {
		t.Fatal("opened trigger is not highlighted")
	}
	e.backObject()
	if e.selectedTrigger() != -1 || !reflect.DeepEqual(e.areaLines(), before) {
		t.Fatal("Back leaves trigger areas highlighted behind G")
	}
}

func TestArmBackRestoresPopupState(t *testing.T) {
	for _, hidden := range []bool{false, true} {
		e := scriptsListEditor(t)
		g := InspectionGoal{Kind: "Condition", Index: 0, OpenKind: "Trigger", OpenIndex: 0}
		e.openReport([]InspectionFinding{{Kind: g.Kind, Index: g.Index}}, "arms", []InspectionGoal{g})
		e.selectFinding(0)
		e.objectHidden, e.objectTop = hidden, 3
		e.openGoal(0)
		e.backObject()
		if e.objectHidden != hidden || e.objectTop != 3 || e.selected != 1 {
			t.Fatalf("Back lost popup state: hidden=%v want=%v top=%d selected=%d", e.objectHidden, hidden, e.objectTop, e.selected)
		}
	}
}

type scrollGoalsHost struct {
	*modesFull
	goals []InspectionGoal
}

func (s *scrollGoalsHost) Goals() []InspectionGoal { return s.goals }

func TestArmBackRestoresManualScrollAfterPanelRebuild(t *testing.T) {
	e := scriptsListEditor(t)
	h := &scrollGoalsHost{modesFull: e.doc.Edits.(*modesFull)}
	for i := 0; i < 60; i++ {
		h.goals = append(h.goals, InspectionGoal{Kind: "Condition", Index: 0, OpenKind: "Trigger", OpenIndex: 0})
	}
	e.doc.Edits = h
	e.openGoals(h, 25)
	e.listSpec()
	top, _ := e.model.Visible()
	e.model.Scroll(25 - top)
	before, ok := e.GoalOpenRect(25)
	if !ok {
		t.Fatal("selected row must remain visible")
	}
	e.openGoal(25)
	e.leave(leaveReport)
	e.backObject()
	after, ok := e.GoalOpenRect(25)
	if top, _ := e.model.Visible(); !ok || before != after || top != 25 || e.ReportSelected() != 25 {
		t.Fatalf("Back lost manual scroll: before=%v after=%v top=%d sel=%d", before, after, top, e.ReportSelected())
	}
}

func TestArmOpenLossControls(t *testing.T) {
	for _, arm := range []string{"Win", "Lose", "Protect"} {
		t.Run(arm, func(t *testing.T) {
			e := scriptsListEditor(t)
			e.openReport([]InspectionFinding{{}}, "arms", []InspectionGoal{{Arm: arm, OpenKind: "Trigger", OpenIndex: 0}})
			if e.goalRecord(0) != 2 {
				t.Fatal("correct link rejected")
			}
			for _, broken := range []InspectionGoal{{Arm: arm}, {Arm: arm, OpenKind: "Trigger", OpenIndex: 9}} {
				e.report.goals[0] = broken
				if e.goalRecord(0) == 2 {
					t.Fatal("lost or wrong link accepted")
				}
			}
		})
	}
}
