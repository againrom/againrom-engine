package sim

import (
	"bytes"
	"encoding/binary"
	"testing"
)

// Independent form adapters preserve the frozen predecessor fixtures.
func widenedAttackNoticePin(old []byte) []byte {
	out := append(bytes.Clone(old), 0, 0, 0, 0)
	out[0] = 93
	return out
}

// widenedEntityIDFloorPin is form94's own independent transcription: an
// eight-byte little-endian floor appended outside every earlier section
// (encode's own outermost wrap), the caller's own transcribed value rather
// than one read off the fixture world — the same independence
// widenedAttackNoticePin and its own predecessors keep.
func widenedEntityIDFloorPin(old []byte, floor uint64) []byte {
	out := binary.LittleEndian.AppendUint64(bytes.Clone(old), floor)
	out[0] = 95
	return append(out, 0, 0, 0, 0)
}

// entityIDFloorDigest1177 is areaHeaderDigest1164's own frozen form91 check,
// carried one step further: form94's own outermost floor wrap goes on top of
// the form93 bytes areaHeaderDigest1164 verifies internally but does not
// itself return.
func entityIDFloorDigest1177(current []byte, previous91 uint64, floor uint64) uint64 {
	old := strippedAttackNoticePin(current)
	old[0] = 91
	if fnv1a(old) != previous91 {
		panic("form91 frozen predecessor digest changed")
	}
	old[0] = 92
	return fnv1a(widenedEntityIDFloorPin(widenedAttackNoticePin(old), floor))
}

func strippedAttackNoticePin(form []byte) []byte {
	out := strippedSpellDeliveryPin(form)
	// Form94's entityIDFloor is appended OUTSIDE attack notices (encode's own
	// outermost wrap), so it must come off before the count below is read, or
	// this reads eight of its bytes as a bogus attack-notice span.
	if len(out) > 0 && out[0] >= 94 {
		out = out[:len(out)-entityIDFloorLen]
		out[0] = 93
	}
	if len(out) > 0 && out[0] >= 93 {
		n := int(binary.LittleEndian.Uint32(out[len(out)-4:]))
		out = out[:len(out)-4-n]
		out[0] = 92
	}
	return out
}

func TestAttackNoticeNativeResume(t *testing.T) {
	w, rule := distantSpellWorld(t)
	w.ordinaryAreaEffect(0, 1, rule, 0)
	for scans := 0; scans < 22; scans++ {
		data, err := w.MarshalBinary()
		if err != nil {
			t.Fatal(err)
		}
		var back World
		if err := back.UnmarshalBinary(data); err != nil {
			t.Fatal(err)
		}
		if back.Hash() != w.Hash() || back.attackNoticeAt(1) != w.attackNoticeAt(1) {
			t.Fatal("notice lost at scan", scans)
		}
		for range 32 {
			Step(w, nil)
			Step(&back, nil)
			if w.Hash() != back.Hash() {
				t.Fatal("resume diverged at scan", scans)
			}
		}
		w.stampAttackNotices(make([]byte, 48*48), []int{1}, true)
	}
}

func TestAttackNoticeMalformedSuffixIsAtomic(t *testing.T) {
	w, _ := distantSpellWorld(t)
	w.flipOnBlow(0, 1)
	valid := mustMarshal(t, w)
	// entityIDFloor (form94) closes the form outside this section entirely.
	end := len(valid) - entityIDFloorLen - spellDeliverySpanLen
	start := end - 11
	for _, edit := range []func([]byte) []byte{
		func(b []byte) []byte { return b[:len(b)-1] },
		func(b []byte) []byte { binary.LittleEndian.PutUint32(b[end-4:], 0xffffffff); return b },
		func(b []byte) []byte { binary.LittleEndian.PutUint32(b[start:], 99); return b },
		func(b []byte) []byte { clear(b[start+4 : start+7]); return b },
		func(b []byte) []byte {
			out := append(bytes.Clone(b[:end-4]), b[start:start+7]...)
			return binary.LittleEndian.AppendUint32(out, 14)
		},
	} {
		before := w.Hash()
		if err := w.UnmarshalBinary(edit(bytes.Clone(valid))); err == nil || w.Hash() != before {
			t.Fatal("malformed notice accepted or partly published", err)
		}
	}
	// A saved order is a separate authoritative owner of the same fields.
	if err := w.ImportSavedGroups(nil, []SavedActorOrder{{Entity: 2}}); err != nil {
		t.Fatal(err)
	}
	data := mustMarshal(t, w)
	bad := append(bytes.Clone(data[:len(data)-entityIDFloorLen-spellDeliverySpanLen-4]), valid[start:start+7]...)
	bad = binary.LittleEndian.AppendUint32(bad, 7)
	before := w.Hash()
	if err := w.UnmarshalBinary(bad); err == nil || w.Hash() != before {
		t.Fatal("duplicate notice authority admitted", err)
	}
}
