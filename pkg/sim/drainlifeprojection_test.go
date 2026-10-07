package sim

import (
	"bytes"
	"testing"
)

func TestDrainLifeWeaponProjectionKeepsTransferFlagsAndState(t *testing.T) {
	rule := SpellRule{ID: 11, MaxRange: 7, DamageMin: 3, DamageMax: 5, TargetsUnit: true}
	caster := wpnCaster(1, 0, 0, 11, 30, 1, 0)
	w := spWorld(t, 1, []SpellRule{rule}, caster, spEnt(2, 1, 0))
	before, err := w.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	hash := w.Hash()
	for _, tc := range []struct {
		power int32
		min   int64
		max   int64
	}{{30, 6, 10}, {0, 3, 5}} {
		entity := caster
		entity.WeaponSpellLevel = tc.power
		got, ok := WeaponSpellCharacteristicsFor(Rules{}, entity, []SpellRule{rule})
		want := WeaponSpellCharacteristics{SpellID: 11, DamageMin: tc.min, DamageMax: tc.max, MaxRange: 7, HasDamage: true}
		if !ok || got != want {
			t.Errorf("synthetic Drain power %d = %+v/%v, want %+v", tc.power, got, ok, want)
		}
	}
	after, err := w.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(before, after) || hash != w.Hash() || w.Spells()[0].Damaging || w.Spells()[0].Restorative {
		t.Fatal("projection changed World bytes/hash or gameplay flags")
	}
	for _, control := range []Entity{{ID: 3, WeaponSpell: 11}, {ID: 4, MaxMana: 1, WeaponSpell: 65535}} {
		if _, ok := WeaponSpellCharacteristicsFor(Rules{}, control, []SpellRule{rule}); ok {
			t.Fatal("non-mage or unknown row obtained a projection")
		}
	}
	heal := rule
	heal.ID, heal.Restorative = 6, true
	caster.WeaponSpell = 6
	if _, ok := WeaponSpellCharacteristicsFor(Rules{}, caster, []SpellRule{heal}); ok {
		t.Fatal("Heal obtained a damaging weapon projection")
	}
}

func TestDrainLifeWeaponReleaseKeepsTransferResistanceAndRawRange(t *testing.T) {
	rule := SpellRule{ID: 11, School: 1, MaxRange: 7, DamageMin: 3, DamageMax: 3, TargetsUnit: true}
	caster := wpnCaster(1, 0, 0, 11, 30, 1, 0)
	caster.HP = 50
	caster.DamageBase, caster.AlwaysHits = 900, true
	victim := spEnt(2, 1, 0)
	victim.Protection = [5]int32{255, 255, 255, 255, 255}
	victim.Resistance = [5]uint8{255, 255, 255, 255, 255}
	w := spWorld(t, 1, []SpellRule{rule}, caster, victim)
	Step(w, []Command{cbOrder(1, 2)})
	if gotCaster, gotVictim := spAt(t, w, 1), spAt(t, w, 2); gotCaster.HP != 56 || gotVictim.HP != 94 {
		t.Fatalf("Drain weapon release caster/target HP = %d/%d, want 56/94", gotCaster.HP, gotVictim.HP)
	}
	if w.Spells()[0].Damaging {
		t.Fatal("Drain release changed the damaging flag")
	}
	for _, tc := range []struct {
		x      int32
		closed bool
	}{{7, true}, {8, false}} {
		victim.X = tc.x
		world := spWorld(t, 1, []SpellRule{rule}, caster, victim)
		if got := world.closedOn(indexOfEntity(world.entities, 1), indexOfEntity(world.entities, 2)); got != tc.closed {
			t.Fatalf("raw range7 at distance%d admission = %v, want %v", tc.x, got, tc.closed)
		}
		if !tc.closed {
			Step(world, []Command{cbOrder(1, 2)})
			if spAt(t, world, 2).HP != 100 || spAt(t, world, 1).HP != 50 {
				t.Fatal("out-of-range target transferred health")
			}
		}
	}
}
