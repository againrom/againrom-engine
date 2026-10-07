package mapload_test

// Where a placement's DWELL comes from (0089 AC-18): the column of whichever
// collection resolved it, and the constructor's own default where the row leaves
// the cell empty or where nothing resolved at all.
//
// The three sources have to be told apart, so no two of the numbers below agree
// and none of them is the default.

import (
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/formats/alm"
	"againrom/pkg/mapload"
)

// The two collections' slots for the column, spelled out here rather than read
// back through the definition tier.
const (
	slotUnitDying  = 33
	slotHumanDying = 23
)

func TestAPlacementCarriesItsOwnRowsDyingTime(t *testing.T) {
	const unitDwell, humanDwell = 47, 19
	def := data.UnitDefaults().DyingTime
	if def == unitDwell || def == humanDwell || unitDwell == humanDwell {
		t.Fatalf("the constructor's default dying time is %d; this fixture cannot tell the "+
			"three sources apart", def)
	}

	m := &alm.Map{
		Width: 40, Height: 40,
		Units: []alm.Unit{
			{X: 0x0C80, Y: 0x0C80, ClassID: 0x40}, // units, matched
			{X: 0x0D80, Y: 0x0C80, ClassID: 7},    // humans, by type
			{X: 0x0E80, Y: 0x0C80, ClassID: 0x55}, // units key, matching nothing
		},
	}
	tbl := &mapload.Table{
		Units: defCollection{{}, {name: "beast", params: defRow(map[int]int32{
			slotUnitType: 0x40, slotUnitFace: 0, slotHealthMax: 30,
			slotUnitDying: unitDwell})}},
		Humans: defCollection{{}, {name: "person", params: humanRow(map[int]int32{
			slotHumanType: 7, slotHumanHealth: 30, slotHumanDying: humanDwell})}},
	}

	w, err := mapload.FromALMWith(m, tbl, mapload.DifficultyNormal)
	if err != nil {
		t.Fatalf("FromALMWith: %v", err)
	}
	got := w.Entities()
	for i, want := range []int32{unitDwell, humanDwell, def} {
		if got[i].DyingTime != want {
			t.Errorf("placement %d carries a dying time of %d, want %d",
				i, got[i].DyingTime, want)
		}
	}
}

// TestAnEmptyDyingCellLeavesTheConstructorsOwn is the other half of the column's
// law, on both collections: an empty cell stores nothing, and what stands is the
// default — not a zero, which would be a body torn down where it fell.
func TestAnEmptyDyingCellLeavesTheConstructorsOwn(t *testing.T) {
	def := data.UnitDefaults().DyingTime
	if def <= 0 {
		t.Fatalf("the constructor's default dying time is %d, which is no dwell at all", def)
	}

	m := &alm.Map{
		Width: 40, Height: 40,
		Units: []alm.Unit{
			{X: 0x0C80, Y: 0x0C80, ClassID: 0x40},
			{X: 0x0D80, Y: 0x0C80, ClassID: 7},
		},
	}
	tbl := &mapload.Table{
		Units: defCollection{{}, {name: "beast", params: defRow(map[int]int32{
			slotUnitType: 0x40, slotUnitFace: 0, slotHealthMax: 30})}},
		Humans: defCollection{{}, {name: "person", params: humanRow(map[int]int32{
			slotHumanType: 7, slotHumanHealth: 30})}},
	}

	w, err := mapload.FromALMWith(m, tbl, mapload.DifficultyNormal)
	if err != nil {
		t.Fatalf("FromALMWith: %v", err)
	}
	for i, e := range w.Entities() {
		if e.DyingTime != def {
			t.Errorf("placement %d carries a dying time of %d, want the constructor's %d",
				i, e.DyingTime, def)
		}
	}
}
