package data

import "testing"

// TestChargenBaseNamesAreTheFourInPublishedOrder pins the order literally:
// archetypeSlot and ChargenBase's fallback both depend on it being exactly
// this sequence and not merely this set.
func TestChargenBaseNamesAreTheFourInPublishedOrder(t *testing.T) {
	want := []string{"PC_Danath", "PC_Naira", "PC_Fergard", "PC_Reniesta"}
	got := ChargenBaseNames()
	if len(got) != len(want) {
		t.Fatalf("ChargenBaseNames() = %v, want %v", got, want)
	}
	for i, name := range want {
		if got[i] != name {
			t.Errorf("ChargenBaseNames()[%d] = %q, want %q", i, got[i], name)
		}
	}
}

// TestFindHumanByNameWalksAscendingFirstMatch is the fourth search beside
// the three defsearch_test.go already covers: ascending from index 1, first
// match wins, an empty-named row is skipped, and a nil collection or a
// missing key answers NotFound.
func TestFindHumanByNameWalksAscendingFirstMatch(t *testing.T) {
	c := testCollection{
		{},                                 // 0: the reserved entry
		{name: "", params: humanRow(1, 1)}, // 1: written but nameless
		{name: "PC_Danath", params: humanRow(0x21, 1)}, // 2
		{name: "PC_Naira", params: humanRow(0x22, 2)},  // 3
		{name: "PC_Danath", params: humanRow(0x23, 3)}, // 4: the duplicate name that must lose
	}
	if got := FindHumanByName(c, "PC_Danath"); got != 2 {
		t.Errorf("FindHumanByName(PC_Danath) = %d, want 2 — the earlier of the two", got)
	}
	if got := FindHumanByName(c, "PC_Naira"); got != 3 {
		t.Errorf("FindHumanByName(PC_Naira) = %d, want 3", got)
	}
	if got := FindHumanByName(c, "PC_Fergard"); got != NotFound {
		t.Errorf("FindHumanByName(PC_Fergard) = %d, want NotFound — the name is not in the collection", got)
	}
	if got := FindHumanByName(c, ""); got != NotFound {
		t.Errorf("FindHumanByName(\"\") = %d, want NotFound — an unwritten row cannot match an empty key", got)
	}
	if got := FindHumanByName(nil, "PC_Danath"); got != NotFound {
		t.Errorf("FindHumanByName(nil, ...) = %d, want NotFound", got)
	}
}

// chargenRow is a Humans row carrying exactly the four columns this file's
// tests read: the type id, the health maximum, the mana maximum and the
// face. Every other column is the sentinel, so the constructor's defaults
// stand for it, which is fine because nothing here reads them. TypeID is
// carried only so a fixture can PROVE it plays no part in ChargenBase's own
// choice — the archetype comes off the slot, never off this column.
func chargenRow(typeID, healthMax, manaMax, face int32) []int32 {
	return row(map[int]int32{humanTypeIDSlot: typeID, 4: healthMax, 5: manaMax, 17: face})
}

// fourBaseRows is a synthetic Humans collection carrying the four
// ChargenBaseNames entries, each distinguishable by its own HealthMax —
// 10, 20, 30, 40 in ChargenBaseNames' own order — so a test can tell which
// row ChargenBase actually returned without reading anything
// archetype-shaped off it.
func fourBaseRows() testCollection {
	return testCollection{
		{}, // 0: the reserved entry
		{name: "PC_Danath", params: chargenRow(0, 10, 0, 0)},   // slot 0: class=false, female=false
		{name: "PC_Naira", params: chargenRow(0, 20, 0, 0)},    // slot 1: class=false, female=true
		{name: "PC_Fergard", params: chargenRow(0, 30, 0, 0)},  // slot 2: class=true, female=false
		{name: "PC_Reniesta", params: chargenRow(0, 40, 0, 0)}, // slot 3: class=true, female=true
	}
}

func TestChargenBaseReachesItsOwnSlot(t *testing.T) {
	c := fourBaseRows()
	for _, tc := range []struct {
		class, female bool
		wantHealthMax int32
		wantIndex     int
	}{
		{false, false, 10, 1}, // PC_Danath
		{false, true, 20, 2},  // PC_Naira
		{true, false, 30, 3},  // PC_Fergard
		{true, true, 40, 4},   // PC_Reniesta
	} {
		d, i, ok := ChargenBase(c, tc.class, tc.female)
		if !ok {
			t.Fatalf("ChargenBase(%v, %v) reported not found", tc.class, tc.female)
		}
		if d.HealthMax != tc.wantHealthMax {
			t.Errorf("ChargenBase(%v, %v).HealthMax = %d, want %d", tc.class, tc.female, d.HealthMax, tc.wantHealthMax)
		}
		if i != tc.wantIndex {
			t.Errorf("ChargenBase(%v, %v) index = %d, want %d — the archetype's own slot", tc.class, tc.female, i, tc.wantIndex)
		}
	}
}

func TestChargenBaseFallsBackToFirstResolvedWhenItsOwnSlotDoesNotResolve(t *testing.T) {
	c := testCollection{
		{},
		{name: "PC_Danath", params: chargenRow(0, 10, 0, 0)}, // slot 0
		{name: "PC_Naira", params: chargenRow(0, 20, 0, 0)},  // slot 1
		// PC_Fergard (slot 2) and PC_Reniesta (slot 3) are absent.
	}
	d, i, ok := ChargenBase(c, true, false) // wants slot 2, PC_Fergard — absent
	if !ok {
		t.Fatalf("ChargenBase reported not found, want the first resolved name")
	}
	if d.HealthMax != 10 {
		t.Errorf("ChargenBase fell back to HealthMax %d, want 10 — PC_Danath, the first name that resolves", d.HealthMax)
	}
	if i != 1 {
		t.Errorf("ChargenBase index = %d, want 1 — PC_Danath's own collection index, not the requested slot's", i)
	}
}

func TestChargenBaseSkipsARowThatFailsToParse(t *testing.T) {
	c := testCollection{
		{},
		{name: "PC_Danath", params: []int32{1, 2, 3}}, // far too short to parse
		{name: "PC_Naira", params: chargenRow(0, 20, 0, 0)},
	}
	d, i, ok := ChargenBase(c, false, false) // wants slot 0, PC_Danath — unparseable
	if !ok {
		t.Fatalf("ChargenBase reported not found; PC_Naira should have resolved")
	}
	if d.HealthMax != 20 {
		t.Errorf("ChargenBase returned HealthMax %d, want PC_Naira's 20 — PC_Danath's unparseable row must be skipped, not matched", d.HealthMax)
	}
	if i != 2 {
		t.Errorf("ChargenBase index = %d, want 2 — PC_Naira's own collection index, the row that actually resolved", i)
	}
}

func TestChargenBaseNoneResolvesIsNoBaseRow(t *testing.T) {
	for _, c := range []Collection{nil, testCollection{}, testCollection{{name: "Someone Else", params: chargenRow(0, 1, 1, 0)}}} {
		d, i, ok := ChargenBase(c, true, false)
		if ok {
			t.Errorf("ChargenBase(%v) reported found, want false", c)
		}
		if d != (HumanDef{}) {
			t.Errorf("ChargenBase(%v) = %+v, want the zero HumanDef", c, d)
		}
		if i != NotFound {
			t.Errorf("ChargenBase(%v) index = %d, want NotFound", c, i)
		}
	}
}

// TestChargenBaseResolvesEachSlotWhenTypeIDCarriesTheDrawnClassID is the
// regression this task exists for. A real install's four rows carry TypeID
// 3, 14, 24, 24 — the drawn class ids of the appearance name chain
// (swordsman, archer, mage_st, mage_st), never a value in 0x21..0x24 (see
// the file header) — and an earlier version of this file decomposed that
// column as the archetype pair, found nothing on any of the four, and fell
// back to the SAME row for every request. With the fix, the SLOT decides,
// not the column, so all four archetypes still reach their own row even
// though TypeID carries exactly these shipped values, including the
// duplicate 24 shared by the two mage rows.
func TestChargenBaseResolvesEachSlotWhenTypeIDCarriesTheDrawnClassID(t *testing.T) {
	c := testCollection{
		{},
		{name: "PC_Danath", params: chargenRow(3, 50, 0, 5)},     // swordsman
		{name: "PC_Naira", params: chargenRow(14, 20, 0, 1)},     // archer
		{name: "PC_Fergard", params: chargenRow(24, 30, 70, 3)},  // mage_st
		{name: "PC_Reniesta", params: chargenRow(24, 30, 70, 1)}, // mage_st, same TypeID as PC_Fergard
	}
	for _, tc := range []struct {
		class, female bool
		wantFace      int32
	}{
		{false, false, 5}, // PC_Danath
		{false, true, 1},  // PC_Naira
		{true, false, 3},  // PC_Fergard
		{true, true, 1},   // PC_Reniesta
	} {
		d, _, ok := ChargenBase(c, tc.class, tc.female)
		if !ok {
			t.Fatalf("ChargenBase(%v, %v) reported not found", tc.class, tc.female)
		}
		if d.Face != tc.wantFace {
			t.Errorf("ChargenBase(%v, %v).Face = %d, want %d", tc.class, tc.female, d.Face, tc.wantFace)
		}
	}
}

func TestHumanDefProfileFollowsTheManaColumn(t *testing.T) {
	for _, tc := range []struct {
		name               string
		healthMax, manaMax int32
		want               Profile
	}{
		{"no columns at all", 0, 0, Profile{Fighter: true, HealthColumn: false, ManaColumn: false}},
		{"health only, no mana", 50, 0, Profile{Fighter: true, HealthColumn: true, ManaColumn: false}},
		{"mana only, no health", 0, 30, Profile{Fighter: false, HealthColumn: false, ManaColumn: true}},
		{"both columns", 50, 30, Profile{Fighter: false, HealthColumn: true, ManaColumn: true}},
	} {
		d, err := NewHumanDef(tc.name, chargenRow(0, tc.healthMax, tc.manaMax, 0))
		if err != nil {
			t.Fatalf("%s: NewHumanDef: %v", tc.name, err)
		}
		if got := d.Profile(); got != tc.want {
			t.Errorf("%s: Profile() = %+v, want %+v", tc.name, got, tc.want)
		}
	}
}

// TestMageSkillNameIsTheTablesOtherHalf mirrors chargen_test.go's own test
// for SkillName: the five shared slots carry the mage half of the shipped
// column titles, and a slot outside 1..5 — including slot 0, which neither
// class's half names — names nothing.
func TestMageSkillNameIsTheTablesOtherHalf(t *testing.T) {
	for _, tc := range []struct {
		slot int32
		want string
	}{
		{SkillGeneral, ""},
		{SkillBlade, "Fire"},
		{SkillAxe, "Water"},
		{SkillBludgen, "Air"},
		{SkillPike, "Earth"},
		{SkillShoot, "Astral"},
		{SkillSlots, ""},
		{-1, ""},
	} {
		if got := MageSkillName(tc.slot); got != tc.want {
			t.Errorf("MageSkillName(%d) = %q, want %q", tc.slot, got, tc.want)
		}
	}
}

func TestSkillNamesReturnsTheFiveInSlotOrder(t *testing.T) {
	wantWarrior := []string{"Blade", "Axe", "Bludgen", "Pike", "Shooting"}
	if got := SkillNames(false); !stringSlicesEqual(got, wantWarrior) {
		t.Errorf("SkillNames(false) = %v, want %v", got, wantWarrior)
	}
	wantMage := []string{"Fire", "Water", "Air", "Earth", "Astral"}
	if got := SkillNames(true); !stringSlicesEqual(got, wantMage) {
		t.Errorf("SkillNames(true) = %v, want %v", got, wantMage)
	}
}

func stringSlicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
