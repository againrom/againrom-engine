package sim

import (
	"crypto/sha256"
	"encoding/hex"
	"testing"
)

// A World whose token rows all fit a byte writes the bytes and hash it wrote
// before a row could be wider: the digests are pinned from that build.
func TestTokenRowsWithinAByteKeepTheirForm(t *testing.T) {
	w := sourceBindingWorld(t)
	w.entities[0].SourceBinding.TokenRow = 255
	form, err := w.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(form)
	const wantForm, wantHash = "4d61a847fbfdcf645e0a6e6d29387ea397c960d2d20f5fcd12c288ef7805484c", uint64(0xd468916c09322196)
	if got := hex.EncodeToString(sum[:]); got != wantForm || w.Hash() != wantHash || form[0] != formatVersion {
		t.Fatalf("form %d %s hash %#x, want %d %s %#x", form[0], got, w.Hash(), formatVersion, wantForm, wantHash)
	}
}

// A token row above a byte keeps its low byte in the binding and its high
// byte in the token row form; the World decodes back to the same row, bytes
// and hash, and a malformed form is refused.
func TestTokenRowAboveAByteRoundTripsThroughItsForm(t *testing.T) {
	w := sourceBindingWorld(t)
	w.entities[0].SourceBinding.TokenRow = 264
	form, err := w.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	if form[0] != tokenRowFormVersion || form[34+3*64+457+11] != 8 {
		t.Fatalf("form %d binding row byte %d, want %d and the low byte 8", form[0], form[34+3*64+457+11], tokenRowFormVersion)
	}
	if err := CheckSaveForm(form); err != nil {
		t.Fatal(err)
	}
	var restored World
	if err := restored.UnmarshalBinary(form); err != nil {
		t.Fatal(err)
	}
	if got := restored.entities[0].SourceBinding.TokenRow; got != 264 || restored.Hash() != w.Hash() {
		t.Fatalf("restored row %d hash %#x, want 264 %#x", got, restored.Hash(), w.Hash())
	}
	again, err := restored.MarshalBinary()
	if err != nil || string(again) != string(form) {
		t.Fatalf("the restored World writes another form: %v", err)
	}
	// The section is one entity index and one high byte per wide row.
	section := len(form) - 9 - 5
	for name, edit := range map[string]func([]byte){
		"high byte 0":   func(b []byte) { b[section+4] = 0 },
		"no such index": func(b []byte) { b[section] = 9 },
	} {
		bad := []byte(string(form))
		edit(bad)
		var refused World
		if err := refused.UnmarshalBinary(bad); err == nil {
			t.Fatalf("%s: the form decodes", name)
		}
	}
}
