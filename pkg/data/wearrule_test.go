package data

import "testing"

// suitRow is a parameter array whose sutableFor cell holds v and whose other
// cells hold nothing a reader here looks at.
func suitRow(v int32) []int32 {
	p := make([]int32, SutableForColumn+1)
	p[SutableForColumn] = v
	return p
}

// TestTheWearRuleAnswersTheWholeTable is 0162 spec AC-1, SC-1: the four column
// values against the two class flags, written out one answer per row rather
// than derived, so a polarity swap fails half of them instead of none.
func TestTheWearRuleAnswersTheWholeTable(t *testing.T) {
	for _, c := range []struct {
		column int32
		mage   bool
		want   bool
	}{
		{0, false, false}, {0, true, false},
		{1, false, true}, {1, true, false},
		{2, false, false}, {2, true, true},
		{3, false, true}, {3, true, true},
	} {
		s, ok := SuitabilityFromParams(suitRow(c.column))
		if !ok {
			t.Fatalf("column %d: read refused", c.column)
		}
		if got := s.Allows(c.mage); got != c.want {
			t.Errorf("sutableFor %d, mage %v: allowed = %v, want %v", c.column, c.mage, got, c.want)
		}
	}
}

func TestASutableForCellIsBitTestedAsItStands(t *testing.T) {
	s, ok := SuitabilityFromParams(suitRow(-1))
	if !ok || !s.Fighter || !s.Mage {
		t.Fatalf("sutableFor -1: %+v ok=%v, want both bits set", s, ok)
	}
	// A value carrying bits above the two read is not an ordinal: only bits 0
	// and 1 are consulted.
	s, _ = SuitabilityFromParams(suitRow(4))
	if s.Fighter || s.Mage {
		t.Errorf("sutableFor 4: %+v, want neither bit", s)
	}
}

// TestARowTooShortIsUsableByNeitherAndAMissingRowIsUnknown is 0162 spec FR-2a
// and SC-3, the split the whole enforcement polarity rests on.
func TestARowTooShortIsUsableByNeitherAndAMissingRowIsUnknown(t *testing.T) {
	s, ok := SuitabilityFromParams([]int32{})
	if !ok {
		t.Fatalf("empty parameter array: read refused, want the zero suitability")
	}
	if s.Allows(false) || s.Allows(true) {
		t.Errorf("empty parameter array: %+v, want usable by neither", s)
	}
	if _, ok := SuitabilityFromParams(nil); ok {
		t.Errorf("nil parameter array: read accepted, want unknown")
	}
}

func TestSuitabilityFromCodeReachesEachOfTheThreeCollections(t *testing.T) {
	weapons := testCollection{{}, {name: "Staff", params: suitRow(2)}}
	shields := testCollection{{}, {name: "Buckler", params: suitRow(1)}}
	armors := testCollection{{}, {name: "x", params: suitRow(1)}, {name: "Robe", params: suitRow(2)}}

	for _, c := range []struct {
		what string
		code ItemCode
		want Suitability
	}{
		{"weapon", ComposeItemCode(0, weaponItemClass, 0, 1), Suitability{Mage: true}},
		{"shield", ComposeItemCode(0, shieldItemClass, 0, 1), Suitability{Fighter: true}},
		{"armour", ComposeItemCode(0, 5, 0, 2), Suitability{Mage: true}},
	} {
		got, ok := SuitabilityFromCode(c.code, weapons, shields, armors)
		if !ok {
			t.Fatalf("%s: read refused", c.what)
		}
		if got != c.want {
			t.Errorf("%s: %+v, want %+v", c.what, got, c.want)
		}
	}
}

// TestAnUnreachableRowIsUnknownAndUnknownPermits is 0162 spec FR-2a, plan R-2.
// A code this build cannot resolve must not become an equip refusal.
func TestAnUnreachableRowIsUnknownAndUnknownPermits(t *testing.T) {
	weapons := testCollection{{}, {name: "Staff", params: suitRow(2)}}
	for _, c := range []struct {
		what string
		code ItemCode
		coll Collection
	}{
		{"item class 14", ComposeItemCode(0, ItemClassCarried, 0, 1), weapons},
		{"row past the end", ComposeItemCode(0, weaponItemClass, 0, 9), weapons},
		{"nil collection", ComposeItemCode(0, weaponItemClass, 0, 1), nil},
	} {
		if _, ok := SuitabilityFromCode(c.code, c.coll, c.coll, c.coll); ok {
			t.Errorf("%s: read accepted, want unknown", c.what)
		}
		allowed, known := AllowsItem(c.code, true, c.coll, c.coll, c.coll)
		if known || !allowed {
			t.Errorf("%s: AllowsItem = (%v, %v), want (true, false)", c.what, allowed, known)
		}
	}
}
