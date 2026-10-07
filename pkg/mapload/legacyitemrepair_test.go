package mapload

import (
	"testing"

	"againrom/pkg/data"
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

func TestRepairLegacyPartyCoversTownItemsAndExactHiredRows(t *testing.T) {
	weapons := make(itemTestCollection, 21)
	weapons[20] = itemTestRow{name: "Short Bow", params: []int32{0, 0, 207}}
	magic := make(itemTestCollection, 13)
	magic[12].params = []int32{10}
	humans := make(itemTestCollection, 3)
	human := make([]int32, data.MinHumanRow)
	for i := range human {
		human[i] = -1
	}
	human[7] = 11 // RotationSpeed, independently pinned from HumanDef's row order.
	humans[2] = itemTestRow{name: "NPC13_2", params: human}
	units := make(itemTestCollection, 3)
	unit := make([]int32, 41)
	for i := range unit {
		unit[i] = -1
	}
	unit[9] = 8 // RotationSpeed in the Units row order.
	units[2] = itemTestRow{name: "Catapult", params: unit}
	table := &Table{
		Units: units, Humans: humans,
		Shapes:    legacyRepairScale{rows: 4, index: 3, factor: 1},
		Materials: legacyRepairScale{rows: 9, index: 8, factor: 267.0 / 207.0},
		Weapons:   weapons, Magic: magic,
	}
	legacyAttack := sim.ItemInstance{Code: 0x8134, Kind: 2, Price: 267,
		Effects: []sim.ItemEffect{{Kind: 12, Operand: 0x0000fb05}}}
	legacyWorn := sim.ItemInstance{Code: 0x8134, Kind: 2, Price: 267,
		Effects: []sim.ItemEffect{{Kind: 12, Operand: 0x0000fb05}}}
	var equipped [sim.EquipSlots]sim.ItemInstance
	equipped[0] = legacyWorn
	party := []PartyMember{
		{Name: "NPC13_2", MercenaryType: 13, Carry: &Carry{
			ItemInstances: []sim.ItemInstance{legacyAttack}, EquippedItems: equipped}},
		{Name: "Catapult", MercenaryType: 1},
		{Name: "not-a-row", MercenaryType: 13},
	}

	got := RepairLegacyParty(party, table)
	if got != (LegacyPartyRepairs{ItemEffects: 2, HiredRotationSpeeds: 2}) {
		t.Fatalf("repairs = %+v, want two effects and two exact row speeds", got)
	}
	if party[0].HiredRotationSpeed != 11 || party[1].HiredRotationSpeed != 8 ||
		party[2].HiredRotationSpeed != 0 {
		t.Fatalf("hired speeds = %d/%d/%d, want 11/8/0",
			party[0].HiredRotationSpeed, party[1].HiredRotationSpeed, party[2].HiredRotationSpeed)
	}
	carried := MemberCarriedItems(party[0], table)
	worn := MemberItemEquipment(party[0], table)
	for where, item := range map[string]sim.ItemInstance{"carried": carried[0], "worn": worn[0]} {
		if item.Effects[0].Operand != 5 || item.Price != 6106 {
			t.Errorf("%s item = %+v, want repaired +5 and live value 6106", where, item)
		}
	}
	if party[0].Carry.Items[0] != legacyAttack.Code || party[0].Carry.Equipped[0] != legacyWorn.Code {
		t.Fatalf("compatibility projections = %#x/%#x", party[0].Carry.Items[0], party[0].Carry.Equipped[0])
	}
	if second := RepairLegacyParty(party, table); second != (LegacyPartyRepairs{}) {
		t.Fatalf("second repair = %+v, want inert", second)
	}
}
