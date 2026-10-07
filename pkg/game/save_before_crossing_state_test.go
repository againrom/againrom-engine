package game

import (
	"bytes"
	"encoding/binary"
	"testing"
)

// Independent test-only peel of the successor's bounded trailing section.
// It never uses a current serializer to construct the historical expectation.
func beforeCrossingStateForm1115(t *testing.T, raw []byte) []byte {
	t.Helper()
	raw = beforeActionClockForm1146(t, raw)
	if len(raw) > 0 && (raw[0] == 83 || raw[0] == 84 || raw[0] == 85) {
		raw = beforeCellStateForm1115(t, raw)
	}
	if len(raw) < 38 || raw[0] != 81 && raw[0] != 82 {
		t.Fatal("unexpected crossing compatibility form", len(raw))
	}
	end := len(raw)
	if raw[0] == 82 {
		span := uint64(binary.LittleEndian.Uint32(raw[end-4:]))
		if span > uint64(end-38) {
			t.Fatal("invalid crossing compatibility suffix span", span, end)
		}
		end -= 4 + int(span)
	}
	out := bytes.Clone(raw[:end])
	out[0] = 81
	return out
}
