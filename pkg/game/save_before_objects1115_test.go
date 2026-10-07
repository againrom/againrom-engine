package game

import (
	"bytes"
	"testing"
)

// Form84 appends the optional object registry. A predecessor must acquire only
// its absent zero4 footer. Independently peel that footer, never regenerate the
// historical native83 bytes or hashes and never discard a populated registry.
func beforeObjects1115Form(t *testing.T, raw []byte) []byte {
	t.Helper()
	raw = beforeActionClockForm1146(t, raw)
	if len(raw) >= 5 && raw[0] == 85 {
		if !bytes.Equal(raw[len(raw)-4:], []byte{0, 0, 0, 0}) {
			t.Fatal("pre-carried-resume native state acquired a populated resume footer")
		}
		raw = bytes.Clone(raw[:len(raw)-4])
		raw[0] = 84
	}
	if len(raw) >= 5 && raw[0] == 84 {
		if !bytes.Equal(raw[len(raw)-4:], []byte{0, 0, 0, 0}) {
			t.Fatal("pre-object native state acquired a populated object footer")
		}
		out := bytes.Clone(raw[:len(raw)-4])
		out[0] = 83
		return out
	}
	if len(raw) == 0 || raw[0] != 83 {
		t.Fatal("unexpected pre-object native form")
	}
	return raw
}
