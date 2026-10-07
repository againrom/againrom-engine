package game

import (
	"slices"
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/formats/alm"
	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// A SAV load keeps the saved entity ids and withdraws the placements of actors
// that are gone, so the loaded map is shorter than the one the ids were minted
// from. Each creature and each person must still carry the sheet of its own
// placement: on mission 10 three creatures that stood after a load carried the
// person sheet Body 30, Reaction 25 in place of their own Body 5, Reaction 60.
func TestLoadedActorsCarryTheirOwnPlacementSheets(t *testing.T) {
	const wolfKey, bearKey = unitsArmKey, unitsArmKey + 1
	wolf := map[int]int32{0: 5, 1: 60, 2: 1, 3: 2}
	bear := map[int]int32{0: 44, 1: 33, 2: 22, 3: 11, 0x1d: bearKey}
	table := &mapload.Table{
		Units:  dbCollection{{}, unitsRow("Wolf", wolf), unitsRow("Bear", bear)},
		Humans: dbCollection{{}, personDBRow("Guard", 30, 25, 25, 25, [data.SkillSlots]int32{}, sheetPersonKey)},
	}
	original := worldFixtureMap()
	original.Units = []alm.Unit{
		{UnitID: 11, ClassID: bearKey},
		{UnitID: 12, ClassID: wolfKey},
		{UnitID: 13, ClassID: sheetPersonKey},
		{UnitID: 14, ClassID: bearKey},
		{UnitID: 15, ClassID: wolfKey},
	}
	type sheet struct {
		band                         ui.CharacterBand
		body, reaction, mind, spirit int
	}
	wolfSheet := sheet{ui.CharacterBandCreature, 5, 60, 1, 2}
	bearSheet := sheet{ui.CharacterBandCreature, 44, 33, 22, 11}
	guard, err := data.NewHumanDef("Guard", table.Humans.EntryParams(1))
	if err != nil {
		t.Fatal(err)
	}
	derived := guard.Hero().Recompute(guard.Profile(), data.Loadout{})
	guardSheet := sheet{ui.CharacterBandPerson, int(derived.Body), int(derived.Reaction), int(derived.Mind), int(derived.Spirit)}
	own := map[sim.EntityID]sheet{0: bearSheet, 1: wolfSheet, 2: guardSheet, 3: bearSheet, 4: wolfSheet}

	world := func(ids ...sim.EntityID) *sim.World {
		t.Helper()
		var entities []sim.Entity
		for _, id := range ids {
			entities = append(entities, sim.Entity{ID: id, X: int32(id) + 2, Y: 3, HP: 10, MaxHP: 10,
				MapUnitID: original.Units[id].UnitID, Humanoid: id == 2})
		}
		w, err := sim.NewStockedWorld(1, sim.Bounds{Width: worldFixtureW, Height: worldFixtureH},
			sim.ModeCanonical, sim.Terrain{}, entities, nil, sim.Relations{}, nil, nil)
		if err != nil {
			t.Fatal(err)
		}
		return w
	}
	open := func(m *alm.Map, w *sim.World, state *SnapshotSAVDocument) *mapWorld {
		return openMission(&Mission{Number: 10, Map: m, World: w, savedDocument: state},
			table, nil, worldFixtureViewer(t, m), missionSource{}, nil, nil)
	}
	check := func(when string, mw *mapWorld, ids ...sim.EntityID) {
		t.Helper()
		draws := make(map[uint32]ui.UnitCharacter)
		for _, d := range mw.entityDraws() {
			draws[d.ID] = d.Char
		}
		for _, id := range ids {
			for _, read := range []struct {
				source string
				got    ui.UnitCharacter
			}{{"sheet", mw.chars[id]}, {"pane", draws[uint32(id)]}} {
				have := sheet{read.got.Band, read.got.Body, read.got.Reaction, read.got.Mind, read.got.Spirit}
				if !read.got.Known || have != own[id] {
					t.Errorf("%s: entity %d (unit %d) %s reads %+v known %t, want its own placement's %+v",
						when, id, original.Units[id].UnitID, read.source, have, read.got.Known, own[id])
				}
			}
		}
	}

	check("fresh", open(original, world(0, 1, 2, 3, 4), nil), 0, 1, 2, 3, 4)

	state := &SnapshotSAVDocument{Document: &sav.DocumentData{World: &sav.DocumentWorldData{}}}
	graph := sav.SavedActorGraph{CurrentPopulation: true}
	for _, id := range []uint16{12, 13, 14, 15} {
		graph.Actors = append(graph.Actors, sav.ActorRecord{CurrentEntity: true, Actor: sav.Actor{MapUnitID: id}})
	}
	retained := *original
	retained.Units = slices.Clone(original.Units)
	retainSavedActorPlacements(&retained, graph, nil, state)
	if !slices.Equal(retained.Units, original.Units[1:]) {
		t.Fatalf("saved population kept placements %+v, want the four survivors", retained.Units)
	}
	check("after LOAD", open(&retained, world(1, 2, 3, 4), state), 1, 2, 3, 4)
}
