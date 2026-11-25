package game

import (
	"encoding/binary"
	"image"
	"slices"
	"testing"

	"againrom/pkg/formats/alm"
	"againrom/pkg/mapedit"
	"againrom/pkg/ui"
)

// secondCellMove is the second game's cell move event (R2-ENGINE-1778).
const secondCellMove = 23

// secondTeleportMap is the fixture second-game map with a type-9 record
// section: a cell record of event 23 at (10, 11) to (14, 9) with power 255,
// and a mode-4 caster at (9, 9), whose event is not decoded for editing.
func secondTeleportMap() []byte {
	le := binary.LittleEndian
	record := func(x, y uint32, mode uint16, spell, power uint16, cells ...[2]uint16) []byte {
		b := make([]byte, 26)
		le.PutUint32(b[4:], x)
		le.PutUint32(b[8:], y)
		le.PutUint16(b[12:], mode)
		le.PutUint16(b[18:], spell)
		le.PutUint16(b[20:], power)
		le.PutUint32(b[22:], uint32(len(cells)))
		for _, c := range cells {
			e := make([]byte, 6)
			le.PutUint16(e[0:], c[0])
			le.PutUint16(e[2:], c[1])
			b = append(b, e...)
		}
		return b
	}
	payload := le.AppendUint32(nil, 2)
	payload = append(payload, record(10, 11, 0, secondCellMove, 255, [2]uint16{10, 11}, [2]uint16{14, 9})...)
	payload = append(payload, record(9, 9, 4, 5, 10)...)
	out := secondFixtureMap()
	le.PutUint32(out[12:], le.Uint32(out[12:])+1)
	h := make([]byte, alm.RecordHeaderSize)
	le.PutUint32(h[0:], 7)
	le.PutUint32(h[4:], alm.RecordHeaderSize)
	le.PutUint32(h[8:], uint32(len(payload)))
	le.PutUint32(h[12:], alm.TypeTileMarkers)
	return append(append(out, h...), payload...)
}

// The second game's cell event of a type-9 record as the stage hands it to
// the writer: event, power, source and target (R2-ENGINE-1776).
func secondCellEvents(t *testing.T, b []byte) map[image.Point][6]byte {
	t.Helper()
	m, err := mapedit.New(b)
	if err != nil {
		t.Fatal(err)
	}
	view, err := m.Map()
	if err != nil {
		t.Fatal(err)
	}
	bindings, err := view.CellBindings()
	if err != nil {
		t.Fatal(err)
	}
	out := map[image.Point][6]byte{}
	for _, c := range bindings {
		out[image.Pt(int(c.X), int(c.Y))] = [6]byte{c.Spell, c.Power, c.SourceX, c.SourceY, c.LastX, c.LastY}
	}
	return out
}

// On a second-game map the editor adds, changes and removes a type-9
// teleport: the palette offers the cell record and event 23, a new
// teleport saves as the original's six-byte cell event with its source at
// its cell (R2-ASSET-387) and power 0, which the move does not read
// (R2-ENGINE-1779); a loaded teleport keeps its power, and a record of
// another mode lists no field.
func TestSecondMapTeleportRecordAddedChangedAndRemoved(t *testing.T) {
	d, err := secondTestInspector(t).inspect(secondTeleportMap(), "second.alm")
	if err != nil {
		t.Fatal(err)
	}
	host := d.Edits.(*mapInspectionEdits)
	if got := secondCellEvents(t, host.model.Bytes()); got[image.Pt(10, 11)] != [6]byte{23, 255, 10, 11, 14, 9} || len(got) != 1 {
		t.Fatalf("the fixture's cell events are %v", got)
	}

	template := host.NewRecordStep(ui.InspectionNewRecord{Kind: ui.KindEnchantment}, "template")
	cell := int64(mapedit.SpellConstructor(0))
	var constructors []int64
	for _, c := range template.Choices {
		if where, ok := template.Templates[c.Value]; ok && where.Constructor == c.Value {
			constructors = append(constructors, c.Value)
		}
	}
	if !slices.Contains(constructors, cell) {
		t.Fatalf("the palette offers constructors %v, want the cell record %d", constructors, cell)
	}
	spell := host.NewRecordStep(ui.InspectionNewRecord{Kind: ui.KindEnchantment, Values: []ui.InspectionFieldValue{{Key: "template", Value: cell}}}, "spell")
	if spell.Absent || !slices.ContainsFunc(spell.Choices, func(c ui.InspectionChoice) bool { return c.Value == secondCellMove && c.Label == "Teleport" }) {
		t.Fatalf("the spell step offers %+v, want the teleport event among the spells", spell)
	}

	add := ui.InspectionNewRecord{Kind: ui.KindEnchantment, Cell: image.Pt(12, 20), HasCell: true, Values: []ui.InspectionFieldValue{
		{Key: "template", Value: cell}, {Key: "spell", Value: secondCellMove},
		{Key: "targetx", Value: 20}, {Key: "targety", Value: 12}}}
	if _, err := host.AddRecord(add); err != nil {
		t.Fatalf("a new teleport: %v", err)
	}
	got := secondCellEvents(t, host.model.Bytes())
	if got[image.Pt(12, 20)] != [6]byte{23, 0, 12, 20, 20, 12} || got[image.Pt(10, 11)] != [6]byte{23, 255, 10, 11, 14, 9} {
		t.Fatalf("after the add the cell events are %v", got)
	}
	rec, err := host.model.Record(mapedit.KindEnchantment, 2)
	if err != nil || len(rec) != alm.SecondMarkerHead+2*alm.SecondMarkerEntry || binary.LittleEndian.Uint16(rec[alm.SecondMarkerMode:]) != 0 {
		t.Fatalf("the new record is % x (%v)", rec, err)
	}

	fields := map[string]int64{}
	for _, f := range host.Fields(ui.KindEnchantment, 0) {
		fields[f.Key] = f.Value
	}
	if len(fields) != 5 || fields["spell"] != 23 || fields["triggerx"] != 10 || fields["triggery"] != 11 || fields["targetx"] != 14 || fields["targety"] != 9 {
		t.Fatalf("the loaded teleport lists %v", fields)
	}
	if f := host.Fields(ui.KindEnchantment, 1); len(f) != 0 {
		t.Fatalf("the caster lists %+v", f)
	}
	if _, err := host.SetField(ui.KindEnchantment, 0, "spell", 5); err == nil {
		t.Fatal("a teleport took event 5")
	}
	if _, err := host.SetFields(ui.KindEnchantment, 0, []ui.InspectionFieldValue{{Key: "triggerx", Value: 15}, {Key: "targety", Value: 10}}); err != nil {
		t.Fatalf("a changed teleport: %v", err)
	}
	if got := secondCellEvents(t, host.model.Bytes()); got[image.Pt(15, 11)] != [6]byte{23, 255, 10, 11, 14, 10} || len(got) != 2 {
		t.Fatalf("after the change the cell events are %v", got)
	}

	if _, err := host.Delete(ui.KindEnchantment, 0); err != nil {
		t.Fatalf("a removed teleport: %v", err)
	}
	if got := secondCellEvents(t, host.model.Bytes()); len(got) != 1 || got[image.Pt(12, 20)] != [6]byte{23, 0, 12, 20, 20, 12} {
		t.Fatalf("after the removal the cell events are %v", got)
	}
	if n := host.model.Count(mapedit.KindEnchantment); n != 2 {
		t.Fatalf("after the removal the map holds %d type-9 records", n)
	}
}
