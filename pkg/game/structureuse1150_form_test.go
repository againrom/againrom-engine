package game

import (
	"bytes"
	"encoding/binary"
	"testing"
)

// Earlier formats had neither immutable use metadata nor a pending use order.
// Peel only that new section when asserting their frozen byte/hash fixtures.
func beforeStructureUseForm1150(t *testing.T, raw []byte) []byte {
	t.Helper()
	raw = beforeSavedFormationForm1159(t, raw)
	if len(raw) == 0 || raw[0] != 89 {
		return raw
	}
	if len(raw) < 38 {
		t.Fatal("truncated structure-use fixture")
	}
	span := uint64(binary.LittleEndian.Uint32(raw[len(raw)-4:]))
	if span > uint64(len(raw)-38) {
		t.Fatal("invalid structure-use fixture span")
	}
	out := bytes.Clone(raw[:len(raw)-4-int(span)])
	out[0] = 88
	return out
}
