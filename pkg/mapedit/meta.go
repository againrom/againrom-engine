package mapedit

import (
	"encoding/binary"
	"fmt"
	"math"

	"againrom/pkg/formats/alm"
)

// The type-0 fields this story mutates, payload-relative — the same frame W and
// H are read from in grid.go. Only the offsets a setter below writes are named:
// the count words, the +0x28/+0x2c pair and the seven trailing text slots are
// out of scope, and leaving them unnamed is what keeps this list from becoming a
// second copy of the type-0 layout that a later correction could leave behind.
const (
	metaAngle   = 0x08
	metaWord0C  = 0x0c
	metaWord10  = 0x10
	metaWord14  = 0x14
	metaBitmask = 0x18
	metaName    = 0x30
	metaWord70  = 0x70
	metaWord74  = 0x74
	metaDesc    = 0x78
)

// SetAngle sets the type-0 f32 at +0x08 to v.
//
// The caller's float32 reaches the field as its own bit pattern, and nothing on
// the way reads the field's current bytes back through a float. That is not an
// optimisation: a recorded angle is any 32-bit pattern the file happens to
// hold, signalling NaNs included, and a value that made a round trip through a
// float could come back quieted — so the field is written, never compared, and
// never re-encoded from what was already there.
func (e *Editor) SetAngle(v float32) error {
	return e.setWord(metaAngle, math.Float32bits(v))
}

// SetWord0C sets the type-0 stored scalar at +0x0c to v as a whole 32-bit word.
//
// It, and the four setters below it, assert no meaning for any bit of v. The
// format's own reader exposes these words raw because nothing has decoded them;
// a setter that interpreted them would be claiming more than the reader does.
func (e *Editor) SetWord0C(v uint32) error { return e.setWord(metaWord0C, v) }

// SetWord10 sets the type-0 stored scalar at +0x10 to v.
func (e *Editor) SetWord10(v uint32) error { return e.setWord(metaWord10, v) }

// SetWord14 sets the type-0 stored scalar at +0x14 to v.
func (e *Editor) SetWord14(v uint32) error { return e.setWord(metaWord14, v) }

// SetWord70 sets the type-0 stored scalar at +0x70 to v.
func (e *Editor) SetWord70(v uint32) error { return e.setWord(metaWord70, v) }

// SetWord74 sets the type-0 stored scalar at +0x74 to v.
func (e *Editor) SetWord74(v uint32) error { return e.setWord(metaWord74, v) }

// SetBitmask sets the type-0 low-bit word at +0x18 to v, on the same terms: the
// whole word, no bit of it interpreted.
func (e *Editor) SetBitmask(v uint32) error { return e.setWord(metaBitmask, v) }

// SetName sets the type-0 name field at +0x30 to s.
//
// The bytes come from pkg/formats/alm, which owns both directions of this
// field's rule — which codec it is, how wide the field is, and that it stays
// terminated. None of that is restated here, so the two directions cannot drift
// apart, and this setter's whole contribution is the offset. A string the
// encoder refuses is a rejection that has written and recorded nothing; an
// accepted one replaces the field's every byte, so a shorter name leaves no
// residue of the one it replaced.
func (e *Editor) SetName(s string) error {
	img, err := alm.EncodeName(s)
	if err != nil {
		return fmt.Errorf("mapedit: %w", err)
	}
	e.writeMeta(metaName, img)
	return nil
}

// SetDescription sets the type-0 description field at +0x78 to s, on exactly
// SetName's terms and through that field's own encoder.
func (e *Editor) SetDescription(s string) error {
	img, err := alm.EncodeDescription(s)
	if err != nil {
		return fmt.Errorf("mapedit: %w", err)
	}
	e.writeMeta(metaDesc, img)
	return nil
}

// setWord is the write path of every type-0 scalar setter: the caller's word as
// its little-endian image over the four bytes at rel.
//
// It returns an error it never produces, and so do its callers. That is the
// contract's shape rather than an oversight — a type-0 scalar has no rejection
// surface, since every 32-bit pattern is a legal value of a field no bit of
// which this package interprets, while the string, grid and unit setters all do
// have one. A caller therefore writes one error path for every mutation instead
// of having to remember which six of them cannot fail.
func (e *Editor) setWord(rel int, v uint32) error {
	var img [4]byte
	binary.LittleEndian.PutUint32(img[:], v)
	e.writeMeta(rel, img[:])
	return nil
}

// writeMeta replaces the len(img) bytes at type-0 payload offset rel with img,
// as the one part of one edit.
//
// The region's width is the image's own length, so a caller cannot name a field
// and hand over the wrong number of bytes for it: a four-byte word writes four
// and a field image writes whatever width the encoder built. Both land inside
// the type-0 payload, whose length acceptance fixed at 632 bytes, so the
// subscript is in bounds by the load invariant.
//
// No setter here can fail for a missing section, and that is a fact about the
// format rather than an omission: type-0 is one of the records alm refuses a
// stream without, so it is located on every buffer this model holds.
func (e *Editor) writeMeta(rel int, img []byte) {
	at := e.locate().required(metaTypeID).abs(rel)
	e.apply(edit{replace(at, e.data[at:at+len(img)], img)})
}
