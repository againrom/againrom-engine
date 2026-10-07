package game

import (
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/formats/alm"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

func TestUnpairedSpeakersRetainNativeHeroModeWithoutPromotingOrdinaryPeople(t *testing.T) {
	const ordinary, hero sim.EntityID = 7, 9
	roster := map[sim.EntityID]mapload.PartyMember{
		ordinary: {PlayerCharacter: true, FigureDir: "mfighter", FigureFace: 1, Class: 3},
		hero:     {PlayerCharacter: true, FigureDir: "mfighter", FigureFace: 1, Class: 3},
	}
	entities := []sim.Entity{
		{ID: ordinary, TypeID: 3, HP: 10, MaxHP: 10},
		{ID: hero, TypeID: sim.HeroTypeID(false, false), HP: 10, MaxHP: 10},
	}
	m := &alm.Map{}
	placed := placedEntities(m, entities, roster, &SnapshotSAVDocument{})
	actors := missionSpeakers(m, nil, roster, nil, nil, placed)
	if len(actors) != 2 {
		t.Fatal("unpaired actors were lost", actors)
	}
	cast := speakerCast{actors: actors, alive: func(sim.EntityID) bool { return true }}
	rec := data.NPCFace{Tokens: npcTokenSet(data.NPCTokenHero, data.NPCTokenNotFemale, data.NPCTokenNotMage, data.NPCTokenFace), Face: 1}
	selected, found := cast.resolve(rec)
	if !found || selected.id != hero {
		t.Fatal("native Hero did not answer his record after losing an ALM pairing", selected, found)
	}
	rec.Tokens = npcTokenSet(data.NPCTokenHuman, data.NPCTokenNotHero, data.NPCTokenFace)
	selected, found = cast.resolve(rec)
	if !found || selected.id != ordinary {
		t.Fatal("ordinary person was promoted to Hero by the roster template", selected, found)
	}
}
