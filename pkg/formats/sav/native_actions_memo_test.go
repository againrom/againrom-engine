package sav

import (
	"reflect"
	"testing"
)

// A remap answered from the memo writes the same state the computation
// writes, for each mode and permutation, and an unseen input is computed.
func TestNativeRemapMemoAnswersAsTheComputation(t *testing.T) {
	payload := `{"Version":1,"Groups":[{"Object":1,"Inline":3,"ID":99,"Authored":true}],"Ownership":[{"Object":2,"Owner":1}],"SpellCasters":[{"Object":2,"Caster":3},{"Object":4,"Caster":3}]}`
	for _, city := range []bool{false, true} {
		for _, permutation := range [][]uint16{{0, 3, 1, 2, 4}, {0, 1, 2, 3, 4}, {0, 3, 0, 2, 1}} {
			state := nativeRemapState(t, payload)
			nativeRemaps.mu.Lock()
			nativeRemaps.entries = nil
			nativeRemaps.mu.Unlock()
			first := nativeRemapDifferential(t, state, permutation, city, !city && permutation[2] == 0)
			if _, hit := nativeRemaps.lookup(mustNativeActions(t, state), permutation, city); hit == (!city && permutation[2] == 0) {
				t.Fatalf("memo holds a result after error=%t", !city && permutation[2] == 0)
			}
			second := nativeRemapDifferential(t, state, permutation, city, !city && permutation[2] == 0)
			if !reflect.DeepEqual(first, second) {
				t.Fatalf("memo answer differs from computation (city %t, %v)", city, permutation)
			}
			if _, hit := nativeRemaps.lookup(mustNativeActions(t, state), permutation, !city); hit {
				t.Fatal("memo answered the other mode")
			}
		}
	}
}

func mustNativeActions(t *testing.T, state DocumentStateData) []byte {
	t.Helper()
	b, ok, err := NativeActions(state)
	if err != nil || !ok {
		t.Fatal("no native actions", err)
	}
	return b
}
