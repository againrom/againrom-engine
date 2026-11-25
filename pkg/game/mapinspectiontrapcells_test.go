package game

import (
	"image"
	"slices"
	"testing"

	"againrom/pkg/mapedit"
	"againrom/pkg/ui"
)

// A first-game teleport trap lists its own cell as Source and its
// destination as Target, each an X field with its Y field to pick on the map;
// the unread source bytes are not listed, and a trap of another spell lists
// the plain fields. Applying the picked cells moves the record and sets the
// target bytes.
func TestFirstGameTeleportTrapListsSourceAndTargetCells(t *testing.T) {
	e := spellMap(t)
	s := e.Document().Edits.(*mapInspectionEdits)
	cell := int64(mapedit.SpellConstructor(0))
	for _, spell := range []int64{firstCellMove, 13} {
		if _, err := s.AddRecord(ui.InspectionNewRecord{Kind: ui.KindEnchantment, Cell: image.Pt(12, 20), HasCell: true, Values: []ui.InspectionFieldValue{
			{Key: "template", Value: cell}, {Key: "spell", Value: spell}, {Key: "power", Value: 9}}}); err != nil {
			t.Fatal(err)
		}
	}
	by := func(fields []ui.InspectionField, key string) (ui.InspectionField, bool) {
		i := slices.IndexFunc(fields, func(f ui.InspectionField) bool { return f.Key == key })
		if i < 0 {
			return ui.InspectionField{}, false
		}
		return fields[i], true
	}
	tp := s.Fields(ui.KindEnchantment, 0)
	src, _ := by(tp, "triggerx")
	dst, _ := by(tp, "targetx")
	if _, has := by(tp, "sourcex"); has || src.CellPair != "triggery" || !src.CellOwn || src.Label != "Source cell X" || dst.CellPair != "targety" || dst.CellOwn || dst.Label != "Target cell X" {
		t.Fatalf("teleport fields %+v", tp)
	}
	other := s.Fields(ui.KindEnchantment, 1)
	if f, _ := by(other, "targetx"); f.CellPair != "" {
		t.Fatalf("a trap of another spell lists a cell pair: %+v", f)
	}
	if _, err := s.SetFields(ui.KindEnchantment, 0, []ui.InspectionFieldValue{{Key: "triggerx", Value: 15}, {Key: "triggery", Value: 16}, {Key: "targetx", Value: 17}, {Key: "targety", Value: 18}}); err != nil {
		t.Fatal(err)
	}
	rec, err := s.model.Record(mapedit.KindEnchantment, 0)
	if err != nil {
		t.Fatal(err)
	}
	if rec[4] != 15 || rec[8] != 16 || rec[32] != 17 || rec[34] != 18 {
		t.Fatalf("record %x", rec)
	}
}
