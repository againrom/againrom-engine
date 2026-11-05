package game

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"maps"
	"reflect"
	"testing"

	"againrom/pkg/mapload"
	"againrom/pkg/render/terrain"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

func TestInstallUnitNamesKeepEveryLoadedLine(t *testing.T) {
	for _, count := range []int{0, 3, 81, 108, 119} {
		t.Run(fmt.Sprint(count), func(t *testing.T) {
			lines := map[int]string{}
			for i := 0; i < count; i++ {
				if i%7 != 5 {
					lines[i] = fmt.Sprintf("Fixture type %d", i)
				}
			}
			loaded := LoadInstallWords(installFixture{UnitNameTextPath: textFile(count, lines)}, TextCode{})
			words := loaded.Words()
			if len(words.UnitNames) != count {
				t.Errorf("loaded %d lines, retained %d", count, len(words.UnitNames))
			}
			for i := -1; i <= count; i++ {
				want := lines[i]
				got, present := ui.DisplayName(ui.InspectionUnit, "", i, &words)
				if got != want || present != (want != "") {
					t.Errorf("line %d = %q/%t, want %q", i, got, present, want)
				}
			}
		})
	}
}

func TestTypeNameAboveFirstTableReachesTheActor(t *testing.T) {
	const id sim.EntityID = 17
	actor := sim.Entity{ID: id, Class: 99, TypeID: 99, X: 1, Y: 1, MapUnitID: 86, HP: 9, MaxHP: 31,
		SourceBinding: sim.SourceBinding{Class: 1, TokenRow: 1, TypeID: 99}}
	w, err := sim.NewWorld(17, sim.Bounds{Width: 8, Height: 8}, sim.ModeCanonical, make([]byte, 64), []sim.Entity{actor})
	if err != nil {
		t.Fatal(err)
	}
	v, err := ui.NewViewer("type caption", terrain.Grid{Width: 8, Height: 8, Tiles: make([]uint16, 64)}, &terrain.Tileset{})
	if err != nil {
		t.Fatal(err)
	}
	words := LoadInstallWords(installFixture{UnitNameTextPath: textFile(119, map[int]string{99: "Fixture necromancer"})}, TextCode{}).Words()
	v.SetWords(words)
	mw := newMapWorld(w, nil, &terrain.UnitSet{Classes: map[int32]*terrain.UnitClass{99: {Name: "internal creature"}}}, v)
	mw.mission = &missionNotices{table: &mapload.Table{Units: dbCollection{{}, {name: "internal creature"}}}}
	mw.actorNames = map[sim.EntityID]string{id: "internal creature"}
	hash := w.Hash()
	if got := mw.shownName(id); got != "Fixture necromancer" {
		t.Errorf("type 99 shown name %q", got)
	}
	mw.authoredNames = missionUnitNames(installFixture{mainPrefix + "text/battle/m10/units.txt": []byte("86=Authored sorcerer\r\n")}, TextCode{}, 10)
	if got := mw.shownName(id); got != "Authored sorcerer" {
		t.Errorf("authored name %q", got)
	}
	if w.Hash() != hash || mw.actorNames[id] != "internal creature" {
		t.Fatal("name projection changed World or stored name")
	}
}

func TestCampaignHireNamesUseTheTypeWithoutChangingState(t *testing.T) {
	f := cellStateFront(t)
	humans, units := f.Table.Humans.(dbCollection), f.Table.Units.(dbCollection)
	keys := []string{"2MercenaryXBowman70", "2MercenaryKnight70", "MercenaryDruid70", "MercenaryTroll70", "MercenaryOrc70", "2MercenaryMage70", "CatapultBase"}
	types := []int32{15, 7, 100, 102, 101, 24, 30}
	lines := map[int]string{33: "Fixture hero type"}
	for i, key := range keys {
		keyID := int32(10000 + 10*(i+1))
		lines[int(types[i])] = fmt.Sprintf("Fixture hire type %d", types[i])
		if i == 0 || i == 1 || i == 5 {
			p := mercenaryHumanRow(types[i], 1, 0, 10)
			p[24] = keyID
			humans = append(humans, dbEntry{name: key, params: p})
		} else {
			p := make([]int32, 56)
			for j := range p {
				p[j] = -1
			}
			p[0], p[4], p[29], p[30], p[31], p[32], p[55] = 20, 100, types[i], 1, 1, 1, keyID
			units = append(units, dbEntry{name: key, params: p})
		}
		f.Units.Classes[types[i]] = worldFixtureArt(16, 16, 8, 16, 3, 3, 1)
	}
	f.Table.Humans, f.Table.Units, f.Table.UnitKeys = humans, units, mapload.ServerUnitKeys
	f.Words = LoadInstallWords(installFixture{UnitNameTextPath: textFile(119, lines)}, TextCode{}).Words()
	town := &Town{second: newSecondCampaign()}
	town.second.bank[768] = 70
	for id := 1; id <= len(keys); id++ {
		town.second.bank[511+id], town.second.bank[531+id] = 1, 1
	}
	party, err := (secondCampaignRules{}).missionParty(nil, town, f.Table)
	if err != nil || len(party) != len(keys) {
		t.Fatal("campaign party", len(party), err)
	}
	for i, p := range party {
		built, ok := mapload.CampaignActor(f.Table, mapload.CampaignActorSpec{Profile: keyedCampaignActor, Identity: i + 1, Stage: 70, Hire: true}, nil)
		if !ok || p.Name != keys[i] || !reflect.DeepEqual(p, built) {
			t.Fatalf("hire %d did not use its template: %+v", i+1, p)
		}
	}
	app := f.App("campaign type captions")
	app.Layout(1024, 768)
	if err := app.OpenMission(f.MissionOpenerWith(10, party)); err != nil {
		t.Fatal(err)
	}
	mw := f.live
	beforeParty := mapload.OwnParty(mw.mission.party)
	beforeNames := maps.Clone(mw.actorNames)
	beforeChars := maps.Clone(mw.chars)
	hash := mw.world.Hash()
	save := func() []byte {
		t.Helper()
		snapshot, label, err := f.Snapshot(true)
		if err != nil {
			t.Fatal(err)
		}
		b, _, err := f.playerMissionSave(snapshot, label)
		if err != nil {
			t.Fatal(err)
		}
		return b
	}
	before := save()
	for i, id := range mw.mission.ids {
		got := mw.shownName(id)
		if want := lines[int(types[i])]; got != want {
			t.Errorf("hire %d shown=%q, want %q", i+1, got, want)
		}
		actor, found := mw.world.Entity(id)
		if !found {
			t.Fatal("missing campaign actor", id)
		}
		name, index, _ := mw.drawnName(&actor, f.Units.Classes[actor.Class], &f.Words)
		if got, ok := ui.PanelSubjectName(ui.PanelSubject{Name: name, UnitNameIndex: index, Words: f.Words}); !ok || got != lines[int(types[i])] {
			t.Errorf("hire %d card=%q/%t", i+1, got, ok)
		}
	}
	after := save()
	if mw.world.Hash() != hash || !reflect.DeepEqual(beforeParty, mw.mission.party) || !maps.Equal(beforeNames, mw.actorNames) || !maps.Equal(beforeChars, mw.chars) || !bytes.Equal(before, after) {
		t.Fatal("caption projection changed party, actor names, World or SAV bytes")
	}
	t.Logf("World=%x SAV=%x bytes=%d", hash, sha256.Sum256(before), len(before))
	if hash != 0x0434ea7a7b4b64b0 || fmt.Sprintf("%x", sha256.Sum256(before)) != "41bdd0922c4a59da9c346244b187be5a4f2dcf5191c1cc52b5a8369eb0f947b3" {
		t.Fatal("World or SAV differs from the unchanged production fixture")
	}
	id := mw.mission.ids[0]
	char := mw.chars[id]
	custom := char
	custom.Name = "Authored hire name"
	mw.chars[id] = custom
	if got := mw.shownName(id); got != custom.Name {
		t.Errorf("explicit hire name %q", got)
	}
	mw.chars[id] = char
	words := mw.view.Words()
	words.UnitNames[15] = ""
	mw.view.SetWords(words)
	if got := mw.shownName(id); got != "" {
		t.Errorf("missing type exposed an internal key %q", got)
	}
	f.Table.Mods.Characters = []mapload.ModCharacter{{Mod: "fixture", Rows: []string{keys[0]}, Name: "Mod hire name"}}
	if got := mw.shownName(id); got != "Mod hire name" {
		t.Errorf("mod hire name %q", got)
	}
}

func TestFirstPartyOwnNameKeepsItsDisplay(t *testing.T) {
	f := cellStateFront(t)
	f.Words = LoadInstallWords(installFixture{UnitNameTextPath: textFile(81, map[int]string{33: "Fixture first type"})}, TextCode{}).Words()
	party := []mapload.PartyMember{{Name: "Authored first hero", Class: 33, PlayerCharacter: true}}
	party, err := (firstCampaignRules{}).missionParty(party, nil, f.Table)
	if err != nil {
		t.Fatal(err)
	}
	if err := f.App("first name control").OpenMission(f.MissionOpenerWith(10, party)); err != nil {
		t.Fatal(err)
	}
	if got := f.live.shownName(f.live.mission.ids[0]); got != "Authored first hero" {
		t.Fatalf("first party shown name %q", got)
	}
}
