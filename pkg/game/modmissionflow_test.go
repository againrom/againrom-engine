package game

import (
	"encoding/json"
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/data"
	"againrom/pkg/mapload"
	"againrom/pkg/mod"
	"againrom/pkg/render/terrain"
	"againrom/pkg/sim"
)

func missionFlowData(t *testing.T, guest bool) mod.MissionData {
	t.Helper()
	src := `[[mission]]
kind = "campaign"
key = "first"
number = 10
AutoGetMission = 11
[[mission]]
kind = "campaign"
key = "detour"
number = 11
map = "maps/detour.alm"
AutoGetMission = 20
Payment = 75
`
	if guest {
		src += `[[mission]]
kind = "campaign"
key = "guest"
number = 20
guest = 22
chapter = 30
gold = 7
`
	}
	d, err := mod.ParseMissions("flow", mod.MissionsFile, []byte(src), func(string) (string, bool) { return "", false }, "en")
	if err != nil {
		t.Fatal(err)
	}
	d.Missions[1].MapBytes = synth.ALM(synth.ALMOptions{Width: 24, Height: 24, Name: "detour", Type7Payload: scriptPayload([]scriptNode{{label: "Drop location", opcode: dropActionOpcode, id: 99, values: []uint32{16, 16}, tags: []uint32{5, 6}}}, nil)})
	return d
}

func missionFlowFront(t *testing.T, guest bool, declarations ...mod.MissionData) *FrontEnd {
	t.Helper()
	f := winRouteFirstFront(t, mod.MissionData{})
	f.Archives.Containers = missionArchive(t, 10, 20, 30)
	f.Tiles = &terrain.Tileset{}
	f.Table.Mods.Set = mod.Set{Base: "rom1-en", Mods: []mod.SetEntry{{ID: "flow", Version: "1", Digest: "synthetic"}}}
	d := missionFlowData(t, guest)
	if len(declarations) != 0 {
		d = declarations[0]
	}
	if _, err := f.SetModMissions(d); err != nil {
		t.Fatal(err)
	}
	return f
}

func openMissionFlow(t *testing.T, f *FrontEnd, n int) {
	t.Helper()
	if _, _, _, _, _, _, _, _, _, _, err := f.MissionOpener(n)(); err != nil {
		t.Fatalf("open %d: %v", n, err)
	}
	if f.liveMission != n {
		t.Fatalf("plays %d, want %d", f.liveMission, n)
	}
}

func winMissionFlow(t *testing.T, f *FrontEnd, want int) {
	t.Helper()
	ms := f.live.mission.state
	next, line := f.FinishMissionWithRoster(f.liveMission, ms.Party, ms.World, ms.Start.IDs, ms.Start.Roster)
	if next != want {
		t.Fatalf("win %d: next %d, want %d: %s", f.liveMission, next, want, line)
	}
}

func missionFlowSave(t *testing.T, f *FrontEnd, auto uint32) []byte {
	t.Helper()
	b := guestMissionSave(t, f)
	c := autoGetCampaign(t, b)
	if c.AutoGetMission != auto {
		t.Fatalf("campaign %+v, want AutoGetMission %d", c, auto)
	}
	if !carriesModMark(b) {
		t.Fatal("no mod mark")
	}
	return b
}

func TestModMissionFlowMainPartySAVContinues(t *testing.T) {
	f := missionFlowFront(t, false)
	f.Table.Humans.(dbCollection)[28].params[5] = 0
	in := f.townInstall()
	hero, ok := in.character(mapload.CharacterSpec{Row: 28, Cells: map[int]string{0: "Wood Staff"}}, "Leader", nil)
	if !ok {
		t.Fatal("main hero definition was refused")
	}
	hero.ID, hero.StartingHero, hero.PlayerCharacter = "hero", true, true
	party := []mapload.PartyMember{hero}
	newGame, err := f.prepareNewGameWith(10, 0, party)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, _, _, _, _, _, _, _, _, err := newGame(); err != nil {
		t.Fatal(err)
	}
	missionFlowSave(t, f, 11)
	winMissionFlow(t, f, 11)
	openMissionFlow(t, f, 11)
	if f.liveParty[0].ID != party[0].ID || f.missionIdentity(11).guest {
		t.Fatal("added mission did not use the main party")
	}
	initialGold := f.live.world.Purse(sim.SelfSlot)
	actor := f.live.mission.ids[0]
	if !f.live.world.CheatAddItem(actor, sim.StackItem(sim.PlainItem(uint16(data.QuestDocumentCode)), 1)) {
		t.Fatal("loot was refused")
	}
	f.live.world.CheatAddGold(sim.SelfSlot, 37)
	b := missionFlowSave(t, f, 20)
	g := missionFlowFront(t, false)
	open, town, err := g.RestoreOriginal(b)
	if err != nil || town {
		t.Fatalf("LOAD: %t %v", town, err)
	}
	if _, _, _, _, _, _, _, _, _, _, err = open(); err != nil {
		t.Fatal(err)
	}
	missionFlowSave(t, g, 20)
	gained := missionFlowExperience(t, g)
	winMissionFlow(t, g, 20)
	openMissionFlow(t, g, 20)
	if got := g.live.world.Purse(sim.SelfSlot); got != initialGold+112 {
		t.Fatalf("added mission purse/payment = %d, want %d", got, initialGold+112)
	}
	e, _ := g.live.world.Entity(g.live.mission.ids[0])
	if e.SkillXP != gained {
		t.Fatalf("experience %v, want %v", e.SkillXP, gained)
	}
	items, _ := g.live.world.CarriedStacks(g.live.mission.ids[0])
	found := false
	for _, item := range items {
		found = found || item.Code == uint16(data.QuestDocumentCode)
	}
	if !found {
		t.Fatal("mission 11's loot did not reach mission 20")
	}
	missionFlowSave(t, g, 0xffffffff)
	winMissionFlow(t, g, 0)
	if !g.Town.Open() || g.Town.Chapter() != 30 || !g.Town.Done(11) {
		t.Fatalf("town %+v", g.Town)
	}
	raw := currentTownSave(t, g)
	h := missionFlowFront(t, false)
	if _, town, err := h.RestoreOriginal(raw); err != nil || !town {
		t.Fatalf("town LOAD: %t %v", town, err)
	}
}

func missionFlowExperience(t *testing.T, f *FrontEnd) [6]int32 {
	t.Helper()
	w, hero := f.live.world, f.live.mission.ids[0]
	if err := w.HeadlessPlace(hero, 8, 8); err != nil {
		t.Fatal(err)
	}
	h, _ := w.Entity(hero)
	before := h.SkillXP
	victim := h
	victim.Owner, victim.HP, victim.MaxHP, victim.GainsXP = 2, 1, 1, false
	victim.SkillXP = [6]int32{0, 1000000}
	enemy, err := w.CheatSummon(victim, nil, [sim.EquipSlots]sim.ItemInstance{}, h.X+1, h.Y)
	if err != nil {
		t.Fatal(err)
	}
	member := mapload.CloneParty(f.liveParty)[0]
	member.ID, member.StartingHero, member.PlayerCharacter = "enemy", false, false
	if f.live.mission.state.Start.Roster == nil {
		f.live.mission.state.Start.Roster = map[sim.EntityID]mapload.PartyMember{}
	}
	f.live.mission.state.Start.Roster[enemy] = member
	for i := 0; i < 2000; i++ {
		sim.Step(w, []sim.Command{sim.Attack(hero, enemy)})
		e, _ := w.Entity(enemy)
		if e.HP < 0 {
			break
		}
	}
	h, _ = w.Entity(hero)
	if h.SkillXP == before {
		e, _ := w.Entity(enemy)
		t.Fatalf("no XP: hero hp=%d damage=%d hit=%d reach=%d gains=%t slot=%d phase=%d targethp=%d at %d,%d vs %d,%d", h.HP, h.DamageBase, h.ToHit, h.Reach, h.GainsXP, h.XPSlot, h.AttackPhase, e.HP, h.X, h.Y, e.X, e.Y)
	}
	return h.SkillXP
}

func TestModMissionFlowGuestBeforeFirstTown(t *testing.T) {
	d := missionFlowData(t, false)
	d.Missions[0].Guest, d.Missions[0].Chapter = 22, 30
	f := missionFlowFront(t, false, d)
	main := f.Carried[0].ID
	oldTown, oldGuest, oldGold := f.Town, f.Town.guest, f.Town.Gold()
	newGame, err := f.prepareNewGameWith(10, 0, f.Carried)
	if err != nil {
		t.Fatal(err)
	}
	if f.Town != oldTown || f.Town.guest != oldGuest || f.Town.Gold() != oldGold {
		t.Fatal("preparing New Game changed the running campaign")
	}
	if _, _, _, _, _, _, _, _, _, _, err := newGame(); err != nil {
		t.Fatal(err)
	}
	if !f.missionIdentity(10).guest || f.liveParty[0].CompanionNPC != 22 {
		t.Fatal("New Game did not select the guest")
	}
	b := missionFlowSave(t, f, 11)
	g := missionFlowFront(t, false, d)
	open, town, err := g.RestoreOriginal(b)
	if err != nil || town {
		t.Fatalf("LOAD %t %v", town, err)
	}
	if _, _, _, _, _, _, _, _, _, _, err = open(); err != nil {
		t.Fatal(err)
	}
	winMissionFlow(t, g, 11)
	if g.Town.Open() || g.Carried[0].ID != main {
		t.Fatal("guest opened the town early or lost the main party")
	}
	openMissionFlow(t, g, 11)
	winMissionFlow(t, g, 20)
	openMissionFlow(t, g, 20)
	winMissionFlow(t, g, 0)
	if _, ok := guestMember(g); !ok {
		t.Fatal("the earlier guest did not join")
	}
}

func TestModMissionFlowGuestCampaignSAVCompletesForMainParty(t *testing.T) {
	f := missionFlowFront(t, true)
	openMissionFlow(t, f, 10)
	winMissionFlow(t, f, 11)
	openMissionFlow(t, f, 11)
	winMissionFlow(t, f, 20)
	mainID, gold := f.Carried[0].ID, f.Town.Gold()
	openMissionFlow(t, f, 20)
	if len(f.liveParty) != 1 || f.liveParty[0].CompanionNPC != 22 || f.live.world.Purse(sim.SelfSlot) != 7 {
		t.Fatal("campaign mission did not open with its guest")
	}
	f.live.world.CheatAddGold(sim.SelfSlot, 1000)
	if !f.live.world.CheatAddItem(f.live.mission.ids[0], sim.StackItem(sim.PlainItem(uint16(data.QuestDocumentCode)), 1)) {
		t.Fatal("guest loot refused")
	}
	b := missionFlowSave(t, f, 0xffffffff)
	g := missionFlowFront(t, true)
	open, town, err := g.RestoreOriginal(b)
	if err != nil || town {
		t.Fatalf("guest LOAD: %t %v", town, err)
	}
	if _, _, _, _, _, _, _, _, _, _, err = open(); err != nil {
		t.Fatal(err)
	}
	winMissionFlow(t, g, 0)
	if !g.Town.Open() || g.Town.Chapter() != 30 || !g.Town.Done(20) || g.Town.Gold() != gold+150 || g.Carried[0].ID != mainID {
		t.Fatalf("waiting party: gold %d want %d chapter %d party %s", g.Town.Gold(), gold+150, g.Town.Chapter(), winRouteParty(g.Carried))
	}
	if joined, ok := guestMember(g); !ok {
		t.Fatal("the town did not grant the guest")
	} else {
		found := false
		for _, item := range joined.CarriedItems {
			found = found || item.Code == uint16(data.QuestDocumentCode)
		}
		if !found {
			t.Fatal("the joining guest lost her carried item")
		}
	}
}

func TestModMissionFlowGuestSideAndArraySAV(t *testing.T) {
	for _, number := range []int{11, 20} {
		d := missionFlowData(t, number == 20)
		if number == 11 {
			d.Missions[1].Guest, d.Missions[1].Chapter = 22, 30
		} else {
			d.Missions[2].Guest, d.Missions[2].Chapter, d.Missions[2].Carry = 0, 0, ""
			d.Missions[2].Party = []mod.GuestMember{{Character: "npc22"}}
		}
		f := missionFlowFront(t, false, d)
		openMissionFlow(t, f, 10)
		winMissionFlow(t, f, 11)
		if number == 20 {
			openMissionFlow(t, f, 11)
			winMissionFlow(t, f, 20)
		}
		main := f.Carried[0].ID
		openMissionFlow(t, f, number)
		if !f.missionIdentity(number).guest || f.liveParty[0].CompanionNPC != 22 {
			t.Fatal("guest party was not selected")
		}
		auto := uint32(20)
		if number == 20 {
			auto = 0xffffffff
		}
		b := missionFlowSave(t, f, auto)
		g := missionFlowFront(t, false, d)
		open, town, err := g.RestoreOriginal(b)
		if err != nil || town {
			t.Fatalf("guest mission %d LOAD: %t %v", number, town, err)
		}
		if _, _, _, _, _, _, _, _, _, _, err = open(); err != nil {
			t.Fatal(err)
		}
		if number == 11 {
			winMissionFlow(t, g, 20)
			openMissionFlow(t, g, 20)
		}
		winMissionFlow(t, g, 0)
		if g.Carried[0].ID != main || !g.Town.Done(number) || !g.Town.Open() {
			t.Fatalf("mission %d did not complete for the main party", number)
		}
	}
}

func TestModMissionFlowEndsAtTownWithoutAutomaticSuccessor(t *testing.T) {
	for _, added := range []bool{false, true} {
		d := missionFlowData(t, false)
		if added {
			d.Missions[1].AutoGetMission = nil
		} else {
			stop := -1
			d.Missions = d.Missions[:1]
			d.Missions[0].AutoGetMission = &stop
		}
		f := missionFlowFront(t, false, d)
		f.resetSessionForNewGame()
		openMissionFlow(t, f, 10)
		if added {
			winMissionFlow(t, f, 11)
			openMissionFlow(t, f, 11)
		}
		winMissionFlow(t, f, 0)
		if !f.Town.Open() || f.Town.currentMain() != 20 || !f.Town.available[20] {
			t.Fatalf("added=%t: town %t main %d available %v", added, f.Town.Open(), f.Town.currentMain(), f.Town.available)
		}
		b := currentTownSave(t, f)
		g := &FrontEnd{InstallResources: f.InstallResources}
		if _, town, err := g.RestoreOriginal(b); err != nil || !town {
			t.Fatalf("town LOAD %t %v", town, err)
		}
		openMissionFlow(t, g, 20)
	}
}

func TestModMissionFlowPendingGuestsKeepTheirSeparateGrants(t *testing.T) {
	a, b := guestDecl(), guestDecl()
	b.Key, b.Number, b.Guest, b.Chapter = "later", 27, 23, 40
	a.Kind, b.Kind = mod.MissionCampaign, mod.MissionCampaign
	member := func(n int) mapload.PartyMember {
		return mapload.PartyMember{ID: (mapload.Membership{Kind: mapload.CompanionMember, Identity: n}).ID(0), CompanionNPC: n, PlayerCharacter: true}
	}
	x, y := member(22), member(23)
	g := &guestMission{mission: b, start: y, member: &y, previous: &guestMission{mission: a, start: x, member: &x}}
	raw, err := json.Marshal(currentSessionData{Guest: g.record()})
	if err != nil {
		t.Fatal(err)
	}
	var s currentSessionData
	if err := json.Unmarshal(raw, &s); err != nil {
		t.Fatal(err)
	}
	restored, err := restoreGuest(&s, mapload.ModContext{Missions: mod.MissionData{Missions: []mod.Mission{a, b}}})
	if err != nil {
		t.Fatal(err)
	}
	town := &Town{guest: restored}
	if p, ok := town.takeGuestJoining(30, 22); !ok || p.CompanionNPC != 22 || town.guest == nil {
		t.Fatal("earlier grant lost the later guest")
	}
	if p, ok := town.takeGuestJoining(40, 23); !ok || p.CompanionNPC != 23 || town.guest != nil {
		t.Fatal("later grant was not consumed")
	}
}
