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
