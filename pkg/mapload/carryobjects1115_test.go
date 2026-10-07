package mapload_test

import (
	"bytes"
	"encoding/binary"
	"reflect"
	"testing"

	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

// The fixture enters through the exact owner/ordinal import door. Equal
// source stacks have different identities; an ID0 slot coexists with them.
func carryObjectsWorld1115(t *testing.T) *sim.World {
	t.Helper()
	source := sim.SourceActor{
		Class: 1, Stats: [14]uint16{9, 11, 13, 17, 19, 0xfff9, 77, 300, 50, 100, 20, 30, 40, 25},
		SkillXP: [6]uint32{11, 13, 17, 19, 23, 29}, Experience: 0xfedcba98,
		ManaFloor: 0xabcd, Sight: 0x1200, TypeID: 777, MoverSpeed: 23,
		Fighter: true, HasSpellbook: true, HasOwner: true, ManaReservePercent: 0x89abcdef,
		Reach: 4, AttackCharge: 11, AttackRelax: 13, EquipmentRuntimePresent: true,
	}
	for i := range source.Base {
		source.Base[i] = byte(31 + i)
	}
	for i := range source.Modifier {
		source.Modifier[i] = byte(61 + i)
	}
	binary.LittleEndian.PutUint16(source.Attack[:], 37)
	for i := 0; i < 6; i++ {
		binary.LittleEndian.PutUint16(source.Attack[2+2*i:], uint16(2*i+2))
	}
	source.Attack[14], source.Attack[15], source.Attack[16] = 7, 3, 2
	source.Attack[22], source.Attack[23] = 0xa5, 0x5a
	binary.LittleEndian.PutUint16(source.Defence[:], 23)
	binary.LittleEndian.PutUint16(source.Defence[2:], 5)
	load := sim.ActorLoadSnapshot{
		Inventory: sim.ActorLoad{Present: true, OwnWeight: -7, ContainerPresent: true, InsertIndex: 7, Accumulator: -123456, Source: source},
		Load:      77, Capacity: 300, Speed: 19,
		Movement: sim.HumanMovement{Present: true, RawSpeed: 19, NativeSpeed: 19, Load: 77, Capacity: 300},
	}
	weapon := sim.ItemInstance{Code: 0x0135, Kind: 2, Price: 0x1234567, WeightPresent: true,
		Effects: []sim.ItemEffect{{Kind: 41, Mode: 1, Operand: 1 | 4<<16}, {Kind: 40, Mode: 3, Operand: 0xfffedddd}},
		SourceEquipment: sim.SourceEquipment{Class: sim.SourceWeapon, DefinitionRow: 39, OwnKind: 4, EffectsUnsupported: true,
			Definition: sim.SourceWeaponDefinition{Present: true, AttackType: 11, Hands: 2, Charge: 13, Relax: 17, Suitable: 27},
			Spell:      sim.SourceItemSpell{Present: true, ID: 5, Range: 17, Defensive: 0x81, ManaCost: 65535}},
	}
	for i := range weapon.SourceEquipment.Attack {
		weapon.SourceEquipment.Attack[i] = byte(7*i + 1)
	}
	for i := range weapon.SourceEquipment.Defence {
		weapon.SourceEquipment.Defence[i] = byte(5*i + 3)
	}
	pack := sim.ItemInstance{Code: 0x0707, Kind: 3, Price: -12345, WeightPresent: true, Weight: -13,
		Effects: []sim.ItemEffect{{Kind: 6, Mode: 0x88, Operand: 0x98765432}, {Kind: 16, Mode: 1, Operand: 0x76543210}, {Kind: 6, Mode: 0x88, Operand: 0x98765432}}}
	var entities []sim.Entity
	var stocks []sim.Stock
	r := &sim.SavedObjects{Version: 1, NextID: 1000}
	var bindings []sim.SavedObjectBinding
	for actorOrdinal, entity := range []sim.EntityID{7, 15} {
		entities = append(entities, sim.Entity{ID: entity, X: int32(actorOrdinal + 1), Y: 1, Owner: sim.SelfSlot, TypeID: sim.HumanTypeID, HP: 50, MaxHP: 100})
		ownLoad := load
		ownLoad.Inventory.InsertIndex += uint32(actorOrdinal)
		ownLoad.Inventory.Accumulator -= int32(actorOrdinal)
		stock := sim.Stock{ID: entity, LoadState: &ownLoad, OrderedStacks: []sim.ItemStack{sim.StackItem(pack, 3), sim.StackItem(pack, 2), sim.StackItem(sim.PlainItem(0x0888), 1)}}
		for j := 0; j < 5; j++ {
			stock.ItemInstances = append(stock.ItemInstances, pack.Clone())
		}
		stock.ItemInstances = append(stock.ItemInstances, sim.PlainItem(0x0888))
		stock.EquippedItems[0] = weapon.Clone()
		stocks = append(stocks, stock)
		owner := sim.SavedObjectOwner{Kind: sim.SavedOwnerActorPack, Entity: entity}
		c := sim.SavedObjectContainer{Owner: owner, Present: true, InsertIndex: ownLoad.Inventory.InsertIndex, Accumulator: ownLoad.Inventory.Accumulator,
			Coverage: sim.SavedObjectCoverage{Unknown: sim.SavedUnknownContainerLoad | sim.SavedUnknownMergePolicy}}
		for itemOrdinal, value := range []sim.ItemStack{stock.OrderedStacks[0], stock.OrderedStacks[1], sim.StackItem(weapon, 1)} {
			id := sim.SavedObjectID(10 + actorOrdinal*3 + itemOrdinal)
			value = value.Clone()
			value.ObjectID = id
			itemOwner, index := owner, uint32(itemOrdinal)
			if itemOrdinal == 2 {
				itemOwner, index = sim.SavedObjectOwner{Kind: sim.SavedOwnerActorWorn, Entity: entity, Slot: 1}, 0
			} else {
				c.Items = append(c.Items, id)
			}
			row := sim.SavedItemObject{ID: id, Origin: sim.SavedObjectOrigin{Kind: sim.SavedObjectOriginal}, Owner: itemOwner, Value: value,
				Token: sim.SavedObjectToken{T0C: value.SourceEquipment.DefinitionRow, T1C: uint32(value.Price), T08: 0xabcdef01, T18: 0x4567, T0E: 0x4321, RuntimeID: 987, Identity: 0xaa000000 | uint32(id), Reference: 0x1234}, F45: 7, F46: 9, F47: 11, F48: 0xfedc}
			for at := range row.Token.Position {
				row.Token.Position[at] = byte(at + itemOrdinal)
			}
			for _, value := range value.Effects {
				child := sim.SavedObjectID(100 + len(r.Effects))
				row.Effects = append(row.Effects, child)
				r.Effects = append(r.Effects, sim.SavedEffectObject{ID: child, Origin: sim.SavedObjectOrigin{Kind: sim.SavedObjectOriginal}, Value: value, E0C: 0x71,
					Token: sim.SavedObjectToken{Identity: uint32(child) | 0xbb000000, T0C: 0x19, T1C: 0xfedcba98}})
			}
			if value.SourceEquipment.Spell.Present {
				row.Spell = sim.SavedObjectID(200 + len(r.Spells))
				r.Spells = append(r.Spells, sim.SavedSpellObject{ID: row.Spell, Origin: sim.SavedObjectOrigin{Kind: sim.SavedObjectOriginal}, Value: value.SourceEquipment.Spell, This: 0xcccdddee})
			}
			r.Items = append(r.Items, row)
			bindings = append(bindings, sim.SavedObjectBinding{ID: id, Owner: itemOwner, Index: index, Value: value})
		}
		c.Items = append(c.Items, 0)
		r.Containers = append(r.Containers, c)
	}
	w, err := sim.NewStockedWorld(1115, sim.Bounds{Width: 8, Height: 8}, sim.ModeCanonical, sim.Terrain{}, entities, nil, sim.Relations{}, nil, stocks)
	if err != nil {
		t.Fatal(err)
	}
	if err := w.ImportSavedObjects(r, nil, bindings...); err != nil {
		t.Fatal(err)
	}
	return w
}

func carryObjectsState1115(t *testing.T, w *sim.World, id sim.EntityID) mapload.Carry {
	t.Helper()
	var c mapload.Carry
	c.Items, _ = w.Carried(id)
	c.ItemInstances, _ = w.CarriedItems(id)
	c.OrderedStacks, _ = w.CarriedStacks(id)
	c.Equipped, _ = w.Equipped(id)
	c.EquippedItems, _ = w.EquippedItems(id)
	for _, e := range w.Entities() {
		if e.ID == id {
			c.SkillXP, c.LiveLoad = e.SkillXP, e.CurrentActorLoad()
			return c
		}
	}
	t.Fatalf("missing test actor %d", id)
	return c
}

func TestCarryObjects1115BoundaryDropsOnlyHandlesAndKeepsSameWorldOwners(t *testing.T) {
	w := carryObjectsWorld1115(t)
	beforeBytes, err := w.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	beforeRegistry := w.SavedObjects()
	heroCarry, joinedCarry := carryObjectsState1115(t, w, 7), carryObjectsState1115(t, w, 15)
	party := []mapload.PartyMember{{ID: "hero", StartingHero: true, Carry: &heroCarry}}
	roster := map[sim.EntityID]mapload.PartyMember{15: {ID: "join:15", Carry: &joinedCarry, PlayerCharacter: true}}
	beforeParty := mapload.CloneParty(party)
	beforeRoster := mapload.CloneParty([]mapload.PartyMember{roster[15]})
	within, ids := mapload.CarryRosterIDs(party, w, []sim.EntityID{7}, roster)
	if !reflect.DeepEqual(ids, []sim.EntityID{7, 15}) || len(within) != 2 {
		t.Fatal("same-world roster failed to include the joined actor", ids)
	}
	boundary := mapload.CarryRoster(party, w, []sim.EntityID{7}, roster)
	if len(boundary) != 2 || boundary[0].ID != "hero" || boundary[1].ID != "join:15" {
		t.Fatal("boundary member order or joiner changed")
	}
	for i, id := range []sim.EntityID{7, 15} {
		live := carryObjectsState1115(t, w, id)
		if !reflect.DeepEqual(within[i].Carry, &live) {
			t.Fatalf("same-world roster changed item identity or other operands for %d", id)
		}
		if len(live.OrderedStacks) != 3 || live.OrderedStacks[0].Count != 3 || live.OrderedStacks[1].Count != 2 || live.OrderedStacks[0].ObjectID == live.OrderedStacks[1].ObjectID || live.OrderedStacks[2].ObjectID != 0 {
			t.Fatal("fixture lost distinct equal stack identities/counts or native slot")
		}
		want := carryObjectsState1115(t, w, id)
		for j := range want.ItemInstances {
			want.ItemInstances[j].ObjectID = 0
		}
		for j := range want.OrderedStacks {
			want.OrderedStacks[j].ObjectID = 0
		}
		for j := range want.EquippedItems {
			want.EquippedItems[j].ObjectID = 0
		}
		if !reflect.DeepEqual(boundary[i].Carry, &want) {
			t.Fatalf("cross-world values changed more than handles for %d\ngot %+v\nwant %+v", id, boundary[i].Carry, want)
		}
		// Every mutable projection is independent, including the repeated
		// equal stacks and the actor's source/load snapshot.
		boundary[i].Carry.ItemInstances[0].Effects[0].Operand++
		boundary[i].Carry.OrderedStacks[0].Effects[0].Mode++
		boundary[i].Carry.EquippedItems[0].Effects[0].Kind++
		boundary[i].Carry.LiveLoad.Inventory.Source.Modifier[3]++
		if !reflect.DeepEqual(within[i].Carry, &live) {
			t.Fatal("boundary aliases same-world projection")
		}
	}
	afterBytes, err := w.MarshalBinary()
	if err != nil || !bytes.Equal(beforeBytes, afterBytes) || !reflect.DeepEqual(beforeRegistry, w.SavedObjects()) {
		t.Fatal("crossing changed finished World or object registry", err)
	}
	if !reflect.DeepEqual(party, beforeParty) || !reflect.DeepEqual([]mapload.PartyMember{roster[15]}, beforeRoster) {
		t.Fatal("crossing changed the prior party/roster owner")
	}
}
