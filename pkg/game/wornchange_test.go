package game

import (
	"testing"

	"againrom/pkg/base"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

var wornStaffSpell = sim.SourceItemSpell{Present: true, ID: 14, Range: 7, ManaCost: 80}

func TestTownWearConstructsUnboundSpellWeapon(t *testing.T) {
	table, weapon := staffTooltipTable(t, 14, "Fire Arrow", 20, 7, 3, 5, 0)
	table.Shapes, table.Materials = unitScale(16), unitScale(16)
	item := mapload.ItemInstanceFromCode(uint16(weapon.Code), table)
	item.Effects = []sim.ItemEffect{{Kind: 41, Operand: 14 | 20<<16}}
	if item.SourceEquipment.Class != 0 || item.WeightPresent {
		t.Fatal("control: shelf item is already constructed")
	}
	got := wearTown(table, 1, item.Clone())
	if got.SourceEquipment.Class != sim.SourceWeapon || !got.SourceEquipment.Spell.Present || got.SourceEquipment.Spell.ID != 14 || !got.WeightPresent {
		t.Fatalf("town equip: %+v", got)
	}
	plain := item.Clone()
	plain.Effects = nil
	if kept := wearTown(table, 1, plain.Clone()); !sim.ItemEqual(kept, plain) {
		t.Fatal("equip changed an item without castSpell")
	}
	missing := item.Clone()
	missing.Effects[0].Operand = 99
	if kept := wearTown(table, 1, missing.Clone()); !sim.ItemEqual(kept, missing) {
		t.Fatal("failed Spell build changed the item")
	}
	second := base.GameROM2.Edition()
	table.Edition = &second
	if kept := wearTown(table, 1, item.Clone()); !sim.ItemEqual(kept, item) {
		t.Fatalf("second-game equip changed the item: %+v", kept)
	}
}

// wornStaff is a classed Staff whose first castSpell Effect names spell 14;
// held selects whether it holds the Spell an equip builds.
func wornStaff(held bool) sim.ItemInstance {
	item := sim.ItemInstance{Code: 0x0102, Kind: 2, WeightPresent: true, Weight: 2,
		Effects: []sim.ItemEffect{{Kind: 12, Operand: 2}, {Kind: 41, Operand: 14 | 20<<16}},
		SourceEquipment: sim.SourceEquipment{Class: sim.SourceWeapon, DefinitionRow: 1, OwnKind: 5,
			Definition: sim.SourceWeaponDefinition{Present: true, AttackType: 1, Hands: 2, Charge: 8, Relax: 4}}}
	if held {
		item.SourceEquipment.Spell = wornStaffSpell
	}
	return item
}

// worldWearing is a first-game World whose party actor 1 wears staff in slot
// 1, with spells as its Spells rules.
func worldWearing(t *testing.T, staff sim.ItemInstance, spells []sim.SpellRule) *sim.World {
	t.Helper()
	e := sim.Entity{ID: 1, X: 2, Y: 2, HP: 100, MaxHP: 100, DyingTime: 200, Owner: sim.SelfSlot}
	w, err := sim.NewStockedSpelledWorld(5, sim.Bounds{Width: 8, Height: 8}, sim.ModeCanonical, sim.Terrain{}, []sim.Entity{e}, nil, sim.Relations{}, nil,
		[]sim.Stock{{ID: 1, EquippedItems: [sim.EquipSlots]sim.ItemInstance{0: staff}}}, spells)
	if err != nil {
		t.Fatal(err)
	}
	return w
}

// worldTakeOffAndWear takes the World actor's staff off into the pack and
// wears it again, returning the item in the pack and the item worn.
func worldTakeOffAndWear(t *testing.T, w *sim.World) (off, on sim.ItemInstance) {
	t.Helper()
	sim.Step(w, []sim.Command{sim.Unequip(1, 1)})
	pack, _ := w.CarriedStacks(1)
	if len(pack) != 1 {
		t.Fatalf("the World pack after the take-off holds %d stacks", len(pack))
	}
	off = pack[0].Instance()
	sim.Step(w, []sim.Command{sim.Equip(1, 0, 1)})
	worn, _ := w.EquippedItems(1)
	return off, worn[0]
}

// TestWorldAndTownTakeOffLeaveTheSameStaff: a staff holding its Spell, taken
// off through the World holder (the unequip command) and through the town
// holder, is one item state in the first game and holds no Spell (ITEM-187).
// Controls: the staff held its Spell before; the town's second-game row keeps
// it.
func TestWorldAndTownTakeOffLeaveTheSameStaff(t *testing.T) {
	staff := wornStaff(true)
	worldOff, _ := worldTakeOffAndWear(t, worldWearing(t, staff.Clone(), nil))
	townOff := takeOffTown(townWornHolder{table: &mapload.Table{}}, 1, staff.Clone())
	if !staff.SourceEquipment.Spell.Present {
		t.Fatal("control: the staff holds no Spell before the take-off")
	}
	if worldOff.SourceEquipment.Spell.Present || !sim.ItemEqual(worldOff, townOff) {
		t.Fatalf("World take-off %+v, town take-off %+v", worldOff.SourceEquipment, townOff.SourceEquipment)
	}
	second := base.GameROM2.Edition()
	if kept := takeOffTown(townWornHolder{table: &mapload.Table{Edition: &second}}, 1, staff.Clone()); kept.SourceEquipment.Spell != wornStaffSpell {
		t.Fatalf("control: the second game's town take-off left %+v", kept.SourceEquipment.Spell)
	}
}

// TestTownTakeOffDropsOnlyItsCityNodeSpellEdge: with the city topology, the
// town take-off drops the Spell edge of the item's own node and no other.
// Control: the second game's row keeps the edge.
func TestTownTakeOffDropsOnlyItsCityNodeSpellEdge(t *testing.T) {
	graph := func() *cityObjectTopology {
		return &cityObjectTopology{Items: []cityItemTopology{{ID: 5, Spell: 9}, {ID: 6, Spell: 10}}}
	}
	g := graph()
	off := takeOffTown(townWornHolder{table: &mapload.Table{}, g: g, id: 5}, 1, wornStaff(true))
	if off.SourceEquipment.Spell.Present || g.Items[0].Spell != 0 || g.Items[1].Spell != 10 {
		t.Fatalf("take-off: item %+v, edges %d and %d", off.SourceEquipment.Spell, g.Items[0].Spell, g.Items[1].Spell)
	}
	second := base.GameROM2.Edition()
	g = graph()
	takeOffTown(townWornHolder{table: &mapload.Table{Edition: &second}, g: g, id: 5}, 1, wornStaff(true))
	if g.Items[0].Spell != 9 {
		t.Fatal("control: the second game's take-off dropped the edge")
	}
}

// TestTownTakeOffClearsTheEdgeOfAValueWithoutASpell: any class.
func TestTownTakeOffClearsTheEdgeOfAValueWithoutASpell(t *testing.T) {
	g := &cityObjectTopology{Items: []cityItemTopology{{ID: 5, Spell: 9}, {ID: 6, Spell: 10}}}
	takeOffTown(townWornHolder{table: &mapload.Table{}, g: g, id: 5}, 1, sim.ItemInstance{Code: 0x3142})
	if g.Items[0].Spell != 0 || g.Items[1].Spell != 10 {
		t.Fatalf("edges %d and %d", g.Items[0].Spell, g.Items[1].Spell)
	}
	second := base.GameROM2.Edition()
	g = &cityObjectTopology{Items: []cityItemTopology{{ID: 5, Spell: 9}}}
	takeOffTown(townWornHolder{table: &mapload.Table{Edition: &second}, g: g, id: 5}, 1, sim.ItemInstance{Code: 0x3142})
	if g.Items[0].Spell != 9 {
		t.Fatal("control: the second game's take-off dropped the edge")
	}
}
