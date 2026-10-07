package sim

import "testing"

func TestCurrentGhostPolicyKeepsNextControlSpirit(t *testing.T) {
	source := hlGhostWorld(t, 51, []SpellRule{{ID: 25, ManaCost: 2, TargetsUnit: true, MaxRange: 4}},
		hlGhostTemplate(), effectMage(1, 1, 1, 1<<25), spEnt(2, 2, 1))
	source.entities[1].HP, source.entities[1].Decay = -10, DecayBones
	policy := source.CurrentPolicy()
	var cold World
	if err := cold.UnmarshalBinary(mustMarshal(t, source)); err != nil {
		t.Fatal(err)
	}
	cold.ghost = GhostTemplate{}
	if err := cold.RestoreCurrentContinuation(&policy, nil, source.Actions(), nil); err != nil {
		t.Fatal(err)
	}
	if source.Hash() != cold.Hash() {
		t.Fatal("Ghost constructor policy changed current World")
	}
	for _, w := range []*World{source, &cold} {
		spRunCast(w, Cast(1, 2, 25))
	}
	if indexOfEntity(cold.entities, 3) < 0 || source.Hash() != cold.Hash() {
		t.Fatal("next Control Spirit used a different current template")
	}
}

func TestCurrentGhostPolicyRejectsMalformedTemplateAtomically(t *testing.T) {
	w := scrollFixtureWorld(t, 2, 1)
	before := w.Hash()
	for _, badDomain := range []bool{true, false} {
		policy := w.CurrentPolicy()
		if badDomain {
			policy.Ghost.Domain = 255
		} else {
			policy.Ghost.XPSlot = 255
		}
		if w.Hash() != before {
			t.Fatal("capture shares Ghost storage with World")
		}
		if err := w.RestoreCurrentContinuation(&policy, nil, w.Actions(), nil); err == nil || w.Hash() != before {
			t.Fatal("invalid Ghost policy accepted or changed World", err)
		}
	}
}
