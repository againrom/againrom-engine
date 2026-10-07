package sim

import (
	"reflect"
	"testing"
)

func TestOriginalDying1144CleanupIsAtomicAndDoesNotReplayDeath(t *testing.T) {
	w := sourceBindingWorld1111(t)
	e := &w.entities[0]
	e.HP, e.Decay, e.Dwell, e.Defence = -8, DecayFallen, 7, 20
	before, rng := w.Hash(), w.rng.state
	good := OriginalDyingActor{ID: e.ID, HP: -8, Timer: 7}
	for _, bad := range []OriginalDyingActor{{ID: 99, HP: -8, Timer: 7}, {ID: e.ID, HP: 1, Timer: 7}, {ID: e.ID, HP: -8, Timer: -1}, {ID: e.ID, HP: -7, Timer: 7}} {
		if err := w.ImportOriginalDyingActors([]OriginalDyingActor{good, bad}); err == nil || w.Hash() != before {
			t.Fatalf("bad batch changed state: %v", err)
		}
	}
	if err := w.ImportOriginalDyingActors([]OriginalDyingActor{good}); err != nil {
		t.Fatal(err)
	}
	if e.HP != -8 || e.Decay != DecayFallen || e.Dwell != 7 || e.Defence != 20 || w.rng.state != rng || len(w.sacks) != 0 {
		t.Fatal("cleanup repeated the first-death transition")
	}
	form, err := w.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	var back World
	if err := back.UnmarshalBinary(form); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(w.entities, back.entities) || w.Hash() != back.Hash() {
		t.Fatal("native wire lost dying state")
	}
	for range 64 {
		Step(w, nil)
		Step(&back, nil)
		if w.Hash() != back.Hash() {
			t.Fatal("native continuation differs")
		}
	}
	plain := sourceBindingWorld1111(t)
	plain.entities[0].HP, plain.entities[0].Decay = 0, DecayFallen
	plain.entities[0].SourceBinding = SourceBinding{}
	if err := plain.ImportOriginalDyingActors([]OriginalDyingActor{{ID: 7, HP: 0}}); err == nil {
		t.Fatal("unbound ALM body admitted as a restored actor")
	}
}
