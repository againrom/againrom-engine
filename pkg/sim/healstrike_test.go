package sim

import (
	"fmt"
	"testing"
)

func hsMelee(t *testing.T, auto uint16, mana int32) *World {
	t.Helper()
	mage := acCaster(1, 2, 2, mana, 1<<6, auto)
	mage.Owner, mage.Reach, mage.DamageBase, mage.AlwaysHits = SelfSlot, 1, 5, true
	mage.AttackCharge, mage.AttackRelax = 6, 4
	ally := spEnt(2, 3, 2)
	ally.Owner = SelfSlot
	foe := spEnt(3, 2, 3)
	foe.Owner, foe.Reach = 2, 1
	foe.HP, foe.MaxHP = 30000, 30000
	return hlWorld(t, 71, acEnemies(t), []SpellRule{hlHeal()}, mage, ally, foe)
}

func hsToSecondCharge(t *testing.T, w *World, ordered bool) {
	t.Helper()
	var first []Command
	if ordered {
		first = []Command{Attack(1, 3)}
	} else {
		// A player's mage of reach below 2 starts no strike on its own
		// (AI-GUARD-007); the unordered series is a mage with a staff's.
		w.entities[indexOfEntity(w.entities, 1)].Reach = 2
	}
	Step(w, first)
	for k := 0; k < 200; k++ {
		if e := spAt(t, w, 1); e.AttackPhase == AttackCharging && spAt(t, w, 3).HP < 30000 {
			return
		}
		Step(w, nil)
	}
	t.Fatal("fixture: the mage never charged a second strike")
}

func hsBooked(w *World) bool {
	_, ok := w.bookCastIndex(1)
	return ok
}

func TestAnArmedHealWaitsForTheRunningStrikeAndEntersAtItsExit(t *testing.T) {
	t.Parallel()

	for _, ordered := range []bool{false, true} {
		t.Run(fmt.Sprint("ordered=", ordered), func(t *testing.T) {
			w := hsMelee(t, 6, 50)
			hsToSecondCharge(t, w, ordered)
			before := spAt(t, w, 1)
			w.entities[indexOfEntity(w.entities, 2)].HP = 95
			foeHP := spAt(t, w, 3).HP
			exit, begin := uint64(0), uint64(0)
			for k := 0; k < 60 && begin == 0; k++ {
				ready := w.orderStateAt(indexOfEntity(w.entities, 1)).phase() == orderPhaseNone
				Step(w, nil)
				if hsBooked(w) {
					begin = w.Tick()
					if !ready {
						t.Fatalf("tick %d: the heal began while the strike still ran", begin)
					}
					break
				}
				if exit == 0 && spAt(t, w, 1).AttackPhase == AttackReady {
					exit = w.Tick()
				}
			}
			if begin == 0 {
				t.Fatal("the armed heal never began while the mage struck the hostile in its reach")
			}
			if begin != exit+1 {
				t.Errorf("the heal began on tick %d, want %d, the first tick after the strike's exit on %d", begin, exit+1, exit)
			}
			if spAt(t, w, 3).HP >= foeHP {
				t.Error("the charging strike was cut: the hostile took no blow before the heal")
			}
			for k := 0; k < 80 && (hsBooked(w) || spAt(t, w, 1).CastWait != 0); k++ {
				Step(w, nil)
			}
			if spAt(t, w, 2).HP != 100 {
				t.Fatal("the heal did not reach the wounded ally")
			}
			after := spAt(t, w, 1)
			if !after.HasAttackTarget || after.AttackTarget != 3 || after.Group != before.Group || after.ActorState != before.ActorState {
				t.Fatalf("after the heal the mage holds victim %v/%d, group %d, state %d; before: victim 3, group %d, state %d",
					after.HasAttackTarget, after.AttackTarget, after.Group, after.ActorState, before.Group, before.ActorState)
			}
			healed := spAt(t, w, 3).HP
			for k := 0; k < 40 && spAt(t, w, 3).HP == healed; k++ {
				Step(w, nil)
			}
			if spAt(t, w, 3).HP == healed {
				t.Error("the mage did not strike its victim again after the heal")
			}
		})
	}
}

func TestAnArmedHealBeginsWhenAPlayersCastGivenInTheStrikeIsAdmitted(t *testing.T) {
	t.Parallel()

	for _, ordered := range []bool{false, true} {
		t.Run(fmt.Sprint("ordered=", ordered), func(t *testing.T) {
			auto, player := hsMelee(t, 6, 50), hsMelee(t, 0, 50)
			hsToSecondCharge(t, auto, ordered)
			hsToSecondCharge(t, player, ordered)
			if auto.Tick() != player.Tick() {
				t.Fatalf("fixture: the two mages reached the charge on ticks %d and %d", auto.Tick(), player.Tick())
			}
			auto.entities[indexOfEntity(auto.entities, 2)].HP = 40
			player.entities[indexOfEntity(player.entities, 2)].HP = 40
			cmds := []Command{Cast(1, 2, 6)}
			begin, admitted := uint64(0), uint64(0)
			for k := 0; k < 60 && (begin == 0 || admitted == 0); k++ {
				Step(auto, nil)
				Step(player, cmds)
				cmds = nil
				if begin == 0 && hsBooked(auto) {
					begin = auto.Tick()
				}
				if admitted == 0 && spAt(t, player, 1).PendingOrder.RowAdmitted {
					admitted = player.Tick()
				}
			}
			if begin == 0 || admitted == 0 {
				t.Fatalf("the armed heal began on tick %d and the player's cast was admitted on tick %d", begin, admitted)
			}
			if begin != admitted {
				t.Errorf("the armed heal began on tick %d, the player's cast given at the same moment was admitted on tick %d", begin, admitted)
			}
		})
	}
}

func TestAnArmedHealWithoutAPatientOrManaLeavesTheStrikeSeriesUnchanged(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name    string
		mana    int32
		patient bool
	}{{"no patient", 50, false}, {"no mana", 5, true}} {
		t.Run(tc.name, func(t *testing.T) {
			armed, unarmed := hsMelee(t, 6, tc.mana), hsMelee(t, 0, tc.mana)
			hsToSecondCharge(t, armed, false)
			hsToSecondCharge(t, unarmed, false)
			if tc.patient {
				armed.entities[indexOfEntity(armed.entities, 2)].HP = 40
				unarmed.entities[indexOfEntity(unarmed.entities, 2)].HP = 40
			}
			for k := 0; k < 200; k++ {
				Step(armed, nil)
				Step(unarmed, nil)
				for _, id := range []EntityID{1, 2, 3} {
					a, u := spAt(t, armed, id), spAt(t, unarmed, id)
					a.AutoSpell, u.AutoSpell = 0, 0
					if fmt.Sprintf("%+v", a) != fmt.Sprintf("%+v", u) {
						t.Fatalf("tick %d: entity %d differs beside the armed mage:\n%+v\n%+v", armed.Tick(), id, a, u)
					}
				}
				if hsBooked(armed) {
					t.Fatalf("tick %d: the armed mage began a cast", armed.Tick())
				}
			}
			if spAt(t, armed, 3).HP >= 30000 {
				t.Error("fixture: the mage never struck")
			}
		})
	}
}
