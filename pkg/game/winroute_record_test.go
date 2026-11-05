package game

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"image"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/base"
	"againrom/pkg/mapload"
	"againrom/pkg/mod"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
	"againrom/pkg/vfs"
)

// winRouteCampaign is guestCampaign with a payment on the wins of 20 and 30.
func winRouteCampaign() Campaign {
	c := guestCampaign()
	c.Chapters = map[int]Chapter{
		20: {Mission: 20, Payment: 150},
		30: {Mission: 30, AddHero: []int{22}, Payment: 300},
		40: {Mission: 40, AddHero: []int{23}},
	}
	c.Last = map[int]bool{40: true}
	return c
}

// winRouteMapFS is a world map whose registry places missions 20, 25, 27, 30
// and 40, so a world-map return names the mission it returns from.
func winRouteMapFS(t *testing.T) *vfs.FS {
	t.Helper()
	dir := t.TempDir()
	write := func(name string, files []synth.File) string {
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, synth.Archive(files), 0o600); err != nil {
			t.Fatal(err)
		}
		return path
	}
	mask := image.NewPaletted(image.Rect(0, 0, 640, 480), worldMapPalette)
	for x := 2; x <= 32; x++ {
		mask.SetColorIndex(x, 23, 1)
	}
	for y := 3; y <= 23; y++ {
		mask.SetColorIndex(2, y, 1)
	}
	mask.SetColorIndex(2, 3, 2)
	mask.SetColorIndex(32, 23, 2)
	mask.SetColorIndex(2, 13, 2)
	objects := []synth.RegNode{
		{Name: "General", Kind: kindDir, Children: []synth.RegNode{{Name: "ObjectCount", Kind: kindInt, Int: 3}}},
		{Name: "MapObject1", Kind: kindDir, Children: []synth.RegNode{
			{Name: "MapPoint", Kind: kindIntArray, Ints: []int32{2, 3}},
			{Name: "MapRect", Kind: kindIntArray, Ints: []int32{1, 2, 4, 5}}}},
		{Name: "MapObject2", Kind: kindDir, Children: []synth.RegNode{
			{Name: "MapPoint", Kind: kindIntArray, Ints: []int32{32, 23}},
			{Name: "MapRect", Kind: kindIntArray, Ints: []int32{30, 20, 8, 7}}}},
		{Name: "MapObject3", Kind: kindDir, Children: []synth.RegNode{
			{Name: "MapPoint", Kind: kindIntArray, Ints: []int32{2, 13}},
			{Name: "MapRect", Kind: kindIntArray, Ints: []int32{1, 12, 4, 5}}}},
		{Name: "MissionObjects", Kind: kindDir, Children: []synth.RegNode{
			{Name: "Mission20", Kind: kindInt, Int: 2},
			{Name: "Mission25", Kind: kindInt, Int: 3},
			{Name: "Mission27", Kind: kindInt, Int: 3},
			{Name: "Mission30", Kind: kindInt, Int: 2},
			{Name: "Mission40", Kind: kindInt, Int: 3}}},
	}
	main := write(MainArchive, []synth.File{{Path: "graphics/global.map/gmap.bmp", Data: synth.BMP24(image.NewRGBA(image.Rect(0, 0, 640, 480)))}})
	graphics := write(GraphicsArchive, []synth.File{{Path: "global.map/pathmap.bmp", Data: synth.BMP8(mask)}})
	scenario := write(ScenarioArchive, []synth.File{{Path: "globalmap.reg", Data: synth.Reg(kindRoot, objects)}})
	fsys, err := vfs.Open([]string{main, graphics, scenario}, nil)
	if err != nil {
		t.Fatal(err)
	}
	return fsys
}

// winRouteFirstFront is a first-game front end over the training city's party
// and table, with npc 22 in the table, the given mods' missions, a fresh town
// over winRouteCampaign and mission 10 won.
func winRouteFirstFront(t *testing.T, missions mod.MissionData) *FrontEnd {
	t.Helper()
	table := companionTable(t)
	humans := table.Humans.(dbCollection)
	humans[28].name, humans[29].name = "PC_Leader", "PC_Companion"
	seed := currentTrainingCity(t)
	s, label, err := seed.Snapshot(false)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := seed.ExportCurrentSave(s, label)
	if err != nil {
		t.Fatal(err)
	}
	f := &FrontEnd{InstallResources: InstallResources{Campaign: resolved(saveCampaign(), nil), Table: table,
		Archives: &Archives{Containers: winRouteMapFS(t)}, Font: resolved(missionFont(), nil)}}
	if _, town, err := f.RestoreOriginal(raw); err != nil || !town {
		t.Fatalf("city LOAD: %t %v", town, err)
	}
	c := winRouteCampaign()
	f.Campaign = resolved(c, nil)
	f.Town, f.Offered = NewTown(c), 0
	f.Carried = f.Carried[:1]
	f.Table.Mods.Missions = missions
	f.Table.Mods.Catalog = mod.InstallCharacters([]int32{22})
	return f
}

// winRouteWorld is a world won by its script with one entity standing for
// each of party's members.
func winRouteWorld(t *testing.T, party []mapload.PartyMember) (*sim.World, []sim.EntityID) {
	t.Helper()
	s, err := sim.NewScript(missionChecks(), []sim.ScriptInstant{{Op: sim.ScriptInstantWin}},
		[]sim.ScriptTrigger{missionTrigger(3, 1, true, 0)})
	if err != nil {
		t.Fatal(err)
	}
	var ents []sim.Entity
	var ids []sim.EntityID
	for i := range party {
		id := sim.EntityID(i)
		ents = append(ents, sim.Entity{ID: id, X: int32(10 + 2*i), Y: 10, HP: 20, MaxHP: 20, Owner: sim.SelfSlot,
			Domain: sim.DomainGround, TypeID: sim.HumanTypeID, GainsXP: true})
		ids = append(ids, id)
	}
	w, err := sim.NewScriptedWorld(1, sim.Bounds{Width: worldFixtureW, Height: worldFixtureH},
		sim.ModeCanonical, make([]byte, worldFixtureW*worldFixtureH), ents, s)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 64 && w.Outcome() == sim.OutcomeUndecided; i++ {
		sim.Step(w, nil)
	}
	if w.Outcome() != sim.OutcomeWon {
		t.Fatal("the fixture world was not won")
	}
	return w, ids
}

func winRouteDigest(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		return "unmarshalable: " + err.Error()
	}
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:8])
}

func winRouteParty(party []mapload.PartyMember) string {
	var parts []string
	var dto []currentPartyMember
	for _, p := range party {
		parts = append(parts, fmt.Sprintf("%s/%s/npc%d/hero%t", p.ID, p.Name, p.CompanionNPC, p.StartingHero))
		dto = append(dto, captureCurrentParty(0, p))
	}
	return "[" + strings.Join(parts, " ") + "] " + winRouteDigest(dto)
}

func winRouteGuest(g *guestMission, numbers []int) string {
	if g == nil {
		return "none"
	}
	out := fmt.Sprintf("record %s", winRouteDigest(g.record()))
	if g.stash != nil {
		out += fmt.Sprintf(" stash %d bytes", len(g.stash))
	}
	if g.member != nil {
		out += " member " + g.member.ID
	}
	for _, n := range numbers {
		out += fmt.Sprintf(" %d:%t/%d", n, g.playing(n), g.returnMission(n))
		if party, purse, ok := g.guestEntry(n); ok {
			out += fmt.Sprintf("/entry%s+%d", winRouteParty(party), purse)
		}
	}
	return out
}

// winRouteRecorder writes what each win leaves as lines.
type winRouteRecorder struct {
	t       *testing.T
	lines   []string
	numbers []int
	// tag names the front end a win is played on.
	tag string
}

func (r *winRouteRecorder) add(format string, args ...any) {
	r.lines = append(r.lines, fmt.Sprintf(format, args...))
}

// state records the front end's place, purse, party and guest.
func (r *winRouteRecorder) state(name string, f *FrontEnd) {
	r.add("%s: offered %d gold %d open %t chapter %d", name, f.Offered, f.Town.Gold(), f.Town.Open(), f.Town.Chapter())
	r.add("%s: party %s", name, winRouteParty(f.Carried))
	r.add("%s: guest %s", name, winRouteGuest(f.Town.guest, r.numbers))
	if g := f.Town.guest; g != nil && g.stash != nil {
		r.add("%s: stash sav %s", name, campaignRecordSAVDigest(r.t, g.stash))
	}
	if s, ok := f.TownScreen().(*townScreen); ok && s.worldMap != nil {
		r.add("%s: world map return %d", name, s.worldMap.returnMission)
	}
}

// save records the SAV a SAVE right after the win writes and what its LOAD
// restores; it answers the front end the LOAD restored, nil when none.
func (r *winRouteRecorder) save(name string, f *FrontEnd) *FrontEnd {
	s, label, err := f.Snapshot(false)
	if err != nil {
		r.add("%s: save refused: %v", name, err)
		return nil
	}
	raw, err := f.ExportCurrentSave(s, label)
	if err != nil {
		r.add("%s: save refused: %v", name, err)
		return nil
	}
	r.add("%s: sav %s", name, campaignRecordSAVDigest(r.t, raw))
	cold := &FrontEnd{InstallResources: f.InstallResources}
	open, town, err := cold.RestoreOriginal(raw)
	if err != nil {
		r.add("%s: load refused: %v", name, err)
		return nil
	}
	r.add("%s: load town %t opener %t", name, town, open != nil)
	r.state(name+" loaded", cold)
	return cold
}

// win acknowledges the won mission n played by party through the transition
// front and records where it leads. It answers the front end the SAV written
// right after the win loads into, nil when none.
func (r *winRouteRecorder) win(f *FrontEnd, n int, party []mapload.PartyMember) *FrontEnd {
	w, ids := winRouteWorld(r.t, party)
	ms := &Mission{Number: n, World: w, Party: mapload.CloneParty(party), Start: mapload.Start{IDs: ids}}
	dest, line, open := continueMission(frontTransitions{f}, n, ms, listAdvance)()
	name := fmt.Sprintf("win %d%s", n, r.tag)
	r.add("%s: dest %v opener %t line %q", name, dest, open != nil, line)
	r.state(name, f)
	r.travelHome(name, f)
	return r.save(name, f)
}

// secondDirect is the second game's won mission 10 finished through
// FinishMission, the entry headless callers use, with the bank held.
func (r *winRouteRecorder) secondDirect() {
	f, _, _ := secondCampaignFixture(r.t, true)
	c := newSecondCampaign()
	c.current, c.available = partyPlace{1, 10}, []partyPlace{{1, 10}}
	f.Town.second = c
	party := []mapload.PartyMember{{ID: "hero", Name: "hero", StartingHero: true, PlayerCharacter: true}}
	next, line := f.FinishMission(10, party, nil, nil)
	r.add("second direct 10: next %d line %q current %v available %v bank %s offered %d", next, line, c.current, c.available, winRouteDigest(c.bank), f.Offered)
	r.add("second direct 10: party %s", winRouteParty(f.Carried))
}

// travelHome walks a pending world-map return home, as the player's return
// trip does, so the town SAVE the win allows can be written.
func (r *winRouteRecorder) travelHome(name string, f *FrontEnd) {
	s, ok := f.TownScreen().(*townScreen)
	if !ok || s.worldMap == nil || s.worldMap.returnMission == 0 {
		return
	}
	ticks := 0
	for ; ticks < 4000 && s.worldMap != nil && s.worldMap.returnMission != 0; ticks++ {
		s.WorldMapTick()
	}
	r.add("%s: home after %d ticks, at the square %t", name, ticks, s.AtTownSquare())
}

// secondWin acknowledges the second game's won mission n from bank stage
// stage through the production panel and records where it leads.
func (r *winRouteRecorder) secondWin(n int, stage int32) {
	t := r.t
	f, app, _ := secondCampaignFixture(t, true)
	c := newSecondCampaign()
	c.bank[768] = stage
	c.current, c.available = partyPlace{1, n}, []partyPlace{{1, n}}
	f.Town.second = c
	script, err := sim.NewROM2Script(missionChecks(), []sim.ScriptInstant{{Op: sim.ScriptInstantWin}}, []sim.ScriptTrigger{missionTrigger(3, 1, true, 0)})
	if err != nil {
		t.Fatal(err)
	}
	hero := []sim.Entity{{ID: 0, X: 10, Y: 10, HP: 20, MaxHP: 20, Owner: sim.SelfSlot, Domain: sim.DomainGround, TypeID: sim.HumanTypeID, GainsXP: true}}
	w, err := sim.NewScriptedWorld(1, sim.Bounds{Width: worldFixtureW, Height: worldFixtureH},
		sim.ModeCanonical, make([]byte, worldFixtureW*worldFixtureH), hero, script)
	if err != nil {
		t.Fatal(err)
	}
	w.SetROM2ScenarioState(c.bank)
	mw, view := missionDriverFor(t, w, nil, nil)
	mw.mission.table = readUnder(base.GameROM2, &mapload.Table{})
	party := []mapload.PartyMember{{ID: "hero", Name: "hero", StartingHero: true, PlayerCharacter: true}}
	ms := &Mission{Number: n, World: w, Party: party, Start: mapload.Start{IDs: []sim.EntityID{0}}}
	advance := continueMission(frontTransitions{f}, n, ms, mw.advanceNotice)
	f.live, f.liveMission = mw, n
	if err := app.OpenMission(openPrepared(preparedMap{viewer: view, tick: mw.deterministicFrame, cadence: mw.setCadenceMode, advance: advance}, func() {})); err != nil {
		t.Fatal(err)
	}
	for tick := 0; tick < 32 && !mw.mission.open; tick++ {
		if err := app.HeadlessStep(); err != nil {
			t.Fatal(err)
		}
	}
	if !mw.mission.open || mw.mission.kind != ui.NoticeSuccess {
		t.Fatal("script victory did not reach production acknowledgement")
	}
	if err := app.HeadlessActivate("notice"); err != nil {
		r.add("second win %d: activate %v", n, err)
	}
	name := fmt.Sprintf("second win %d", n)
	r.add("%s: screen %v live %t message %q", name, app.Screen(), f.live != nil, app.HeadlessMessage())
	r.add("%s: current %v available %v bank %s offered %d won %t", name, c.current, c.available, winRouteDigest(c.bank), f.Offered, f.Town.won[n])
	r.add("%s: party %s", name, winRouteParty(f.Carried))
	r.save(name, f)
}

// winRouteRecord is what every win of the fixtures leaves: the first game's
// chapters over to the ending, a two-link mod chain after the win of 20, and
// the second game's departures that open a town, a mission and the map.
func winRouteRecord(t *testing.T, chain mod.MissionData) []string {
	r := &winRouteRecorder{t: t, numbers: []int{20, 25, 27, 30}}
	f := winRouteFirstFront(t, mod.MissionData{})
	r.state("start", f)
	var loaded *FrontEnd
	for _, n := range []int{10, 20, 30, 40} {
		if back := r.win(f, n, f.Carried); n == 20 {
			loaded = back
		}
	}
	// The town SAV after the win of 20 plays on: a restored campaign's win.
	r.tag = " after the load"
	for _, n := range []int{30, 40} {
		r.win(loaded, n, loaded.Carried)
	}
	r.tag = ""

	g := winRouteFirstFront(t, chain)
	r.win(g, 10, g.Carried)
	r.win(g, 20, g.Carried)
	for _, n := range []int{25, 27} {
		party, _, ok := g.Town.guest.guestEntry(n)
		if !ok {
			r.add("guest %d: no entry", n)
			continue
		}
		r.win(g, n, party)
	}

	for _, c := range []struct {
		n     int
		stage int32
	}{{10, 10}, {20, 20}, {40, 40}} {
		r.secondWin(c.n, c.stage)
	}
	r.secondDirect()
	return r.lines
}

// winRouteChain is the two-link chain the record plays: link 25 after the
// win of 20, whose guest joins chapter 30, and link 27 after it.
func winRouteChain() mod.MissionData {
	return mod.MissionData{Missions: []mod.Mission{guestDecl(), chainLink("m", "next", 9, 27, 25, "npc22")}}
}

// Every win of both games and of a two-link mod chain leads where it led,
// pays, joins and carries what it did, keeps and returns the guest's stash
// as it did, and the SAV a SAVE writes right after it, and what that SAV
// loads, are the base's.
func TestWinRouteRecordIsTheBaseRecord(t *testing.T) {
	got := winRouteRecord(t, winRouteChain())
	if strings.Join(got, "\n") == strings.Join(winRouteBaseRecord, "\n") {
		return
	}
	for i := 0; i < len(got) || i < len(winRouteBaseRecord); i++ {
		var g, w string
		if i < len(got) {
			g = got[i]
		}
		if i < len(winRouteBaseRecord) {
			w = winRouteBaseRecord[i]
		}
		if g != w {
			t.Errorf("line %d:\n got %s\nwant %s", i, g, w)
		}
	}
	t.Log("record:\n\t" + strings.Join(quoteLines(got), ",\n\t") + ",")
}

func quoteLines(lines []string) []string {
	out := make([]string, len(lines))
	for i, l := range lines {
		out[i] = fmt.Sprintf("%q", l)
	}
	return out
}

// winRouteBaseRecord is winRouteRecord at the base, before the win finisher
// and the session mission identity.
var winRouteBaseRecord = []string{
	"start: offered 0 gold 100 open false chapter 10",
	"start: party [hero/Leader/npc0/herotrue] f3c3d71e78bb8bf2",
	"start: guest none",
	"win 10: dest 3 opener true line \"mission 10 won - your party carries over; mission 20 follows and opens now\"",
	"win 10: offered 0 gold 0 open false chapter 20",
	"win 10: party [hero/Leader/npc0/herotrue] c83fce8936b63663",
	"win 10: guest none",
	"win 10: save refused: there is no game to save yet",
	"win 20: dest 4 opener false line \"mission 20 won - scenario payment +150 gold - your party carries over; the town takes it from here\"",
	"win 20: offered 30 gold 150 open true chapter 30",
	"win 20: party [hero/Leader/npc0/herotrue npc:22/PC_Companion/npc22/herofalse] f3992b89ed1c3def",
	"win 20: guest none",
	"win 20: world map return 20",
	"win 20: home after 7 ticks, at the square true",
	"win 20: sav 708496d39adb1dbd38e7b6a9",
	"win 20: load town true opener false",
	"win 20 loaded: offered 30 gold 150 open true chapter 30",
	"win 20 loaded: party [hero/Leader/npc0/herotrue npc:22/PC_Companion/npc22/herofalse] f3992b89ed1c3def",
	"win 20 loaded: guest none",
	"win 30: dest 4 opener false line \"mission 30 won - scenario payment +300 gold - your party carries over; the town takes it from here\"",
	"win 30: offered 40 gold 300 open true chapter 40",
	"win 30: party [hero/Leader/npc0/herotrue npc:22/PC_Companion/npc22/herofalse] 9d36ba7b39e8fdfa",
	"win 30: guest none",
	"win 30: world map return 30",
	"win 30: home after 7 ticks, at the square true",
	"win 30: sav d09d59a5b0664fe35f89028d",
	"win 30: load town true opener false",
	"win 30 loaded: offered 40 gold 300 open true chapter 40",
	"win 30 loaded: party [hero/Leader/npc0/herotrue npc:22/PC_Companion/npc22/herofalse] 9d36ba7b39e8fdfa",
	"win 30 loaded: guest none",
	"win 40: dest 6 opener false line \"\"",
	"win 40: offered 0 gold 0 open true chapter 0",
	"win 40: party [hero/Leader/npc0/herotrue npc:22/PC_Companion/npc22/herofalse] 9d36ba7b39e8fdfa",
	"win 40: guest none",
	"win 40: world map return 0",
	"win 40: save refused: the campaign is complete; there is no save point after its ending",
	"win 30 after the load: dest 4 opener false line \"mission 30 won - scenario payment +300 gold - your party carries over; the town takes it from here\"",
	"win 30 after the load: offered 40 gold 300 open true chapter 40",
	"win 30 after the load: party [hero/Leader/npc0/herotrue npc:22/PC_Companion/npc22/herofalse] 9d36ba7b39e8fdfa",
	"win 30 after the load: guest none",
	"win 30 after the load: world map return 30",
	"win 30 after the load: home after 7 ticks, at the square true",
	"win 30 after the load: sav 3036973e7fc067ce4415690c",
	"win 30 after the load: load town true opener false",
	"win 30 after the load loaded: offered 40 gold 300 open true chapter 40",
	"win 30 after the load loaded: party [hero/Leader/npc0/herotrue npc:22/PC_Companion/npc22/herofalse] 9d36ba7b39e8fdfa",
	"win 30 after the load loaded: guest none",
	"win 40 after the load: dest 6 opener false line \"\"",
	"win 40 after the load: offered 0 gold 0 open true chapter 0",
	"win 40 after the load: party [hero/Leader/npc0/herotrue npc:22/PC_Companion/npc22/herofalse] 9d36ba7b39e8fdfa",
	"win 40 after the load: guest none",
	"win 40 after the load: world map return 0",
	"win 40 after the load: save refused: the campaign is complete; there is no save point after its ending",
	"win 10: dest 3 opener true line \"mission 10 won - your party carries over; mission 20 follows and opens now\"",
	"win 10: offered 0 gold 0 open false chapter 20",
	"win 10: party [hero/Leader/npc0/herotrue] c83fce8936b63663",
	"win 10: guest none",
	"win 10: save refused: there is no game to save yet",
	"win 20: dest 3 opener true line \"mission 20 won - scenario payment +150 gold - the party waits; guest mission 25 opens now\"",
	"win 20: offered 0 gold 150 open true chapter 30",
	"win 20: party [hero/Leader/npc0/herotrue] c83fce8936b63663",
	"win 20: guest record ea8c6158225a5f43 stash 3719 bytes 20:false/20 25:true/20/entry[npc:22/PC_Companion/npc22/herotrue] be173abe294ab983+0 27:false/27 30:false/30",
	"win 20: stash sav d007a0fe8527f7ae343286bb",
	"win 20: sav c56a8e27eb3e9b001eb30fa0",
	"win 20: load town true opener false",
	"win 20 loaded: offered 0 gold 150 open true chapter 30",
	"win 20 loaded: party [hero/Leader/npc0/herotrue npc:22/PC_Companion/npc22/herofalse] f3992b89ed1c3def",
	"win 20 loaded: guest record ea8c6158225a5f43 stash 3719 bytes 20:false/20 25:true/20/entry[npc:22/PC_Companion/npc22/herotrue] be173abe294ab983+0 27:false/27 30:false/30",
	"win 20 loaded: stash sav d007a0fe8527f7ae343286bb",
	"win 25: dest 3 opener true line \"mission 25 won - the guest party goes on to mission 27\"",
	"win 25: offered 0 gold 0 open true chapter 30",
	"win 25: party [npc:22/PC_Companion/npc22/herotrue] aa1123dbf2d90d54",
	"win 25: guest record 47f764ec0b503746 stash 3719 bytes member npc:22 20:false/20 25:false/20 27:true/20/entry[npc:22/PC_Companion/npc22/herotrue] aa1123dbf2d90d54+0 30:false/30",
	"win 25: stash sav d007a0fe8527f7ae343286bb",
	"win 25: sav c954a7df83d41d6256bc66c5",
	"win 25: load town true opener false",
	"win 25 loaded: offered 0 gold 0 open true chapter 30",
	"win 25 loaded: party [npc:22/PC_Companion/npc22/herotrue] aa1123dbf2d90d54",
	"win 25 loaded: guest record 47f764ec0b503746 stash 3719 bytes member npc:22 20:false/20 25:false/20 27:true/20/entry[npc:22/PC_Companion/npc22/herotrue] aa1123dbf2d90d54+0 30:false/30",
	"win 25 loaded: stash sav d007a0fe8527f7ae343286bb",
	"win 27: dest 4 opener false line \"mission 27 won - PC_Companion returns; the party goes on to the town\"",
	"win 27: offered 30 gold 150 open true chapter 30",
	"win 27: party [hero/Leader/npc0/herotrue npc:22/PC_Companion/npc22/herofalse] 9d36ba7b39e8fdfa",
	"win 27: guest none",
	"win 27: world map return 27",
	"win 27: home after 2 ticks, at the square true",
	"win 27: sav fac1fc08a5e748d20f12c128",
	"win 27: load town true opener false",
	"win 27 loaded: offered 30 gold 150 open true chapter 30",
	"win 27 loaded: party [hero/Leader/npc0/herotrue npc:22/PC_Companion/npc22/herofalse] 9d36ba7b39e8fdfa",
	"win 27 loaded: guest none",
	"second win 10: screen picker live true message \"read scenario/20.alm: readfile scenario/20.alm: file does not exist\"",
	"second win 10: current {0 0} available [{1 20}] bank 3c12238b11c3de8f offered 20 won true",
	"second win 10: party [hero/hero/npc0/herotrue] f1804fdb984b176b",
	"second win 10: save refused: saving is not available for this game yet",
	"second win 20: screen town live false message \"\"",
	"second win 20: current {0 0} available [{2 2}] bank efb8440f6848221c offered 0 won true",
	"second win 20: party [hero/hero/npc0/herotrue] f1804fdb984b176b",
	"second win 20: save refused: saving is not available for this game yet",
	"second win 40: screen picker live true message \"read scenario/50.alm: readfile scenario/50.alm: file does not exist\"",
	"second win 40: current {0 0} available [{1 50} {1 60}] bank 313b7e5ef6b6cadf offered 50 won true",
	"second win 40: party [hero/hero/npc0/herotrue] f1804fdb984b176b",
	"second win 40: save refused: saving is not available for this game yet",
	"second direct 10: next 0 line \"mission 10 won - your party carries over; this install declares no campaign, so nothing names what follows\" current {1 10} available [{1 10}] bank 707e59a48abe4e6b offered 0",
	"second direct 10: party [hero/hero/npc0/herotrue] 87637b35dbf681f3",
}
