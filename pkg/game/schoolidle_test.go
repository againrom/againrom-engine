package game

import (
	"testing"
	"time"
)

// TestSchoolShineStepsEveryHalfSecondModuloFive is TOWN-500's idle clock: the
// slot index steps once, modulo five, on a paint at least 500 ms after the
// last step, and a late paint never steps twice.
func TestSchoolShineStepsEveryHalfSecondModuloFive(t *testing.T) {
	s := &townScreen{}
	at := time.Unix(1000, 0)
	s.advanceSchoolShine(at)
	steps := []struct {
		after time.Duration
		cycle int
	}{
		{499 * time.Millisecond, 0}, {500 * time.Millisecond, 1}, {1600 * time.Millisecond, 2},
		{2100 * time.Millisecond, 3}, {2600 * time.Millisecond, 4}, {3100 * time.Millisecond, 0},
	}
	for _, step := range steps {
		s.advanceSchoolShine(at.Add(step.after))
		if got := s.schoolTrainingStatic.shineCycle; got != step.cycle {
			t.Fatalf("at +%v the idle slot is %d, want %d", step.after, got, step.cycle)
		}
	}
}

// TestSchoolViewCarriesTheIdleSlotInTheSharedOrder: the view names the slot the
// idle clock stands on in the shared slot order, which swaps the stored third
// and fourth slot, and carries none before the first school paint.
func TestSchoolViewCarriesTheIdleSlotInTheSharedOrder(t *testing.T) {
	f := shellFrontEnd()
	now := time.Unix(1000, 0)
	f.TownAnimationNow = func() time.Time { return now }
	s := f.townUI
	s.atSquare()
	s.Choose(2)
	if v := s.TownSurface(); v.SchoolIdleShine {
		t.Fatal("the idle shine showed before any school paint")
	}
	s.AdvanceTownSurfaceAnimation()
	want := []int{0, 1, 3, 2, 4, 0}
	for cycle, slot := range want {
		if cycle > 0 {
			now = now.Add(500 * time.Millisecond)
			s.AdvanceTownSurfaceAnimation()
		}
		if v := s.TownSurface(); !v.SchoolIdleShine || v.SchoolIdleSlot != slot {
			t.Fatalf("stored slot %d: view idle = %v slot %d, want slot %d", cycle, v.SchoolIdleShine, v.SchoolIdleSlot, slot)
		}
	}
}
