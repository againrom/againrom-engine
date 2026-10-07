package mapload_test

import (
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/mapload"
)

// slotWeaponRange is the Weapons row's own range column, transcribed here for
// the reason every other slot number in this suite is: a test asserting that
// a resolved reach came off this column must not read the column's position
// out of the code it is testing. human_test.go's weaponRow leaves this cell
// empty on purpose — that suite is not testing reach at all — so this file
// writes its own rows.
const slotWeaponRange = 0xb

// srcBowRange is the range this file's bow states. It is NOT 1, the
// constructor's own floor, so a build that never read the column would still
// produce it — and it is not the sight range's own 5 or the health's 30
// either, so a field crossed with a neighbour shows rather than agreeing by
// accident.
const srcBowRange int32 = 6

// reachWeapons is the weapon collection every test below resolves against: a
// bow whose attack type is RANGED — 0xb, at meleeAttackTypes and past it —
// and states srcBowRange, and a dagger whose attack type is melee and whose
// own range cell is left empty.
func reachWeapons() defCollection {
	bow := weaponRow(0xb, 30, 40, 100, 0, -1, -1)
	bow[slotWeaponRange] = srcBowRange
	dagger := weaponRow(data.SkillBlade, 3, 5, 4, 1, -1, -1)
	return defCollection{
		{},
		{name: "Bow", params: bow},
		{name: "Dagger", params: dagger},
	}
}

// reachTable resolves key 0x40 — sightrange_test.go's own creature key — to
// one units row carrying the given trailing equipment strings, against the
// item collections handed in. It is that file's `creature` table with the
// two strings this file is about added, so the two suites are read as the
// same placement asked two different questions.
func reachTable(equipment []string, weapons data.Collection, shapes, materials data.ScaleTable) *mapload.Table {
	return &mapload.Table{
		Units: defCollection{{}, {name: "k40", params: defRow(map[int]int32{
			slotUnitType: 0x40, slotUnitFace: 1, slotHealthMax: 30,
		}), strings: equipment}},
		Humans: defCollection{},
		Shapes: shapes, Materials: materials, Weapons: weapons,
	}
}

func TestAUnitsReachIsItsRowsFirstResolvingWeapon(t *testing.T) {
	t.Parallel()

	weapons := reachWeapons()
	for _, tc := range []struct {
		name      string
		equipment []string
		want      uint8
	}{
		{"a bow, alone", []string{"Bow"}, uint8(srcBowRange)},
		{"armour first, then the bow", []string{"Mail", "Bow"}, uint8(srcBowRange)},
		{"a melee weapon whose row states no range", []string{"Dagger"}, 1},
		{"no equipment string at all", nil, 1},
		{"an unresolvable name", []string{"Nonexistent"}, 1},
		{"empty cells only", []string{"", ""}, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tbl := reachTable(tc.equipment, weapons, identityScale(), identityScale())
			if got := srcLoad(t, srcMap(0x40), tbl).Reach; got != tc.want {
				t.Errorf("reach = %d, want %d", got, tc.want)
			}
		})
	}

	// The bow and the constructor's floor must actually differ, or every case
	// above that names 1 passes whether or not the loader read the string at
	// all.
	if uint8(srcBowRange) == 1 {
		t.Fatal("the fixture's bow states the constructor's own floor; this file proves nothing")
	}
}

func TestAUnitWithNoItemCollectionHasReach1(t *testing.T) {
	t.Parallel()

	weapons := reachWeapons()
	for _, tc := range []struct {
		name              string
		shapes, materials data.ScaleTable
		weapons           data.Collection
	}{
		{"no shapes", nil, identityScale(), weapons},
		{"no materials", identityScale(), nil, weapons},
		{"no weapons", identityScale(), identityScale(), nil},
		{"none of the three", nil, nil, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tbl := reachTable([]string{"Bow"}, tc.weapons, tc.shapes, tc.materials)
			if got := srcLoad(t, srcMap(0x40), tbl).Reach; got != 1 {
				t.Errorf("reach = %d, want 1", got)
			}
		})
	}
}
