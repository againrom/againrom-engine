package sim

import "testing"

func TestFillReturnedPoolsFillsALivingHeroOnly(t *testing.T) {
	w := mustWorld(t, 7, Bounds{Width: 8, Height: 8}, []Entity{
		{ID: 1, X: 2, Y: 2, HP: 60, MaxHP: 100, Mana: 20, MaxMana: 80, Speed: 1},
		{ID: 2, X: 4, Y: 4, HP: -12, MaxHP: 100, Mana: 5, MaxMana: 80, Speed: 1},
	})
	if !w.FillReturnedPools(1) || w.FillReturnedPools(2) || w.FillReturnedPools(9) {
		t.Fatal("the fill took a fallen or missing unit or refused a living one")
	}
	if e, _ := w.Entity(1); e.HP != 100 || e.Mana != 80 {
		t.Fatalf("living hero at %d/100 health %d/80 mana, want both full", e.HP, e.Mana)
	}
	if e, _ := w.Entity(2); e.HP != -12 || e.Mana != 5 {
		t.Fatalf("fallen unit changed to %d health %d mana", e.HP, e.Mana)
	}
}

func TestReturnedExperienceIsNineTenthsInThirtyTwoBits(t *testing.T) {
	for _, c := range []struct{ xp, want int32 }{
		{0, 0}, {1, 0}, {9, 8}, {10, 9}, {90, 81}, {1593, 1433}, {2400, 2160},
		{238609294, 214748364}, {238609295, -214748364},
	} {
		if got := returnedExperience(c.xp); got != c.want {
			t.Errorf("returnedExperience(%d) = %d, want %d", c.xp, got, c.want)
		}
	}
}
