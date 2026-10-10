package mapload_test

import (
	"testing"

	"againrom/pkg/formats/alm"
	"againrom/pkg/mapload"
)

const slotUnitServerID = 55

func wideUnitRow(serverID int32) []int32 {
	p := make([]int32, slotUnitServerID+1)
	for i := range p {
		p[i] = -1
	}
	p[slotUnitServerID] = serverID
	return p
}

func TestSecondGameResolvesByServerID(t *testing.T) {
	tbl := armTable(t)
	tbl.Units = defCollection{
		{},
		{name: "early", params: wideUnitRow(2000)},
		{name: "", params: wideUnitRow(2001)},
		{name: "named", params: wideUnitRow(2001)},
		{name: "late", params: wideUnitRow(2000)},
	}
	tbl.UnitKeys, tbl.SpellArms, tbl.FreshPlayers = mapload.ServerUnitKeys, mapload.SecondGameSpellArms, mapload.NoFreshPlayers
	for _, tc := range []struct {
		name  string
		unit  alm.Unit
		arm   mapload.Arm
		index int
	}{
		{"the earlier of two entries on one id", alm.Unit{ServerID: 2000}, mapload.ArmUnits, 1},
		{"a nameless entry is passed", alm.Unit{ServerID: 2001}, mapload.ArmUnits, 3},
		{"the first game's class key is not read", alm.Unit{ClassID: 0x40, ServerID: 2000}, mapload.ArmUnits, 1},
		{"an id naming nothing", alm.Unit{ServerID: 9}, mapload.ArmUnits, 0},
		{"the person flag takes the humans id", alm.Unit{Flags: 0x10, ServerID: 901}, mapload.ArmServerID, 3},
		{"the person flag with an id naming nothing", alm.Unit{Flags: 0x10, ServerID: 2000}, mapload.ArmServerID, 0},
	} {
		got := mapload.Resolve(tc.unit, tbl)
		if got.Arm != tc.arm || got.Index != tc.index {
			t.Errorf("%s: Resolve = {%v %d}, want {%v %d}", tc.name, got.Arm, got.Index, tc.arm, tc.index)
		}
	}
	tbl.UnitKeys, tbl.SpellArms, tbl.FreshPlayers = mapload.ClassUnitKeys, nil, mapload.SlotPlayers
	if got := mapload.Resolve(alm.Unit{ClassID: 0x40, ClassSubID: 1, ServerID: 2000}, tbl); got.Arm != mapload.ArmUnits {
		t.Errorf("the first game's table resolved by the second game's key: %+v", got)
	}
}
