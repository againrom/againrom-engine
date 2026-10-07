package ui

import (
	"image"
	"strings"
	"testing"
)

// The character sheet on the unit information panel (DD-6a, DD-6b).

// sheetSubject is a fully described unit: a name, a health pair, a cell, the
// generated fighter's character and the numbers it comes to.
func sheetSubject() PanelSubject {
	return PanelSubject{
		ID: 7, Name: "Warrior", HP: 63, MaxHP: 100, Cell: image.Pt(12, 34),
		Combat: UnitCombat{Known: true,
			DamageBase: 10, DamageSpread: 6, ToHit: 49, Defence: 8, Absorption: 0,
			AttackCharge: 9, AttackRelax: 5},
		Speed: 17,
		Char: UnitCharacter{Known: true,
			Body: 43, Reaction: 26, Mind: 15, Spirit: 15,
			// Slot 1 (Blade) trained to level 10, every other slot at zero —
			// nonzero AND zero at once, so the row this fixture feeds witnesses both
			// a real level and the zeros this story stops hiding.
			Skills: [PanelSkillSlots]int{0, 10, 0, 0, 0, 0}, Weapon: "Iron Short Sword",
			// Nonzero and mutually distinct, so the two new rows this fixture feeds
			// witness their own real text rather than three rows of zeroes.
			Experience: 1593, Protection: [5]int{12, 12, 12, 12, 12}, Resistance: [5]int{1, 2, 3, 4, 5},
			// Nonzero and distinct from every other number here, so the sight
			// row 0140 adds witnesses its own value rather than a zero that
			// would pass whether the field were plumbed through or not.
			Sight: 6},
		// The carried load, nonzero and distinct from every other number here
		// so the row 1025 adds witnesses its own value, and with the flag set
		// so a subject that filled it in is told apart from one that did not.
		// 3 and 5 are the two digits the fractional convention separates:
		// 35 prints as "3.5", and a build that dropped the fraction would
		// print "35" and a build that divided twice "0.3".
		Weight: 35, WeightKnown: true,
	}
}

// rowRow is one drawn row as rowsOf reports it: the left cell, and — when
// the row carries a resolved right one — the right cell beside it. It
// mirrors panelLine's own two cells rather than panelItem.String's joined
// text, so a test can name a row's right label without depending on the
// two-space separator PanelStatement documents.
type rowRow struct {
	label, value           string
	right                  bool
	rightLabel, rightValue string
}

// rowsOf is what the panel actually states for a subject, both cells of every
// row, in the order it states them. panelLines is the WHOLE of the text
// composition draws, so asserting over it asserts over everything the
// picture says.
func rowsOf(s PanelSubject) []rowRow {
	lines := panelLines(AuthoredPanelLayout(), panelFont(), s)
	out := make([]rowRow, len(lines))
	for i, ln := range lines {
		out[i] = rowRow{label: ln.label, value: ln.value,
			right: ln.right, rightLabel: ln.rightLabel, rightValue: ln.rightValue}
	}
	return out
}

func hasRowLabel(rows []rowRow, label string) bool {
	for _, r := range rows {
		if r.label == label || (r.right && r.rightLabel == label) {
			return true
		}
	}
	return false
}

// The whole sheet, row for row, in the order the owner gave it (0140).
//
// IT IS ASSERTED AS AN ORDER AND NOT AS A SET, which is what it was always
// for: a panel listing the same rows rearranged would pass a set check and
// be wrong about the only thing this test is about.
//
// sheetSubject is fully known, carries no mana pool, no always-hits mark and no
// worn set, and Selected is set to 1 here so the count row stays absent. Every
// consequence of those four absences is visible below: the mana heading and its
// pair are gone with the pool, ALWAYS HITS leaves SWING alone on its line, and
// WORN states nothing.
func TestAPanelStatesACharacterInTheOwnersOrder(t *testing.T) {
	s := sheetSubject()
	s.Selected = 1
	got := rowsOf(s)
	want := []rowRow{
		{label: "", value: "Warrior"},
		// The statistics against the pools. MIND and SPIRIT lose their right
		// cells with the mana pool; the statistics themselves are untouched.
		{label: "BODY", value: "43", right: true, rightLabel: "HEALTH", rightValue: ""},
		{label: "AGILITY", value: "26", right: true, rightLabel: "", rightValue: "63/100"},
		{label: "MIND", value: "15"},
		{label: "SPIRIT", value: "15"},
		// What a blow throws, what it meets.
		{label: "DMG", value: "10-16", right: true, rightLabel: "ABSORB", rightValue: "0"},
		{label: "ATTACK", value: "49", right: true, rightLabel: "DEFENSE", rightValue: "8"},
		// The two columns under their headings, then the slot the photograph
		// does not have and this tree does.
		{label: "SKILLS", value: "", right: true, rightLabel: "RESISTANCE", rightValue: ""},
		{label: "BLADE", value: "10", right: true, rightLabel: "FIRE", rightValue: "12"},
		{label: "AXE", value: "0", right: true, rightLabel: "WATER", rightValue: "12"},
		{label: "BLUDGEON", value: "0", right: true, rightLabel: "AIR", rightValue: "12"},
		{label: "PIKE", value: "0", right: true, rightLabel: "EARTH", rightValue: "12"},
		{label: "SHOOTING", value: "0", right: true, rightLabel: "ASTRAL", rightValue: "12"},
		{label: "GENERAL", value: "0"},
		// 1025's carried-weight row: 35 drawn with one fractional digit.
		{label: "WEIGHT", value: "3.5"},
		// The totals.
		{label: "XP", value: "1593"},
		{label: "SIGHT", value: "6.0"},
		{label: "SPEED", value: "17"},
		// Ours, below the sheet.
		{label: "WEAPON", value: "Iron Short Sword"},
		{label: "SWING", value: "9/5"},
		{label: "CELL", value: "12, 34"},
	}
	if len(got) != len(want) {
		t.Fatalf("the panel states %d row(s), want %d:\n%v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("row %d = %+v, want %+v", i, got[i], want[i])
		}
	}

	// THE WEAPON-KIND FAMILY IS GONE FROM THE PANEL and this says so out loud,
	// because a row that quietly stops being drawn is exactly what a row-order
	// test can hide behind a length change. The five numbers are Resistance's
	// own {1,2,3,4,5} in this fixture; for a real person nothing derives them
	// and for a real creature they are the same five the skill column prints.
	for _, r := range got {
		if r.value == "1 2 3 4 5" || r.rightValue == "1 2 3 4 5" {
			t.Errorf("the weapon-kind family is still drawn: %+v", r)
		}
	}
}

// AC-14 — a unit the loader knows no character for states its numbers and
// NOT a statistic, a skill or a weapon. The five stored slots do not change
// identity for a mage; their labels do. This is the exact live failure from
// save 666: Reniesta's level 10 is slot 2 (Water), and presenting that same
// value as AXE is a corrupt character sheet even when the entity beneath it
// is otherwise correct.
func TestAMagePanelLabelsSlotTwoWaterRatherThanAxe(t *testing.T) {
	s := sheetSubject()
	s.Char.Mage = true
	s.Char.Skills = [PanelSkillSlots]int{0, 0, 10, 0, 0, 0}

	rows := rowsOf(s)
	want := map[string]string{
		"FIRE": "0", "WATER": "10", "AIR": "0", "EARTH": "0", "ASTRAL": "0",
	}
	for label, value := range want {
		found := false
		for _, row := range rows {
			if row.label == label && row.value == value {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("mage panel has no %s %s row: %+v", label, value, rows)
		}
	}
	if hasRowLabel(rows, "AXE") || hasRowLabel(rows, "BLADE") {
		t.Fatalf("mage panel retained fighter labels: %+v", rows)
	}
	if got := CharacterSkillName(true, 2); got != "Water" {
		t.Fatalf("shared live/headless slot name = %q, want Water", got)
	}
}

func TestAUnitWithNoCharacterStatesOnlyItsNumbers(t *testing.T) {
	s := sheetSubject()
	s.Char = UnitCharacter{}
	s.Selected = 4

	got := rowsOf(s)
	want := []rowRow{
		{label: "SELECTED", value: "4"},
		{label: "", value: "Warrior"},
		// The health block slides into the left column: the statistics it was
		// paired with state nothing.
		{label: "HEALTH", value: ""},
		{label: "", value: "63/100"},
		{label: "DMG", value: "10-16", right: true, rightLabel: "ABSORB", rightValue: "0"},
		{label: "ATTACK", value: "49", right: true, rightLabel: "DEFENSE", rightValue: "8"},
		// THE WEIGHT ROW SURVIVES Char.Known BEING FALSE, and that is the
		// point of stating it here: a load is real for a creature carrying
		// loot, so its gate is the subject's own WeightKnown and not either
		// of the two blocks' flags.
		{label: "WEIGHT", value: "3.5"},
		{label: "SPEED", value: "17"},
		{label: "SWING", value: "9/5"},
		{label: "CELL", value: "12, 34"},
	}
	if len(got) != len(want) {
		t.Fatalf("the panel states %d row(s), want %d:\n%v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("row %d = %+v, want %+v", i, got[i], want[i])
		}
	}

	// A character that IS known but holds nothing worth a row drops WEAPON, not
	// SKILL: a bare hero holds no weapon and still states no such row, but
	// SKILL now states for every Known character, zeros included — an
	// untrained hero draws it the same as a trained one, which is this story's
	// own fix (AC-7) and the one place this test's old expectation is now wrong
	// on purpose.
	s.Char = UnitCharacter{Known: true, Body: 43, Reaction: 26, Mind: 15, Spirit: 15}
	rows := rowsOf(s)
	if hasRowLabel(rows, "WEAPON") {
		t.Errorf("an untrained bare hero states WEAPON; rows were %+v", rows)
	}
	for _, present := range []string{"BODY", "SPIRIT", "DMG", "SKILLS", "BLADE", "SHOOTING"} {
		if !hasRowLabel(rows, present) {
			t.Errorf("an untrained bare hero does not state %q; rows were %+v", present, rows)
		}
	}
}

// AC-14a — A SUBJECT TOLD NEITHER GROUP IS THE PANEL IT WAS, and this asserts it
// pixel for pixel against a layout holding only the four rows that existed
// before this story. That is an identity rather than a resemblance: it would
// fail if a new row stated a zero, if a dropped row cost space, or if any
// existing row, colour or measurement had moved.
func TestASubjectToldNeitherGroupComposesThePanelItWas(t *testing.T) {
	s := PanelSubject{ID: 7, Name: "Warrior", HP: 63, MaxHP: 100, Cell: image.Pt(12, 34), Selected: 4}

	before := AuthoredPanelLayout()
	before.Rows = []PanelRow{
		{Field: PanelFieldCount, Label: "SELECTED"},
		{Field: PanelFieldName},
		// The health block in its 0140 shape — a heading and the pair under
		// it. This list is "the rows that exist before a character or a combat
		// block is told", so it follows the arrangement; what it must not
		// contain is any field either of those two groups fills.
		{Field: PanelFieldHealthHeading, Label: "HEALTH"},
		{Field: PanelFieldHealth},
		{Field: PanelFieldCell, Label: "CELL"},
	}

	f := panelFont()
	now, was := composePanel(AuthoredPanelLayout(), f, s), composePanel(before, f, s)
	if now == nil || was == nil {
		t.Fatal("composePanel returned nil for a subject with a font")
	}
	if now.Bounds() != was.Bounds() {
		t.Fatalf("the box is %v where it was %v — an untold value took space",
			now.Bounds(), was.Bounds())
	}
	for i := range was.Pix {
		if now.Pix[i] != was.Pix[i] {
			t.Fatalf("byte %d of the picture differs: %#02x where it was %#02x",
				i, now.Pix[i], was.Pix[i])
		}
	}
}

// AC-15 — the damage row is the SHEET'S OWN COMPOSITION, base and base + spread,
// which are the roll's own bounds. `10-6` is what a row printing the pair
// verbatim would say, and it names a range the roll cannot produce.
func TestTheDamageRowIsTheSheetsOwnComposition(t *testing.T) {
	s := PanelSubject{Combat: UnitCombat{Known: true, DamageBase: 10, DamageSpread: 6}}
	got, ok := panelText(s, PanelFieldDamage)
	if !ok {
		t.Fatal("a known combat block stated no damage row")
	}
	if got != "10-16" {
		t.Errorf("the damage row reads %q, want %q", got, "10-16")
	}

	// A pair with no spread is a roll of one value, and the row says so rather
	// than hiding one end.
	s.Combat.DamageSpread = 0
	if got, _ := panelText(s, PanelFieldDamage); got != "10-10" {
		t.Errorf("a spreadless pair reads %q, want %q", got, "10-10")
	}
}

// AC-16 — the armor-piercing caption follows the original actor predicate,
// not UnitCombat.AlwaysHits. It is a caption rather than a quantity, so the
// cell is absent rather than negative.
// Owner-authored sheet semantics: a staff has one Damage value, the interval
// of the spell its attack actually releases. Its unused physical pair appears
// nowhere on the sheet.
func TestAWeaponSpellMakesDamageTheEffectiveSpellInterval(t *testing.T) {
	s := PanelSubject{Combat: UnitCombat{
		Known: true, DamageBase: 2, DamageSpread: 0,
		WeaponSpellDamageKnown: true, SpellDamageBase: 20, SpellDamageSpread: 4,
	}}
	if got, ok := panelText(s, PanelFieldDamage); !ok || got != "20-24" {
		t.Fatalf("DMG row = %q, %v; want the live spell interval 20-24", got, ok)
	}
}

func TestANonDamagingWeaponSpellHasNoDamageRow(t *testing.T) {
	s := PanelSubject{Combat: UnitCombat{Known: true, DamageBase: 20, WeaponSpellKnown: true}}
	if got, ok := panelText(s, PanelFieldDamage); ok {
		t.Fatalf("non-damaging staff states a DMG row %q", got)
	}
	s.Combat.WeaponSpellKnown = false
	if got, ok := panelText(s, PanelFieldDamage); !ok || got != "20-20" {
		t.Fatalf("ordinary weapon DMG = %q, %v; want 20-20", got, ok)
	}
}

func TestTheArmorPiercingRowUsesTheOriginalFlagsAndByte(t *testing.T) {
	s := sheetSubject()
	if _, ok := panelText(s, PanelFieldArmorPiercing); ok {
		t.Error("a subject with no original actor projection stated ARMOR PIERCING")
	}
	s.OriginalPanel = OriginalPanelActor{Known: true, Byte14A: 2}
	got, ok := panelText(s, PanelFieldArmorPiercing)
	if !ok || got != "" {
		t.Errorf("flags 0 and byte +0x14a=2 state (%q, %v), want an empty heading value and true", got, ok)
	}
	bare := sheetSubject()
	bare.OriginalPanel = OriginalPanelActor{Known: true}
	before, after := rowsOf(bare), rowsOf(s)
	if len(before) != len(after) {
		t.Errorf("setting the original predicate changed the row count from %d to %d, want unchanged — "+
			"ARMOR PIERCING shares XP's row", len(before), len(after))
	}
	if hasRowLabel(before, "ARMOR PIERCING") {
		t.Error("a subject with no original projection states an ARMOR PIERCING cell")
	}
	if !hasRowLabel(after, "ARMOR PIERCING") {
		t.Error("the original flags/byte predicate states no ARMOR PIERCING cell")
	}
	for _, flags := range []uint32{0x1, 0x10, 0x11} {
		blocked := s
		blocked.OriginalPanel.Flags = flags
		if _, ok := panelText(blocked, PanelFieldArmorPiercing); ok {
			t.Errorf("actor flags %#x did not suppress ARMOR PIERCING", flags)
		}
	}
	withoutByte := s
	withoutByte.OriginalPanel.Byte14A = 0
	if _, ok := panelText(withoutByte, PanelFieldArmorPiercing); ok {
		t.Error("zero actor byte +0x14a did not suppress ARMOR PIERCING")
	}
}

// THE TWO RESOLVERS STAY TOTAL AND DISJOINT over the shared field space, which
// is what makes a field written into the wrong layout omit its row rather than
// state a wrong value. The twelve numbers this story spends are `4..15`; the
// readout's are `16..31`.
func TestTheSharedFieldSpaceStaysDisjoint(t *testing.T) {
	mine := []PanelField{
		PanelFieldBody, PanelFieldReaction, PanelFieldMind, PanelFieldSpirit,
		PanelFieldSkill, PanelFieldWeapon, PanelFieldDamage, PanelFieldToHit,
		PanelFieldDefence, PanelFieldAbsorption, PanelFieldSwing, PanelFieldArmorPiercing,
		PanelFieldMoveSpeed,
	}
	seen := map[PanelField]bool{}
	for _, f := range mine {
		// `4..15` was this box's reservation and is spent; `16..31` is the
		// readout's; the thirteenth field starts a third range at 32.
		if (f < 4 || f > 15) && f != PanelFieldMoveSpeed {
			t.Errorf("field %d is neither inside the reservation 4..15 nor the allocated 32", f)
		}
		if f >= 16 && f <= 31 {
			t.Errorf("field %d is inside the readout's own range 16..31", f)
		}
		if seen[f] {
			t.Errorf("field number %d is allocated twice", f)
		}
		seen[f] = true

		// The readout must not answer for any of them, whatever it is told.
		full := readoutSubject{HasUnit: true, OnMap: true, Entities: 3, Frames: 60}
		if _, ok := readoutText(full, f); ok {
			t.Errorf("the readout resolved panel field %d", f)
		}
	}
	if len(seen) != 13 {
		t.Fatalf("%d distinct field numbers, want 13", len(seen))
	}
	// And this box still answers for none of the readout's.
	for _, f := range []PanelField{
		PanelFieldCadence, PanelFieldPeriod, PanelFieldTick, PanelFieldDigest,
		PanelFieldCursor, PanelFieldSpeed, PanelFieldGroup, PanelFieldCrossing,
		PanelFieldEntities, PanelFieldFrames, PanelFieldSetting, PanelFieldAttack,
	} {
		if _, ok := panelText(sheetSubject(), f); ok {
			t.Errorf("the unit panel resolved readout field %d", f)
		}
	}
}

// R-2, AC-7 — THE WHOLE SHEET FITS THE WINDOW. The box is fit-to-content
// and anchored to a BOTTOM corner, so it grows upward from its margin; a
// sheet taller than the window would run off the top with nothing to stop
// it. Measured rather than argued, and measured with every field stated —
// the count, the mark, the mana pool and the worn set — so it is the
// tallest the panel gets (AC-7).
//
// THE ROW COUNT GREW IN 0140 and this measurement is the reason the growth is
// bounded rather than merely intended: the owner's arrangement turns two
// five-wide families into ten labelled rows, and a panel that no longer fits
// its own window is exactly the failure a taller sheet would produce silently.
func TestTheWholeSheetFitsTheDefaultWindow(t *testing.T) {
	s := sheetSubject()
	s.Combat.AlwaysHits = true
	s.Selected = 4
	s.Mana, s.MaxMana = 30, 40
	s.Worn[0], s.Worn[1] = "Soft Boots", "Leather Cap"

	l, f := AuthoredPanelLayout(), panelFont()
	lines := panelLines(l, f, s)
	// Every row of the layout drawn at once — the check that this really is
	// the fullest case rather than merely a large one. It is the layout's own
	// length because 0140's arrangement has no row that can only be drawn
	// beside another; a subject that states everything states all of them.
	if len(lines) != len(l.Rows) {
		t.Fatalf("the fullest ordinary panel states %d row(s), want all %d layout rows",
			len(lines), len(l.Rows))
	}
	box := panelBox(l, f, lines)
	if h := box.Y + 2*l.Margin.Y; h > DefaultWindowH {
		t.Errorf("the fullest panel needs %d pixels of height and the window is %d",
			h, DefaultWindowH)
	}
	at := panelOrigin(l, image.Pt(DefaultWindowW, DefaultWindowH), box)
	if at.X < 0 || at.Y < 0 {
		t.Errorf("the fullest panel is placed at %v, which is off the window", at)
	}
}

// AC-5 — the multiset of PanelField values across every cell of
// AuthoredPanelLayout — the left cell of every row and the right cell of
// every row that has one — is exactly the field set this test enumerates,
// each appearing once.
func TestAuthoredPanelLayoutStatesExactlyTheEnumeratedFields(t *testing.T) {
	want := map[PanelField]bool{
		PanelFieldCount: true, PanelFieldName: true,
		PanelFieldHealth: true, PanelFieldMana: true,
		PanelFieldHealthHeading: true, PanelFieldManaHeading: true,
		PanelFieldCell: true, PanelFieldMoveSpeed: true,
		PanelFieldBody: true, PanelFieldReaction: true,
		PanelFieldMind: true, PanelFieldSpirit: true,
		PanelFieldExperience: true, PanelFieldWeapon: true,
		PanelFieldWorn:   true,
		PanelFieldDamage: true, PanelFieldToHit: true,
		PanelFieldDefence: true, PanelFieldAbsorption: true,
		PanelFieldSwing: true, PanelFieldArmorPiercing: true,
		PanelFieldSight: true,
		// 1025's carried-weight row.
		PanelFieldWeight: true, PanelFieldSpellcaster: true,

		// The two columns, one field per drawn number since 0140, under two
		// headings of their own.
		PanelFieldSkillsHeading: true, PanelFieldResistHeading: true,
		PanelFieldSkillGeneral: true, PanelFieldSkillBlade: true,
		PanelFieldSkillAxe: true, PanelFieldSkillBludgeon: true,
		PanelFieldSkillPike: true, PanelFieldSkillShooting: true,
		PanelFieldProtFire: true, PanelFieldProtWater: true,
		PanelFieldProtAir: true, PanelFieldProtEarth: true,
		PanelFieldProtAstral: true,
	}
	// THREE FIELDS THIS LAYOUT DELIBERATELY NO LONGER STATES, named here rather
	// than merely absent above, because "not in the set" and "removed on
	// purpose" look identical to a reader of a map literal. Each is still
	// defined and still resolvable — another layout may want a family on one
	// line — and the layout below must state none of them.
	gone := map[PanelField]string{
		PanelFieldSkill:      "the six slots space-separated, replaced by one row per slot",
		PanelFieldProtection: "the elemental family on one line, replaced by five labelled rows",
		PanelFieldResistance: "the weapon-kind family: a duplicate for a creature, five zeros for a person",
	}

	got := map[PanelField]int{}
	for _, r := range AuthoredPanelLayout().Rows {
		got[r.Field]++
		if r.Right != nil {
			got[r.Right.Field]++
		}
	}
	for f := range want {
		if got[f] != 1 {
			t.Errorf("field %d appears %d time(s) in the layout, want exactly 1", f, got[f])
		}
	}
	for f, why := range gone {
		if got[f] != 0 {
			t.Errorf("field %d is stated by the layout and was removed from it: %s", f, why)
		}
	}
	for f := range got {
		if !want[f] {
			t.Errorf("field %d is stated by the layout and is not in the hand-written set", f)
		}
	}

	// NO FIELD THIS LAYOUT STATES IS ONE THE READOUT ANSWERS FOR. The shared
	// number space is what makes that worth asserting: the two boxes allocate
	// out of one range, so a new panel field taking a number the readout
	// already holds would have both boxes resolving it and neither saying so.
	full := readoutSubject{HasUnit: true, OnMap: true, Entities: 3, Frames: 60}
	for f := range got {
		if _, ok := readoutText(full, f); ok {
			t.Errorf("the readout also resolves panel field %d", f)
		}
	}
}

// panelStatementCellCount counts the CELLS a stated set holds rather than
// its entries: PanelStatement's own doc says a two-cell row's entry joins
// both cells with two spaces, the one substring no label or value in this
// build contains on its own — ALWAYS HITS included, which is two words
// joined by a single space.
func panelStatementCellCount(rows []string) int {
	n := 0
	for _, r := range rows {
		n++
		if strings.Contains(r, "  ") {
			n++
		}
	}
	return n
}

func TestTheShippedLayoutDrawsTheCanonicalRowCounts(t *testing.T) {
	full := sheetSubject()
	full.Selected = 1

	got := PanelStatement(AuthoredPanelLayout(), full)
	if len(got) != 21 {
		t.Fatalf("a fully-known subject draws %d row(s), want 21: %v", len(got), got)
	}
	if n := panelStatementCellCount(got); n != 31 {
		t.Errorf("a fully-known subject's stated set holds %d value(s), want 31: %v", n, got)
	}

	placed := full
	placed.Char = UnitCharacter{}
	got = PanelStatement(AuthoredPanelLayout(), placed)
	if len(got) != 9 {
		t.Fatalf("a placed subject draws %d row(s), want 9: %v", len(got), got)
	}
	if n := panelStatementCellCount(got); n != 11 {
		t.Errorf("a placed subject's stated set holds %d value(s), want 11: %v", n, got)
	}
}

func TestAChangedNumberRebuildsThePictureAndAnUnchangedOneDoesNot(t *testing.T) {
	v := panelViewer(t)
	v.SetFont(panelFont())

	e := panelEntity(1, "Warrior", 63, 100, 12, 34)
	e.Combat = sheetSubject().Combat
	e.Char = sheetSubject().Char
	v.SetEntities([]MapEntity{e})
	v.sel = selection{1}

	if _, _, ok := v.panelPresent(); !ok {
		t.Fatal("no panel for a selected entity")
	}
	builds := v.panelBuilds
	v.SetEntities([]MapEntity{e})
	v.panelPresent()
	if v.panelBuilds != builds {
		t.Errorf("an unchanged frame rebuilt the picture (%d builds, was %d)", v.panelBuilds, builds)
	}

	for _, change := range []func(*MapEntity){
		func(e *MapEntity) { e.Char.Body++ },
		func(e *MapEntity) { e.Combat.ToHit++ },
		func(e *MapEntity) { e.Char.Weapon = "Bronze Pike" },
	} {
		change(&e)
		v.SetEntities([]MapEntity{e})
		v.panelPresent()
		if v.panelBuilds != builds+1 {
			t.Errorf("a changed value did not rebuild the picture (%d builds, was %d)",
				v.panelBuilds, builds)
		}
		builds = v.panelBuilds
	}
}
