package ui

import (
	"slices"
	"testing"
)

// The trigger window's add-action press focuses the new row.
func TestTriggerWindowAddActionFocusesTheNewRow(t *testing.T) {
	e, _ := wizardChoiceEditor(t)
	e.Layout(800, 600)
	s, _, ok := e.objectSpec()
	if !ok {
		t.Fatal("the wizard builds no popup")
	}
	row, k := -1, -1
	for n := range s.Rows {
		if i := slices.Index(e.obj.rowActs[n], actAddAct); i >= 0 {
			row, k = n, i
			break
		}
	}
	if row < 0 {
		t.Fatal("no row of the trigger window adds an action")
	}
	before := e.obj.wiz.kindRows(false)
	e.scriptRowButton(row, k)
	if got := e.obj.wiz.kindRows(false); got != before+1 {
		t.Fatalf("the press left %d action rows, want %d", got, before+1)
	}
	last := -1
	for r, w := range e.obj.wiz.rows {
		if !w.condition {
			last = r
		}
	}
	want := e.dialogFieldAt(e.obj.wiz.wizardKey(last, "item"))
	if want < 0 || e.obj.focus != want {
		t.Fatalf("the focus is on field %d, want field %d of the new action row", e.obj.focus, want)
	}
}

// The trigger window moves a focused action row with Alt+Up and Alt+Down,
// the focus going with the row, as the dialog window does.
func TestTriggerWindowMovesARowWithAltArrows(t *testing.T) {
	e, _ := wizardChoiceEditor(t)
	e.Layout(800, 600)
	var acts []int
	for r, w := range e.obj.wiz.rows {
		if !w.condition {
			acts = append(acts, r)
		}
	}
	if len(acts) != 2 {
		t.Fatalf("the trigger has %d action rows, want 2", len(acts))
	}
	first, second := e.obj.wiz.rows[acts[0]].origItem, e.obj.wiz.rows[acts[1]].origItem
	e.focusObjectField(e.dialogFieldAt(e.obj.wiz.wizardKey(acts[1], "item")))
	_ = e.Key("cardup")
	if got := e.obj.wiz.rows[acts[0]].origItem; got != second || e.obj.wiz.rows[acts[1]].origItem != first {
		t.Fatalf("Alt+Up left the action rows %d, %d", got, e.obj.wiz.rows[acts[1]].origItem)
	}
	if want := e.dialogFieldAt(e.obj.wiz.wizardKey(acts[0], "item")); want < 0 || e.obj.focus != want {
		t.Fatalf("the focus is on field %d, want field %d of the moved row", e.obj.focus, want)
	}
	_ = e.Key("cardup")
	if e.obj.wiz.rows[acts[0]].origItem != second {
		t.Fatal("Alt+Up on the first action row moved it")
	}
	_ = e.Key("carddown")
	if e.obj.wiz.rows[acts[0]].origItem != first {
		t.Fatal("Alt+Down did not move the row back")
	}
}
