package game

import (
	"encoding/binary"
	"testing"
)

func beforeSpellDelivery1183(t *testing.T, form []byte) []byte {
	t.Helper()
	out := beforeAutoHealing1191(t, form)
	if len(out) == 0 || out[0] != 95 {
		return out
	}
	n := int(binary.LittleEndian.Uint32(out[len(out)-4:]))
	if n > len(out)-4 {
		t.Fatal("invalid form95 fixture")
	}
	if n != 0 {
		raw := out[len(out)-4-n : len(out)-4]
		rows := int(binary.LittleEndian.Uint32(raw))
		if n != 12+rows*10 || binary.LittleEndian.Uint64(raw[4+rows*10:]) != 0 {
			t.Fatal("historical fixture has a paid or pending cast")
		}
	}
	out = out[:len(out)-4-n]
	out[0] = 94
	return out
}
