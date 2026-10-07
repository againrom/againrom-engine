package sim

import "testing"

// haWorld is a participant's party in a row at y=2: a warrior (1) and a mage
// with no weapon spell that knows Heal (2), a wounded ally (3), and a hostile
// (4) standing far enough away that nobody reaches it during the test.
func haWorld(t *testing.T, mageKnows uint32, mageWeaponSpell uint16) *World {
	t.Helper()
	warrior := spEnt(1, 2, 2)
	warrior.Owner, warrior.Reach, warrior.ScanRange = SelfSlot, 1, sightRings
	mage := acCaster(2, 3, 2, 200, mageKnows, 0)
	mage.Owner = SelfSlot
	mage.WeaponSpell = mageWeaponSpell
	hurt := spEnt(3, 4, 2)
	hurt.Owner, hurt.HP = SelfSlot, 40
	hostile := spEnt(4, 14, 2)
	hostile.Owner = 2
	rel := engRel(t, [3]uint32{SelfSlot, 2, relationHostile}, [3]uint32{2, SelfSlot, relationHostile})
	return hlWorld(t, 91, rel, []SpellRule{hlHeal(), hlArrow()}, warrior, mage, hurt, hostile)
}

func haRun(w *World, first []Command, ticks int) {
	Step(w, first)
	for range ticks - 1 {
		Step(w, nil)
	}
}

// TestAGroupAttackOrderLeavesAStafflessHealerHealing: the party is selected and
// ordered onto the hostile. The warrior takes the order; the mage with no staff
// has nothing to strike with and keeps healing the wounded ally instead of
// walking off to fight bare-handed.
func TestAGroupAttackOrderLeavesAStafflessHealerHealing(t *testing.T) {
	w := haWorld(t, 1<<6, 0)
	haRun(w, []Command{Attack(1, 4), Attack(2, 4)}, 40)
	if e := spAt(t, w, 1); !e.HasAttackTarget || e.AttackTarget != 4 {
		t.Errorf("the warrior did not take the attack order: %+v", e)
	}
	if e := spAt(t, w, 2); e.HasAttackTarget || underCommand(e) {
		t.Errorf("the staffless healer took the group's attack order: target=%v/%d", e.HasAttackTarget, e.AttackTarget)
	}
	if got := spAt(t, w, 3).HP; got <= 40 {
		t.Errorf("the wounded ally holds %d health, want the healer's autoheal to have landed", got)
	}
}

// TestALoneStafflessMageObeysAnAttackOrder: a mage ordered alone, with no group,
// is told exactly what to do and does it.
func TestALoneStafflessMageObeysAnAttackOrder(t *testing.T) {
	w := haWorld(t, 1<<6, 0)
	haRun(w, []Command{Attack(2, 4)}, 4)
	if e := spAt(t, w, 2); !e.HasAttackTarget || e.AttackTarget != 4 {
		t.Errorf("the lone mage did not take its attack order: %+v", e)
	}
}

// TestAMageWithAStaffOrAWithoutHealingTakesAGroupAttackOrder: only the mage that
// both lacks a weapon and can heal is left out of a group's order.
func TestAMageWithAStaffOrAWithoutHealingTakesAGroupAttackOrder(t *testing.T) {
	for _, tc := range []struct {
		name  string
		known uint32
		staff uint16
	}{
		{"staff in hand", 1<<6 | 1<<1, 1},
		{"no Heal in the book", 1 << 1, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := haWorld(t, tc.known, tc.staff)
			haRun(w, []Command{Attack(1, 4), Attack(2, 4)}, 4)
			if e := spAt(t, w, 2); !e.HasAttackTarget || e.AttackTarget != 4 {
				t.Errorf("the mage did not take the group's attack order: %+v", e)
			}
		})
	}
}
