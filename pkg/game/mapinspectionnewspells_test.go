package game

import (
	"bytes"
	"encoding/binary"
	"image"
	"strconv"
	"strings"
	"testing"

	"againrom/pkg/mapedit"
	"againrom/pkg/ui"
)

// cellSpellBytes is one type-9 record on the wire: the +0x00 word, the cell, the
// kind, the object id, spell, power and each element's three words.
func cellSpellBytes(tag, x, y uint32, kind uint16, id uint32, spell, power uint16, elements ...[3]uint16) []byte {
	b := make([]byte, 26, 26+6*len(elements))
	binary.LittleEndian.PutUint32(b, tag)
	binary.LittleEndian.PutUint32(b[4:], x)
	binary.LittleEndian.PutUint32(b[8:], y)
	binary.LittleEndian.PutUint16(b[12:], kind)
	binary.LittleEndian.PutUint32(b[14:], id)
	binary.LittleEndian.PutUint16(b[18:], spell)
	binary.LittleEndian.PutUint16(b[20:], power)
	binary.LittleEndian.PutUint32(b[22:], uint32(len(elements)))
	for _, el := range elements {
		b = binary.LittleEndian.AppendUint16(b, el[0])
		b = binary.LittleEndian.AppendUint16(b, el[1])
		b = binary.LittleEndian.AppendUint16(b, el[2])
	}
	return b
}

// The Add window adds a trap, a unit spell and a casting structure on a map
// with no type-9 record. Each new record holds what the original editor's
// constructor and that dialog's Add write (ALM-EDITT9-328), and a new trap's
// element word 2 is 0 (DIV-4045).
func TestEditorCellSpellsAddWithoutATemplate(t *testing.T) {
	e := spellMap(t)
	s := e.Document().Edits.(*mapInspectionEdits)
	original := s.model.Bytes()
	if n := s.model.Count(mapedit.KindEnchantment); n != 0 {
		t.Fatalf("the fixture holds %d type-9 records", n)
	}
	view, err := s.model.Map()
	if err != nil {
		t.Fatal(err)
	}
	unit := uint32(view.Units[0].UnitID)
	for i, tc := range []struct {
		name   string
		values map[string]string
		want   []byte
	}{
		{"trap", map[string]string{"template": strconv.Itoa(mapedit.SpellConstructor(0)), "link": "0", "cell": "5,6", "spell": "13", "power": "20", "castkind": "2"},
			cellSpellBytes(0, 5, 6, 2, 0, 13, 20, [3]uint16{5, 6, 0}, [3]uint16{5, 6, 0})},
		{"unit spell", map[string]string{"template": strconv.Itoa(mapedit.SpellConstructor(8)), "cell": "9,10", "spell": "7", "power": "30"},
			cellSpellBytes(0, 9, 10, 8, unit, 7, 30)},
		{"casting structure off its anchor", map[string]string{"template": strconv.Itoa(mapedit.SpellConstructor(4)), "link": "7", "cell": "4,5", "spell": "5", "power": "200", "range": "240"},
			cellSpellBytes(240, 4, 5, 4, 7, 5, 200)},
	} {
		if err := e.Key("casts"); err != nil {
			t.Fatal(err)
		}
		addSteps(t, e, tc.values)
		if n := s.model.Count(mapedit.KindEnchantment); n != i+1 {
			t.Fatalf("%s: %d records, want %d", tc.name, n, i+1)
		}
		got, _ := s.model.Record(mapedit.KindEnchantment, i)
		if !bytes.Equal(got, tc.want) {
			t.Fatalf("%s: %x want %x", tc.name, got, tc.want)
		}
		if e.Selected() != recordIndex(t, e, "Enchantment", i) {
			t.Fatalf("%s: the new record's popup did not open", tc.name)
		}
	}
	for s.model.CanUndo() {
		if err := e.Key("undo"); err != nil {
			t.Fatal(err)
		}
	}
	if !bytes.Equal(s.model.Bytes(), original) {
		t.Fatal("undo did not restore the map without type-9 records")
	}
}

// A casting structure's power and scan range take 0..255 through the
// popup's key and pointer Apply; 256 is refused with a message and writes
// nothing (DIV-4046).
func TestEditorCastingStructureFieldsStopAt255(t *testing.T) {
	e := spellMap(t, cellSpellBytes(240, 4, 5, 4, 7, 5, 100))
	s := e.Document().Edits.(*mapInspectionEdits)
	e.Select(recordIndex(t, e, "Enchantment", 0), false)
	fields := s.Fields("Enchantment", 0)
	fieldAt := map[string]int{}
	for i, f := range fields {
		fieldAt[f.Key] = i
	}
	if len(fields) != 3 || fieldAt["power"] != 1 || fieldAt["range"] != 2 || !strings.Contains(fields[2].Label, "Scan range") {
		t.Fatalf("casting structure fields %+v", fields)
	}
	pointer := func(field int, text string) {
		t.Helper()
		r, ok := e.FieldRect(field)
		messageClick(t, e, r, ok)
		_ = e.Key("selectall")
		e.TextInput(text)
		r, ok = e.PopupButtonRect(0)
		messageClick(t, e, r, ok)
	}
	for _, route := range []string{"key", "pointer"} {
		for _, key := range []string{"power", "range"} {
			offset, width := 20, 2
			if key == "range" {
				offset, width = 0, 4
			}
			for _, v := range []int64{0, 255, 256} {
				before := s.model.Bytes()
				old, _ := s.model.Record(mapedit.KindEnchantment, 0)
				at := bytes.Index(before, old)
				e.Select(recordIndex(t, e, "Enchantment", 0), false)
				if route == "key" {
					_ = enterField(e, fieldAt[key], strconv.FormatInt(v, 10))
				} else {
					pointer(fieldAt[key], strconv.FormatInt(v, 10))
				}
				if v == 256 {
					shown := e.Message() + strings.Join(e.PopupLines(), " ")
					if !bytes.Equal(before, s.model.Bytes()) || !strings.Contains(shown, "0..255") {
						t.Fatalf("%s %s 256: bytes changed or no message (%q)", route, key, shown)
					}
					_ = e.Key("escape")
					continue
				}
				want := bytes.Clone(before)
				if width == 4 {
					binary.LittleEndian.PutUint32(want[at+offset:], uint32(v))
				} else {
					binary.LittleEndian.PutUint16(want[at+offset:], uint16(v))
				}
				if at < 0 || !bytes.Equal(want, s.model.Bytes()) {
					rec, _ := s.model.Record(mapedit.KindEnchantment, 0)
					t.Fatalf("%s %s %d: the map is not the record with that one field: %x was %x (%q)", route, key, v, rec, old, e.Message())
				}
			}
		}
	}
}

// A casting structure record off its structure's anchor cell, as most
// shipped ones are (ALM-T9BIND-335), edits and serves as a template; a
// record whose id names no object is listed as unbound and takes no edit.
func TestEditorCellSpellsBindByIdAlone(t *testing.T) {
	off := cellSpellBytes(50, 4, 5, 4, 7, 5, 150)
	unboundCaster := cellSpellBytes(9, 10, 10, 4, 99, 5, 20)
	unboundBook := cellSpellBytes(0, 11, 11, 8, 555, 6, 20)
	e := spellMap(t, off, unboundCaster, unboundBook)
	s := e.Document().Edits.(*mapInspectionEdits)
	e.Select(recordIndex(t, e, "Enchantment", 0), false)
	recordFieldInput(t, e, "power", 151)
	want := bytes.Clone(off)
	binary.LittleEndian.PutUint16(want[20:], 151)
	if got, _ := s.model.Record(mapedit.KindEnchantment, 0); !bytes.Equal(got, want) {
		t.Fatalf("off-anchor edit: %x want %x", got, want)
	}
	if err := e.Key("casts"); err != nil {
		t.Fatal(err)
	}
	addSteps(t, e, map[string]string{"template": "0", "cell": "5,5"})
	copied := bytes.Clone(want)
	binary.LittleEndian.PutUint32(copied[4:], 5)
	binary.LittleEndian.PutUint32(copied[8:], 5)
	if got, _ := s.model.Record(mapedit.KindEnchantment, 3); !bytes.Equal(got, copied) {
		t.Fatalf("off-anchor copy: %x want %x", got, copied)
	}
	before := s.model.Bytes()
	unbound := strings.TrimSuffix(strings.SplitN(s.inspector.word("editor.cast.unbound"), "(", 2)[1], ")")
	for _, i := range []int{1, 2} {
		r := e.Document().Records[recordIndex(t, e, "Enchantment", i)]
		if !strings.Contains(r.Label, strings.Fields(unbound)[0]) {
			t.Fatalf("record %d is not listed as unbound: %q", i, r.Label)
		}
		if f := s.Fields("Enchantment", i); len(f) != 0 {
			t.Fatalf("unbound record %d offers fields %+v", i, f)
		}
		if _, err := s.SetField("Enchantment", i, "power", 30); err == nil {
			t.Fatalf("unbound record %d took an edit", i)
		}
	}
	if !bytes.Equal(before, s.model.Bytes()) {
		t.Fatal("an unbound record changed")
	}
	add := ui.InspectionNewRecord{Kind: "Enchantment", HasCell: true, Cell: image.Pt(6, 6), Values: []ui.InspectionFieldValue{
		{Key: "template", Value: int64(mapedit.SpellConstructor(4))}, {Key: "link", Value: 99}, {Key: "spell", Value: 5}}}
	if _, err := s.AddRecord(add); err == nil || !bytes.Equal(before, s.model.Bytes()) {
		t.Fatal("a casting structure naming no structure was added", err)
	}
}
