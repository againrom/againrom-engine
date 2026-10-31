package sim

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"testing"
)

// structureRowWorld is a 4x4 World holding one structure loaded from a file:
// a Building at 1/2 with archive index 3, Token key 21 and Token row 74.
func structureRowWorld(t *testing.T) *World {
	t.Helper()
	w, err := NewWorld(11, Bounds{Width: 4, Height: 4}, ModeCanonical, make([]byte, 16), nil)
	if err != nil {
		t.Fatal(err)
	}
	st := Structure{ID: 7, Kind: 1, Col: 1, Row: 2, Width: 1, Height: 1, Attach: 1, Blocking: 1}
	s := SavedStructure{ID: st.ID, Class: SavedBuilding, SourceKey: 21, ArchiveIndex: 3, RuntimeID: 31,
		Position: [12]byte{1, 2}, Token0C: 74, Token0E: 1, Token18: 2, Kind: 1, Blocking: st.Blocking}
	s.Base52[14], s.Base52[15] = st.Width, st.Height
	binary.LittleEndian.PutUint32(s.Base52[18:], st.Blocking)
	cells := []SavedStructureCell{{Cell: 2<<8 | 1, ID: st.ID, HasStructure: true}}
	if err := w.ImportOriginalStructures([]Structure{st}, []SavedStructure{s}, cells, make([]byte, 16)); err != nil {
		t.Fatal(err)
	}
	return w
}

// A World whose structure rows all fit a byte writes the bytes and hash it
// wrote before a structure row could be wider: the digests are pinned from
// that build.
func TestStructureRowsWithinAByteKeepTheirForm(t *testing.T) {
	w := structureRowWorld(t)
	form, err := w.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(form)
	const wantForm, wantHash = "8a0c272d912caccbe24115790a9466504fa5be48f488b53decc67f74161d6331", uint64(0xfb4114809efa98f4)
	if got := hex.EncodeToString(sum[:]); got != wantForm || w.Hash() != wantHash {
		t.Fatalf("form %d %s hash %#x, want %s %#x", form[0], got, w.Hash(), wantForm, wantHash)
	}
}

// structureRowRoundTrip decodes form and requires the same World back: the
// same bytes and hash, and the saved structure want.
func structureRowRoundTrip(t *testing.T, w *World, form []byte, want SavedStructure) {
	t.Helper()
	if err := CheckSaveForm(form); err != nil {
		t.Fatal(err)
	}
	var restored World
	if err := restored.UnmarshalBinary(form); err != nil {
		t.Fatal(err)
	}
	got, _, _ := restored.SavedStructures()
	if len(got) != 1 || got[0].Token0C != want.Token0C || got[0].Class != want.Class || got[0].Pointer6C != want.Pointer6C || restored.Hash() != w.Hash() {
		t.Fatalf("restored %+v hash %#x, want row %d class %d word %#x hash %#x", got, restored.Hash(), want.Token0C, want.Class, want.Pointer6C, w.Hash())
	}
	again, err := restored.MarshalBinary()
	if err != nil || string(again) != string(form) {
		t.Fatalf("the restored World writes another form: %v", err)
	}
}

// A saved structure's row above a byte keeps its low byte in the structure
// section and its high byte in the token row form, after the entity pairs,
// under the structure mark. The World decodes back to the same row, bytes
// and hash; a pair naming no structure is refused.
func TestStructureRowAboveAByteRoundTripsThroughItsForm(t *testing.T) {
	w := structureRowWorld(t)
	w.savedStructures[0].Token0C = 330
	form, err := w.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	section := len(form) - 9 - tokenRowPairLen
	if form[0] != tokenRowFormVersion || binary.LittleEndian.Uint32(form[section:]) != tokenRowStructure || form[section+4] != 1 {
		t.Fatalf("form %d pair % x, want %d and structure 0 high byte 1", form[0], form[section:section+tokenRowPairLen], tokenRowFormVersion)
	}
	structureRowRoundTrip(t, w, form, w.savedStructures[0])
	for name, edit := range map[string]func([]byte){
		"no such structure": func(b []byte) { b[section] = 1 },
		"high byte 0":       func(b []byte) { b[section+4] = 0 },
	} {
		bad := []byte(string(form))
		edit(bad)
		var refused World
		if err := refused.UnmarshalBinary(bad); err == nil {
			t.Fatalf("%s: the form decodes", name)
		}
	}
}

// A Pointer keeps its class and its +0x6c word through the structure
// section; a Building holding a Pointer word is refused.
func TestSavedPointerKeepsItsWordThroughTheForm(t *testing.T) {
	w := structureRowWorld(t)
	plain, err := w.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	w.savedStructures[0].Class, w.savedStructures[0].Pointer6C = SavedPointer, 0x01020304
	form, err := w.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	if len(form) != len(plain)+4 {
		t.Fatalf("the Pointer's form is %d bytes, the Building's %d; want 4 more", len(form), len(plain))
	}
	structureRowRoundTrip(t, w, form, w.savedStructures[0])
	w.savedStructures[0].Class = SavedBuilding
	if _, err := w.MarshalBinary(); err == nil {
		t.Fatal("a Building with a Pointer word marshals")
	}
}
