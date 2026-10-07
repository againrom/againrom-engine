package game

import (
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/mapload"
	"againrom/pkg/render/terrain"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

func joinedLiveScript(t *testing.T, id sim.EntityID) *sim.Script {
	t.Helper()
	s, err := sim.NewScript(
		[]sim.ScriptCheck{
			{Op: sim.ScriptCheckConstant, Register: 0, Args: [10]int32{1}},
			{Op: sim.ScriptCheckConstant, Register: 1, Args: [10]int32{1}},
		},
		[]sim.ScriptInstant{{Op: sim.ScriptInstantGiveUnit, Unit: id, HasUnit: true,
			Player: sim.SelfSlot, HasPlayer: true}},
		[]sim.ScriptTrigger{{Pairs: [3]sim.ScriptPair{{Left: 0, Right: 1, Cmp: sim.ScriptCmpEQ, Used: true}},
			Instants: [4]int32{0, sim.ScriptNone, sim.ScriptNone, sim.ScriptNone}, Once: true}},
	)
	if err != nil {
		t.Fatalf("NewScript: %v", err)
	}
	return s
}

func TestScriptHandoverJoinsEveryOrdinaryHeroSurfaceInTheSameTick(t *testing.T) {
	const joinID sim.EntityID = 0
	world, err := sim.NewScriptedWorld(1, sim.Bounds{Width: 8, Height: 8}, sim.ModeCanonical, nil,
		[]sim.Entity{
			{ID: joinID, X: 2, Y: 2, HP: 30, MaxHP: 30, Owner: 2, TypeID: sim.HumanTypeID,
				Skill:   [data.SkillSlots]int32{0, 25, 3, 0, 0, 1},
				SkillXP: [data.SkillSlots]int32{0, 9834, 331, 0, 0, 100}},
			{ID: 1, X: 1, Y: 1, HP: 40, MaxHP: 40, Owner: sim.SelfSlot, TypeID: sim.HumanTypeID},
		}, joinedLiveScript(t, joinID))
	if err != nil {
		t.Fatalf("NewScriptedWorld: %v", err)
	}
	primary := mapload.PartyMember{ID: "hero", Name: "Danath", StartingHero: true, PlayerCharacter: true}
	brian := mapload.PartyMember{ID: "join:0", Name: "Brian", PlayerCharacter: true,
		CompanionNPC: 25, Class: 5, Hero: data.Hero{Skill: [data.SkillSlots]int32{0, 25, 3, 0, 0, 1}}}
	mission := &Mission{Number: 40, World: world, Party: []mapload.PartyMember{primary},
		Start: mapload.Start{IDs: []sim.EntityID{1}, Roster: map[sim.EntityID]mapload.PartyMember{joinID: brian}}}
	v, err := ui.NewViewer("joined", terrain.Grid{Width: 8, Height: 8, Tiles: make([]uint16, 64)}, &terrain.Tileset{})
	if err != nil {
		t.Fatalf("NewViewer: %v", err)
	}
	mw := openMission(mission, nil, nil, v, nil, nil, nil)
	joined := false
	for i := 0; i < 32; i++ {
		if got := mw.tick(); got != 0 {
			t.Fatalf("tick applied %d player orders, want none", got)
		}
		if entity, ok := mw.entity(joinID); ok && entity.Owner == sim.SelfSlot {
			joined = true
			// The party assertion immediately below is deliberately in the same
			// iteration that first observes the simulation handover.
			break
		}
	}
	if !joined {
		t.Fatal("script did not hand Brian to the player")
	}
	if len(mw.mission.party) != 2 || len(mw.mission.ids) != 2 || mw.mission.ids[1] != joinID {
		t.Fatalf("live party/ids = %#v/%v, want primary then Brian/%d", mw.mission.party, mw.mission.ids, joinID)
	}
	got := mw.mission.party[1]
	if got.Name != "Brian" || got.CompanionNPC != 25 || got.Carry == nil ||
		got.Carry.SkillXP != ([data.SkillSlots]int32{0, 9834, 331, 0, 0, 100}) {
		t.Fatalf("joined Brian lost live identity/xp: %+v", got)
	}
	if !mw.isGuarded(joinID) {
		t.Fatal("joined Brian is not an ordinary guarded player character")
	}
	found := false
	for _, draw := range mw.entityDraws() {
		if draw.ID == uint32(joinID) {
			found = true
			if !draw.PlayerCharacter || draw.Name != "Brian" {
				t.Fatalf("joined draw = player %v name %q", draw.PlayerCharacter, draw.Name)
			}
		}
	}
	if !found {
		t.Fatal("joined actor has no live map projection")
	}
	mw.switchInventorySubject(uint32(joinID))
	if !mw.invSubjectSet || mw.invSubject.ID != uint32(joinID) {
		t.Fatalf("joined inventory subject = set %v id %d", mw.invSubjectSet, mw.invSubject.ID)
	}

	front := &FrontEnd{CampaignSession: CampaignSession{live: mw, liveMission: 40, liveParty: []mapload.PartyMember{primary}}}
	if live := front.LiveParty(); len(live) != 2 || live[1].Name != "Brian" {
		t.Fatalf("LiveParty = %#v, want the joined hero immediately", live)
	}
	snap, _, err := front.Snapshot(true)
	if err != nil {
		t.Fatalf("Snapshot: %v", err)
	}
	if len(snap.Party) != 1 || snap.Party[0].ID != "hero" {
		t.Fatalf("mission-save mint prefix = %#v, want only the entry hero", snap.Party)
	}
	var saved sim.World
	if err := saved.UnmarshalBinary(snap.World); err != nil {
		t.Fatalf("saved world: %v", err)
	}
	if survivors := saved.BoundarySurvivors(sim.SelfSlot); len(survivors) != 2 {
		t.Fatalf("saved world survivors = %v, want primary and joined actor", survivors)
	}
}

func TestJoinedHeroSyncInstallsEveryMissionLocalFigureIdentity(t *testing.T) {
	cases := []struct {
		name string
		npc  int
		dir  data.FigureDir
		face int
	}{
		{name: "Brian fixed npc25", npc: 25, dir: data.FigureDirManFighter, face: 1},
		{name: "npc23 Danath", npc: 23, dir: data.FigureDirManFighter, face: 5},
		{name: "npc23 Naira", npc: 23, dir: data.FigureDirWomanFighter, face: 1},
		{name: "npc24 Fergard", npc: 24, dir: data.FigureDirManMage, face: 3},
		{name: "npc24 Reniesta", npc: 24, dir: data.FigureDirWomanMage, face: 1},
		{name: "npc24 Danath", npc: 24, dir: data.FigureDirManFighter, face: 5},
		{name: "npc24 Naira", npc: 24, dir: data.FigureDirWomanFighter, face: 1},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			const joinID sim.EntityID = 0
			world, err := sim.NewScriptedWorld(1, sim.Bounds{Width: 2, Height: 2}, sim.ModeCanonical, nil,
				[]sim.Entity{{ID: joinID, HP: 1, MaxHP: 1, Owner: sim.SelfSlot, TypeID: sim.HumanTypeID}}, nil)
			if err != nil {
				t.Fatalf("NewScriptedWorld: %v", err)
			}
			member := mapload.PartyMember{ID: "join:0", Name: tc.name, CompanionNPC: tc.npc,
				PlayerCharacter: true, FigureDir: string(tc.dir), FigureFace: tc.face}
			state := &Mission{World: world,
				Start: mapload.Start{Roster: map[sim.EntityID]mapload.PartyMember{joinID: member}}}
			mw := &mapWorld{world: world, figures: map[sim.EntityID]figureID{
				joinID: {Dir: data.FigureDirManMage, Face: 9}},
				mission: &missionNotices{state: state}}

			mw.syncJoinedHeroes()

			if len(mw.mission.party) != 1 || len(mw.mission.ids) != 1 || mw.mission.ids[0] != joinID {
				t.Fatalf("joined party/ids = %#v/%v", mw.mission.party, mw.mission.ids)
			}
			want := figureID{Dir: tc.dir, Face: tc.face, Hero: true}
			if got := mw.figures[joinID]; got != want {
				t.Fatalf("joined figure = %+v, want %+v", got, want)
			}
		})
	}
}
