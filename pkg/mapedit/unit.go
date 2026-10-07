package mapedit

import (
	"encoding/binary"
	"fmt"
)

// The type-6 record as this story addresses it: a fixed 70 bytes whose first two
// words are the fixed-point coordinates, and the type-0 word that says how many
// of them there are. Nothing else about the record is named, because nothing else
// about it is touched — the other 62 bytes hold class keys, an owner slot,
// sentinel runs and id words this package neither reads nor writes.
const (
	unitTypeID = 6

	unitRecordSize = 70
	unitX          = 0x00
	unitY          = 0x04

	metaCount6 = 0x24
)

// UnitCount returns the number of type-6 records in the map, and zero on a map
// that carries no type-6 record at all.
//
// It is read from #type6 at type-0 +0x24, not from the type-6 payload's length.
// The two agree on every stream this model holds — acceptance ties the payload to
// exactly 70 bytes per count, and the format's own unit walk is driven by the
// count word — so this is a choice about which word *is* the count, not about
// which one is right. Dividing the payload length by 70 would agree today and
// would hide, on a stream where the container disagreed with itself, a divergence
// this package would then be silently papering over.
//
// With the record absent the count word is INERT rather than authoritative: the
// engine's unit case never runs, so the figure names records nothing will read,
// and a shipped map advertises hundreds it does not contain. The decoded view
// has no units there, and this count is the same zero — which is what keeps
// "the view's unit count equals this one" true on every map.
func (e *Editor) UnitCount() int { return e.unitCount(e.locate()) }

// UnitRecord returns the 70 bytes of the record at file-order index i as a fresh
// copy, or an error and no record if there is no such index.
//
// The copy is the point rather than a courtesy. Nothing in this package can
// synthesize the 50 bytes the decoded view does not expose, so the cheap way to a
// record this model will accept is to clone one that is already in the map,
// change the fields the caller understands and place it back. A window into the
// buffer would turn that workflow into a way to edit the map without going
// through an edit, undo and redo included.
func (e *Editor) UnitRecord(i int) ([]byte, error) {
	f := e.locate()
	units, err := f.optional("read a unit", unitTypeID)
	if err != nil {
		return nil, err
	}
	if err := e.checkUnitIndex(f, "read", i); err != nil {
		return nil, err
	}
	at := units.abs(i * unitRecordSize)
	return cloneBytes(e.data[at : at+unitRecordSize]), nil
}

// PlaceUnit appends record after the last type-6 record and returns its
// file-order index — the new highest one. A rejected call returns -1 and an
// error, having written and recorded nothing.
//
// record must be exactly 70 bytes and is copied verbatim: no byte of it is
// synthesized, normalized or reinterpreted, so a record a caller cloned out of a
// map is the record the file gets back, its undecoded bytes included. That is why
// the argument is a whole record and not a typed value — a typed value would have
// to invent the 50 bytes it cannot name, and inventing them as zeros is the
// failure this whole package exists to avoid.
//
// The edit is three parts: the record at the end of the type-6 payload, that
// record's own payloadSize word, and #type6. The file header's dataSize is
// deliberately not among them — the format's reader ignores it, so it is carried
// like every other byte rather than recomputed from a formula nothing here owns.
//
// All three parts address a type-6 record, so a map that carries none is a
// rejection and not the first unit of a new roster. Adding the record is a real
// and tractable feature — a record header, an empty payload spliced in and the
// file header's own record count incremented — but it is the one mutation that
// changes the roster rather than a record, and it is out of this story's scope.
func (e *Editor) PlaceUnit(record []byte) (int, error) {
	if len(record) != unitRecordSize {
		return -1, fmt.Errorf("mapedit: cannot place a %d-byte record: a type-6 record is %d bytes",
			len(record), unitRecordSize)
	}
	f := e.locate()
	units, err := f.optional("place a unit", unitTypeID)
	if err != nil {
		return -1, err
	}
	n := e.unitCount(f)
	e.apply(edit{
		replace(units.abs(n*unitRecordSize), nil, record),
		e.sizeWordPart(units, unitRecordSize),
		e.countWordPart(f, 1),
	})
	return n, nil
}

// MoveUnit sets the X and Y words of the record at index i to the raw
// fixed-point values x and y (the integer tile is value>>8), leaving that
// record's other 62 bytes exactly as they arrived.
//
// The two words are adjacent, so the edit is one eight-byte part and there is no
// ninth byte for it to reach. Neither value is checked against W or H: the
// contract makes a coordinate the caller's business, and a bounds rule here would
// make this package the arbiter of what the game can load — which is not what
// this story decided, and not where that decision belongs.
func (e *Editor) MoveUnit(i int, x, y uint32) error {
	f := e.locate()
	units, err := f.optional("move a unit", unitTypeID)
	if err != nil {
		return err
	}
	if err := e.checkUnitIndex(f, "move", i); err != nil {
		return err
	}

	var img [unitY + 4]byte
	binary.LittleEndian.PutUint32(img[unitX:], x)
	binary.LittleEndian.PutUint32(img[unitY:], y)

	at := units.abs(i*unitRecordSize + unitX)
	e.apply(edit{replace(at, e.data[at:at+len(img)], img[:])})
	return nil
}

// DeleteUnit removes the record at index i, decrementing #type6 and shifting
// every later record — and so every later index — down by one.
//
// It is the exact mirror of PlaceUnit, in the same three parts, which is what
// makes a place followed by the delete of its own index reproduce the image it
// started from.
func (e *Editor) DeleteUnit(i int) error {
	f := e.locate()
	units, err := f.optional("delete a unit", unitTypeID)
	if err != nil {
		return err
	}
	if err := e.checkUnitIndex(f, "delete", i); err != nil {
		return err
	}
	at := units.abs(i * unitRecordSize)
	e.apply(edit{
		replace(at, e.data[at:at+unitRecordSize], nil),
		e.sizeWordPart(units, -unitRecordSize),
		e.countWordPart(f, -1),
	})
	return nil
}

// unitCount reads #type6 out of an already-located frame, and answers zero
// where the frame has no type-6 record for that word to be about.
//
// The u32 becomes an int without a range check because acceptance leaves no room
// for one to matter: the type-6 payload is exactly 70 bytes per count and its
// length is an int, so the count is at most that length over seventy on any
// platform this builds for. That coupling is what the absence test restores —
// with no payload there is no such bound, and the word may say anything.
func (e *Editor) unitCount(f frame) int {
	if !f.carries(unitTypeID) {
		return 0
	}
	return int(e.u32(f.required(metaTypeID).abs(metaCount6)))
}

// checkUnitIndex is the whole rejection surface the three index-taking setters
// share, run before any of them constructs an edit. what names the operation so a
// caller reading the error knows which call it came from.
func (e *Editor) checkUnitIndex(f frame, what string, i int) error {
	if n := e.unitCount(f); i < 0 || i >= n {
		return fmt.Errorf("mapedit: cannot %s unit %d: the map has %d", what, i, n)
	}
	return nil
}

// sizeWordPart is the part that moves a located record's own payloadSize word,
// in its 20-byte header, by delta bytes.
func (e *Editor) sizeWordPart(sp span, delta int) part {
	return e.wordPart(sp.headerOff+recPayloadSize, delta)
}

// countWordPart is the part that moves #type6, in the type-0 payload, by delta.
func (e *Editor) countWordPart(f frame, delta int) part {
	return e.wordPart(f.required(metaTypeID).abs(metaCount6), delta)
}

// wordPart builds the part that adds delta to the little-endian word at absolute
// offset at. Both callers only ever add or subtract one record's worth, and both
// are reached only after the caller has established that the word is large enough
// to take it: a delete has already found the index it is removing.
func (e *Editor) wordPart(at, delta int) part {
	var img [4]byte
	binary.LittleEndian.PutUint32(img[:], uint32(int64(e.u32(at))+int64(delta)))
	return replace(at, e.data[at:at+len(img)], img[:])
}
