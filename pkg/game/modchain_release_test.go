package game

import (
	"bytes"
	"os"
	"slices"
	"strings"
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/mapload"
	"againrom/pkg/mod"
	"againrom/pkg/modrt"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

const (
	chainModID    = "guest-chain"
	chainSecondNo = 26
)

// chainFront is a front end on the install under the mods ids of the folder
// dir, every kind of their data applied through the game's one apply step.
func chainFront(t *testing.T, dir string, ids ...string) *FrontEnd {
	t.Helper()
	f := releaseFront(t)
	f.SetDeterministicFrames(true)
	entries, err := mod.Resolve(dir, ids)
	if err != nil {
		t.Fatal(err)
	}
	res, err := modrt.Load(entries, BaseID(InspectInstall(os.Getenv("AGAINROM_ASSETS"))), nil, modrt.Options{})
	if err != nil {
		t.Fatal(err)
	}
	if err := f.SetMods(res.Rules, res.Set, false); err != nil {
		t.Fatal(err)
	}
	if _, err := f.SetModData(res.Data); err != nil {
		t.Fatal(err)
	}
	return f
}

// chainSecond plays the chain's first link through guestPlay, wins it through
// the map driver's continuation and opens the second link. It answers the
// guest's experience at the end of the first link.
func chainSecond(t *testing.T, f *FrontEnd, app *ui.App) [6]int32 {
	t.Helper()
	gained := guestPlay(t, f, app)
	stash := f.Town.guest.stash
	if dest, line := guestWin(t, f); dest != ui.NoticeToMission || !strings.Contains(line, "goes on to mission 26") {
		t.Fatalf("the first link's win led to %v: %q", dest, line)
	}
	g := f.Town.guest
	if g == nil || !g.playing(chainSecondNo) || g.member == nil || g.member.Carry == nil || g.member.Carry.SkillXP != gained {
		t.Fatalf("after the first link the guest record is %+v", g)
	}
	if !bytes.Equal(g.stash, stash) || slices.ContainsFunc(f.Carried, func(p mapload.PartyMember) bool { return p.CompanionNPC == 0 }) {
		t.Fatalf("the waiting party came back between the links: %v", modJoinRoster(f))
	}
	if err := app.OpenMission(f.MissionOpener(chainSecondNo)); err != nil {
		t.Fatalf("open the second link: %v", err)
	}
	return gained
}

// humansRow is the index of the definition row named name.
func humansRow(t *testing.T, f *FrontEnd, name string) int {
	t.Helper()
	for i := 1; i < f.Table.Humans.Len(); i++ {
		if f.Table.Humans.EntryName(i) == name {
			return i
		}
	}
	t.Fatalf("no definition row %q", name)
	return 0
}

// rowHero is the statistics the definition row of that name states.
func rowHero(t *testing.T, f *FrontEnd, name string) data.Hero {
	t.Helper()
	d, err := data.NewHumanDef(name, f.Table.Humans.EntryParams(humansRow(t, f, name)))
	if err != nil {
		t.Fatal(err)
	}
	return d.Hero()
}

// TestReleaseModMissionChainPlaysAndReturns proves, on the lawful install, a
// chain of two guest missions: mission 20's win opens mission 25 with
// companion 22 alone; her win opens mission 26 with no town between, where
// she plays on as she ended mission 25 beside the mod's sellsword at level 2
// with the sword and mail the mission sets, with the declared purse. The
// second win brings back the waiting party exactly as the town would have
// received it without the mod, in the town the first win opens; she joins
// it, and the sellsword stays behind. A SAVE and cold LOAD inside mission 26
// continues the chain to the same town, and a LOAD without the mod refuses.
func TestReleaseModMissionChainPlaysAndReturns(t *testing.T) {
	dir := assembledMods(t, chainModID, chainModID)

	plain := guestFront(t, dir)
	if _, next := guestRoute(t, plain, []int{0, 0, 0}); next != 0 || plain.Town.Chapter() != 30 {
		t.Fatalf("unmodded win of mission 20 answered %d in chapter %d", next, plain.Town.Chapter())
	}
	want := guestDigest(t, plain)

	f := chainFront(t, dir, chainModID)
	app, next := guestRoute(t, f, []int{0, 0, 0})
	if next != guestMissionNo || f.campaign().route(f, 20, next) != winMission {
		t.Fatalf("mission 20's win answered %d", next)
	}
	gained := chainSecond(t, f, app)
	party := f.LiveParty()
	if len(party) != 2 || party[0].CompanionNPC != guestCompanion || !party[0].StartingHero ||
		party[0].Carry == nil || party[0].Carry.SkillXP != gained || party[1].ID != "char:sellsword" || party[1].StartingHero {
		t.Fatalf("the second link opened with %+v", party)
	}
	if got := f.live.world.Purse(sim.SelfSlot); got != 100 || f.live.mission.state.Address != "scenario/26.alm" {
		t.Fatalf("the second link opened %q with purse %d", f.live.mission.state.Address, got)
	}
	sword := party[1]
	name, _ := encodeSaveLabel(map[string]string{"en": "Sellsword", "ru": "Наёмница"}[modrt.LanguageFor(BaseID(InspectInstall(os.Getenv("AGAINROM_ASSETS"))))], f.textSelector())
	if sword.Name != name {
		t.Fatalf("the sellsword is named %q, want %q", sword.Name, name)
	}
	if sword.Hero != rowHero(t, f, "NPC06_2") || rowHero(t, f, "NPC06_2") == rowHero(t, f, "NPC06_1") {
		t.Fatalf("the sellsword at level 2 has %+v, row NPC06_2 states %+v", sword.Hero, rowHero(t, f, "NPC06_2"))
	}
	own := f.Table.Humans.EntryStrings(humansRow(t, f, "NPC06_2"))
	cells := make([]string, data.HumanCellCount)
	copy(cells, own)
	cells[data.HumanCellWeapon], cells[data.HumanCellArmour] = "Iron Short Sword", "Hard Leather Mail"
	for i := data.HumanCellArmour + 1; i < data.HumanCellCount; i++ {
		cells[i] = ""
	}
	set, _, _, err := mapload.HumanRowEquipment(cells, f.Table)
	if err != nil {
		t.Fatal(err)
	}
	shipped, _, _, err := mapload.HumanRowEquipment(own, f.Table)
	if err != nil {
		t.Fatal(err)
	}
	for i := range set {
		if sword.Worn[i] != set[i].Code {
			t.Fatalf("the sellsword wears %v, the mission sets %v", sword.Worn, set)
		}
	}
	if slices.EqualFunc(set[:], shipped[:], func(a, b sim.ItemInstance) bool { return a.Code == b.Code }) {
		t.Fatal("the set equipment is the row's own")
	}

	if dest, line := guestWin(t, f); dest != ui.NoticeToTown {
		t.Fatalf("the second link's win led to %v: %q", dest, line)
	}
	joined, ok := guestMember(f)
	if !ok || f.Town.guest != nil || f.Town.Chapter() != 30 || !f.Town.Open() {
		t.Fatalf("after the chain: roster %v chapter %d", modJoinRoster(f), f.Town.Chapter())
	}
	if slices.ContainsFunc(f.Carried, func(p mapload.PartyMember) bool { return p.ID == "char:sellsword" }) {
		t.Fatal("the sellsword came home with the party")
	}
	if got := guestDigest(t, f); got != want {
		t.Fatalf("the town after the chain differs from the town without the mod:\n got %s\nwant %s", got, want)
	}
	if joined.Carry == nil || joined.Carry.SkillXP != gained {
		t.Fatalf("the guest joined with %v, she ended the chain with %v", joined.Carry, gained)
	}

	// The same chain with a SAVE and a cold LOAD inside mission 26.
	g := chainFront(t, dir, chainModID)
	gapp, _ := guestRoute(t, g, []int{0, 0, 0})
	chainSecond(t, g, gapp)
	saved := guestMissionSave(t, g)
	h := chainFront(t, dir, chainModID)
	open, town, err := h.RestoreOriginal(saved)
	if err != nil || town {
		t.Fatalf("cold LOAD of the second link: town %v %v", town, err)
	}
	happ := h.App("guest chain load")
	if err := happ.OpenMission(open); err != nil {
		t.Fatal(err)
	}
	if !h.Town.guest.playing(chainSecondNo) || !bytes.Equal(h.Town.guest.stash, g.Town.guest.stash) ||
		len(h.Town.guest.rest) != 1 || h.Town.guest.member == nil || len(h.Town.guest.links()) != 2 {
		t.Fatal("the cold LOAD lost the chain")
	}
	if _, _, err := h.LiveCompleteCampaign(); err != nil {
		t.Fatal(err)
	}
	if got := guestDigest(t, h); got != want {
		t.Fatalf("the town after a LOAD inside the second link differs:\n got %s\nwant %s", got, want)
	}
	if reloaded, ok := guestMember(h); !ok || guestMemberJSON(t, reloaded) != guestMemberJSON(t, joined) {
		t.Fatal("the guest joined differently after the LOAD")
	}
	if _, _, err := guestFront(t, dir).RestoreOriginal(saved); err == nil || !strings.Contains(err.Error(), chainModID) {
		t.Fatalf("a LOAD without the mod: %v", err)
	}
}

// TestReleaseGuestPartyBuildsCharactersAtTheirLevels proves the party of a
// guest mission is built from the one character catalog at the level each
// member sets: companion 22 at level 3 takes the row of the third band of
// the composed record, the row the town grants in a mission of that band,
// and the mod's sellsword at level 1 takes the first row of her family.
func TestReleaseGuestPartyBuildsCharactersAtTheirLevels(t *testing.T) {
	dir := assembledMods(t, chainModID, chainModID)
	f := chainFront(t, dir, chainModID)
	ref := f.ChargenParty(ui.ChargenResult{Name: "Guest", Choices: []int{0, 0, 0}, Stats: []int{31, 27, 24, 29}})
	m := mod.Mission{Party: []mod.GuestMember{{Character: "npc22", Level: 3}, {Character: "sellsword", Level: 1}}}
	party, err := f.guestParty(m, 30, ref, nil)
	if err != nil || len(party) != 2 {
		t.Fatalf("party %v %v", party, err)
	}
	band, _ := townCompanion(f.townInstall(), 30, guestCompanion, ref)
	third, _ := townCompanion(f.townInstall(), 2*guestLevelMissions, guestCompanion, ref)
	if party[0].Hero != third.Hero || party[0].Hero == band.Hero || !party[0].StartingHero || party[0].CompanionNPC != guestCompanion {
		t.Fatalf("companion 22 at level 3 has %+v; the third band's row %+v, the town's %+v", party[0].Hero, third.Hero, band.Hero)
	}
	if party[1].Hero != rowHero(t, f, "NPC06_1") || party[1].StartingHero {
		t.Fatalf("the sellsword at level 1 has %+v", party[1].Hero)
	}
	if _, err := f.guestParty(mod.Mission{Party: []mod.GuestMember{{Character: "sellsword", Level: 4}, {Character: "npc999"}}}, 30, ref, nil); err == nil ||
		!strings.Contains(err.Error(), `character "npc999" is not in the character catalog`) {
		t.Fatalf("an unknown character: %v", err)
	}
}
