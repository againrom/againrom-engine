package mapedit

import (
	"cmp"
	"encoding/binary"
	"fmt"
	"slices"

	"againrom/pkg/formats/alm"
)

// The container framing this model navigates by (docs/0003-alm-container): a
// 20-byte file header whose +0x0c word counts the records, then that many
// records, each a 20-byte header followed by its pure payload. Only the pieces
// a navigator needs are named here — the count, a record's own length word, its
// typeId, and the type-0 record every field offset in this package is relative
// to.
const (
	fileHeaderSize   = 20
	recordHeaderSize = 20

	fhRecordCount = 0x0c

	// typeIDCount is the size of the typeId INDEX SPACE, not a roster size: a
	// record whose id is at or above it selects no section here, exactly as it
	// selects no case in the engine's own loader.
	typeIDCount = 10

	recPayloadSize = 0x08
	recTypeID      = 0x0c

	metaTypeID = 0
)

// Editor is a headless edit session over one accepted ROM1 .alm map.
//
// Its whole state is the accepted bytes, an ordered log of the edits applied
// to them and a cursor into that log. Nothing derived is stored: record spans,
// the map dimensions, the unit count and every field offset are computed from
// the current bytes where they are used. A length-changing edit therefore has
// no cached index to invalidate, which removes the stale-index failure instead
// of defending against it — the reason to pay for a record walk per call.
//
// An Editor is a single-goroutine session and is not safe for concurrent use.
type Editor struct {
	data   []byte // the accepted stream; the only thing this model really holds
	log    []edit // the accepted mutations, oldest first
	cursor int    // len(log) with nothing undone; the redo boundary otherwise
}

// New loads an ALM byte stream into a fresh edit model.
//
// Acceptance is alm.OpenDocument's alone and is taken once, here: this package
// adds no validation walk of its own, because two expressions of one accept
// set are two things that can disagree. A rejected stream yields a wrapped
// error and a nil model, never a partial one and never a panic.
//
// A thin roster is therefore loaded, not refused. Only type-0, type-1 and
// type-2 are guaranteed to be in it; a map with no overlay plane and a map with
// no units are ordinary maps here, and it is the setter addressing a record the
// file does not carry that fails, one call at a time.
//
// The buffer the model edits is that document's own write-back — the bytes
// acceptance was taken on. The caller's slice is therefore never the edited
// state, and mutating it afterwards changes nothing here.
func New(data []byte) (*Editor, error) {
	doc, err := alm.OpenDocument(data)
	if err != nil {
		return nil, fmt.Errorf("mapedit: %w", err)
	}
	return &Editor{data: doc.Write()}, nil
}

// Bytes returns the current serialized map in a fresh buffer. Mutating it
// affects neither the model nor any other call's result, and a buffer handed
// out before an edit still holds the bytes as they were when it was taken.
func (e *Editor) Bytes() []byte { return cloneBytes(e.data) }

// Clone preserves the complete undo/redo branch for a transactional edit.
// Parts are immutable after creation; the log slice and current bytes are not.
func (e *Editor) Clone() *Editor {
	return &Editor{data: e.Bytes(), log: slices.Clone(e.log), cursor: e.cursor}
}

// Map returns the interpreted view of the current bytes: alm.Open of a fresh
// copy of them, run per call.
//
// No decoded state is stored, so no accessor can report what the bytes do not
// say, and every "the view is unchanged" clause of this package's contract
// collapses to byte equality. The copy is not a courtesy to alm.Open: it is
// what makes the guarantee this package's own rather than a consequence of
// which pieces that reader happens to copy on its way out.
func (e *Editor) Map() (*alm.Map, error) { return alm.Open(e.Bytes()) }

// CanUndo reports whether there is an accepted mutation left to undo.
func (e *Editor) CanUndo() bool { return e.cursor > 0 }

// CanRedo reports whether there is an undone mutation left to re-apply.
func (e *Editor) CanRedo() bool { return e.cursor < len(e.log) }

// Undo reverts the most recent not-yet-undone mutation, restoring the bytes to
// exactly their state before it, and reports whether there was one to revert.
// With nothing to undo it is a no-op that changes nothing and returns false.
func (e *Editor) Undo() bool {
	if !e.CanUndo() {
		return false
	}
	e.cursor--
	e.splice(e.log[e.cursor], revert)
	return true
}

// Redo re-applies the most recently undone mutation byte-identically and
// reports whether there was one to re-apply. With nothing to redo it is a
// no-op that changes nothing and returns false.
func (e *Editor) Redo() bool {
	if !e.CanRedo() {
		return false
	}
	e.splice(e.log[e.cursor], forward)
	e.cursor++
	return true
}

// part is one splice of an edit: a pre-image offset, the bytes it removes
// there and the bytes it inserts in their place. Both are held as copies, so a
// history entry depends neither on the buffer it was read out of nor on a
// slice the caller may still change.
type part struct {
	off      int
	removed  []byte
	inserted []byte
}

// edit is one accepted mutation: the parts it writes, applied and reverted as
// a group. It is at once the unit the applier writes and the unit the history
// remembers, which is what makes "every accepted mutation is recorded" a shape
// rather than a discipline.
type edit []part

// replace builds a part: at pre-image offset off, removed gives way to
// inserted.
func replace(off int, removed, inserted []byte) part {
	return part{off: off, removed: cloneBytes(removed), inserted: cloneBytes(inserted)}
}

// apply writes an accepted mutation and records it, with no branch between the
// two: every mutation that reaches the buffer is in the history — including
// one whose new value equals the old, since nothing here inspects the values —
// and there is no path that writes without appending. Appending at the cursor
// drops whatever had been undone, which is the discard-the-redo-history rule
// stated once in the applier rather than in every setter.
//
// A setter has finished rejecting before it calls this; nothing below can
// fail, which is why a rejection can have written and recorded nothing.
func (e *Editor) apply(ed edit) {
	e.splice(ed, forward)
	e.log = append(e.log[:e.cursor], ed)
	e.cursor = len(e.log)
}

// direction selects which side of an edit's parts is written.
type direction bool

const (
	forward direction = false // removed -> inserted, as the setter built it
	revert  direction = true  // inserted -> removed, undoing it
)

// splice is the only function in this package that writes to the buffer. Every
// setter reaches it through apply, Undo and Redo call it directly, and nothing
// else assigns to e.data.
//
// Forward, it replaces each part's removed bytes with its inserted bytes in
// descending pre-image offset order: a part's recorded offset is still valid
// when its turn comes because everything below it has not moved yet. Revert
// walks ascending instead, and for the mirror reason: undoing a part cancels
// the shift it contributed, so by the time a later part's turn comes every
// part below it is back where it was recorded and its own pre-image offset is
// valid again. Neither order is a preference and neither is safe alone; both
// hold only because the parts of one edit never overlap, which is the rule the
// setters owe this function.
func (e *Editor) splice(ed edit, dir direction) {
	ordered := slices.Clone(ed)
	slices.SortFunc(ordered, func(a, b part) int { return cmp.Compare(a.off, b.off) })
	if dir == forward {
		slices.Reverse(ordered)
	}
	for _, p := range ordered {
		out, in := p.removed, p.inserted
		if dir == revert {
			out, in = p.inserted, p.removed
		}
		if len(in) == len(out) {
			copy(e.data[p.off:], in)
			continue
		}
		buf := make([]byte, 0, len(e.data)-len(out)+len(in))
		buf = append(buf, e.data[:p.off]...)
		buf = append(buf, in...)
		buf = append(buf, e.data[p.off+len(out):]...)
		e.data = buf
	}
}

// span locates one record in the current buffer: the absolute offsets of its
// 20-byte header and of its payload, and that payload's byte length.
type span struct {
	headerOff   int
	payloadOff  int
	payloadSize int
}

// abs turns a payload-relative offset inside this record into an absolute
// offset in the buffer.
//
// It is a method on a LOCATED record and no longer on the frame, which is the
// shape of the whole absent-section fix: an offset cannot be computed for a
// typeId any more, only for a record the walk actually found and handed over.
func (s span) abs(rel int) int { return s.payloadOff + rel }

// frame is the current buffer's records indexed by typeId, together with the
// flag that says whether the stream really carried each one.
//
// The flag is the revision. alm accepts three records and up with only type-0,
// type-1 and type-2 guaranteed, so an id the file omits has no record to point
// at — and the frame's own zero value would have pointed it at offset 0, the
// FILE HEADER's first byte. Absence is representable here instead, and the two
// accessors below are the only way to read the array: required for the ids
// acceptance guarantees, optional for the ids it does not.
type frame struct {
	record  [typeIDCount]span
	present [typeIDCount]bool
}

// required returns the record carrying typeId tid, for the three ids
// acceptance guarantees: type-0, and the type-1 and type-2 planes the engine's
// world builder dereferences per cell with no null test, which is why alm
// refuses a stream missing either. It cannot fail and does not pretend it can —
// a rejection branch here would be one no accepted input can reach.
func (f frame) required(tid uint32) span { return f.record[tid] }

// optional returns the record carrying typeId tid, or an error naming the
// operation when the accepted stream does not carry it. Every section this
// model addresses beside the three above is reached through here, and the
// error is a rejection like any other: the caller has written and recorded
// nothing.
//
// The span it returns beside the error is the one locate left for an absent
// id — past EOF, not offset zero — so even a caller that dropped the error
// would take a slice subscript out of range rather than edit the file header.
func (f frame) optional(what string, tid uint32) (span, error) {
	if !f.present[tid] {
		return f.record[tid], fmt.Errorf("mapedit: cannot %s: this map carries no type-%d record", what, tid)
	}
	return f.record[tid], nil
}

// carries reports whether the accepted stream really held the record with this
// typeId — the same distinction alm.Map.Present draws, read off this model's
// own walk.
func (f frame) carries(tid uint32) bool { return f.present[tid] }

// locate walks the record frame of the current buffer and returns it. It is
// computed per call and never stored — that is what lets a length-changing
// edit be a plain splice with nothing to fix up afterwards.
//
// The loop bound is the FILE HEADER's own record count and not a constant ten.
// A stream alm accepts carries at least three records, may carry more than ten,
// and may end with bytes no record covers; walking a fixed ten would both miss
// a record past the tenth and read a record out of a trailer the count never
// reaches. An id at or above typeIDCount selects no entry, and a repeated id
// overwrites — last-wins, which is what alm's own walk does.
//
// The walk reads no byte it has not bounds-checked, so a frame this package
// somehow broke would yield a short walk rather than a panic.
func (e *Editor) locate() frame {
	var f frame
	// An absent record is located ONE BYTE PAST EOF rather than left at the
	// array's zero value. Offset zero is the file header's first byte, and
	// resolving an absent section's payload-relative offsets against it is
	// exactly how this model would corrupt an accepted map silently. From here
	// every derived offset is out of range and the subscript panics instead.
	past := len(e.data) + 1
	for tid := range f.record {
		f.record[tid] = span{headerOff: past, payloadOff: past}
	}

	n := e.u32(fhRecordCount)
	cursor := fileHeaderSize
	for i := uint32(0); i < n; i++ {
		if cursor+recordHeaderSize > len(e.data) {
			break
		}
		start := cursor + recordHeaderSize
		size := uint64(binary.LittleEndian.Uint32(e.data[cursor+recPayloadSize:]))
		if size > uint64(len(e.data)-start) {
			break
		}
		if tid := binary.LittleEndian.Uint32(e.data[cursor+recTypeID:]); tid < typeIDCount {
			f.record[tid] = span{headerOff: cursor, payloadOff: start, payloadSize: int(size)}
			f.present[tid] = true
		}
		cursor = start + int(size)
	}
	return f
}

// u32 reads the little-endian word at an absolute offset of the current
// buffer.
//
// It is unchecked, and what makes that sound is where its offsets come from:
// each is either a file-header field of a stream alm accepted, or the abs of a
// record the walk located, inside a payload whose length that same acceptance
// fixed — the type-0 payload, for one, is its constant 632 bytes. A subscript
// outside them would be a broken invariant, not input. The invariant used to be
// "all ten records exist"; it is now "the record was found", and reaching every
// optional section through a lookup that can fail is what re-establishes it.
func (e *Editor) u32(off int) uint32 { return binary.LittleEndian.Uint32(e.data[off:]) }

// cloneBytes returns a fresh copy of b. A nil or empty input yields a non-nil
// zero-length slice, so a part that inserts or removes nothing still holds a
// slice rather than a nil.
func cloneBytes(b []byte) []byte {
	out := make([]byte, len(b))
	copy(out, b)
	return out
}
