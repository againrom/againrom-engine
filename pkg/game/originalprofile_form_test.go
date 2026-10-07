package game

import (
	"encoding/binary"
	"testing"
)

// Compatibility-test peel only; genuine predecessor evidence is separately
// produced by exact 1104 in sim/originalprofile_test.go.
func preInstanceWeightForm1109(t *testing.T, b []byte) []byte {
	t.Helper()
	out := append([]byte(nil), b...)
	if len(out) != 0 && out[0] >= 81 {
		out = beforeStrideStateForm1115(t, out)
	}
	if out[0] >= 80 {
		span := int(binary.LittleEndian.Uint32(out[len(out)-4:]))
		if span > len(out)-38 {
			t.Fatal("invalid Player-container suffix in test form")
		}
		out = out[:len(out)-4-span]
		out[0] = 79
	}
	if out[0] >= 79 {
		span := int(binary.LittleEndian.Uint32(out[len(out)-4:]))
		if span > len(out)-38 {
			t.Fatal("invalid Structure suffix in test form")
		}
		out = out[:len(out)-4-span]
		out[0] = 78
	}
	if out[0] >= 78 {
		span := int(binary.LittleEndian.Uint32(out[len(out)-4:]))
		if span > len(out)-38 {
			t.Fatal("invalid Group suffix in test form")
		}
		out = out[:len(out)-4-span]
		out[0] = 77
	}
	if out[0] >= 77 {
		out = out[:len(out)-5]
		out[0] = 76
	}
	if out[0] >= 76 {
		base := 34 + 3*int(binary.LittleEndian.Uint32(out[30:34]))
		for i := int(binary.LittleEndian.Uint32(out[25:29])) - 1; i >= 0; i-- {
			at := base + 492*i + 457
			out = append(out[:at], out[at+35:]...)
		}
		out[0] = 75
	}
	if out[0] >= 75 {
		end := len(out) - 2500 - 4
		span := int(binary.LittleEndian.Uint32(out[end:]) & 0x7fffffff)
		out = append(out[:end-span], out[end+4:]...)
		out[0] = 74
	}
	if out[0] >= 74 {
		end := len(out) - 2500 - 4
		span := int(binary.LittleEndian.Uint32(out[end:]))
		out = append(out[:end-span], out[end+4:]...)
		out[0] = 73
	}
	return out
}

func preCurrentProfileForm1107(t *testing.T, b []byte) []byte {
	t.Helper()
	out := preInstanceWeightForm1109(t, b)
	if out[0] < 73 {
		return out
	}
	cells := int(binary.LittleEndian.Uint32(out[30:34]))
	n := int(binary.LittleEndian.Uint32(out[25:29]))
	base := 34 + 3*cells
	for i := n - 1; i >= 0; i-- {
		at := base + 457*i + 456
		out = append(out[:at], out[at+1:]...)
	}
	out[0] = 72
	return out
}
