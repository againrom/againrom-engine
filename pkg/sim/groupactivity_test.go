package sim

import "testing"

// TestAnUnnamedGroupDecidesOnlyWithAPartyUnitNear checks the activity gate
// itself (AI-ACTIVITY-324). A Group the script does not name (+0x48 = 0) has
// +0x45 rebuilt each tick from the party's coverage: with no party unit near it
// holds zero and the Group does not decide, so its Roam order never rolls its
// counter (+0x15) or its Group cell (+0x0a). Once a party unit comes near,
// +0x45 counts the members and the same order is evaluated.
func TestAnUnnamedGroupDecidesOnlyWithAPartyUnitNear(t *testing.T) {
	w, err := NewWorld(17, Bounds{Width: 80, Height: 80}, ModeCanonical, nil, []Entity{
		{ID: 1, Owner: 9, Group: 19, X: 30, Y: 30, HP: 100, MaxHP: 100},
		{ID: 2, Owner: 9, Group: 19, X: 31, Y: 30, HP: 100, MaxHP: 100},
		{ID: 3, Owner: SelfSlot, X: 75, Y: 75, HP: 100, MaxHP: 100},
	})
	if err != nil {
		t.Fatal(err)
	}
	if g := regGroup(t, w, 9, 19); g.AI[0x48] != 0 {
		t.Fatal("the fixture's Group is named by a script")
	}
	regGroup(t, w, 9, 19).AI[0x15] = 50
	w.cmdGroupOrder(ScriptInstant{Op: ScriptInstantGroupOrder, HasGroup: true, Group: 19, Args: [10]int32{17, 70, 60}})
	evaluated := func() bool {
		g := regGroup(t, w, 9, 19)
		return g.AI[0x15] != 50 || g.AI[10] != 30 || g.AI[11] != 30
	}
	for range 130 {
		Step(w, nil)
	}
	if g := regGroup(t, w, 9, 19); g.AI[0x45] != 0 || evaluated() {
		t.Fatalf("a Group with no party unit near decided: +0x45=%d", g.AI[0x45])
	}

	w.entities[2].X, w.entities[2].Y = 34, 40
	for range 130 {
		Step(w, nil)
	}
	if g := regGroup(t, w, 9, 19); g.AI[0x45] != 2 || !evaluated() {
		t.Fatalf("a Group with a party unit near did not decide: +0x45=%d", g.AI[0x45])
	}
}
