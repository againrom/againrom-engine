package ui

import "testing"

// scenarioOpensNothing fails when the editor selected a record, opened a
// popup or highlighted a trigger since the selection was cleared.
func scenarioOpensNothing(t *testing.T, e *MapEditor, route string) {
	t.Helper()
	if e.Selected() != -1 {
		t.Fatalf("%s selected record %d", route, e.Selected())
	}
	if _, _, ok := e.objectSpec(); ok {
		t.Fatalf("%s opened a popup", route)
	}
	if e.selectedTrigger() >= 0 {
		t.Fatalf("%s opened trigger %d", route, e.selectedTrigger())
	}
}

// TestScenarioBranchClickOpensNoTrigger: a click on the tree's Scenario
// branch, docked or floating, opens or closes the branch and selects
// nothing; a click on a trigger row still opens that trigger.
func TestScenarioBranchClickOpensNoTrigger(t *testing.T) {
	for _, docked := range []bool{true, false} {
		e := scriptsListEditor(t)
		e.selected = -1
		e.showPanel(panelLeft, true)
		if !docked {
			e.undockPanel(panelLeft)
		}
		e.treeSpec()
		branch := e.TreeBranchRow("scripts")
		if branch < 0 {
			t.Fatal("no Scenario branch")
		}
		clickTreeRow(t, e, branch)
		e.treeSpec()
		scenarioOpensNothing(t, e, "the branch click")
		if !e.tree.closed["scripts"] {
			t.Fatalf("docked %v: the branch click did not close the open branch", docked)
		}
		clickTreeRow(t, e, e.TreeBranchRow("scripts"))
		e.treeSpec()
		scenarioOpensNothing(t, e, "the second branch click")
		if e.tree.closed["scripts"] {
			t.Fatalf("docked %v: the second click did not reopen the branch", docked)
		}
		clickTreeRow(t, e, e.TreeMemberRow("scripts", 0))
		if e.Selected() != 2 || e.selectedTrigger() != 0 {
			t.Fatalf("docked %v: the trigger row selected %d, trigger %d", docked, e.Selected(), e.selectedTrigger())
		}
	}
}

// TestScenarioEntryOpensNoTrigger: the bar's Scenario entry and its key
// show the Scenario palette and open no trigger; the palette's trigger row
// still opens it.
func TestScenarioEntryOpensNoTrigger(t *testing.T) {
	for _, route := range []string{"bar", "key"} {
		e := scriptsListEditor(t)
		e.selected = -1
		if route == "bar" {
			chromeClick(e, barPoint(t, e, "scripts"))
		} else {
			_ = e.Key("scripts")
		}
		if e.filter != editorScriptsFilter || e.listKind() != "records" {
			t.Fatalf("%s did not show the Scenario palette", route)
		}
		scenarioOpensNothing(t, e, route)
		rows, kinds, _ := e.scenarioRows()
		n := -1
		for i := range rows {
			if kinds[i].member && kinds[i].record == 2 {
				n = i
			}
		}
		if n < 0 {
			t.Fatal("no trigger row in the Scenario palette")
		}
		e.listChoose(n, true)
		if e.Selected() != 2 || e.selectedTrigger() != 0 {
			t.Fatalf("%s: the palette's trigger row selected %d", route, e.Selected())
		}
	}
}

// TestScenarioClickClosesTheOpenTrigger: with a trigger's popup open from its
// tree row, a click on the Scenario branch, the bar's Scenario entry or its
// key leaves that trigger and opens no popup; the trigger's row opens it again.
func TestScenarioClickClosesTheOpenTrigger(t *testing.T) {
	for _, route := range []string{"branch", "bar", "key"} {
		e := scriptsListEditor(t)
		e.selected = -1
		e.showPanel(panelLeft, true)
		e.treeSpec()
		clickTreeRow(t, e, e.TreeMemberRow("scripts", 0))
		if e.Selected() != 2 || e.selectedTrigger() != 0 {
			t.Fatalf("%s: the trigger row selected %d", route, e.Selected())
		}
		switch route {
		case "branch":
			clickTreeRow(t, e, e.TreeBranchRow("scripts"))
		case "bar":
			chromeClick(e, barPoint(t, e, "scripts"))
		default:
			_ = e.Key("scripts")
		}
		e.treeSpec()
		scenarioOpensNothing(t, e, route)
		if route == "branch" {
			clickTreeRow(t, e, e.TreeBranchRow("scripts"))
			e.treeSpec()
		}
		clickTreeRow(t, e, e.TreeMemberRow("scripts", 0))
		if e.Selected() != 2 || e.selectedTrigger() != 0 {
			t.Fatalf("%s: the trigger row did not open the trigger again", route)
		}
	}
}

// TestScenarioBranchClickLeavesAnotherRecord: the Scenario branch click
// deselects a record of another kind as the bar's entry does.
func TestScenarioBranchClickLeavesAnotherRecord(t *testing.T) {
	e := scriptsListEditor(t)
	e.showPanel(panelLeft, true)
	e.Select(0, false)
	e.treeSpec()
	clickTreeRow(t, e, e.TreeBranchRow("scripts"))
	e.treeSpec()
	scenarioOpensNothing(t, e, "the branch click")
}
