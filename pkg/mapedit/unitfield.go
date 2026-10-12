package mapedit

import (
	"encoding/binary"
	"fmt"
	"math"
)

// UnitFieldCheck names the reference check a host runs on a set value beyond
// the field's own range. This package knows no installed table or roster, so
// it states which check a field needs and the host performs it.
type UnitFieldCheck int

const (
	CheckRangeOnly    UnitFieldCheck = iota // the range below is the whole rule
	CheckRosterSlot                         // a 1-based slot in the type-5 roster (ALM-OWN-039)
	CheckClassKey                           // a class key the installed tables hold (ALM-CLS-038)
	CheckDefinitionID                       // zero, or a definition id the installed tables hold
)

// UnitField is one settable integer field of the 70-byte type-6 record.
// Offsets are file offsets into the record (ALM-UNIT-040).
type UnitField struct {
	Key      string // stable name a caller addresses the field by
	Label    string // window text
	Offset   int
	Width    int // 2 or 4 bytes, little-endian
	Signed   bool
	Min, Max int64
	Check    UnitFieldCheck
}

// unitFields is the one table of type-6 fields a caller may set. A later
// field is a row here, not a new setter.
var unitFields = []UnitField{
	{Key: "owner", Label: "Owner slot", Offset: 0x14, Width: 4, Min: 1, Max: math.MaxUint32, Check: CheckRosterSlot},
	{Key: "class", Label: "Class", Offset: 0x08, Width: 2, Signed: true, Min: math.MinInt16, Max: math.MaxInt16, Check: CheckClassKey},
	{Key: "definition", Label: "Definition id", Offset: 0x10, Width: 4, Min: 0, Max: math.MaxUint32, Check: CheckDefinitionID},
	{Key: "hp", Label: "Hit points (-1 = default)", Offset: 0x20, Width: 2, Signed: true, Min: math.MinInt16, Max: math.MaxInt16},
}

// UnitFields returns a copy of the settable type-6 fields in window order.
func UnitFields() []UnitField { return append([]UnitField(nil), unitFields...) }

// UnitFieldByKey returns the field named key.
func UnitFieldByKey(key string) (UnitField, bool) {
	for _, f := range unitFields {
		if f.Key == key {
			return f, true
		}
	}
	return UnitField{}, false
}

// UnitFieldValue reads the field named key from the record at index i.
func (e *Editor) UnitFieldValue(i int, key string) (int64, error) {
	r, err := e.UnitRecord(i)
	if err != nil {
		return 0, err
	}
	f, ok := UnitFieldByKey(key)
	if !ok {
		return 0, fmt.Errorf("mapedit: no unit field %q", key)
	}
	return f.decode(r[f.Offset:]), nil
}

// SetUnitField writes v into the field named key of the record at index i as
// one edit, leaving the record's other bytes as they are. A value outside the
// field's range is refused and writes nothing; the field's Check is the
// caller's to run.
func (e *Editor) SetUnitField(i int, key string, v int64) error {
	f, ok := UnitFieldByKey(key)
	if !ok {
		return fmt.Errorf("mapedit: no unit field %q", key)
	}
	if v < f.Min || v > f.Max {
		return fmt.Errorf("mapedit: %s %d is outside %d..%d", f.Label, v, f.Min, f.Max)
	}
	fr := e.locate()
	units, err := fr.optional("set a unit field", unitTypeID)
	if err != nil {
		return err
	}
	if err := e.checkUnitIndex(fr, "set a field of", i); err != nil {
		return err
	}
	img := make([]byte, f.Width)
	if f.Width == 2 {
		binary.LittleEndian.PutUint16(img, uint16(v))
	} else {
		binary.LittleEndian.PutUint32(img, uint32(v))
	}
	at := units.abs(i*unitRecordSize + f.Offset)
	e.apply(edit{replace(at, e.data[at:at+f.Width], img)})
	return nil
}

func (f UnitField) decode(b []byte) int64 {
	switch {
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
