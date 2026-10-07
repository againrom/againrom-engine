package sim

import (
	"encoding/binary"
	"fmt"
)

// The SCRIPT-STATE SECTION the last arms need (version 50): the per-player
// formation modes and the cell-record tails.
//
// It sits between the casting section and the script section, for the reason the
// casting section sits where it does: decodeScript consumes the rest of the
// buffer and returns no used count, so a section behind it would have to give it
// one.
//
// The FORMATION BLOCK is relationSlots bytes with NO COUNT IN FRONT OF IT. Its
// length is a compile-time constant of this build, exactly as the purse block's
// is, and a count would be a second statement of a number the code already
// knows and could disagree with. Every byte is carried whole: all 256 values are
// modes and 254 of them alias to one behaviour, so there is nothing to refuse.
//
// One CELL TAIL record, 8 bytes:
//
//	+0    2      cell key, uint16
//	+2    6      the six tail bytes, in the order the helper stores them
//
// The key is the packed cell and is the whole of where the tail stands — see
// cellTail's own doc for why no x and y are stored beside it. The six bytes are
// carried as they are: the writer's third and fifth are zero and its fourth and
// sixth are equal, but that is the ARM's property and not the record's, and a
// form that stored only what the arm can produce would refuse a state the arm
// itself could reach if a later story gave the record a second writer.
const (
	tailCountLen  = 4
	tailRecordLen = 2 + cellTailLen
)

// scriptStateSectionLen is how many bytes w's formation block and cell tails
// occupy together.
func (w *World) scriptStateSectionLen() int {
	return relationSlots + tailCountLen + tailRecordLen*len(w.cellTails)
}

// encodeScriptState writes the section into b at off and returns the offset past
// it.
//
// THE TAILS ARE WRITTEN IN THE WORLD'S OWN ORDER, ascending by key, which
// setCellTail maintains. Nothing is sorted here: an order established by the arm
// that builds the state is the canonical one, and re-deriving it in the encoder
// would be a second place it could be got wrong.
func (w *World) encodeScriptState(b []byte, off int) int {
	for i, m := range w.formations {
		b[off+i] = m
	}
	off += relationSlots
	binary.LittleEndian.PutUint32(b[off:off+4], uint32(len(w.cellTails)))
	off += tailCountLen
	for _, t := range w.cellTails {
		binary.LittleEndian.PutUint16(b[off:off+2], t.Key)
		copy(b[off+2:off+tailRecordLen], t.Bytes[:])
		off += tailRecordLen
	}
	return off
}

// decodeScriptState reads the section and returns the block, the tails and how
// many bytes it consumed.
//
// THE DECLARED COUNT IS BOUNDED AGAINST THE BUFFER BEFORE A SINGLE RECORD IS
// ALLOCATED, on decodeGroups' and decodeCasting's own ground: a declared count
// cannot ask for memory the form does not carry the bytes for.
//
// ONE STATE IS REFUSED: tails out of ascending key order, or two tails on one
// key. The order is canonical and one tail per cell is the record's own rule, so
// a form carrying either encodes differently from the world it decodes to and
// the digest stops being a function of the logical world. It is the same refusal
// the effect list already takes and it is the only one here. Unknown spell
// bytes remain stored but cannot cast without a matching world spell row.
func decodeScriptState(data []byte) ([relationSlots]uint8, []cellTail, int, error) {
	var modes [relationSlots]uint8
	if len(data) < relationSlots+tailCountLen {
		return modes, nil, 0, fmt.Errorf(
			"sim: byte form truncated: the formation block and the cell-tail count need %d byte(s), %d left",
			relationSlots+tailCountLen, len(data))
	}
	copy(modes[:], data[:relationSlots])
	off := relationSlots
	n := binary.LittleEndian.Uint32(data[off : off+tailCountLen])
	off += tailCountLen
	span := int64(n) * tailRecordLen
	if avail := int64(len(data) - off); span > avail {
		return modes, nil, 0, fmt.Errorf(
			"sim: byte form declares %d cell tail(s), whose records are %d byte(s), and carries %d byte(s) after the count",
			n, span, avail)
	}
	tails := make([]cellTail, 0, n)
	for i := 0; i < int(n); i++ {
		o := off + tailRecordLen*i
		var t cellTail
		t.Key = binary.LittleEndian.Uint16(data[o : o+2])
		copy(t.Bytes[:], data[o+2:o+tailRecordLen])
		if i > 0 && t.Key <= tails[i-1].Key {
			return modes, nil, 0, fmt.Errorf(
				"sim: cell tail %d is at key %d, which is not past its predecessor's %d",
				i, t.Key, tails[i-1].Key)
		}
		tails = append(tails, t)
	}
	if n == 0 {
		tails = nil
	}
	return modes, tails, off + int(span), nil
}
