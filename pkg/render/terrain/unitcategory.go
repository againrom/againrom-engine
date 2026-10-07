package terrain

// UnitCategory separates registration populations sharing one sprite format.
// ANIM-CATEGORY-084 joins ordinary/alternate CUnit and CAirUnit to selectors
// 2/4/3. Zero is an ordinary unit, including artless native fixtures.
type UnitCategory uint8

const (
	UnitOrdinary UnitCategory = iota
	UnitAlternate
	UnitAir
)

// UnitCategoryFor reads the placed class before any art substitution.
// REG-UNITS-061 and ANIM-CATEGORY-084: nonzero Z selects CAirUnit regardless
// of corpse stage; CUnit stages >=2 use the alternate registration.
// Native state has no equivalent of drawable client flag +0x18c & 0x80.
// This function does not substitute an unrelated simulation flag for it.
func UnitCategoryFor(class *UnitClass, corpseStage uint8) UnitCategory {
	if class != nil && class.Z != 0 {
		return UnitAir
	}
	if corpseStage >= 2 {
		return UnitAlternate
	}
	return UnitOrdinary
}
