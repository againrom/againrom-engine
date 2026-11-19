package game

import (
	"strings"
	"testing"

	"againrom/pkg/ui"
)

// checksWindow opens the dialogue window of the mill on a map that names
// variables 10 and 12, with node start and node thanks and, on start, replies
// whose texts are English only.
func checksWindow(t *testing.T, replies int) (*ui.MapEditor, *mapInspectionEdits) {
	t.Helper()
	e, _ := openGoalMap(t, millMap(t))
	s := formHost(e)
	if _, err := s.ExtendScript(); err != nil {
		t.Fatal(err)
	}
	startDialogueRow(t, e, "mill")
	dlgType(t, e, "d.name", "start")
	dlgType(t, e, "d.text.en", "Hello")
	dlgPress(t, e, "d.node", "addnode")
	dlgType(t, e, "d.name", "thanks")
	dlgType(t, e, "d.text.en", "Thanks")
	dlgChoose(t, e, "d.node", "start")
	for i := 0; i < replies; i++ {
		dlgPress(t, e, "d.reply", "addreply")
		dlgType(t, e, "d.r.en", "Reply")
	}
	return e, s
}

func refusedWith(t *testing.T, e *ui.MapEditor, s *mapInspectionEdits, want string) {
	t.Helper()
	dlgApply(t, e)
	if lines := trigLines(e); !strings.Contains(lines, want) {
		t.Fatalf("Apply did not say %q:\n%s", want, lines)
	}
	if len(s.DialogueNames()) != 0 {
		t.Fatalf("a refused dialogue was written: %v", s.DialogueNames())
	}
}

func TestDialogueWindowRefusesAFifthReply(t *testing.T) {
	e, _ := checksWindow(t, 4)
	dlgPress(t, e, "d.reply", "addreply")
	if lines := trigLines(e); !strings.Contains(lines, "more than four replies") {
		t.Fatalf("the fifth reply was not refused in words:\n%s", lines)
	}
}

func TestDialogueWindowRefusesAMissingNextNode(t *testing.T) {
	e, s := checksWindow(t, 1)
	dlgChoose(t, e, "d.next", "thanks")
	dlgChoose(t, e, "d.node", "thanks")
	dlgPress(t, e, "d.node", "removenode")
	dlgChoose(t, e, "d.node", "start")
	refusedWith(t, e, s, "does not exist")
}

func TestDialogueWindowRefusesAnEmptyAndATwiceUsedNodeName(t *testing.T) {
	e, s := checksWindow(t, 1)
	dlgType(t, e, "d.name", "")
	refusedWith(t, e, s, "has no name")
	dlgType(t, e, "d.name", "thanks")
	refusedWith(t, e, s, "Two nodes are named thanks")
}

func TestDialogueWindowRefusesAnIncompleteOutcome(t *testing.T) {
	e, s := checksWindow(t, 1)
	dlgAdd(t, e, true)
	dlgChoose(t, e, "a0.item", "Set a variable")
	dlgType(t, e, "a0.p1", "")
	dlgApply(t, e)
	if !e.DialogueOpen() || len(s.DialogueNames()) != 0 {
		t.Fatalf("an outcome with no value was written:\n%s", trigLines(e))
	}
}

func TestDialogueWindowTakesATextInOneLanguageAndRefusesAUsedDelete(t *testing.T) {
	e, s := checksWindow(t, 2)
	dlgApply(t, e)
	if got := s.DialogueNames(); len(got) != 1 || got[0] != "mill" {
		t.Fatalf("a text in one language was refused:\n%s", trigLines(e))
	}
	r, ok := e.FindControl(ui.ControlQuery{Role: ui.ControlPopupButton, N: 2})
	trigClick(t, e, r, ok, "Delete")
	if !strings.Contains(trigLines(e), "Change those triggers first") || len(s.DialogueNames()) != 1 {
		t.Fatalf("a dialogue a trigger starts was deleted:\n%s", trigLines(e))
	}
}
