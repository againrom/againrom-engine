package game

import (
	"bytes"
	"image"
	"strings"
	"testing"

	"againrom/pkg/mapedit"
	"againrom/pkg/ui"
)

const trapTakenWord = "already holds a trap"

func trapRequest(cell image.Point, spell int64) ui.InspectionNewRecord {
	return ui.InspectionNewRecord{Kind: ui.KindEnchantment, HasCell: true, Cell: cell, Values: []ui.InspectionFieldValue{
		{Key: "template", Value: int64(mapedit.SpellConstructor(0))}, {Key: "spell", Value: spell}, {Key: "power", Value: 40},
	}}
}

// trapCellRules runs the placing and moving rules on host, whose map holds a
// trap at taken and a trap at index mover on a cell of its own.
func trapCellRules(t *testing.T, host *mapInspectionEdits, taken, free image.Point, mover int, spell int64, moves bool) {
	t.Helper()
	refused := func(what string, err error) {
		t.Helper()
		if err == nil || !strings.Contains(err.Error(), trapTakenWord) {
			t.Fatalf("%s: error %v, want the taken-cell refusal", what, err)
		}
	}
	before := host.model.Bytes()
	_, err := host.AddRecord(trapRequest(taken, spell))
	refused("place on a taken cell", err)
	if !bytes.Equal(before, host.model.Bytes()) {
		t.Fatal("a refused placement changed the map")
	}
	source := func(c image.Point) error {
		_, err := host.SetFields(ui.KindEnchantment, mover, []ui.InspectionFieldValue{{Key: "triggerx", Value: int64(c.X)}, {Key: "triggery", Value: int64(c.Y)}})
		return err
	}
	refused("Source onto a taken cell", source(taken))
	if !bytes.Equal(before, host.model.Bytes()) {
		t.Fatal("a refused Source change changed the map")
	}
	if moves {
		_, err = host.Move(ui.KindEnchantment, mover, taken)
		refused("move onto a taken cell", err)
		if !bytes.Equal(before, host.model.Bytes()) {
			t.Fatal("a refused move changed the map")
		}
	}
	own := func() image.Point {
		x, _ := host.model.FieldValue(mapedit.KindEnchantment, mover, "triggerx")
		y, _ := host.model.FieldValue(mapedit.KindEnchantment, mover, "triggery")
		return image.Pt(int(x), int(y))
	}
	if err := source(own()); err != nil {
		t.Fatalf("Source onto its own cell: %v", err)
	}
	if err := source(free); err != nil || own() != free {
		t.Fatalf("Source onto a free cell: %v, now %v", err, own())
	}
	if _, err := host.AddRecord(trapRequest(image.Pt(free.X+1, free.Y), spell)); err != nil {
		t.Fatalf("place on a free cell: %v", err)
	}
}

func TestEditorRefusesATrapOnATakenCellInTheFirstGame(t *testing.T) {
	e := cellSpellFixture(t)
	host := e.Document().Edits.(*mapInspectionEdits)
	if _, err := host.AddRecord(trapRequest(image.Pt(10, 10), 13)); err != nil {
		t.Fatal(err)
	}
	mover := host.model.Count(mapedit.KindEnchantment) - 1
	trapCellRules(t, host, image.Pt(3, 4), image.Pt(20, 20), mover, 13, true)
}

func TestEditorRefusesATrapOnATakenCellInTheSecondGame(t *testing.T) {
	raw, _ := secondCellRecordsMap()
	d, err := secondTestInspector(t).inspect(raw, "second.alm")
	if err != nil {
		t.Fatal(err)
	}
	trapCellRules(t, d.Edits.(*mapInspectionEdits), image.Pt(12, 20), image.Pt(20, 18), 0, 5, false)
}
