package sim

import (
	"fmt"
	"reflect"
	"testing"
)

func TestEffectAttachmentRecordsForBothBackings(t *testing.T) {
	type step struct {
		rule      SpellRule
		kind      EffectKind
		magnitude int32
		duration  uint16
		mode      EffectMode
		caster    EntityID
		want      []attachedEffect
		source    []attachedEffect
	}
	bless := secondRow(SpellRule{ID: 20})
	curse := secondRow(SpellRule{ID: 28})
	slow := secondRow(SpellRule{ID: 29})
	poison := secondRow(SpellRule{ID: 6})
	haste := SpellRule{ID: 24}
	for _, tc := range []struct {
		name  string
		steps []step
	}{
		{"sequence", []step{
			{rule: bless, kind: EffectBless, magnitude: 5, duration: 100, mode: EffectDuration, caster: 1,
				want: []attachedEffect{{Target: 2, Caster: 1, HasCaster: true, Spell: 20, Kind: EffectBless, Mode: EffectDuration, Magnitude: 5, Remaining: 100}}},
			{rule: bless, kind: EffectBless, magnitude: 2, duration: 300, mode: EffectDuration, caster: 99,
				want: []attachedEffect{{Target: 2, Caster: 1, HasCaster: true, Spell: 20, Kind: EffectBless, Mode: EffectDuration, Magnitude: 5, Remaining: 300}}},
			{rule: bless, kind: EffectBless, magnitude: 8, duration: 200, mode: EffectDuration, caster: 99,
				want: []attachedEffect{{Target: 2, Caster: 1, HasCaster: true, Spell: 20, Kind: EffectBless, Mode: EffectDuration, Magnitude: 8, Remaining: 300}}},
			{rule: bless, kind: EffectBless, magnitude: -4, duration: 400, mode: EffectDuration, caster: 99,
				want: []attachedEffect{{Target: 2, Caster: 1, HasCaster: true, Spell: 20, Kind: EffectBless, Mode: EffectDuration, Magnitude: 8, Remaining: 400}}},
			{rule: curse, kind: EffectCurse, magnitude: 7, duration: 100, mode: EffectDuration, caster: 1},
			{rule: haste, kind: EffectSpeed, magnitude: -5, duration: 300, mode: EffectDuration, caster: 1,
				want: []attachedEffect{{Target: 2, Caster: 1, HasCaster: true, Spell: 24, Kind: EffectSpeed, Mode: EffectDuration, Magnitude: -5, Remaining: 300}}},
			{rule: haste, kind: EffectSpeed, magnitude: -2, duration: 100, mode: EffectDuration, caster: 1,
				want: []attachedEffect{{Target: 2, Caster: 1, HasCaster: true, Spell: 24, Kind: EffectSpeed, Mode: EffectDuration, Magnitude: -2, Remaining: 100}}},
		}},
		{"negative renewal", []step{
			{rule: slow, kind: EffectSpeed, magnitude: -8, duration: 300, mode: EffectDuration, caster: 1,
				want: []attachedEffect{{Target: 2, Caster: 1, HasCaster: true, Spell: 29, Kind: EffectSpeed, Mode: EffectDuration, Magnitude: -8, Remaining: 300}}},
			{rule: slow, kind: EffectSpeed, magnitude: -3, duration: 100, mode: EffectDuration, caster: 99,
				want: []attachedEffect{{Target: 2, Caster: 1, HasCaster: true, Spell: 29, Kind: EffectSpeed, Mode: EffectDuration, Magnitude: -3, Remaining: 300}}},
			{rule: slow, kind: EffectSpeed, magnitude: -6, duration: 400, mode: EffectDuration, caster: 99,
				want: []attachedEffect{{Target: 2, Caster: 1, HasCaster: true, Spell: 29, Kind: EffectSpeed, Mode: EffectDuration, Magnitude: -3, Remaining: 400}}},
		}},
		{"clamp", []step{
			{rule: haste, kind: EffectSpeed, magnitude: -20, duration: 100, mode: EffectDuration, caster: 1,
				want:   []attachedEffect{{Target: 2, Caster: 1, HasCaster: true, Spell: 24, Kind: EffectSpeed, Mode: EffectDuration, Magnitude: -9, Remaining: 100}},
				source: []attachedEffect{{Target: 2, Caster: 1, HasCaster: true, Spell: 24, Kind: EffectSpeed, Mode: EffectDuration, Magnitude: -20, Remaining: 100}}},
		}},
		{"absent caster", []step{
			{rule: haste, kind: EffectSpeed, magnitude: 3, duration: 100, mode: EffectDuration, caster: 99,
				want:   []attachedEffect{{Target: 2, Spell: 24, Kind: EffectSpeed, Mode: EffectDuration, Magnitude: 3, Remaining: 100}},
				source: []attachedEffect{{Target: 2, Caster: 99, Spell: 24, Kind: EffectSpeed, Mode: EffectDuration, Magnitude: 3, Remaining: 100}}},
		}},
		{"continuous refresh", []step{
			{rule: haste, kind: EffectSpeed, magnitude: 3, duration: 100, mode: EffectContinuous, caster: 1,
				want: []attachedEffect{{Target: 2, Caster: 1, HasCaster: true, Spell: 24, Kind: EffectSpeed, Mode: EffectContinuous, Magnitude: 3, Remaining: 100}}},
			{rule: haste, kind: EffectAbsorption, magnitude: 9, duration: 20, mode: EffectContinuous, caster: 99,
				want: []attachedEffect{{Target: 2, Caster: 1, HasCaster: true, Spell: 24, Kind: EffectSpeed, Mode: EffectContinuous, Magnitude: 3, Remaining: 20}}},
		}},
		{"poison renewal", []step{
			{rule: poison, kind: EffectHealth, magnitude: -6, duration: 100, mode: EffectContinuous, caster: 1,
				want: []attachedEffect{{Target: 2, Caster: 1, HasCaster: true, Spell: 6, Kind: EffectHealth, Mode: EffectContinuous, Magnitude: -6, Remaining: 100}}},
			{rule: poison, kind: EffectHealth, magnitude: -3, duration: 300, mode: EffectContinuous, caster: 99,
				want: []attachedEffect{{Target: 2, Caster: 1, HasCaster: true, Spell: 6, Kind: EffectHealth, Mode: EffectContinuous, Magnitude: -3, Remaining: 300}}},
		}},
	} {
		for _, source := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/source=%t", tc.name, source), func(t *testing.T) {
				target := spEnt(2, 3, 2)
				target.Speed = 10
				target.NativeBasis = NativeActorBasis{ScalarsPresent: true, ScalarKnown: 1 << ScalarU144}
				if source {
					target.ActorLoad = ActorLoad{Present: true, ContainerPresent: true, Source: SourceActor{Class: 1}}
					target.ActorLoad.Source.Stats[4] = 10
					target.Capacity = 100
				}
				rules := []SpellRule{bless, curse, slow, poison, haste}
				w := spWorld(t, 1, rules, spEnt(1, 2, 2), target)
				w.BindSourceDerive(func(s SourceActor, _ int32, _ Rules) (SourceActor, error) { return s, nil })
				for i, s := range tc.steps {
					if !w.attachEffect(2, s.caster, s.rule, s.kind, s.magnitude, s.duration, s.mode) {
						t.Fatalf("step %d refused", i)
					}
					want := s.want
					if source && s.source != nil {
						want = s.source
					}
					if len(w.attached) != len(want) || len(want) > 0 && !reflect.DeepEqual(w.attached, want) {
						t.Fatalf("step %d: records %+v, want %+v", i, w.attached, want)
					}
					t.Logf("step %d hash %x", i, w.Hash())
				}
			})
		}
	}
}
