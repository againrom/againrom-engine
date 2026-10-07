package sim

import (
	"fmt"
	"reflect"
	"testing"
)

func TestCurrentPoolsFollowRuntimeMutations(t *testing.T) {
	for _, original := range []bool{false, true} {
		for _, class := range []uint8{0, 1, 2} {
			t.Run(fmt.Sprintf("original%t/class%d", original, class), func(t *testing.T) {
				w := mustWorld(t, 7, Bounds{Width: 8, Height: 8}, []Entity{{ID: 1, X: 2, Y: 2, HP: 60, MaxHP: 100, Mana: 20, MaxMana: 80, Speed: 1, Capacity: 300, HealthRegenPeriod: 100, ManaRegenPeriod: 80}})
				w.BindSourceDerive(func(s SourceActor, _ int32, _ Rules) (SourceActor, error) { return s, nil })
				e := &w.entities[0]
				if original {
					e.CurrentProfileBasis = ProfileOriginalCurrent
				}
				var basis SourceActor
				if class != 0 {
					basis = SourceActor{Class: class, Stats: [14]uint16{9, 8, 7, 6, 1, 0, 0, 300, 60, 100, 100, 20, 80, 80}}
					basis.Base[23], basis.Modifier[63] = 0xa7, 0xb8
					e.ActorLoad = ActorLoad{Present: true, Source: basis}
				}
				check := func(hp, mana int32) {
					t.Helper()
					e := w.entities[0]
					want := basis
					if class != 0 {
						want.Stats[8], want.Stats[11] = uint16(hp), uint16(mana)
					}
					if e.HP != hp || e.Mana != mana || e.ActorLoad.Source != want {
						t.Fatalf("pools %d/%d want %d/%d; source=%+v want %+v", e.HP, e.Mana, hp, mana, e.ActorLoad.Source, want)
					}
				}
				Step(w, []Command{Damage(1, 7)})
				check(53, 20)
				w.applySpellDamage(0, SpellRule{DamageMin: 3, DamageMax: 3}, 0)
				check(50, 20)
				w.applySpellHealing(0, SpellRule{DamageMin: 6, DamageMax: 6}, 0)
				check(56, 20)
				w.regenerateActors(true)
				check(58, 21)
				debitBook(&w.entities[0], SpellRule{ManaCost: 4})
				check(58, 17)
				debitBook(&w.entities[0], SpellRule{ManaCost: -1, bookInstance: true})
				check(58, 18)
				if !w.attachEffect(1, ^EntityID(0), SpellRule{ID: 8}, EffectHealth, -7, 128, EffectContinuous) {
					t.Fatal("poison attachment")
				}
				check(51, 18)
				w.stepAttachedEffects()
				check(44, 18)
				if err := w.HeadlessHeal(1); err != nil {
					t.Fatal(err)
				}
				check(100, 18)
				hash := w.Hash()
				if w.UseCarriedPotion(1, -1) || w.HeadlessHeal(99) == nil || w.Hash() != hash {
					t.Fatal("refused potion/heal changed world")
				}
			})
		}
	}
}

func TestCurrentPoolWordsKeepSignedDebit(t *testing.T) {
	for _, tc := range []struct {
		mana, cost, want int32
	}{
		{0, -32768, -32768},
		{0, -1, 1},
		{32767, 32767, 0},
	} {
		e := Entity{Mana: tc.mana, ActorLoad: ActorLoad{Present: true, Source: SourceActor{Class: 2}}}
		debitBook(&e, SpellRule{ManaCost: tc.cost, bookInstance: true})
		if e.Mana != tc.want || e.ActorLoad.Source.Stats[11] != uint16(tc.want) {
			t.Fatal("debit changed signed arithmetic or left stale source word", tc, e)
		}
	}
}

func TestCurrentPoolImportRequiresExactWirePopulation(t *testing.T) {
	good := OriginalActorPools{ID: 1, HP: 7, MaxHP: 31, Mana: 5, MaxMana: 23}
	wire := OriginalActorPools{ID: 2, HP: -32768, MaxHP: 65535, Mana: 65535, MaxMana: 101}
	for _, tc := range []struct {
		name string
		last OriginalActorPools
		ids  []EntityID
	}{
		{"legacy unsigned word", wire, nil},
		{"unrelated current node", wire, []EntityID{1}},
		{"repeated current node", wire, []EntityID{2, 2}},
		{"missing current node", wire, []EntityID{2, 99}},
		{"signed health underflow", OriginalActorPools{ID: 2, HP: -32769}, []EntityID{2}},
		{"signed health overflow", OriginalActorPools{ID: 2, HP: 32768}, []EntityID{2}},
		{"negative maximum health", OriginalActorPools{ID: 2, MaxHP: -1}, []EntityID{2}},
		{"wide maximum health", OriginalActorPools{ID: 2, MaxHP: 65536}, []EntityID{2}},
		{"decoded negative mana", OriginalActorPools{ID: 2, Mana: -1}, []EntityID{2}},
		{"wide mana", OriginalActorPools{ID: 2, Mana: 65536}, []EntityID{2}},
		{"negative maximum mana", OriginalActorPools{ID: 2, MaxMana: -1}, []EntityID{2}},
		{"wide maximum mana", OriginalActorPools{ID: 2, MaxMana: 65536}, []EntityID{2}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := poolImportWorld(t)
			hash := w.Hash()
			if err := w.ImportOriginalActorPools([]OriginalActorPools{good, tc.last}, tc.ids...); err == nil || w.Hash() != hash {
				t.Fatal("invalid late pool changed candidate", err)
			}
		})
	}
	w := poolImportWorld(t)
	before := w.Entities()
	if err := w.ImportOriginalActorPools([]OriginalActorPools{good, wire}, 2); err != nil {
		t.Fatal(err)
	}
	before[0].HP, before[0].MaxHP, before[0].Mana, before[0].MaxMana = 7, 31, 5, 23
	before[1].HP, before[1].MaxHP, before[1].Mana, before[1].MaxMana = -32768, 65535, 65535, 101
	if !reflect.DeepEqual(before, w.Entities()) {
		t.Fatal("wire admission changed another field or applied a premature signed cast")
	}
}
