package data_test

import (
	"math"
	"strings"
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/data"
	"againrom/pkg/formats/databin"
)

// spellRow builds a Spells row out of the established scalar slots, so a test
// states the numbers it cares about and not nineteen cells. A caller that
// exercises Complication authors slot 0 on the returned row.
//
// The cells are laid out by PARAMETER SLOT, matching LoadSpells' own reading;
// every slot the contract does not read stays the format's own empty cell,
// -1 — except Mana Cost, which LoadSpells refuses when negative, so a filler
// row that does not care about its own cost is given manaCost explicitly
// rather than -1.
func spellRow(name string, manaCost, school, target, maxRange, dmgMin, dmgMax, defensive int32) synth.DataBinRow {
	p := make([]int32, lastSpellRowSlots)
	for i := range p {
		p[i] = -1
	}
	p[1], p[2], p[4], p[6], p[16], p[17], p[18] = manaCost, school, target, maxRange, dmgMin, dmgMax, defensive
	// SLOT 8 IS AUTHORED AS A POINT ROW rather than left at the empty cell.
	// LoadSpells reads `Distribution system` as "1 is a point effect, anything
	// else is an area effect", so -1 would make every row this helper builds an
	// area row — which is not what any of the callers below are describing,
	// and is the opposite of the shipped table's own majority of 18 point rows
	// to 10 area ones. A test that wants an area row builds it with
	// spellRowArea.
	p[8] = 1
	return synth.DataBinRow{Name: name, Params: p}
}

func spellRowArea(name string, manaCost, school, target, maxRange, dmgMin, dmgMax, defensive,
	distribution, areaDuration int32) synth.DataBinRow {
	row := spellRow(name, manaCost, school, target, maxRange, dmgMin, dmgMax, defensive)
	row.Params[8] = distribution
	row.Params[11] = areaDuration
	return row
}

const lastSpellRowSlots = 19

// spells parses one synthetic definition table carrying only a Spells
// collection and hands back the collection LoadSpells reads. Never a game
// install — the rows are built in Go, as pkg/data/weapon_test.go's tables
// helper builds its own three collections.
func spells(t *testing.T, rows []synth.DataBinRow) data.Collection {
	t.Helper()
	f, err := databin.Parse(synth.DataBin{
		Rows: [synth.DataBinCollections][]synth.DataBinRow{
			synth.DataBinSpells: rows,
		},
	}.Bytes())
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	return f.Collection(databin.Spells)
}

func spellDamageFR8(dmin, dmax, power int32) (base, spread int32) {
	base = dmin * (power + 30) / 30
	spread = dmax*(power+30)/30 - base
	return base, spread
}

// TestLoadSpellsFireArrowAndTheNineDamageRows is AC-1 and AC-2 together.
//
// AC-1: Fire Arrow's own row loads its named fields exactly.
//
// AC-2: at power 0 (f = 1) OUR reading — base = DamageMin, spread =
// DamageMax - DamageMin — reproduces every one of the nine shipped damage
// rows' own damageMin..damageMax band exactly (MAGIC-DMG-005's corpus:
// 4-8, 7-13, 1-3, 8-16, 10-14, 3-5, 5-15, 5-15, 5-25 — Fire Arrow, Fire Ball,
// Wall of Fire, Heal, Acid Stream, Drain Life, Lightning, Prismatic Spray,
// Meteor Storm), while the RIVAL reading — the second column taken as a
// maximum rather than a spread, so its band is damageMin..damageMin+damageMax
// — reproduces none of them, because every one of the nine ships a positive
// DamageMin.
func TestLoadSpellsFireArrowAndTheNineDamageRows(t *testing.T) {
	rows := []synth.DataBinRow{
		spellRow("Fire Arrow", 3, 1, 1, 7, 4, 8, 0),
		spellRow("Fire Ball", 30, 1, 2, 10, 7, 13, 0),
		spellRow("Wall of Fire", 30, 1, 2, 6, 1, 3, 0),
		spellRow("Heal", 5, 2, 1, 6, 8, 16, 0),
		spellRow("Acid Stream", 100, 2, 2, 3, 10, 14, 0),
		spellRow("Drain Life", 20, 2, 1, 3, 3, 5, 0),
		spellRow("Lightning", 10, 3, 1, 8, 5, 15, 0),
		spellRow("Prismatic Spray", 80, 3, 2, 7, 5, 15, 0),
		spellRow("Meteor Storm", 100, 4, 2, 10, 5, 25, 0),
	}
	rows[0].Params[0] = 3 // Complication Level
	c := spells(t, rows)

	got, err := data.LoadSpells(c)
	if err != nil {
		t.Fatalf("LoadSpells: %v", err)
	}
	if len(got) != len(rows) {
		t.Fatalf("LoadSpells returned %d spell(s), want %d", len(got), len(rows))
	}

	// AC-1.
	want := data.Spell{
		Name: "Fire Arrow", Complication: 3, ManaCost: 3, School: 1, TargetsUnit: true, MaxRange: 7, Distribution: 1,
		DamageMin: 4, DamageMax: 8, Defensive: false, Damaging: true,
	}
	if got[0] != want {
		t.Fatalf("LoadSpells()[0] = %+v\nwant %+v", got[0], want)
	}

	// AC-2.
	ours, rival := 0, 0
	for _, sp := range got {
		base, spread := spellDamageFR8(sp.DamageMin, sp.DamageMax, 0)
		if base == sp.DamageMin && base+spread == sp.DamageMax {
			ours++
		}
		rivalLo, rivalHi := sp.DamageMin, sp.DamageMin+sp.DamageMax
		if rivalLo == sp.DamageMin && rivalHi == sp.DamageMax {
			rival++
		}
	}
	if ours != 9 {
		t.Errorf("our reading (base = DamageMin, spread = DamageMax - DamageMin) reproduced %d of 9 rows, want 9", ours)
	}
	if rival != 0 {
		t.Errorf("the rival reading (second column as a maximum) reproduced %d of 9 rows, want 0", rival)
	}
}

// TestLoadSpellsComplicationUsesCanonicalByteBounds fixes both authored
// edges before map loading narrows the value into sim.SpellRule.
func TestLoadSpellsComplicationUsesCanonicalByteBounds(t *testing.T) {
	rows := []synth.DataBinRow{
		spellRow("empty complication", 1, 1, 1, 1, 0, 0, 0),
		spellRow("large complication", 1, 1, 1, 1, 0, 0, 0),
	}
	rows[0].Params[0] = -1
	rows[1].Params[0] = 300
	got, err := data.LoadSpells(spells(t, rows))
	if err != nil {
		t.Fatalf("LoadSpells: %v", err)
	}
	if got[0].Complication != 0 || got[1].Complication != 255 {
		t.Fatalf("Complication bounds = %d, %d; want 0, 255", got[0].Complication, got[1].Complication)
	}
}

func TestSpellDamageFR8DivergesFromTheFloatReferenceBy(t *testing.T) {
	const (
		maxPower  = 100
		maxColumn = 1000
	)

	count := 0
	logged := false
	var firstPower, firstColumn, firstInt int32
	var firstFloat float64

	for power := int32(0); power <= maxPower; power++ {
		for x := int32(0); x <= maxColumn; x++ {
			intForm := x * (power + 30) / 30
			floatForm := math.Trunc(float64(x) * (float64(power)/30 + 1))
			if int32(floatForm) != intForm {
				count++
				if !logged {
					firstPower, firstColumn, firstInt, firstFloat = power, x, intForm, floatForm
					logged = true
				}
			}
		}
	}

	t.Logf("FR-8 divergence: %d of %d (power, column) pair(s) disagree; first at power=%d column=%d: integer form=%d float reference=%v",
		count, (maxPower+1)*(maxColumn+1), firstPower, firstColumn, firstInt, firstFloat)

	// Measured by this sweep, not assumed: power=12, column=45 is the first
	// disagreement (integer form 63, float reference 62, since
	// float64(12)/30 rounds down to 0.39999999999999991118). A change to
	// either form's arithmetic changes this count and fails here.
	const wantDivergences = 1234
	if count != wantDivergences {
		t.Fatalf("FR-8 divergence count = %d, want the measured %d", count, wantDivergences)
	}
}

func TestLoadSpellsDamagingExcludesHealAndDrainLifeByID(t *testing.T) {
	rows := make([]synth.DataBinRow, 11)
	rows[0] = spellRow("Ordinary Bolt", 3, 1, 1, 7, 4, 8, 0) // id 1: damaging
	for i, name := range []string{"Filler 2", "Filler 3", "Filler 4", "Filler 5"} {
		rows[1+i] = spellRow(name, 0, 1, 0, 0, 0, 0, 0) // ids 2-5: no damage pair
	}
	rows[5] = spellRow("Heal", 5, 2, 1, 6, 8, 16, 0) // id 6: excluded by id
	for i, name := range []string{"Filler 7", "Filler 8", "Filler 9", "Filler 10"} {
		rows[6+i] = spellRow(name, 0, 1, 0, 0, 0, 0, 0) // ids 7-10: no damage pair
	}
	rows[10] = spellRow("Drain Life", 20, 2, 1, 3, 3, 5, 0) // id 11: excluded by id

	c := spells(t, rows)
	got, err := data.LoadSpells(c)
	if err != nil {
		t.Fatalf("LoadSpells: %v", err)
	}
	if len(got) != len(rows) {
		t.Fatalf("LoadSpells returned %d spell(s), want %d", len(got), len(rows))
	}

	for id, want := range map[int]bool{
		1:  true,  // Ordinary Bolt: positive pair, not excluded
		2:  false, // filler: no damage pair
		6:  false, // Heal: positive pair, excluded by id
		10: false, // filler: no damage pair
		11: false, // Drain Life: positive pair, excluded by id
	} {
		sp := got[id-1]
		if sp.Damaging != want {
			t.Errorf("spell %d (%s) Damaging = %v, want %v", id, sp.Name, sp.Damaging, want)
		}
	}
}

func TestLoadSpellsRefusesARowTooShortToReachSlot18(t *testing.T) {
	c := spells(t, []synth.DataBinRow{{Name: "Stub", Params: []int32{1, 2, 3}}})

	_, err := data.LoadSpells(c)
	if err == nil {
		t.Fatal("LoadSpells accepted a row shorter than slot 18")
	}
	if !strings.Contains(err.Error(), "Stub") {
		t.Errorf("error %q does not name the row it refused", err)
	}
}

func TestLoadSpellsRefusesANegativeManaCost(t *testing.T) {
	c := spells(t, []synth.DataBinRow{spellRow("Free Spell", -1, 1, 1, 5, 1, 2, 0)})

	_, err := data.LoadSpells(c)
	if err == nil {
		t.Fatal("LoadSpells accepted a negative mana cost")
	}
	if !strings.Contains(err.Error(), "Free Spell") {
		t.Errorf("error %q does not name the row it refused", err)
	}
}

// TestLoadSpellsOnANilCollectionLoadsNoSpells is the "no table" state a
// caller (T4) holds without a branch of its own.
func TestLoadSpellsOnANilCollectionLoadsNoSpells(t *testing.T) {
	got, err := data.LoadSpells(nil)
	if err != nil {
		t.Fatalf("LoadSpells(nil): %v", err)
	}
	if got != nil {
		t.Fatalf("LoadSpells(nil) = %#v, want nil", got)
	}
}

func TestLoadSpellsRestorativeIsHealAloneAndNeverAlsoDamaging(t *testing.T) {
	rows := make([]synth.DataBinRow, 11)
	rows[0] = spellRow("Ordinary Bolt", 3, 1, 1, 7, 4, 8, 0)
	for i, name := range []string{"Filler 2", "Filler 3", "Filler 4", "Filler 5"} {
		rows[1+i] = spellRow(name, 0, 1, 0, 0, 0, 0, 0)
	}
	rows[5] = spellRow("Heal", 5, 2, 1, 6, 8, 16, 0)
	for i, name := range []string{"Filler 7", "Filler 8", "Filler 9", "Filler 10"} {
		rows[6+i] = spellRow(name, 0, 1, 0, 0, 0, 0, 0)
	}
	rows[10] = spellRow("Drain Life", 20, 2, 1, 3, 3, 5, 0)

	got, err := data.LoadSpells(spells(t, rows))
	if err != nil {
		t.Fatalf("LoadSpells: %v", err)
	}

	for id, want := range map[int]bool{
		1:  false, // Ordinary Bolt: damage, not healing
		2:  false, // filler: no damage pair, so no band to restore either
		6:  true,  // Heal: positive pair at the heal id
		11: false, // Drain Life: positive pair, and a third arm this build has none of
	} {
		sp := got[id-1]
		if sp.Restorative != want {
			t.Errorf("spell %d (%s) Restorative = %v, want %v", id, sp.Name, sp.Restorative, want)
		}
	}

	// And no row is both, which is what "exact complement" means here.
	for i, sp := range got {
		if sp.Damaging && sp.Restorative {
			t.Errorf("spell %d (%s) is both Damaging and Restorative", i+1, sp.Name)
		}
	}
}

func TestAHealRowWithNoDamagePairIsNeitherFlag(t *testing.T) {
	rows := make([]synth.DataBinRow, 6)
	for i := range rows[:5] {
		rows[i] = spellRow("Filler", 0, 1, 0, 0, 0, 0, 0)
	}
	rows[5] = spellRow("Heal", 5, 2, 1, 6, -1, -1, 0) // id 6, both columns empty

	got, err := data.LoadSpells(spells(t, rows))
	if err != nil {
		t.Fatalf("LoadSpells: %v", err)
	}
	if sp := got[5]; sp.Restorative || sp.Damaging {
		t.Errorf("a Heal row with no damage pair loads Damaging=%v Restorative=%v, want both false",
			sp.Damaging, sp.Restorative)
	}
}
