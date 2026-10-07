package game

import (
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

func currentChildGraph1172(t *testing.T) (*sim.SavedObjects, sim.SavedItemObject) {
	t.Helper()
	value := sim.ItemStack{Code: 0x0111, Count: 3, Kind: 2, Price: 91, WeightPresent: true, Weight: -7,
		Effects: []sim.ItemEffect{{Kind: 3, Mode: 2, Operand: 0x81234567}, {Kind: 3, Mode: 2, Operand: 0x81234567}},
		SourceEquipment: sim.SourceEquipment{Class: sim.SourceWeapon, DefinitionRow: 9, OwnKind: 4,
			Spell: sim.SourceItemSpell{Present: true, ID: 12, Range: 8, Defensive: 1, ManaCost: 17}}}
	value.SourceEquipment.Attack[3], value.SourceEquipment.Defence[7] = 28, 51
	next := sim.SavedObjectID(1)
	row, effects, spell, err := sim.ConstructSavedItem(value, func() (sim.SavedObjectID, sim.SavedObjectToken, error) {
		id := next
		next++
		return id, sim.SavedObjectToken{Identity: 0x71000000 + uint32(id)*16, RuntimeID: uint32(id) + 50,
			T0C: 9, T0E: 33, T08: 17, T18: 29, T1C: 43, Position: [12]byte{3, 5, 7, 11}}, nil
	})
	if err != nil || spell == nil {
		t.Fatal(err)
	}
	row.InFlight = 0
	row.F45, row.F46, row.F47, row.F48 = 19, 23, 29, 0x3125
	r := &sim.SavedObjects{Version: sim.SavedObjectsVersion, NextID: next, Items: []sim.SavedItemObject{row}, Effects: effects, Spells: []sim.SavedSpellObject{*spell},
		Containers: []sim.SavedObjectContainer{{Owner: sim.SavedObjectOwner{Kind: sim.SavedOwnerActorPack, Entity: 7}, Present: true, InsertIndex: 9, Accumulator: -17, Items: []sim.SavedObjectID{row.ID}}}}
	if err := r.ValidateNoInFlight(); err != nil {
		t.Fatal(err)
	}
	return r, row
}

func TestCityHoldings1172CurrentChildFieldsAndLosses(t *testing.T) {
	r, row := currentChildGraph1172(t)
	g, err := cityItemGraphFromWorld(r, 7)
	if err != nil {
		t.Fatal(err)
	}
	if len(g.Objects) != 4 || len(g.Inventory) != 1 || g.InsertIndex != 9 || g.Accumulator != -17 {
		t.Fatal("current closed graph lost nodes or stored bookkeeping")
	}
	item := &g.Objects[g.Inventory[0]-1]
	lootToken1172(t, *item, row.Token)
	for name, value := range map[string]uint32{"F40": 0x0111, "F42": 3, "F44": 2, "F45": 19, "F46": 23, "F47": 29, "F48": 0x3125, "F4A": 0xfff9} {
		if itemObjectValue1115(t, *item, name) != value {
			t.Fatal("current retained Item field lost", name)
		}
	}
	refs, _ := savedObjectRefs(item, "Effects")
	if len(refs) != 2 || refs[0] == refs[1] {
		t.Fatal("equal-valued child identities collapsed")
	}
	for i, ref := range refs {
		child := g.Objects[ref-1]
		lootToken1172(t, child, r.Effects[i].Token)
		if itemObjectValue1115(t, child, "E3C") != 3 || itemObjectValue1115(t, child, "E3D") != 2 || itemObjectValue1115(t, child, "E40") != 0x81234567 {
			t.Fatal("ordered current Effect operands lost")
		}
	}
	spells, _ := savedObjectRefs(item, "WeaponSpell")
	if len(spells) != 1 || spells[0] == 0 || itemObjectValue1115(t, g.Objects[spells[0]-1], "This") != r.Spells[0].This {
		t.Fatal("owned Spell identity lost")
	}
	value := row.Value.Clone()
	value.ObjectID = 0
	p := mapload.PartyMember{Carry: &mapload.Carry{OrderedStacks: []sim.ItemStack{value}, ItemInstances: []sim.ItemInstance{value.Instance(), value.Instance(), value.Instance()},
		LiveLoad: &sim.ActorLoadSnapshot{}}}
	p.Carry.LiveLoad.Inventory.ContainerPresent = true
	p.Carry.LiveLoad.Inventory.InsertIndex = 9
	p.Carry.LiveLoad.Inventory.Accumulator = -17
	if err := checkCityItemGraph(g, p); err != nil {
		t.Fatal("current graph/stack admission", err)
	}
	lifetime := cloneCityItemGraph(g)
	savedObjectSetValue(&lifetime.Objects[refs[0]-1], "E0C", 1)
	if err := sav.ValidateCityItemGraph(lifetime); err != nil {
		t.Fatal("lifetime control must be structurally valid", err)
	}
	if err := checkCityItemGraph(lifetime, p); err == nil {
		t.Fatal("unsupported persisted Effect lifetime was admitted")
	}
	for name, edit := range map[string]func(*sim.SavedObjects){
		"foreign owner":               func(r *sim.SavedObjects) { r.Containers[0].Owner.Entity = 8 },
		"collapsed child":             func(r *sim.SavedObjects) { r.Items[0].Effects[1] = r.Items[0].Effects[0] },
		"stale Spell":                 func(r *sim.SavedObjects) { r.Items[0].Spell = 999 },
		"unsupported Effect lifetime": func(r *sim.SavedObjects) { r.Effects[0].E0C = 1 },
	} {
		t.Run(name, func(t *testing.T) {
			bad := r.Clone()
			edit(bad)
			if _, err := cityItemGraphFromWorld(bad, 7); err == nil {
				t.Fatal("current graph loss was admitted")
			}
		})
	}
}
