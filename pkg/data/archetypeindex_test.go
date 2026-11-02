package data

import "testing"

// Each table indexed by a hero's class and sex reads the entry it read before
// the index came from one place: archetype order (female +1, mage +2) for the
// base names and traits, picture order (mage +1, female +2) for the faces.
func TestArchetypeIndexedTablesKeepTheirEntries(t *testing.T) {
	for _, c := range []struct {
		mage, female bool
		archetype    int
		picture      int
	}{
		{false, false, 0, 0}, {false, true, 1, 2}, {true, false, 2, 1}, {true, true, 3, 3},
	} {
		if got := archetypeSlot(c.mage, c.female); got != c.archetype {
			t.Errorf("archetypeSlot(%v, %v) = %d, want %d", c.mage, c.female, got, c.archetype)
		}
		if got := (HeroTraits{Mage: c.mage, Female: c.female}).archetype(); got != c.archetype {
			t.Errorf("HeroTraits(%v, %v).archetype() = %d, want %d", c.mage, c.female, got, c.archetype)
		}
		v := NPCFace{Archetype: true, ArchetypeFaces: [4]int{10, 11, 12, 13}}
		if c.mage {
			v.Tokens |= NPCTokens(NPCTokenMage)
		}
		if c.female {
			v.Tokens |= NPCTokens(NPCTokenFemale)
		}
		_, face, _, ok := v.SynthesisedFigure("", false)
		if !ok || face != 10+c.picture {
			t.Errorf("SynthesisedFigure(mage %v, female %v) face = %d, want %d", c.mage, c.female, face, 10+c.picture)
		}
	}
}
