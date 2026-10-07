package databin

import (
	"encoding/binary"
	"fmt"
	"math"
)

// ID names one collection. The order is the file's own: the eight groups in
// serialize order, and inside a group its collections in the order they are
// written. An ID is therefore both a name and a position in the walk.
type ID int

// The eleven collections, in file order.
const (
	Shapes ID = iota
	Materials
	Magic
	Armors
	Shields
	Weapons
	MagicItems
	Units
	Humans
	Buildings
	Spells
)

// NumCollections is how many collections the file holds. It is derived from the
// last id rather than written out, so the grammar table below cannot fall out of
// step with the constants above without failing to compile.
const NumCollections = int(Spells) + 1

var collectionNames = [NumCollections]string{
	Shapes: "Shapes", Materials: "Materials", Magic: "Magic",
	Armors: "Armors", Shields: "Shields", Weapons: "Weapons",
	MagicItems: "MagicItems", Units: "Units", Humans: "Humans",
	Buildings: "Buildings", Spells: "Spells",
}

// String is the collection's own table name, which is what an error message and
// a census report call it. It names a table, never a column.
func (id ID) String() string {
	if id < 0 || int(id) >= NumCollections {
		return fmt.Sprintf("collection(%d)", int(id))
	}
	return collectionNames[id]
}

// payload is the shape of an entry's body after its name. Entry sizes differ by
// class, and this is the whole of that difference.
type payload int

const (
	// nine raw doubles and NO parameter array at all
	payloadDoubles payload = iota
	// a parameter array, then the row's own raw run and trailing strings
	payloadParams
	// a parameter array, a fixed raw block, then a second parameter array
	payloadParamsRawParams
)

const (
	// doubleRecord is the nine-double record's byte length.
	//
	// The bytes are kept whole and handed over as they arrived; EntryDoubles
	// reads them as numbers for a caller that wants one. Which slot means what
	// is the CONSUMER's business and not this package's — this file knows only
	// that there are nine of them and where they start.
	doubleRecord = 9 * 8
	// doublesPerRecord is that count, so a reader states the bound rather than
	// dividing the byte length by eight at every use.
	doublesPerRecord = 9
	// rawBlock is the raw run an armour, shield or weapon entry carries between
	// its two parameter arrays.
	rawBlock = 10
)

// collectionGrammar is one row of the file's grammar: which group a collection
// belongs to, whether it opens that group (and so is preceded by the titles its
// siblings share), which of the two counting rules it follows, and what shape
// its entries take.
type collectionGrammar struct {
	id       ID
	group    string // the group's letter, so an error says which one
	first    bool   // opens its group: the shared title array is read here
	oneBased bool   // entry 0 is allocated and never written
	kind     payload
	raw      int // raw bytes after the parameter array, before the strings
	// strings is how many trailing strings the entry writes, and it is a FIXED
	// COUNT and not a counted array: the field is sized by the class, so its
	// elements are written back to back with no count word of their own. The
	// group title arrays ARE counted, which is what makes the two easy to
	// conflate and what made this the story's declared risk.
	strings int
}

// grammar is the file, as data. The walk is one loop over this table and one
// switch on the payload kind: there is no per-collection branch and no
// per-group function, so the group order, the two counting rules and the
// shared title arrays are each written down exactly once.
//
// Its type is fixed at NumCollections entries, which is what makes a collection
// added to the id list without a row here a compile error rather than a silent
// short walk.
var grammar = [NumCollections]collectionGrammar{
	{Shapes, "A", true, false, payloadDoubles, 0, 0},
	{Materials, "A", false, false, payloadDoubles, 0, 0},
	{Magic, "B", true, false, payloadParams, 0, 0},
	{Armors, "C", true, true, payloadParamsRawParams, rawBlock, 0},
	{Shields, "C", false, true, payloadParamsRawParams, rawBlock, 0},
	{Weapons, "C", false, true, payloadParamsRawParams, rawBlock, 0},
	{MagicItems, "D", true, true, payloadParams, 1, 1},
	{Units, "E", true, true, payloadParams, 0, 2},
	{Humans, "F", true, true, payloadParams, 0, 10},
	{Buildings, "G", true, true, payloadParams, 0, 0},
	{Spells, "H", true, true, payloadParams, 0, 1},
}

// Entry is one row of one collection, verbatim.
//
// Which fields carry meaning follows from the collection, not from the entry: a
// shapes entry has no parameter array at all and a units entry has no second
// one. A field the collection's grammar does not write is nil, which is why nil
// and empty are kept apart everywhere below.
type Entry struct {
	// Name is the entry's name bytes as the file holds them. NO CHARACTER
	// ENCODING IS APPLIED — the format defines none — so this is not guaranteed
	// to be valid UTF-8, and whatever displays one chooses and states its own
	// convention.
	Name string

	// Params is the entry's parameter array, one int32 per stored cell, in
	// column order: slot i is the collection's column i+1. An empty cell is
	// stored as -1 and arrives as -1, which is what makes the skip test the
	// obvious comparison in the tier that consumes it. Empty-but-non-nil for an
	// array that is present and zero-length; nil where the grammar writes none.
	Params []int32

	// Raw is the entry's undecoded record bytes — the nine doubles, the block
	// between an armour's two arrays, or a magic item's single byte. It is a
	// copy: a parsed file shares no memory with the input.
	Raw []byte

	// Strings is the entry's trailing strings in file order: two for a unit, ten
	// for a human, one for a magic item or a spell. The run is a FIXED COUNT and
	// carries no count word of its own, unlike the group title arrays — so its
	// length here is the class's, always, and never a number read off the wire.
	// nil where the grammar writes none.
	Strings []string

	// Params2 is the second parameter array an armour, shield or weapon entry
	// carries after its raw block. This contract gives it no meaning and hands
	// it over whole rather than dropping it. nil elsewhere.
	Params2 []int32
}

// Collection is one by-value table: its entries at their own indices, and the
// column titles of the group it belongs to.
type Collection struct {
	ID ID

	// Titles are the group's column titles, and a group's collections SHARE one
	// array — armors, shields and weapons read the same titles, and so do shapes
	// and materials. They are handed over as read; nothing here maps a title to
	// a slot.
	Titles []string

	// OneBased reports that entry 0 was allocated and never written, so entries
	// run 1 to len(Entries)-1 and index 0 is present and empty. A subscript is
	// then the game's own index rather than that index minus one.
	OneBased bool

	// Entries holds every entry at its own index, including the reserved index 0
	// of a one-based collection.
	Entries []Entry
}

// Len, EntryName and EntryParams are the whole of what a search over a
// collection needs. They exist so a consuming tier can read a collection through
// an interface of its own declaring, without importing this package and without
// a copy of the entries being made to cross the boundary.
func (c *Collection) Len() int { return len(c.Entries) }

// EntryName is entry i's name bytes.
func (c *Collection) EntryName(i int) string { return c.Entries[i].Name }

// EntryParams is entry i's parameter array, verbatim, sentinels included.
func (c *Collection) EntryParams(i int) []int32 { return c.Entries[i].Params }

// EntryStrings is entry i's trailing strings, verbatim and in file order.
//
// They are the entry's own tail rather than a second array beside it: a row's
// grammar states how many follow its parameters, and this returns exactly that
// many. A cell an entry leaves empty comes back as the empty string and is NOT
// dropped, so the i'th string here is the i'th string of the row and a consumer
// counting positions is counting the file's. An entry that names nothing at all
// therefore answers a run of empty strings rather than a shorter slice, and only
// an entry the parse never wrote — the reserved index 0 of a one-based
// collection — answers nil.
//
// It returns the slice the parse built rather than a copy. That is the same
// trade EntryParams makes and for the same reason — a string is immutable, so
// the only thing a caller can reach through the slice is the slice's own
// backing, which no consumer of this interface writes to.
func (c *Collection) EntryStrings(i int) []string { return c.Entries[i].Strings }

// EntryDoubles is entry i's nine doubles, in slot order, or nil for an entry
// whose collection writes none.
//
// The record is nine little-endian float64 laid end to end, and slot j is the
// j'th of them. That is the whole of the decode: the record the game holds is
// larger than what is serialized here — a head, then these nine — and slot j
// sits at the head's own length plus eight times j, so the serialized bytes
// begin exactly at slot 0 and no offset of ours is involved.
//
// IT IS A FRESH SLICE, not a view onto the entry's bytes, so a caller cannot
// reach the parsed file through it. Nine values is small enough that copying is
// cheaper to reason about than the alias would be.
//
// It returns nil, rather than panicking or padding, for an entry that carries
// no such record — the same nil-versus-empty distinction Params keeps — and for
// a record short of nine, which Parse cannot produce and a hand-built Entry can.
func (c *Collection) EntryDoubles(i int) []float64 {
	raw := c.Entries[i].Raw
	if len(raw) < doubleRecord {
		return nil
	}
	out := make([]float64, doublesPerRecord)
	for j := range out {
		out[j] = math.Float64frombits(binary.LittleEndian.Uint64(raw[8*j:]))
	}
	return out
}

// EntryRaw is entry i's undecoded record bytes: the nine doubles of a shape or
// a material, the ten-byte block an armour, shield or weapon entry carries
// between its two parameter arrays, or a magic item's single byte. nil for an
// entry whose collection writes none.
//
// IT IS A COPY, so a caller cannot reach the parsed file through it. That is the
// same discipline EntryDoubles keeps and for the same reason; a raw block is a
// handful of bytes and copying it costs less than reasoning about the alias.
//
// This package still interprets nothing. What the block holds is the consuming
// tier's business — pkg/data reads an armour, shield or weapon block as the five
// 16-bit material masks that class carries, one per tier.
func (c *Collection) EntryRaw(i int) []byte {
	raw := c.Entries[i].Raw
	if len(raw) == 0 {
		return nil
	}
	out := make([]byte, len(raw))
	copy(out, raw)
	return out
}

// File is a parsed table.
type File struct {
	// Collections holds the eleven collections at their own ids: index it with
	// an ID, never with a position of your own counting.
	Collections []Collection

	// Consumed is how many bytes the walk read. It equals the input's length on
	// every file Parse returns — residue is an error, not slack — and is exposed
	// so a report can print the walk's own cursor against the node's size rather
	// than printing one number twice.
	Consumed int
}

// Collection returns the collection at id, or nil if id names none.
func (f *File) Collection(id ID) *Collection {
	if id < 0 || int(id) >= len(f.Collections) {
		return nil
	}
	return &f.Collections[id]
}

// Wire primitives. A string is a u8 length that escapes to a u16 at 0xFF; a
// string array and a parameter array are both counted by a u16; a collection
// count is a plain u32.
const (
	stringEscape = 0xFF
	// undefinedEscape is the u16 length that would escape again, to a wider form
	// the published primitive does not define. It is REFUSED rather than
	// guessed: a longer string may well exist in some file, and inventing its
	// encoding here would be this parser asserting a grammar it has no source
	// for.
	undefinedEscape = 0xFFFF
)

// Parse walks data and returns the eleven collections.
//
// The walk consumes the payload EXACTLY. A length past the end, an unsatisfiable
// count, an undefined string-length escape and any byte left over after the last
// group are each an error naming the group, the collection, the entry and the
// offset it was found at. A malformed stream yields a nil *File and never a
// partially filled table: there is no half-read collection to mistake for a
// short one.
//
// Parse retains no reference to data. Names, strings, parameters and raw blocks
// are all copied out during the walk, so a caller may reuse or modify the slice
// afterwards.
func Parse(data []byte) (*File, error) { return ParseWith(data, ROM1Layout) }

// Layout holds the widths that differ between the games' files.
type Layout struct {
	// ArmorRaw is the raw run between the two parameter arrays of an armour,
	// shield or weapon entry.
	ArmorRaw int
}

// The two known layouts.
var (
	ROM1Layout = Layout{ArmorRaw: rawBlock}
	ROM2Layout = Layout{ArmorRaw: 14}
)

// ParseWith is Parse over the given layout.
func ParseWith(data []byte, layout Layout) (*File, error) {
	c := &cursor{b: data, entry: -1}
	f := &File{Collections: make([]Collection, NumCollections)}

	// titles carries the current group's array across its collections: a group's
	// collections share one array, so it is read at the group's first collection
	// and handed to each of them.
	var titles []string

	for _, row := range grammar {
		c.group, c.coll, c.entry = row.group, row.id.String(), -1

		if row.first {
			var err error
			if titles, err = c.strArray(); err != nil {
				return nil, err
			}
		}

		n, err := c.u32()
		if err != nil {
			return nil, err
		}
		// Every written entry costs at least its own name length byte, so a
		// count above the bytes left cannot be satisfied however the entries are
		// shaped. Testing it before allocating is what keeps a corrupt count
		// word from becoming an allocation instead of an error.
		if uint64(n) > uint64(c.remaining()) {
			return nil, c.errf("collection count %d cannot be satisfied by the %d byte(s) left", n, c.remaining())
		}

		col := Collection{ID: row.id, Titles: titles, OneBased: row.oneBased, Entries: make([]Entry, n)}
		first := 0
		if row.oneBased {
			first = 1 // entry 0 is allocated and never written
		}
		for i := first; i < int(n); i++ {
			c.entry = i
			if row.kind == payloadParamsRawParams {
				row.raw = layout.ArmorRaw
			}
			e, err := c.readEntry(row)
			if err != nil {
				return nil, err
			}
			col.Entries[i] = e
		}
		c.entry = -1
		f.Collections[row.id] = col
	}

	if c.remaining() != 0 {
		return nil, fmt.Errorf("databin: %d byte(s) left over after the last group, at +%#x of %d",
			c.remaining(), c.off, len(data))
	}
	f.Consumed = c.off
	return f, nil
}

// readEntry reads one entry: the name, then the body its collection's row
// describes.
func (c *cursor) readEntry(row collectionGrammar) (Entry, error) {
	var e Entry
	var err error
	if e.Name, err = c.str(); err != nil {
		return Entry{}, err
	}
	if row.kind == payloadDoubles {
		// No parameter array at all, and no strings: nine doubles, kept raw
		// because their slots are undecoded.
		if e.Raw, err = c.raw(doubleRecord); err != nil {
			return Entry{}, err
		}
		return e, nil
	}
	if e.Params, err = c.params(); err != nil {
		return Entry{}, err
	}
	if row.raw > 0 {
		if e.Raw, err = c.raw(row.raw); err != nil {
			return Entry{}, err
		}
	}
	if row.kind == payloadParamsRawParams {
		if e.Params2, err = c.params(); err != nil {
			return Entry{}, err
		}
	}
	// A FIXED COUNT of strings, written back to back with no count word: the
	// field is sized by the class, so the serializer has no count to write.
	if row.strings > 0 {
		e.Strings = make([]string, row.strings)
		for i := range e.Strings {
			if e.Strings[i], err = c.str(); err != nil {
				return Entry{}, err
			}
		}
	}
	return e, nil
}

// cursor is one bounds-checked walk of the payload. Every primitive fails
// rather than reslicing past the end, and the cursor carries the group, the
// collection and the entry it is inside, so every error names where it was.
type cursor struct {
	b     []byte
	off   int
	group string
	coll  string
	entry int // -1 when the cursor is not inside an entry
}

func (c *cursor) remaining() int { return len(c.b) - c.off }

func (c *cursor) errf(format string, a ...any) error {
	where := "group " + c.group + " " + c.coll
	if c.entry >= 0 {
		where = fmt.Sprintf("%s entry %d", where, c.entry)
	}
	return fmt.Errorf("databin: %s at +%#x: %s", where, c.off, fmt.Sprintf(format, a...))
}

// take advances over n bytes and returns them as a window into the input; every
// caller that keeps them copies first.
func (c *cursor) take(n int) ([]byte, error) {
	if n < 0 || n > c.remaining() {
		return nil, c.errf("wants %d byte(s), %d left", n, c.remaining())
	}
	b := c.b[c.off : c.off+n]
	c.off += n
	return b, nil
}

func (c *cursor) u8() (byte, error) {
	b, err := c.take(1)
	if err != nil {
		return 0, err
	}
	return b[0], nil
}

func (c *cursor) u16() (uint16, error) {
	b, err := c.take(2)
	if err != nil {
		return 0, err
	}
	return binary.LittleEndian.Uint16(b), nil
}

func (c *cursor) u32() (uint32, error) {
	b, err := c.take(4)
	if err != nil {
		return 0, err
	}
	return binary.LittleEndian.Uint32(b), nil
}

// raw copies n bytes out of the input, so a parsed file shares no memory with
// the slice it was parsed from.
func (c *cursor) raw(n int) ([]byte, error) {
	b, err := c.take(n)
	if err != nil {
		return nil, c.errf("raw record of %d byte(s) runs past the end (%d left)", n, c.remaining())
	}
	return append([]byte(nil), b...), nil
}

// str reads one length-prefixed string. Converting the bytes to a string copies
// them, so the result does not alias the input.
func (c *cursor) str() (string, error) {
	n, err := c.u8()
	if err != nil {
		return "", err
	}
	length := int(n)
	if n == stringEscape {
		w, werr := c.u16()
		if werr != nil {
			return "", werr
		}
		if w == undefinedEscape {
			return "", c.errf("string length escape %#04x is not defined by this grammar", w)
		}
		length = int(w)
	}
	b, err := c.take(length)
	if err != nil {
		return "", c.errf("string of %d byte(s) runs past the end (%d left)", length, c.remaining())
	}
	return string(b), nil
}

// strArray reads a u16-counted array of strings. It is the GROUP TITLES' shape
// and no entry's: an entry's trailing strings are a fixed run with no count.
func (c *cursor) strArray() ([]string, error) {
	n, err := c.u16()
	if err != nil {
		return nil, err
	}
	// Each string costs at least its own length byte.
	if int(n) > c.remaining() {
		return nil, c.errf("string array of %d cannot be satisfied by the %d byte(s) left", n, c.remaining())
	}
	out := make([]string, n)
	for i := range out {
		if out[i], err = c.str(); err != nil {
			return nil, err
		}
	}
	return out, nil
}

// params reads a u16-counted array of raw u32s as int32, so a -1 cell reads as
// -1 rather than as 4294967295.
func (c *cursor) params() ([]int32, error) {
	n, err := c.u16()
	if err != nil {
		return nil, err
	}
	b, err := c.take(int(n) * 4)
	if err != nil {
		return nil, c.errf("parameter array of %d value(s) runs past the end (%d byte(s) left)", n, c.remaining())
	}
	out := make([]int32, n)
	for i := range out {
		out[i] = int32(binary.LittleEndian.Uint32(b[i*4:]))
	}
	return out, nil
}
