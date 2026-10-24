package sav

import "testing"

func TestNativeWorldEffectTokenBindingsRelocateAndRetainAbsentValues(t *testing.T) {
	state := DocumentStateData{}
	if err := SetNativeActions(&state, []byte(`{"Version":1,"WorldEffectTokens":[{"Object":1,"Value":{"T18":17}},{"Object":2,"Value":{"T18":23}}]}`)); err != nil {
		t.Fatal(err)
	}
	if err := remapNativeActionObjects(&state, []uint16{0, 2, 1}); err != nil {
		t.Fatal(err)
	}
	requireActionPayload(t, state, `{"Version":1,"WorldEffectTokens":[{"Object":2,"Value":{"T18":17}},{"Object":1,"Value":{"T18":23}}]}`)
	if err := remapNativeActionObjects(&state, []uint16{0, 1, 0}); err != nil {
		t.Fatal(err)
	}
	requireActionPayload(t, state, `{"Version":1,"WorldEffectTokens":[{"Object":0,"Value":{"T18":17}},{"Object":1,"Value":{"T18":23}}]}`)
	if err := remapNativeActionObjects(&state, []uint16{0, 1}); err != nil {
		t.Fatal(err)
	}
}
