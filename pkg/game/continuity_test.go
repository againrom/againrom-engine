package game

// The continuity hotfix, the front end's half: dismissing a WON mission's
// banner is what carries the party forward, opens the campaign's own
// declared successor where one can be opened, and otherwise names what
// follows.
//
// The loader's half — what a carry is and what a mint does with one — is
// decided in pkg/mapload/carry_test.go. What is decided HERE is the wiring:
// which answer of the driver's is recognised as a win, what the front end
// then remembers, which successor the campaign's own key names, and which of
// the six sentences the map list — or the successor's own map screen, for
// the one that never shows a sentence at all — is given.

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/data"
	"againrom/pkg/formats/reg"
	"againrom/pkg/mapload"
	"againrom/pkg/render/terrain"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
	"againrom/pkg/vfs"
)

// The experience the finished mission's entity ends on — a residue-bearing
// number, on carry_test.go's own reasoning: 700 is level 5, which accounts for
// only 610, so a front end that carried the level instead of the integer would
// be visible here too.
var continuityXP = [data.SkillSlots]int32{700, 0, 0, 0, 0, 0}

// continuityFront is a front end holding the fixture campaign and nothing else
// — no archives, no table, no assets. Everything under test reads the campaign
// and the arguments it is given, so an install would add nothing but a
// dependency on having one.
//
// ITS MISSION 10 DECLARES SUCCESSOR 20 UNDER AutoGetMission (scenarioFixture,
// campaign_test.go). That is why the cases here that must NOT advance — a
// dialogue paging on, a lost mission, a mission still running — are driven
// over mission 10 too: the wrapper's own gate is the destination and the
// outcome, checked BEFORE it ever asks the campaign anything, so a fixture
// whose declared successor sits right there is the sharper test of it.
func continuityFront(t *testing.T) *FrontEnd {
	t.Helper()
	r, err := reg.Parse(scenarioFixture())
	if err != nil {
		t.Fatalf("parse the scenario fixture: %v", err)
	}
	c := ReadCampaign(r)
	// The town is built over the campaign here for the reason NewFrontEnd
	// builds one there: a front end holding a campaign and no town for it is a
	// state this package never produces.
	return &FrontEnd{InstallResources: InstallResources{Campaign: resolved(c, nil)}, CampaignSession: CampaignSession{Town: NewTown(c)}}
}

// continuityMission is a mission whose world has been driven to the outcome
// asked for, with one party member standing in it.
//
// THE WIN IS THE SCRIPT'S OWN. A world's outcome latches inside pkg/sim and
// nothing above it can write one, which is the point: what this file asserts
// about a won mission is asserted over a mission that really was won.
func continuityMission(t *testing.T, n int, op int32) *Mission {
	t.Helper()

	s, err := sim.NewScript(missionChecks(),
		[]sim.ScriptInstant{{Op: op}},
		[]sim.ScriptTrigger{missionTrigger(3, 1, true, 0)})
	if err != nil {
		t.Fatalf("NewScript: %v", err)
	}
	ents := []sim.Entity{{
		ID: 0, X: 10, Y: 10, HP: 20, MaxHP: 20,
		Owner: sim.SelfSlot, Domain: sim.DomainGround,
		SkillXP: continuityXP, GainsXP: true, TypeID: sim.HumanTypeID,
	}}
	w, err := sim.NewScriptedWorld(1, sim.Bounds{Width: worldFixtureW, Height: worldFixtureH},
		sim.ModeCanonical, make([]byte, worldFixtureW*worldFixtureH), ents, s)
	if err != nil {
		t.Fatalf("NewScriptedWorld: %v", err)
	}
	for i := 0; i < 64 && w.Outcome() == sim.OutcomeUndecided; i++ {
		sim.Step(w, nil)
	}
	return &Mission{
		Number: n, World: w,
		Party: []mapload.PartyMember{{Class: 100}},
		Start: mapload.Start{IDs: []sim.EntityID{0}},
	}
}

// listAdvance stands in for the driver's own answer on the win path. The
// driver decides that answer and world_test.go decides the driver; what is
// under test here is only what the wrapper does with it. It answers a nil
// opener, exactly as world.go's own advanceNotice does: nothing above
// pkg/sim can recognise a campaign, so the driver never has one of its own to
// offer — the opener that reaches a successor is always continuity's own,
// built after FinishMission returns, never the driver's.
func listAdvance(_ ...ui.NoticeAction) (ui.NoticeDest, string, ui.MapOpener) {
	return ui.NoticeToMapList, TownNotBuiltMessage, nil
}

// missionArchive is a scenario container holding the given mission numbers'
// maps, plus the npc.reg every mission start reads for. A number named in a
// campaign but NOT passed here is what AC-8's fixtures use to make a
// declared successor's map genuinely absent from the archive, rather than
// merely untried.
func missionArchive(t *testing.T, maps ...int) *vfs.FS {
	t.Helper()
	files := []synth.File{{Path: "npc.reg", Data: synth.NPCReg(nil)}}
	for _, n := range maps {
		files = append(files, synth.File{
			Path: strconv.Itoa(n) + ".alm",
			Data: synth.ALM(synth.ALMOptions{Width: 24, Height: 24}),
		})
	}
	dir := t.TempDir()
	path := filepath.Join(dir, ScenarioArchive)
	if err := os.WriteFile(path, synth.Archive(files), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
	containers, err := vfs.Open([]string{path}, nil)
	if err != nil {
		t.Fatalf("vfs.Open: %v", err)
	}
	return containers
}

// advanceFront is a front end that can actually OPEN what it advances to: the
// fixture campaign (scenarioFixture), whose mission 10 declares successor 20,
// over an archive holding both maps. Everything else a FrontEnd carries — the
// table, the two art bundles, the font — is left nil, on loadMap's own
// tolerance of a hand-assembled front end (frontend.go's own field docs):
// none of them gate whether a mission opens, only what it draws.
func advanceFront(t *testing.T) *FrontEnd {
	t.Helper()
	r, err := reg.Parse(scenarioFixture())
	if err != nil {
		t.Fatalf("parse the scenario fixture: %v", err)
	}
	return &FrontEnd{InstallResources: InstallResources{Campaign: resolved(ReadCampaign(r), nil), Archives: &Archives{Containers: missionArchive(t, 10, 20)}, Tiles: &terrain.Tileset{}}}
}

func miniCampaign(t *testing.T, sections ...synth.RegNode) Campaign {
	t.Helper()
	r, err := reg.Parse(synth.Reg(kindRoot, sections))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	return ReadCampaign(r)
}

func declaring(n int, v int32) synth.RegNode {
	return synth.RegNode{Name: "Mission" + strconv.Itoa(n), Kind: kindDir,
		Children: []synth.RegNode{{Name: "AutoGetMission", Kind: kindInt, Int: v}}}
}

// Dismissing a won mission whose declared successor CAN be opened opens it
// directly: the wrapper's own destination moves to NoticeToMission, the map
// list is never named, and the opener it hands back genuinely starts that
// mission (AC-4, SC-3).
func TestDismissingAWonMissionOpensItsDeclaredSuccessor(t *testing.T) {
	f := advanceFront(t)
	ms := continuityMission(t, 10, sim.ScriptInstantWin)
	if ms.World.Outcome() != sim.OutcomeWon {
		t.Fatalf("setup: outcome %v, want won", ms.World.Outcome())
	}

	dest, msg, open := f.continuity(10, ms, listAdvance)()

	if dest != ui.NoticeToMission {
		t.Errorf("destination %v, want NoticeToMission — the fixture's mission 10 declares successor 20", dest)
	}
	want := "mission 10 won - your party carries over; mission 20 follows and opens now"
	if msg != want {
		t.Errorf("message %q,\n           want %q", msg, want)
	}
	if len(f.Carried) != 1 || f.Carried[0].Carry == nil {
		t.Fatalf("Carried = %+v, want one member holding a carry", f.Carried)
	}
	if got := f.Carried[0].Carry.SkillXP; got != continuityXP {
		t.Errorf("carried experience %v, want %v — the integer the entity ended on", got, continuityXP)
	}
	if f.Offered != 0 {
		t.Errorf("Offered = %d after a direct advance, want 0 — it opened, nothing was offered", f.Offered)
	}

	if open == nil {
		t.Fatal("no opener came back; FR-3 requires one for a successor this tree can address")
	}
	v, tick, _, _, _, _, _, _, _, _, err := open()
	if err != nil {
		t.Fatalf("the opener FR-3 promises failed: %v", err)
	}
	if v == nil || tick == nil {
		t.Fatal("the opener returned an incomplete map screen")
	}
}

// AC-5: the party opening the successor is not just a party — it is the
// hero the won mission ended with, at the exact experience CarryParty
// captured. This drives the SAME StartMission MissionOpenerWith calls
// internally, over f.Carried the win itself produced, so what is asserted is
// the number that would in fact reach the successor's own entity and not a
// value re-derived to match it.
func TestTheSuccessorsPartyIsTheOneThatFinished(t *testing.T) {
	f := advanceFront(t)
	ms := continuityMission(t, 10, sim.ScriptInstantWin)

	successor, _ := f.FinishMission(10, ms.Party, ms.World, ms.Start.IDs)
	if successor != 20 {
		t.Fatalf("setup: successor %d, want 20", successor)
	}

	next, err := StartMission(f.Archives.Containers, successor, f.Table, openDifficulty, f.Carried)
	if err != nil {
		t.Fatalf("StartMission(%d): %v", successor, err)
	}
	hero, ok := heroEntity(next)
	if !ok {
		t.Fatal("the successor started with no hero entity")
	}
	if hero.SkillXP != continuityXP {
		t.Errorf("the successor's hero holds skill xp %v, want %v — the carried party's own", hero.SkillXP, continuityXP)
	}
}

// companionTable is a definition table whose npc 22 grant is a female mage
// with her own Humans row, armour and weapon.
func companionTable(t *testing.T) *mapload.Table {
	t.Helper()
	humans := make(dbCollection, 30)
	humans[28] = dbEntry{name: "PC_Fergard", params: humansParams(30, 70, 3)}
	humans[29] = dbEntry{name: "PC_Reniesta", params: humansParams(30, 70, 1),
		strings: []string{"Wood Staff", "", "Uncommon Dress", "Uncommon Cloak", "", "", "", "", "", ""}}
	// The campaign selector resolves composed npc22 by the Humans row's server
	// id, exactly as the installed collection does.
	humans[28].params[0x18] = 28
	humans[29].params[0x18] = 29
	humans[28].params[0x10], humans[29].params[0x10] = 24, 24
	humans[29].params[0x12] = 1
	npcReg, err := reg.Parse(synth.Reg(0x11, []synth.RegNode{{Name: "npc22", Kind: 0x01,
		Children: []synth.RegNode{
			{Name: "Flags", Kind: 0x00, Str: "Hero,Mage,!MySex,Start"},
			{Name: "DataBinID", Kind: 0x02, Int: 26},
		}}}))
	if err != nil {
		t.Fatalf("parse npc registry: %v", err)
	}
	armorParams := func(slot int32) []int32 {
		p := make([]int32, 11)
		for i := range p {
			p[i] = -1
		}
		p[4] = slot
		return p
	}
	armors := dbCollection{
		{},
		{name: "Uncommon Dress", params: armorParams(7)},
		{name: "Uncommon Cloak", params: armorParams(8)},
	}
	weapons := dbCollection{
		{},
		{name: "Wood Staff", params: chargenWeaponParams(data.SkillBlade)},
	}
	return &mapload.Table{Humans: humans, NPC: data.LoadNPCDefs(npcReg), Weapons: weapons, Armors: armors, Shapes: emptyScale{}, Materials: emptyScale{}}
}

func TestMission30AddHeroCreatesASeparatePersistentCompanion(t *testing.T) {
	table := companionTable(t)
	f := &FrontEnd{InstallResources: InstallResources{Campaign: resolved(Campaign{Chapters: map[int]Chapter{30: {Mission: 30, AddHero: []int{22}}}}, nil), Table: table}, CampaignSession: CampaignSession{Carried: []mapload.PartyMember{{Name: "Danath", PlayerCharacter: true, StartingHero: true, FigureDir: string(data.FigureDirManFighter)}}}}

	f.addChapterCompanions(30)
	f.addChapterCompanions(30)

	if len(f.Carried) != 2 {
		t.Fatalf("party length = %d, want one primary hero and one companion", len(f.Carried))
	}
	got := f.Carried[1]
	if got.Name != "Reniesta" || !got.PlayerCharacter || got.StartingHero || got.MercenaryType != 0 || got.CompanionNPC != 22 {
		t.Fatalf("companion = %+v, want the separate non-mercenary npc22 hero Reniesta", got)
	}
	if !got.Mage || got.FigureDir != string(data.FigureDirWomanMage) {
		t.Fatalf("companion class/figure = mage %v, %q; want female mage", got.Mage, got.FigureDir)
	}
	dress, err := data.ResolveArmor("Uncommon Dress", f.Table.Shapes, f.Table.Materials, f.Table.Armors)
	if err != nil {
		t.Fatalf("ResolveArmor(Uncommon Dress): %v", err)
	}
	cloak, err := data.ResolveArmor("Uncommon Cloak", f.Table.Shapes, f.Table.Materials, f.Table.Armors)
	if err != nil {
		t.Fatalf("ResolveArmor(Uncommon Cloak): %v", err)
	}
	if got.Worn[6] != uint16(dress.Code) || got.Worn[7] != uint16(cloak.Code) {
		t.Fatalf("Reniesta worn slots 7/8 = %#04x/%#04x, want shipped dress/cloak %#04x/%#04x",
			got.Worn[6], got.Worn[7], uint16(dress.Code), uint16(cloak.Code))
	}
	if got.Weapon == nil || got.Worn[0] != uint16(got.Weapon.Code) || got.Weapon.Name != "Wood Staff" {
		t.Fatalf("Reniesta staff = %+v, worn slot 1 %#04x; want her Humans-row Wood Staff in hand",
			got.Weapon, got.Worn[0])
	}
}

func TestAFreshMissionStartsWithTheTownPurse(t *testing.T) {
	f := advanceFront(t)
	f.Town = NewTown(f.Campaign.Value())
	f.Town.gold = 511

	if _, _, _, _, _, _, _, _, _, _, err := f.MissionOpener(10)(); err != nil {
		t.Fatalf("MissionOpener: %v", err)
	}
	if f.live == nil || f.live.world == nil {
		t.Fatal("MissionOpener installed no live world")
	}
	if got := f.live.world.Purse(sim.SelfSlot); got != 511 {
		t.Fatalf("fresh mission purse = %d, want town gold 511", got)
	}
}

func TestAnOriginalMissionStartsWithItsPlayersSavedPurse(t *testing.T) {
	f := advanceFront(t)
	f.Town = NewTown(f.Campaign.Value())
	f.Town.gold = 11
	saved := uint32(700)

	open := f.missionOpener(10, f.NextParty(), nil, &saved, nil)
	if _, _, _, _, _, _, _, _, _, _, err := open(); err != nil {
		t.Fatalf("original mission opener: %v", err)
	}
	if f.live == nil || f.live.world == nil {
		t.Fatal("original mission opener installed no live world")
	}
	if got := f.live.world.Purse(sim.SelfSlot); got != saved {
		t.Fatalf("original mission purse = %d, want Player record's %d", got, saved)
	}
}

func TestFinishingAMissionCarriesItsPurseBackToTown(t *testing.T) {
	f := advanceFront(t)
	f.Town = NewTown(f.Campaign.Value())
	ms := continuityMission(t, 10, sim.ScriptInstantWin)
	if !ms.World.SetPurse(sim.SelfSlot, 733) {
		t.Fatal("SetPurse refused SelfSlot")
	}

	f.FinishMission(10, ms.Party, ms.World, ms.Start.IDs)

	if got := f.Town.Gold(); got != 733 {
		t.Fatalf("town gold after FinishMission = %d, want mission purse 733", got)
	}
}

func TestTheSuccessorDoesNotInheritTheSavesCell(t *testing.T) {
	f := advanceFront(t)
	ms := continuityMission(t, 10, sim.ScriptInstantWin)
	// The party as a resume builds it: every member carries a Saved.
	saved := mapload.Cell{X: 35, Y: 29}
	ms.Party[0].Saved = &mapload.Saved{
		Cell: saved, HP: 7, MaxHP: 131, Mana: 0, MaxMana: 0, MapUnitID: 21,
	}

	successor, _ := f.FinishMission(10, ms.Party, ms.World, ms.Start.IDs)
	if successor != 20 {
		t.Fatalf("setup: successor %d, want 20", successor)
	}
	for i, p := range f.Carried {
		if p.Saved != nil {
			t.Fatalf("carried member %d still holds a Saved: %+v", i, *p.Saved)
		}
	}

	next, err := StartMission(f.Archives.Containers, successor, f.Table, openDifficulty, f.Carried)
	if err != nil {
		t.Fatalf("StartMission(%d): %v", successor, err)
	}
	if next.Start.Cells[0] == saved {
		t.Errorf("the successor placed the hero at the previous mission's cell %v", saved)
	}
	if next.Start.Cells[0] != next.Start.Drop {
		t.Errorf("the hero stands at %v, want the successor map's own drop cell %v",
			next.Start.Cells[0], next.Start.Drop)
	}
	// His health is the successor's fold and not the seven the save recorded.
	hero, ok := heroEntity(next)
	if !ok {
		t.Fatal("the successor started with no hero entity")
	}
	if hero.HP == 7 || hero.HP != hero.MaxHP {
		t.Errorf("the hero opens the successor at %d/%d hp, want a full fold",
			hero.HP, hero.MaxHP)
	}
}

func TestARefusedSuccessorNamesBothMissionsAndCarriesTheParty(t *testing.T) {
	for _, declared := range []int32{0, -2} {
		t.Run(fmt.Sprintf("declared %d", declared), func(t *testing.T) {
			f := &FrontEnd{InstallResources: InstallResources{Campaign: resolved(miniCampaign(t, declaring(10, declared)), nil)}}
			ms := continuityMission(t, 10, sim.ScriptInstantWin)

			dest, msg, open := f.continuity(10, ms, listAdvance)()

			if dest != ui.NoticeToMapList {
				t.Errorf("destination %v, want the map list — a non-positive successor must not open", dest)
			}
			if open != nil {
				t.Errorf("an opener came back for a refused successor; FR-7 says nothing is read")
			}
			want := fmt.Sprintf("mission 10 won - your party carries over; "+
				"the campaign declares mission %d follows and this tree can open no such mission: choose a row",
				declared)
			if msg != want {
				t.Errorf("message %q,\n           want %q", msg, want)
			}
			if len(f.Carried) != 1 {
				t.Errorf("Carried = %+v, want the party even though the successor was refused", f.Carried)
			}
			if f.Offered != 0 {
				t.Errorf("Offered = %d, want 0 — nothing was offered, it was refused", f.Offered)
			}
		})
	}
}

func TestAnUnopenableDeclaredSuccessorFailsAtTheOpener(t *testing.T) {
	f := &FrontEnd{InstallResources: InstallResources{Campaign: resolved(miniCampaign(t, declaring(10, 99)), nil), Archives: &Archives{Containers: missionArchive(t, 10)}, Tiles: &terrain.Tileset{}}}
	ms := continuityMission(t, 10, sim.ScriptInstantWin)

	dest, msg, open := f.continuity(10, ms, listAdvance)()

	if dest != ui.NoticeToMission {
		t.Fatalf("destination %v, want NoticeToMission — 99 is a mission this tree can ADDRESS; refusal is FR-7's alone", dest)
	}
	if open == nil {
		t.Fatal("no opener at all for a mission this tree can address")
	}
	_, _, _, _, _, _, _, _, _, _, err := open()
	if err == nil {
		t.Fatal("the opener succeeded on a mission absent from the archive")
	}
	if err.Error() == msg {
		t.Errorf("the opener's own failure %q repeats the seam's sentence %q — FR-8 needs them told apart", err, msg)
	}
	if len(f.Carried) != 1 {
		t.Errorf("Carried = %+v, want the party even though the successor's map will not open", f.Carried)
	}
}

// Dismissing a won mission at the town's boundary OPENS THE TOWN and goes
// there. Mission 20 declares no successor of its own, so this stays on
// NextMission's ascending order exactly as before 0142 — what changed is
// the destination, not how the boundary is found.
//
// The name no longer spells a sentence. It spelled "the number stands in for
// the gate" while the town was not built, and 0142 built it; a test named for
// a placeholder outlives the placeholder.
func TestAWinAtTheTownsBoundaryOpensTheTown(t *testing.T) {
	f := continuityFront(t)
	ms := continuityMission(t, 20, sim.ScriptInstantWin)

	dest, msg, open := f.continuity(20, ms, listAdvance)()

	if dest != ui.NoticeToTown {
		t.Errorf("destination %v, want NoticeToTown", dest)
	}
	if open != nil {
		t.Error("an opener crossed with NoticeToTown — the town is a screen the front end already holds")
	}
	want := "mission 20 won - your party carries over; the town takes it from here"
	if msg != want {
		t.Errorf("message %q,\n           want %q", msg, want)
	}
	if !f.Town.Open() {
		t.Error("the town did not open at the campaign's own town boundary")
	}
	if got := f.Town.Chapter(); got != 30 {
		t.Errorf("chapter %d, want 30 — the lowest main mission a building offers and that is not won", got)
	}
	if f.Offered != 30 {
		t.Errorf("Offered = %d, want 30", f.Offered)
	}
}

// A LOST mission ends in the town too, once the town is open — and the
// party is not carried and the mission is not marked done.
//
// It is the same front end and the same mission twice: the first call opens
// the town by winning 20, the second loses 30 into it.
func TestALostMissionExitsToMenuEvenOnceTownIsOpen(t *testing.T) {
	f := continuityFront(t)
	f.continuity(20, continuityMission(t, 20, sim.ScriptInstantWin), listAdvance)()
	if !f.Town.Open() {
		t.Fatal("the town did not open")
	}
	carried := len(f.Carried)

	lost := continuityMission(t, 30, sim.ScriptInstantLose)
	dest, msg, open := f.continuity(30, lost, menuAdvance)()

	if dest != ui.NoticeToMenu {
		t.Errorf("destination %v, want NoticeToMenu even with an open town", dest)
	}
	if open != nil {
		t.Error("an opener crossed with a lost mission")
	}
	if msg != "" {
		t.Errorf("terminal exit carried a return-to-town message: %q", msg)
	}
	if f.Town.Done(30) {
		t.Error("a LOST mission was marked done")
	}
	if len(f.Carried) != carried {
		t.Errorf("Carried moved on a loss: %d, was %d", len(f.Carried), carried)
	}
}

// A defeat with NO town open still goes to the main menu, exactly as it did
// before 0142.
func TestALostMissionStillReachesTheMenuBeforeTheTownIsOpen(t *testing.T) {
	f := continuityFront(t)
	ms := continuityMission(t, 10, sim.ScriptInstantLose)

	dest, _, _ := f.continuity(10, ms, menuAdvance)()

	if dest != ui.NoticeToMenu {
		t.Errorf("destination %v, want NoticeToMenu — the town has not been reached", dest)
	}
}

// menuAdvance is the driver's own answer for a LOST mission: the main menu,
// no sentence, no opener. It stands beside listAdvance for the same reason —
// continuity wraps the driver and the driver is not what is under test here.
func menuAdvance(_ ...ui.NoticeAction) (ui.NoticeDest, string, ui.MapOpener) {
	return ui.NoticeToMenu, "", nil
}

// The two ends of the ladder, stated directly on the method so the sentences
// are decided somewhere a reader can find them all together (AC-12).
func TestTheSentencesAtTheEndsOfTheLadder(t *testing.T) {
	party := []mapload.PartyMember{{Class: 100}}

	last := continuityFront(t)
	successor, got := last.FinishMission(40, party, nil, nil)
	want := "mission 40 won - your party carries over; it is the last mission the campaign declares"
	if got != want {
		t.Errorf("last mission: %q,\n         want %q", got, want)
	}
	if successor != 0 {
		t.Errorf("successor = %d for the last mission, want 0 — DD-3's zero means none", successor)
	}
	if last.Offered != 0 {
		t.Errorf("Offered = %d after the last mission, want 0", last.Offered)
	}

	// An install whose scenario registry would not read still carries the
	// party; it merely cannot name what follows.
	blind := &FrontEnd{}
	successor, got = blind.FinishMission(10, party, nil, nil)
	want = "mission 10 won - your party carries over; this install declares no campaign, " +
		"so nothing names what follows"
	if got != want {
		t.Errorf("no campaign: %q,\n        want %q", got, want)
	}
	if successor != 0 {
		t.Errorf("successor = %d with no campaign, want 0", successor)
	}
	if len(blind.Carried) != 1 {
		t.Fatalf("Carried = %+v, want the party even with no campaign", blind.Carried)
	}
}

// The three answers the wrapper must leave exactly alone. A wrapper that acted
// on the destination alone would carry a party out of any future screen that
// happened to return to the map list, and one that acted on any dismissal at
// all would carry it out of a paged dialogue.
//
// EVERY CASE IS DRIVEN OVER MISSION 10, WHOSE FIXTURE DECLARES A SUCCESSOR
// (continuityFront's own note): the gate under test is the destination and
// the outcome, asked before the campaign is ever consulted, so a mission
// that WOULD advance if either gate were skipped is the sharper witness that
// neither is.
func TestTheWrapperTouchesNothingButAWinReachingTheMapList(t *testing.T) {
	cases := []struct {
		name string
		op   int32
		adv  ui.MapAdvance
		dest ui.NoticeDest
		msg  string
	}{
		{"a dialogue paging on", sim.ScriptInstantWin,
			func(...ui.NoticeAction) (ui.NoticeDest, string, ui.MapOpener) { return ui.NoticeStay, "", nil }, ui.NoticeStay, ""},
		{"a lost mission leaving for the menu", sim.ScriptInstantLose,
			func(...ui.NoticeAction) (ui.NoticeDest, string, ui.MapOpener) { return ui.NoticeToMenu, "", nil }, ui.NoticeToMenu, ""},
		{"a map list reached by a mission still running", sim.ScriptNone,
			listAdvance, ui.NoticeToMapList, TownNotBuiltMessage},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := continuityFront(t)
			ms := continuityMission(t, 10, tc.op)

			dest, msg, _ := f.continuity(10, ms, tc.adv)()

			if dest != tc.dest || msg != tc.msg {
				t.Errorf("advance = %v/%q, want %v/%q", dest, msg, tc.dest, tc.msg)
			}
			if f.Carried != nil || f.Offered != 0 {
				t.Errorf("the front end remembered %+v / mission %d; it should have remembered nothing",
					f.Carried, f.Offered)
			}
		})
	}
}

// SC-5: every path off a win carries the party — a successor that opens, one
// that names none, one that is refused, one whose declared map will not
// open, and an install with no campaign at all.
func TestEveryEndingCarriesTheParty(t *testing.T) {
	cases := []struct {
		name string
		f    *FrontEnd
		n    int
	}{
		{"a successor that opens", advanceFront(t), 10},
		{"no successor declared", continuityFront(t), 20},
		{"a refused non-positive successor", &FrontEnd{InstallResources: InstallResources{Campaign: resolved(miniCampaign(t, declaring(10, 0)), nil)}}, 10},
		{"a declared successor whose map will not open", &FrontEnd{InstallResources: InstallResources{Campaign: resolved(miniCampaign(t, declaring(10, 99)), nil), Archives: &Archives{Containers: missionArchive(t, 10)}, Tiles: &terrain.Tileset{}}}, 10},
		{"no campaign at all", &FrontEnd{}, 10},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ms := continuityMission(t, tc.n, sim.ScriptInstantWin)
			_, _, open := tc.f.continuity(tc.n, ms, listAdvance)()
			if open != nil {
				// May fail — only the carry is under test on this arm; the
				// opener's own success or failure is each covered on its own
				// dedicated test above.
				open()
			}
			if len(tc.f.Carried) != 1 || tc.f.Carried[0].Carry == nil {
				t.Errorf("Carried = %+v, want one member holding a carry", tc.f.Carried)
			}
		})
	}
}

// Before any mission is won, the party a mission opens with is exactly the one
// this front end always opened with.
func TestBeforeTheFirstWinTheDefaultPartyIsUnchanged(t *testing.T) {
	f := continuityFront(t)
	got := f.NextParty()
	want := MissionParty(f.StartWeapon.Value(), f.Bodies, f.Table)

	if len(got) != len(want) {
		t.Fatalf("NextParty gave %d member(s), MissionParty gives %d", len(got), len(want))
	}
	for i := range got {
		if got[i].Carry != nil {
			t.Errorf("member %d arrives holding a carry before any mission was won", i)
		}
		if got[i].Class != want[i].Class || got[i].Hero != want[i].Hero {
			t.Errorf("member %d = %+v, want %+v", i, got[i], want[i])
		}
	}
}

// A message the MAP LIST is given must name the mission it is pointing at —
// the whole of "offering" the next mission, at this tier, is that the number
// reaches the player.
//
// MISSION 10's OWN RESULT IS NO LONGER ONE OF THESE: the fixture campaign
// declares its own successor under AutoGetMission (scenarioFixture), so
// mission 10 now advances directly and is exercised by
// TestDismissingAWonMissionOpensItsDeclaredSuccessor instead. 15 stands in
// for it here — no section names it, so NextMission's own ascending order is
// all that can answer, and it answers exactly what 10 did before this story:
// mission 20, not town-offered.
//
// A TOWN OFFER IS NO LONGER ONE OF THESE EITHER. While the town was not
// built, its number was a placeholder the map list had to state; the town
// now decides which of its own missions is walked out to, and the number on
// the way there would be a second answer to a question one screen already
// owns. What the town-offered arm must do instead is OPEN THE TOWN, which is
// the case below this one.
func TestTheMapListSentenceNamesTheMissionItOffers(t *testing.T) {
	f := continuityFront(t)
	party := []mapload.PartyMember{{Class: 100}}

	for _, won := range []int{15} {
		offer, ok := f.Campaign.Value().NextMission(won)
		if !ok {
			t.Fatalf("the fixture names no successor to mission %d", won)
		}
		if offer.Town {
			t.Fatalf("mission %d's successor is town-offered; this case is about the map list", won)
		}
		_, msg := f.FinishMission(won, party, nil, nil)
		if !strings.Contains(msg, "mission "+strconv.Itoa(offer.Mission)) {
			t.Errorf("after mission %d the map list is told %q, which does not name mission %d",
				won, msg, offer.Mission)
		}
	}
}

// Every town-offered win opens the town, and the chapter it stands in is the
// lowest main mission a building offers that is not yet won.
//
// THE SIDE MISSION IS THE DISCRIMINATING CASE. Winning 31 must NOT move the
// chapter off 30: a side mission is something a building offered beside a step
// of the campaign, not a step of it, and a stored chapter advanced on every
// win would silently skip mission 30 for a player who took the side job first.
func TestATownOfferOpensTheTownAndTheChapterIsDerived(t *testing.T) {
	f := continuityFront(t)
	party := []mapload.PartyMember{{Class: 100}}

	for _, won := range []int{20, 31} {
		offer, ok := f.Campaign.Value().NextMission(won)
		if !ok || !offer.Town {
			t.Fatalf("mission %d's successor is not town-offered in the fixture", won)
		}
		if _, msg := f.FinishMission(won, party, nil, nil); msg == "" {
			t.Errorf("winning mission %d said nothing", won)
		}
		if !f.Town.Open() {
			t.Fatalf("the town is not open after winning mission %d", won)
		}
		if got := f.Town.Chapter(); got != 30 {
			t.Errorf("after mission %d the chapter is %d, want 30", won, got)
		}
	}

	// And winning the chapter's own MAIN mission does move it.
	f.FinishMission(30, party, nil, nil)
	if got := f.Town.Chapter(); got == 30 {
		t.Error("the chapter stayed at 30 after mission 30 was won")
	}
}

func TestAnAdvanceClearsWhatTheMapListWasLastTold(t *testing.T) {
	f := advanceFront(t)
	// Mission 30's offer is town-bound; this helper otherwise leaves Town unset.
	f.Town = NewTown(f.Campaign.Value())

	// Mission 30 declares no successor in the fixture, so this win goes
	// through NextMission and leaves its answer in Offered.
	if next, _ := f.FinishMission(30, nil, nil, nil); next != 0 {
		t.Fatalf("setup: mission 30 advanced to %d; this case needs a win that reaches the map list", next)
	}
	if f.Offered == 0 {
		t.Fatal("setup: the ladder wrote nothing to Offered, so clearing it cannot be measured")
	}
	stale := f.Offered

	if next, _ := f.FinishMission(10, nil, nil, nil); next != 20 {
		t.Fatalf("mission 10 answered %d, want 20 — the fixture declares it", next)
	}
	if f.Offered != 0 {
		t.Errorf("Offered = %d after an advance, want 0 — it still holds %d, the row the PREVIOUS win offered", f.Offered, stale)
	}
}
