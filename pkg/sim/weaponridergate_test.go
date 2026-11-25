package sim

import "testing"

// A fighter's weapon Spell other than Fire Ball fires only when the blow's
// damage and the target's health after it are above 0: a blow that leaves
// the target at 0 casts nothing, one that leaves it at 1 casts the Spell
// (R2-ENGINE-1829).
func TestSecondGameRiderNeedsTheTargetAboveZeroAfterTheBlow(t *testing.T) {
	for _, c := range []struct {
		hp   int32
		cast bool
	}{
		{5, false}, {6, true},
	} {
		caster := wpnCaster(1, 0, 0, 1, 30, 1, 0)
		caster.MaxMana, caster.Mana = 0, 0
		caster.DamageBase, caster.DamageSpread, caster.AlwaysHits = 5, 0, true
		victim := spEnt(2, 1, 0)
		victim.HP = c.hp
		w := spSecondWorld(t, 1, []SpellRule{wpnRule(6, 6, 5)}, caster, victim)
		Step(w, []Command{cbOrder(1, 2)})
		for k := 0; k < 100 && spAt(t, w, 2).HP == c.hp; k++ {
			Step(w, nil)
		}
		if hp := spAt(t, w, 2).HP; hp == c.hp || (hp != c.hp-5) != c.cast {
			t.Errorf("victim at %d before a blow of 5: health %d, want the rider cast %v", c.hp, hp, c.cast)
		}
	}
}
