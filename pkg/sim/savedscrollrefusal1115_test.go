package sim

import "testing"

func TestSavedScrollCompletionPreservesSharedChildAndRefund(t *testing.T) {
	w := scrollWorld1090(t, 2, 1)
	r := bindOperationsPack1115(t, w, 0)
	r.Effects[0].ExternalReferences = 1
	hp, next := w.entities[1].HP, r.NextID
	child := r.Effects[0]
	Step(w, []Command{UseScroll(1, 0, 2)})
	if casts := w.ScrollCasts(); len(casts) != 1 || casts[0].Reservation == 0 || casts[0].Item.ObjectID != 10 {
		t.Fatal("scroll did not reserve its exact current Item")
	}
	refunded := reloadOperations1115(t, w)
	if !refunded.cancelScroll(0) || len(refunded.scrollCasts) != 0 || len(refunded.carried[0]) != 1 || refunded.carried[0][0].ObjectID != 10 || refunded.carried[0][0].Count != 1 || refunded.savedObjects.Effects[0] != child {
		t.Fatal("cancelled scroll did not return its exact Item and shared child")
	}
	reloadOperations1115(t, refunded)
	for range 20 {
		cold := reloadOperations1115(t, w)
		Step(w, nil)
		Step(cold, nil)
		if w.Hash() != cold.Hash() {
			t.Fatal("shared-child scroll completion changed native continuation")
		}
	}
	casts := w.ScrollCasts()
	if len(casts) != 0 || w.entities[1].HP >= hp || w.savedObjects.NextID != next+1 || !w.savedObjects.item(10).Retired || len(w.savedObjects.Locations(10)) != 0 || w.savedObjects.Effects[0] != child {
		t.Fatal("completion lost the surviving child or retained the consumed Item", casts)
	}
	reloadOperations1115(t, w)
}
