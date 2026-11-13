package sim

import "testing"

// healTicks builds a world of four wounded actors whose runtime ids are 5
// to 8, steps it through four mana passes and returns, per actor, the
// sub-ticks on which its health moved.
func healTicks(t *testing.T, second bool) map[EntityID][]uint64 {
	t.Helper()
	var ents []Entity
	for k := int32(0); k < 4; k++ {
		e := Entity{ID: EntityID(k + 1), X: 1 + 2*k, Y: 1, HP: 10, MaxHP: 50, HealthRegenPeriod: 25}
		e.SourceBinding.RuntimeID = uint32(5 + k)
		ents = append(ents, e)
	}
	var s *Script
	if second {
		s = secondGameScript(t, nil, nil, nil)
	}
	w, err := NewStockedSpelledWorld(1, Bounds{Width: 16, Height: 16}, ModeCanonical, Terrain{}, ents, s, Relations{}, nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	moved := map[EntityID][]uint64{}
	for w.Tick() < 4*manaRegenCycle {
		before := map[EntityID]int32{}
		for _, e := range w.Entities() {
			before[e.ID] = e.HP
		}
		tick := w.Tick()
		Step(w, nil)
		for _, e := range w.Entities() {
			if e.HP != before[e.ID] {
				moved[e.ID] = append(moved[e.ID], tick)
			}
		}
	}
	return moved
}

// R2-ENGINE-030: the second game heals an actor on every fourth pass, on the
// pass whose low two bits equal its +4's.
func TestSecondGameHealsEachActorOnItsOwnPhase(t *testing.T) {
	t.Parallel()
	moved := healTicks(t, true)
	for id := EntityID(1); id <= 4; id++ {
		phase := (uint64(id) + 4) % 4 // runtime id id+4
		want := phase*manaRegenCycle + regenPhase
		if got := moved[id]; len(got) != 1 || got[0] != want {
			t.Errorf("actor %d (runtime id %d) healed on %v, want [%d]", id, id+4, got, want)
		}
	}
}

// HERO-REGEN-021: the first game heals every actor on the same pass.
func TestFirstGameHealsEveryActorOnOnePass(t *testing.T) {
	t.Parallel()
	moved := healTicks(t, false)
	for id := EntityID(1); id <= 4; id++ {
		if got := moved[id]; len(got) != 1 || got[0] != regenPhase {
			t.Errorf("actor %d healed on %v, want [%d]", id, got, regenPhase)
		}
	}
}
