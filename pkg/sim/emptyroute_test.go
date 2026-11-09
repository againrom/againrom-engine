package sim

import (
	"encoding/binary"
	"testing"
)

func TestRePickUsesTheCurrentAttackRange(t *testing.T) {
	t.Parallel()
	// R2-ENGINE-1019: the range operand comes from the shared attack-range rule.
	for _, second := range []bool{false, true} {
		self := asFighter(asHero, SelfSlot, 30, 40)
		self.MaxMana, self.Mana, self.AutoSpell, self.KnownSpells = 100, 100, 1, 1<<1
		w := asWorld(t, second, self, asFoe(asFoeA, 33, 40))
		w.spells = []SpellRule{wpnRule(6, 6, 5)}
		if w.attackRange(0) < 3 {
			t.Fatal("fixture does not distinguish attack range from physical reach")
		}
		got := w.reacquisitionVictim(0)
		if (got == 1) != second {
			t.Fatalf("second=%v re-pick=%d; want the effective range only in the second game", second, got)
		}
	}
}

func TestEmptyNonMoveRoutePreparesAcquisitionWithoutChangingTheMemberOrder(t *testing.T) {
	t.Parallel()
	// R2-ENGINE-1108 and R2-ENGINE-1019: other orders re-pick into activity 6.
	for _, state := range []uint32{3, 8, 0xb, 0xc, 0x16} {
		w := asWorld(t, true, asFighter(asHero, SelfSlot, 30, 40), asFoe(asFoeA, 31, 40))
		w.commandGroup([]int{0}, orderNone, cell{})
		w.entities[0].ActorState = uint8(state)
		o := w.syncSavedActorCommand(0)
		o.Raw[8] = 1
		binary.LittleEndian.PutUint32(o.Raw[0x20:], 0x12345678)
		w.answerRefusedWalk(0, searchNear)
		e := w.entities[0]
		if !e.HasAttackTarget || e.AttackTarget != asFoeA || !e.AcquirePursuit || o.Raw[8] != 6 || o.State != state {
			t.Fatalf("member order %d: activity=%d target=%v/%d acquisition=%v order=%d", state, o.Raw[8], e.HasAttackTarget, e.AttackTarget, e.AcquirePursuit, o.State)
		}
		if binary.LittleEndian.Uint32(o.Raw[0xc:]) != w.currentActionKey(asFoeA, AttackTargetUnit) || binary.LittleEndian.Uint32(o.Raw[0x20:]) != 0x12345678 {
			t.Fatal("re-pick missed its prepared target or changed the assigned target")
		}
	}
}

func TestEmptyRouteWithoutATargetUsesTheOwnerSupportByte(t *testing.T) {
	t.Parallel()
	// R2-ENGINE-1019: no target writes 0 for a player and 11 for a computer.
	for _, owner := range []uint32{SelfSlot, asSide} {
		self := asFighter(asHero, owner, 30, 40)
		w := asWorld(t, true, self)
		w.commandGroup([]int{0}, orderNone, cell{})
		o := w.savedOrder(asHero)
		o.Raw[8] = 1
		binary.LittleEndian.PutUint32(o.Raw[0xc:], 0x12345678)
		w.answerRefusedWalk(0, searchNear)
		want := byte(0)
		if owner == asSide {
			want = 0xb
		}
		if o.Raw[8] != want || binary.LittleEndian.Uint32(o.Raw[0xc:]) != 0x12345678 || w.entities[0].HasAttackTarget {
			t.Fatalf("owner=%d activity=%d, want %d with target bytes retained", owner, o.Raw[8], want)
		}
	}
}

func TestEmptyRouteMageExceptionWritesThePickBeforeIdling(t *testing.T) {
	t.Parallel()
	// R2-ENGINE-1019/1106: player mage/range exception follows target selection.
	self := asFighter(asHero, SelfSlot, 30, 40)
	self.MaxMana = 100
	w := asWorld(t, true, self, asFoe(asFoeA, 31, 40))
	w.commandDefend([]int{0}, asHero)
	w.savedOrder(asHero).Raw[8] = 1
	w.answerRefusedWalk(0, searchNear)
	o, e := w.savedOrder(asHero), w.entities[0]
	if !e.PursuitIdle || !e.HasAttackTarget || e.AttackTarget != asFoeA || o.Raw[8] != 0 || binary.LittleEndian.Uint32(o.Raw[0xc:]) != w.currentActionKey(asFoeA, AttackTargetUnit) {
		t.Fatal("mage re-pick did not retain its selected target with activity zero")
	}
}
