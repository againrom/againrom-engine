package sav

import (
	"bytes"
	"testing"
)

func TestCurrentEffectWidthAddressesRelocateAndRetire(t *testing.T) {
	var state DocumentStateData
	payload := `{"Version":1,"EffectWidths":[{"Object":1,"Actor":2,"Spell":20,"Wire":40000,"Lift":65536,"Anchor":[3,4]}]}`
	if err := SetNativeActions(&state, []byte(payload)); err != nil {
		t.Fatal(err)
	}
	if err := remapNativeActionObjects(&state, []uint16{0, 3, 1}); err != nil {
		t.Fatal(err)
	}
	requireActionPayload(t, state, `{"Version":1,"EffectWidths":[{"Object":3,"Actor":1,"Spell":20,"Wire":40000,"Lift":65536,"Anchor":[3,4]}]}`)
	for _, permutation := range [][]uint16{{0, 1, 2, 0}, {0, 0, 2, 1}} {
		copy := state
		if err := remapNativeActionObjects(&copy, permutation); err != nil {
			t.Fatal(err)
		}
		requireActionPayload(t, copy, `{"Version":1,"EffectWidths":[]}`)
	}
}

func TestCurrentEffectWidthAddressErrorsAreAtomic(t *testing.T) {
	for _, payload := range []string{
		`{"Version":1,"EffectWidths":[{"Object":0,"Actor":1}]}`,
		`{"Version":1,"EffectWidths":[{"Object":1,"Actor":0}]}`,
		`{"Version":1,"EffectWidths":[{"Object":1,"Actor":9}]}`,
		`{"Version":1,"EffectWidths":[{"Object":1,"Actor":1.5}]}`,
	} {
		var state DocumentStateData
		if err := SetNativeActions(&state, []byte(payload)); err != nil {
			t.Fatal(err)
		}
		before, _, _ := NativeActions(state)
		if err := remapNativeActionObjects(&state, []uint16{0, 1}); err == nil {
			t.Fatal("invalid width address accepted")
		}
		after, _, _ := NativeActions(state)
		if !bytes.Equal(before, after) {
			t.Fatal("invalid width address changed the leaf")
		}
	}
}
