package game

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

func sessionHeadWorld(t *testing.T) *sim.World {
	t.Helper()
	w, err := sim.NewWorld(7, sim.Bounds{Width: 4, Height: 4}, sim.ModeCanonical, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	return w
}

func TestCurrentSessionHeadPreservesNativeAbsenceAndOrdinaryTimer(t *testing.T) {
	for _, retained := range [][48]byte{{}, {0: 17, 47: 29}} {
		w := sessionHeadWorld(t)
		before := w.Hash()
		for cycle := range 2 {
			doc := sav.DocumentData{World: &sav.DocumentWorldData{}}
			doc.World.Session.Raw08 = savedSessionHead(w.RawSessionHead(), retained)
			wire := doc.World.Session.Raw08
			if retained == ([48]byte{}) && (binary.LittleEndian.Uint64(wire[16:24]) != 10000000 || binary.LittleEndian.Uint32(wire[32:36]) != 10000) {
				t.Fatal("ordinary timer constructor changed", wire)
			}
			a := currentActionData{Version: 1, AbsentSessionHead: captureAbsentSessionHead(&doc, w)}
			raw, err := json.Marshal(a)
			if err != nil {
				t.Fatal(err)
			}
			if err := sav.SetNativeActions(&doc.State, raw); err != nil {
				t.Fatal(err)
			}
			policy, err := readCurrentActions(&doc)
			if err != nil {
				t.Fatal(err)
			}
			w.SetRawSessionHead(wire)
			if err := restoreAbsentSessionHead(w, &doc, policy.AbsentSessionHead); err != nil {
				t.Fatal(err)
			}
			if w.Hash() != before || doc.World.Session.Raw08 != wire {
				t.Fatal("native absence or ordinary head changed", cycle)
			}
		}
	}
}

func TestCurrentSessionHeadEveryOrdinaryByteEditWins(t *testing.T) {
	w := sessionHeadWorld(t)
	doc := sav.DocumentData{World: &sav.DocumentWorldData{}}
	doc.World.Session.Raw08 = savedSessionHead([48]byte{}, [48]byte{})
	anchor := captureAbsentSessionHead(&doc, w)
	original := doc.World.Session.Raw08
	for offset := range original {
		doc.World.Session.Raw08 = original
		doc.World.Session.Raw08[offset] ^= 0x5a
		w.SetRawSessionHead(doc.World.Session.Raw08)
		if err := restoreAbsentSessionHead(w, &doc, anchor); err != nil {
			t.Fatal(err)
		}
		if w.RawSessionHead() != doc.World.Session.Raw08 {
			t.Fatal("absence policy hid an ordinary byte edit", offset)
		}
		if got := captureAbsentSessionHead(&doc, w); got != nil {
			t.Fatal("present current head acquired absence policy", offset)
		}
	}
}

func TestCurrentSessionHeadMalformedPolicyIsAtomicAndLegacyIsOrdinary(t *testing.T) {
	w := sessionHeadWorld(t)
	head := [48]byte{0: 3, 16: 7, 47: 11}
	w.SetRawSessionHead(head)
	doc := sav.DocumentData{World: &sav.DocumentWorldData{}}
	doc.World.Session.Raw08 = head
	for _, anchor := range [][]byte{{1}, make([]byte, 31), make([]byte, 33)} {
		if err := restoreAbsentSessionHead(w, &doc, anchor); err == nil || w.RawSessionHead() != head {
			t.Fatal("malformed policy accepted or mutated current head", len(anchor), err)
		}
	}
	anchor := captureAbsentSessionHead(&doc, sessionHeadWorld(t))
	if err := restoreAbsentSessionHead(w, &sav.DocumentData{}, anchor); err == nil || w.RawSessionHead() != head {
		t.Fatal("city accepted a mission head policy", err)
	}
	if err := restoreAbsentSessionHead(w, &doc, nil); err != nil || w.RawSessionHead() != head {
		t.Fatal("legacy policy altered ordinary data", err)
	}
	bad := append([]byte(nil), anchor...)
	bad[0] ^= 1
	if bytes.Equal(bad, anchor) {
		t.Fatal("ineffective anchor mutation")
	}
	if err := restoreAbsentSessionHead(w, &doc, bad); err != nil || w.RawSessionHead() != head {
		t.Fatal("stale anchor altered ordinary data", err)
	}
}
