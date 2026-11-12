package sim

import (
	"encoding/binary"
	"testing"
)

// supportPress issues L's order to ids as one press.
func supportPress(w *World, ids ...EntityID) {
	var cmds []Command
	for _, id := range ids {
		cmds = append(cmds, GroupStance(id, OrderSupport, 7))
	}
	Step(w, cmds)
}

// supportHeld fails unless id holds L's order whole: member order 0x17 on the
// entity and its SAV order, prepared 0, the command cell its own cell, and a
// Group at programme 0x13.
func supportHeld(t *testing.T, w *World, id EntityID) *SavedGroup {
	t.Helper()
	e := laEnt(t, w, id)
	o := w.savedOrder(id)
	g := w.savedGroupFor(id)
	if e.ActorState != actorStateSupport || o == nil || o.State != uint32(actorStateSupport) || g == nil || g.AI[0x20] != programmeSupport {
		t.Fatalf("entity %d: state %#x, order %+v, group %+v; want 0x17 under programme 0x13", id, e.ActorState, o, g)
	}
	if o.Raw[8] != 0 || o.Raw[0] != uint8(e.PostX) || o.Raw[1] != uint8(e.PostY) || binary.LittleEndian.Uint32(o.Raw[0x50:]) != 0 {
		t.Fatalf("entity %d: command block prepared %d cell (%d,%d) +0x50 %d, post (%d,%d)", id, o.Raw[8], o.Raw[0], o.Raw[1], binary.LittleEndian.Uint32(o.Raw[0x50:]), e.PostX, e.PostY)
	}
	return g
}

func TestSecondGameSupportOrderResetsEachSelectedUnit(t *testing.T) {
	w := asWorld(t, true, asFighter(asHero, SelfSlot, 30, 40), asFighter(3, SelfSlot, 30, 42), asFoe(asFoeA, 40, 40))
	Step(w, []Command{Attack(asHero, asFoeA), GroupMoveTo(3, CellPoint{X: 20, Y: 42}, 1)})
	asRun(w, 3)
	if e := laEnt(t, w, asHero); !e.HasAttackTarget {
		t.Fatal("fixture: the hero holds no attack order before L")
	}
	supportPress(w, asHero, 3)
	a, b := supportHeld(t, w, asHero), supportHeld(t, w, 3)
	if a.ID != b.ID {
		t.Fatalf("one press made Groups %d and %d, want one", a.ID, b.ID)
	}
	for _, id := range []EntityID{asHero, 3} {
		e := laEnt(t, w, id)
		if e.HasAttackTarget || e.HasPendingAttackTarget || e.HasTarget && (e.TargetX != e.X || e.TargetY != e.Y) || e.PostX != e.X || e.PostY != e.Y {
			t.Fatalf("entity %d keeps an order after L: %+v", id, e)
		}
	}
}

func TestSecondGameSupportOrderKeepsANonCasterStanding(t *testing.T) {
	for _, tc := range []struct {
		name    string
		foeX    int32
		attacks bool
		press   bool
	}{
		{"idle, enemy in sight", 33, false, false},
		{"L, enemy in sight", 33, false, true},
		{"idle, enemy attacks", 31, true, false},
		{"L, enemy attacks", 31, true, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			foe := asFighter(asFoeA, asSide, tc.foeX, 40)
			foe.DamageBase = 1
			w := asWorld(t, true, asFighter(asHero, SelfSlot, 30, 40), foe)
			if tc.press {
				supportPress(w, asHero)
			}
			if tc.attacks {
				Step(w, []Command{Attack(asFoeA, asHero)})
			}
			fought := false
			for range 200 {
				Step(w, nil)
				e := laEnt(t, w, asHero)
				fought = fought || e.HasAttackTarget || e.HasPendingAttackTarget || e.X != 30 || e.Y != 40
			}
			if fought == tc.press {
				t.Fatalf("hero fought or moved = %v under L = %v", fought, tc.press)
			}
			if tc.press {
				supportHeld(t, w, asHero)
			}
			if hero := laEnt(t, w, asHero); tc.attacks && tc.press && hero.HP == hero.MaxHP {
				t.Fatal("fixture: the enemy never struck the hero")
			}
		})
	}
}

// supportCasts presses L for the mage of computerSupportFixture and returns
// the casts it makes over n steps.
func supportCasts(t *testing.T, w *World, n int) []CastEvent {
	t.Helper()
	Step(w, []Command{GroupStance(1, OrderSupport, 7)})
	var out []CastEvent
	for range n {
		for _, ev := range StepObserved(w, nil) {
			if ev.Caster == 1 {
				out = append(out, ev)
			}
		}
	}
	supportHeld(t, w, 1)
	return out
}

func TestSecondGameSupportOrderCasterHealsThenShieldsThenHealsAnAlly(t *testing.T) {
	w, shield, heal := computerSupportFixture(t, true, SelfSlot, 0, 60, true, true, 3)
	w.entities[0].HP, w.entities[1].HP = 50, 50
	first := map[[2]uint16]int{}
	for k, ev := range supportCasts(t, w, 150) {
		key := [2]uint16{ev.Spell, uint16(ev.Target)}
		if _, ok := first[key]; !ok {
			first[key] = k + 1
		}
	}
	self, own, ally := first[[2]uint16{heal, 1}], first[[2]uint16{shield, 1}], first[[2]uint16{heal, 2}]
	if self == 0 || own == 0 || ally == 0 || !(self < own && own < ally) {
		t.Fatalf("first self heal %d, self shield %d, ally heal %d; want all, in that order", self, own, ally)
	}
}

func TestSecondGameSupportOrderCasterWaitsForFiveMana(t *testing.T) {
	for _, mana := range []int32{4, 5} {
		w, _, heal := computerSupportFixture(t, true, SelfSlot, 0, mana, true, false, 3)
		w.entities[1].HP = 50
		casts := 0
		for _, ev := range supportCasts(t, w, 60) {
			if ev.Spell == heal && ev.Target == 2 {
				casts++
			}
		}
		if (casts > 0) != (mana >= 5) {
			t.Fatalf("mana %d: %d ally heals", mana, casts)
		}
	}
}

func TestSecondGameSupportOrderEndsOnAnotherOrder(t *testing.T) {
	for _, tc := range []struct {
		name  string
		order Command
	}{
		{"guard", GroupStance(asHero, OrderGuard, 9)},
		{"stand ground", GroupStance(asHero, OrderStandGround, 9)},
		{"move", GroupMoveTo(asHero, CellPoint{X: 25, Y: 40}, 9)},
		{"attack", Attack(asHero, asFoeA)},
		{"single move", MoveTo(asHero, CellPoint{X: 25, Y: 40})},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := asWorld(t, true, asFighter(asHero, SelfSlot, 30, 40), asFoe(asFoeA, 36, 40))
			supportPress(w, asHero)
			supportHeld(t, w, asHero)
			Step(w, []Command{tc.order})
			e, o, g := laEnt(t, w, asHero), w.savedOrder(asHero), w.savedGroupFor(asHero)
			if e.ActorState == actorStateSupport || o == nil || o.State == uint32(actorStateSupport) || g == nil || g.AI[0x20] == programmeSupport {
				t.Fatalf("after %s: state %#x, order %+v, group %+v", tc.name, e.ActorState, o, g)
			}
		})
	}
	w, _, heal := computerSupportFixture(t, true, SelfSlot, 0, 60, true, false, 3)
	w.entities[1].HP = 50
	supportCasts(t, w, 1)
	Step(w, []Command{Cast(1, 2, SpellID(heal))})
	if e := w.entities[0]; e.ActorState == actorStateSupport || w.savedOrder(1).State == uint32(actorStateSupport) {
		t.Fatalf("a cast order left state %#x", e.ActorState)
	}
}

func TestFirstGameRefusesTheSupportOrder(t *testing.T) {
	build := func() *World {
		return asWorld(t, false, asFighter(asHero, SelfSlot, 30, 40), asFoe(asFoeA, 33, 40))
	}
	pressed, idle := build(), build()
	supportPress(pressed, asHero)
	Step(idle, nil)
	if pressed.Hash() != idle.Hash() {
		t.Fatal("the first game took the second game's L order")
	}
	if e := laEnt(t, pressed, asHero); e.ActorState == actorStateSupport {
		t.Fatalf("first-game state %#x", e.ActorState)
	}
}

func TestSecondGameSupportOrderSurvivesTheByteForm(t *testing.T) {
	foe := asFighter(asFoeA, asSide, 31, 40)
	foe.DamageBase = 1
	w := asWorld(t, true, asFighter(asHero, SelfSlot, 30, 40), foe)
	supportPress(w, asHero)
	Step(w, []Command{Attack(asFoeA, asHero)})
	back := worldRoundTripForTest(t, w)
	supportHeld(t, back, asHero)
	for range 60 {
		Step(back, nil)
		if e := laEnt(t, back, asHero); e.HasAttackTarget || e.X != 30 || e.Y != 40 {
			t.Fatalf("restored hero left L's order: %+v", e)
		}
	}
	supportHeld(t, back, asHero)
}
