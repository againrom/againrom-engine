package sim

import (
	"bytes"
	"encoding/binary"
)

// Peel only the new suffix before comparing the frozen historical byte forms.
// Never discard a paid cast or a pending effect to make an old pin agree.
func strippedSpellDeliveryPin(form []byte) []byte {
	out := bytes.Clone(form)
	if len(out) == 0 || out[0] != 95 {
		return out
	}
	n := int(binary.LittleEndian.Uint32(out[len(out)-4:]))
	if n > len(out)-4 {
		panic("invalid form95 fixture")
	}
	if n != 0 {
		raw := out[len(out)-4-n : len(out)-4]
		rows := int(binary.LittleEndian.Uint32(raw))
		if n != 12+rows*10 || binary.LittleEndian.Uint64(raw[4+rows*10:]) != 0 {
			panic("historical pin carries a paid or pending cast")
		}
	}
	out = out[:len(out)-4-n]
	out[0] = 94
	return out
}
