package mapload_test

// Where a placement's rate comes from (AC-7).
//
// The one thing this file has to separate is the two sources: a matched units
// entry's own column, and the base constructor's default for everything else. A
// fixture in which the two agreed would pass whichever the loader read.

import (
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/formats/alm"
	"againrom/pkg/mapload"
)

// slotSpeed is the Units row slot the speed column occupies, transcribed here
// beside the others rather than read back through the definition tier.
const slotSpeed = 8

// speedRow is one units row carrying a named speed, and nothing else this test
// reads: a type key to match on, a face of zero, a health maximum so the row is
// a whole definition, and the column itself.
func speedRow(typeID, speed int32) []int32 {
	return defRow(map[int]int32{
		slotUnitType: typeID, slotUnitFace: 0, slotHealthMax: 30, slotSpeed: speed})
}

// TestOnlyAMatchedUnitsEntryYieldsItsOwnSpeed is AC-7 and SC-7.
//
// The map carries every arm — npc, server id, humans, a units key matching
// nothing, and a units key matching a row — and the table's two units rows carry
// speeds that are neither each other's nor the default. So a loader reading the
// wrong row, or reading a column for an arm that resolves nothing, lands on a
// number this table names.
func TestOnlyAMatchedUnitsEntryYieldsItsOwnSpeed(t *testing.T) {
	const slow, quick = 8, 35
	def := data.UnitDefaults().Speed
	if def == slow || def == quick {
		t.Fatalf("the constructor's default speed is %d, which one of the two rows also carries — "+
			"this fixture could not tell the column from the default", def)
	}

	m := &alm.Map{
		Width: 40, Height: 40,
		Units: []alm.Unit{
			{X: 0x0C80, Y: 0x0C80, ClassID: 7, Flags: 1},   // npc
			{X: 0x0D80, Y: 0x0C80, ClassID: 7, DefID: 900}, // server id
			{X: 0x0E80, Y: 0x0C80, ClassID: 7},             // humans, by type
			{X: 0x0F80, Y: 0x0C80, ClassID: 0x55},          // units, no match
			{X: 0x1080, Y: 0x0C80, ClassID: 0x40},          // units, matched — slow
			{X: 0x1180, Y: 0x0C80, ClassID: 0x41},          // units, matched — quick
		},
	}
	tbl := &mapload.Table{Units: defCollection{
		{},
		{name: "slow", params: speedRow(0x40, slow)},
		{name: "quick", params: speedRow(0x41, quick)},
	}}

	w, err := mapload.FromALMWith(m, tbl, mapload.DifficultyNormal)
	if err != nil {
		t.Fatalf("FromALMWith: %v", err)
	}
	want := []int32{def, def, def, def, slow, quick}
	for i, e := range w.Entities() {
		if e.Speed != want[i] {
			t.Errorf("placement %d carries speed %d, want %d", i, e.Speed, want[i])
		}
	}

	for i, e := range w.Entities() {
		if e.Speed <= 0 {
			t.Errorf("placement %d is unrated at speed %d", i, e.Speed)
		}
	}

	// And with NO table nothing resolves, so every one of the six takes the
	// default — which is what says the column is reached through the resolution
	// and not through the placement.
	for i, e := range mapload.FromALM(m).Entities() {
		if e.Speed != def {
			t.Errorf("table-free placement %d carries speed %d, want the default %d", i, e.Speed, def)
		}
	}
}

// TestASpeedAndAHealthComeOffOneResolution: the two are read in one arm, so no
// placement can take a class's health and another's speed, or one and not the
// other. Measured by moving the health of the matched row and requiring the
// speed to move with the same placements and no others.
func TestASpeedAndAHealthComeOffOneResolution(t *testing.T) {
	m := &alm.Map{
		Width: 40, Height: 40,
		Units: []alm.Unit{
			{X: 0x0C80, Y: 0x0C80, ClassID: 0x40}, // matched
			{X: 0x0D80, Y: 0x0C80, ClassID: 0x55}, // units, no match
		},
	}
	row := defRow(map[int]int32{
		slotUnitType: 0x40, slotUnitFace: 0, slotHealthMax: 77, slotSpeed: 21})
	tbl := &mapload.Table{Units: defCollection{{}, {name: "one", params: row}}}

	w, err := mapload.FromALMWith(m, tbl, mapload.DifficultyNormal)
	if err != nil {
		t.Fatalf("FromALMWith: %v", err)
	}
	got := w.Entities()
	if got[0].Speed != 21 || got[0].MaxHP != 77 {
		t.Errorf("the matched placement is %d/%d speed/health, want 21/77", got[0].Speed, got[0].MaxHP)
	}
	if got[1].Speed != data.UnitDefaults().Speed || got[1].MaxHP != mapload.SpawnHP {
		t.Errorf("the unmatched placement is %d/%d speed/health, want the two provisional values %d/%d",
			got[1].Speed, got[1].MaxHP, data.UnitDefaults().Speed, mapload.SpawnHP)
	}
}

// TestADefaultSpeedIsTheConstructorsOwn pins the exported constant against the
// definition tier's own defaults, so a change to either is a failure here rather
// than a silent re-rating of every unresolved placement on every map.
func TestADefaultSpeedIsTheConstructorsOwn(t *testing.T) {
	if mapload.DefaultSpeed != data.UnitDefaults().Speed {
		t.Errorf("DefaultSpeed is %d and the constructor's own is %d",
			mapload.DefaultSpeed, data.UnitDefaults().Speed)
	}
	if mapload.DefaultSpeed <= 0 {
		t.Errorf("DefaultSpeed is %d, which is a mover with no rate at all", mapload.DefaultSpeed)
	}
	// A world built from it must be RATED, which is the property the constant
	// exists for, and it is asked of a world rather than of the number.
	m := &alm.Map{Width: 40, Height: 40, Units: []alm.Unit{{X: 0x0C80, Y: 0x0C80, ClassID: 7}}}
	if e := mapload.FromALM(m).Entities()[0]; e.Speed != mapload.DefaultSpeed {
		t.Errorf("a table-free placement carries speed %d, want %d", e.Speed, mapload.DefaultSpeed)
	}
}
