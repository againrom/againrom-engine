package sim

import (
	"reflect"
	"testing"
)

func TestCorpseStateObservationIsDetachedAndReusesStorage(t *testing.T) {
	w := dcWorld(t, Entity{ID: 9, X: 3, Y: 3, Owner: 2, HP: -20, MaxHP: 100, Decay: 3},
		Entity{ID: 1, X: 1, Y: 1, Owner: 1, HP: 100, MaxHP: 100})
	before := w.Hash()
	storage := make([]CorpseState, 1, 3)
	storage[0] = CorpseState{ID: 99}
	got := w.AppendCorpseStates(storage)
	want := []CorpseState{{ID: 99}, {ID: 1, Owner: 1}, {ID: 9, Owner: 2, Stage: 3}}
	if !reflect.DeepEqual(got, want) || &got[0] != &storage[0] {
		t.Fatalf("observation=%+v; storage was not appended/reused", got)
	}
	got[1].Owner, got[2].Stage = 7, 0
	if w.Hash() != before {
		t.Fatal("mutating the detached observation changed world state")
	}
	if again := w.AppendCorpseStates(got[:0]); !reflect.DeepEqual(again, want[1:]) {
		t.Fatalf("caller edits reached world: %+v", again)
	}
	if allocs := testing.AllocsPerRun(100, func() { got = w.AppendCorpseStates(got[:0]) }); allocs != 0 {
		t.Fatalf("reused observation allocates %g times", allocs)
	}
}
