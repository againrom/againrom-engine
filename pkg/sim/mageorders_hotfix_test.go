package sim

import "testing"

// moWorld is the human participant's weapon-spell mage at (4,8) with Fire
// Arrow and Shield in its book, a nearer hostile (id 2) at (7,8) and a farther
// one (id 3) at (9,8). Both hostiles are far too healthy to die and carry no
// attack, so every change in their health is the mage's doing. auto is the
// spell armed for autocast, zero for none.
func moWorld(t *testing.T, auto uint16) *World {
	t.Helper()
	mage := wpnCaster(1, 4, 8, 1, 30, 4, 4)
	mage.Owner = SelfSlot
	mage.Mind, mage.MaxMana, mage.Mana = 30, 500, 500
	mage.KnownSpells = 1<<1 | 1<<18 | 1<<26
	mage.AutoSpell = auto
	mage.AttackCharge, mage.AttackRelax = 8, 12
	near := spEnt(2, 7, 8)
	near.Owner = 2
	near.HP, near.MaxHP = 5000, 5000
	far := spEnt(3, 9, 8)
	far.Owner = 2
	far.HP, far.MaxHP = 5000, 5000
	rel := engRel(t, [3]uint32{SelfSlot, 2, relationHostile}, [3]uint32{2, SelfSlot, 2})
	return hlWorld(t, 5, rel, []SpellRule{hlArrow(), acbShield(), {ID: 26, ManaCost: 7, MaxRange: 10}}, mage, near, far)
}

const (
	moMage = EntityID(1)
	moNear = EntityID(2)
	moFar  = EntityID(3)
)

func moHealth(w *World, id EntityID) int32 { return w.entities[indexOfEntity(w.entities, id)].HP }

func moMageEntity(w *World) Entity { return w.entities[indexOfEntity(w.entities, moMage)] }

func moShielded(w *World) bool {
	_, standing := effectIndex(w.attached, moMage, 18)
	return standing
}

// TestAnArmedMageOrderedOntoTheFartherEnemyFiresOnlyAtIt is the retargeting
// half: an armed offensive row fired at the nearest hostile in range whatever
// enemy the mage was ordered onto, so the ordered enemy took nothing while the
// nearer one died.
func TestAnArmedMageOrderedOntoTheFartherEnemyFiresOnlyAtIt(t *testing.T) {
	w := moWorld(t, 1)
	Step(w, []Command{Attack(moMage, moFar)})
	for range 200 {
		Step(w, nil)
	}
	if hp := moHealth(w, moNear); hp != 5000 {
		t.Errorf("the mage fired at the nearer enemy it was not ordered onto: it lost %d health", 5000-hp)
	}
	if hp := moHealth(w, moFar); hp >= 5000 {
		t.Errorf("the enemy the mage was ordered onto took no damage in 200 ticks")
	}
	if m := moMageEntity(w); !m.HasAttackTarget || m.AttackTarget != moFar {
		t.Errorf("the mage no longer holds its ordered victim: %+v", m)
	}
}

// TestAnAttackOrderIsTakenOnEveryTickOfAnArmedMagesCycle is the ignored-order
// half: the attack arm dropped an order whenever a book cast was winding up or
// recovering, and an armed mage with an enemy in range is in one or the other
// nearly every tick, so an order onto the farther enemy was mostly ignored.
func TestAnAttackOrderIsTakenOnEveryTickOfAnArmedMagesCycle(t *testing.T) {
	missed := 0
	for offset := range 60 {
		w := moWorld(t, 1)
		Step(w, []Command{Attack(moMage, moNear)})
		for range offset {
			Step(w, nil)
		}
		Step(w, []Command{Attack(moMage, moFar)})
		m := moMageEntity(w)
		id, kind, accepted := m.RequestedAttackTarget()
		if !accepted || id != moFar || kind != AttackTargetUnit || m.ActorState != actorStateEngage {
			missed++
			t.Errorf("order on tick %d of the cycle was ignored: requested victim %v/%d state %d", offset,
				accepted, id, m.ActorState)
		}
		farHP := moHealth(w, moFar)
		for range 160 {
			Step(w, nil)
		}
		if moHealth(w, moFar) >= farHP {
			t.Errorf("accepted order on offset %d never reached its requested victim", offset)
		}
	}
	if missed != 0 {
		t.Errorf("%d of 60 orders were ignored", missed)
	}
}

// TestAnAttackOrderGivenDuringAWindUpWaitsForTheCast pins what the order does
// beside a cast it does not own: the cast is neither cancelled nor refunded and
// the body stays where it is, the victim is taken at once, and the fight on
// that victim begins after the cast.
func TestAnAttackOrderGivenDuringAWindUpWaitsForTheCast(t *testing.T) {
	w := moWorld(t, 0)
	mana := moMageEntity(w).Mana
	Step(w, []Command{Cast(moMage, moMage, 18)})
	if _, pending := w.bookCastIndex(moMage); !pending {
		t.Fatal("the fixture admitted no cast")
	}
	Step(w, []Command{Attack(moMage, moFar)})
	m := moMageEntity(w)
	if !m.HasAttackTarget || m.AttackTarget != moFar || m.ActorState != actorStateEngage {
		t.Fatalf("the order was not taken beside the cast: victim %v/%d state %d", m.HasAttackTarget, m.AttackTarget, m.ActorState)
	}
	if _, pending := w.bookCastIndex(moMage); !pending && !moShielded(w) {
		t.Error("the order cancelled the pending cast")
	}
	if hp := moHealth(w, moFar); hp != 5000 {
		t.Errorf("the mage struck while its cast owned it: enemy lost %d", 5000-hp)
	}
	for range 80 {
		Step(w, nil)
	}
	if got, want := moMageEntity(w).Mana, mana-acbShield().ManaCost; !moShielded(w) || got != want {
		t.Errorf("the cast did not apply and pay once: shielded=%v mana %d want %d", moShielded(w), got, want)
	}
	if hp := moHealth(w, moFar); hp >= 5000 {
		t.Error("the mage never attacked the victim it was ordered onto")
	}
}

// TestASelfCastDuringAnOrderedAttackInterruptsCastsAndResumes is the
// self-cast half. The cast was admitted, but the manual cast cleared the
// victim with the rest of the mage's action and left it standing in acquire, so
// the fight never came back.
func TestASelfCastDuringAnOrderedAttackInterruptsCastsAndResumes(t *testing.T) {
	for _, tc := range []struct {
		name string
		auto uint16
	}{{"unarmed", 0}, {"armed", 1}} {
		t.Run(tc.name, func(t *testing.T) {
			w := moWorld(t, tc.auto)
			Step(w, []Command{Attack(moMage, moFar)})
			for range 20 {
				Step(w, nil)
			}
			if hp := moHealth(w, moFar); hp >= 5000 {
				t.Fatal("the fixture never began the fight")
			}

			Step(w, []Command{Cast(moMage, moMage, 18)})
			m := moMageEntity(w)
			if _, pending := w.bookCastIndex(moMage); !pending {
				t.Fatal("the self-cast was not admitted")
			}
			if !m.HasAttackTarget || m.AttackTarget != moFar || m.ActorState != actorStateEngage {
				t.Fatalf("the cast dropped the ordered attack: victim %v/%d state %d", m.HasAttackTarget, m.AttackTarget, m.ActorState)
			}
			if m.AttackPhase != AttackReady || m.AttackCountdown != 0 {
				t.Errorf("the cast did not interrupt the loaded attack cycle: phase %d count %d", m.AttackPhase, m.AttackCountdown)
			}

			back := worldRoundTripForTest(t, w)
			farAtCast, nearAtCast := moHealth(w, moFar), moHealth(w, moNear)
			applied := false
			for range 200 {
				Step(w, nil)
				Step(back, nil)
				if w.Hash() != back.Hash() {
					t.Fatalf("native reload diverged at tick %d", w.Tick())
				}
				applied = applied || moShielded(w)
				if !applied && moHealth(w, moFar) != farAtCast {
					t.Fatalf("the enemy was struck before the self-cast applied, at tick %d", w.Tick())
				}
			}
			if !moShielded(w) {
				t.Error("the self-cast never applied")
			}
			if hp := moHealth(w, moFar); hp >= farAtCast {
				t.Error("the mage did not resume attacking the enemy it was ordered onto")
			}
			if hp := moHealth(w, moNear); hp != nearAtCast {
				t.Errorf("the mage turned on the nearer enemy after the cast: it lost %d", nearAtCast-hp)
			}
			if m := moMageEntity(w); !m.HasAttackTarget || m.AttackTarget != moFar {
				t.Errorf("the mage no longer holds its ordered victim: %+v", m)
			}
		})
	}
}

// TestASelfCastWithNoOrderedAttackStillStandsInAcquire keeps the rest of the
// manual cast as it was: a mage holding no ordered victim takes acquire in
// place, and a cast at a cell, which is a repositioning as often as an attack,
// does not bring an ordered attack back.
func TestASelfCastWithNoOrderedAttackStillStandsInAcquire(t *testing.T) {
	w := moWorld(t, 0)
	Step(w, []Command{Cast(moMage, moMage, 18)})
	if m := moMageEntity(w); m.HasAttackTarget || m.ActorState != actorStateAcquire {
		t.Errorf("an idle mage's cast left state %d with victim %v", m.ActorState, m.HasAttackTarget)
	}

	w = moWorld(t, 0)
	Step(w, []Command{Attack(moMage, moFar)})
	for range 20 {
		Step(w, nil)
	}
	Step(w, []Command{CastAt(moMage, 26, CellPoint{X: 3, Y: 8})})
	if m := moMageEntity(w); m.HasAttackTarget || m.ActorState != actorStateAcquire {
		t.Errorf("a cast at a cell resumed the attack: state %d victim %v", m.ActorState, m.HasAttackTarget)
	}
}
