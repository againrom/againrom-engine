package game

import (
	"reflect"
	"strings"
	"testing"
)

func TestSavedNewGroup1115LateGapIsAtomic(t *testing.T) {
	front := groupDocumentFront1115(t)
	open, town, err := front.RestoreOriginal(groupDocumentLiteral1115(t, front))
	if err != nil || town {
		t.Fatal(town, err)
	}
	if err := front.App("new Group atomic projection").OpenMission(open); err != nil {
		t.Fatal(err)
	}
	before := groupDocumentSnapshot1115(t, front)
	state, err := cloneSavedDocument(before.SavedDocument)
	if err != nil {
		t.Fatal(err)
	}
	actor := registryActors1111(t, front.live.world)[35]
	front.live.enqueue(uint32(actor.ID), 20, 16)
	front.live.tick()
	groups, orders, _ := front.live.world.SavedGroups()
	if len(groups) != 2 || !groups[1].Authored || groups[1].ContainerID == 0 {
		t.Fatal("missing generated Group", groups)
	}
	hash := front.live.world.Hash()
	if err := projectSavedGroups(state, front.live.world); err != nil {
		t.Fatal(err)
	}
	if state.GroupBindings.Unavailable != "" || len(state.GroupBindings.Groups) != 2 || hash != front.live.world.Hash() {
		t.Fatal("ordinary move did not publish its complete new Group")
	}
	if _, err := cloneSavedDocument(state); err != nil {
		t.Fatal("new Group/binding transaction is inconsistent", err)
	}
	state, err = cloneSavedDocument(before.SavedDocument)
	if err != nil {
		t.Fatal(err)
	}
	// A malformed final order must roll back even a newly allocated Group.
	orders[len(orders)-1].Patrol = make([]uint16, savedGroupFieldListLimit+1)
	if err := front.live.world.ImportSavedGroups(groups, orders); err != nil {
		t.Fatal(err)
	}
	hash = front.live.world.Hash()
	if err := projectSavedGroups(state, front.live.world); err == nil || !strings.Contains(err.Error(), "patrol exceeds list bound") {
		t.Fatal("invalid order after new Group creation was not rejected", err)
	}
	if !reflect.DeepEqual(state, before.SavedDocument) || hash != front.live.world.Hash() {
		t.Fatal("late invalid order leaked new Group, changed Player/actor bindings or mutated World")
	}
}
