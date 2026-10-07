package sim

import "testing"

func TestAGuardedUnitStillDyingIsNotYetALoss(t *testing.T) {
	s := mustScript(t,
		[]ScriptCheck{
			{Op: ScriptCheckVIP, Register: 0, Unit: 1, HasUnit: true},
			{Op: ScriptCheckVIP, Register: 1, Unit: 2, HasUnit: true},
		},
		nil,
		[]ScriptTrigger{{Pairs: [3]ScriptPair{pair(0, 1, ScriptCmpNE)},
			Instants: acts(), Once: false}})

	const dying = 120
	w := scriptWorld(t, s, []Entity{
		{ID: 1, HP: 5, MaxHP: 5, DyingTime: dying},
		{ID: 2, HP: 5, MaxHP: 5, DyingTime: dying},
	})

	Step(w, []Command{{Kind: KindKill, Entity: 1}})
	if i := indexOfEntity(w.entities, 1); i < 0 {
		t.Fatal("the guarded unit left the world on the tick it fell")
	} else if w.entities[i].Decay != DecayFallen || w.entities[i].Dwell == 0 {
		t.Fatalf("the fixture did not install the dying state: decay %d, dwell %d",
			w.entities[i].Decay, w.entities[i].Dwell)
	}

	// He is lying there with time left. Several script passes run over him.
	scriptTicks(w, 40, nil)
	if _, lost := w.ScriptCounters(); lost != 0 {
		t.Fatalf("the lose counter is %d while the guarded unit is still dying, want 0", lost)
	}
	if got := w.Outcome(); got != OutcomeUndecided {
		t.Fatalf("the outcome is %d while the guarded unit is still dying, want undecided", got)
	}

	// Spending the presentation time does not finish an exact-zero body. ROM1
	// keeps zero as a fixed point and its VIP check waits for the -10 teardown
	// boundary, leaving authored rescue objectives healable indefinitely.
	scriptTicks(w, dying, nil)
	if _, lost := w.ScriptCounters(); lost != 0 {
		t.Fatalf("the lose counter moved to %d for an HP-0 body after dwell", lost)
	}

	// Once damage has actually finished that body and the window is closed,
	// the loss side effect becomes eligible on its next script pass.
	i := indexOfEntity(w.entities, 1)
	w.entities[i].HP, w.entities[i].Decay, w.entities[i].Dwell = -10, DecayBones, 0
	scriptTicks(w, 32, nil)
	if _, lost := w.ScriptCounters(); lost == 0 {
		t.Error("the lose counter never moved after HP reached -10")
	}
}

// TestTheDyingWindowIsTheSamePredicateEverywhereItIsAsked states the property
// the change rests on: the routing planes and the cell records already carve
// out exactly this window, and the script now asks the same question. A build
// where the three disagreed would draw a body the routing plane says is not
// there.
func TestTheDyingWindowIsTheSamePredicateEverywhereItIsAsked(t *testing.T) {
	for _, c := range []struct {
		name string
		e    Entity
		want bool
	}{
		{"alive", Entity{ID: 1, HP: 5, MaxHP: 5}, false},
		{"fallen with time left", Entity{ID: 1, HP: 0, MaxHP: 5, Decay: DecayFallen, Dwell: 3}, true},
		{"fallen with the time spent", Entity{ID: 1, HP: 0, MaxHP: 5, Decay: DecayFallen}, false},
		{"past the first stage", Entity{ID: 1, HP: -50, MaxHP: 5, Decay: DecayBones, Dwell: 3}, false},
		{"felled below zero, time left", Entity{ID: 1, HP: -1, MaxHP: 5, Decay: DecayFallen, Dwell: 3}, true},
	} {
		if got := c.e.Dying(); got != c.want {
			t.Errorf("%s: Dying() = %v, want %v", c.name, got, c.want)
		}
		if got := scriptDead(c.e); c.e.Dying() && got {
			t.Errorf("%s: scriptDead reports %v for a unit that is still dying", c.name, got)
		}
	}
}
