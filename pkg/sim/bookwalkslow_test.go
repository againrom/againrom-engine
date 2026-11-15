package sim

import "testing"

// R2-ENGINE-1247: the engage walk never chooses Slow. The book holds Slow and Fire Arrow.
func TestAComputerMageNeverChoosesSlowInTheEngageWalk(t *testing.T) {
	slow := SpellRule{ID: 29, ManaCost: 3, School: 1, MaxRange: 7,
		DamageMin: 1, DamageMax: 2, TargetsUnit: true, Damaging: true}
	for _, d := range []ScriptDialect{ScriptROM2, ScriptROM1} {
		// The first game's book walk stops at row 28 as well
		arrows := 0
		for seed := uint64(1); seed <= 60; seed++ {
			mage := spMage(1, 2, 8, 30, 100, 100, 1<<1|1<<29)
			mage.Owner, mage.Speed, mage.AttackCharge, mage.AttackRelax = 2, 30, 8, 4
			near := spEnt(2, 3, 8)
			near.Owner = 1
			w := hlWorld(t, seed, acEnemies(t), []SpellRule{hlArrow(), slow}, mage, near)
			if d == ScriptROM2 {
				w.rom2, w.game = &rom2ScriptState{}, newGameProfile(ScriptROM2)
			}
			if !w.mageEngageCast(0, 2) {
				t.Fatalf("dialect %v seed %d: the engage cast nothing", d, seed)
			}
			k, ok := w.bookCastIndex(1)
			if !ok || w.bookCasts[k].Spell == 29 {
				t.Fatalf("dialect %v seed %d: the mage's cast is %+v", d, seed, w.bookCasts)
			}
			arrows++
		}
		if arrows != 60 {
			t.Fatalf("dialect %v: %d Fire Arrow casts of 60", d, arrows)
		}
	}
}
