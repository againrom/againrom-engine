package sim

import "testing"

// TestAFreshRegistryHoldsOnlyTheConstructorsAIBytes checks the AI bytes a
// fresh World's registry gives each owner: the controlled Player's Stand
// Ground stores +0x20 = 3 and leaves the notice base +0x38, roam counter and
// commanded cell at the constructor's zero (AI-449, AI-451, SAV-1219); owner 0
// holds no stance. An older engine's state keeps its native Group records' AI
// bytes: a byte form without the session clock and without a registry decodes
// into a registry built from them, and that registry keeps its mark through
// the byte form. The loss control is that native registry itself, whose
// controlled Group carries the native notice base.
func TestAFreshRegistryHoldsOnlyTheConstructorsAIBytes(t *testing.T) {
	w, err := NewWorld(17, Bounds{Width: 40, Height: 40}, ModeCanonical, nil, []Entity{
		{ID: 1, Owner: 9, Group: 19, X: 10, Y: 10, HP: 100, MaxHP: 100, Reach: 1, ScanRange: 3},
		{ID: 2, Owner: SelfSlot, X: 30, Y: 30, HP: 100, MaxHP: 100, ScanRange: 5},
		{ID: 3, Owner: 0, Group: 5, X: 20, Y: 20, HP: 100, MaxHP: 100},
	})
	if err != nil {
		t.Fatal(err)
	}
	if base := w.nativeGroupRecord(SelfSlot, 0); base == nil || base.base == 0 {
		t.Fatal("the fixture's controlled Group has no native notice base to copy")
	}
	self, guard, none := regGroup(t, w, SelfSlot, 0), regGroup(t, w, 9, 19), regGroup(t, w, 0, 5)
	if self.AI[0x20] != orderStandGround || self.AI[0x38] != 0 || self.AI[0x15] != 0 || self.AI[0x0a] != 0 || self.AI[0x0b] != 0 {
		t.Fatalf("controlled Group +0x20=%d +0x38=%d +0x15=%d +0x0a=%d +0x0b=%d, want 3, 0, 0, 0, 0",
			self.AI[0x20], self.AI[0x38], self.AI[0x15], self.AI[0x0a], self.AI[0x0b])
	}
	if guard.AI[0x20] != orderGuard || guard.AI[0x38] != guard.AI[0x2d] {
		t.Fatalf("Guard Group +0x20=%d +0x38=%d, want %d and +0x2d=%d", guard.AI[0x20], guard.AI[0x38], orderGuard, guard.AI[0x2d])
	}
	if none.AI[0x20] != 0 || none.AI[0x38] != 0 {
		t.Fatalf("owner-0 Group +0x20=%d +0x38=%d, want 0, 0", none.AI[0x20], none.AI[0x38])
	}
	if w.GroupsNative() {
		t.Fatal("a fresh World reports a native registry")
	}

	// An older engine's byte form: no session clock and no registry.
	w.hasSessionClock, w.fullTick = false, 0
	held := w.savedGroups
	w.savedGroups = nil
	older, err := w.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	w.savedGroups, w.hasSessionClock = held, true
	var cold World
	if err := cold.UnmarshalBinary(older); err != nil {
		t.Fatal(err)
	}
	if !cold.GroupsNative() {
		t.Fatal("an older byte form did not build its registry from the native Group records")
	}
	native := regGroup(t, &cold, SelfSlot, 0)
	if want := cold.nativeGroupRecord(SelfSlot, 0).base; native.AI[0x38] != want || want == 0 {
		t.Fatalf("older form's controlled Group +0x38=%d, want the native notice base %d", native.AI[0x38], want)
	}
	if cold.nativeGroupRecord(0, 5) != nil || regGroup(t, &cold, 0, 5).AI[0x20] != 0 {
		t.Fatal("owner 0 holds a native Group record or a stance")
	}

	again, err := cold.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	var next World
	if err := next.UnmarshalBinary(again); err != nil || !next.GroupsNative() || next.Hash() != cold.Hash() {
		t.Fatal("a native registry lost its mark through the byte form", err)
	}
}
