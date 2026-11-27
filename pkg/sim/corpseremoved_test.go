package sim

import "testing"

func TestCorpseRemovedReadsTheSameThroughEveryReader(t *testing.T) {
	w, err := NewWorld(1, Bounds{Width: 8, Height: 8}, ModeCanonical, nil, []Entity{
		{ID: 1, X: 2, Y: 2, HP: 10, MaxHP: 10},
	})
	if err != nil {
		t.Fatal(err)
	}
	Step(w, []Command{Damage(1, 20)})
	removedAt, last := -1, DecayNone
	for tick := 0; tick < 6000 && len(w.Entities()) == 1; tick++ {
		e := w.Entities()[0]
		removed := terminalRegistryActor(&e)
		if removed != (!e.Alive() && (e.CorpseDecayed || e.Decay >= DecayBones)) {
			t.Fatalf("tick %d: registry disagrees with the rule: %+v", tick, e)
		}
		if removed && removedAt < 0 {
			removedAt = tick
		}
		if removedAt >= 0 && !removed {
			t.Fatalf("tick %d: removed corpse read present again", tick)
		}
		if e.Decay != last {
			last = e.Decay
			data, err := w.MarshalBinary()
			if err != nil {
				t.Fatal(err)
			}
			var cold World
			if err := cold.UnmarshalBinary(data); err != nil {
				t.Fatalf("tick %d: decode: %v", tick, err)
			}
			ce := cold.Entities()[0]
			if got := terminalRegistryActor(&ce); got != removed || ce.CorpseRemoved() != e.CorpseRemoved() {
				t.Fatalf("tick %d stage %d: decoded world reads removed=%v, live %v", tick, e.Decay, got, removed)
			}
			fresh, err := NewWorld(1, Bounds{Width: 8, Height: 8}, ModeCanonical, nil, []Entity{e})
			if err != nil {
				t.Fatalf("tick %d: new world: %v", tick, err)
			}
			fe := fresh.Entities()[0]
			if got := terminalRegistryActor(&fe); got != removed {
				t.Fatalf("tick %d stage %d: new world reads removed=%v, live %v", tick, e.Decay, got, removed)
			}
		}
		Step(w, nil)
	}
	if removedAt != 0 || last < DecayBones {
		t.Fatalf("removed at tick %d, last stage %d: want removed from the first tick after the fall and a walk to the bones", removedAt, last)
	}
}
