package game

import (
	"bytes"
	"encoding/binary"
	"hash/fnv"
	"testing"

	"againrom/pkg/sim"
)

// Independent peel for frozen predecessor fixtures. New action-clock state is
// tested separately; every byte the predecessor could write remains exact.
func beforeActionClockForm1146(t *testing.T, raw []byte) []byte {
	t.Helper()
	raw = beforeStructureUseForm1150(t, raw)
	if len(raw) > 0 && raw[0] == 88 {
		if len(raw) < 38 {
			t.Fatal("truncated fire-history fixture")
		}
		span := uint64(binary.LittleEndian.Uint32(raw[len(raw)-4:]))
		if span%2 != 0 || span > 2*65536 || span > uint64(len(raw)-38) {
			t.Fatal("invalid fire-history fixture span")
		}
		// Earlier forms could not store this history. Their LOAD default is
		// asserted separately; preserve every byte they could represent.
		raw = bytes.Clone(raw[:len(raw)-4-int(span)])
		raw[0] = 87
	}
	if len(raw) > 0 && raw[0] == 87 {
		if len(raw) < 38 || !bytes.Equal(raw[len(raw)-4:], []byte{0, 0, 0, 0}) {
			t.Fatal("historical fixture acquired native Roam counter")
		}
		raw = bytes.Clone(raw[:len(raw)-4])
		raw[0] = 86
	}
	if len(raw) == 0 || raw[0] != 86 {
		return raw
	}
	if len(raw) < 38 {
		t.Fatal("truncated action-clock fixture")
	}
	span := uint64(binary.LittleEndian.Uint32(raw[len(raw)-4:]))
	if span%8 != 0 || span > uint64(len(raw)-38) {
		t.Fatal("invalid action-clock fixture span")
	}
	out := bytes.Clone(raw[:len(raw)-4-int(span)])
	out[0] = 85
	return out
}

func beforeActionClockHash1146(t *testing.T, w *sim.World) uint64 {
	t.Helper()
	raw, err := w.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	h := fnv.New64a()
	_, _ = h.Write(beforeActionClockForm1146(t, raw))
	return h.Sum64()
}
