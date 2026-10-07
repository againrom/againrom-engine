package data_test

import (
	"testing"

	"againrom/pkg/data"
)

// The hero appearance law (AC-1..AC-6, plan SC-1..SC-4).

// The seventeen arms, transcribed here INDEPENDENTLY of the table under test:
// this list is the evidence and the map is the implementation, so a typo in
// either is a disagreement rather than a shared mistake.
var publishedArms = []struct {
	name data.HeroBody
	id   int32
}{
	{"unarmed", 1}, {"unarmed_", 2},
	{"swordsman", 3}, {"swordsman_", 4}, {"swordsman2h", 5},
	{"axeman", 7}, {"axeman_", 8}, {"axeman2h", 9},
	{"clubman", 10}, {"clubman_", 11},
	{"pikeman", 12}, {"pikeman_", 13},
	{"archer", 14}, {"bowman", 14}, {"xbowman", 15},
	{"mage_st", 24}, {"mage", 23},
}

func TestEveryPublishedNameAnswersItsClass(t *testing.T) {
	if len(publishedArms) != 17 {
		t.Fatalf("the chain has 17 arms, this list has %d", len(publishedArms))
	}
	distinct := make(map[int32]bool)
	for _, a := range publishedArms {
		id, ok := data.HeroBodyClass(a.name)
		if !ok {
			t.Errorf("HeroBodyClass(%q): no arm matched, want one", a.name)
			continue
		}
		if id != a.id {
			t.Errorf("HeroBodyClass(%q) = %d, want %d", a.name, id, a.id)
		}
		distinct[id] = true
	}
	// Seventeen names, SIXTEEN keys: archer and bowman share one and nothing
	// else does. The published summary says fifteen; the seventeen arms it
	// enumerates, and the raw-byte reproduction of them, give sixteen. This
	// assertion follows the enumeration.
	if len(distinct) != 16 {
		t.Errorf("the arms give %d distinct class keys, want 16", len(distinct))
	}
}

func TestAnUnmatchedNameFallsBackOnTheForcedClass(t *testing.T) {
	// The last two are the point: appending the shield suffix is unconditional
	// in the original, so it can compose a name no arm carries, and what the
	// character is then drawn as is the fallback rather than an error.
	for _, name := range []data.HeroBody{"", "swordsman3h", "Swordsman", "unarmed__",
		"swordsman2h_", "archer_"} {
		id, ok := data.HeroBodyClass(name)
		if ok {
			t.Errorf("HeroBodyClass(%q): an arm matched, want none", name)
		}
		if id != data.HeroUnmatchedClass {
			t.Errorf("HeroBodyClass(%q) = %d, want the forced %d",
				name, id, data.HeroUnmatchedClass)
		}
	}
	if data.HeroUnmatchedClass != 1 {
		t.Errorf("the forced class is %d, want 1", data.HeroUnmatchedClass)
	}
}

func TestTheNameIsComposedFromWhatIsWorn(t *testing.T) {
	for _, tc := range []struct {
		base                data.HeroBody
		shield, mage, dying bool
		want                data.HeroBody
		why                 string
	}{
		{base: "swordsman", want: "swordsman", why: "a living fighter with nothing else keeps his name"},
		{base: "swordsman", shield: true, want: "swordsman_", why: "an occupied second slot appends the suffix"},
		{base: "unarmed", want: "unarmed", why: "the empty hand is untouched for a fighter"},
		{base: "unarmed", mage: true, want: "mage", why: "a mage whose name came out unarmed is substituted"},
		{base: "swordsman", mage: true, want: "swordsman", why: "the mage substitution fires only on unarmed"},
		{base: "swordsman", dying: true, want: "unarmed", why: "a dying fighter is forced into the unarmed body"},
		{base: "swordsman", shield: true, dying: true, want: "unarmed",
			why: "the dying substitution replaces the whole composed name, shield included"},
		{base: "swordsman", mage: true, dying: true, want: "mage_st", why: "a dying mage takes the staff body"},
		{base: "unarmed", mage: true, dying: true, want: "mage_st", why: "and does so from the empty hand too"},
	} {
		got := data.HeroBodyName(tc.base, tc.shield, tc.mage, tc.dying)
		if got != tc.want {
			t.Errorf("HeroBodyName(%q, shield=%v, mage=%v, dying=%v) = %q, want %q — %s",
				tc.base, tc.shield, tc.mage, tc.dying, got, tc.want, tc.why)
		}
	}
}

func TestTheDirectoryIsTheThreeWay(t *testing.T) {
	// The sixteen blocks, transcribed independently of the table under test.
	wantMaterial := [16]string{
		"heroes", "heroes", "heroes", "heroes", "heroes", "heroes", "heroes", "heroes",
		"heroes_l", "heroes_l", "heroes_l", "heroes_l", "heroes_l", "heroes_l",
		"heroes", "heroes_l",
	}
	for m, want := range wantMaterial {
		got, ok := data.HeroMaterialDir(m)
		if !ok || got != want {
			t.Errorf("HeroMaterialDir(%d) = %q, %v; want %q, true", m, got, ok, want)
		}
		// The armour arm of the three-way is that same value.
		if d, ok := data.HeroBodyDir(false, m, true); !ok || d != want {
			t.Errorf("HeroBodyDir(fighter, %d, armoured) = %q, %v; want %q, true", m, d, ok, want)
		}
		// And BOTH special arms answer their own regardless of the material.
		if d, ok := data.HeroBodyDir(true, m, true); !ok || d != "heroes" {
			t.Errorf("HeroBodyDir(mage, %d, armoured) = %q, %v; want heroes, true", m, d, ok)
		}
		if d, ok := data.HeroBodyDir(false, m, false); !ok || d != "heroes_l" {
			t.Errorf("HeroBodyDir(fighter, %d, bare) = %q, %v; want heroes_l, true", m, d, ok)
		}
	}
	for _, m := range []int{-1, 16, 4096} {
		if d, ok := data.HeroMaterialDir(m); ok {
			t.Errorf("HeroMaterialDir(%d) = %q, true; want the out-of-range refusal", m, d)
		}
		if d, ok := data.HeroBodyDir(false, m, true); ok {
			t.Errorf("HeroBodyDir(fighter, %d, armoured) = %q, true; want the refusal", m, d)
		}
	}
}

func TestTheSheetAddressIsComposed(t *testing.T) {
	const dir, body = "heroes_l", data.HeroBody("swordsman")
	if got, want := data.HeroBodyBase(dir, body), "units/heroes_l/swordsman/sprites"; got != want {
		t.Errorf("HeroBodyBase = %q, want %q", got, want)
	}
	if got, want := data.HeroSheetPath(dir, body), "units/heroes_l/swordsman/sprites.256"; got != want {
		t.Errorf("HeroSheetPath = %q, want %q", got, want)
	}
	// The sibling differs by ONE insertion and nothing else.
	if got, want := data.HeroOverlayPath(dir, body), "units/heroes_l/swordsman/spritesb.256"; got != want {
		t.Errorf("HeroOverlayPath = %q, want %q", got, want)
	}
	// An empty argument at either end addresses nothing rather than a bare
	// extension — the answer a class resolving no File already gives.
	for _, tc := range []struct {
		dir  string
		body data.HeroBody
	}{{"", body}, {dir, ""}} {
		if got := data.HeroSheetPath(tc.dir, tc.body); got != "" {
			t.Errorf("HeroSheetPath(%q, %q) = %q, want the empty address", tc.dir, tc.body, got)
		}
		if got := data.HeroOverlayPath(tc.dir, tc.body); got != "" {
			t.Errorf("HeroOverlayPath(%q, %q) = %q, want the empty address", tc.dir, tc.body, got)
		}
	}
}

// 0134's own tests, below: the equipment slot that feeds the directory arm,
// the bundle key, and the one composed derivation (AC-6..AC-9, AC-16, plan
// D-1, D-2, D-3).

// itemCode packs the four fields the same way itemcode.go's own bit layout
// does -- A bits 15..12, B 11..8, C 7..5, D 4..0 -- so a test can name an
// equipment slot's code by the fields that matter to it without importing
// anything beyond what ItemCode already exposes.
func itemCode(a, b, c, d int) data.ItemCode {
	return data.ItemCode(uint16(a)<<12 | uint16(b)<<8 | uint16(c)<<5 | uint16(d))
}

func TestHeroArmourFactsReadsSlotEight(t *testing.T) {
	if data.HeroArmourSlot != 8 {
		t.Fatalf("HeroArmourSlot = %d, want 8", data.HeroArmourSlot)
	}
	var e data.Equipment
	// An empty slot 8: field D zero, whatever field A carries.
	e.SetCode(data.HeroArmourSlot, itemCode(5, 0, 0, 0))
	if material, armoured := data.HeroArmourFacts(e); armoured || material != 5 {
		t.Errorf("HeroArmourFacts(empty slot 8, material 5) = %d, %v; want 5, false",
			material, armoured)
	}
	// An occupied slot 8 names its own material.
	for m := 0; m < data.HeroMaterials; m++ {
		e = data.Equipment{}
		e.SetCode(data.HeroArmourSlot, itemCode(m, 0, 0, 1))
		material, armoured := data.HeroArmourFacts(e)
		if !armoured || material != m {
			t.Errorf("HeroArmourFacts(occupied slot 8, material %d) = %d, %v; want %d, true",
				m, material, armoured, m)
		}
	}
	// A slot other than 8 does not move the answer.
	var other data.Equipment
	other.SetCode(1, itemCode(9, 0, 0, 3))
	if material, armoured := data.HeroArmourFacts(other); armoured || material != 0 {
		t.Errorf("HeroArmourFacts(slot 8 untouched) = %d, %v; want 0, false", material, armoured)
	}
}

func TestHeroBodyKeyJoinsDirectoryAndName(t *testing.T) {
	if got, want := data.HeroBodyKey("heroes_l", "swordsman"), "heroes_l/swordsman"; got != want {
		t.Errorf("HeroBodyKey = %q, want %q", got, want)
	}
	k1 := data.HeroBodyKey("heroes", "mage")
	k2 := data.HeroBodyKey("heroes_l", "mage")
	if k1 == k2 {
		t.Errorf("HeroBodyKey(heroes, mage) = HeroBodyKey(heroes_l, mage) = %q, want distinct keys", k1)
	}
	// An empty directory or an empty name answers "", HeroBodyBase's own
	// rule for the same two inputs.
	for _, tc := range []struct {
		dir  string
		body data.HeroBody
	}{{"", "mage"}, {"heroes", ""}, {"", ""}} {
		if got := data.HeroBodyKey(tc.dir, tc.body); got != "" {
			t.Errorf("HeroBodyKey(%q, %q) = %q, want the empty key", tc.dir, tc.body, got)
		}
	}
}

func TestHeroAppearanceComposesTheName(t *testing.T) {
	// A synthetic list: row 1 (index 0) is the bare hand, row 2 (index 1) a
	// swordsman. HeroBodyFor reads slot 1's field D as the one-based row.
	l := data.BodyList{"unarmed", "swordsman", "archer"}

	// AC-6: an occupied slot 2 appends the suffix.
	var e data.Equipment
	e.SetCode(1, itemCode(0, 0, 0, 2)) // slot 1: row 2 -> "swordsman"
	e.SetCode(2, itemCode(0, 0, 0, 1)) // slot 2: occupied
	name, _, _, ok := data.HeroAppearance(l, e, false, false)
	if !ok || name != "swordsman_" {
		t.Errorf("HeroAppearance(shield) name = %q, ok=%v; want \"swordsman_\", true", name, ok)
	}

	// AC-7: the bare-handed name, for a mage, substitutes to the mage body.
	var bare data.Equipment // slot 1 empty: field D zero -> row 1 -> "unarmed"
	name, _, _, ok = data.HeroAppearance(l, bare, true, false)
	if !ok || name != data.BodyMage {
		t.Errorf("HeroAppearance(mage, bare) name = %q, ok=%v; want %q, true", name, ok, data.BodyMage)
	}
	// The same equipment, not a mage, keeps the bare name -- the mage
	// substitution does not fire uninvited.
	name, _, _, ok = data.HeroAppearance(l, bare, false, false)
	if !ok || name != data.BodyUnarmed {
		t.Errorf("HeroAppearance(fighter, bare) name = %q, ok=%v; want %q, true", name, ok, data.BodyUnarmed)
	}
}

func TestHeroAppearanceComposesTheDirectory(t *testing.T) {
	l := data.BodyList{"unarmed"}

	// AC-8: slot 8's material, for a non-mage, names that material's own
	// directory -- every one of the sixteen.
	wantMaterial := [16]string{
		"heroes", "heroes", "heroes", "heroes", "heroes", "heroes", "heroes", "heroes",
		"heroes_l", "heroes_l", "heroes_l", "heroes_l", "heroes_l", "heroes_l",
		"heroes", "heroes_l",
	}
	for m, want := range wantMaterial {
		var e data.Equipment
		e.SetCode(data.HeroArmourSlot, itemCode(m, 0, 0, 1)) // occupied, material m
		_, dir, _, ok := data.HeroAppearance(l, e, false, false)
		if !ok || dir != want {
			t.Errorf("HeroAppearance(non-mage, material %d) dir = %q, ok=%v; want %q, true",
				m, dir, ok, want)
		}
	}

	// AC-16: a mage takes the mage directory whatever slot 8 holds, occupied
	// or not -- and separately, a non-mage whose slot 8 is empty takes the
	// unarmoured directory.
	var occupied data.Equipment
	occupied.SetCode(data.HeroArmourSlot, itemCode(9, 0, 0, 1)) // an heroes_l material
	if _, dir, _, ok := data.HeroAppearance(l, occupied, true, false); !ok || dir != "heroes" {
		t.Errorf("HeroAppearance(mage, occupied slot 8) dir = %q, ok=%v; want heroes, true", dir, ok)
	}
	var empty data.Equipment
	if _, dir, _, ok := data.HeroAppearance(l, empty, true, false); !ok || dir != "heroes" {
		t.Errorf("HeroAppearance(mage, empty slot 8) dir = %q, ok=%v; want heroes, true", dir, ok)
	}
	if _, dir, _, ok := data.HeroAppearance(l, empty, false, false); !ok || dir != "heroes_l" {
		t.Errorf("HeroAppearance(non-mage, empty slot 8) dir = %q, ok=%v; want heroes_l, true", dir, ok)
	}
}

func TestHeroAppearanceIgnoresSex(t *testing.T) {
	// AC-9: nothing about sex enters this law's signature at all, so two
	// characters differing only in sex -- which this derivation never reads
	// -- derive the identical name, directory and class from the same
	// equipment. The absence of a sex parameter IS the property; this test
	// exercises it by deriving twice off one equipment set and requiring the
	// three answers to agree, exactly as two sexes sharing that equipment
	// would.
	l := data.BodyList{"unarmed", "swordsman"}
	var e data.Equipment
	e.SetCode(1, itemCode(0, 0, 0, 2))
	e.SetCode(data.HeroArmourSlot, itemCode(3, 0, 0, 1))

	name1, dir1, class1, ok1 := data.HeroAppearance(l, e, false, false)
	name2, dir2, class2, ok2 := data.HeroAppearance(l, e, false, false)
	if name1 != name2 || dir1 != dir2 || class1 != class2 || ok1 != ok2 {
		t.Errorf("HeroAppearance disagreed across two calls with identical equipment: "+
			"(%q, %q, %d, %v) vs (%q, %q, %d, %v)",
			name1, dir1, class1, ok1, name2, dir2, class2, ok2)
	}
}

func TestHeroAppearanceIsTotalAndTheClassMatchesTheName(t *testing.T) {
	var empty data.Equipment
	var noList data.BodyList
	for _, mage := range []bool{false, true} {
		for _, dying := range []bool{false, true} {
			name, dir, class, ok := data.HeroAppearance(noList, empty, mage, dying)
			// Total means it answers something and does not panic; an empty
			// list is exactly the case HeroBodyFor's own refusal covers, so
			// ok is false here -- the law's own fallback, not a defect.
			if ok {
				t.Errorf("HeroAppearance(mage=%v, dying=%v, empty list) ok=true, want false (empty list refuses)",
					mage, dying)
			}
			wantClass, _ := data.HeroBodyClass(name)
			if class != wantClass {
				t.Errorf("HeroAppearance(mage=%v, dying=%v) class = %d, want HeroBodyClass(%q) = %d",
					mage, dying, class, name, wantClass)
			}
			// The directory is still answered -- HeroBodyDir is total over
			// an empty (unoccupied) slot 8 regardless of the empty list.
			if dir == "" {
				t.Errorf("HeroAppearance(mage=%v, dying=%v, empty list) dir = \"\", want a real directory",
					mage, dying)
			}
		}
	}

	l := data.BodyList{"unarmed", "swordsman", "axeman", "archer"}
	for _, tc := range []struct {
		row         int
		shield      bool
		mage, dying bool
	}{
		{row: 1, shield: false, mage: false, dying: false},
		{row: 1, shield: false, mage: true, dying: false},
		{row: 2, shield: true, mage: false, dying: false},
		{row: 3, shield: false, mage: false, dying: true},
		{row: 4, shield: true, mage: true, dying: true},
	} {
		var e data.Equipment
		e.SetCode(1, itemCode(0, 0, 0, tc.row))
		if tc.shield {
			e.SetCode(2, itemCode(0, 0, 0, 1))
		}
		name, _, class, _ := data.HeroAppearance(l, e, tc.mage, tc.dying)
		wantClass, _ := data.HeroBodyClass(name)
		if class != wantClass {
			t.Errorf("HeroAppearance(row=%d shield=%v mage=%v dying=%v) class = %d, want %d for name %q",
				tc.row, tc.shield, tc.mage, tc.dying, class, wantClass, name)
		}
	}
}
