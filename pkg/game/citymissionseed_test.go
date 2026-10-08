package game

import (
	"reflect"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

func seededCityMission(t *testing.T) (*FrontEnd, *cityObjectTopology) {
	t.Helper()
	w, state, party := cityReturnObjectFixture(t)
	carried, graph, err := currentMissionCityParty(w, state, party, []sim.EntityID{7}, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	carried[0].FigureFace, carried[0].FigureDir = 3, "fighter"
	carried[0].StartingHero, carried[0].PlayerCharacter = true, true
	f := cellStateFront(t)
	definitions := currentPoolFixtureFront(t, 91)
	f.Table, f.Humans = definitions.Table, definitions.Humans
	f.Campaign = definitions.Campaign
	f.Town = NewTown(f.Campaign.Value())
	f.Town.cityObjects = graph.Clone()
	if err := f.App("current city departure").OpenMission(f.MissionOpenerWith(10, carried)); err != nil {
		t.Fatal("current city mission entry", err)
	}
	if f.live.mission.state.savedDocument != nil || !reflect.DeepEqual(f.Town.cityObjects, graph) {
		t.Fatal("mission entry fabricated source provenance or changed city graph")
	}
	return f, graph
}

func TestCityMissionSeedKeepsBookWeaponAndRepeatedEffectIdentities(t *testing.T) {
	f, graph := seededCityMission(t)
	registry := f.live.world.SavedObjects()
	if registry == nil || len(registry.BookRoots) != 1 || len(registry.Items) != 1 || len(registry.Containers) != 1 {
		t.Fatal("current mission registry omits city roots", registry)
	}
	item := registry.Items[0]
	if item.ID != graph.Roots[0].Pack[0] || len(item.Effects) != 2 || item.Effects[0] != item.Effects[1] || item.Spell != registry.BookRoots[0].Slots[11] || registry.NextID < graph.NextID {
		t.Fatal("current entry split or reminted existing identities", registry)
	}
	pack, _ := f.live.world.CarriedStacks(f.live.mission.ids[0])
	if len(pack) != 1 || pack[0].Count != 3 || pack[0].ObjectID != item.ID {
		t.Fatal("mission construction flattened counted roots", pack)
	}
	back, current, err := currentMissionCityParty(f.live.world, nil, f.live.mission.party, f.live.mission.ids, nil, f.Table)
	if err != nil {
		t.Fatal("current return without source document", err)
	}
	if !reflect.DeepEqual(current.Roots, graph.Roots) || !reflect.DeepEqual(current.Books, graph.Books) || back[0].Carry.OrderedStacks[0].ObjectID != 0 {
		t.Fatal("city round trip lost identity topology or leaked native handles", current, graph)
	}
}

func TestCityMissionSeedTransfersObservedItemRecordAtomically(t *testing.T) {
	w, state, party := cityReturnObjectFixture(t)
	carried, graph, err := currentMissionCityParty(w, state, party, []sim.EntityID{7}, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	source := w.SavedObjects().Items[0]
	history := sim.NativeItemRecord{Class: sim.SourceWeapon, Token: source.Token, F45: 73, F47: 91, F48: 0x9876}
	history.Token.T1C = 0
	history.Token.Position[0], history.Token.Position[11] = 0xa5, 0x5a
	stack := carried[0].Carry.OrderedStacks[0].Clone()
	stack.NativeRecord = &history
	carried[0].Carry.OrderedStacks = []sim.ItemStack{stack}
	for i := range carried[0].Carry.ItemInstances {
		carried[0].Carry.ItemInstances[i] = stack.Instance()
	}
	if err := mapload.ValidatePartyLoad(carried[0]); err != nil {
		t.Fatal(err)
	}
	makeWorld := func(t *testing.T) *sim.World {
		t.Helper()
		next, err := sim.NewStockedWorld(17, sim.Bounds{Width: 16, Height: 16}, sim.ModeCanonical, sim.Terrain{},
			w.Entities(), nil, sim.Relations{}, nil, []sim.Stock{{ID: 7, OrderedStacks: []sim.ItemStack{stack}, ItemInstances: []sim.ItemInstance{stack.Instance(), stack.Instance(), stack.Instance()}}})
		if err != nil {
			t.Fatal(err)
		}
		return next
	}
	for _, corrupt := range []string{"valid", "Position", "service"} {
		t.Run(corrupt, func(t *testing.T) {
			next := makeWorld(t)
			r, bindings, err := currentCityMissionObjects(graph, carried, []sim.EntityID{7}, next, nil)
			if err != nil {
				t.Fatal(err)
			}
			before := next.Hash()
			if corrupt == "Position" {
				r.Items[0].Token.Position[0]++
			} else if corrupt == "service" {
				r.Items[0].F47++
			}
			err = next.ImportSavedObjects(r, nil, bindings...)
			if corrupt != "valid" {
				if err == nil || next.Hash() != before {
					t.Fatal("conflicting native history accepted or mutated World", err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			pack, _ := next.CarriedStacks(7)
			row, ok := next.SavedObjects().Item(pack[0].ObjectID)
			token := row.Token
			token.T1C = 0
			if !ok || pack[0].NativeRecord != nil || token != history.Token || row.F45 != 73 || row.F47 != 91 || row.F48 != 0x9876 {
				t.Fatal("current record lost while binding exact item")
			}
			raw, err := next.MarshalBinary()
			var cold sim.World
			if err != nil || cold.UnmarshalBinary(raw) != nil || cold.Hash() != next.Hash() {
				t.Fatal("bound record lost canonical cold World", err)
			}
		})
	}
}

func TestCityMissionSeedUsesActualWornSpellWithoutChangingParty(t *testing.T) {
	w, _, party := cityReturnObjectFixture(t)
	source := w.SavedObjects().Items[0]
	actual := source.Value.Instance()
	actual.ObjectID, actual.Kind = 0, 0
	history := sim.NativeItemRecord{Class: sim.SourceWeapon, Token: source.Token, F45: 73, F47: 91, F48: 0x9876}
	history.Token.T0C, history.Token.T1C = actual.SourceEquipment.DefinitionRow, 0
	actual.NativeRecord = &history
	input := actual.Clone()
	input.SourceEquipment.Spell = sim.SourceItemSpell{}
	party[0].Carry = &mapload.Carry{Equipped: [sim.EquipSlots]uint16{input.Code}, EquippedItems: [sim.EquipSlots]sim.ItemInstance{input}}
	before := mapload.CloneParty(party)
	graph := &cityObjectTopology{Version: cityObjectTopologyVersion, NextID: source.ID + 1,
		Items: []cityItemTopology{{ID: source.ID}},
		Roots: []cityPartyObjectRoots{{PartyID: []byte(party[0].ID), Worn: [sim.EquipSlots]sim.SavedObjectID{source.ID}}}}
	next, err := sim.NewStockedWorld(17, sim.Bounds{Width: 16, Height: 16}, sim.ModeCanonical, sim.Terrain{},
		[]sim.Entity{{ID: 7, HP: 10, MaxHP: 10}}, nil, sim.Relations{}, nil,
		[]sim.Stock{{ID: 7, EquippedItems: [sim.EquipSlots]sim.ItemInstance{actual}}})
	if err != nil {
		t.Fatal(err)
	}
	r, bindings, err := currentCityMissionObjects(graph, party, []sim.EntityID{7}, next, nil)
	if err != nil {
		t.Fatal(err)
	}
	row, ok := r.Item(source.ID)
	if !ok || row.Value.SourceEquipment.Spell != actual.SourceEquipment.Spell || row.Spell == 0 {
		t.Fatal("mission seed lost the actual post-equip Spell", row)
	}
	if len(r.Spells) != 1 || r.Spells[0].ID != row.Spell || r.Spells[0].Value != actual.SourceEquipment.Spell {
		t.Fatal("mission seed Spell child differs from actual worn value", r.Spells)
	}
	if err := next.ImportSavedObjects(r, nil, bindings...); err != nil {
		t.Fatal("bind actual post-equip item", err)
	}
	worn, _ := next.EquippedItems(7)
	token := row.Token
	token.T1C = 0
	if worn[0].ObjectID != source.ID || worn[0].Kind != 0 || worn[0].NativeRecord != nil || token != history.Token || row.F45 != 73 || row.F47 != 91 || row.F48 != 0x9876 || !reflect.DeepEqual(party, before) {
		t.Fatal("mission seed changed the input party, identity or native history")
	}
}

func TestCityMissionSeedPreservesAbsentSourceTokenDefinition(t *testing.T) {
	w, _, party := cityReturnObjectFixture(t)
	source := w.SavedObjects().Items[0]
	value := source.Value.Instance()
	value.ObjectID, value.Kind = 0, 0
	value.Effects, value.SourceEquipment = nil, sim.SourceEquipment{}
	history := sim.NativeItemRecord{Token: source.Token, F45: 73, F47: 91, F48: 0x9876}
	history.Token.T0C, history.Token.T1C = 14, 0
	value.NativeRecord = &history
	party[0].Carry = &mapload.Carry{Items: []uint16{value.Code}, ItemInstances: []sim.ItemInstance{value}, OrderedStacks: []sim.ItemStack{sim.StackItem(value, 1)}}
	before := mapload.CloneParty(party)
	graph := &cityObjectTopology{Version: cityObjectTopologyVersion, NextID: source.ID + 1,
		Items: []cityItemTopology{{ID: source.ID}},
		Roots: []cityPartyObjectRoots{{PartyID: []byte(party[0].ID), Pack: []sim.SavedObjectID{source.ID}}}}
	next, err := sim.NewStockedWorld(17, sim.Bounds{Width: 16, Height: 16}, sim.ModeCanonical, sim.Terrain{},
		[]sim.Entity{{ID: 7, HP: 10, MaxHP: 10}}, nil, sim.Relations{}, nil,
		[]sim.Stock{{ID: 7, ItemInstances: []sim.ItemInstance{value}}})
	if err != nil {
		t.Fatal(err)
	}
	r, bindings, err := currentCityMissionObjects(graph, party, []sim.EntityID{7}, next, nil)
	if err != nil {
		t.Fatal(err)
	}
	row, ok := r.Item(source.ID)
	if !ok || row.Token.T0C != 14 {
		t.Fatal("absent current source erased observed Token.T0C", row)
	}
	if err := next.ImportSavedObjects(r, nil, bindings...); err != nil {
		t.Fatal("bind absent-source native item", err)
	}
	pack, _ := next.CarriedStacks(7)
	token := row.Token
	token.T1C = 0
	if len(pack) != 1 || pack[0].ObjectID != source.ID || pack[0].Kind != 0 || pack[0].NativeRecord != nil || token != history.Token || row.F45 != 73 || row.F47 != 91 || row.F48 != 0x9876 || !reflect.DeepEqual(party, before) {
		t.Fatal("mission seed changed the input party, identity or native history")
	}
}

func TestCityMissionSeedUsesActualConstructedArmorWithoutChangingParty(t *testing.T) {
	w, _, party := cityReturnObjectFixture(t)
	source := w.SavedObjects().Items[0]
	actual := sim.ItemInstance{Code: 0x5441, Kind: 1, Price: source.Value.Price, Weight: 2, WeightPresent: true,
		SourceEquipment: sim.SourceEquipment{Class: sim.SourceArmor, DefinitionRow: 1, OwnKind: 4}}
	actual.SourceEquipment.Defence[0] = 1
	history := sim.NativeItemRecord{Class: sim.SourceArmor, Token: source.Token, F45: 73, F47: 91, F48: 0x9876}
	history.Token.T0C, history.Token.T1C = 1, 0
	actual.NativeRecord = &history
	input := actual.Clone()
	input.Weight, input.WeightPresent, input.SourceEquipment = 0, false, sim.SourceEquipment{}
	party[0].Carry = &mapload.Carry{}
	party[0].Carry.Equipped[3], party[0].Carry.EquippedItems[3] = input.Code, input
	graph := &cityObjectTopology{Version: cityObjectTopologyVersion, NextID: source.ID + 1,
		Items: []cityItemTopology{{ID: source.ID}}, Roots: []cityPartyObjectRoots{{PartyID: []byte(party[0].ID)}}}
	graph.Roots[0].Worn[3] = source.ID
	beforeParty, beforeGraph := mapload.CloneParty(party), graph.Clone()
	stock := sim.Stock{ID: 7}
	stock.EquippedItems[3] = actual
	next, err := sim.NewStockedWorld(17, sim.Bounds{Width: 16, Height: 16}, sim.ModeCanonical, sim.Terrain{},
		[]sim.Entity{{ID: 7, HP: 10, MaxHP: 10}}, nil, sim.Relations{}, nil, []sim.Stock{stock})
	if err != nil {
		t.Fatal(err)
	}
	r, bindings, err := currentCityMissionObjects(graph, party, []sim.EntityID{7}, next, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(party, beforeParty) || !reflect.DeepEqual(graph, beforeGraph) {
		t.Fatal("constructed armor seeding changed the input Party or topology")
	}
	if err := next.ImportSavedObjects(r, nil, bindings...); err != nil {
		t.Fatal("bind actual constructed armor from absent current source", err)
	}
	worn, _ := next.EquippedItems(7)
	actual.ObjectID, actual.NativeRecord = source.ID, nil
	row, ok := next.SavedObjects().Item(source.ID)
	token := row.Token
	token.T1C = 0
	if !ok || !sim.StackStateEqual(row.Value, sim.StackItem(actual, 1)) || !reflect.DeepEqual(worn[3], actual) || token != history.Token || row.F45 != 73 || row.F47 != 91 || row.F48 != 0x9876 || !reflect.DeepEqual(party, beforeParty) || !reflect.DeepEqual(graph, beforeGraph) {
		t.Fatal("bound armor lost current weight/source, identity, native history or immutable inputs")
	}
}

func TestCityMissionSeedKeepsFreshItemIDsWhenEquipAddsSpell(t *testing.T) {
	w, _, party := cityReturnObjectFixture(t)
	actual := w.SavedObjects().Items[0].Value.Instance()
	actual.ObjectID = 0
	input := actual.Clone()
	input.SourceEquipment.Spell = sim.SourceItemSpell{}
	party[0].Carry = &mapload.Carry{Equipped: [sim.EquipSlots]uint16{input.Code}, EquippedItems: [sim.EquipSlots]sim.ItemInstance{input}}
	fresh := sim.ItemInstance{Code: 0x5441, Kind: 1, Weight: 2, WeightPresent: true,
		SourceEquipment: sim.SourceEquipment{Class: sim.SourceArmor, DefinitionRow: 1, OwnKind: 4}}
	fresh.SourceEquipment.Defence[0] = 1
	hire := mapload.PartyMember{ID: "hired", MercenaryType: 6, Hero: eqHero(), Carry: &mapload.Carry{}}
	hire.Carry.Equipped[3], hire.Carry.EquippedItems[3] = fresh.Code, fresh
	party = append(party, hire)
	graph := &cityObjectTopology{Version: cityObjectTopologyVersion, NextID: 3,
		Items: []cityItemTopology{{ID: 1, Effects: []sim.SavedObjectID{2, 2}}}, Effects: []sim.SavedObjectID{2},
		Roots: []cityPartyObjectRoots{{PartyID: []byte(party[0].ID), Worn: [sim.EquipSlots]sim.SavedObjectID{1}}}}
	beforeParty, beforeGraph := mapload.CloneParty(party), graph.Clone()
	city, err := newCityObjectProjection(graph, party, nil)
	if err != nil {
		t.Fatal(err)
	}
	coldGraph := city.graph
	beforeCold := coldGraph.Clone()
	var registry *sim.SavedObjects
	var expectedBindings []sim.SavedObjectBinding
	var hash uint64
	for cycle, entry := range []*cityObjectTopology{coldGraph, graph} {
		stock := []sim.Stock{{ID: 7, EquippedItems: [sim.EquipSlots]sim.ItemInstance{actual}}, {ID: 13}}
		stock[1].EquippedItems[3] = fresh
		next, err := sim.NewStockedWorld(17, sim.Bounds{Width: 16, Height: 16}, sim.ModeCanonical, sim.Terrain{},
			[]sim.Entity{{ID: 7, HP: 10, MaxHP: 10}, {ID: 13, HP: 10, MaxHP: 10}}, nil, sim.Relations{}, nil, stock)
		if err != nil {
			t.Fatal(err)
		}
		r, bindings, err := currentCityMissionObjects(entry, party, []sim.EntityID{7, 13}, next, nil)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(party, beforeParty) || !reflect.DeepEqual(graph, beforeGraph) || !reflect.DeepEqual(coldGraph, beforeCold) {
			t.Fatal("mission entry changed input Party or topology")
		}
		if cycle == 0 {
			registry, expectedBindings = r, bindings
		} else if !reflect.DeepEqual(r, registry) || !reflect.DeepEqual(bindings, expectedBindings) {
			t.Fatal("post-equip Spell changed fresh hire identities between raw and materialized city graphs", r, registry, bindings, expectedBindings)
		}
		if err := next.ImportSavedObjects(r, nil, bindings...); err != nil {
			t.Fatal("bind current weapon Spell and fresh hire item", err)
		}
		if cycle == 0 {
			hash = next.Hash()
		} else if next.Hash() != hash {
			t.Fatal("raw and materialized city graphs produced different current Worlds")
		}
	}
}

func TestCityMissionSeedEquipSpellKeyDoesNotDependOnCityPlayerKey(t *testing.T) {
	w, _, party := cityReturnObjectFixture(t)
	actual := w.SavedObjects().Items[0].Value.Instance()
	actual.ObjectID = 0
	input := actual.Clone()
	input.SourceEquipment.Spell = sim.SourceItemSpell{}
	party[0].Carry = &mapload.Carry{Equipped: [sim.EquipSlots]uint16{input.Code}, EquippedItems: [sim.EquipSlots]sim.ItemInstance{input}}
	entity := w.Entities()[0]
	party[0].Book, party[0].KnownSpells = entity.Book, entity.KnownSpells
	graph := &cityObjectTopology{Version: cityObjectTopologyVersion, NextID: 4,
		Items: []cityItemTopology{{ID: 1, Effects: []sim.SavedObjectID{2, 2}}}, Effects: []sim.SavedObjectID{2}, Spells: []sim.SavedObjectID{3},
		Roots:        []cityPartyObjectRoots{{PartyID: []byte(party[0].ID), Worn: [sim.EquipSlots]sim.SavedObjectID{1}}},
		Books:        []cityBookTopology{{PartyID: []byte(party[0].ID)}},
		SpellRecords: map[sim.SavedObjectID]sim.SavedSpellObject{3: {ID: 3, This: 0x01000001, Value: actual.SourceEquipment.Spell}}}
	graph.Books[0].Slots[11] = 3
	beforeParty, beforeGraph := mapload.CloneParty(party), graph.Clone()
	var expected *sim.SavedObjects
	var expectedBindings []sim.SavedObjectBinding
	for cycle, playerKey := range []uint32{16777220, 16777232} {
		namespace := sav.DocumentData{Version: sav.DocumentDataVersion, Players: []uint16{1},
			Objects: []sav.DocumentRecordData{{Class: "Player", Values: []sav.DocumentValueData{{Name: "This", Value: playerKey}}}}}
		used := map[uint32]bool{playerKey: true}
		for offset := uint32(0); offset < 14; offset++ {
			if offset == 4 {
				continue
			}
			key := uint32(0x01000000) + offset
			used[key] = true
			namespace.Objects = append(namespace.Objects, sav.DocumentRecordData{Class: "Unit", Values: []sav.DocumentValueData{{Name: "Identity", Value: key}}})
		}
		beforeNamespace := namespace
		beforeNamespace.Players = append([]uint16(nil), namespace.Players...)
		beforeNamespace.Objects = append([]sav.DocumentRecordData(nil), namespace.Objects...)
		for i := range beforeNamespace.Objects {
			beforeNamespace.Objects[i].Values = append([]sav.DocumentValueData(nil), namespace.Objects[i].Values...)
		}
		next, err := sim.NewStockedWorld(17, sim.Bounds{Width: 16, Height: 16}, sim.ModeCanonical, sim.Terrain{},
			[]sim.Entity{entity}, nil, sim.Relations{}, nil, []sim.Stock{{ID: 7, EquippedItems: [sim.EquipSlots]sim.ItemInstance{actual}}})
		if err != nil {
			t.Fatal(err)
		}
		r, bindings, err := currentCityMissionObjects(graph, party, []sim.EntityID{7}, next, nil, namespace)
		if err != nil {
			t.Fatal(err)
		}
		if len(r.Spells) != 2 || r.Spells[0].ID != 3 || r.Spells[0].This != 0x01000001 || used[r.Spells[1].This] {
			t.Fatal("mission equip Spell collides with city keys or changes the retained book key", r.Spells)
		}
		if cycle == 0 {
			expected, expectedBindings = r, bindings
		} else if !reflect.DeepEqual(r, expected) || !reflect.DeepEqual(bindings, expectedBindings) {
			t.Fatal("mission-only Spell constructor key depends on city Player.This", expected.Spells, r.Spells)
		}
		if !reflect.DeepEqual(party, beforeParty) || !reflect.DeepEqual(graph, beforeGraph) || !reflect.DeepEqual(namespace, beforeNamespace) {
			t.Fatal("mission equip Spell construction changed an input")
		}
		if err := next.ImportSavedObjects(r, nil, bindings...); err != nil {
			t.Fatal("bind retained book and current equip Spell", err)
		}
	}
}

func TestCityMissionSeedBookAliasesSurviveOrdinarySAVEditsAndTwoColdLoads(t *testing.T) {
	f, _ := seededCityMission(t)
	wantedItem := f.live.world.SavedObjects().Items[0].ID
	wantedSpell := f.live.world.SavedObjects().Items[0].Spell
	for cycle := 0; cycle < 2; cycle++ {
		snapshot, label, err := f.Snapshot(true)
		if err != nil {
			t.Fatal(err)
		}
		raw, err := f.ExportCurrentSave(snapshot, label)
		if err != nil {
			t.Fatal("ordinary SAVE", cycle, err)
		}
		doc, err := sav.DecodeDocumentData(raw)
		if err != nil {
			t.Fatal(err)
		}
		a, err := readCurrentActions(&doc)
		if err != nil || a.Inventory == nil || len(a.Inventory.BookActors) != 1 {
			t.Fatal("current book root presence is absent", err)
		}
		indices := map[sim.SavedObjectID]uint16{}
		for _, row := range a.Ownership {
			indices[row.ID] = row.Object
		}
		var actor uint16
		for _, binding := range a.Bindings {
			if binding.ID == a.Inventory.BookActors[0] && !binding.Structure && !binding.Missing {
				actor = binding.Object
			}
		}
		if actor == 0 || indices[wantedItem] == 0 || indices[wantedSpell] == 0 {
			t.Fatal("current roots lost ordinary bindings")
		}
		book, _ := savedObjectRefs(&doc.Objects[actor-1], "Spells")
		weapon, _ := savedObjectRefs(&doc.Objects[indices[wantedItem]-1], "WeaponSpell")
		if len(book) <= 11 || len(weapon) != 1 || book[11] != weapon[0] || weapon[0] != indices[wantedSpell] {
			t.Fatal("ordinary SAVE split a shared book/weapon Spell", book, weapon)
		}
		if cycle == 0 {
			savedObjectSetValue(&doc.Objects[weapon[0]-1], "S09", 73)
			savedObjectSetValue(&doc.Objects[weapon[0]-1], "S0C", 54321)
			raw, err = sav.EncodeDocumentData(doc)
			if err != nil {
				t.Fatal(err)
			}
		}
		cold := cellStateFront(t)
		cold.Campaign, cold.Table, cold.Humans = f.Campaign, f.Table, f.Humans
		open, town, err := cold.RestoreOriginal(raw)
		if err == nil && !town {
			err = cold.App("current book root cold LOAD").OpenMission(open)
		}
		if err != nil || town {
			t.Fatal("ordinary cold LOAD", cycle, town, err)
		}
		r := cold.live.world.SavedObjects()
		if r == nil || len(r.BookRoots) != 1 || r.BookRoots[0].Slots[11] != wantedSpell {
			t.Fatal("cold LOAD lost native book root", r)
		}
		item, present := r.Item(wantedItem)
		if !present || item.Spell != wantedSpell || item.Value.SourceEquipment.Spell.Range != 73 || item.Value.SourceEquipment.Spell.ManaCost != 54321 {
			t.Fatal("ordinary shared Spell edit did not reach weapon", item)
		}
		for _, entity := range cold.live.world.Entities() {
			if entity.ID == r.BookRoots[0].Entity && (entity.Book.Slots[11].Range != 73 || entity.Book.Slots[11].ManaCost != 54321) {
				t.Fatal("ordinary shared Spell edit did not reach current book")
			}
		}
		f = cold
	}
}

func TestCityMissionSeedRejectsBrokenBindingsWithoutMutation(t *testing.T) {
	w, state, party := cityReturnObjectFixture(t)
	carried, graph, err := currentMissionCityParty(w, state, party, []sim.EntityID{7}, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	before := w.Hash()
	original := graph.Clone()
	for _, ids := range [][]sim.EntityID{nil, {999}} {
		if _, _, err := currentCityMissionObjects(graph, mapload.CloneParty(carried), ids, w, nil); err == nil {
			t.Fatal("invalid runtime binding admitted")
		}
	}
	if before != w.Hash() || !reflect.DeepEqual(original, graph) {
		t.Fatal("failed entry changed live world or city topology")
	}
}
