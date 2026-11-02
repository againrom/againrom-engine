package sim

import "testing"

// The persisted type id of each hero kind is the base id plus the kind's
// position in archetype order; the four values are written to saves.
func TestHeroTypeIDKeepsItsFourPersistedValues(t *testing.T) {
	for _, c := range []struct {
		mage, female bool
		add          int32
	}{
		{false, false, 0}, {false, true, 1}, {true, false, 2}, {true, true, 3},
	} {
		if got := HeroTypeID(c.mage, c.female); got != HumanTypeID+c.add {
			t.Errorf("HeroTypeID(%v, %v) = %d, want %d", c.mage, c.female, got, HumanTypeID+c.add)
		}
	}
}
