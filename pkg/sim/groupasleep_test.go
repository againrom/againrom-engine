package sim

import "testing"

// asleepPursuitWorld is a Guard Group of one member, 6 cells from the local
// player's unit, inside its activity cover.
func asleepPursuitWorld(t *testing.T) *World {
	t.Helper()
	w := mustWorld(t, 7, Bounds{Width: 128, Height: 128}, []Entity{
		{ID: 1, Owner: SelfSlot, Group: 1, X: 20, Y: 20, HP: 100, MaxHP: 100, Speed: 20, ScanRange: 8},
		{ID: 2, Owner: 2, Group: 8, X: 26, Y: 20, HP: 100, MaxHP: 100, Speed: 20, ScanRange: 8, PostX: 26, PostY: 20},
	})
	w.relations = engRel(t, [3]uint32{1, 2, relationHostile}, [3]uint32{2, 1, relationHostile})
	g := SavedGroup{ID: 8, Members: []SavedGroupMember{{Entity: 2, Bound: true}}}
	g.AI[0x20], g.AI[0x45], g.AI[0x38] = orderGuard, 1, 8
	order := SavedActorOrder{Entity: 2}
	order.Raw[0], order.Raw[1] = 26, 20
	if err := w.ImportSavedGroups([]SavedGroup{g}, []SavedActorOrder{order}); err != nil {
		t.Fatal(err)
	}
	return w
}

func (w *World) asleepMember() Entity { return w.entities[indexOfEntity(w.entities, 2)] }

// A pursuer whose Group the activity rebuild leaves at zero runs no order at
// zero progress (AI-350, AI-364): when its victim is relocated out of the
// cover, the member keeps its order and victim (AI-750) and stands once its
// current step ends; it resumes when the victim is covered again.
func TestAnAsleepGroupsPursuerStandsWhenItsVictimLeavesTheCover(t *testing.T) {
	w := asleepPursuitWorld(t)
	for i := 0; i < 64 && !w.asleepMember().HasAttackTarget; i++ {
		Step(w, nil)
	}
	if m := w.asleepMember(); !m.HasAttackTarget || m.AttackTarget != 1 {
		t.Fatalf("the Guard member did not take the player's unit: %+v", m.AttackTarget)
	}
	if err := w.HeadlessPlace(1, 100, 100); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 32; i++ {
		Step(w, nil)
	}
	if w.savedGroups.Groups[0].AI[0x45] != 0 {
		t.Fatal("the activity rebuild still counts the uncovered member")
	}
	m := w.asleepMember()
	for i := 0; i < 400; i++ {
		Step(w, nil)
	}
	after := w.asleepMember()
	if d := cellOf(&m).chebyshevTo(cellOf(&after)); d > 1 || after.Transit != 0 {
		t.Fatalf("the member walked %d cells after its Group fell asleep: %d,%d to %d,%d", d, m.X, m.Y, after.X, after.Y)
	}
	if !after.HasAttackTarget || after.AttackTarget != 1 {
		t.Fatal("the asleep member lost its victim")
	}
	if err := w.HeadlessPlace(1, after.X+6, after.Y); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 64; i++ {
		Step(w, nil)
	}
	if back := w.asleepMember(); cellOf(&back).chebyshevTo(cellOf(&after)) == 0 {
		t.Fatal("the member did not resume once its Group was covered again")
	}
}
