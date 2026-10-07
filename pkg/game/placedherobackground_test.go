package game

import (
	"image/color"
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/data"
	"againrom/pkg/formats/alm"
	"againrom/pkg/formats/reg"
	"againrom/pkg/mapload"
	"againrom/pkg/render/terrain"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// A placed person's information portrait draws the warrior hero background
// only when the actor's own type id takes the hero arm (UNIT-PICT-035,
// HERO-FIGURE-144). The loader makes every placement's roster template a
// PlayerCharacter, so that flag cannot decide it: a zero-mode Humans
// placement keeps its table type id and an exact Hero NPC placement takes the
// player-character band (PARTY-M20-031). The map script's own hand-over
// changes the owner, not the construction mode.
func TestPlacedPersonPortraitDrawsHeroBackgroundOnlyForAHeroTypeID(t *testing.T) {
	const (
		peasantType, peasantFace = 1, 17
		paladinType, paladinFace = 5, 1
		paladinServerID          = 777
		heroFace                 = 5
		mapSide                  = 128
	)
	row := func(name string, typeID, face, serverID int32) dbEntry {
		p := humansParams(20, 0, face)
		p[16], p[18], p[24] = typeID, 0, serverID
		return dbEntry{name: name, params: p}
	}
	npcReg, err := reg.Parse(synth.Reg(kindRoot, []synth.RegNode{
		{Name: "npc25", Kind: kindDir, Children: []synth.RegNode{
			{Name: "Flags", Kind: 0x00, Str: "Hero,Human"},
			{Name: "DataBinID", Kind: kindInt, Int: paladinServerID},
		}},
	}))
	if err != nil {
		t.Fatal(err)
	}
	table := &mapload.Table{
		Humans: dbCollection{{},
			row("Peasant", peasantType, peasantFace, -1),
			row("Paladin", paladinType, paladinFace, paladinServerID)},
		NPC: data.LoadNPCDefs(npcReg),
	}
	const peasant, paladin sim.EntityID = 0, 1
	m := &alm.Map{Width: mapSide, Height: mapSide,
		Tiles: make([]uint16, mapSide*mapSide), Overlay: make([]uint8, mapSide*mapSide),
		Units: []alm.Unit{
			{X: 0x0C80, Y: 0x0C80, ClassID: peasantType, UnitID: 1, Owner: 2},
			{X: 0x0E80, Y: 0x0C80, ClassID: paladinType, ClassSubID: 25, Flags: 1, UnitID: 2, Owner: 2},
		}}
	party := []mapload.PartyMember{{ID: "hero", Name: "Hero", StartingHero: true, PlayerCharacter: true,
		Class: 1, FigureDir: string(data.FigureDirManFighter), FigureFace: heroFace}}
	// The map's own program hands both placed people to the player on its first
	// pass, through the ordinary script instant.
	give := func(id sim.EntityID) sim.ScriptInstant {
		return sim.ScriptInstant{Op: sim.ScriptInstantGiveUnit, Unit: id, HasUnit: true,
			Player: sim.SelfSlot, HasPlayer: true}
	}
	script, err := sim.NewScript(
		[]sim.ScriptCheck{
			{Op: sim.ScriptCheckConstant, Register: 0, Args: [10]int32{1}},
			{Op: sim.ScriptCheckConstant, Register: 1, Args: [10]int32{1}},
		},
		[]sim.ScriptInstant{give(peasant), give(paladin)},
		[]sim.ScriptTrigger{{Pairs: [3]sim.ScriptPair{{Left: 0, Right: 1, Cmp: sim.ScriptCmpEQ, Used: true}},
			Instants: [4]int32{0, 1, sim.ScriptNone, sim.ScriptNone}, Once: true}},
	)
	if err != nil {
		t.Fatal(err)
	}
	world, start, err := mapload.StartMissionScripted(m, table, mapload.DifficultyNormal, party, script)
	if err != nil {
		t.Fatal(err)
	}
	if len(start.IDs) != 1 || start.Roster[peasant].PlayerCharacter != true || start.Roster[paladin].PlayerCharacter != true {
		t.Fatalf("fixture start: ids %v roster %+v; want one hero and two PlayerCharacter templates", start.IDs, start.Roster)
	}
	hero := start.IDs[0]

	// Every body sheet paints only pixel (0,0). The background sheet paints all
	// four pixels, so pixel (1,1) shows the background or nothing.
	background := color.RGBA{B: 0xff, A: 0xff}
	body := invSparseSheet(color.RGBA{R: 0xff}, 0)
	src := missionSource{
		graphicsPrefix + data.ItemFigureBasePath(data.FigureDirManFighter, peasantFace): body,
		graphicsPrefix + data.ItemFigureBasePath(data.FigureDirManFighter, paladinFace): body,
		graphicsPrefix + data.ItemFigureBasePath(data.FigureDirManFighter, heroFace):    body,
		graphicsPrefix + "interface/heroback/backm.256":                                 invSparseSheet(color.RGBA{B: 0xff}, 0, 1, 2, 3),
	}
	v, err := ui.NewViewer("placed hero background", terrain.Grid{Width: mapSide, Height: mapSide,
		Tiles: m.Tiles}, &terrain.Tileset{})
	if err != nil {
		t.Fatal(err)
	}
	ms := &Mission{Number: 10, Map: m, World: world, Start: start, Party: mapload.OwnParty(party)}
	live := openMission(ms, table, nil, v, src, nil, nil)

	check := func(when string) {
		t.Helper()
		for _, tc := range []struct {
			name string
			id   sim.EntityID
			want color.RGBA
		}{
			{"zero-mode peasant", peasant, color.RGBA{}},
			{"Hero NPC placement", paladin, background},
			{"entering hero", hero, background},
		} {
			e, ok := live.entity(tc.id)
			if !ok {
				t.Fatalf("%s: %s is not in the world", when, tc.name)
			}
			inspected := live.inspectionUnitPicture(uint32(tc.id))
			if inspected == nil || inspected.RGBAAt(0, 0) != (color.RGBA{R: 0xff, A: 0xff}) {
				t.Fatalf("%s: %s inspection portrait lost its body: %v", when, tc.name, inspected)
			}
			if got := inspected.RGBAAt(1, 1); got != tc.want {
				t.Errorf("%s: %s (type id %#x) inspection portrait behind the body = %v, want %v",
					when, tc.name, e.TypeID, got, tc.want)
			}
			if tc.id == hero {
				continue // the party's own subject reaches the box through its inventory figure
			}
			if selected := live.unitPicture(tc.id, e.Class); selected == nil || selected.RGBAAt(1, 1) != tc.want {
				t.Errorf("%s: %s selected portrait behind the body = %v, want %v", when, tc.name, selected, tc.want)
			}
		}
	}
	check("at mission open")

	for i := 0; i < 32; i++ {
		live.tick()
		p, _ := live.entity(peasant)
		q, _ := live.entity(paladin)
		if p.Owner == sim.SelfSlot && q.Owner == sim.SelfSlot {
			break
		}
	}
	if p, _ := live.entity(peasant); p.Owner != sim.SelfSlot {
		t.Fatal("the map script did not hand the peasant to the player")
	}
	if len(live.mission.ids) != 2 || live.mission.ids[1] != paladin {
		t.Fatalf("live party ids = %v, want the hero and the Hero NPC joiner", live.mission.ids)
	}
	check("after the hand-over")
}
