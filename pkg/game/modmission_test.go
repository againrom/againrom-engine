package game

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"againrom/pkg/mapload"
	"againrom/pkg/mod"
	"againrom/pkg/vfs"
)

// guestCampaign is a first-game campaign whose mission 20 opens the town of
// chapter 30, which grants companion 22.
func guestCampaign() Campaign {
	return Campaign{Main: []int{10, 20, 30, 40}, Side: []int{31}, Auto: map[int]int{10: 20},
		Offered: []int{30, 31, 40}, Chapters: map[int]Chapter{
			30: {Mission: 30, AddHero: []int{22}},
			40: {Mission: 40, AddHero: []int{23}},
		}}
}

func guestFixtureFront(t *testing.T) *FrontEnd {
	t.Helper()
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "scenario"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "scenario", "26.alm"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	fs, err := vfs.Open(nil, []string{dir})
	if err != nil {
		t.Fatal(err)
	}
	return &FrontEnd{InstallResources: InstallResources{Campaign: resolved(guestCampaign(), nil),
		Table: &mapload.Table{}, Archives: &Archives{Containers: fs}}}
}

func guestDecl() mod.Mission {
	return mod.Mission{Mod: "m", File: mod.MissionsFile, Line: 2, Key: "prologue", Number: 25, After: 20, Guest: 22,
		Chapter: 30, Carry: mod.CarryJoin, Title: "Title", Brief: "Briefing",
		MapBytes: []byte("MAP"), TextFiles: map[string][]byte{"event01.txt": []byte("E1")}}
}

func TestSetModMissionsRefusesWhatTheCampaignOrInstallDoesNotBearOut(t *testing.T) {
	for name, c := range map[string]struct {
		edit func(*mod.Mission)
		want string
	}{
		"main number":   {func(m *mod.Mission) { m.Number = 50 }, "number 50 is a multiple of 10"},
		"side mission":  {func(m *mod.Mission) { m.Number = 31 }, "number 31 is a mission of this install's campaign"},
		"installed map": {func(m *mod.Mission) { m.Number = 26 }, "number 26 is a map of this install (scenario/26.alm)"},
		"after side":    {func(m *mod.Mission) { m.After = 31 }, "after 31 is not a main mission"},
		"auto advance":  {func(m *mod.Mission) { m.After = 10 }, "mission 10 is followed by mission 20, not by a town"},
		"last mission":  {func(m *mod.Mission) { m.After = 40 }, "the win of mission 40 opens no town"},
		"chapter":       {func(m *mod.Mission) { m.Chapter = 40 }, "the win of mission 20 opens the town of chapter 30, not chapter 40"},
		"guest":         {func(m *mod.Mission) { m.Guest = 23 }, "the town of chapter 30 does not grant companion 23"},
	} {
		f := guestFixtureFront(t)
		m := guestDecl()
		c.edit(&m)
		err := f.SetModMissions(mod.MissionData{Missions: []mod.Mission{m}})
		if err == nil || !strings.Contains(err.Error(), `mod "m": data/missions.toml:2: `+c.want) {
			t.Errorf("%s: %v, want %q", name, err, c.want)
		}
		if !f.Table.Mods.Missions.Empty() {
			t.Errorf("%s: a refused declaration was recorded", name)
		}
	}
}

func TestSetModMissionsServesTheModsMapAndTextsUnderTheMissionNumber(t *testing.T) {
	f := guestFixtureFront(t)
	shared := f.Archives.Containers
	if err := f.SetModMissions(mod.MissionData{Missions: []mod.Mission{guestDecl()}}); err != nil {
		t.Fatal(err)
	}
	for address, want := range map[string]string{
		"scenario/25.alm":                   "MAP",
		"main/text/battle/m25/event01.txt":  "E1",
		"main/text/battle/m25/title.txt":    "Title",
		"main/text/battle/m25/briefing.txt": "Briefing",
	} {
		if got, err := f.Archives.Containers.ReadFile(address); err != nil || string(got) != want {
			t.Errorf("%s: %q %v, want %q", address, got, err, want)
		}
	}
	if _, err := shared.ReadFile("scenario/25.alm"); err == nil {
		t.Error("the overlay reached the shared filesystem")
	}
	if got, ok := f.Table.Mods.Missions.AfterWin(20); !ok || got.Key != "prologue" {
		t.Fatalf("recorded %+v", f.Table.Mods.Missions)
	}
}

func TestRecordWinOpensTheGuestMissionBeforeTheTown(t *testing.T) {
	c := guestCampaign()
	for name, tc := range map[string]struct {
		guest       func(int) (int, string)
		next        int
		arrived     bool
		wantMessage string
	}{
		"no guest":  {nil, 0, true, "mission 20 won - your party carries over; the town takes it from here"},
		"guest":     {func(n int) (int, string) { return 25, "" }, 25, false, "the party waits; guest mission 25 opens now"},
		"refused":   {func(n int) (int, string) { return 0, "no" }, 0, true, "the town takes it from here (the guest mission did not start: no)"},
		"none here": {func(n int) (int, string) { return 0, "" }, 0, true, "mission 20 won - your party carries over; the town takes it from here"},
	} {
		s := CampaignSession{Town: NewTown(c)}
		s.Town.Won(10)
		arrived, welcomed := false, false
		next, msg := s.recordWin(c, 20, nil, nil, nil, nil, townArrival{
			arrive: func() { arrived = true }, welcome: func(int) { welcomed = true }, guest: tc.guest})
		if next != tc.next || arrived != tc.arrived || welcomed != tc.arrived || !strings.Contains(msg, tc.wantMessage) {
			t.Errorf("%s: next %d arrived %v welcomed %v %q", name, next, arrived, welcomed, msg)
		}
		if !s.Town.Done(20) {
			t.Errorf("%s: the win was not recorded", name)
		}
	}
}

func TestGuestJoinsInHerCarriedStateOnlyThroughHerGrant(t *testing.T) {
	member := mapload.PartyMember{ID: "npc:22", Name: "Guest", CompanionNPC: 22, StartingHero: true, PlayerCharacter: true}
	g := &guestMission{mission: guestDecl(), start: member, stash: []byte("stash")}
	if !g.playing(25) || g.playing(20) || (*guestMission)(nil).playing(25) {
		t.Fatal("playing")
	}
	if g.returnMission(25) != 20 || g.returnMission(30) != 30 || (*guestMission)(nil).returnMission(25) != 25 {
		t.Fatal("returnMission")
	}
	if party, purse, ok := g.guestEntry(25); !ok || len(party) != 1 || party[0].Name != "Guest" || purse != 0 {
		t.Fatalf("guestEntry %v %d %v", party, purse, ok)
	}
	if _, ok := g.joining(30, 22); ok {
		t.Fatal("a guest still playing joined")
	}
	g.stash, g.member = nil, &member
	got, ok := g.joining(30, 22)
	if !ok || got.StartingHero || got.Name != "Guest" {
		t.Fatalf("joining %+v %v", got, ok)
	}
	for _, other := range [][2]int{{40, 22}, {30, 23}} {
		if _, ok := g.joining(other[0], other[1]); ok {
			t.Fatalf("joined through chapter %d companion %d", other[0], other[1])
		}
	}
	s := CampaignSession{Carried: []mapload.PartyMember{{ID: "hero", StartingHero: true}}, Town: &Town{guest: g}}
	if !s.carryTownCompanion(townInstall{campaign: guestCampaign()}, 30, 22) || s.Town.guest != nil ||
		len(s.Carried) != 2 || s.Carried[1].Name != "Guest" || s.Carried[1].StartingHero {
		t.Fatalf("the grant took %+v, guest %v", s.Carried, s.Town.guest)
	}
}

func TestRestoreGuestRefusesASaveTheActiveModsDoNotDeclare(t *testing.T) {
	ctx := mapload.ModContext{Missions: mod.MissionData{Missions: []mod.Mission{guestDecl()}}}
	start := captureCurrentParty(0, mapload.PartyMember{ID: "npc:22", CompanionNPC: 22})
	other := captureCurrentParty(0, mapload.PartyMember{ID: "npc:23", CompanionNPC: 23})
	record := func(edit func(*currentGuest)) *currentSessionData {
		r := &currentGuest{Mod: "m", Key: "prologue", Number: 25, Stash: []byte("s"), Start: &start}
		edit(r)
		return &currentSessionData{Guest: r}
	}
	for name, c := range map[string]struct {
		s    *currentSessionData
		want string
	}{
		"other mod": {record(func(r *currentGuest) { r.Mod = "x" }), `mission "prologue" of mod "x", which no active mod declares`},
		"number":    {record(func(r *currentGuest) { r.Number = 26 }), "as mission 26; the active mod declares mission 25"},
		"both":      {record(func(r *currentGuest) { r.Member = &start }), "invalid current guest record"},
		"neither":   {record(func(r *currentGuest) { r.Stash = nil }), "invalid current guest record"},
		"no start":  {record(func(r *currentGuest) { r.Start = nil }), "invalid current guest record"},
		"companion": {record(func(r *currentGuest) { r.Stash, r.Member = nil, &other }), "names companion 23, not 22"},
	} {
		if _, err := restoreGuest(c.s, ctx); err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%s: %v, want %q", name, err, c.want)
		}
	}
	if g, err := restoreGuest(record(func(*currentGuest) {}), ctx); err != nil || !g.playing(25) || string(g.stash) != "s" {
		t.Fatalf("a valid record: %+v %v", g, err)
	}
	if g, err := restoreGuest(&currentSessionData{}, mapload.ModContext{}); g != nil || err != nil {
		t.Fatal("a save without a guest gained one")
	}
}

func TestAssembleModMissionWritesOnlyIntoAModFolderOutsideTheInstall(t *testing.T) {
	install := t.TempDir()
	inside := filepath.Join(install, "mods", "m")
	outside := t.TempDir()
	for _, c := range []struct {
		lang, out, want string
	}{
		{"../en", outside, "is not a language code"},
		{"en", install, "is inside the install"},
		{"en", inside, "is inside the install"},
		{"en", outside, "is not a mod folder"},
	} {
		if _, err := AssembleModMission(install, c.lang, 10, c.out); err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%s %s: %v, want %q", c.lang, c.out, err, c.want)
		}
	}
	if entries, _ := os.ReadDir(install); len(entries) != 0 {
		t.Fatalf("the install folder gained %v", entries)
	}
}
