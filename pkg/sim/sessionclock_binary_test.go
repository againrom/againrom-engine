package sim

import (
	"bytes"
	"encoding/binary"
	"testing"
)

// Independent form77 transcription: the full form76 payload is unchanged,
// followed by absent presence and four zero FullTick bytes for native worlds.
func widenedSessionClockPin(old []byte) []byte {
	out := append(append([]byte(nil), old...), 0, 0, 0, 0, 0)
	out[0] = 77
	return widenedSavedGroupPin(out)
}

func strippedSessionClockPin(form []byte) []byte {
	out := strippedNativeStridePin(form)
	if out[0] >= 80 {
		span := int(binary.LittleEndian.Uint32(out[len(out)-4:]))
		out = out[:len(out)-4-span]
		out[0] = 79
	}
	if out[0] >= 79 {
		span := int(binary.LittleEndian.Uint32(out[len(out)-4:]))
		out = out[:len(out)-4-span]
		out[0] = 78
	}
	if out[0] >= 78 {
		span := int(binary.LittleEndian.Uint32(out[len(out)-4:]))
		out = out[:len(out)-4-span]
		out[0] = 77
	}
	if out[0] >= 77 {
		out = out[:len(out)-5]
		out[0] = 76
	}
	return out
}

func TestSessionClockNativeHashBothWordsAndFreshNextPhase(t *testing.T) {
	w := clockWorld1112(t, 12, 584, []Entity{{ID: 1, HP: 50, MaxHP: 100, HealthRegenPeriod: 100}})
	b, err := w.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	// entityIDFloor (form94) closes the form outside this section entirely.
	end := len(b) - entityIDFloorLen - spellDeliverySpanLen
	if b[0] != formatVersion || !bytes.Equal(b[end-65:end-60], []byte{1, 0x48, 2, 0, 0}) || binary.LittleEndian.Uint64(b[1:9]) != 12 {
		t.Fatal("literal form77 suffix/header", b[end-65:end-60])
	}
	var cold World
	if err := cold.UnmarshalBinary(b); err != nil {
		t.Fatal(err)
	}
	assertClock1112(t, &cold, 12, 584)
	if cold.Hash() != w.Hash() {
		t.Fatal("native hash")
	}
	other := clockWorld1112(t, 12, 585, w.entities)
	if w.Hash() == other.Hash() {
		t.Fatal("independent FullTick is not hashed")
	}
	Step(w, nil)
	Step(&cold, nil)
	if cold.entities[0].HP != 52 || w.Hash() != cold.Hash() {
		t.Fatal("fresh native next phase lost full-counter filter")
	}
	assertClock1112(t, &cold, 13, 584)
	Step(&cold, nil)
	assertClock1112(t, &cold, 14, 584)
	Step(&cold, nil)
	assertClock1112(t, &cold, 15, 585)
}

func TestSessionClockNativeMalformedAndLateRefusalsAreAtomic(t *testing.T) {
	w := clockWorld1112(t, 9343, 584, nil)
	valid, err := w.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	for _, edit := range []func([]byte) []byte{
		func(b []byte) []byte { b[len(b)-entityIDFloorLen-spellDeliverySpanLen-65] = 2; return b },
		func(b []byte) []byte { b[len(b)-entityIDFloorLen-spellDeliverySpanLen-65] = 0; return b },
		func(b []byte) []byte { binary.LittleEndian.PutUint64(b[1:9], 0x100000000); return b },
		func(b []byte) []byte { return b[:len(b)-1] },
		func(b []byte) []byte {
			// A different valid pair followed by a later bad grid count: the
			// suffix decoder cannot commit its clock before the world passes.
			binary.LittleEndian.PutUint64(b[1:9], 12)
			binary.LittleEndian.PutUint32(b[len(b)-entityIDFloorLen-spellDeliverySpanLen-48:], 99)
			binary.LittleEndian.PutUint32(b[30:34], 0xffffffff)
			return b
		},
	} {
		bad := edit(append([]byte(nil), valid...))
		before := w.Hash()
		if err := w.UnmarshalBinary(bad); err == nil || w.Hash() != before {
			t.Fatal("native refusal changed receiver", err)
		}
		assertClock1112(t, w, 9343, 584)
	}
}
