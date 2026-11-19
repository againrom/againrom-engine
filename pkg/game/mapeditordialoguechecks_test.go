package game

import (
	"strings"
	"testing"

	"againrom/pkg/ui"
)

// checksWindow opens the dialog window on a map that names variables 10 and
// 12, with the lines Hello and Thanks and, on Hello, a reply choice of the
// given number of replies whose texts are English only.
func checksWindow(t *testing.T, replies int) (*ui.MapEditor, *mapInspectionEdits) {
	t.Helper()
	e, _ := openGoalMap(t, millMap(t))
	s := formHost(e)
	if _, err := s.ExtendScript(); err != nil {
		t.Fatal(err)
	}
	startDialogueRow(t, e)
	dlgType(t, e, ui.DialogueLineText, "Hello")
	dlgAddLine(t, e, "Thanks")
	dlgSelect(t, e, 1)
	dlgChoose(t, e, "d.after", "Reply choice")
	dlgType(t, e, "d.rt", "Reply")
	for i := 1; i < replies; i++ {
		dlgPress(t, e, "addreply")
		dlgType(t, e, "d.rt", "Reply")
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

func TestDialogueWindowOffersNoFifthReply(t *testing.T) {
	e, _ := checksWindow(t, 3)
	if _, ok := e.DialogueButton("addreply"); !ok {
		t.Fatalf("a line with three replies offers no fourth:\n%s", trigLines(e))
	}
	dlgPress(t, e, "addreply")
	dlgType(t, e, "d.rt", "Reply")
	if _, ok := e.DialogueButton("addreply"); ok {
		t.Fatalf("a line with four replies offers a fifth:\n%s", trigLines(e))
	}
}

func TestDialogueWindowRefusesAReplyToARemovedLine(t *testing.T) {
	e, s := checksWindow(t, 1)
	dlgChoose(t, e, "d.rnext", "Thanks")
	dlgSelect(t, e, 2)
	dlgPress(t, e, "removerow")
	dlgSelect(t, e, 1)
	refusedWith(t, e, s, "leads to a line that was removed")
}

func TestDialogueWindowRefusesAReplyChoiceWithNoReply(t *testing.T) {
	e, s := checksWindow(t, 1)
	dlgPress(t, e, "removereply")
	refusedWith(t, e, s, "has no reply")
}

func TestDialogueWindowRefusesAnIncompleteOutcome(t *testing.T) {
	e, s := checksWindow(t, 1)
	dlgAdd(t, e, true)
	dlgChoose(t, e, "a0.item", "Set a variable")
	dlgType(t, e, "a0.p1", "")
	dlgApply(t, e)
	if len(s.DialogueNames()) != 0 {
		t.Fatalf("an outcome with no value was written:\n%s", trigLines(e))
	}
}

func TestDialogueWindowTakesATextInOneLanguageAndRefusesAUsedDelete(t *testing.T) {
	e, s := checksWindow(t, 2)
	dlgApply(t, e)
	got := s.DialogueNames()
	if len(got) != 1 {
		t.Fatalf("a text in one language was refused:\n%s", trigLines(e))
	}
	r, ok := e.FindControl(ui.ControlQuery{Role: ui.ControlPopupButton, N: 2})
	trigClick(t, e, r, ok, "Delete")
	if !strings.Contains(trigLines(e), "Change those triggers first") || len(s.DialogueNames()) != 1 {
		t.Fatalf("a dialogue a trigger starts was deleted:\n%s", trigLines(e))
	}
}

// The When not available choice is written as the reply's own flag, which the
// play tests of the offered replies read: hidden leaves the reply out and
// disabled greys it.
func TestDialogueReplyConditionIsWrittenHiddenOrDisabledAsSet(t *testing.T) {
	for _, choice := range []string{"Hidden", "Disabled"} {
		t.Run(choice, func(t *testing.T) {
			e, s := checksWindow(t, 1)
			dlgChoose(t, e, "d.runavail", choice)
			dlgAdd(t, e, false)
			dlgChoose(t, e, "c0.item", "Variable compared with a number")
			dlgChoose(t, e, "c0.p0", "12")
			dlgApply(t, e)
			names := s.DialogueNames()
			if len(names) != 1 {
				t.Fatalf("the dialogue was refused:\n%s", trigLines(e))
			}
			d, err := s.ReadDialogue(names[0])
			if err != nil || len(d.Nodes) == 0 || len(d.Nodes[0].Replies) != 1 {
				t.Fatalf("the file holds %+v (%v)", d, err)
			}
			if r := d.Nodes[0].Replies[0]; len(r.Pairs) != 1 || r.Disabled != (choice == "Disabled") {
				t.Fatalf("a %s reply was written %+v", choice, r)
			}
		})
	}
}
