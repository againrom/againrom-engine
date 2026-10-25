package game

import (
	"path/filepath"
	"testing"

	"againrom/internal/firstgame"
	"againrom/pkg/ui"
)

// chooseRow opens field n's drop-down and clicks choice row; a negative row
// counts from the end.
func chooseRow(t *testing.T, e *ui.MapEditor, n, row int) {
	t.Helper()
	r, ok := e.FieldRect(n)
	messageClick(t, e, r, ok)
	choices := e.DropdownChoices()
	if row < 0 {
		row += len(choices)
	}
	if row < 0 || row >= len(choices) {
		t.Fatalf("field %d offers %d choices, want row %d", n, len(choices), row)
	}
	e.ScrollDropdown(-len(choices))
	for range len(choices) + 10 {
		if r, ok := e.DropdownRowRect(row); ok {
			messageClick(t, e, r, ok)
			return
		}
		e.ScrollDropdown(1)
	}
	t.Fatalf("choice row %d of field %d never showed", row, n)
}

// A trigger and a message added to an installed map through the wizard leave
// the report without an error.
func TestReleaseInstalledMapNewTriggerAndMessageReportsNoError(t *testing.T) {
	firstgame.SkipOnSecondGame(t)
	x := installedInspector(t)
	e := x.Editor()
	e.Layout(1280, 800)
	if err := e.Open("scenario/10.alm"); err != nil {
		t.Fatal(err)
	}
	s := e.Document().Edits.(*mapInspectionEdits)
	pages := len(s.workspace.Pages)
	wizNewTrigger(t, e, "Greeting")
	wizPress(t, e, "+action", 0)
	chooseRow(t, e, wizField(t, e, "a0.item"), 3)
	wizApply(t, e)
	chooseRow(t, e, wizField(t, e, "a0.p0"), -1)
	e.TextInput("A new message")
	messageButton(t, e, 0)
	sc := wizScript(t, e.Document().SourceBytes)
	last := sc.Triggers[len(sc.Triggers)-1]
	if last.Name != "Greeting" || len(last.Acts) == 0 || wizNode(t, sc.Actions, last.Acts[0]).Opcode != 2 || len(s.workspace.Pages) != pages+1 {
		t.Fatalf("the route did not add the trigger, its message action and one page: %+v, pages %d to %d", last, pages, len(s.workspace.Pages))
	}
	rows := s.Report()
	for _, f := range rows {
		if f.Error {
			t.Fatalf("error after a new trigger and message: %s", f.Line())
		}
	}
	if _, err := s.SaveAs(filepath.Join(t.TempDir(), "greeting.alm")); err != nil {
		t.Fatalf("Save As refused: %v", err)
	}
	t.Logf("lang=%s findings=%d, no error", x.pageLanguage(), len(rows))
}
