package game

import (
	"testing"

	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

// TestReleaseWorldAndTownWornChangesGiveTheSameStaff: the first town
// companion's installed staff, taken off and worn again through the World
// holder (unequip and equip commands over the installed Spells rules) and
// through the town holder over the installed table, is the same item state at
// each step: no Spell in the pack, the built Spell when worn (ITEM-187).
// Control: the staff holds its Spell before, and the take-off changes it.
func TestReleaseWorldAndTownWornChangesGiveTheSameStaff(t *testing.T) {
	f, i := townCompanionStaff(t, false)
	staff := f.Carried[i].WornItems[0].Clone()
	if !staff.SourceEquipment.Spell.Present {
		t.Fatalf("control: the companion's staff holds %+v", staff.SourceEquipment.Spell)
	}
	worldOff, worldOn := worldTakeOffAndWear(t, worldWearing(t, staff.Clone(), mapload.SpellRules(f.Table)))
	townOff := takeOffTown(townWornHolder{table: f.Table}, 1, staff.Clone())
	townOn := wearTown(f.Table, 1, townOff.Clone())
	if townOff.SourceEquipment.Spell.Present || !sim.ItemEqual(worldOff, townOff) {
		t.Fatalf("taken off: World %+v, town %+v", worldOff.SourceEquipment, townOff.SourceEquipment)
	}
	if !townOn.SourceEquipment.Spell.Present || !sim.ItemEqual(worldOn, townOn) {
		t.Fatalf("worn again: World %+v, town %+v", worldOn.SourceEquipment, townOn.SourceEquipment)
	}
	if townOn.SourceEquipment.Spell != staff.SourceEquipment.Spell {
		t.Fatalf("worn again the staff holds %+v, joined with %+v", townOn.SourceEquipment.Spell, staff.SourceEquipment.Spell)
	}
}
