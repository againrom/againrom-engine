package sim

import (
	"encoding/binary"
	"fmt"
)

// RepairDanglingReferences rewrites a byte form written by an OLDER BUILD OF
// THIS PROGRAM whose removal path left a reference into its own entity slice
// pointing at a record the form does not carry, and reports how many fields it
// normalised.
//
// IT EXISTS FOR SAVES ALREADY ON DISK AND FOR NOTHING ELSE (owner). remove
// and the raise arm no longer produce this shape, so no save taken from this
// build onward needs it; the owner had five current-format saves that could
// not be loaded at all, one of them 175,434 ticks into mission 30, and a
// player's own progress is not something a fixed encoder gives back.
//
// IT DOES NOT WEAKEN THE DECODER, which is the point of doing this here
// rather than by relaxing the refusal. This converts a form our encoder
// should never have written INTO the form it should have, once, in a
// separate file the player chooses to run.
//
// THE VERSION IS NOT TOUCHED and a form of any other version is refused: the
// offsets below belong to one layout, and this function is worth exactly as
// much as the assertion that they are the current one's.
//
// The input is not modified.
func RepairDanglingReferences(data []byte) ([]byte, int, error) {
	if len(data) < headerLen {
		return nil, 0, fmt.Errorf("sim: byte form truncated: %d byte(s), header alone is %d", len(data), headerLen)
	}
	if data[0] != formatVersion {
		return nil, 0, fmt.Errorf("sim: byte form version %d, and this repair knows the layout of %d only",
			data[0], formatVersion)
	}
	b := Bounds{
		Width:  int32(binary.LittleEndian.Uint32(data[17:21])),
		Height: int32(binary.LittleEndian.Uint32(data[21:25])),
	}
	cells := gridCells(b)
	if int64(len(data)-headerLen) < 3*cells+relationLen {
		return nil, 0, fmt.Errorf("sim: byte form truncated: %d byte(s) after a %d-byte header",
			len(data)-headerLen, headerLen)
	}
	records := headerLen + 3*int(cells)
	n := int(binary.LittleEndian.Uint32(data[25:29]))
	if span := int64(n) * entityLen; span > int64(len(data)-records) {
		return nil, 0, fmt.Errorf("sim: byte form declares %d entities and does not carry their records", n)
	}

	out := append([]byte(nil), data...)

	// Every id the form actually carries, read before anything is judged: a
	// reference is legal exactly when this set holds it, which is the second
	// pass the decoder performs and cannot be answered one record at a time.
	held := make(map[uint32]bool, n)
	for i := 0; i < n; i++ {
		o := records + entityLen*i
		held[binary.LittleEndian.Uint32(out[o:o+4])] = true
	}

	fixed := 0
	for i := 0; i < n; i++ {
		o := records + entityLen*i
		if out[o+265] == 1 && !held[binary.LittleEndian.Uint32(out[o+261:o+265])] {
			binary.LittleEndian.PutUint32(out[o+261:o+265], 0)
			out[o+265] = 0
			fixed++
		}
		// The attack half of the same pair. remove always cleared this one, so
		// no save is expected to carry it; the raise arm did not, and a form
		// that has one is refused by the same second pass for the same reason.
		if out[o+48] == 1 && !held[binary.LittleEndian.Uint32(out[o+44:o+48])] {
			binary.LittleEndian.PutUint32(out[o+44:o+48], 0)
			out[o+48] = 0
			out[o+49] = byte(AttackReady)
			binary.LittleEndian.PutUint32(out[o+50:o+54], 0)
			fixed++
		}
	}
	return out, fixed, nil
}
