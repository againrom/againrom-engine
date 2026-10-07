package game

import (
	"encoding/binary"
	"hash/fnv"
	"testing"

	"againrom/pkg/sim"
)

// Independent test-only peel: form81 appends its bounded stride section and
// four-byte span after the complete form80 payload. Retained nonzero strides
// are allowed here: their absence immediately after old LOAD is checked apart.
// Never regenerate an old golden or parse its expectation via production code.
func beforeStrideStateForm1115(t *testing.T, raw []byte) []byte {
	t.Helper()
	raw = beforeActionClockForm1146(t, raw)
	if len(raw) > 0 && raw[0] >= 82 && raw[0] <= 85 {
		raw = beforeCrossingStateForm1115(t, raw)
	}
	if len(raw) < 38 || raw[0] != 80 && raw[0] != 81 {
		t.Fatal("unexpected stride compatibility form", len(raw))
	}
	end := len(raw)
	if raw[0] == 81 {
		span := uint64(binary.LittleEndian.Uint32(raw[end-4:]))
		if span > uint64(end-38) {
			t.Fatal("invalid stride compatibility suffix span", span, end)
		}
		end -= 4 + int(span)
	}
	out := append([]byte(nil), raw[:end]...)
	out[0] = 80
	return out
}

func beforeStrideState1115Hash(t *testing.T, w *sim.World) uint64 {
	t.Helper()
	raw, err := w.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	h := fnv.New64a()
	_, _ = h.Write(beforeStrideStateForm1115(t, raw))
	return h.Sum64()
}
