package ui

import (
	"testing"

	"againrom/pkg/scriptprofile"
)

// On a second-game map the trigger wizard opens with every row kept: the
// first game's ready items do not read the second game's operations. A new
// row is refused in words, a row moves through moveRow and Apply writes the
// kept rows in their new order.
func TestSecondGameTriggerWizardKeepsEveryRowAndWritesNoNewNode(t *testing.T) {
	e, host := wizardChoiceEditor(t)
	if r := e.obj.wiz.rows; len(r) != 4 || r[2].item < 0 {
		t.Fatalf("the first game's wizard does not read its ready action: %+v", r)
	}
	e.doc.Script.Limits = scriptprofile.Second
	v, ok := e.scriptPopup()
	if !ok || v.trigger == nil {
		t.Fatal("no trigger popup")
	}
	e.loadWizard(v)
	w := e.obj.wiz
	for r, row := range w.rows {
		if row.item != -1 {
			t.Fatalf("second-game row %d reads as first-game item %d", r, row.item)
		}
	}
	e.wizardRowButton(actAddAct, 0, false)
	if want := e.word("editor.wizard.norows"); e.obj.note != want || len(w.rows) != 4 {
		t.Fatalf("a new action row: note %q, %d rows; want %q, 4 rows", e.obj.note, len(w.rows), want)
	}
	if to := w.moveRow(2, 1); to != 3 || w.rows[2].act != 12 || w.rows[3].act != 11 {
		t.Fatalf("moveRow took the action to %d: %+v", to, w.rows)
	}
	e.applyWizard(v)
	if len(host.plans) != 1 {
		t.Fatalf("%d writes, note %q", len(host.plans), e.obj.note)
	}
	p := host.plans[0]
	if len(p.Actions) != 2 || p.Actions[0] != (WizardNode{ID: 12, Keep: true}) || p.Actions[1] != (WizardNode{ID: 11, Keep: true}) || len(p.Pairs) != 2 || !p.Pairs[0].Left.Keep {
		t.Fatalf("the write is %+v", p)
	}
}
