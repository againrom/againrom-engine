package sim

import (
	"fmt"
	"testing"
)

func TestPoison1164SignedResistanceAndDirectHealth(t *testing.T) {
	for _, tc := range []struct{ magnitude, protection, damage int32 }{
		{-7, 0, 7}, {-7, 50, 4}, {-7, 100, 0}, {-7, 150, -3},
		{-7, -50, 11}, {7, 0, -7}, {7, 50, -3}, {-1, 99, 0},
	} {
		for _, source := range []bool{false, true} {
			t.Run(fmt.Sprintf("%d/%d/source%t", tc.magnitude, tc.protection, source), func(t *testing.T) {
				var w *World
				if source {
					w = sourceMutationWorld(t, PlainItem(0xe01))
					w.sourceDerive = func(SourceActor, int32, Rules) (SourceActor, error) {
						t.Fatal("Poison invoked the potion/derive path")
						return SourceActor{}, nil
					}
				} else {
					w = mustWorld(t, 1164, Bounds{32, 32}, []Entity{spEnt(1, 5, 5)})
				}
				e := &w.entities[0]
				e.HP, e.MaxHP, e.Protection[1] = 100, 100, tc.protection
				if !w.attachEffect(e.ID, ^EntityID(0), SpellRule{ID: 8}, EffectHealth, tc.magnitude, 128, EffectContinuous) {
					t.Fatal("Poison attachment refused")
				}
				if e = &w.entities[0]; e.HP != 100-tc.damage || w.attached[0].Magnitude != tc.magnitude || w.attached[0].Remaining != 128 {
					t.Fatal("computed direct HP/magnitude/counter", e.HP, w.attached)
				}
				w.stepAttachedEffects()
				if w.entities[0].HP != 100-2*tc.damage || w.attached[0].Remaining != 127 {
					t.Fatal("old128 actor phase", w.entities[0].HP, w.attached)
				}
			})
		}
	}
}

func TestPoison1164RefreshPreservesSourceMagnitudeAndPhase(t *testing.T) {
	w := mustWorld(t, 1164, Bounds{32, 32}, []Entity{spEnt(1, 5, 5), spEnt(2, 10, 10), spEnt(3, 15, 15)})
	w.entities[0].HP, w.entities[0].MaxHP = 100, 100
	rule := SpellRule{ID: 8}
	if !w.attachEffect(1, 2, rule, EffectHealth, -4, 128, EffectContinuous) {
		t.Fatal("first attach")
	}
	if !w.attachEffect(1, 3, rule, EffectHealth, -20, 9, EffectContinuous) {
		t.Fatal("refresh")
	}
	if e := w.attached[0]; w.entities[0].HP != 96 || e.Caster != 2 || !e.HasCaster || e.Magnitude != -4 || e.Remaining != 9 {
		t.Fatal("refresh overwrote payload/source or applied", e)
	}
	w.stepAttachedEffects()
	if w.entities[0].HP != 96 || w.attached[0].Remaining != 8 {
		t.Fatal("old9 must not pulse")
	}
	var cold World
	if err := cold.UnmarshalBinary(mustMarshal(t, w)); err != nil {
		t.Fatal(err)
	}
	for _, current := range []*World{w, &cold} {
		current.stepAttachedEffects()
		if current.entities[0].HP != 92 || current.attached[0].Remaining != 7 {
			t.Fatal("old8 must pulse after SAVE/LOAD")
		}
		current.attached[0].Remaining = 1
		current.stepAttachedEffects()
		if current.entities[0].HP != 92 || len(current.attached) != 0 {
			t.Fatal("old1 must remove without zero pulse")
		}
	}
	if w.Hash() != cold.Hash() {
		t.Fatal("native continuation differs")
	}
}

func TestPoison1164FrozenCounterAndNonzeroSourceGate(t *testing.T) {
	for _, hp := range []int32{-1, 0, 1} {
		for _, protection := range []int32{0, 100} {
			w := mustWorld(t, 1164, Bounds{32, 32}, []Entity{spEnt(1, 5, 5), spEnt(2, 10, 10)})
			w.entities[0].HP, w.entities[0].MaxHP, w.entities[0].Protection[1] = 100, 100, protection
			w.entities[1].HP = hp
			if hp <= 0 {
				w.clearFelled(1)
			}
			w.attachEffect(1, 2, SpellRule{ID: 8}, EffectHealth, -4, 9601, EffectContinuous)
			wantSource := hp >= 0 || protection == 100
			if w.attached[0].HasCaster != wantSource {
				t.Fatal("source negative/zero or computed-zero gate", hp, protection, w.attached)
			}
			before := w.Hash()
			w.stepAttachedEffects()
			if w.Hash() != before {
				t.Fatal("counter above9600 must freeze")
			}
		}
	}
}

func TestPoison1164AwardsComputedDamage(t *testing.T) {
	for _, p := range []int32{50, 100} {
		caster, target := spEnt(1, 5, 5), spEnt(2, 10, 10)
		caster.Owner, caster.GainsXP, caster.TypeID, caster.XPSlot = 2, true, HumanTypeID, 3
		target.Owner, target.XPValue, target.Protection[1] = 3, 1000, p
		w := mustWorld(t, 1164, Bounds{32, 32}, []Entity{caster, target})
		w.spells = []SpellRule{{ID: 8}}
		w.attachEffect(2, 1, w.spells[0], EffectHealth, -7, 128, EffectContinuous)
		want := int32(5) // damage4: ((1000*4/(2*100)+1)*30)/120
		if p == 100 {
			want = 0
		}
		if w.entities[0].SkillXP[3] != want {
			t.Fatal("award used nominal magnitude or awarded zero result", p, w.entities[0].SkillXP)
		}
	}
}

func TestPoison1164CurrentRetainedTerminalActionSkipsActorPhase(t *testing.T) {
	for _, tc := range []struct {
		current bool
		action  uint32
		skip    bool
	}{{true, 0x10, true}, {false, 0x10, false}, {true, 0x0f, false}} {
		w := mustWorld(t, 1164, Bounds{32, 32}, []Entity{spEnt(1, 5, 5)})
		w.entities[0].HP = 100
		w.attachEffect(1, ^EntityID(0), SpellRule{ID: 8}, EffectHealth, -4, 8, EffectContinuous)
		// A controlled phase-gate input; no claim of a complete saved mover.
		w.savedMotion = &savedActorMotionState{Motions: []SavedActorMotion{{Entity: 1, Current: tc.current, ActorAction: tc.action}}}
		w.stepAttachedEffects()
		wantHP, wantRemaining := int32(92), uint16(7)
		if tc.skip {
			wantHP, wantRemaining = 96, 8
		}
		if w.entities[0].HP != wantHP || w.attached[0].Remaining != wantRemaining {
			t.Fatal("current retained terminal-action gate", tc, w.entities[0].HP, w.attached[0].Remaining)
		}
	}
}
