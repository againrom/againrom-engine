package sim

import "testing"

func fireWallHotfixWorld(t *testing.T) *World {
	t.Helper()
	rule := SpellRule{ID: 3, Area: true, Distribution: distributionWall, Radius: 2,
		AreaDuration: 1, School: 1, MaxRange: 30}
	w, err := NewStockedSpelledWorld(0xf17e, Bounds{Width: 64, Height: 64}, ModeCanonical,
		Terrain{}, nil, nil, Relations{}, nil, nil, []SpellRule{rule})
	if err != nil {
		t.Fatal(err)
	}
	return w
}

func TestWallOfFireKeepsIndependentClocksAndOnlyNewestCellOwner(t *testing.T) {
	w := fireWallHotfixWorld(t)
	rule := w.spells[0]
	for i := 0; i < 8; i++ {
		if !w.landArea(rule, 0, 0, false, 20, 20, 24, 20, nil) {
			t.Fatalf("Wall of Fire %d was refused; want more than the ordinary six slots", i+1)
		}
	}
	if got := len(w.CellEffects()); got != 8 {
		t.Fatalf("one anchor retains %d walls, want all 8", got)
	}
	for i, e := range w.CellEffects() {
		want := 0
		if i == 7 {
			want = 10
		}
		if len(e.Cells) != want {
			t.Fatalf("wall %d retains %d cells, want only the newest owner to retain its 5x2 footprint", i, len(e.Cells))
		}
	}
	if got := w.FireWallCount(24, 20); got != 1 {
		t.Fatalf("shared cell tracks %d walls, want one", got)
	}

	form, err := w.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary with eight overlapping walls: %v", err)
	}
	back := &World{}
	if err := back.UnmarshalBinary(form); err != nil {
		t.Fatalf("UnmarshalWorld with eight overlapping walls: %v", err)
	}
	if got := back.FireWallCount(24, 20); got != 1 {
		t.Fatalf("save/load retained %d walls on the shared cell, want one", got)
	}
}

func TestOverlappingFireWallsExpireAndDecrementIndependently(t *testing.T) {
	w := fireWallHotfixWorld(t)
	short := w.spells[0]
	long := short
	long.AreaDuration = 2
	if !w.landArea(short, 0, 0, false, 20, 20, 24, 20, nil) ||
		!w.landArea(long, 0, 0, false, 20, 20, 24, 20, nil) {
		t.Fatal("the two overlapping walls did not both land")
	}
	if got := w.FireWallCount(24, 20); got != 1 {
		t.Fatalf("initial count is %d, want one", got)
	}
	for i := 0; i < 17; i++ {
		Step(w, nil)
	}
	if got := w.FireWallCount(24, 20); got != 1 {
		t.Fatalf("after the short wall expired the count is %d, want the long wall's 1", got)
	}
	for i := 0; i < 16; i++ {
		Step(w, nil)
	}
	if got := w.FireWallCount(24, 20); got != 0 {
		t.Fatalf("after both walls expired the count is %d, want 0", got)
	}
}

func TestExistingFireCoverageDoesNotRejectAnotherWallPlacement(t *testing.T) {
	w := fireWallHotfixWorld(t)
	rule := w.spells[0]
	if !w.landArea(rule, 0, 0, false, 20, 20, 24, 20, nil) {
		t.Fatal("first wall did not land")
	}
	// The second anchor is already inside the first wall's footprint. It must
	// still accept a new wall and retain both clocks while transferring shared cells.
	if !w.landArea(rule, 0, 0, false, 20, 20, 25, 20, nil) {
		t.Fatal("a cell already covered by fire rejected another wall")
	}
	if got := w.FireWallCount(25, 20); got != 1 {
		t.Fatalf("the already-burning cell tracks %d walls, want one", got)
	}
}
