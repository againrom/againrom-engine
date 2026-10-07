package main

import (
	"bytes"
	"strings"
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/mapload"
)

// Everything in this file is SYNTHETIC (golden rule 2; spec plan D-8): every
// collection is built in the test, and every name in it is this test's own
// invention, never a shipped one. wearcheck's own read of a lawful install
// is exercised by hand and its output is what verification.md records
// (AC-6) — nothing here opens an archive.

// testEntry and testCollection are a minimal data.Collection.
type testEntry struct {
	name    string
	params  []int32
	strings []string
}

type testCollection []testEntry

func (c testCollection) Len() int                    { return len(c) }
func (c testCollection) EntryName(i int) string      { return c[i].name }
func (c testCollection) EntryParams(i int) []int32   { return c[i].params }
func (c testCollection) EntryStrings(i int) []string { return c[i].strings }

// emptyScaleTable is a data.ScaleTable with no entries: takePrefix's own nil
// arm and this arm agree (itemparse.go's own doc on takePrefix), so a name
// with no shape or material word this test authors resolves through index 0
// unchanged, exactly as an install's own name with no such word does.
type emptyScaleTable struct{}

func (emptyScaleTable) Len() int                   { return 0 }
func (emptyScaleTable) EntryName(int) string       { return "" }
func (emptyScaleTable) EntryDoubles(int) []float64 { return nil }

// weaponParams builds a Weapons row long enough for ResolveWeapon to read
// (weaponRelaxSlot, weapon.go's own slot 0xd) and for this file's own
// twoHanded to read the Hands column (weaponHandsColumn, 0xe) — sixteen
// cells, attack type melee (index 5, any value under 0xa), hands at index
// 0xe.
func weaponParams(hands int32) []int32 {
	p := make([]int32, 16)
	p[5] = 1 // attack type: melee
	p[0xe] = hands
	return p
}

// armorParams builds an Armors row carrying one cell this file reads:
// armorSlotColumn (wear.go's own param 4).
func armorParams(slot int32) []int32 {
	return []int32{0, 0, 0, 0, slot}
}

func testTable() *mapload.Table {
	weapons := testCollection{
		{name: ""}, // reserved
		{name: "Dagger", params: weaponParams(1)},
		{name: "Poleaxe", params: weaponParams(2)},
	}
	shields := testCollection{
		{name: ""},
		{name: "Round"},
	}
	armors := testCollection{
		{name: ""},
		{name: "Chain Mail", params: armorParams(4)},
		{name: "Helmet", params: armorParams(4)},
		{name: "Boots", params: armorParams(7)},
		{name: "BadArmor", params: armorParams(0)},
	}
	return &mapload.Table{
		Shapes:    emptyScaleTable{},
		Materials: emptyScaleTable{},
		Weapons:   weapons,
		Shields:   shields,
		Armors:    armors,
	}
}

// TestResolveRowOrdinary is AC-1's own shape: a weapon, a shield and one
// armour, each landing in its own slot, nothing carried and nothing dropped.
func TestResolveRowOrdinary(t *testing.T) {
	tbl := testTable()
	names := []string{"Dagger", "Round Shield", "Chain Mail", "", "", "", "", "", "", ""}
	cells, worn, carried, collisions := resolveRow(1, names, tbl)

	if collisions != 0 {
		t.Errorf("collisions = %d, want 0", collisions)
	}
	if len(carried) != 0 {
		t.Errorf("carried = %v, want none", carried)
	}
	if worn[0] == 0 {
		t.Error("slot 1 (weapon) is empty")
	}
	if worn[1] == 0 {
		t.Error("slot 2 (shield) is empty")
	}
	if worn[3] == 0 {
		t.Error("slot 4 (armour) is empty")
	}
	for _, c := range cells {
		if c.cell < cellArmorFrom+1 && c.outcome != outcomeWorn && c.name != "" {
			t.Errorf("cell %d (%s) = %v, want worn", c.cell, c.name, c.outcome)
		}
	}
	for _, c := range cells[3:] {
		if c.outcome != outcomeEmpty {
			t.Errorf("cell %d = %v, want empty", c.cell, c.outcome)
		}
	}
}

// TestResolveRowTwoHandedDisplacement is AC-3's own shape: a two-handed
// weapon in cell 0 and a shield in cell 1 leave the shield worn, slot 1
// empty and the weapon carried.
func TestResolveRowTwoHandedDisplacement(t *testing.T) {
	tbl := testTable()
	names := []string{"Poleaxe", "Round Shield", "", "", "", "", "", "", "", ""}
	cells, worn, carried, _ := resolveRow(1, names, tbl)

	if worn[0] != 0 {
		t.Errorf("slot 1 = 0x%04x, want empty (the weapon is displaced)", worn[0])
	}
	if worn[1] == 0 {
		t.Error("slot 2 (shield) is empty")
	}
	if len(carried) != 1 {
		t.Fatalf("carried = %v, want exactly the weapon", carried)
	}
	if cells[0].outcome != outcomeCarried {
		t.Errorf("weapon cell outcome = %v, want carried", cells[0].outcome)
	}
	if cells[0].reason == "" {
		t.Error("a carried cell must carry a reason")
	}
}

// TestResolveRowOneHandedIsNotDisplaced is AC-3's other half: a one-handed
// weapon and a shield both stay worn.
func TestResolveRowOneHandedIsNotDisplaced(t *testing.T) {
	tbl := testTable()
	names := []string{"Dagger", "Round Shield", "", "", "", "", "", "", "", ""}
	cells, worn, carried, _ := resolveRow(1, names, tbl)

	if worn[0] == 0 || worn[1] == 0 {
		t.Errorf("worn = %v, want slots 1 and 2 both filled", worn)
	}
	if len(carried) != 0 {
		t.Errorf("carried = %v, want none", carried)
	}
	if cells[0].outcome != outcomeWorn {
		t.Errorf("weapon cell outcome = %v, want worn", cells[0].outcome)
	}
}

// TestResolveRowBadSlotIsCarried is AC-4's own shape: an armour whose Slot
// column is 0 lands in the container, not a slot.
func TestResolveRowBadSlotIsCarried(t *testing.T) {
	tbl := testTable()
	names := []string{"", "", "BadArmor", "", "", "", "", "", "", ""}
	cells, worn, carried, _ := resolveRow(1, names, tbl)

	if worn != [data.EquipSlots]uint16{} {
		t.Errorf("worn = %v, want all empty", worn)
	}
	if len(carried) != 1 {
		t.Fatalf("carried = %v, want exactly the one armour", carried)
	}
	if cells[2].outcome != outcomeCarried {
		t.Errorf("cell 2 outcome = %v, want carried", cells[2].outcome)
	}
}

// TestResolveRowUnresolvableNameIsDropped is AC-5's own shape: an armour
// name that names no row is neither worn nor carried.
func TestResolveRowUnresolvableNameIsDropped(t *testing.T) {
	tbl := testTable()
	names := []string{"", "", "Nonexistent Item", "", "", "", "", "", "", ""}
	cells, worn, carried, _ := resolveRow(1, names, tbl)

	if worn != [data.EquipSlots]uint16{} {
		t.Errorf("worn = %v, want all empty", worn)
	}
	if len(carried) != 0 {
		t.Errorf("carried = %v, want none", carried)
	}
	if cells[2].outcome != outcomeDropped {
		t.Errorf("cell 2 outcome = %v, want dropped", cells[2].outcome)
	}
}

func TestResolveRowDestinationCollision(t *testing.T) {
	tbl := testTable()
	names := []string{"", "", "Chain Mail", "", "", "", "Helmet", "", "", ""}
	cells, worn, _, collisions := resolveRow(1, names, tbl)

	if collisions != 1 {
		t.Errorf("collisions = %d, want 1", collisions)
	}
	if worn[3] == 0 {
		t.Error("slot 4 is empty; want the second cell's code to have landed there")
	}
	if cells[2].outcome != outcomeWorn || cells[6].outcome != outcomeWorn {
		t.Errorf("both colliding cells should read worn: cell2=%v cell6=%v", cells[2].outcome, cells[6].outcome)
	}
}

func TestResolveRowMissingCollectionRefusesOnlyItsClass(t *testing.T) {
	tbl := testTable()
	tbl.Armors = nil
	names := []string{"Dagger", "Round Shield", "Chain Mail", "", "", "", "", "", "", ""}
	cells, worn, _, _ := resolveRow(1, names, tbl)

	if worn[0] == 0 || worn[1] == 0 {
		t.Errorf("worn = %v, want weapon and shield still armed", worn)
	}
	if cells[2].outcome != outcomeDropped {
		t.Errorf("armour cell outcome = %v, want dropped", cells[2].outcome)
	}
	if !strings.Contains(cells[2].reason, "Armors") {
		t.Errorf("reason = %q, want it to name the missing collection", cells[2].reason)
	}
}

// TestScanTalliesAcrossRows exercises scan() end to end over a small
// synthetic Humans collection covering an ordinary row, a two-handed
// displacement, a bad-slot carry, an unresolvable drop and a same-row
// collision — and checks the report's own totals agree with what each row,
// read by hand, says it should contribute.
func TestScanTalliesAcrossRows(t *testing.T) {
	tbl := testTable()
	tbl.Humans = testCollection{
		{name: ""}, // reserved
		{name: "Ordinary", strings: []string{"Dagger", "Round Shield", "Chain Mail", "", "", "", "", "", "", ""}},
		{name: "TwoHanded", strings: []string{"Poleaxe", "Round Shield", "", "", "", "", "", "", "", ""}},
		{name: "BadSlot", strings: []string{"", "", "BadArmor", "", "", "", "", "", "", ""}},
		{name: "Unresolvable", strings: []string{"", "", "Ghost Item", "", "", "", "", "", "", ""}},
		{name: "Collision", strings: []string{"", "", "Chain Mail", "", "", "", "Helmet", "", "", ""}},
		{name: "Bare"}, // no strings at all: every cell reads "" and is empty
	}

	rep := scan(tbl)

	if rep.totalRows != 6 {
		t.Errorf("totalRows = %d, want 6", rep.totalRows)
	}
	if rep.namedRows != 5 {
		t.Errorf("namedRows = %d, want 5 (every row but the bare one)", rep.namedRows)
	}
	if rep.collisions != 1 {
		t.Errorf("collisions = %d, want 1", rep.collisions)
	}

	// weapon: Ordinary worn, TwoHanded resolved-then-carried — 2 used, 1
	// worn, 1 carried, 0 dropped.
	if w := rep.classes[classWeapon]; w != (classStats{used: 2, worn: 1, carried: 1, dropped: 0}) {
		t.Errorf("weapon stats = %+v, want {2 1 1 0}", w)
	}
	// shield: Ordinary and TwoHanded both worn — 2 used, 2 worn.
	if s := rep.classes[classShield]; s != (classStats{used: 2, worn: 2, carried: 0, dropped: 0}) {
		t.Errorf("shield stats = %+v, want {2 2 0 0}", s)
	}
	// armour: Ordinary (worn), BadSlot (carried), Unresolvable (dropped),
	// Collision x2 (both counted worn per resolveRow's own rule) — 5 used,
	// 3 worn, 1 carried, 1 dropped.
	if a := rep.classes[classArmor]; a != (classStats{used: 5, worn: 3, carried: 1, dropped: 1}) {
		t.Errorf("armour stats = %+v, want {5 3 1 1}", a)
	}

	// Anomalies: the displaced weapon, the bad-slot armour and the
	// unresolvable armour — three, no more and no fewer, and the
	// collision does not itself add one (it is a distinct count).
	if len(rep.anomalies) != 3 {
		t.Errorf("anomalies = %d, want 3", len(rep.anomalies))
	}

	// Slot histogram: slot 1 has one occupant (Ordinary's Dagger; the
	// Poleaxe never lands there), slot 2 has two (both shields), slot 4
	// has two (Ordinary's Chain Mail and the SURVIVING half of the
	// collision — the collision leaves one physical occupant, not two).
	if got := rep.histogram[1]; got != 1 {
		t.Errorf("histogram[1] = %d, want 1", got)
	}
	if got := rep.histogram[2]; got != 2 {
		t.Errorf("histogram[2] = %d, want 2", got)
	}
	if got := rep.histogram[4]; got != 2 {
		t.Errorf("histogram[4] = %d, want 2 (one per row, collisions still leave one occupant)", got)
	}

	// The ordinary row and the two-handed row both qualify for the
	// sample (clean: no drop, no carry, no collision, per this file's own
	// scan()); BadSlot, Unresolvable and Collision do not, and Bare names
	// nothing so it is not "ordinary" either.
	for _, s := range rep.samples {
		if s.row != 1 && s.row != 2 {
			t.Errorf("sample includes row %d, want only the two clean rows", s.row)
		}
	}
}

func TestPrintReportDoesNotPanicOnAnEmptyReport(t *testing.T) {
	var buf bytes.Buffer
	printReport(&buf, report{})
	if !strings.Contains(buf.String(), "0 Humans rows scanned") {
		t.Errorf("output = %q, want it to state zero rows", buf.String())
	}
	if !strings.Contains(buf.String(), "none") {
		t.Error("an empty anomaly list should say so")
	}
}

// TestRunRefusesBeforeOpeningAnyArchive mirrors cmd/missionrun's own pattern:
// every refusal below is reached on its arguments alone, with the asset root
// pointed at an empty temp directory that holds no install, so none of them
// may read one.
func TestRunRefusesBeforeOpeningAnyArchive(t *testing.T) {
	t.Setenv("AGAINROM_ASSETS", "")
	for _, tc := range []struct {
		name string
		args []string
		want string
	}{
		{"no asset root", []string{}, "asset root"},
		{"an unknown flag", []string{"-nosuchflag"}, "flag"},
		// -spells IS A RECOGNISED FLAG (0139 spec AC-1): it must parse
		// cleanly and reach the SAME "no asset root" refusal every other
		// well-formed argument list reaches; an unregistered flag would
		// instead fail earlier, with flag.Parse's own "flag provided but
		// not defined".
		{"-spells with no asset root", []string{"-spells"}, "asset root"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer
			if err := run(tc.args, &buf); err == nil {
				t.Fatalf("run(%v) returned no error; it printed %q", tc.args, buf.String())
			} else if !strings.Contains(err.Error(), tc.want) {
				t.Errorf("run(%v) = %v, want an error naming %q", tc.args, err, tc.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// -spells (0139 spec AC-1): every castSpell= token this install's Humans and
// Units rows name, and whether it resolves to a Spells row.
// ---------------------------------------------------------------------------

// TestScanSpellsFindsTokensInBothCollections is AC-1's own claim over BOTH
// definition tiers: a Humans row and a Units row each naming a
// castSpell-attached weapon both surface, each resolved through the SAME
// data.ResolveWeapon and mapload.SpellIDByToken production uses.
func TestScanSpellsFindsTokensInBothCollections(t *testing.T) {
	tbl := testTable()
	tbl.Humans = testCollection{
		{}, // reserved
		{name: "Shaman", strings: []string{"Dagger{castSpell=Fire_Arrow:10}", "", "", "", "", "", "", "", "", ""}},
	}
	tbl.Units = testCollection{
		{}, // reserved
		{name: "Catapult", strings: []string{"Dagger{castSpell=Fire_Ball:20}", ""}},
	}
	tbl.Spells = testCollection{
		{}, // reserved
		{name: "Fire Arrow"},
		{name: "Fire Ball"},
	}

	toks := scanSpells(tbl)
	if len(toks) != 2 {
		t.Fatalf("scanSpells found %d token(s), want 2: %+v", len(toks), toks)
	}
	bySource := map[string]spellToken{}
	for _, tk := range toks {
		bySource[tk.source] = tk
	}
	if tk := bySource["Humans"]; tk.weaponErr != "" || tk.spellName != "Fire_Arrow" || tk.spellPower != 10 ||
		!tk.resolved || tk.spellID != 1 {
		t.Errorf("Humans token = %+v, want Fire_Arrow:10 resolved to spell id 1", tk)
	}
	if tk := bySource["Units"]; tk.weaponErr != "" || tk.spellName != "Fire_Ball" || tk.spellPower != 20 ||
		!tk.resolved || tk.spellID != 2 {
		t.Errorf("Units token = %+v, want Fire_Ball:20 resolved to spell id 2", tk)
	}

	var buf bytes.Buffer
	if err := printSpellReport(&buf, toks); err != nil {
		t.Errorf("printSpellReport returned %v for two resolved tokens, want nil", err)
	}
	if !strings.Contains(buf.String(), "2 castSpell= token(s) found") {
		t.Errorf("output = %q, want the found count", buf.String())
	}
}

// TestScanSpellsReportsATokenThatNamesNoSpellsRow is FR-1b's own negative
// half, at the tool: a token whose weapon resolves but whose NAME matches
// no row of the installed Spells collection prints "NO SPELLS ROW" and
// printSpellReport returns a non-nil error — AC-1's own claim is that
// EVERY token resolves, so this is the shape a real gap would take.
func TestScanSpellsReportsATokenThatNamesNoSpellsRow(t *testing.T) {
	tbl := testTable()
	tbl.Humans = testCollection{
		{},
		{name: "Shaman", strings: []string{"Dagger{castSpell=Teleport:5}", "", "", "", "", "", "", "", "", ""}},
	}
	tbl.Spells = testCollection{{}, {name: "Fire Arrow"}}

	toks := scanSpells(tbl)
	if len(toks) != 1 {
		t.Fatalf("scanSpells found %d token(s), want 1", len(toks))
	}
	if toks[0].resolved {
		t.Error("Teleport resolved against a table naming only Fire Arrow")
	}

	var buf bytes.Buffer
	err := printSpellReport(&buf, toks)
	if err == nil {
		t.Error("printSpellReport returned nil for an unresolved token, want a non-nil error")
	}
	if !strings.Contains(buf.String(), "NO SPELLS ROW") {
		t.Errorf("output = %q, want it to name the unresolved token", buf.String())
	}
}

func TestScanSpellsReportsAWeaponThatDoesNotResolve(t *testing.T) {
	tbl := testTable()
	tbl.Humans = testCollection{
		{},
		{name: "Ghost", strings: []string{"Ghost Item{castSpell=Fire_Arrow:10}", "", "", "", "", "", "", "", "", ""}},
	}
	toks := scanSpells(tbl)
	if len(toks) != 1 || toks[0].weaponErr == "" {
		t.Fatalf("scanSpells = %+v, want one token with a non-empty weaponErr", toks)
	}
	if toks[0].resolved {
		t.Error("a token whose weapon did not resolve at all reported resolved=true")
	}
}

func TestScanSpellsAnswersNoTokensForANilOrEmptyTable(t *testing.T) {
	if got := scanSpells(nil); got != nil {
		t.Errorf("scanSpells(nil) = %v, want nil", got)
	}
	if got := scanSpells(&mapload.Table{}); got != nil {
		t.Errorf("scanSpells(&mapload.Table{}) = %v, want nil", got)
	}
}

// scaleOne is a one-entry scale table whose every factor is 1, so a swept
// weight is its row's own column and this test states one number per row. The
// entry is NAMED because the weapon arm resolves a code by recomposing the
// shape, material and row names and parsing them back, so the shape and the
// material ladders must not answer one string.
type scaleOne struct{ label string }

func (scaleOne) Len() int               { return 1 }
func (s scaleOne) EntryName(int) string { return s.label }
func (scaleOne) EntryDoubles(int) []float64 {
	return []float64{1, 1, 1, 1, 1, 1, 1, 1, 1}
}

// TestSweepWeightsCountsTheColumnItReads is the -weights mode over a synthetic
// table: three collections, one written row each, and a reserved zeroth entry
// each collection allocates and never writes.
//
// The sweep is one shape and one material wide here, so each collection's
// triple count is its own row count and the refusals are the reserved rows.
func TestSweepWeightsCountsTheColumnItReads(t *testing.T) {
	// The row is wide enough for the widest column any of the three arms
	// reads: the Armors absorption at 10 and the Weapons relax cell at 13.
	const weightColumn, widestColumn = 3, 14
	row := func(weight int32) []int32 {
		p := make([]int32, widestColumn+1)
		for i := range p {
			p[i] = -1
		}
		p[4] = 4 // the Armors slot cell; the other two collections ignore it
		p[weightColumn] = weight
		return p
	}
	tbl := &mapload.Table{
		Shapes: scaleOne{"Plain"}, Materials: scaleOne{"Iron"},
		Weapons: testCollection{{}, {name: "W", params: row(23)}},
		Shields: testCollection{{}, {name: "S", params: row(7)}},
		Armors:  testCollection{{}, {name: "A", params: row(-30)}},
	}

	got := sweepWeights(tbl)
	if len(got) != 3 {
		t.Fatalf("the sweep answered %d collection(s), want 3", len(got))
	}
	for _, want := range []struct {
		name     string
		max      int32
		negative int
	}{
		{"Weapons", 23, 0},
		{"Shields", 7, 0},
		{"Armors", -29, 1},
	} {
		var s weightSweep
		for _, c := range got {
			if c.name == want.name {
				s = c
			}
		}
		if s.name != want.name {
			t.Fatalf("the sweep answered nothing for %s", want.name)
		}
		if s.triples != 2 || s.refused != 1 {
			t.Errorf("%s swept %d code(s) and refused %d, want 2 and 1 — the reserved row",
				s.name, s.triples, s.refused)
		}
		if s.max != want.max {
			t.Errorf("%s's heaviest resolved weight is %d, want %d", s.name, s.max, want.max)
		}
		if s.negative != want.negative {
			t.Errorf("%s counted %d negative weight(s), want %d", s.name, s.negative, want.negative)
		}
	}

	// A table with no ladder at all scales nothing and answers nothing, rather
	// than reporting a range over an unscaled column.
	if got := sweepWeights(&mapload.Table{Weapons: testCollection{{}, {name: "W", params: row(23)}}}); got != nil {
		t.Errorf("a table with no shape ladder answered %v, want nothing", got)
	}
}
