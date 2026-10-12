package mapedit

import (
	"encoding/binary"
	"fmt"
)

// Kind names a record section an edit addresses by file-order index.
type Kind int

const (
	KindUnit      Kind = iota // type 6, 70-byte placements (ALM-UNIT-018)
	KindStructure             // type 4, 20 bytes plus the 0x21 extension (ALM-OBJ-019, ALM-OBJ-034)
	KindLoot                  // type 8, a ground sack or a unit's stock (ALM-SACK-065)
)

// section is one row of the record-section table: where a kind's records
// live, the grammar that measures one, its coordinate words and the hooks a
// grammar needs. Place, clone, move, delete and the field table read only
// this row, so a further kind is a row and not a second editor.
type section struct {
	name   string
	typeID uint32
	count  int // the type-0 word that counts the records
	x, y   int // fixed-point coordinate words, /256 is the cell

	// length measures the record at the start of b, or answers -1 when it
	// does not fit.
	length func(e *Editor, b []byte) int
	// cellless reports a record whose coordinates the game never reads.
	cellless func(rec []byte) bool
	// renumber gives a clone its identity; nil copies every byte.
	renumber func(e *Editor, w walk, rec []byte) error
	// reshape restores the grammar after a field write; nil keeps the length.
	reshape func(rec []byte) []byte
	// fields lists the settable fields of one record.
	fields func(e *Editor, rec []byte) []Field
}

var sections = map[Kind]*section{
	KindUnit: {name: "unit", typeID: unitTypeID, count: metaCount6, x: unitX, y: unitY,
		length:   func(_ *Editor, b []byte) int { return fixedLength(b, unitRecordSize) },
		renumber: renumberUnit,
		fields:   func(*Editor, []byte) []Field { return unitFields },
	},
	KindStructure: {name: "structure", typeID: structureTypeID, count: metaCount4, x: 0x00, y: 0x04,
		length:  structureLength,
		reshape: reshapeStructure,
		fields:  structureFields,
	},
	KindLoot: {name: "sack", typeID: lootTypeID, count: metaCount8, x: lootX, y: lootY,
		length:   lootLength,
		cellless: func(rec []byte) bool { return binary.LittleEndian.Uint32(rec[lootOwner:]) != 0 },
		fields:   lootFields,
	},
}

func fixedLength(b []byte, n int) int {
	if len(b) < n {
		return -1
	}
	return n
}

func sectionOf(k Kind) (*section, error) {
	if s, ok := sections[k]; ok {
		return s, nil
	}
	return nil, fmt.Errorf("mapedit: no record section %d", k)
}

// walk is one section located in the current buffer: its record and the
// payload-relative start of each record, with the payload end last.
type walk struct {
	frame  frame
	sec    *section
	span   span
	starts []int
}

func (w walk) n() int { return len(w.starts) - 1 }

// at is the absolute offset and the length of record i.
func (w walk) at(i int) (int, int) {
	return w.span.abs(w.starts[i]), w.starts[i+1] - w.starts[i]
}

// walkSection locates k's record and walks its count word's worth of records
// through the grammar. A section that does not consume its payload exactly
// is refused: no edit is built over bytes the grammar cannot account for.
func (e *Editor) walkSection(k Kind, what string) (walk, error) {
	sec, err := sectionOf(k)
	if err != nil {
		return walk{}, err
	}
	f := e.locate()
	sp, err := f.optional(what, sec.typeID)
	if err != nil {
		return walk{}, err
	}
	w := walk{frame: f, sec: sec, span: sp, starts: []int{0}}
	n := e.u32(f.required(metaTypeID).abs(sec.count))
	payload := e.data[sp.payloadOff : sp.payloadOff+sp.payloadSize]
	off := 0
	for i := uint32(0); i < n; i++ {
		l := sec.length(e, payload[off:])
		if l <= 0 {
			return walk{}, fmt.Errorf("mapedit: cannot %s: %s record %d does not fit in the type-%d payload", what, sec.name, i, sec.typeID)
		}
		off += l
		w.starts = append(w.starts, off)
	}
	if off != sp.payloadSize {
		return walk{}, fmt.Errorf("mapedit: cannot %s: the type-%d walk consumed %d of %d payload bytes", what, sec.typeID, off, sp.payloadSize)
	}
	return w, nil
}

// walkIndex is walkSection plus the index check every index-taking operation
// shares.
func (e *Editor) walkIndex(k Kind, what string, i int) (walk, error) {
	w, err := e.walkSection(k, what+" a "+sectionName(k))
	if err != nil {
		return walk{}, err
	}
	if i < 0 || i >= w.n() {
		return walk{}, fmt.Errorf("mapedit: cannot %s %s %d: the map has %d", what, w.sec.name, i, w.n())
	}
	return w, nil
}

func sectionName(k Kind) string {
	if s, ok := sections[k]; ok {
		return s.name
	}
	return "record"
}

// Count returns the number of k's records, zero on a map that carries no such
// record or whose section does not walk.
func (e *Editor) Count(k Kind) int {
	w, err := e.walkSection(k, "count")
	if err != nil {
		return 0
	}
	return w.n()
}

// Record returns a fresh copy of k's record at file-order index i.
func (e *Editor) Record(k Kind, i int) ([]byte, error) {
	w, err := e.walkIndex(k, "read", i)
	if err != nil {
		return nil, err
	}
	at, l := w.at(i)
	return cloneBytes(e.data[at : at+l]), nil
}

// Position returns the raw fixed-point X and Y words of k's record i, and an
// error for a record whose coordinates the game never reads.
func (e *Editor) Position(k Kind, i int) (x, y uint32, err error) {
	rec, err := e.Record(k, i)
	if err != nil {
		return 0, 0, err
	}
	sec := sections[k]
	if sec.cellless != nil && sec.cellless(rec) {
		return 0, 0, fmt.Errorf("mapedit: %s %d is carried by a unit and has no cell", sec.name, i)
	}
	return binary.LittleEndian.Uint32(rec[sec.x:]), binary.LittleEndian.Uint32(rec[sec.y:]), nil
}

// Place appends record after k's last record and returns its index. The
// record is copied verbatim and must be exactly one record of k's grammar.
// The edit is three parts: the record, the section's payloadSize word and
// its type-0 count word.
func (e *Editor) Place(k Kind, record []byte) (int, error) {
	w, err := e.walkSection(k, "place a "+sectionName(k))
	if err != nil {
		return -1, err
	}
	if w.sec.length(e, record) != len(record) {
		return -1, fmt.Errorf("mapedit: cannot place a %d-byte record: it is not one type-%d record", len(record), w.sec.typeID)
	}
	return e.appendRecord(w, record), nil
}

func (e *Editor) appendRecord(w walk, record []byte) int {
	n := w.n()
	e.apply(edit{
		replace(w.span.abs(w.starts[n]), nil, record),
		e.sizeWordPart(w.span, len(record)),
		e.wordPart(w.frame.required(metaTypeID).abs(w.sec.count), 1),
	})
	return n
}

// CloneRecord appends a copy of k's record i with its X and Y words set to x and y
// and returns the new index, as one edit. The section's renumber rule gives
// the copy its identity; every other byte is the source's. A record without
// a cell is refused.
func (e *Editor) CloneRecord(k Kind, i int, x, y uint32) (int, error) {
	w, err := e.walkIndex(k, "clone", i)
	if err != nil {
		return -1, err
	}
	at, l := w.at(i)
	rec := cloneBytes(e.data[at : at+l])
	if w.sec.cellless != nil && w.sec.cellless(rec) {
		return -1, fmt.Errorf("mapedit: cannot clone %s %d: it is carried by a unit and has no cell", w.sec.name, i)
	}
	binary.LittleEndian.PutUint32(rec[w.sec.x:], x)
	binary.LittleEndian.PutUint32(rec[w.sec.y:], y)
	if w.sec.renumber != nil {
		if err := w.sec.renumber(e, w, rec); err != nil {
			return -1, fmt.Errorf("mapedit: cannot clone %s %d: %w", w.sec.name, i, err)
		}
	}
	return e.appendRecord(w, rec), nil
}

// Move sets the X and Y words of k's record i to the raw fixed-point values
// x and y, leaving every other byte as it arrived. Neither value is checked
// against the map; the host owns that check.
func (e *Editor) Move(k Kind, i int, x, y uint32) error {
	w, err := e.walkIndex(k, "move", i)
	if err != nil {
		return err
	}
	at, l := w.at(i)
	if w.sec.cellless != nil && w.sec.cellless(e.data[at:at+l]) {
		return fmt.Errorf("mapedit: cannot move %s %d: it is carried by a unit and has no cell", w.sec.name, i)
	}
	var img [8]byte
	binary.LittleEndian.PutUint32(img[0:], x)
	binary.LittleEndian.PutUint32(img[4:], y)
	if w.sec.y == w.sec.x+4 {
		p := at + w.sec.x
		e.apply(edit{replace(p, e.data[p:p+8], img[:])})
		return nil
	}
	px, py := at+w.sec.x, at+w.sec.y
	e.apply(edit{replace(px, e.data[px:px+4], img[:4]), replace(py, e.data[py:py+4], img[4:])})
	return nil
}

// Delete removes k's record i, shrinking the payloadSize and count words and
// shifting every later index down by one: the mirror of Place.
func (e *Editor) Delete(k Kind, i int) error {
	w, err := e.walkIndex(k, "delete", i)
	if err != nil {
		return err
	}
	at, l := w.at(i)
	e.apply(edit{
		replace(at, e.data[at:at+l], nil),
		e.sizeWordPart(w.span, -l),
		e.wordPart(w.frame.required(metaTypeID).abs(w.sec.count), -1),
	})
	return nil
}
