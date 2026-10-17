package mapedit

import (
	"encoding/binary"
	"fmt"
	"math"
	"strings"
	"unicode/utf8"

	"againrom/pkg/formats/alm"
)

// FieldCheck names the reference check a host runs on a set value beyond the
// field's own range. This package knows no installed table or roster, so it
// states which check a field needs and the host performs it.
type FieldCheck int

const (
	CheckRangeOnly       FieldCheck = iota // the range below is the whole rule
	CheckRosterSlot                        // a 1-based slot in the type-5 roster (ALM-OWN-039)
	CheckClassKey                          // a class key the installed tables hold (ALM-CLS-038)
	CheckDefinitionID                      // zero, or a definition id the installed tables hold
	CheckStructureKind                     // an ID the installed structures.reg holds (ALM-CLS-036)
	CheckItemCode                          // an item code the installed item tables hold; 0 passes
	CheckScriptUnit                        // a unit id a type-6 record holds, or a hero-band value (ALM-TRIG-046)
	CheckScriptStructure                   // a structure id a type-4 record's +0x12 word holds (ALM-TRIG-046)
	CheckScriptGroup                       // a group id a type-6 record holds (ALM-TRIG-046)
	CheckScriptPlayer
	CheckStructureIdentity
	CheckItemRecipe
	CheckUnitTier // a secondary key that selects a Units row of the unit's class (ALM-CLS-052)
)

// Field is one settable field of a record, at a record-relative
// offset. A field with Element set is one element of a repeated list whose
// u32 count sits at Counter: 0 removes the element, and the row whose Offset
// is the record's end appends one.
type Field struct {
	Key      string // stable name a caller addresses the field by
	Label    string // window text
	Offset   int
	Width    int // byte width; numeric fields are little-endian
	Signed   bool
	Text     *alm.TextField // a text field and its codec; nil: a number
	Min, Max int64
	Check    FieldCheck
	Element  int
	Counter  int
}

// UnitField and UnitFieldCheck are the type-6 names of Field and FieldCheck.
type (
	UnitField      = Field
	UnitFieldCheck = FieldCheck
)

// unitFields is the one table of type-6 fields a caller may set. Offsets are
// file offsets into the record (ALM-UNIT-040).
var unitFields = []Field{
	{Key: "owner", Label: "Owner slot", Offset: 0x14, Width: 4, Min: 1, Max: math.MaxUint32, Check: CheckRosterSlot},
	{Key: "class", Label: "Class", Offset: 0x08, Width: 2, Signed: true, Min: math.MinInt16, Max: math.MaxInt16, Check: CheckClassKey},
	{Key: "tier", Label: "Tier", Offset: 0x0a, Width: 2, Min: 0, Max: math.MaxUint16, Check: CheckUnitTier},
	{Key: "definition", Label: "Human definition (0 = class)", Offset: 0x10, Width: 4, Min: 0, Max: math.MaxUint32, Check: CheckDefinitionID},
	{Key: "hp", Label: "Hit points (-1 = default)", Offset: 0x20, Width: 2, Signed: true, Min: math.MinInt16, Max: math.MaxInt16},
	{Key: "group", Label: "Group id (0 = none)", Offset: 0x42, Width: 4, Min: 0, Max: math.MaxUint32},
}

// Fields returns the settable fields of k's record i in window order.
func (e *Editor) Fields(k Kind, i int) ([]Field, error) {
	w, err := e.walkIndex(k, "list the fields of", i)
	if err != nil {
		return nil, err
	}
	at, l := w.at(i)
	return w.sec.fields(e, e.data[at:at+l]), nil
}

func fieldByKey(fields []Field, key string) (Field, bool) {
	for _, f := range fields {
		if f.Key == key {
			return f, true
		}
	}
	return Field{}, false
}

// FieldValue reads the field named key of k's record i. The appending row
// reads 0.
func (e *Editor) FieldValue(k Kind, i int, key string) (int64, error) {
	rec, err := e.Record(k, i)
	if err != nil {
		return 0, err
	}
	f, ok := fieldByKey(sections[k].fields(e, rec), key)
	if !ok {
		return 0, fmt.Errorf("mapedit: no %s field %q", sections[k].name, key)
	}
	if f.Text != nil {
		return 0, fmt.Errorf("mapedit: %s is a text field", f.Label)
	}
	if f.Offset >= len(rec) {
		return 0, nil
	}
	return f.decode(rec[f.Offset:]), nil
}

// FieldSet is one field of a record and the value to write into it.
type FieldSet struct {
	Key   string
	Value int64
}

// SetField writes v into the field named key of k's record i as one edit. A
// value outside the field's range is refused and writes nothing; the field's
// Check is the caller's to run. A write that keeps the record's length edits
// the field's bytes alone; one that adds or removes an element, or the 0x21
// extension, replaces the record and moves the payloadSize word.
func (e *Editor) SetField(k Kind, i int, key string, v int64) error {
	return e.SetFields(k, i, FieldSet{key, v})
}

// SetFields writes several fields of one record as one edit, so one Undo
// reverts them together. Each write follows SetField's rules and sees the
// record the writes before it left; a key named twice is refused.
func (e *Editor) SetFields(k Kind, i int, sets ...FieldSet) error {
	if len(sets) == 0 {
		return fmt.Errorf("mapedit: no field to set")
	}
	w, err := e.walkIndex(k, "set a field of", i)
	if err != nil {
		return err
	}
	at, l := w.at(i)
	rec := e.data[at : at+l]
	next := cloneBytes(rec)
	lo, hi := l, 0
	var extra []part
	seen := map[string]bool{}
	for _, set := range sets {
		f, ok := fieldByKey(w.sec.fields(e, next), set.Key)
		if !ok {
			return fmt.Errorf("mapedit: no %s field %q", w.sec.name, set.Key)
		}
		if f.Text != nil {
			return fmt.Errorf("mapedit: %s is a text field", f.Label)
		}
		if seen[set.Key] {
			return fmt.Errorf("mapedit: %s field %q is named twice", w.sec.name, set.Key)
		}
		seen[set.Key] = true
		if set.Value < f.Min || set.Value > f.Max {
			return fmt.Errorf("mapedit: %s %d is outside %d..%d", f.Label, set.Value, f.Min, f.Max)
		}
		if f.Offset >= len(next) && set.Value == 0 {
			return fmt.Errorf("mapedit: %s 0 adds nothing", f.Label)
		}
		if f.Check == CheckStructureIdentity {
			parts, err := e.structureIdentityParts(i, f.decode(next[f.Offset:]), set.Value)
			if err != nil {
				return err
			}
			extra = append(extra, parts...)
		}
		next = f.write(next, set.Value)
		if w.sec.reshape != nil {
			next = w.sec.reshape(next)
		}
		if w.sec.length(e, next) != len(next) {
			return fmt.Errorf("mapedit: %s %d would not be one type-%d record", f.Label, set.Value, w.sec.typeID)
		}
		lo, hi = min(lo, f.Offset), max(hi, f.Offset+f.Width)
	}
	if len(next) == l {
		e.apply(append(edit{replace(at+lo, rec[lo:hi], next[lo:hi])}, extra...))
		return nil
	}
	e.apply(append(edit{replace(at, rec, next), e.sizeWordPart(w.span, len(next)-l)}, extra...))
	return nil
}

// TextFieldValue reads a text field through its codec, stopping at its first NUL.
func (e *Editor) TextFieldValue(k Kind, i int, key string) (string, error) {
	rec, err := e.Record(k, i)
	if err != nil {
		return "", err
	}
	f, ok := fieldByKey(sections[k].fields(e, rec), key)
	if !ok || f.Text == nil {
		return "", fmt.Errorf("mapedit: no %s text field %q", sections[k].name, key)
	}
	return f.Text.Decode(rec[f.Offset : f.Offset+f.Width]), nil
}

// SetTextField writes one text field through its codec as one undo step.
func (e *Editor) SetTextField(k Kind, i int, key, value string) error {
	w, err := e.walkIndex(k, "set a text field of", i)
	if err != nil {
		return err
	}
	at, l := w.at(i)
	f, ok := fieldByKey(w.sec.fields(e, e.data[at:at+l]), key)
	if !ok || f.Text == nil {
		return fmt.Errorf("mapedit: no %s text field %q", w.sec.name, key)
	}
	if !utf8.ValidString(value) || strings.IndexByte(value, 0) >= 0 {
		return fmt.Errorf("mapedit: %s needs valid text without NUL", f.Label)
	}
	out, err := f.Text.Image(value)
	if err != nil {
		return fmt.Errorf("mapedit: %s: %w", f.Label, err)
	}
	p := at + f.Offset
	e.apply(edit{replace(p, e.data[p:p+f.Width], out)})
	return nil
}

// write returns a fresh copy of rec with v written into f: in place, as a
// new element at the record's end, or, for an element set to 0, without it.
func (f Field) write(rec []byte, v int64) []byte {
	out := cloneBytes(rec)
	if f.Element == 0 {
		f.put(out[f.Offset:], v)
		return out
	}
	count := binary.LittleEndian.Uint32(out[f.Counter:])
	switch {
	case f.Offset >= len(rec):
		if v == 0 {
			return out
		}
		el := make([]byte, f.Element)
		f.put(el, v)
		out = append(out, el...)
		count++
	case v == 0:
		out = append(out[:f.Offset], out[f.Offset+f.Element:]...)
		count--
	default:
		f.put(out[f.Offset:], v)
	}
	binary.LittleEndian.PutUint32(out[f.Counter:], count)
	return out
}

func (f Field) put(b []byte, v int64) {
	switch f.Width {
	case 1:
		b[0] = byte(v)
	case 2:
		binary.LittleEndian.PutUint16(b, uint16(v))
	default:
		binary.LittleEndian.PutUint32(b, uint32(v))
	}
}

func (f Field) decode(b []byte) int64 {
	switch {
	case f.Width == 1:
		return int64(b[0])
	case f.Width == 2 && f.Signed:
		return int64(int16(binary.LittleEndian.Uint16(b)))
	case f.Width == 2:
		return int64(binary.LittleEndian.Uint16(b))
	case f.Signed:
		return int64(int32(binary.LittleEndian.Uint32(b)))
	default:
		return int64(binary.LittleEndian.Uint32(b))
	}
}
