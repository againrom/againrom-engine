package sim

import (
	"bytes"
	"encoding/binary"
	"reflect"
	"testing"
)

// Independent literal form76 layout, not the production encoder or constants.
func widenedSourceBindingPin(old []byte) []byte {
	base := 34 + 3*int(binary.LittleEndian.Uint32(old[30:34]))
	n := int(binary.LittleEndian.Uint32(old[25:29]))
	out := append([]byte(nil), old[:base]...)
	for i := 0; i < n; i++ {
		at := base + 457*i
		out = append(out, old[at:at+457]...)
		out = append(out, make([]byte, 35)...)
	}
	out = append(out, old[base+457*n:]...)
	out[0] = 76
	return out
}

func strippedSourceBindingPin(form []byte) []byte {
	out := strippedSessionClockPin(form)
	if out[0] < 76 {
		return out
	}
	base := 34 + 3*int(binary.LittleEndian.Uint32(out[30:34]))
	for i := int(binary.LittleEndian.Uint32(out[25:29])) - 1; i >= 0; i-- {
		at := base + 492*i + 457
		out = append(out[:at], out[at+35:]...)
	}
	out[0] = 75
	return out
}

func sourceBindingWorld1111(t *testing.T) *World {
	t.Helper()
	w, err := NewWorld(123, Bounds{Width: 8, Height: 8}, ModeCanonical, make([]byte, 64), []Entity{{ID: 7, X: 2, Y: 2, HP: 9, MaxHP: 31}})
	if err != nil {
		t.Fatal(err)
	}
	e := &w.entities[0]
	e.SourceBinding = SourceBinding{Class: 1, ArchiveIndex: 4, Identity: 0x10000004, RuntimeID: 73, TokenRow: 7, TypeID: 35,
		Face: 3, ClassFlags: 0xa5, DisplayBacking: 0x12345678, GroupIndex: 1, GroupSelector: 41, GroupOwnerKey: 0x10000002, GroupOwnerSlot: 9, GroupOwnerResolved: true}
	e.ActorLoad = ActorLoad{Present: true, Source: SourceActor{Class: 1}}
	return w
}

func TestSourceBinding1111WireAndHash(t *testing.T) {
	w := sourceBindingWorld1111(t)
	form, err := w.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	const at = 34 + 3*64 + 457
	want := []byte{1, 4, 0, 4, 0, 0, 16, 73, 0, 0, 0, 7, 35, 0, 3, 0xa5, 0x78, 0x56, 0x34, 0x12, 1, 0, 0, 0, 41, 0, 0, 0, 2, 0, 0, 16, 9, 0, 1}
	if form[0] != formatVersion || !bytes.Equal(form[at:at+35], want) || binary.Size(SourceBinding{}) != 35 {
		t.Fatalf("literal source tail: %x", form[at:at+35])
	}
	var restored World
	if err := restored.UnmarshalBinary(form); err != nil || restored.Hash() != w.Hash() || !reflect.DeepEqual(restored.entities[0].SourceBinding, w.entities[0].SourceBinding) {
		t.Fatalf("roundtrip: %v", err)
	}
	for i := range want {
		changed := append([]byte(nil), form...)
		changed[at+i] ^= 1
		if fnv1a(changed) == w.Hash() {
			t.Fatalf("unhashed source byte%d", i)
		}
	}
	bad := append([]byte(nil), form...)
	bad[at+34] = 2
	before := restored.Hash()
	if err := restored.UnmarshalBinary(bad); err == nil || restored.Hash() != before {
		t.Fatal("noncanonical Group owner presence published")
	}
}

func TestSourceBinding1111AdmissionLateFailureIsAtomic(t *testing.T) {
	w := sourceBindingWorld1111(t)
	e := w.entities[0]
	e.ID, e.SourceBinding.ArchiveIndex, e.SourceBinding.Identity, e.TokenSize = 8, 5, 0x10000005, 1
	late := e
	late.ID, late.SourceBinding.ArchiveIndex, late.SourceBinding.Identity, late.TokenSize = 9, 6, 0x10000006, 0
	before := w.Hash()
	if err := w.ImportOriginalLivingActors([]OriginalLivingActor{{e, true}, {late, true}}); err == nil || w.Hash() != before || len(w.entities) != 1 {
		t.Fatal("late source actor published")
	}
	late.TokenSize, late.SourceBinding.ArchiveIndex = 1, 5
	if err := w.ImportOriginalLivingActors([]OriginalLivingActor{{e, true}, {late, true}}); err == nil || w.Hash() != before {
		t.Fatal("duplicate source archive identity published")
	}
	if err := w.ImportOriginalLivingActors([]OriginalLivingActor{{e, true}}); err != nil || len(w.entities) != 2 || w.entities[1].ID != 8 {
		t.Fatalf("admission: %v", err)
	}
}
