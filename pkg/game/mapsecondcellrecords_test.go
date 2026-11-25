package game

import (
	"bytes"
	"encoding/binary"
	"image"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"againrom/pkg/formats/alm"
	"againrom/pkg/mapedit"
	"againrom/pkg/ui"
)

// secondCellRecordsMap is the fixture second-game map with a type-3 byte of
// 52 at (15, 15) and four type-9 records: a trap of spell 5 and power 70 at
// (10, 11), a teleport at (12, 20), a caster of spell 10, power 255 and
// radius 6 on structure 1, and an item record at cell 0, 0. The trap's
// entry words and the caster's radius dword hold bytes no claim reads.
// at holds each record's file offset.
func secondCellRecordsMap() (b []byte, at []int) {
	le := binary.LittleEndian
	record := func(x, y uint32, mode uint16, key uint32, spell, power byte, cells ...[3]uint16) []byte {
		r := make([]byte, 26)
		le.PutUint32(r[4:], x)
		le.PutUint32(r[8:], y)
		le.PutUint16(r[12:], mode)
		le.PutUint32(r[14:], key)
		r[18], r[20] = spell, power
		le.PutUint32(r[22:], uint32(len(cells)))
		for _, c := range cells {
			r = le.AppendUint16(le.AppendUint16(le.AppendUint16(r, c[0]), c[1]), c[2])
		}
		return r
	}
	caster := record(12, 12, 4, 1, 10, 255)
	le.PutUint32(caster[0:], 6|0x0100)
	records := [][]byte{
		record(10, 11, 0, 0, 5, 70, [3]uint16{10, 11, 9396}, [3]uint16{10, 11, 9396}),
		record(12, 20, 0, 0, secondCellMove, 255, [3]uint16{12, 20, 0}, [3]uint16{14, 9, 0}),
		caster,
		record(0, 0, 1, 0, 0, 0, [3]uint16{6, 10, 0}),
	}
	payload := le.AppendUint32(nil, uint32(len(records)))
	out := secondFixtureMap()
	out[alm.FileHeaderSize+4*alm.RecordHeaderSize+660+3*32*32+15*32+15] = 52
	le.PutUint32(out[12:], le.Uint32(out[12:])+1)
	h := make([]byte, alm.RecordHeaderSize)
	le.PutUint32(h[0:], 7)
	le.PutUint32(h[4:], alm.RecordHeaderSize)
	le.PutUint32(h[12:], alm.TypeTileMarkers)
	start := len(out) + alm.RecordHeaderSize
	for _, r := range records {
		at = append(at, start+len(payload))
		payload = append(payload, r...)
	}
	le.PutUint32(h[8:], uint32(len(payload)))
	return append(append(out, h...), payload...), at
}

func secondCellRecordFields(host *mapInspectionEdits, index int) map[string]int64 {
	out := map[string]int64{}
	for _, f := range host.Fields(ui.KindEnchantment, index) {
		out[f.Key] = f.Value
	}
	return out
}

// On a second-game map a trap reads as its cell event's spell, power, cell,
// source and target (R2-ENGINE-1698), a structure caster as its spell, power
// and scan radius (R2-ENGINE-1009, R2-ENGINE-1560), and a type-3 byte by its
// value with what the game does with it (R2-ENGINE-1667); an item record
// lists no field.
func TestSecondMapCellRecordsReadWithNamedFields(t *testing.T) {
	raw, _ := secondCellRecordsMap()
	if m, err := alm.OpenROM2(raw); err != nil || m.Overlay[15*32+15] != 52 || len(m.Enchantments) != 4 {
		t.Fatalf("the fixture does not decode: %v", err)
	}
	x := secondTestInspector(t)
	d, err := x.inspect(raw, "second.alm")
	if err != nil {
		t.Fatal(err)
	}
	host := d.Edits.(*mapInspectionEdits)
	want := map[int]map[string]int64{
		0: {"spell": 5, "power": 70, "triggerx": 10, "triggery": 11, "sourcex": 10, "sourcey": 11, "targetx": 10, "targety": 11},
		1: {"spell": secondCellMove, "triggerx": 12, "triggery": 20, "targetx": 14, "targety": 9},
		2: {"spell": 10, "power": 255, "range": 6},
		3: {},
	}
	for i, w := range want {
		if got := secondCellRecordFields(host, i); !mapsEqual(got, w) {
			t.Fatalf("type-9 record %d lists %v, want %v", i, got, w)
		}
	}
	if got := recordByKind(t, d, ui.KindEnchantment, 0).Label; got != "Trap: 5, power 70, source 10,11, target 10,11" {
		t.Fatalf("the trap reads %q", got)
	}
	scenery := recordByKind(t, d, ui.KindScenery, 15*32+15)
	if scenery.Label != "Type-3 byte 52" || !slices.Contains(scenery.Details, x.word("editor.second.scenery.detail")) {
		t.Fatalf("the type-3 byte reads %q %q", scenery.Label, scenery.Details)
	}
}

func mapsEqual(a, b map[string]int64) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if w, ok := b[k]; !ok || w != v {
			return false
		}
	}
	return true
}

// A changed trap and caster save as the stage reads them: the event, power
// and target bytes of the trap, the spell, power and radius bytes of the
// caster, and no other byte of the file. A trap refuses the teleport's event.
func TestSecondMapCellRecordsChangeAsTheClaimsLayout(t *testing.T) {
	raw, at := secondCellRecordsMap()
	x := secondTestInspector(t)
	d, err := x.inspect(raw, "second.alm")
	if err != nil {
		t.Fatal(err)
	}
	host := d.Edits.(*mapInspectionEdits)
	if _, err := host.SetFields(ui.KindEnchantment, 0, []ui.InspectionFieldValue{
		{Key: "spell", Value: 6}, {Key: "power", Value: 40}, {Key: "targetx", Value: 12}, {Key: "targety", Value: 13}}); err != nil {
		t.Fatalf("a changed trap: %v", err)
	}
	if _, err := host.SetFields(ui.KindEnchantment, 2, []ui.InspectionFieldValue{
		{Key: "spell", Value: 2}, {Key: "power", Value: 30}, {Key: "range", Value: 5}}); err != nil {
		t.Fatalf("a changed caster: %v", err)
	}
	if _, err := host.SetField(ui.KindEnchantment, 0, "spell", secondCellMove); err == nil || !strings.Contains(err.Error(), "teleport") {
		t.Fatalf("a trap took the teleport's event: %v", err)
	}
	if got := secondCellEvents(t, host.model.Bytes()); got[image.Pt(10, 11)] != [6]byte{6, 40, 10, 11, 12, 13} {
		t.Fatalf("the trap's cell event is %v", got[image.Pt(10, 11)])
	}
	view, err := host.model.Map()
	if err != nil {
		t.Fatal(err)
	}
	if got := view.BuildingCasters(); len(got) != 1 || got[0] != (alm.BuildingCasterRecord{Key: 1, X: 12, Y: 12, Spell: 2, Power: 30, Reach: 5}) {
		t.Fatalf("the casters are %+v", got)
	}
	want := slices.Clone(raw)
	for off, b := range map[int]byte{at[0] + 18: 6, at[0] + 20: 40, at[0] + 32: 12, at[0] + 34: 13, at[2] + 18: 2, at[2] + 20: 30, at[2]: 5} {
		want[off] = b
	}
	target, err := host.SaveAs(filepath.Join(t.TempDir(), "changed.alm"))
	if err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if at := firstDiffering(want, got); at >= 0 {
		t.Fatalf("the saved map differs from the expected bytes at offset %d", at)
	}
}

func firstDiffering(a, b []byte) int {
	if bytes.Equal(a, b) {
		return -1
	}
	for i := range min(len(a), len(b)) {
		if a[i] != b[i] {
			return i
		}
	}
	return min(len(a), len(b))
}

// On a second-game map the palette builds a trap of a chosen spell and power
// and a structure caster, a trap area fills a box with traps, and a removed
// trap takes its cell event alone. The power step follows the spell: a
// teleport takes none.
func TestSecondMapTrapAndCasterAddedAndRemoved(t *testing.T) {
	raw, _ := secondCellRecordsMap()
	d, err := secondTestInspector(t).inspect(raw, "second.alm")
	if err != nil {
		t.Fatal(err)
	}
	host := d.Edits.(*mapInspectionEdits)
	cell, structure := int64(mapedit.SpellConstructor(0)), int64(mapedit.SpellConstructor(4))
	template := host.NewRecordStep(ui.InspectionNewRecord{Kind: ui.KindEnchantment}, "template")
	var constructors []int64
	for _, c := range template.Choices {
		if where, ok := template.Templates[c.Value]; ok && where.Constructor == c.Value {
			constructors = append(constructors, c.Value)
		}
	}
	if !slices.Equal(constructors, []int64{cell, structure}) || template.Templates[structure].Target != ui.KindStructure {
		t.Fatalf("the palette offers constructors %v, want the cell record and the structure caster", constructors)
	}
	step := func(spell int64, key string) ui.InspectionNewStep {
		return host.NewRecordStep(ui.InspectionNewRecord{Kind: ui.KindEnchantment, Values: []ui.InspectionFieldValue{{Key: "template", Value: cell}, {Key: "spell", Value: spell}}}, key)
	}
	if power := step(5, "power"); power.Absent || power.Bounds == nil || power.Bounds.Max != 255 {
		t.Fatalf("a trap's power step is %+v", power)
	}
	if power := step(secondCellMove, "power"); !power.Absent {
		t.Fatalf("a teleport offers a power step %+v", power)
	}

	add := func(r ui.InspectionNewRecord) {
		t.Helper()
		r.Kind, r.HasCell = ui.KindEnchantment, true
		if _, err := host.AddRecord(r); err != nil {
			t.Fatalf("add %+v: %v", r, err)
		}
	}
	add(ui.InspectionNewRecord{Cell: image.Pt(20, 18), Values: []ui.InspectionFieldValue{{Key: "template", Value: cell}, {Key: "spell", Value: 5}, {Key: "power", Value: 60}}})
	add(ui.InspectionNewRecord{Cell: image.Pt(16, 16), Values: []ui.InspectionFieldValue{{Key: "template", Value: structure}, {Key: "link", Value: 2},
		{Key: "spell", Value: 10}, {Key: "power", Value: 255}, {Key: "range", Value: 6}}})
	add(ui.InspectionNewRecord{Cell: image.Pt(9, 9), Area: image.Rect(9, 9, 11, 10), Values: []ui.InspectionFieldValue{{Key: "template", Value: cell}, {Key: "spell", Value: 6}, {Key: "power", Value: 20}}})

	events := secondCellEvents(t, host.model.Bytes())
	for c, want := range map[image.Point][6]byte{{20, 18}: {5, 60, 20, 18, 20, 18}, {9, 9}: {6, 20, 9, 9, 9, 9}, {10, 9}: {6, 20, 10, 9, 10, 9}, {10, 11}: {5, 70, 10, 11, 10, 11}} {
		if events[c] != want {
			t.Fatalf("the cell event at %v is %v, want %v", c, events[c], want)
		}
	}
	rec, err := host.model.Record(mapedit.KindEnchantment, 4)
	if err != nil || len(rec) != alm.SecondMarkerHead+2*alm.SecondMarkerEntry || binary.LittleEndian.Uint16(rec[alm.SecondMarkerMode:]) != 0 {
		t.Fatalf("the new trap is % x (%v)", rec, err)
	}
	view, err := host.model.Map()
	if err != nil {
		t.Fatal(err)
	}
	if got := view.BuildingCasters(); len(got) != 2 || got[1] != (alm.BuildingCasterRecord{Key: 2, X: 16, Y: 16, Spell: 10, Power: 255, Reach: 6}) {
		t.Fatalf("the casters are %+v", got)
	}

	if _, err := host.Delete(ui.KindEnchantment, 0); err != nil {
		t.Fatalf("a removed trap: %v", err)
	}
	after := secondCellEvents(t, host.model.Bytes())
	if _, ok := after[image.Pt(10, 11)]; ok || len(after) != len(events)-1 || host.model.Count(mapedit.KindEnchantment) != 7 {
		t.Fatalf("after the removal the cell events are %v", after)
	}
}
