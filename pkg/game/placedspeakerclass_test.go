package game

import (
	"slices"
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/formats/alm"
	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

// A scenario-placed person is drawn with the class his own Humans row names
// (UNIT-APPEAR-030), and the dialogue figure draws the horse under a rider of
// class 0x11 to 0x15 (HERO-DOLL-078, DLG-FIGURE-020). The roster template that
// stands for a placed person carries another class: the equipment law rewrites
// it to a body class when the mission opens, and a SAV load replaces it with
// the class the entity was placed with. Neither may decide the horse. The
// Lancer speaks with his horse whichever of them his template carries; a person
// whose row is no rider speaks without one even when his template says rider;
// a Hero-mode placement never draws the horse.
func TestPlacedSpeakerHorseFollowsHisRowClassNotHisRosterClass(t *testing.T) {
	const (
		lancer, lancerServer = 21, 208
		footman, footServer  = 3, 209
		bodyClass            = 13
	)
	row := func(name string, typeID, face, server int32) dbEntry {
		p := figureRow(typeID, face, 0)
		p[24] = server
		return dbEntry{name: name, params: p}
	}
	table := dressTable()
	table.Humans = append(slices.Clone(table.Humans.(dbCollection)),
		row("NPC08_2", lancer, 2, lancerServer), row("NPC03_2", footman, 5, footServer))
	faces := map[int32]data.NPCFace{
		59: {Kind: data.NPCFigure, Dir: data.FigureDirManFighter, Face: 2,
			Tokens: npcTokenSet(data.NPCTokenHuman, data.NPCTokenNotFemale, data.NPCTokenNotMage, data.NPCTokenFace)},
		60: {Kind: data.NPCFigure, Dir: data.FigureDirManFighter, Face: 5,
			Tokens: npcTokenSet(data.NPCTokenHuman, data.NPCTokenNotFemale, data.NPCTokenNotMage, data.NPCTokenFace)},
		25: {Kind: data.NPCFigure, Dir: data.FigureDirManFighter, Face: dressPaladinFace,
			Tokens: npcTokenSet(data.NPCTokenHero, data.NPCTokenFace, data.NPCTokenNotFemale, data.NPCTokenNotMage)},
	}
	original := worldFixtureMap()
	original.Units = []alm.Unit{
		{UnitID: 10, ClassID: 64},
		{UnitID: 11, ClassID: lancer, DefID: lancerServer},
		{UnitID: 12, ClassID: footman, DefID: footServer},
		{UnitID: 13, ClassID: 1, ClassSubID: 25, Flags: 1},
	}
	world := func(ids ...sim.EntityID) *sim.World {
		t.Helper()
		var entities []sim.Entity
		for _, id := range ids {
			entities = append(entities, sim.Entity{ID: id, X: int32(id) + 2, Y: 3, HP: 10, MaxHP: 10,
				MapUnitID: original.Units[id].UnitID, Humanoid: id != 0})
		}
		w, err := sim.NewStockedWorld(1, sim.Bounds{Width: worldFixtureW, Height: worldFixtureH},
			sim.ModeCanonical, sim.Terrain{}, entities, nil, sim.Relations{}, nil, nil)
		if err != nil {
			t.Fatal(err)
		}
		return w
	}
	roster := func(lancerClass, footmanClass int32) map[sim.EntityID]mapload.PartyMember {
		return map[sim.EntityID]mapload.PartyMember{
			1: {Name: "Lancer", Class: lancerClass, Body: "pikeman_",
				FigureDir: string(data.FigureDirManFighter), FigureFace: 2},
			2: {Name: "Footman", Class: footmanClass, Body: "swordsman_",
				FigureDir: string(data.FigureDirManFighter), FigureFace: 5},
			3: {PlayerCharacter: true, Class: lancer, CompanionNPC: 25,
				FigureDir: string(data.FigureDirManFighter), FigureFace: dressPaladinFace},
		}
	}
	open := func(m *alm.Map, w *sim.World, roster map[sim.EntityID]mapload.PartyMember, state *SnapshotSAVDocument) *mapWorld {
		return openMission(&Mission{Number: 71, Map: m, World: w,
			Start: mapload.Start{Roster: roster}, savedDocument: state},
			table, nil, worldFixtureViewer(t, m), missionSource{}, nil, faces)
	}
	check := func(when string, mw *mapWorld) {
		t.Helper()
		if key := composeKey(t, mw, 59); key.fig != (figureID{Dir: data.FigureDirManFighter, Face: 2, Horse: true}) {
			t.Errorf("%s: the Lancer speaks as %+v, want the man fighter 2 on his horse: a row of class %d rides", when, key.fig, lancer)
		}
		if key := composeKey(t, mw, 60); key.fig != (figureID{Dir: data.FigureDirManFighter, Face: 5}) {
			t.Errorf("%s: the footman speaks as %+v, want the man fighter 5 with no horse: his row is class %d", when, key.fig, footman)
		}
		if key := composeKey(t, mw, 25); key.fig != (figureID{Dir: data.FigureDirManFighter, Face: dressPaladinFace, Hero: true}) {
			t.Errorf("%s: the Hero placement speaks as %+v, want the hero figure with no horse", when, key.fig)
		}
	}

	// The class a fresh roster template carries after the equipment law, and
	// the class a load hands back (the entity's placement class, a rider's).
	check("fresh, template of the body class", open(original, world(0, 1, 2, 3), roster(bodyClass, footman), nil))
	check("loaded, template of the placement class", open(original, world(0, 1, 2, 3), roster(lancer, lancer), nil))

	// A load that withdrew the creature before the three persons pairs each
	// entity with the placement carrying its own map id, so the row read is his
	// own and not the neighbour's.
	state := &SnapshotSAVDocument{Document: &sav.DocumentData{World: &sav.DocumentWorldData{}}}
	graph := sav.SavedActorGraph{CurrentPopulation: true}
	for _, id := range []uint16{11, 12, 13} {
		graph.Actors = append(graph.Actors, sav.ActorRecord{CurrentEntity: true, Actor: sav.Actor{MapUnitID: id}})
	}
	retained := *original
	retained.Units = slices.Clone(original.Units)
	retainSavedActorPlacements(&retained, graph, nil, state)
	if !slices.Equal(retained.Units, original.Units[1:]) {
		t.Fatalf("saved population kept placements %+v, want the three persons", retained.Units)
	}
	check("loaded after a withdrawal", open(&retained, world(1, 2, 3), roster(lancer, lancer), state))
}
