// Package databin_test exercises the walk from outside, over the exported
// surface alone.
//
// Everything here is SYNTHETIC: every stream is built byte by byte in this file
// from the documented wire primitives, so the suite opens no file, reads no game
// install and needs no archive. The import block is the whole evidence for that
// claim.
package databin_test

import (
	"bytes"
	"encoding/binary"
	"reflect"
	"strings"
	"testing"

	"againrom/pkg/formats/databin"
)

// wire writes the file's own primitives. It is the fixture builder, and it is
// deliberately a WRITER rather than a table of pre-baked bytes: a hand-typed hex
// blob would have to be re-typed for every malformed variant below, and the one
// that got re-typed wrong would look like a parser defect.
type wire struct{ b []byte }

func (w *wire) u8(v byte)     { w.b = append(w.b, v) }
func (w *wire) raw(b []byte)  { w.b = append(w.b, b...) }
func (w *wire) u16(v uint16)  { w.b = binary.LittleEndian.AppendUint16(w.b, v) }
func (w *wire) u32(v uint32)  { w.b = binary.LittleEndian.AppendUint32(w.b, v) }
func (w *wire) i32(v int32)   { w.u32(uint32(v)) }
func (w *wire) bytes() []byte { return append([]byte(nil), w.b...) }

// str writes a length-prefixed string, taking the u16 escape at 0xFF and above.
// 0xFF itself cannot be spelled in the byte form, since that byte IS the escape.
func (w *wire) str(s string) {
	if len(s) >= 0xFF {
		w.u8(0xFF)
		w.u16(uint16(len(s)))
	} else {
		w.u8(byte(len(s)))
	}
	w.raw([]byte(s))
}

// strArray writes a COUNTED array — the group title arrays' shape.
func (w *wire) strArray(ss []string) {
	w.u16(uint16(len(ss)))
	for _, s := range ss {
		w.str(s)
	}
}

// strs writes a fixed run of strings with NO count word — an entry's own
// trailing strings. The two writers exist side by side on purpose: the fixture
// has to be able to spell both, or a walk that confused them would be tested
// against a stream that made the same mistake.
func (w *wire) strs(ss []string) {
	for _, s := range ss {
		w.str(s)
	}
}

func (w *wire) params(p []int32) {
	w.u16(uint16(len(p)))
	for _, v := range p {
		w.i32(v)
	}
}

// The fixture's own values, named so the assertions read them back rather than
// re-typing them: what is asserted is that the walk returns WHAT WAS WRITTEN.
// The structural claims — index 0 of a one-based collection, the first written
// entry's index, the residue — are written out literally instead.
var (
	titlesA = []string{"colA0", "colA1"}
	titlesB = []string{"colB0"}
	titlesC = []string{"colC0", "colC1"}
	titlesD = []string{"colD0"}
	titlesE = []string{"colE0"}
	titlesF = []string{"colF0"}
	titlesG = []string{"colG0"}
	titlesH = []string{"colH0"}

	shape0Raw = bytes.Repeat([]byte{0x11}, 72)
	shape1Raw = bytes.Repeat([]byte{0x22}, 72)
	matRaw    = bytes.Repeat([]byte{0x33}, 72)
	armorRaw  = []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}
	shieldRaw = bytes.Repeat([]byte{0xAB}, 10)
	itemRaw   = []byte{0x7F}

	// A name past the u8 length, so the escape is exercised on a value the
	// single byte cannot hold.
	longName = strings.Repeat("N", 300)

	magic0Params = []int32{1, -1, 3}
	armorParams  = []int32{7, -1}
	armorParams2 = []int32{8, 9}
	shieldParams = []int32{5}
	itemParams   = []int32{-1}
	unitParams   = []int32{
		30, 31, 32, 33, 34, 35, 36, 37, 38, 39,
		40, 41, 42, 43, 44, 45, 46, 47, 48, 49,
		50, 51, 52, 53, 54, 55, 56, 57, 58, 59,
		60, 61, 62, 63, 64, 65, 66, 67,
	}
	humanParams      = []int32{-1, 2, 3, 4, 5}
	buildingParams   = []int32{3, 2, 4, 300, 0, 0}
	spellParams      = []int32{1, 2}
	unitStrings      = []string{"eq", "sp"}
	emptyUnitStrings = []string{"", ""}
	humanStrings     = []string{"a", "b", "c", "d", "e", "f", "g", "h", "i", "j"}
	magicItemString  = "wand"
	spellString      = "fire"
)

// goodStream writes one stream covering all eight groups, every entry kind, a
// 0-based and a 1-based collection, an empty parameter array, an empty name and
// a name at the length escape.
func goodStream() []byte {
	w := &wire{}

	// Group A — the two collections with no parameter array at all, 0-based.
	w.strArray(titlesA)
	w.u32(2) // Shapes
	w.str("Sh0")
	w.raw(shape0Raw)
	w.str("Sh1")
	w.raw(shape1Raw)
	w.u32(1) // Materials
	w.str("Mt0")
	w.raw(matRaw)

	// Group B — parameters alone, 0-based. The second entry's array is present
	// and empty, which is not the same as absent.
	w.strArray(titlesB)
	w.u32(2) // Magic
	w.str("Mg0")
	w.params(magic0Params)
	w.str("Mg1")
	w.params(nil)

	// Group C — two parameter arrays around a raw block, 1-based, three
	// collections sharing one title array. Weapons' count is 1, so it has the
	// reserved entry 0 and nothing else.
	w.strArray(titlesC)
	w.u32(2) // Armors: entry 1 only
	w.str(longName)
	w.params(armorParams)
	w.raw(armorRaw)
	w.params(armorParams2)
	w.u32(2) // Shields: entry 1 only
	w.str("Sd1")
	w.params(shieldParams)
	w.raw(shieldRaw)
	w.params(nil)
	w.u32(1) // Weapons: nothing written

	// Group D — parameters, one raw byte, one string.
	w.strArray(titlesD)
	w.u32(2) // MagicItems
	w.str("Mi1")
	w.params(itemParams)
	w.raw(itemRaw)
	w.str(magicItemString)

	// Group E — parameters and TWO strings written back to back, with no count
	// word: the field is sized by the class, so the serializer has none to
	// write. The title array above IS counted, and telling the two apart is the
	// difference between tiling this file and mis-reading everything after it.
	// Entry 2 is empty-named with an empty array and two empty strings, which is
	// the shape a search has to skip.
	w.strArray(titlesE)
	w.u32(3) // Units
	w.str("Un1")
	w.params(unitParams)
	w.strs(unitStrings)
	w.str("")
	w.params(nil)
	w.strs(emptyUnitStrings)

	// Group F — the same shape with ten strings.
	w.strArray(titlesF)
	w.u32(2) // Humans
	w.str("Hu1")
	w.params(humanParams)
	w.strs(humanStrings)

	// Group G — parameters alone, 1-based this time.
	w.strArray(titlesG)
	w.u32(2) // Buildings
	w.str("Bl1")
	w.params(buildingParams)

	// Group H — parameters and one string.
	w.strArray(titlesH)
	w.u32(2) // Spells
	w.str("Sp1")
	w.params(spellParams)
	w.str(spellString)

	return w.bytes()
}

// TestParseReadsBackWhatWasWritten is AC-1: every group, every entry kind, the
// shared title arrays, the two counting rules, and no byte left over.
func TestParseReadsBackWhatWasWritten(t *testing.T) {
	in := goodStream()
	f, err := databin.Parse(in)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	if f.Consumed != len(in) {
		t.Errorf("the walk consumed %d of %d byte(s); the file must tile exactly", f.Consumed, len(in))
	}

	// The titles a group's collections share are ONE array, so the siblings must
	// come back equal — this is the structural fact a per-collection title read
	// would get wrong while still looking right on the first collection.
	for _, tc := range []struct {
		id   databin.ID
		want []string
	}{
		{databin.Shapes, titlesA}, {databin.Materials, titlesA},
		{databin.Magic, titlesB},
		{databin.Armors, titlesC}, {databin.Shields, titlesC}, {databin.Weapons, titlesC},
		{databin.MagicItems, titlesD},
		{databin.Units, titlesE}, {databin.Humans, titlesF},
		{databin.Buildings, titlesG}, {databin.Spells, titlesH},
	} {
		if got := f.Collection(tc.id).Titles; !reflect.DeepEqual(got, tc.want) {
			t.Errorf("%v titles = %q, want %q", tc.id, got, tc.want)
		}
	}

	// The 0-based collections: every entry the count names is written, so entry
	// 0 carries a name.
	shapes := f.Collection(databin.Shapes)
	if shapes.OneBased {
		t.Error("Shapes reports one-based; group A writes every entry the count names")
	}
	if got := shapes.Len(); got != 2 {
		t.Fatalf("Shapes holds %d entries, want 2", got)
	}
	if got := shapes.Entries[0].Name; got != "Sh0" {
		t.Errorf("Shapes[0].Name = %q, want %q — a 0-based collection writes index 0", got, "Sh0")
	}
	if got := shapes.Entries[1].Raw; !bytes.Equal(got, shape1Raw) {
		t.Errorf("Shapes[1].Raw = % x, want % x", got, shape1Raw)
	}
	if got := shapes.Entries[0].Params; got != nil {
		t.Errorf("Shapes[0].Params = %v, want nil — this kind has no parameter array at all", got)
	}
	if got := f.Collection(databin.Materials).Entries[0].Raw; !bytes.Equal(got, matRaw) {
		t.Errorf("Materials[0].Raw = % x, want % x", got, matRaw)
	}

	// An empty parameter array is present, not absent.
	magic := f.Collection(databin.Magic)
	if got := magic.Entries[0].Params; !reflect.DeepEqual(got, magic0Params) {
		t.Errorf("Magic[0].Params = %v, want %v", got, magic0Params)
	}
	if got := magic.Entries[1].Params; got == nil || len(got) != 0 {
		t.Errorf("Magic[1].Params = %v, want an empty non-nil array", got)
	}

	// The 1-based collections: index 0 is present and empty, and the first
	// written entry is at index 1.
	armors := f.Collection(databin.Armors)
	if !armors.OneBased {
		t.Error("Armors does not report one-based; group C skips entry 0")
	}
	if got := armors.Len(); got != 2 {
		t.Fatalf("Armors holds %d entries, want 2 (the reserved 0 and one written)", got)
	}
	if got := armors.Entries[0]; !reflect.DeepEqual(got, databin.Entry{}) {
		t.Errorf("Armors[0] = %+v, want the empty entry", got)
	}
	if got := armors.Entries[1].Name; got != longName {
		t.Errorf("Armors[1].Name has length %d, want %d — the u16 length escape", len(got), len(longName))
	}
	if got := armors.Entries[1].Params; !reflect.DeepEqual(got, armorParams) {
		t.Errorf("Armors[1].Params = %v, want %v", got, armorParams)
	}
	if got := armors.Entries[1].Raw; !bytes.Equal(got, armorRaw) {
		t.Errorf("Armors[1].Raw = % x, want % x", got, armorRaw)
	}
	if got := armors.Entries[1].Params2; !reflect.DeepEqual(got, armorParams2) {
		t.Errorf("Armors[1].Params2 = %v, want %v — a second array this contract gives no meaning to is still handed over", got, armorParams2)
	}
	if got := f.Collection(databin.Shields).Entries[1].Params; !reflect.DeepEqual(got, shieldParams) {
		t.Errorf("Shields[1].Params = %v, want %v", got, shieldParams)
	}

	// A one-based collection whose count is 1 holds the reserved entry alone.
	weapons := f.Collection(databin.Weapons)
	if got := weapons.Len(); got != 1 {
		t.Errorf("Weapons holds %d entries, want 1 (the reserved 0 alone)", got)
	} else if !reflect.DeepEqual(weapons.Entries[0], databin.Entry{}) {
		t.Errorf("Weapons[0] = %+v, want the empty entry", weapons.Entries[0])
	}

	items := f.Collection(databin.MagicItems)
	if got := items.Entries[1].Raw; !bytes.Equal(got, itemRaw) {
		t.Errorf("MagicItems[1].Raw = % x, want % x", got, itemRaw)
	}
	if got := items.Entries[1].Strings; !reflect.DeepEqual(got, []string{magicItemString}) {
		t.Errorf("MagicItems[1].Strings = %q, want [%q]", got, magicItemString)
	}

	units := f.Collection(databin.Units)
	if got := units.EntryName(1); got != "Un1" {
		t.Errorf("Units[1] name = %q, want %q", got, "Un1")
	}
	if got := units.EntryParams(1); !reflect.DeepEqual(got, unitParams) {
		t.Errorf("Units[1] params = %v, want %v", got, unitParams)
	}
	if got := units.Entries[1].Strings; !reflect.DeepEqual(got, unitStrings) {
		t.Errorf("Units[1].Strings = %q, want %q", got, unitStrings)
	}
	if got := units.EntryName(2); got != "" {
		t.Errorf("Units[2] name = %q, want the empty name", got)
	}
	// An entry that NAMES nothing still carries its grammar's own count of
	// cells, each empty — the tail is a fixed shape and not a list that shrinks —
	// while the reserved entry a one-based collection never writes carries none.
	if got := units.EntryStrings(2); !reflect.DeepEqual(got, []string{"", ""}) {
		t.Errorf("Units EntryStrings(2) = %q, want two empty cells", got)
	}
	if got := units.EntryStrings(0); got != nil {
		t.Errorf("Units EntryStrings(0) = %q, want nil for the reserved entry", got)
	}

	humans := f.Collection(databin.Humans)
	if got := humans.Entries[1].Strings; !reflect.DeepEqual(got, humanStrings) {
		t.Errorf("Humans[1].Strings = %q, want %q", got, humanStrings)
	}
	// The ACCESSOR, beside the field it reads: a consuming tier reaches the
	// trailing strings through the interface and never through Entries, so the
	// two have to be witnessed answering alike on one entry.
	if got := humans.EntryStrings(1); !reflect.DeepEqual(got, humanStrings) {
		t.Errorf("Humans EntryStrings(1) = %q, want %q", got, humanStrings)
	}
	if got := humans.EntryParams(1); !reflect.DeepEqual(got, humanParams) {
		t.Errorf("Humans[1] params = %v, want %v", got, humanParams)
	}
	if got := f.Collection(databin.Buildings).EntryParams(1); !reflect.DeepEqual(got, buildingParams) {
		t.Errorf("Buildings[1] params = %v, want %v", got, buildingParams)
	}
	if got := f.Collection(databin.Spells).Entries[1].Strings; !reflect.DeepEqual(got, []string{spellString}) {
		t.Errorf("Spells[1].Strings = %q, want [%q]", got, spellString)
	}
}

// emptyGroup writes a group with no titles and one zero-count collection per
// sibling, which is the shortest valid form of a group.
func emptyGroup(w *wire, collections int) {
	w.strArray(nil)
	for i := 0; i < collections; i++ {
		w.u32(0)
	}
}

// TestParseRefusesMalformedStreams is AC-2: five malformed streams, each refused
// with a located error and no table returned.
func TestParseRefusesMalformedStreams(t *testing.T) {
	// A parameter count past the end, inside group B's first entry.
	paramPastEnd := func() []byte {
		w := &wire{}
		emptyGroup(w, 2) // group A
		w.strArray(titlesB)
		w.u32(1)
		w.str("Mg0")
		w.u16(0xFFFF) // a count nothing behind it can satisfy
		return w.bytes()
	}()

	// An unsatisfiable collection count, at group A's first collection.
	countPastEnd := func() []byte {
		w := &wire{}
		w.strArray(nil)
		w.u32(0xFFFFFFFF)
		return w.bytes()
	}()

	// A string whose length runs past the end, in group A's title array.
	stringPastEnd := func() []byte {
		w := &wire{}
		w.u16(1) // one title
		w.u8(10) // saying ten bytes
		w.raw([]byte("ab"))
		return w.bytes()
	}()

	// A length escape the published primitive does not define.
	undefinedEscape := func() []byte {
		w := &wire{}
		w.u16(1) // one title
		w.u8(0xFF)
		w.u16(0xFFFF)
		return w.bytes()
	}()

	// One byte past the last group.
	trailing := append(goodStream(), 0x00)

	for _, tc := range []struct {
		name string
		in   []byte
		want string // a substring the message must carry, so it says WHERE
	}{
		{"a parameter count past the end", paramPastEnd, "group B Magic entry 0"},
		{"an unsatisfiable collection count", countPastEnd, "group A Shapes"},
		{"a string past the end", stringPastEnd, "group A Shapes"},
		{"an undefined length escape", undefinedEscape, "escape"},
		{"one trailing byte", trailing, "left over"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f, err := databin.Parse(tc.in)
			if err == nil {
				t.Fatalf("Parse accepted %s", tc.name)
			}
			if f != nil {
				t.Errorf("Parse returned a table beside its error; a refusal must yield none")
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Errorf("error %q does not name where: want a mention of %q", err, tc.want)
			}
		})
	}
}

func TestParseIsAFunctionOfItsBytes(t *testing.T) {
	in := goodStream()

	first, err := databin.Parse(in)
	if err != nil {
		t.Fatalf("first Parse: %v", err)
	}
	second, err := databin.Parse(in)
	if err != nil {
		t.Fatalf("second Parse: %v", err)
	}
	if !reflect.DeepEqual(first, second) {
		t.Error("two parses of one stream differ; the walk depends on something other than its input")
	}

	scribbled := append([]byte(nil), in...)
	kept, err := databin.Parse(scribbled)
	if err != nil {
		t.Fatalf("Parse of the copy: %v", err)
	}
	for i := range scribbled {
		scribbled[i] = 0xEE
	}
	if !reflect.DeepEqual(kept, first) {
		t.Error("overwriting the input changed the parsed table; something in it aliases the input")
	}
}

// TestCollectionIDsAreTheFileOrder pins that every id names itself and that the
// walk visits them in the order the ids are numbered — the property that makes
// "index it with an ID" true rather than a convention.
func TestCollectionIDsAreTheFileOrder(t *testing.T) {
	f, err := databin.Parse(goodStream())
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if got := len(f.Collections); got != databin.NumCollections {
		t.Fatalf("parsed %d collections, want %d", got, databin.NumCollections)
	}
	for i := range f.Collections {
		if got := int(f.Collections[i].ID); got != i {
			t.Errorf("Collections[%d].ID = %d; a collection must sit at its own id", i, got)
		}
	}
	if got := databin.Units.String(); got != "Units" {
		t.Errorf("Units.String() = %q, want %q", got, "Units")
	}
	if f.Collection(databin.ID(-1)) != nil || f.Collection(databin.ID(databin.NumCollections)) != nil {
		t.Error("Collection accepted an id outside the file's own set")
	}
}
