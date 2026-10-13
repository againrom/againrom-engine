package mapedit

import (
	"encoding/binary"
	"fmt"
	"slices"
)

// Kind names a record section an edit addresses by file-order index.
type Kind int

const (
	KindUnit      Kind = iota // type 6, 70-byte placements (ALM-UNIT-018)
	KindStructure             // type 4, 20 bytes plus the 0x21 extension (ALM-OBJ-019, ALM-OBJ-034)
	KindLoot                  // type 8, a ground sack or a unit's stock (ALM-SACK-065)
	KindAction                // type 7 action node array (ALM-TRIG-044, ALM-TRIG-045)
	KindCondition             // type 7 condition node array
	KindTrigger               // type 7 trigger array, 184-byte records (ALM-TRIG-047)
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

	// layout, when set, finds the records inside a payload that holds several
	// counted arrays, in place of the type-0 count word and a payload that is
	// one array.
	layout func(e *Editor, sp span) (arrayLayout, error)

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
	KindAction:    scriptSection("action", 0),
	KindCondition: scriptSection("condition", 1),
	KindTrigger:   scriptSection("trigger", 2),
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
	frame   frame
	sec     *section
	span    span
	countAt int // absolute offset of the word that counts the records
	starts  []int
}

// arrayLayout is where one counted array sits in its payload: the absolute
// offset of its count word, the payload-relative start of its first record,
// the record count and the payload bytes that follow the last record.
type arrayLayout struct {
	countAt, first, n, after int
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
	lay := arrayLayout{countAt: f.required(metaTypeID).abs(sec.count)}
	lay.n = int(e.u32(lay.countAt))
	if sec.layout != nil {
		lay, err = sec.layout(e, sp)
		if err != nil {
			return walk{}, fmt.Errorf("mapedit: cannot %s: %w", what, err)
		}
	}
	w := walk{frame: f, sec: sec, span: sp, countAt: lay.countAt, starts: []int{lay.first}}
	payload := e.data[sp.payloadOff : sp.payloadOff+sp.payloadSize]
	off := lay.first
	for i := 0; i < lay.n; i++ {
		l := sec.length(e, payload[off:])
		if l <= 0 {
			return walk{}, fmt.Errorf("mapedit: cannot %s: %s record %d does not fit in the type-%d payload", what, sec.name, i, sec.typeID)
		}
		off += l
		w.starts = append(w.starts, off)
	}
	if off+lay.after != sp.payloadSize {
		return walk{}, fmt.Errorf("mapedit: cannot %s: the type-%d walk consumed %d of %d payload bytes", what, sec.typeID, off+lay.after, sp.payloadSize)
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
		e.wordPart(w.countAt, 1),
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
	ed := edit{
		replace(at, e.data[at:at+l], nil),
		e.sizeWordPart(w.span, -l),
		e.wordPart(w.countAt, -1),
	}
	if k == KindLoot {
		ed = append(ed, e.unitLinkParts(i)...)
	}
	e.apply(ed)
	return nil
}

// unitLinkParts renumbers the units' loot links (the word at +0x18, a 1-based
// index into the type-8 array) for the removal of loot record i: a link to it
// becomes 0 and a link above it falls by one, so every other link still names
// the record it named (DIV-3255). A map without units has none.
func (e *Editor) unitLinkParts(i int) []part {
	if !e.locate().carries(unitTypeID) {
		return nil
	}
	uw, err := e.walkSection(KindUnit, "renumber unit loot links")
	if err != nil {
		return nil
	}
	var parts []part
	for u := 0; u < uw.n(); u++ {
		at, _ := uw.at(u)
		p := at + unitLootLink
		old := e.u32(p)
		nv := old
		switch {
		case old == uint32(i)+1:
			nv = 0
		case old > uint32(i)+1:
			nv = old - 1
		}
		if nv != old {
			var img [4]byte
			binary.LittleEndian.PutUint32(img[:], nv)
			parts = append(parts, replace(p, e.data[p:p+4], img[:]))
		}
	}
	return parts
}

// Placement is one record a PlaceAll batch appends to its kind's section.
type Placement struct {
	Kind   Kind
	Record []byte
}

// PlaceAll appends every record of batch, in order within its kind, as one
// edit and returns each record's new index in batch order. The kinds must be
// arrays of one counted payload (the type-7 script): the edit is the records,
// each array's count word and the payload's size word, so one Undo reverts the
// whole batch to the exact earlier bytes. A record of a kind is checked
// against that kind's grammar, as Place does.
func (e *Editor) PlaceAll(batch []Placement) ([]int, error) {
	if len(batch) == 0 {
		return nil, fmt.Errorf("mapedit: nothing to place")
	}
	walks := map[Kind]walk{}
	added := map[Kind][]byte{}
	counts := map[Kind]int{}
	index := make([]int, len(batch))
	for i, p := range batch {
		w, ok := walks[p.Kind]
		if !ok {
			var err error
			if w, err = e.walkSection(p.Kind, "place a "+sectionName(p.Kind)); err != nil {
				return nil, err
			}
			if w.sec.layout == nil {
				return nil, fmt.Errorf("mapedit: cannot place several records: the %s section is not one array of a counted payload", w.sec.name)
			}
			for _, other := range walks {
				if other.span != w.span {
					return nil, fmt.Errorf("mapedit: cannot place records of different payloads in one edit")
				}
			}
			walks[p.Kind] = w
		}
		if w.sec.length(e, p.Record) != len(p.Record) {
			return nil, fmt.Errorf("mapedit: cannot place a %d-byte record: it is not one type-%d %s record", len(p.Record), w.sec.typeID, w.sec.name)
		}
		index[i] = w.n() + counts[p.Kind]
		counts[p.Kind]++
		added[p.Kind] = append(added[p.Kind], p.Record...)
	}
	// An array's new records end where the next array's count word begins, so
	// the two parts share an offset and are one splice: records, then the word.
	type splice struct {
		records, removed, word []byte
	}
	var kinds []Kind
	for k := range walks {
		kinds = append(kinds, k)
	}
	slices.Sort(kinds)
	at := map[int]*splice{}
	slot := func(off int) *splice {
		if at[off] == nil {
			at[off] = &splice{}
		}
		return at[off]
	}
	var sp span
	total := 0
	for _, k := range kinds {
		w := walks[k]
		sp = w.span
		end := w.span.abs(w.starts[w.n()])
		slot(end).records = added[k]
		word := e.wordPart(w.countAt, counts[k])
		s := slot(w.countAt)
		s.removed, s.word = word.removed, word.inserted
		total += len(added[k])
	}
	var offs []int
	for off := range at {
		offs = append(offs, off)
	}
	slices.Sort(offs)
	var ed edit
	for _, off := range offs {
		s := at[off]
		ed = append(ed, replace(off, s.removed, append(slices.Clone(s.records), s.word...)))
	}
	ed = append(ed, e.sizeWordPart(sp, total))
	e.apply(ed)
	return index, nil
}
