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

// A SAV load keeps the saved entity ids and withdraws the placements of actors
// that are gone, so the loaded map is shorter than the one the ids were minted
// from. A dialogue speaker must still be the person standing on the map, in
// his own worn set, with the Hero bit of his own placement. On the owner's
// mission-10 save three earlier creatures were gone, and Sarindar (a man mage
// in a robe) spoke as the bare record figure.
func TestLoadedSpeakersFollowTheirOwnPlacements(t *testing.T) {
	const mage = 0x17
	table := dressTable()
	table.Humans = append(slices.Clone(table.Humans.(dbCollection)),
		dbEntry{name: "M10_Merchant", params: figureRow(mage, 6, 0)})
	helm, robe := dressArmorCode(t, "PlateHelm"), dressArmorCode(t, "Robe")
	faces := map[int32]data.NPCFace{
		25: {Kind: data.NPCFigure, Dir: data.FigureDirManFighter, Face: dressPaladinFace,
			Tokens: npcTokenSet(data.NPCTokenHero, data.NPCTokenFace,
				data.NPCTokenNotFemale, data.NPCTokenNotMage)},
		52: {Kind: data.NPCFigure, Dir: data.FigureDirManMage, Face: 6,
			Tokens: npcTokenSet(data.NPCTokenMage, data.NPCTokenHuman,
				data.NPCTokenNotFemale, data.NPCTokenFace)},
	}
	paladin := mapload.PartyMember{PlayerCharacter: true, Class: dressPaladinType, CompanionNPC: 25,
		FigureDir: string(data.FigureDirManFighter), FigureFace: dressPaladinFace}
	merchant := mapload.PartyMember{PlayerCharacter: true, Class: mage, Mage: true,
		FigureDir: string(data.FigureDirManMage), FigureFace: 6}

	original := worldFixtureMap()
	original.Units = []alm.Unit{
		{UnitID: 11, ClassID: 64},
		{UnitID: 12, ClassID: 64},
		{UnitID: 13, ClassID: 1, ClassSubID: 25, Flags: 1},
		{UnitID: 14, ClassID: 64},
		{UnitID: 15, ClassID: mage},
	}
	var paladinWorn, merchantWorn [sim.EquipSlots]uint16
	paladinWorn[dressHelmSlot-1] = helm
	merchantWorn[dressRobeSlot-1] = robe
	world := func(ids ...sim.EntityID) *sim.World {
		t.Helper()
		var entities []sim.Entity
		for _, id := range ids {
			entities = append(entities, sim.Entity{ID: id, X: int32(id) + 2, Y: 3, HP: 10, MaxHP: 10,
				MapUnitID: original.Units[id].UnitID, Humanoid: id == 2 || id == 4})
		}
		w, err := sim.NewStockedWorld(1, sim.Bounds{Width: worldFixtureW, Height: worldFixtureH},
			sim.ModeCanonical, sim.Terrain{}, entities, nil, sim.Relations{}, nil,
			[]sim.Stock{{ID: 2, Equipped: paladinWorn}, {ID: 4, Equipped: merchantWorn}})
		if err != nil {
			t.Fatal(err)
		}
		return w
	}
	open := func(m *alm.Map, w *sim.World, state *SnapshotSAVDocument) *mapWorld {
		roster := map[sim.EntityID]mapload.PartyMember{2: paladin, 4: merchant}
		return openMission(&Mission{Number: 10, Map: m, World: w,
			Start: mapload.Start{Roster: roster}, savedDocument: state},
			table, nil, worldFixtureViewer(t, m), missionSource{}, nil, faces)
	}
	check := func(when string, mw *mapWorld) {
		t.Helper()
		var ids []sim.EntityID
		for _, a := range mw.speakerActors {
			ids = append(ids, a.id)
		}
		if !slices.Equal(ids, []sim.EntityID{2, 4}) {
			t.Errorf("%s: speaker candidates %v, want the two placed persons [2 4]", when, ids)
		}
		key := composeKey(t, mw, 52)
		if key.fig != (figureID{Dir: data.FigureDirManMage, Face: 6}) {
			t.Errorf("%s: npc52 composed figure %+v, want the man mage 6", when, key.fig)
		}
		if code, _ := key.eq.Code(dressRobeSlot); code != data.ItemCode(robe) {
			t.Errorf("%s: npc52 composed slot %d with %#x, want his own robe %#x; whole set %+v",
				when, dressRobeSlot, code, robe, key.eq)
		}
		key = composeKey(t, mw, 25)
		if !key.fig.Hero {
			t.Errorf("%s: npc25 composed %+v without the Hero bit of his own placement", when, key.fig)
		}
		if code, _ := key.eq.Code(dressHelmSlot); code != data.ItemCode(helm) {
			t.Errorf("%s: npc25 composed slot %d with %#x, want his own helm %#x; whole set %+v",
				when, dressHelmSlot, code, helm, key.eq)
		}
	}

	check("fresh", open(original, world(0, 1, 2, 3, 4), nil))

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
	check("after LOAD", open(&retained, world(1, 2, 3, 4), state))
}
