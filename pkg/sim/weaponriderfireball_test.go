package sim

import "testing"

// riderPastCutoffWorld is a fighter whose weapon carries rule, striking a
// victim the blow takes below -10, with a bystander beside the victim.
func riderPastCutoffWorld(t *testing.T, second bool, rule SpellRule) *World {
	t.Helper()
	caster := wpnCaster(1, 0, 0, uint16(rule.ID), 40, 1, 0)
	caster.MaxMana, caster.Mana = 0, 0
	caster.DamageBase, caster.AlwaysHits = 900, true
	victim := spEnt(2, 1, 0)
	victim.HP = 5
	bystander := spEnt(3, 1, 1)
	var w *World
	if second {
		w = spWorld(t, 1, []SpellRule{rule}, caster, victim, bystander)
		var err error
		w, err = NewControlledScriptWorld(w, secondGameScript(t, nil, nil, nil))
		if err != nil {
			t.Fatal(err)
		}
	} else {
		w = spWorld(t, 1, []SpellRule{rule}, caster, victim, bystander)
	}
	Step(w, []Command{cbOrder(1, 2)})
	for k := 0; k < 100 && spAt(t, w, 2).HP == 5; k++ {
		Step(w, nil)
	}
	if hp := spAt(t, w, 2).HP; hp > -10 {
		t.Fatalf("fixture: the victim is at %d, want the blow to take it to -10 or below", hp)
	}
	return w
}

func TestSecondGameFireBallWeaponFiresAfterABlowPastMinusTen(t *testing.T) {
	fire := SpellRule{ID: 2, Area: true, Radius: 1, School: 1, MaxRange: 5,
		DamageMin: 10, DamageMax: 10, Damaging: true}
	w := riderPastCutoffWorld(t, true, fire)
	if hp := spAt(t, w, 3).HP; hp >= 100 {
		t.Errorf("the bystander is at %d, want the ball to have hit it", hp)
	}
	other := fire
	other.ID = 3
	w = riderPastCutoffWorld(t, true, other)
	if hp := spAt(t, w, 3).HP; hp != 100 {
		t.Errorf("a weapon area spell other than Fire Ball hit the bystander: %d", hp)
	}
}

func TestFirstGameFireBallWeaponFiresAfterABlowPastMinusTen(t *testing.T) {
	fire := SpellRule{ID: 2, Area: true, Radius: 1, School: 1, MaxRange: 5,
		DamageMin: 10, DamageMax: 10, Damaging: true}
	w := riderPastCutoffWorld(t, false, fire)
	if hp := spAt(t, w, 3).HP; hp >= 100 {
		t.Errorf("the bystander is at %d, want the first game's ball to have hit it", hp)
	}
	other := fire
	other.ID = 3
	w = riderPastCutoffWorld(t, false, other)
	if hp := spAt(t, w, 3).HP; hp != 100 {
		t.Errorf("a first-game weapon area spell other than Fire Ball hit the bystander: %d", hp)
	}

	caster := wpnCaster(1, 0, 0, uint16(fire.ID), 40, 1, 0)
	caster.MaxMana, caster.Mana = 0, 0
	caster.XPSlot, caster.Owner, caster.GainsXP, caster.Mind, caster.TypeID = 4, 2, true, 60, HumanTypeID
	caster.Skill[4], caster.SkillXP[4] = 10, skillXPFor(10)
	caster.DamageBase, caster.AlwaysHits = 900, true
	victim := spEnt(2, 1, 0)
	victim.Owner, victim.HP = 3, 5
	xpWorld := spWorld(t, 1, []SpellRule{fire}, caster, victim)
	Step(xpWorld, []Command{cbOrder(1, 2)})
	if hp := spAt(t, xpWorld, 2).HP; hp > -10 {
		t.Fatalf("fixture: the victim is at %d, want -10 or below", hp)
	}
	if got := spAt(t, xpWorld, 1).SkillXP[4]; got != skillXPFor(10) {
		t.Errorf("blow past -10 paid skill XP %d, want %d (none)", got, skillXPFor(10))
	}
}

func TestFireBallRiderWorldKeepsItsHashThroughARoundTrip(t *testing.T) {
	fire := SpellRule{ID: 2, Area: true, Radius: 1, School: 1, MaxRange: 5,
		DamageMin: 10, DamageMax: 10, Damaging: true}
	for _, second := range []bool{false, true} {
		w := riderPastCutoffWorld(t, second, fire)
		data, err := w.MarshalBinary()
		if err != nil {
			t.Fatalf("second=%v MarshalBinary: %v", second, err)
		}
		var back World
		if err := back.UnmarshalBinary(data); err != nil {
			t.Fatalf("second=%v UnmarshalBinary: %v", second, err)
		}
		if back.Hash() != w.Hash() {
			t.Errorf("second=%v round trip hashes %#016x, the original %#016x", second, back.Hash(), w.Hash())
		}
	}
}
