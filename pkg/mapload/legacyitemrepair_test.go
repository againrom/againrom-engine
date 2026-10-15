package mapload

import (
	"testing"

	"againrom/pkg/sim"
)

type legacyRepairScale struct {
	rows   int
	index  int
	factor float64
}

func (s legacyRepairScale) Len() int           { return s.rows }
func (legacyRepairScale) EntryName(int) string { return "" }
func (s legacyRepairScale) EntryDoubles(i int) []float64 {
	row := []float64{1, 1, 1, 1, 1, 1, 1, 1, 1}
	if i == s.index {
		row[2] = s.factor
	}
	return row
}

func TestRepriceItemInstanceUsesTheLiveBaseAndCompleteRepairedEffects(t *testing.T) {
	weapons := make(itemTestCollection, 21)
	weapons[20] = itemTestRow{name: "Short Bow", params: []int32{0, 0, 207}}
	magic := make(itemTestCollection, 13)
	magic[12].params = []int32{10}
	table := &Table{
		Shapes:    legacyRepairScale{rows: 4, index: 3, factor: 1},
		Materials: legacyRepairScale{rows: 9, index: 8, factor: 267.0 / 207.0},
		Weapons:   weapons,
		Magic:     magic,
	}
	item := sim.ItemInstance{Code: 0x8134, Kind: 2, Price: 207, Effects: []sim.ItemEffect{
		{Kind: 12, Operand: 5},
	}}
	if got := RepriceItemInstance(item, table); got != 6106 {
		t.Fatalf("repaired mission-70 Short Bow value = %d, want 6106", got)
	}
	if got := RepriceItemInstance(item, nil); got != 207 {
		t.Fatalf("incomplete table value = %d, want stored 207 preserved", got)
	}
}
