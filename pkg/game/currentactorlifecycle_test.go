package game

import (
	"fmt"
	"reflect"
	"slices"
	"testing"

	"againrom/pkg/formats/alm"
	"againrom/pkg/formats/sav"
)

func TestCurrentActorRetirementClosure(t *testing.T) {
	record := func(children ...uint16) sav.DocumentRecordData {
		return sav.DocumentRecordData{RefSlots: []sav.DocumentRefsData{{Name: "children", Objects: children}}}
	}
	doc := sav.DocumentData{Players: []uint16{1}, Objects: []sav.DocumentRecordData{
		record(2), record(5), record(4, 5, 6), record(3), record(), record(7), record(), record(),
	}}
	// 3/4 are an exclusive retired cycle; 5 is shared with a live root.
	// 6/7 model an active reserved item and child; 8 is an unrelated orphan.
	got, err := retiredActorClosure(&doc, []uint16{3}, []uint16{6})
	if err != nil || !slices.Equal(got, []uint16{3, 4}) {
		t.Fatal("retired closure changed shared, reserved or unrelated objects", got, err)
	}
	doc.DeadActors = []uint16{3}
	if got, err := retiredActorClosure(&doc, []uint16{3}, nil); err != nil || len(got) != 0 {
		t.Fatal("held corpse graph retired", got, err)
	}
	doc.DeadActors = nil
	if got, err := retiredActorClosure(&doc, []uint16{3}, []uint16{3}); err != nil || len(got) != 0 {
		t.Fatal("live actor graph retired", got, err)
	}
	doc.Objects[7].Inline = []sav.DocumentInlineData{{Name: "nested", Record: record(4)}}
	if got, err := retiredActorClosure(&doc, []uint16{3}, nil); err != nil || len(got) != 0 {
		t.Fatal("surviving inline reference lost", got, err)
	}
	doc.Objects[7] = record(99)
	if _, err := retiredActorClosure(&doc, []uint16{3}, nil); err == nil {
		t.Fatal("malformed noncandidate reference was ignored")
	}
}

func TestSavedActorPlacementsRequireCompletePopulation(t *testing.T) {
	state := &SnapshotSAVDocument{Document: &sav.DocumentData{World: &sav.DocumentWorldData{}}}
	for _, key := range []uint32{10, 20, 30, 40} {
		state.Document.Objects = append(state.Document.Objects, sav.DocumentRecordData{Class: "Unit", Values: []sav.DocumentValueData{{Name: "Identity", Value: key}}})
	}
	graph := sav.SavedActorGraph{Actors: []sav.ActorRecord{
		{Identity: 10, Actor: sav.Actor{MapUnitID: 1}},
		{Identity: 20, Actor: sav.Actor{MapUnitID: 999}}, // spawned/current actor
		{Identity: 30, Actor: sav.Actor{MapUnitID: 3, Stage: 1, HP: -1, RuntimeID: 30}},
	}}
	dead := []sav.DeadActor{{Identity: 30, MapUnitID: 3}, {Identity: 40, MapUnitID: 4}}
	units := []alm.Unit{{UnitID: 1}, {UnitID: 2}, {UnitID: 3}, {UnitID: 4}, {UnitID: 0}, {UnitID: 5}, {UnitID: 5}}
	m := &alm.Map{Units: slices.Clone(units)}
	retainSavedActorPlacements(m, graph, dead, state)
	if !reflect.DeepEqual(m.Units, append(slices.Clone(units[:1]), units[2:]...)) || len(graph.Actors) != 3 {
		t.Fatal("live, dying, held, spawned or ambiguous placements changed", m.Units)
	}
	for _, name := range []string{"missing dead", "missing world", "unbound graph", "duplicate key"} {
		t.Run(name, func(t *testing.T) {
			m.Units = slices.Clone(units)
			doc := *state.Document
			doc.Objects = slices.Clone(doc.Objects)
			partial := &SnapshotSAVDocument{Document: &doc}
			roots := slices.Clone(dead)
			switch name {
			case "missing dead":
				roots = nil
			case "missing world":
				doc.World = nil
			case "unbound graph":
				doc.Objects = doc.Objects[1:]
			case "duplicate key":
				doc.Objects = append(doc.Objects, doc.Objects[0])
			}
			retainSavedActorPlacements(m, graph, roots, partial)
			if !reflect.DeepEqual(m.Units, units) {
				t.Fatal("incomplete authority removed a placement")
			}
		})
	}
}

func TestCurrentActorPlacementsRetainDeadManagerBindings(t *testing.T) {
	state := &SnapshotSAVDocument{Document: &sav.DocumentData{World: &sav.DocumentWorldData{}}}
	graph := sav.SavedActorGraph{CurrentPopulation: true, Actors: []sav.ActorRecord{{CurrentEntity: true, Actor: sav.Actor{MapUnitID: 2}}}}
	m := &alm.Map{Units: []alm.Unit{{UnitID: 1}, {UnitID: 2}, {UnitID: 3}, {UnitID: 0}}}
	retainSavedActorPlacements(m, graph, []sav.DeadActor{{MapUnitID: 3}}, state)
	if !reflect.DeepEqual(m.Units, []alm.Unit{{UnitID: 2}, {UnitID: 3}}) {
		t.Fatal("current population lost the separately owned dead-manager placement", m.Units)
	}
	m.Units = []alm.Unit{{UnitID: 1}, {UnitID: 0}}
	retainSavedActorPlacements(m, sav.SavedActorGraph{CurrentPopulation: true}, nil, state)
	if len(m.Units) != 0 {
		t.Fatal("explicit empty population resurrected map actors", m.Units)
	}
}

func TestCurrentActorPopulationIncludesBoundLateRoots(t *testing.T) {
	body := &poolFixtureActor{mapID: 91, cell: 0x0605, stage: 4, hp: uint16(65536 - 120), maxHP: 321, mana: 17, maxMana: 53,
		book:     []*poolFixtureSpell{nil, nil, {id: 3, rangeByte: 11, cost: 23}},
		holdings: &holdingFixture{items: []*holdingFixtureItem{{class: "Item", code: 0x0e06, count: 4, kind: 3, price: 19}}}}
	file, err := sav.Open(savedContainer(poolFixtureBody([]*poolFixturePlayer{{}}, []*poolFixtureActor{body})))
	if err != nil {
		t.Fatal(err)
	}
	dead, err := file.DeadActors()
	if err != nil || len(dead) != 1 {
		t.Fatal("late body fixture", dead, err)
	}
	ordinary, err := file.ActorGraph()
	if err != nil || len(ordinary.Actors) != 0 {
		t.Fatal("ordinary actor population changed", ordinary, err)
	}
	current, err := file.CurrentActorGraph([]uint16{dead[0].ArchiveIndex})
	if err != nil || len(current.Actors) != 1 {
		t.Fatal("current late root was not admitted", current, err)
	}
	actor := current.Actors[0]
	if !actor.CurrentEntity || actor.Stage != 4 || actor.HP != -120 || actor.Character.Basis == nil {
		t.Fatal("current late root lost ordinary lifecycle or basis", actor)
	}
	// Change the ordinary attack bytes after admission. The explicit population
	// supplies membership only; every reader must see these new values.
	file.Body[body.off+45+14], file.Body[body.off+45+15] = 83, 9
	archives := currentActorArchives(current)
	pools, err := file.ActorPools(archives...)
	if err != nil || len(pools) != 1 || pools[0].HP != body.hp || pools[0].MaxHP != 321 || pools[0].Mana != 17 || pools[0].MaxMana != 53 {
		t.Fatal("late current pools", pools, err)
	}
	profiles, err := file.ActorCurrentProfiles(archives...)
	if err != nil || len(profiles) != 1 || profiles[0].DamageBase != 83 || profiles[0].DamageSpread != 9 {
		t.Fatal("late current combat", profiles, err)
	}
	holdings, err := file.ActorHoldings(archives...)
	if err != nil || len(holdings) != 1 || holdings[0].Basis == nil || len(holdings[0].Items) != 1 || holdings[0].Items[0].Stack != 4 || holdings[0].Items[0].Price != 19 {
		t.Fatal("late current holdings", holdings, err)
	}
	books, err := file.ActorSpellbooks(archives...)
	if err != nil || len(books) != 1 || !books[0].HasSpellbook || books[0].KnownSpells() != 1<<3 || books[0].Spells[0].Range != 11 || books[0].Spells[0].ManaCost != 23 {
		t.Fatal("late current book", books, err)
	}
	if plain, err := file.ActorPools(); err != nil || len(plain) != 0 {
		t.Fatal("ordinary pools admitted a late corpse", plain, err)
	}
	if plain, err := file.ActorCurrentProfiles(); err != nil || len(plain) != 0 {
		t.Fatal("ordinary profiles admitted a late corpse", plain, err)
	}
	if plain, err := file.ActorHoldings(); err != nil || len(plain) != 0 {
		t.Fatal("ordinary holdings admitted a late corpse", plain, err)
	}
	if plain, err := file.ActorSpellbooks(); err != nil || len(plain) != 0 {
		t.Fatal("ordinary books admitted a late corpse", plain, err)
	}
	file.Body = file.Body[:len(file.Body)-3]
	if partial, err := file.ActorPools(archives...); err == nil || len(partial) != 0 {
		t.Fatal("partial late pools", partial, err)
	}
	if partial, err := file.ActorCurrentProfiles(archives...); err == nil || len(partial) != 0 {
		t.Fatal("partial late profiles", partial, err)
	}
	if partial, err := file.ActorHoldings(archives...); err == nil || len(partial) != 0 {
		t.Fatal("partial late holdings", partial, err)
	}
	if partial, err := file.ActorSpellbooks(archives...); err == nil || len(partial) != 0 {
		t.Fatal("partial late books", partial, err)
	}
}

func TestCurrentAttachmentOnlyRepairsExactObsoleteMarker(t *testing.T) {
	_, ms := actorEffectFixture(t)
	row := ms.savedDocument.ActorEffects.Rows[0]
	marker := fmt.Sprintf("current world SAV unavailable: actor %d effect %d caster persistence is not established; preserve this state in .ags", row.Entity, row.Spell)
	current := worldSaveUnsupportedf("actor %d effect %d caster persistence is not established", row.Entity, row.Spell).Error()
	for _, reason := range []string{marker, current, "unknown callback", marker + "; unknown callback"} {
		state, err := cloneSavedDocument(ms.savedDocument)
		if err != nil {
			t.Fatal(err)
		}
		state.ActorEffects.Unavailable = reason
		if err := projectSavedActorEffects(state, ms.World); err != nil {
			t.Fatal(err)
		}
		want := reason
		if reason == marker {
			want = ""
		}
		if state.ActorEffects.Unavailable != want {
			t.Fatal("unrelated marker cleared", reason)
		}
	}
	state, _ := cloneSavedDocument(ms.savedDocument)
	state.ActorEffects.Unavailable = marker
	state.ActorEffects.Rows[0].Entity = 999
	if err := projectSavedActorEffects(state, ms.World); err == nil {
		t.Fatal("obsolete marker concealed malformed target binding")
	}
}
