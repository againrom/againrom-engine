package game

import (
	"bytes"
	"image"
	"slices"
	"testing"

	"againrom/pkg/mapedit"
	"againrom/pkg/mapload"
	"againrom/pkg/scriptprofile"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// firstCellMove is the first game's relocating cell spell (TRIG-CELLTAIL-035).
const firstCellMove = 26

// On a first-game map the editor adds a type-9 trap of spell 26 with its
// cell, its destination and its power; it reads as a teleport, changes,
// saves as the record the original turns into the tail {26, power, source,
// destination} (TERR-229, ALM-TRIG-049) and reloads into a world holding
// that tail. Action 25 offers no teleport form.
func TestFirstMapTeleportTrapAddedChangedSavedAndReloaded(t *testing.T) {
	e := spellMap(t)
	s := e.Document().Edits.(*mapInspectionEdits)
	cell := int64(mapedit.SpellConstructor(0))
	spell := s.NewRecordStep(ui.InspectionNewRecord{Kind: ui.KindEnchantment, Values: []ui.InspectionFieldValue{{Key: "template", Value: cell}}}, "spell")
	if spell.Absent || !slices.ContainsFunc(spell.Choices, func(c ui.InspectionChoice) bool { return c.Value == firstCellMove }) {
		t.Fatalf("the spell step offers %+v, want spell 26", spell.Choices)
	}
	add := ui.InspectionNewRecord{Kind: ui.KindEnchantment, Cell: image.Pt(12, 20), HasCell: true, Values: []ui.InspectionFieldValue{
		{Key: "template", Value: cell}, {Key: "spell", Value: firstCellMove}, {Key: "power", Value: 9},
		{Key: "targetx", Value: 20}, {Key: "targety", Value: 12}}}
	doc, err := s.AddRecord(add)
	if err != nil {
		t.Fatalf("a new teleport trap: %v", err)
	}
	want := cellSpellBytes(0, 12, 20, 0, 0, firstCellMove, 9, [3]uint16{12, 20, 0}, [3]uint16{20, 12, 0})
	if got, err := s.model.Record(mapedit.KindEnchantment, 0); err != nil || !bytes.Equal(got, want) {
		t.Fatalf("the new record is %x (%v), want %x", got, err, want)
	}
	if i := slices.IndexFunc(doc.Records, func(r ui.InspectionRecord) bool { return r.Kind == ui.KindEnchantment && r.Index == 0 }); i < 0 || doc.Records[i].Label != "Teleport: 12,20 to 20,12" {
		t.Fatalf("the trap is record %d of %+v", i, doc.Records)
	}
	if _, err := s.SetFields(ui.KindEnchantment, 0, []ui.InspectionFieldValue{{Key: "triggerx", Value: 13}, {Key: "targety", Value: 14}, {Key: "power", Value: 40}}); err != nil {
		t.Fatalf("a changed teleport trap: %v", err)
	}
	saved, err := s.model.SaveBytes()
	if err != nil {
		t.Fatal(err)
	}
	reopened, err := mapedit.New(saved)
	if err != nil {
		t.Fatal(err)
	}
	view, err := reopened.Map()
	if err != nil {
		t.Fatal(err)
	}
	bindings, err := view.CellBindings()
	if err != nil || len(bindings) != 1 {
		t.Fatalf("the saved map binds %+v (%v)", bindings, err)
	}
	if b := bindings[0]; b.X != 13 || b.Y != 20 || [6]byte{b.Spell, b.Power, b.SourceX, b.SourceY, b.LastX, b.LastY} != [6]byte{firstCellMove, 40, 12, 20, 20, 14} {
		t.Fatalf("the saved record binds %+v", b)
	}
	w, err := mapload.FromALMWith(view, nil, mapload.DifficultyNormal)
	if err != nil {
		t.Fatal(err)
	}
	tail := sim.CellTail{X: 13, Y: 20, Bytes: [6]byte{firstCellMove, 40, 12, 20, 20, 14}}
	if got := w.CellTails(); len(got) != 1 || got[0] != tail {
		t.Fatalf("the reloaded world holds %+v, want %+v", got, tail)
	}

	row, _ := scriptprofile.OpRow(scriptprofile.OpsFirst, false, 25)
	if _, ok := row.Variant(firstCellMove); ok || slices.ContainsFunc(row.Presets, func(p scriptprofile.WizardPreset) bool { return p.Command == firstCellMove }) {
		t.Fatalf("first-game action 25 offers a teleport: %+v", row)
	}
}
