package game

import (
	"testing"

	"againrom/pkg/mapload"
	"againrom/pkg/render/terrain"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

func npcLineFixtureWorld(t *testing.T, id sim.EntityID, typeID int32, word uint16, own string) *mapWorld {
	t.Helper()
	actor := sim.Entity{ID: id, Class: typeID, TypeID: typeID, X: 1, Y: 1, HP: 9, MaxHP: 31,
		SourceBinding: sim.SourceBinding{Class: 1, TokenRow: 1, TypeID: uint16(typeID)}}
	w, err := sim.NewWorld(17, sim.Bounds{Width: 8, Height: 8}, sim.ModeCanonical, make([]byte, 64), []sim.Entity{actor})
	if err != nil {
		t.Fatal(err)
	}
	v, err := ui.NewViewer("npc line", terrain.Grid{Width: 8, Height: 8, Tiles: make([]uint16, 64)}, &terrain.Tileset{})
	if err != nil {
		t.Fatal(err)
	}
	words := LoadInstallWords(installFixture{
		UnitNameTextPath: textFile(10, map[int]string{1: "Fixture peasant"}),
		NPCNameTextPath:  textFile(2200, map[int]string{2111: "Fixture elder", 0: "Fixture mercenary", 6: "Fixture catapult"}),
	}, TextCode{}).Words()
	v.SetWords(words)
	mw := newMapWorld(w, nil, &terrain.UnitSet{Classes: map[int32]*terrain.UnitClass{typeID: {Name: "internal person"}}}, v)
	mw.mission = &missionNotices{table: &mapload.Table{Units: dbCollection{{}, {name: "internal person"}}}}
	mw.mission.npcKeys = map[sim.EntityID]uint16{id: word}
	mw.actorNames = map[sim.EntityID]string{id: own}
	if own != "" {
		mw.actorNames[id] = "internal person"
	}
	return mw
}

func TestPlacedNPCWordNamesTheActorFromTheNPCTable(t *testing.T) {
	const id sim.EntityID = 5
	for _, c := range []struct {
		name string
		word uint16
		want string
	}{
		{"headman", 2112, "Fixture elder"},
		{"empty line", 2000, "Fixture peasant"},
		{"no word", 0, "Fixture peasant"},
		{"word beyond the table", 9000, "Fixture peasant"},
	} {
		mw := npcLineFixtureWorld(t, id, 1, c.word, "internal person")
		hash := mw.world.Hash()
		if got := mw.shownName(id); got != c.want {
			t.Errorf("%s: shown %q, want %q", c.name, got, c.want)
		}
		if mw.world.Hash() != hash || mw.actorNames[id] != "internal person" {
			t.Errorf("%s: the projection changed the World or the stored name", c.name)
		}
	}
	mw := npcLineFixtureWorld(t, id, 1, 2112, "internal person")
	mw.actorNames[id] = "Own name"
	if got := mw.shownName(id); got != "Own name" {
		t.Errorf("a stored own name shows %q", got)
	}
	mw = npcLineFixtureWorld(t, id, 1, 2112, "internal person")
	mw.mission.npcKeys = nil
	if got := mw.shownName(id); got != "Fixture peasant" {
		t.Errorf("a game without NPC words shows %q", got)
	}
	mw = npcLineFixtureWorld(t, id, 1, 2112, "internal person")
	mw.chars = map[sim.EntityID]ui.UnitCharacter{id: {Name: "Fixture peasant", Known: true}}
	if got := mw.shownName(id); got != "Fixture elder" {
		t.Errorf("a roster person named by its type label shows %q", got)
	}
	mw.chars = map[sim.EntityID]ui.UnitCharacter{id: {Name: "Hero name"}}
	if got := mw.shownName(id); got != "Hero name" {
		t.Errorf("a named character shows %q", got)
	}
}

func TestHireIDNamesItsNPCLine(t *testing.T) {
	words := LoadInstallWords(installFixture{
		UnitNameTextPath: textFile(120, map[int]string{15: "Fixture type", 100: "Fixture druid"}),
		NPCNameTextPath:  textFile(30, map[int]string{0: "Fixture mercenary", 6: "Fixture catapult", 7: "Fixture eight"}),
	}, TextCode{}).Words()
	in := &InstallResources{Table: &mapload.Table{}}
	for _, c := range []struct {
		id    int
		class int
		want  string
	}{
		{1, 15, "Fixture mercenary"},
		{7, 100, "Fixture catapult"},
		{8, 100, "Fixture druid"},
		{2, 15, "Fixture type"},
	} {
		p := mapload.PartyMember{Temporary: true, CampaignHire: uint8(c.id), Class: int32(c.class)}
		if got := in.definitionName(p, c.class, &words, ""); got != c.want {
			t.Errorf("hire %d shown %q, want %q", c.id, got, c.want)
		}
	}
	// A first-game party member has no hire ID and keeps its class row.
	if got := in.definitionName(mapload.PartyMember{Class: 15}, 15, &words, ""); got != "Fixture type" {
		t.Errorf("party member shown %q", got)
	}
}
