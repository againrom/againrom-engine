package ui

import (
	"slices"
	"testing"
)

func scenarioWithTrigger(t *testing.T) *MapEditor {
	t.Helper()
	e := scriptsListEditor(t)
	e.Layout(listWindowScreen.X, listWindowScreen.Y)
	if err := e.Key("scripts"); err != nil {
		t.Fatal(err)
	}
	_, rows, _ := e.scenarioRows()
	row := slices.IndexFunc(rows, func(r treeRow) bool { return r.member && r.record >= 0 && e.doc.Records[r.record].Kind == "Trigger" })
	box, ok := e.FindControl(ControlQuery{Role: ControlListRow, N: row})
	if row < 0 || !ok {
		t.Fatal("no trigger row in Scenario")
	}
	chromeClick(e, center(box))
	if _, _, ok := e.objectSpec(); !ok {
		t.Fatal("the trigger window did not open")
	}
	return e
}

func TestScenarioCloseKeepsTriggerWindow(t *testing.T) {
	e := scenarioWithTrigger(t)
	sel := e.Selected()
	e.cancelPopup(popList)
	if e.listShown {
		t.Fatal("the Scenario list stands")
	}
	if _, _, ok := e.objectSpec(); !ok || e.Selected() != sel {
		t.Errorf("trigger window gone after closing the list: ok %v, selected %d, want %d", ok, e.Selected(), sel)
	}
	for _, how := range []string{"scripts", "objects"} {
		t.Run(how, func(t *testing.T) {
			e := scenarioWithTrigger(t)
			e.barAct(how)
			if _, _, ok := e.objectSpec(); ok {
				t.Errorf("trigger window stands after the %s entry", how)
			}
		})
	}
}

func TestListCloseLeavesEveryChild(t *testing.T) {
	for _, child := range []editorPopupID{popObject, popPrompt} {
		e := scriptsListEditor(t)
		e.Layout(listWindowScreen.X, listWindowScreen.Y)
		stack := e.popupStack()
		if !slices.Contains(stack[:], child) {
			t.Fatalf("popup %d is not in the stack", child)
		}
		if err := e.Key("scripts"); err != nil {
			t.Fatal(err)
		}
		switch child {
		case popObject:
			e.Select(2, false)
		case popPrompt:
			e.openPrompt(editorPrompt{kind: promptLeave})
		}
		if _, ok := e.movableSpec(child); !ok {
			t.Fatalf("child %d did not open", child)
		}
		e.cancelPopup(popList)
		if _, ok := e.movableSpec(child); !ok {
			t.Errorf("child %d closed with the list", child)
		}
	}
}
