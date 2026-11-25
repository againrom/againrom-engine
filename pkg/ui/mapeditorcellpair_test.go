package ui

import (
	"image"
	"strconv"
	"testing"
)

// cellPairHost shows a unit with a Source and a Target cell as X and Y fields
// whose map pick sets both.
type cellPairHost struct{ *popupHost }

func (h cellPairHost) Fields(kind RecordKind, index int) []InspectionField {
	return []InspectionField{
		{Key: "triggerx", Label: "Source cell X", Value: 5, CellPair: "triggery", CellOwn: true},
		{Key: "triggery", Label: "Source cell Y", Value: 6},
		{Key: "targetx", Label: "Target cell X", Value: 7, CellPair: "targety"},
		{Key: "targety", Label: "Target cell Y", Value: 8},
	}
}

func cellPairEditor(t *testing.T) *MapEditor {
	t.Helper()
	e, h := popupEditor(t)
	_ = e.Open("popups")
	e.doc.Edits = cellPairHost{h}
	e.Select(0, false)
	e.objectSpec()
	return e
}

func fieldBuf(e *MapEditor, key string) string {
	for i, f := range e.obj.fields {
		if f.Key == key {
			return e.obj.buf[i]
		}
	}
	return "?"
}

// A map click after a cell pair's On map button sets both of its fields, Esc
// keeps them, and the record's own cell pair leaves no cell row of its own.
func TestCellPairFieldsTakeAMapClickAndKeepOnEscape(t *testing.T) {
	e := cellPairEditor(t)
	if e.obj.cell != -1 {
		t.Fatalf("the own cell pair left cell row %d", e.obj.cell)
	}
	row := func(key string) int {
		for i, f := range e.obj.fields {
			if f.Key == key {
				return e.fieldRow(i)
			}
		}
		t.Fatalf("no field %s", key)
		return -1
	}
	for _, c := range []struct {
		key, x, y string
		at        image.Point
	}{{"targetx", "targetx", "targety", image.Pt(12, 13)}, {"triggerx", "triggerx", "triggery", image.Pt(20, 21)}} {
		e.objectRowPress(popupRowButton, row(c.key))
		if e.pick != pickScriptCell {
			t.Fatalf("%s: no pick armed", c.key)
		}
		e.ShowCell(c.at)
		ox, oy := e.mapOrigin()
		e.pickAt(e.CellPoint(c.at).Sub(image.Pt(int(ox), int(oy))))
		if fieldBuf(e, c.x) != strconv.Itoa(c.at.X) || fieldBuf(e, c.y) != strconv.Itoa(c.at.Y) {
			t.Fatalf("%s: fields %q,%q after a click at %v", c.key, fieldBuf(e, c.x), fieldBuf(e, c.y), c.at)
		}
	}
	e.objectRowPress(popupRowButton, row("targetx"))
	_ = e.Key("escape")
	if e.pick != pickNone || fieldBuf(e, "targetx") != "12" || fieldBuf(e, "targety") != "13" {
		t.Fatalf("Esc left pick %q, target %q,%q", e.pick, fieldBuf(e, "targetx"), fieldBuf(e, "targety"))
	}
}

// A placed record whose fields hold a cell pair opens its window; a record
// without one is placed on as before.
func TestPlacedRecordWithACellPairOpensItsWindow(t *testing.T) {
	e := cellPairEditor(t)
	if !e.placedCellPair(RecordRef{Kind: KindEnchantment, Index: 0}) || e.placedCellPair(RecordRef{Kind: KindUnit, Index: 0}) {
		t.Fatal("placedCellPair misreads the fields")
	}
}
