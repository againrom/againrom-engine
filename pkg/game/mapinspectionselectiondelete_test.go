package game

import (
	"bytes"
	"image"
	"testing"

	"againrom/pkg/mapedit"
	"againrom/pkg/ui"
)

// A selection delete takes a record as the single delete does: a script
// node a trigger still names is refused and the map keeps its bytes.
func TestSelectionDeleteKeepsAScriptNamedRecord(t *testing.T) {
	e, _ := openGoalMap(t, formMap(t, 2))
	s := formHost(e)
	before := bytes.Clone(s.model.Bytes())
	view, err := s.model.Map()
	if err != nil {
		t.Fatal(err)
	}
	sc, err := view.Script()
	if err != nil {
		t.Fatal(err)
	}
	index := -1
	for i, n := range sc.Actions {
		if n.ID == 1 {
			index = i
		}
	}
	if index < 0 || !scriptReferenced(sc, mapedit.KindAction, 1) {
		t.Fatal("fixture has no referenced action")
	}
	if _, err := s.Delete(ui.KindAction, index); err == nil {
		t.Fatal("the single delete took a referenced action")
	}
	out, err := s.DeleteRecords([]ui.RecordRef{{Kind: ui.KindAction, Index: index}})
	if err != nil {
		t.Fatal(err)
	}
	if len(out.Refusals) != 1 || out.Doc != nil || !bytes.Equal(before, s.model.Bytes()) || s.CanUndo() {
		t.Fatalf("selection delete changed the map or did not refuse: %+v", out)
	}
}

// A selection never takes an enchantment record, so no part of a spell set is
// deleted by it; the single delete takes the whole set as one undo step.
func TestSelectionDeleteLeavesASpellSetWhole(t *testing.T) {
	e := cellSpellFixtureWithPower(t, 40)
	e.Layout(1920, 1080)
	s := e.Document().Edits.(*mapInspectionEdits)
	before := s.model.Bytes()
	palettePress(t, e, ui.ControlQuery{Role: ui.ControlBar, Key: "cellspells"})
	palettePress(t, e, ui.ControlQuery{Role: ui.ControlListRow, N: 0})
	if err := e.Key("traparea"); err != nil {
		t.Fatal(err)
	}
	box := image.Rect(10, 10, 13, 12)
	dragCell(e, box.Min, box.Max.Sub(image.Pt(1, 1)))
	added := s.model.Bytes()
	if len(s.model.SpellSet(5)) != 6 {
		t.Fatal("stroke membership missing")
	}
	out, err := s.DeleteRecords([]ui.RecordRef{{Kind: ui.KindEnchantment, Index: 5}})
	if err != nil {
		t.Fatal(err)
	}
	if len(out.Refusals) != 1 || out.Doc != nil || !bytes.Equal(added, s.model.Bytes()) || len(s.model.SpellSet(5)) != 6 {
		t.Fatalf("selection delete took part of the set: %+v", out)
	}
	if _, err := s.Delete(ui.KindEnchantment, 5); err != nil || !bytes.Equal(before, s.model.Bytes()) {
		t.Fatal("the single delete did not take the whole set", err)
	}
	if _, err := s.Undo(); err != nil || !bytes.Equal(added, s.model.Bytes()) {
		t.Fatal("one undo did not restore the whole set", err)
	}
}
