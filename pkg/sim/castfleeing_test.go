package sim

import (
	"fmt"
	"testing"
)

const fleeRange = 2

// fleeWorld holds the player's mage at (1,8), spell range and weapon reach 4,
// and a hostile fast walker at (4,8). weapon gives the mage a spell weapon;
// otherwise it casts from its book.
func fleeWorld(t *testing.T, second, weapon bool) *World {
	t.Helper()
	caster := spMage(1, 1, 8, 30, 100, 100, 1<<1)
	caster.Owner, caster.Speed, caster.Reach = SelfSlot, 30, fleeRange
	caster.AttackCharge, caster.AttackRelax = 40, 4
	if weapon {
		caster.WeaponSpell = 1
	}
	enemy := spEnt(2, 1+fleeRange, 8)
	enemy.Owner, enemy.HP, enemy.MaxHP, enemy.Speed = 2, 5000, 5000, 255
	rule := hlArrow()
	rule.MaxRange = fleeRange
	w := hlWorld(t, 7, acEnemies(t), []SpellRule{rule}, caster, enemy)
	holdGroups(w)
	if second {
		w.rom2, w.game = &rom2ScriptState{}, newGameProfile(ScriptROM2)
	}
	return w
}

// TestACastLandsOnATargetThatWalksOutOfRangeDuringTheWindUp: the target gets a
// move order away from the caster after the cast was decided in range, walks
// beyond the range, and is still hit at the release frame. Both games, a book
// cast and a spell weapon's attack.
func TestACastLandsOnATargetThatWalksOutOfRangeDuringTheWindUp(t *testing.T) {
	for _, second := range []bool{false, true} {
		for _, weapon := range []bool{false, true} {
			t.Run(fmt.Sprintf("second=%v weapon=%v", second, weapon), func(t *testing.T) {
				w := fleeWorld(t, second, weapon)
				order := Cast(1, 2, 1)
				if weapon {
					order = Attack(1, 2)
				}
				if c, e := spAt(t, w, 1), spAt(t, w, 2); edgeDistance(&c, &e) > fleeRange {
					t.Fatalf("second=%v weapon=%v: out of range at the order", second, weapon)
				}
				Step(w, []Command{order})
				winding := func() bool {
					if weapon {
						return spAt(t, w, 1).AttackPhase == AttackCasting
					}
					k, ok := w.bookCastIndex(1)
					return ok && w.bookCasts[k].Phase == bookCharging
				}
				for n := 0; !winding() && n < 40; n++ {
					Step(w, nil)
				}
				if !winding() {
					t.Fatalf("second=%v weapon=%v: the wind-up never began", second, weapon)
				}
				Step(w, []Command{MoveTo(2, CellPoint{X: 15, Y: 8})})
				released := false
				var edge uint8
				for range 200 {
					for _, ev := range StepObserved(w, nil) {
						if ev.Caster == 1 && ev.Target == 2 && !released {
							released = true
							c, e := spAt(t, w, 1), spAt(t, w, 2)
							edge = edgeDistance(&c, &e)
						}
					}
				}
				if !released || edge <= fleeRange {
					t.Fatalf("second=%v weapon=%v: release %v at edge distance %d, want a release beyond range %d",
						second, weapon, released, edge, fleeRange)
				}
				if !weapon && spAt(t, w, 1).Mana != 97 {
					t.Fatalf("second=%v weapon=%v: mana %d, want 97", second, weapon, spAt(t, w, 1).Mana)
				}
			})
		}
	}
}
