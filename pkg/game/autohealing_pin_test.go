package game

import (
	"bytes"
	"encoding/binary"
	"testing"
)

// Construct an explicit legacy-policy fixture. Form95 cannot carry the new
// player policy and deliberately resumes the historical quarter-pool rule.
// This only removes the additive policy table; actor floors and every earlier
// byte stay intact. Frozen predecessor files are never regenerated here.
func beforeAutoHealing1191(t *testing.T, form []byte) []byte {
	t.Helper()
	form = beforeNativeTrainingForm(t, form)
	out := bytes.Clone(form)
	if len(out) == 0 || out[0] != 96 {
		return out
	}
	if len(out) < 46 || string(out[len(out)-4:]) != "AHL1" {
		t.Fatal("invalid form96 fixture footer")
	}
	span := int(binary.LittleEndian.Uint16(out[len(out)-6:]))
	start := len(out) - 6 - span
	if span < 6 || span > 251 || start < 34 || span != 1+5*int(out[start]) {
		t.Fatal("invalid form96 fixture policy table")
	}
	prior := -1
	for i := start + 1; i < len(out)-6; i += 5 {
		slot := int(out[i])
		if slot >= 50 || slot <= prior {
			t.Fatal("invalid form96 fixture player order")
		}
		prior = slot
	}
	out = out[:start]
	out[0] = 95
	return out
}
