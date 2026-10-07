package game

import (
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/mapload"
)

func TestReleaseShieldDefinitionAndHumanCellPopulation(t *testing.T) {
	f := releaseFront(t)
	rows, defenceRows, absorptionRows := 0, 0, 0
	for row := 1; row < f.Table.Shields.Len(); row++ {
		if f.Table.Shields.EntryName(row) == "" {
			continue
		}
		rows++
		p := f.Table.Shields.EntryParams(row)
		if len(p) <= 10 {
			t.Fatalf("Shields row %d has %d runtime columns, want defence 9 and absorption 10", row, len(p))
		}
		if p[9] > 0 {
			defenceRows++
		}
		if p[10] > 0 {
			absorptionRows++
		}
	}
	rangedRows, rangedHandsTwo, rangedOtherHands := 0, 0, 0
	var rangedWitnessName string
	var rangedWitnessHands int32
	for row := 1; row < f.Table.Weapons.Len(); row++ {
		p := f.Table.Weapons.EntryParams(row)
		if len(p) <= 14 || p[5] < 10 {
			continue
		}
		rangedRows++
		if rangedWitnessName == "" {
			rangedWitnessName, rangedWitnessHands = f.Table.Weapons.EntryName(row), p[14]
		}
		if p[14] == 2 {
			rangedHandsTwo++
		} else {
			rangedOtherHands++
		}
	}

	authored, resolved, enchanted, rejected := 0, 0, 0, 0
	weaponAndShield, incompatiblePairs := 0, 0
	var witness data.ItemCode
	for row := 1; row < f.Table.Humans.Len(); row++ {
		cells := f.Table.Humans.EntryStrings(row)
		if len(cells) <= 1 || cells[1] == "" {
			continue
		}
		authored++
		worn, _, weapon, err := mapload.HumanRowEquipment(cells, f.Table)
		if err != nil {
			rejected++
			continue
		}
		if worn[1].Empty() {
			continue
		}
		if weapon != nil {
			weaponAndShield++
			if mapload.WeaponBlocksShield(weapon.Code, f.Table) {
				incompatiblePairs++
			}
		}
		resolved++
		if len(worn[1].Effects) != 0 {
			enchanted++
		}
		if witness == 0 {
			witness = data.ItemCode(worn[1].Code)
		}
	}
	if rows != 9 || defenceRows != 9 || absorptionRows != 5 {
		t.Fatalf("shield definitions = rows %d, positive defence %d, positive absorption %d; want 9/9/5",
			rows, defenceRows, absorptionRows)
	}
	if rangedRows != 1 || rangedHandsTwo != 0 || rangedOtherHands != 1 ||
		rangedWitnessName != "Flame Thrower" || rangedWitnessHands != -1 {
		t.Fatalf("ranged definitions = %d hands2 %d other %d witness %q/hands%d; want 1/0/1 Flame Thrower/hands-1",
			rangedRows, rangedHandsTwo, rangedOtherHands, rangedWitnessName, rangedWitnessHands)
	}
	if authored != 46 || resolved != 46 || enchanted != 5 || rejected != 0 ||
		weaponAndShield != 46 || incompatiblePairs != 0 {
		t.Fatalf("human shield cells = authored %d resolved %d enchanted %d rejected %d pairs %d incompatible %d; want 46/46/5/0/46/0",
			authored, resolved, enchanted, rejected, weaponAndShield, incompatiblePairs)
	}
	piece, err := data.ShieldFromCode(witness, f.Table.Shapes, f.Table.Materials, f.Table.Shields)
	if err != nil {
		t.Fatalf("ShieldFromCode(first live cell): %v", err)
	}
	if witness != data.ItemCode(0x8207) || piece.Defence != 10 || piece.Absorption != 1 {
		t.Fatalf("first live shield = %#04x protection %d/%d, want 0x8207 10/1",
			uint16(witness), piece.Defence, piece.Absorption)
	}
	if name, ok := f.Table.Names.NameFor(witness); !ok || name == "" {
		t.Fatalf("installed item-name table has no name for shield %#04x", uint16(witness))
	}
	t.Logf("shield census: definitions=%d defence=%d absorption=%d ranged=%d hands2=%d other_hands=%d ranged_witness=%q/hands%d human_cells=%d resolved=%d enchanted=%d rejected=%d pairs=%d incompatible=%d witness=%#04x stats=%d/%d",
		rows, defenceRows, absorptionRows, rangedRows, rangedHandsTwo, rangedOtherHands, rangedWitnessName, rangedWitnessHands, authored, resolved, enchanted, rejected, weaponAndShield, incompatiblePairs,
		uint16(witness), piece.Defence, piece.Absorption)
}
