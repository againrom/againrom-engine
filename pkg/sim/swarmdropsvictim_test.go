package sim

import "testing"

// A Swarm member on the commanded cell with no target assigned loses its victim (AI-910).
func TestASwarmMemberOnItsCellDropsAVictimNothingScores(t *testing.T) {
	t.Parallel()

	m := engFighter(1, 5, 5, 5)
	foe := engFighter(2, 9, 6, 5)
	w := engWorld(t, engRel(t, [3]uint32{5, 9, 1}), m, foe)
	setGroupCommand(t, w, 5, 0, orderSwarm, 5, 5)
	i := indexOfEntity(w.entities, 1)
	if !w.orderAttack(i, 2) {
		t.Fatal("fixture: the member refused its victim")
	}
	if e := w.entities[i]; !e.HasAttackTarget || e.HasTarget || e.Transit != 0 {
		t.Fatalf("fixture: member must hold the victim standing on its cell, got %+v", e)
	}
	w.relations = engRel(t)

	regDecide(t, w, 5, 0, []int{i})

	if e := w.entities[i]; e.HasAttackTarget && e.PendingOrder.Kind != PendingRelease {
		t.Errorf("a Swarm member on its cell kept victim %d when the group scored nothing", e.AttackTarget)
	}
	if o := w.savedOrder(1); o == nil || o.Raw[8] != 0xb {
		t.Errorf("the member's order byte is not 0xb")
	}
}
