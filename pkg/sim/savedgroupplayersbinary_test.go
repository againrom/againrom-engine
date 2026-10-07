package sim

import (
	"bytes"
	"encoding/binary"
	"reflect"
	"testing"
)

// Independent form80 extension of historical pins: the form79 prefix is
// unchanged and one zero uint32 means absent exact Player provenance.
func widenedSavedGroupPlayerPin(old []byte) []byte {
	out := append(append([]byte(nil), old...), 0, 0, 0, 0)
	out[0] = 80
	return widenedNativeStridePin(out)
}

func TestSavedGroupPlayers80IndependentHistoricalPin(t *testing.T) {
	for _, tc := range []struct {
		form []byte
		hash uint64
	}{{pinBytes, prePlayerContainersPinDigest}, {rtfBytes, prePlayerContainersRoutedDigest}} {
		tc.form = strippedNativeStridePin(tc.form)
		old := append([]byte(nil), tc.form[:len(tc.form)-4]...)
		old[0] = 79
		if fnv1a(old) != tc.hash {
			t.Fatal("form80 adapter changed an immutable form79 pin")
		}
		if !bytes.Equal(tc.form[len(tc.form)-4:], []byte{0, 0, 0, 0}) || tc.form[0] != 80 {
			t.Fatal("form80 pin must carry only absent Player-container provenance")
		}
	}
}

func TestSavedGroupPlayers80NativeSection(t *testing.T) {
	w := savedGroupWorld(t)
	absent, err := w.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	if err := w.ImportSavedGroupPlayers([]SavedGroupPlayer{{101, 1}, {205, 1}}, []SavedGroupContainer{{71, 101}, {72, 205}}); err != nil {
		t.Fatal(err)
	}
	form, err := w.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	// Presence, highwater72, Players101/205 at duplicate slot1, exact Group
	// mapping71->101/72->205. The original Group section was not widened.
	want := []byte{1, 72, 0, 0, 0, 2, 0, 0, 0, 101, 0, 0, 0, 1, 0, 0, 0, 205, 0, 0, 0, 1, 0, 0, 0,
		2, 0, 0, 0, 71, 0, 0, 0, 101, 0, 0, 0, 72, 0, 0, 0, 205, 0, 0, 0, 45, 0, 0, 0}
	prefix := strippedNativeStridePin(form)
	absentPrefix := strippedNativeStridePin(absent)
	if prefix[0] != 80 || !bytes.Equal(prefix[:len(prefix)-len(want)], absentPrefix[:len(absentPrefix)-4]) || !bytes.Equal(prefix[len(prefix)-len(want):], want) {
		t.Fatal("form80 literal extension differs")
	}
	var restored World
	if err := restored.UnmarshalBinary(form); err != nil {
		t.Fatal(err)
	}
	if restored.Hash() != w.Hash() || !reflect.DeepEqual(restored.savedGroups, w.savedGroups) {
		t.Fatal("native Player containers/highwater lost")
	}
	clear(form)
	players, present := restored.SavedGroupPlayers()
	if !present || !reflect.DeepEqual(players, []SavedGroupPlayer{{101, 1}, {205, 1}}) {
		t.Fatal("native Player registry aliases source bytes")
	}
}

func TestSavedGroupPlayers80MalformedPayloadAtomic(t *testing.T) {
	w := savedGroupWorld(t)
	if err := w.ImportSavedGroupPlayers([]SavedGroupPlayer{{101, 1}, {205, 2}}, []SavedGroupContainer{{71, 101}, {72, 205}}); err != nil {
		t.Fatal(err)
	}
	valid, err := w.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	start := len(valid) - entityIDFloorLen - spellDeliverySpanLen - 97
	for _, tc := range []struct {
		name string
		edit func([]byte) []byte
	}{
		{"presence", func(b []byte) []byte { b[start] = 2; return b }},
		{"absent with players", func(b []byte) []byte { b[start] = 0; return b }},
		{"hostile Player count", func(b []byte) []byte { binary.LittleEndian.PutUint32(b[start+5:], ^uint32(0)); return b }},
		{"hostile Group count", func(b []byte) []byte { binary.LittleEndian.PutUint32(b[start+25:], ^uint32(0)); return b }},
		{"zero Player", func(b []byte) []byte { binary.LittleEndian.PutUint32(b[start+9:], 0); return b }},
		{"duplicate Player", func(b []byte) []byte { binary.LittleEndian.PutUint32(b[start+17:], 101); return b }},
		{"unsorted Player", func(b []byte) []byte { binary.LittleEndian.PutUint32(b[start+9:], 301); return b }},
		{"zero Group", func(b []byte) []byte { binary.LittleEndian.PutUint32(b[start+29:], 0); return b }},
		{"duplicate Group", func(b []byte) []byte { binary.LittleEndian.PutUint32(b[start+37:], 71); return b }},
		{"missing Group", func(b []byte) []byte { binary.LittleEndian.PutUint32(b[start+37:], 73); return b }},
		{"unknown Player", func(b []byte) []byte { binary.LittleEndian.PutUint32(b[start+33:], 999); return b }},
		{"traversal order", func(b []byte) []byte {
			binary.LittleEndian.PutUint32(b[start+33:], 205)
			binary.LittleEndian.PutUint32(b[start+41:], 101)
			return b
		}},
		{"highwater", func(b []byte) []byte { binary.LittleEndian.PutUint32(b[start+1:], 71); return b }},
		{"hostile span", func(b []byte) []byte {
			binary.LittleEndian.PutUint32(b[len(b)-entityIDFloorLen-spellDeliverySpanLen-48:], ^uint32(0))
			return b
		}},
		{"truncated footer", func(b []byte) []byte { return b[:len(b)-1] }},
		{"late bad grid", func(b []byte) []byte { binary.LittleEndian.PutUint32(b[30:], ^uint32(0)); return b }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			bad := tc.edit(append([]byte(nil), valid...))
			before := w.Hash()
			if err := w.UnmarshalBinary(bad); err == nil || w.Hash() != before {
				t.Fatal("malformed Player payload partially published", err)
			}
		})
	}
}

func TestSavedGroupPlayers80AbsentRegistryHighwaterAndExhaustion(t *testing.T) {
	w := savedGroupWorld(t)
	groups, orders, _ := w.SavedGroups()
	if err := w.ImportSavedGroups(groups[:1], orders); err != nil {
		t.Fatal(err)
	}
	form, err := w.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	// No Player identities, but removed ID72 must remain reserved. This
	// 13-byte payload carries absent flag, highwater72, and two zero counts.
	want := []byte{0, 72, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 13, 0, 0, 0}
	old := strippedNativeStridePin(form)
	if !bytes.Equal(old[len(old)-len(want):], want) {
		t.Fatal("absent registry discarded removed-ID highwater")
	}
	var restored World
	if err := restored.UnmarshalBinary(form); err != nil {
		t.Fatal(err)
	}
	if players, present := restored.SavedGroupPlayers(); present || players != nil {
		t.Fatal("highwater alone fabricated Player identities")
	}
	restored.commandSavedGroup([]int{0}, 0, cell{})
	if g := restored.savedGroupFor(restored.entities[0].ID); g == nil || g.ID != 73 || g.ContainerID != 0 {
		t.Fatal("absent-registry SAVE reused a removed identity", g)
	}
	if err := w.ImportSavedGroups(groups, orders); err == nil {
		t.Fatal("explicit Group update resurrected a removed identity")
	}
	if err := w.ImportSavedGroupPlayers([]SavedGroupPlayer{{101, 1}}, []SavedGroupContainer{{71, 101}}); err != nil {
		t.Fatal(err)
	}
	w.savedGroups.HighWater = ^uint32(0)
	before := w.Hash()
	if w.newSavedCommandGroup([]int{0}, 0, cell{}, true, 101, true) || w.Hash() != before {
		t.Fatal("exhausted identity counter mutated command state")
	}
}
