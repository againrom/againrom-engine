package sim

import "testing"

// turningPaymentWorld is castFacingWorld with the mage a player's and its
// Fire Arrow a projectile, a row that pays before its release.
func turningPaymentWorld(t *testing.T, second bool) *World {
	t.Helper()
	w := castFacingWorld(t, second)
	w.entities[0].Owner, w.entities[1].Owner = SelfSlot, 2
	for k := range w.spells {
		if w.spells[k].ID == 1 {
			w.spells[k].Delivery, w.spells[k].EffectSpeed = 2, 200
		}
	}
	return w
}

// scriptedTurningPaymentWorld is turningPaymentWorld's second game built with
// a second-game script, so its byte form carries the game.
func scriptedTurningPaymentWorld(t *testing.T) *World {
	t.Helper()
	mage := spMage(1, 8, 8, 30, 100, 100, 1<<1)
	mage.Owner, mage.Speed, mage.AttackCharge, mage.AttackRelax = SelfSlot, 30, 8, 4
	mage.RotationSpeed, mage.Facing, mage.DesiredFacing = 4, 0, 0
	victim := spEnt(2, 8, 10)
	victim.Owner, victim.HP, victim.MaxHP = 2, 500, 500
	rel := engRel(t, [3]uint32{1, 2, relationHostile}, [3]uint32{2, 1, relationHostile})
	arrow := hlArrow()
	arrow.Delivery, arrow.EffectSpeed = 2, 200
	s, err := newScript(ScriptROM2, nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	w, err := NewStockedSpelledWorld(7, Bounds{Width: 16, Height: 16}, ModeCanonical, Terrain{}, []Entity{mage, victim}, s, rel, nil, nil, []SpellRule{arrow})
	if err != nil {
		t.Fatal(err)
	}
	if !playsSecondGame(w) {
		t.Fatal("not a second-game World")
	}
	return w
}

// manaDebits steps w until its caster releases the Fire Arrow and returns
// each fall of the caster's mana and whether the caster faced its target on
// the tick of that fall. first is the step that carries the order, if any.
func manaDebits(t *testing.T, w *World, first []Command) (falls []int32, facing []bool) {
	t.Helper()
	cmds := first
	for range 120 {
		before := w.entities[0].Mana
		events := StepObserved(w, cmds)
		cmds = nil
		e := w.entities[0]
		if e.Mana < before {
			falls = append(falls, before-e.Mana)
			facing = append(facing, !e.Turning() && e.Facing == w.headingBetween(e, w.entities[1]))
		}
		for _, ev := range events {
			if ev.Caster == 1 && ev.Spell == 1 {
				return falls, facing
			}
		}
	}
	t.Fatal("no Fire Arrow released in 120 ticks")
	return nil, nil
}

// A second-game cast at a unit behind its caster pays its mana once, when
// the caster faces the target and the wind-up starts: the cast routine that
// subtracts the cost (R2-ENGINE-544) runs after the cast initializer
// (R2-ENGINE-543), which the order reaches only once the facing equals the
// heading (R2-ENGINE-926). The first game still pays at the order, while the
// caster turns. Fails while the second game pays at the order.
func TestASecondGameCastPaysWhenItsCasterFacesTheTarget(t *testing.T) {
	for _, second := range []bool{false, true} {
		w := turningPaymentWorld(t, second)
		cost := hlArrow().ManaCost
		falls, facing := manaDebits(t, w, []Command{Cast(1, 2, 1)})
		if len(falls) != 1 || falls[0] != cost {
			t.Fatalf("second %v: mana fell by %v, want once by %d", second, falls, cost)
		}
		if facing[0] != second {
			t.Fatalf("second %v: the caster faced its target when it paid: %v, want %v", second, facing[0], second)
		}
	}
}

// A second-game SAVE taken while the caster still turns loads and the cast
// pays once, when the turn ends; a SAVE an older engine took after paying at
// the order loads and pays nothing more.
func TestASecondGameCastSavedDuringItsTurnPaysOnce(t *testing.T) {
	for _, paidAtOrder := range []bool{false, true} {
		w := scriptedTurningPaymentWorld(t)
		cost := hlArrow().ManaCost
		StepObserved(w, []Command{Cast(1, 2, 1)})
		k, ok := w.bookCastIndex(1)
		if !ok || w.bookCasts[k].Phase != bookCharging || !w.entities[0].Turning() {
			t.Fatalf("fixture: the order holds %+v, turning %v", w.bookCasts, w.entities[0].Turning())
		}
		full := w.entities[0].Mana
		if paidAtOrder {
			rule, _ := w.bookSpell(w.entities[0], 1)
			debitBook(&w.entities[0], rule)
			w.bookCasts[k].Paid = true
		}
		back := worldRoundTripForTest(t, w)
		if !playsSecondGame(back) {
			t.Fatal("fixture: the loaded World does not run the second game")
		}
		falls, _ := manaDebits(t, back, nil)
		want := []int32{cost}
		if paidAtOrder {
			want = nil
		}
		if len(falls) != len(want) || len(want) == 1 && falls[0] != cost {
			t.Fatalf("paid at order %v: mana fell by %v after the LOAD from %d, want %v", paidAtOrder, falls, full, want)
		}
	}
}
