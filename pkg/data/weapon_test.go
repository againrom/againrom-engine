package data_test

import (
	"strings"
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/data"
	"againrom/pkg/formats/databin"
)

// weaponRow builds a Weapons row out of the slots this contract reads, so a
// test states the four numbers it cares about and not fourteen cells.
//
// The cells are laid out by RUNTIME column, which is what the parameter array's
// index already is; every cell the contract does not read is the format's own
// empty one, which is also what proves those cells are not being read.
func weaponRow(name string, min, max, toHit, defence, kind, rng, charge, relax int32) synth.DataBinRow {
	p := []int32{-1, -1, -1, -1, -1, kind, min, max, toHit, defence, -1, rng, charge, relax, -1}
	return synth.DataBinRow{Name: name, Params: p}
}

// scaleRow builds a shape or material row carrying the three factors this
// contract reads at their own slots.
func scaleRow(name string, damage, toHit, defence float64) synth.DataBinRow {
	d := make([]float64, 9)
	d[4], d[5], d[6] = damage, toHit, defence
	return synth.DataBinRow{Name: name, Doubles: d}
}

// tables parses one synthetic definition table and hands back the three
// collections the resolver reads.
func tables(t *testing.T, shapes, materials, weapons []synth.DataBinRow) (data.ScaleTable, data.ScaleTable, data.Collection) {
	t.Helper()
	f, err := databin.Parse(synth.DataBin{
		Rows: [synth.DataBinCollections][]synth.DataBinRow{
			synth.DataBinShapes:    shapes,
			synth.DataBinMaterials: materials,
			synth.DataBinWeapons:   weapons,
		},
	}.Bytes())
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	return f.Collection(databin.Shapes), f.Collection(databin.Materials), f.Collection(databin.Weapons)
}

// shipped is the shape of the file the chargen literals are resolved against:
// two shapes, three materials (one of them two words), and three rows.
func shipped(t *testing.T) (data.ScaleTable, data.ScaleTable, data.Collection) {
	t.Helper()
	return tables(t,
		[]synth.DataBinRow{
			scaleRow("Common", 0.2, 1, 1),
			scaleRow("Uncommon", 0.4, 2, 1),
		},
		[]synth.DataBinRow{
			scaleRow("Iron", 1, 1, 1),
			scaleRow("Wood", 0.5, 1, 1),
			scaleRow("Magic Wood", 2, 1, 1),
		},
		[]synth.DataBinRow{
			weaponRow("Short Sword", 23, 40, 23, 0, 1, 1, 9, 5),
			weaponRow("Short Bow", 10, 14, 8, 0, 5, 6, 12, 6),
			weaponRow("Flame Thrower", 30, 60, 0, 0, 11, 8, 20, 10),
		})
}

// A NAME WITH NO SHAPE WORD resolves through index 0 of the shape table, which
// is what the item's own zeroed byte holds — so the shipped Iron Short Sword,
// whose leading word is a material, is scaled by 0.2 and not by 1.
//
// This case is the one the retracted ladder got wrong, and the numbers below are
// what separate the two readings: under a shape factor of 1 the same row would
// carry (23, 17).
func TestAWeaponWithNoShapeWordScalesThroughIndexZero(t *testing.T) {
	s, m, w := shipped(t)

	got, err := data.ResolveWeapon("Iron Short Sword", s, m, w)
	if err != nil {
		t.Fatalf("ResolveWeapon: %v", err)
	}
	// Damage through 0.2 x 1; to-hit through the ladder's NEXT slot, 1 x 1, so
	// the two are visibly not the same factor.
	//
	// Row is 1: the Weapons collection is one-based and "Short Sword" is the
	// fixture's first written row, right after the reserved entry 0.
	want := data.Weapon{
		Name: "Iron Short Sword", Row: 1, Code: 0x0101, DamageBase: 5, DamageSpread: 3,
		ToHit: 23, Defence: 0, AttackType: 1, ChargeTime: 9, RelaxTime: 5, Range: 1,
	}
	if got != want {
		t.Fatalf("ResolveWeapon = %+v\nwant %+v", got, want)
	}
}

// A leading shape word is consumed and its factor multiplies the material's.
func TestALeadingShapeWordIsConsumedAndItsFactorMultiplies(t *testing.T) {
	s, m, w := shipped(t)

	got, err := data.ResolveWeapon("Uncommon Iron Short Sword", s, m, w)
	if err != nil {
		t.Fatalf("ResolveWeapon: %v", err)
	}
	// 0.4 x 1: base = ftol(23*0.4+0.5) = 9, spread = ftol(40*0.4-9+0.5) = 7.
	if got.DamageBase != 9 || got.DamageSpread != 7 {
		t.Errorf("damage (%d, %d), want (9, 7)", got.DamageBase, got.DamageSpread)
	}
	// The to-hit factor is a DIFFERENT slot of the same ladder: 2 x 1.
	if got.ToHit != 46 {
		t.Errorf("toHit %d, want 46 — the to-hit slot, not the damage slot", got.ToHit)
	}
}

// A material of MORE THAN ONE WORD is matched whole, and beats its own one-word
// suffix, which is the case the split rule exists for.
func TestATwoWordMaterialBeatsItsOneWordSuffix(t *testing.T) {
	s, m, w := shipped(t)

	got, err := data.ResolveWeapon("Uncommon Magic Wood Short Bow", s, m, w)
	if err != nil {
		t.Fatalf("ResolveWeapon: %v", err)
	}
	// Magic Wood is 2, Wood is 0.5: 0.4 x 2 = 0.8 against 0.4 x 0.5 = 0.2, so
	// the two readings are 8 and 2 and cannot be confused.
	if got.DamageBase != 8 {
		t.Fatalf("base %d, want 8 — the two-word material, not its suffix", got.DamageBase)
	}
}

// THE SECOND NUMBER IS A SPREAD AND NOT A MAXIMUM. The fill subtracts the
// already-rounded base from the scaled maximum, so the pair a weapon carries is
// (base, spread) — the reading four independent instruments killed the rival of.
//
// Subtracting before or after the truncation is algebraically the same, because
// the base is an integer; what is NOT the same, and what this asserts, is
// whether the subtraction happens at all. Under the (min, max) reading the
// shipped row below carries (5, 8) rather than (5, 3).
func TestTheSecondNumberIsASpreadAndNotAMaximum(t *testing.T) {
	s, m, w := shipped(t)

	got, err := data.ResolveWeapon("Iron Short Sword", s, m, w)
	if err != nil {
		t.Fatalf("ResolveWeapon: %v", err)
	}
	if got.DamageSpread != 3 {
		t.Fatalf("spread %d, want 3 — 8 is the scaled maximum, i.e. the rival reading",
			got.DamageSpread)
	}
	// And the sheet band the two numbers compose is the row's own scaled range.
	if lo, hi := got.DamageBase, got.DamageBase+got.DamageSpread; lo != 5 || hi != 8 {
		t.Fatalf("band %d-%d, want 5-8", lo, hi)
	}
}

// THE ROUNDING IS HALF UP AND THE TRUNCATION IS TOWARD ZERO. 23 x 0.2 is 4.6:
// half up gives 5, and a plain truncation of the product gives 4.
func TestTheScaleRoundsHalfUp(t *testing.T) {
	s, m, w := shipped(t)

	got, err := data.ResolveWeapon("Iron Short Sword", s, m, w)
	if err != nil {
		t.Fatalf("ResolveWeapon: %v", err)
	}
	if got.DamageBase != 5 {
		t.Fatalf("base %d, want 5 — 4 is the product truncated without the addend",
			got.DamageBase)
	}
}

// An empty range cell reads as one; every other cell this contract does not
// read stays empty and is not read.
func TestAnEmptyRangeCellReadsAsOne(t *testing.T) {
	s, m, w := tables(t, nil, nil,
		[]synth.DataBinRow{weaponRow("Blade", 4, 6, 0, 0, 1, -1, 8, 4)})

	got, err := data.ResolveWeapon("Blade", s, m, w)
	if err != nil {
		t.Fatalf("ResolveWeapon: %v", err)
	}
	if got.Range != 1 {
		t.Fatalf("Range %d, want 1 for an empty cell", got.Range)
	}
}

// AC-1: a ranged row RESOLVES rather than being refused — scaled exactly
// as a melee row is — and the weapon it yields reports the row's own
// attack type, so a caller can tell the two kinds apart through it and
// through Ranged()/Melee().
func TestARangedRowResolvesAndReportsItsAttackType(t *testing.T) {
	s, m, w := shipped(t)

	got, err := data.ResolveWeapon("Flame Thrower", s, m, w)
	if err != nil {
		t.Fatalf("ResolveWeapon: %v — FR-1, a ranged row resolves", err)
	}
	if got.AttackType != 11 {
		t.Errorf("AttackType = %d, want 11 — the row's own", got.AttackType)
	}
	if !got.Ranged() {
		t.Errorf("Ranged() = false for attack type %d, want true", got.AttackType)
	}
	if got.Melee() {
		t.Errorf("Melee() = true for attack type %d, want false", got.AttackType)
	}
}

func TestWeaponFromCodeRoundTripsEveryNameTheResolverAnswers(t *testing.T) {
	s, m, w := shipped(t)

	for _, name := range []string{
		"Iron Short Sword",
		"Uncommon Iron Short Sword",
		"Uncommon Magic Wood Short Bow",
		"Short Bow",
		"Flame Thrower",
	} {
		t.Run(name, func(t *testing.T) {
			want, err := data.ResolveWeapon(name, s, m, w)
			if err != nil {
				t.Fatalf("ResolveWeapon(%q): %v", name, err)
			}
			got, err := data.WeaponFromCode(want.Code, s, m, w)
			if err != nil {
				t.Fatalf("WeaponFromCode(0x%04x): %v", uint16(want.Code), err)
			}
			// Name apart: WeaponFromCode recomposes a name from the code's own three
			// indices, which need not read back the exact literal ResolveWeapon was
			// given — an absent shape word and shape index 0's own entry name
			// resolve through the identical factor, not through the identical string.
			got.Name, want.Name = "", ""
			if got != want {
				t.Errorf("WeaponFromCode(ResolveWeapon(%q).Code) = %+v\nwant %+v", name, got, want)
			}
		})
	}
}

func TestWeaponFromCodeRefusesACodeWhoseClassIsNotAWeapon(t *testing.T) {
	s, m, w := shipped(t)

	code := data.ItemCode(uint16(data.ItemClassCarried) << 8)
	if _, err := data.WeaponFromCode(code, s, m, w); err == nil {
		t.Fatalf("WeaponFromCode(class=%d): accepted, want refused", data.ItemClassCarried)
	}
}

// A remainder naming no row is refused, and the refusal says which remainder —
// which is what makes a wrong split rule visible instead of silent.
func TestARemainderNamingNoRowIsRefused(t *testing.T) {
	s, m, w := shipped(t)

	if _, err := data.ResolveWeapon("Iron Halberd", s, m, w); err == nil {
		t.Fatal("ResolveWeapon accepted a name no row answers")
	} else if !strings.Contains(err.Error(), "Halberd") {
		t.Errorf("error %q does not name the remainder it looked for", err)
	}
}

// A row too short for the slots this contract reads is refused rather than
// indexed past.
func TestAShortRowIsRefused(t *testing.T) {
	s, m, w := tables(t, nil, nil,
		[]synth.DataBinRow{{Name: "Stub", Params: []int32{1, 2, 3}}})

	if _, err := data.ResolveWeapon("Stub", s, m, w); err == nil {
		t.Fatal("ResolveWeapon accepted a row shorter than the slots it reads")
	}
}

// A table entry found as a BARE SUBSTRING is taken, word boundary or not
// (FR-2a): `Wood` IS taken out of `Woodland`. The positional rebuild then
// drops len("Wood")+1 == 5 bytes from the FRONT of the subject regardless of
// where the occurrence actually sat — here it sat at offset 0, so the drop
// happens to be the same 5 bytes the occurrence itself covered — leaving
// "and Axe", which no row wears, so the resolution refuses rather than
// reading the row's own columns through index 0.
func TestATableEntryFoundAsABareSubstringIsTakenAndMangles(t *testing.T) {
	s, m, w := tables(t,
		nil,
		[]synth.DataBinRow{scaleRow("Bronze", 1, 1, 1), scaleRow("Wood", 0.5, 1, 1)},
		[]synth.DataBinRow{weaponRow("Woodland Axe", 10, 20, 0, 0, 2, 1, 8, 4)})

	_, err := data.ResolveWeapon("Woodland Axe", s, m, w)
	if err == nil {
		t.Fatal("ResolveWeapon accepted \"Woodland Axe\" — `Wood` should have been taken out of it as a bare substring")
	}
	// "and Axe" is what survives the positional rebuild: `Wood` is found at
	// offset 0, so s[:0] is empty and the second slice starts
	// len("Wood")+1 == 5 bytes into "Woodland Axe" — s[5:] — which is "and
	// Axe". The mangled residue names no row.
	if !strings.Contains(err.Error(), "and Axe") {
		t.Errorf("error %q does not name the mangled residue \"and Axe\"", err)
	}
}

// A name with neither word keeps the two decoded defaults distinct: shape 0,
// but material 15 (`None`). ITEM-PICT-048 establishes this asymmetry; using
// material 0 makes shipped unqualified clothing address art that does not exist.
func TestAnUnmatchedWordUsesShapeZeroAndMaterialNone(t *testing.T) {
	materials := make([]synth.DataBinRow, 16)
	materials[0] = scaleRow("Bronze", 8, 1, 1)
	materials[15] = scaleRow("None", 2, 1, 1)
	s, m, w := tables(t,
		[]synth.DataBinRow{scaleRow("Common", 0.25, 1, 1), scaleRow("Uncommon", 4, 1, 1)},
		materials,
		[]synth.DataBinRow{weaponRow("Axe", 10, 20, 0, 0, 2, 1, 8, 4)})

	got, err := data.ResolveWeapon("Axe", s, m, w)
	if err != nil {
		t.Fatalf("ResolveWeapon: %v", err)
	}
	// Shape 0 (0.25) x material 15 (2): base 5, spread 5.
	if got.DamageBase != 5 || got.DamageSpread != 5 {
		t.Fatalf("damage (%d, %d), want (5, 5) through shape 0 and material 15",
			got.DamageBase, got.DamageSpread)
	}
	if got.Code.A() != 15 || got.Code.C() != 0 {
		t.Fatalf("code fields material/shape = %d/%d, want 15/0", got.Code.A(), got.Code.C())
	}
}

func TestResolveWeaponFillsRowFromTheIndexItAlreadyFound(t *testing.T) {
	s, m, w := shipped(t)

	for _, tc := range []struct {
		name string
		want int32
	}{
		{"Iron Short Sword", 1}, // "Short Sword" is the fixture's first row
		{"Short Bow", 2},        // "Short Bow" is the fixture's second row
	} {
		got, err := data.ResolveWeapon(tc.name, s, m, w)
		if err != nil {
			t.Fatalf("ResolveWeapon(%q): %v", tc.name, err)
		}
		if got.Row != tc.want {
			t.Errorf("ResolveWeapon(%q).Row = %d, want %d", tc.name, got.Row, tc.want)
		}
	}
}

func TestResolveWeaponComposesTheCodeFromShapeMaterialAndRow(t *testing.T) {
	s, m, w := shipped(t)

	const weaponClass = 1 // field B: the equipment slot a weapon occupies.
	for _, tc := range []struct {
		name                 string
		shape, material, row int
	}{
		// "Iron Short Sword": no shape word matches, so shape falls to index
		// 0 — the item constructor's own zeroed byte, not a search result.
		{"Iron Short Sword", 0, 0, 1},
		// A leading shape word IS the index takePrefix returns for it.
		{"Uncommon Iron Short Sword", 1, 0, 1},
		// A two-word material is matched whole: "Magic Wood" is index 2, not
		// its one-word suffix "Wood" at index 1.
		{"Uncommon Magic Wood Short Bow", 1, 2, 2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := data.ResolveWeapon(tc.name, s, m, w)
			if err != nil {
				t.Fatalf("ResolveWeapon(%q): %v", tc.name, err)
			}
			want := data.ItemCode(tc.material<<12 | weaponClass<<8 | tc.shape<<5 | tc.row)
			if got.Code != want {
				t.Errorf("ResolveWeapon(%q).Code = 0x%04x, want 0x%04x (shape=%d material=%d row=%d)",
					tc.name, uint16(got.Code), uint16(want), tc.shape, tc.material, tc.row)
			}
		})
	}
}

// Melee is the predicate the fold branches on, and every weapon that resolves
// satisfies it.
func TestEveryResolvedWeaponIsMelee(t *testing.T) {
	s, m, w := shipped(t)
	for _, n := range []string{"Iron Short Sword", "Uncommon Magic Wood Short Bow"} {
		got, err := data.ResolveWeapon(n, s, m, w)
		if err != nil {
			t.Fatalf("ResolveWeapon(%q): %v", n, err)
		}
		if !got.Melee() {
			t.Errorf("%q resolved to attack type %d, which is not melee", n, got.AttackType)
		}
	}
}

func TestATrailingSuffixIsStrippedBeforeMatching(t *testing.T) {
	s, m, w := shipped(t)

	plain, err := data.ResolveWeapon("Iron Short Sword", s, m, w)
	if err != nil {
		t.Fatalf("ResolveWeapon(plain): %v", err)
	}
	suffixed, err := data.ResolveWeapon("Iron Short Sword {fire}", s, m, w)
	if err != nil {
		t.Fatalf("ResolveWeapon(suffixed): %v", err)
	}
	// Name carries the ORIGINAL literal regardless — that is Name's own
	// contract, not the strip's — so it is normalised out here and only the
	// resolved numbers, which the strip does govern, are compared.
	plain.Name, suffixed.Name = "", ""
	if suffixed != plain {
		t.Fatalf("ResolveWeapon with a {…} suffix = %+v, want the unsuffixed %+v", suffixed, plain)
	}

	rng, ok := data.WeaponRange("Short Bow {arrow}", s, m, w)
	if !ok {
		t.Fatal("WeaponRange refused a name carrying a suffix it should have stripped")
	}
	plainRng, ok := data.WeaponRange("Short Bow", s, m, w)
	if !ok {
		t.Fatal("WeaponRange refused the unsuffixed name")
	}
	if rng != plainRng {
		t.Fatalf("WeaponRange with a {…} suffix = %d, want the unsuffixed %d", rng, plainRng)
	}
}

func TestWeaponRangeReadsTheRangeColumnOrOneWhenEmpty(t *testing.T) {
	for _, tc := range []struct {
		name string
		cell int32
		want int32
	}{
		{"an empty cell", -1, 1},
		{"a stated range", 20, 20},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s, m, w := tables(t, nil, nil,
				[]synth.DataBinRow{weaponRow("Blade", 4, 6, 0, 0, 1, tc.cell, 8, 4)})

			got, ok := data.WeaponRange("Blade", s, m, w)
			if !ok {
				t.Fatal("WeaponRange refused a row it should answer")
			}
			if got != tc.want {
				t.Errorf("WeaponRange = %d, want %d", got, tc.want)
			}
		})
	}
}

// A row too short for the range slot is refused — reported through the
// boolean, not indexed past.
func TestWeaponRangeRefusesARowShorterThanItsSlot(t *testing.T) {
	s, m, w := tables(t, nil, nil,
		[]synth.DataBinRow{{Name: "Stub", Params: []int32{1, 2, 3, 4, 5}}})

	if got, ok := data.WeaponRange("Stub", s, m, w); ok {
		t.Fatalf("WeaponRange accepted a row shorter than the range slot, got %d", got)
	}
}

// A remainder naming no row is refused through the boolean.
func TestWeaponRangeRefusesANameMatchingNoRow(t *testing.T) {
	s, m, w := shipped(t)

	if got, ok := data.WeaponRange("Iron Halberd", s, m, w); ok {
		t.Fatalf("WeaponRange accepted a name no row answers, got %d", got)
	}
}

func TestARangedRowsRangeAgreesThroughBothEntryPoints(t *testing.T) {
	s, m, w := shipped(t)

	viaRange, ok := data.WeaponRange("Flame Thrower", s, m, w)
	if !ok {
		t.Fatal("WeaponRange refused a ranged row; FR-1 refuses no row on its attack type")
	}
	if viaRange != 8 {
		t.Errorf("WeaponRange = %d, want the row's own range 8", viaRange)
	}

	viaResolve, err := data.ResolveWeapon("Flame Thrower", s, m, w)
	if err != nil {
		t.Fatalf("ResolveWeapon: %v — FR-1, a ranged row resolves", err)
	}
	if viaResolve.Range != viaRange {
		t.Errorf("ResolveWeapon.Range = %d, want WeaponRange's own %d", viaResolve.Range, viaRange)
	}
}

// castTables holds one Weapons row, "Wood Staff", so a test can resolve the
// two shipped attachment forms 0139's spec Concrete guidance names against
// a real ResolveWeapon call rather than a fixture struct literal.
func castTables(t *testing.T) (data.ScaleTable, data.ScaleTable, data.Collection) {
	t.Helper()
	return tables(t, nil, nil,
		[]synth.DataBinRow{weaponRow("Wood Staff", 4, 6, 0, 0, 1, 6, 8, 4)})
}

// TestResolveWeaponParsesTheCastSpellAttachment is AC-2's positive half: a
// name ending in `{castSpell=TOKEN:LEVEL}` resolves to a weapon carrying
// TOKEN and LEVEL — checked against BOTH shipped forms 0139's spec
// measures, which differ in exactly the byte stripSuffix already handles:
// the space before `{` is present in one and absent in the other.
func TestResolveWeaponParsesTheCastSpellAttachment(t *testing.T) {
	s, m, w := castTables(t)

	for _, tc := range []struct {
		name      string
		wantToken string
		wantLevel int32
	}{
		{"Wood Staff {castSpell=Fire_Arrow:10}", "Fire_Arrow", 10},
		{"Wood Staff{castSpell=Fire_Ball:70}", "Fire_Ball", 70},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := data.ResolveWeapon(tc.name, s, m, w)
			if err != nil {
				t.Fatalf("ResolveWeapon(%q): %v", tc.name, err)
			}
			if got.SpellName != tc.wantToken || got.SpellPower != tc.wantLevel {
				t.Errorf("ResolveWeapon(%q) spell = (%q, %d), want (%q, %d)",
					tc.name, got.SpellName, got.SpellPower, tc.wantToken, tc.wantLevel)
			}
		})
	}
}

func TestAWeaponWithAndWithoutTheAttachmentAgreeOnEveryOtherField(t *testing.T) {
	s, m, w := castTables(t)

	plain, err := data.ResolveWeapon("Wood Staff", s, m, w)
	if err != nil {
		t.Fatalf("ResolveWeapon(plain): %v", err)
	}
	if plain.SpellName != "" || plain.SpellPower != 0 {
		t.Fatalf("a name with no attachment carries a spell: (%q, %d)", plain.SpellName, plain.SpellPower)
	}

	cast, err := data.ResolveWeapon("Wood Staff {castSpell=Fire_Arrow:10}", s, m, w)
	if err != nil {
		t.Fatalf("ResolveWeapon(cast): %v", err)
	}
	if cast.SpellName != "Fire_Arrow" || cast.SpellPower != 10 {
		t.Fatalf("ResolveWeapon(cast) spell = (%q, %d), want (\"Fire_Arrow\", 10)", cast.SpellName, cast.SpellPower)
	}

	cast.SpellName, cast.SpellPower = "", 0
	cast.Name, plain.Name = "", "" // the literal each was resolved from differs by construction
	if cast != plain {
		t.Fatalf("ResolveWeapon(with attachment) = %+v\nwant %+v (every field but the spell pair, and Name)", cast, plain)
	}
}

// TestAMalformedAttachmentLeavesNoSpellAndStillResolves is AC-3 and AC-15,
// at the two failures FR-1a's grammar itself can name (a token unresolvable
// against a Spells row is FR-1b's lookup, a later task's — see
// itemparse_test.go's own coverage of that split): the weapon still
// resolves, every other field agrees with the unsuffixed name, and the
// spell pair is the zero value rather than half of one.
func TestAMalformedAttachmentLeavesNoSpellAndStillResolves(t *testing.T) {
	s, m, w := castTables(t)

	plain, err := data.ResolveWeapon("Wood Staff", s, m, w)
	if err != nil {
		t.Fatalf("ResolveWeapon(plain): %v", err)
	}

	for _, tc := range []struct{ label, name string }{
		{"no colon", "Wood Staff {castSpell=Fire_Arrow10}"},
		{"a level that is not a run of digits", "Wood Staff {castSpell=Fire_Arrow:ten}"},
	} {
		t.Run(tc.label, func(t *testing.T) {
			got, err := data.ResolveWeapon(tc.name, s, m, w)
			if err != nil {
				t.Fatalf("ResolveWeapon(%q): %v — a malformed attachment must not refuse the weapon", tc.name, err)
			}
			if got.SpellName != "" || got.SpellPower != 0 {
				t.Fatalf("ResolveWeapon(%q) spell = (%q, %d), want (\"\", 0)", tc.name, got.SpellName, got.SpellPower)
			}
			got.Name = plain.Name
			if got != plain {
				t.Fatalf("a malformed attachment moved a field besides Name: got %+v\nwant %+v", got, plain)
			}
		})
	}
}
