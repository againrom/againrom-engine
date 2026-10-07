package game

import (
	"reflect"
	"slices"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

func cityReturnObjectFixture(t *testing.T) (*sim.World, *SnapshotSAVDocument, []mapload.PartyMember) {
	t.Helper()
	value := sim.ItemStack{Code: 0x0111, Count: 3, Kind: 2, Price: 91, WeightPresent: true, Weight: -7,
		Effects: []sim.ItemEffect{{Kind: 3, Mode: 2, Operand: 0x81234567}, {Kind: 3, Mode: 2, Operand: 0x81234567}},
		SourceEquipment: sim.SourceEquipment{Class: sim.SourceWeapon, DefinitionRow: 9, OwnKind: 4,
			Spell: sim.SourceItemSpell{Present: true, ID: 12, Range: 8, Defensive: 1, ManaCost: 17}}}
	nextID := sim.SavedObjectID(1)
	row, effects, child, err := sim.ConstructSavedItem(value, func() (sim.SavedObjectID, sim.SavedObjectToken, error) {
		id := nextID
		nextID++
		return id, sim.SavedObjectToken{Identity: uint32(id) + 100}, nil
	})
	if err != nil || child == nil {
		t.Fatal(err)
	}
	row.InFlight = 0
	owner := sim.SavedObjectOwner{Kind: sim.SavedOwnerActorPack, Entity: 7}
	r := &sim.SavedObjects{Version: sim.SavedObjectsVersion, NextID: nextID, Items: []sim.SavedItemObject{row},
		Effects: effects, Spells: []sim.SavedSpellObject{*child},
		Containers: []sim.SavedObjectContainer{{Owner: owner, Present: true, Items: []sim.SavedObjectID{row.ID}}}}
	// The same Effect occurs twice, and the book and weapon own one Spell.
	row.Effects[1] = row.Effects[0]
	r.Items[0] = row
	r.Effects = r.Effects[:1]
	r.Spells[0].ExternalReferences = 1
	other := row
	other.ID, other.Value = r.NextID, row.Value.Clone()
	other.Value.ObjectID = other.ID
	otherOwner := sim.SavedObjectOwner{Kind: sim.SavedOwnerActorPack, Entity: 13}
	r.NextID++
	r.Items = append(r.Items, other)
	r.Containers = append(r.Containers, sim.SavedObjectContainer{Owner: otherOwner, Present: true, Items: []sim.SavedObjectID{other.ID}})
	native := row.Value.Clone()
	native.ObjectID = 0
	book := sim.Spellbook{State: sim.BookPresent}
	spell := r.Spells[0].Value
	book.Slots[spell.ID-1] = sim.BookSpell{Range: spell.Range, Defensive: spell.Defensive, ManaCost: spell.ManaCost}
	w, err := sim.NewStockedWorld(17, sim.Bounds{Width: 16, Height: 16}, sim.ModeCanonical, sim.Terrain{},
		[]sim.Entity{{ID: 7, X: 2, Y: 2, HP: 10, MaxHP: 10, KnownSpells: 1 << spell.ID, Book: book},
			{ID: 13, X: 3, Y: 3, HP: 10, MaxHP: 10, Owner: sim.SelfSlot, TypeID: sim.PersistLow}},
		nil, sim.Relations{}, nil, []sim.Stock{
			{ID: 7, ItemInstances: []sim.ItemInstance{native.Instance(), native.Instance(), native.Instance()}},
			{ID: 13, ItemInstances: []sim.ItemInstance{native.Instance(), native.Instance(), native.Instance()}},
		})
	if err != nil {
		t.Fatal(err)
	}
	if err := w.ImportSavedObjects(r, nil, sim.SavedObjectBinding{ID: row.ID, Owner: owner, Value: row.Value},
		sim.SavedObjectBinding{ID: other.ID, Owner: otherOwner, Value: other.Value}); err != nil {
		t.Fatal(err)
	}
	refs := make([]uint16, spell.ID)
	refs[spell.ID-1] = 2
	doc := sav.DocumentData{Objects: []sav.DocumentRecordData{
		{Class: "Human", RefSlots: []sav.DocumentRefsData{{Name: "Spells", Objects: refs}}},
		{Class: "Spell"},
	}}
	state := &SnapshotSAVDocument{Document: &doc, Actors: []SnapshotSAVActor{{EntityID: 7, ObjectIndex: 1}},
		Objects: &SnapshotSAVObjectBindings{Version: 2, Spells: []SnapshotSAVObjectBinding{{ID: row.Spell, ObjectIndex: 2}}}}
	return w, state, []mapload.PartyMember{{ID: "hero", Name: "current hero", Hero: eqHero()}}
}

func TestCityMissionReturnUsesJoinedIdentityAndDropsHiredRoots(t *testing.T) {
	for _, hired := range []bool{false, true} {
		w, state, party := cityReturnObjectFixture(t)
		joined := mapload.PartyMember{ID: "joined", Hero: eqHero()}
		if hired {
			joined.MercenaryType = 1
		}
		carried, graph, err := currentMissionCityParty(w, state, party, []sim.EntityID{7}, map[sim.EntityID]mapload.PartyMember{13: joined}, nil)
		if err != nil {
			t.Fatal(err)
		}
		count := 2
		if hired {
			count = 1
		}
		if len(carried) != count || len(graph.Roots) != count || len(graph.Items) != count || graph.NextID != w.SavedObjects().NextID {
			t.Fatal("survivor selection changed topology or allocation floor", hired, carried, graph)
		}
		if !hired && (string(graph.Roots[1].PartyID) != "joined" || graph.Roots[1].Pack[0] != w.SavedObjects().Items[1].ID ||
			graph.Items[0].Spell != graph.Items[1].Spell || !slices.Equal(graph.Items[0].Effects, graph.Items[1].Effects)) {
			t.Fatal("joined actor borrowed another member's roots or lost shared children", graph)
		}
	}
}

func TestCityMissionReturnRetainsCountedRootsAndSharedChildren(t *testing.T) {
	w, state, party := cityReturnObjectFixture(t)
	wantWorld, wantRegistry, wantParty := w.Hash(), w.SavedObjects(), mapload.CloneParty(party)
	carried, graph, err := currentMissionCityParty(w, state, party, []sim.EntityID{7}, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	row := wantRegistry.Items[0]
	if graph.NextID != wantRegistry.NextID || len(graph.Roots) != 1 || !slices.Equal(graph.Roots[0].Pack, []sim.SavedObjectID{row.ID}) ||
		len(graph.Items) != 1 || !slices.Equal(graph.Items[0].Effects, row.Effects) || graph.Items[0].Spell != row.Spell ||
		graph.Books[0].Slots[11] != row.Spell {
		t.Fatal("return lost exact item/child/book topology", graph)
	}
	stacks := carried[0].Carry.OrderedStacks
	if len(stacks) != 1 || stacks[0].Count != 3 || stacks[0].ObjectID != 0 || stacks[0].Price != row.Value.Price {
		t.Fatal("return lost counted current stack or leaked world handle", stacks)
	}
	if w.Hash() != wantWorld || !reflect.DeepEqual(w.SavedObjects(), wantRegistry) || !reflect.DeepEqual(party, wantParty) {
		t.Fatal("return capture changed its inputs")
	}
	// The city snapshot's own clone boundary retains these exact edges.
	town := &Town{cityObjects: graph}
	var snapshot Snapshot
	snapshotTown(town, &snapshot)
	if !reflect.DeepEqual(snapshot.CityObjects, graph) {
		t.Fatal("city snapshot lost return topology")
	}
	graph.Items[0].Effects[0] = 999
	if snapshot.CityObjects.Items[0].Effects[0] == 999 {
		t.Fatal("city snapshot shares return edges")
	}
}

func TestCityMissionReturnBookMutationSplitsOnlyChangedEdge(t *testing.T) {
	w, state, party := cityReturnObjectFixture(t)
	before := w.SavedObjects()
	entity := w.Entities()[0]
	entity.Book.Slots[11].Range++
	if err := w.ImportOriginalActorSpellbooks([]sim.OriginalActorSpellbook{{ID: 7, KnownSpells: entity.KnownSpells, Book: entity.Book}}); err != nil {
		t.Fatal(err)
	}
	carried, graph, err := currentMissionCityParty(w, state, party, []sim.EntityID{7}, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if graph.Items[0].Spell != before.Items[0].Spell || graph.Books[0].Slots[11] != before.NextID || graph.NextID != before.NextID+1 ||
		carried[0].Book.Slots[11].Range != entity.Book.Slots[11].Range {
		t.Fatal("book mutation changed weapon identity or lost current value", graph)
	}
	if !reflect.DeepEqual(w.SavedObjects(), before) {
		t.Fatal("book capture changed weapon registry")
	}
}

func TestCityMissionReturnMalformedBookDoesNotChangeWorld(t *testing.T) {
	for _, edit := range []func(*SnapshotSAVDocument){
		func(s *SnapshotSAVDocument) { s.Actors = append(s.Actors, s.Actors[0]) },
		func(s *SnapshotSAVDocument) { s.Actors[0].ObjectIndex = 3 },
		func(s *SnapshotSAVDocument) { s.Document.Objects[0].RefSlots[0].Objects[11] = 3 },
		func(s *SnapshotSAVDocument) { s.Document.Objects[1].Class = "Effect" },
	} {
		w, state, party := cityReturnObjectFixture(t)
		before, oldParty := w.Hash(), mapload.CloneParty(party)
		edit(state)
		if _, _, err := currentMissionCityParty(w, state, party, []sim.EntityID{7}, nil, nil); err == nil {
			t.Fatal("malformed book accepted")
		}
		if w.Hash() != before || !reflect.DeepEqual(party, oldParty) {
			t.Fatal("failed capture changed world or input party")
		}
	}
}

func TestCityMissionReturnProductionPathsInstallTheSameGraph(t *testing.T) {
	w, state, party := cityReturnObjectFixture(t)
	campaign := Campaign{Main: []int{10}, Offered: []int{10}, Chapters: map[int]Chapter{10: {Mission: 10}}}
	f := &FrontEnd{}
	f.Campaign = resolved(campaign, nil)
	f.Town = NewTown(campaign)
	f.Town.Arrive()
	f.Carried, f.Offered, f.liveMission = mapload.CloneParty(party), 10, 10
	f.live = &mapWorld{world: w, mission: &missionNotices{number: 10, party: party, ids: []sim.EntityID{7},
		state: &Mission{Number: 10, World: w, Party: party, Start: mapload.Start{IDs: []sim.EntityID{7}}, savedDocument: state}}}
	var captured Snapshot
	snapshotTown(f.Town, &captured)
	captured.Party, captured.Mission, captured.Offered = mapload.CloneParty(party), 10, 10
	before := w.Hash()
	draft, city, _, err := f.citySnapshotFromMission(captured)
	if err != nil {
		t.Fatal("detached city boundary", err)
	}
	if f.Town.cityObjects != nil || w.Hash() != before {
		t.Fatal("detached boundary changed live session")
	}
	if next, line := f.FinishMissionWithRoster(10, party, w, []sim.EntityID{7}, nil); next < 0 {
		t.Fatal("live city boundary", line)
	}
	if f.Town.cityObjects == nil || !reflect.DeepEqual(f.Town.cityObjects, city.CityObjects) || !reflect.DeepEqual(f.Town.cityObjects, draft.Town.cityObjects) {
		t.Fatal("live/detached boundary topology differs", f.Town.cityObjects, city.CityObjects)
	}
	if f.Carried[0].Carry.OrderedStacks[0].Count != 3 || draft.Carried[0].Carry.OrderedStacks[0].Count != 3 {
		t.Fatal("production boundary lost stack quantity")
	}
}
