package synth

import (
	"encoding/binary"
	"fmt"
	"math"
)

// The eleven collections of the definition table, in the order the file writes
// them. A caller indexes DataBin.Rows with one of these.
//
// They are spelled here and not imported from the parser, like every other
// builder in this package: a fixture derived from the code under test proves
// only that the code agrees with itself. What keeps the two in step is the
// round-trip test beside this file, which parses what this writes.
const (
	DataBinShapes = iota
	DataBinMaterials
	DataBinMagic
	DataBinArmors
	DataBinShields
	DataBinWeapons
	DataBinMagicItems
	DataBinUnits
	DataBinHumans
	DataBinBuildings
	DataBinSpells

	// DataBinCollections is how many there are, derived from the last id so a
	// collection added above without a grammar row below fails to compile.
	DataBinCollections
)

// The three entry shapes the format defines. Which one a collection uses is a
// property of the collection, not of an entry.
const (
	// dbDoubles is a nine-double raw record and NO parameter array at all.
	dbDoubles = iota
	// dbParams is a parameter array, then the collection's raw run and its
	// fixed run of trailing strings.
	dbParams
	// dbParamsRawParams is a parameter array, a fixed raw block, then a second
	// parameter array.
	dbParamsRawParams
)

const (
	// dbDoublesPerRecord is how many doubles a shapes or materials record holds.
	dbDoublesPerRecord = 9
	// dbDoubleRecord is that record's byte length.
	dbDoubleRecord = dbDoublesPerRecord * 8
	// dbRawBlock is the raw run between an armour, shield or weapon entry's two
	// parameter arrays.
	dbRawBlock = 10
)

// dbGrammar is the file as data: for each collection, whether it opens its
// group — and so is preceded by the title array its siblings share — whether
// entry 0 is allocated and never written, its entry shape, the raw run after
// its parameter array and how many strings follow that.
//
// The string count is a FIXED RUN and not a counted array. The group title
// arrays ARE counted, and conflating the two is the mistake this format invites.
var dbGrammar = [DataBinCollections]struct {
	first    bool
	oneBased bool
	kind     int
	raw      int
	strings  int
}{
	DataBinShapes:     {true, false, dbDoubles, 0, 0},
	DataBinMaterials:  {false, false, dbDoubles, 0, 0},
	DataBinMagic:      {true, false, dbParams, 0, 0},
	DataBinArmors:     {true, true, dbParamsRawParams, dbRawBlock, 0},
	DataBinShields:    {false, true, dbParamsRawParams, dbRawBlock, 0},
	DataBinWeapons:    {false, true, dbParamsRawParams, dbRawBlock, 0},
	DataBinMagicItems: {true, true, dbParams, 1, 1},
	DataBinUnits:      {true, true, dbParams, 0, 2},
	DataBinHumans:     {true, true, dbParams, 0, 10},
	DataBinBuildings:  {true, true, dbParams, 0, 0},
	DataBinSpells:     {true, true, dbParams, 0, 1},
}

// DataBinRow is one written entry: its name, the parameter cells of its row,
// and — for a collection whose entries are nine doubles — those doubles.
//
// It carries no raw block and no trailing strings. The builder writes whatever
// the collection's grammar demands of those, zeroed — they are structure a
// stream must have to parse, and no test in this tree reads them.
type DataBinRow struct {
	Name   string
	Params []int32

	// Doubles is the nine-double record of a shapes or materials row. Fewer
	// than nine are written where they fall and the rest come out zero; more
	// than nine panics, because a fixture that silently dropped a slot would be
	// a fixture testing something else. Ignored by every other collection,
	// whose entries have no such record.
	Doubles []float64
}

// DataBin is a definition table to be written.
//
// Rows is indexed by collection and holds only the entries that are WRITTEN: for
// a one-based collection the builder emits a count one larger and leaves entry 0
// unwritten, so a caller's first row is the file's entry 1 and a subscript is the
// game's own index rather than that index minus one.
//
// A one-based collection with NO rows is written with a count of 0 rather than
// the 1 that would allocate only the reserved entry. Both describe a collection
// holding nothing, and the parser's guard — that a count above the bytes left
// cannot be satisfied — is written against the count rather than against the
// count minus one, so the second form is refused at the end of a stream where
// there are no bytes left to satisfy it. Every shipped collection has entries,
// so this edge is reachable from a fixture and from nothing else.
//
// Titles is written once per group, the same array for every group. The format
// gives each group its own; nothing in this tree reads one, and a single array
// keeps a fixture's intent — its rows — from being buried in eight of them.
type DataBin struct {
	Titles []string
	Rows   [DataBinCollections][]DataBinRow
}

// DataBinUnitsTable is the common fixture: a table whose Units and Humans
// collections hold the given rows and whose other nine are empty.
func DataBinUnitsTable(units, humans []DataBinRow) []byte {
	var d DataBin
	d.Rows[DataBinUnits] = units
	d.Rows[DataBinHumans] = humans
	return d.Bytes()
}

// Bytes writes the table. It panics rather than emit a stream no reader accepts:
// a string too long for the format's own length escape has nowhere to go, and a
// fixture that silently truncated one would be a fixture testing something else.
func (d DataBin) Bytes() []byte {
	var b []byte
	for id := 0; id < DataBinCollections; id++ {
		row := dbGrammar[id]
		if row.first {
			b = dbStrArray(b, d.Titles)
		}

		written := d.Rows[id]
		n := len(written)
		if row.oneBased && n > 0 {
			n++ // entry 0 is allocated and never written
		}
		b = binary.LittleEndian.AppendUint32(b, uint32(n))

		for _, e := range written {
			b = dbEntry(b, row.kind, row.raw, row.strings, e)
		}
	}
	return b
}

// dbEntry writes one entry: its name, then the body its collection's shape
// describes.
func dbEntry(b []byte, kind, raw, strs int, e DataBinRow) []byte {
	b = dbStr(b, e.Name)
	if kind == dbDoubles {
		// No parameter array and no strings: nine doubles, slot j at 8j.
		if len(e.Doubles) > dbDoublesPerRecord {
			panic(fmt.Sprintf("synth: a row of %d doubles does not fit the nine-double record",
				len(e.Doubles)))
		}
		rec := make([]byte, dbDoubleRecord)
		for j, v := range e.Doubles {
			binary.LittleEndian.PutUint64(rec[8*j:], math.Float64bits(v))
		}
		return append(b, rec...)
	}

	b = dbParamArray(b, e.Params)
	if kind == dbParamsRawParams {
		b = append(b, make([]byte, raw)...)
		return dbParamArray(b, nil)
	}
	b = append(b, make([]byte, raw)...)
	for i := 0; i < strs; i++ {
		b = dbStr(b, "")
	}
	return b
}

// dbParamArray writes a parameter array: a u16 count and that many little-endian
// dwords. A cell is written as the 32 bits it holds, so -1 — the format's empty
// cell — goes out as all ones.
func dbParamArray(b []byte, p []int32) []byte {
	if len(p) > 0xFFFF {
		panic(fmt.Sprintf("synth: a parameter array of %d cells does not fit a u16 count", len(p)))
	}
	b = binary.LittleEndian.AppendUint16(b, uint16(len(p)))
	for _, v := range p {
		b = binary.LittleEndian.AppendUint32(b, uint32(v))
	}
	return b
}

// dbStrArray writes a COUNTED array of strings: a u16 count and that many
// strings. Only a group's titles take this form.
func dbStrArray(b []byte, s []string) []byte {
	if len(s) > 0xFFFF {
		panic(fmt.Sprintf("synth: a title array of %d strings does not fit a u16 count", len(s)))
	}
	b = binary.LittleEndian.AppendUint16(b, uint16(len(s)))
	for _, v := range s {
		b = dbStr(b, v)
	}
	return b
}

// dbStr writes one string: a u8 length, escaping to a u16 at 0xFF.
//
// 0xFFFF is the escape the format does not define a wider form for, so a string
// that long cannot be written at all rather than being written as something a
// reader must guess at.
func dbStr(b []byte, s string) []byte {
	switch n := len(s); {
	case n < 0xFF:
		b = append(b, byte(n))
	case n < 0xFFFF:
		b = append(b, 0xFF)
		b = binary.LittleEndian.AppendUint16(b, uint16(n))
	default:
		panic(fmt.Sprintf("synth: a string of %d bytes reaches the undefined length escape", n))
	}
	return append(b, s...)
}
