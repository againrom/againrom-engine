package game

import (
	"bytes"
	"encoding/binary"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// identityDollTable is the doll table with the identity rows 30 and 31.
func identityDollTable() *mapload.Table {
	t := secondDollTable()
	t.Humans = identityTable().Humans
	return t
}

// identityTownSave is a Kaarg town SAVE after the inn talk joined identity
// 22: the hero and the companion, whose third school is his largest, 500
// gold, missions 30 and 31 open.
func identityTownSave(t *testing.T) []byte {
	t.Helper()
	f, _, _ := secondCampaignFixtureFiles(t, false, identityNames())
	f.Table = identityDollTable()
	f.Carried = []mapload.PartyMember{{ID: "hero", Name: "Aldor", PlayerCharacter: true, StartingHero: true}}
	c := newSecondCampaign()
	c.current, c.available = partyPlace{2, 2}, []partyPlace{{2, 2}, {1, 30}, {1, 31}}
	c.bank[768] = 30
	f.Town.second, f.Town.gold = c, 500
	f.Town.settleCityGroups(f.Carried)
	if !f.CampaignSession.addPartyIdentity(f.townInstall(), 22) {
		t.Fatal("the producer added no identity 22")
	}
	f.Carried[1].Hero.Skill[3] = 30
	return exportTown(t, f)
}

func exportTown(t *testing.T, f *FrontEnd) []byte {
	t.Helper()
	s, label, err := f.Snapshot(false)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := f.ExportCurrentSave(s, label)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func coldIdentityTown(t *testing.T, raw []byte) *FrontEnd {
	t.Helper()
	cold, _, _ := secondCampaignFixtureFiles(t, false, identityNames())
	cold.Table = identityDollTable()
	if _, town, err := cold.RestoreOriginal(raw); err != nil || !town {
		t.Fatal("cold town LOAD", town, err)
	}
	return cold
}

// checkIdentityRecords: the file's records hold the companion as the
// original's addition producer builds him: his identity 22 in U148, the
// Humans row of serverID 10220, the main skill of a `_Hero` template, in a
// Player group of his own.
func checkIdentityRecords(t *testing.T, name string, raw []byte) {
	t.Helper()
	doc, err := sav.DecodeDocumentData(raw)
	if err != nil {
		t.Fatal(err)
	}
	var identities []uint32
	for i := range doc.Objects {
		r := &doc.Objects[i]
		if r.Class != "Human" {
			continue
		}
		id, _ := savedStructureValue(r, "U148")
		identities = append(identities, id)
		if id != 22 {
			continue
		}
		row, _ := savedStructureValue(r, "T0C")
		if main := binary.LittleEndian.Uint32(documentRawBytes(t, r, "H1CC")); row != 30 || main != 3 {
			t.Fatalf("%s: the companion holds row %d and main skill %d, want 30 and 3", name, row, main)
		}
	}
	if !slices.Equal(identities, []uint32{21, 22}) {
		t.Fatalf("%s: the Humans hold identities %v, want 21 and 22", name, identities)
	}
	groups := doc.Objects[doc.Players[0]-1].Groups
	if len(groups) != 2 {
		t.Fatalf("%s: the Player holds %d groups, want one per member", name, len(groups))
	}
	for _, g := range groups {
		if refs, _ := savedObjectRefs(&g, "Actors"); len(refs) != 1 {
			t.Fatalf("%s: a group holds %d actors, want one", name, len(refs))
		}
	}
}

// A town file holding the companion the inn talk joined keeps him: the
// engine's SAVE writes him in the second game's records, a LOAD of the
// whole file or of its records alone restores him as identity 22, and the
// resave writes him again.
func TestSecondTownFileKeepsTheJoinedCompanion(t *testing.T) {
	raw := identityTownSave(t)
	for _, file := range []struct {
		name string
		raw  []byte
	}{{"its records alone", withoutEngineDocument(t, raw)}, {"the whole file", raw}} {
		cold := coldIdentityTown(t, file.raw)
		if got := carriedIdentities(cold.Carried, cold.Table); !slices.Equal(got, []int{21, 22}) || cold.Carried[1].ID != "npc:22" {
			t.Fatalf("%s: LOAD carries identities %v, want 21 and 22", file.name, got)
		}
		checkIdentityRecords(t, file.name+" resaved", exportTown(t, cold))
	}
	checkIdentityRecords(t, "the engine's SAVE", raw)
}

// druidsTownFile is an original town 3 file after mission 50's departure:
// the hero, identity 22 and identity 23 in groups of their own, the hero
// and identity 22 each carrying a staff and a robe in their own orders.
func druidsTownFile() []byte {
	staff := secondPiece{class: "Weapon", code: 0x812d, stack: 1, row: 13, kind: 2, price: 3819}
	robe := secondPiece{class: "Armor", code: 0xf72d, stack: 1, row: 13, kind: 1, price: 250}
	hero := secondFixtureHero()
	hero.bag = []secondPiece{staff, robe}
	iglez := identityMissionRecord()
	iglez.bag = []secondPiece{robe, staff}
	dina := identityMissionRecord()
	dina.defRow, dina.mapUnitID, dina.key, dina.identity = 31, 126, 0x0ff00003, 23
	tail := secondSavedTail(0, [2]uint32{3, placeTown})
	binary.LittleEndian.PutUint32(tail[12+4*768:], 60)
	return labelledContainer(sav.SecondGame.Magic, "druids", secondSavedPartyOf(0, 44, []secondOriginalHero{hero, iglez, dina}, false, 0, nil),
		append(shortcutsStore(), tail...))
}

// humanPacks is each Human's Inventory item codes by its identity.
func humanPacks(t *testing.T, raw []byte) map[uint32][]uint16 {
	t.Helper()
	doc, err := sav.DecodeDocumentData(raw)
	if err != nil {
		t.Fatal(err)
	}
	out := map[uint32][]uint16{}
	for i := range doc.Objects {
		r := &doc.Objects[i]
		if r.Class != "Human" {
			continue
		}
		id, _ := savedStructureValue(r, "U148")
		refs, _ := savedObjectRefs(r, "Inventory")
		codes := []uint16{}
		for _, ref := range refs {
			code, _ := savedStructureValue(&doc.Objects[ref-1], "F40")
			codes = append(codes, uint16(code))
		}
		out[id&0xffff] = codes
	}
	return out
}

// A LOAD of an original town file puts each carried piece where the file
// holds it: the hero's pack goes to the campaign's bag, and each companion
// keeps his own pack in its order. The SAVE after it writes each member's
// pack in his own Human again.
func TestSecondTownFileLoadKeepsEachCompanionsPack(t *testing.T) {
	cold := coldIdentityTown(t, druidsTownFile())
	if got := carriedIdentities(cold.Carried, cold.Table); !slices.Equal(got, []int{21, 22, 23}) {
		t.Fatalf("LOAD carries identities %v, want 21, 22 and 23", got)
	}
	codes := func(m mapload.PartyMember) []uint16 {
		var out []uint16
		for _, it := range m.CarriedItems {
			out = append(out, it.Code)
		}
		return out
	}
	packs := map[string][]uint16{}
	for _, m := range cold.Carried {
		packs[m.ID] = codes(m)
	}
	if !slices.Equal(packs["npc:22"], []uint16{0xf72d, 0x812d}) || len(packs["npc:23"]) != 0 {
		t.Fatalf("the companions carry %v, want identity 22's robe and staff", packs)
	}
	c := cold.Town.second
	if c == nil || c.current != (partyPlace{placeTown, 3}) || len(c.party) != 1 || !slices.Equal(codes(c.party[0]), []uint16{0x812d, 0xf72d}) {
		t.Fatalf("the campaign stands at %+v with bag %v, want town 3 and the hero's staff and robe", c.current, c.party)
	}
	saved := humanPacks(t, exportTown(t, cold))
	if !slices.Equal(saved[22], []uint16{0xf72d, 0x812d}) || len(saved[23]) != 0 || !slices.Equal(saved[21], []uint16{0x812d, 0xf72d}) {
		t.Fatalf("the SAVE writes packs %v, want each member's own", saved)
	}
}

// The pair list, the two words after it and each location's 16 raw bytes
// of a loaded tail are written back by the town SAVE and by the SAVE after
// a LOAD of that file; a location the file did not list writes zero.
func TestSecondTownTailWritesTheLoadedWords(t *testing.T) {
	doc, err := sav.DecodeDocumentData(withoutEngineDocument(t, identityTownSave(t)))
	if err != nil {
		t.Fatal(err)
	}
	raw16 := func(x, y uint32) (out [16]byte) {
		for i, v := range []uint32{x, y, x, y} {
			binary.LittleEndian.PutUint32(out[4*i:], v)
		}
		return out
	}
	tail := doc.SecondTail
	tail.Pairs, tail.PairTail = [][2]uint32{{1, 1}, {7, 9}}, [2]uint32{0xb8, 0x06}
	for i, at := range [][2]uint32{{370, 305}, {331, 395}, {443, 225}} {
		tail.Locations[i].Raw = raw16(at[0], at[1])
	}
	loaded, err := sav.EncodeDocumentData(doc)
	if err != nil {
		t.Fatal(err)
	}
	cold := coldIdentityTown(t, loaded)
	cold.Town.second.available = append(cold.Town.second.available, partyPlace{1, 32})
	first := exportTown(t, cold)
	again := exportTown(t, coldIdentityTown(t, first))
	for name, raw := range map[string][]byte{"the resave": first, "its resave": again} {
		got, err := sav.DecodeDocumentData(raw)
		if err != nil {
			t.Fatal(err)
		}
		g := got.SecondTail
		if !slices.Equal(g.Pairs, tail.Pairs) || g.PairTail != tail.PairTail || len(g.Locations) != 4 ||
			!slices.Equal(g.Locations[:3], tail.Locations) || g.Locations[3].Raw != ([16]byte{}) {
			t.Fatalf("%s: pairs %v, words %#x, locations %+v", name, g.Pairs, g.PairTail, g.Locations)
		}
	}
}

// A Kaarg town file written after the engine's own play: a LOAD of an
// original mission 20 file, its victory taken, the map's travel to Kaarg and
// SAVE. A cold LOAD of that file stands at Kaarg with the file's campaign
// position, bank, hero, gold and bag, and its resave writes the same bytes.
func TestSecondTownFileAfterMissionTwentyLoadsBack(t *testing.T) {
	hero := secondMissionFixtureHero()
	tail := secondSavedTail(0, [2]uint32{20, placeMission})
	binary.LittleEndian.PutUint32(tail[12+4*768:], 20)
	file := labelledContainer(sav.SecondGame.Magic, "mission", secondSavedDocument(20, 44, &hero, true),
		append(secondMissionStore(4, 0, -1), tail...))
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "game0001.sav"), file, 0o600); err != nil {
		t.Fatal(err)
	}
	content := secondMissionContent{side: 128, font: true, mission: 20}
	f, app := secondMissionLoadOf(t, content, dir, "game0001.sav")
	if f.live == nil || f.liveMission != 20 {
		t.Fatalf("the LOAD opened mission %d", f.liveMission)
	}
	secondContinuedWin(t, f, app)
	if err := app.HeadlessActivate("notice"); err != nil {
		t.Fatal(err)
	}
	screen := f.TownScreen().(*townScreen)
	secondMapFixturePoints(screen)
	if err := secondMapTravel(app, screen, partyPlace{2, 2}); err != nil {
		t.Fatal(err, app.HeadlessMessage())
	}
	c := f.Town.second
	if c.current != (partyPlace{2, 2}) || c.bank[916] != 1 || c.bank[768] != 30 {
		t.Fatalf("the victory and the travel left the party at %v, stage %d", c.current, c.bank[768])
	}
	raw := exportTown(t, f)
	cold := secondMissionFrontOf(t, content)
	if _, town, err := cold.RestoreOriginal(raw); err != nil || !town {
		t.Fatal("cold town LOAD", town, err)
	}
	got := cold.Town.second
	if got.current != c.current || !slices.Equal(got.available, c.available) || got.bank != c.bank || cold.Town.gold != f.Town.gold {
		t.Fatalf("the LOAD stands at %v of %v with %d gold, want %v of %v with %d",
			got.current, got.available, cold.Town.gold, c.current, c.available, f.Town.gold)
	}
	party := cold.NextParty()
	if len(party) != 1 || party[0].Name != hero.name || party[0].Hero.Mind != 43 {
		t.Fatalf("the LOAD carries %d members, first %q", len(party), party[0].Name)
	}
	items := func(f *FrontEnd) (out []uint16) {
		for _, m := range f.Town.second.party {
			for _, it := range m.CarriedItems {
				out = append(out, it.Code)
			}
		}
		return out
	}
	if want := items(f); len(want) == 0 || !slices.Equal(items(cold), want) {
		t.Fatalf("the LOAD's bag holds %v, want %v", items(cold), want)
	}
	if again := exportTown(t, cold); !bytes.Equal(again, raw) {
		t.Fatal("the resave of the Kaarg file writes other bytes")
	}
}

// After a LOAD of an original mission file the campaign's party is the
// World's hero, not a fresh one.
func TestSecondMissionFileLoadCarriesTheWorldHero(t *testing.T) {
	hero := secondMissionFixtureHero()
	f, _, _ := secondMissionLoadFile(t, secondMissionFile(hero, secondMissionStore(4, 0, -1)))
	party := f.NextParty()
	if len(party) != 1 || !party[0].StartingHero || party[0].Name != hero.name || party[0].Hero.Mind != 43 {
		t.Fatalf("the campaign's party after the LOAD is %d members, first %q mind %d; want the file's %q",
			len(party), party[0].Name, party[0].Hero.Mind, hero.name)
	}
}

// identityMissionRecord is identity 22 as an original mission file holds
// him after the Kaarg inn talk: a Human of the hero's Player in a group of
// his own, Humans row 30, U148 22 and no name.
func identityMissionRecord() secondOriginalHero {
	h := secondMissionFixtureHero()
	h.name, h.defRow, h.mapUnitID, h.key, h.identity = "", 30, 125, 0x0ff00002, 22
	h.cell = [2]uint8{13, 21}
	h.held, h.worn, h.bag, h.spells, h.bookSlots = nil, nil, nil, nil, 0
	h.skills, h.mainSkill = [6]uint16{0, 25, 0, 0, 0, 10}, 1
	return h
}

// identityMissionLoad LOADs an original mission 30 file holding the hero
// and identity 22 into a second-game front whose Humans hold the identity
// rows.
func identityMissionLoad(t *testing.T) (*FrontEnd, *ui.App) {
	t.Helper()
	party := []secondOriginalHero{secondMissionFixtureHero(), identityMissionRecord()}
	file := labelledContainer(sav.SecondGame.Magic, "mission", secondSavedPartyOf(30, 44, party, true, 0, nil),
		append(secondMissionStore(4, 0, -1), secondSavedTail(0, [2]uint32{30, placeMission})...))
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "game0001.sav"), file, 0o600); err != nil {
		t.Fatal(err)
	}
	f := secondMissionFrontOf(t, secondMissionContent{font: true, mission: 30})
	humans := slices.Clone(f.Table.Humans.(dbCollection))
	rows := identityTable().Humans.(dbCollection)
	for len(humans) < len(rows) {
		humans = append(humans, dbEntry{})
	}
	humans[30], humans[31] = rows[30], rows[31]
	f.Table.Humans = humans
	app := f.App("second-mission")
	app.Layout(1024, 768)
	save, list, load := f.SaveSeams(SaveStore{Dir: dir}, OriginalStore{}, nil)
	app.SetSaveSeams(save, list, load)
	groundAppLoad(t, app, list, "game0001.sav")
	return f, app
}

// A LOAD of an original mission file holding the companion the Kaarg inn
// talk joined restores him as the party member identity 22 (R2-ENGINE-106),
// as the engine's own join carries him, not as an ally the map lent; his
// fall loses the mission with reason 3, identity - 19 (R2-ENGINE-625).
func TestSecondMissionFileLoadKeepsTheJoinedCompanion(t *testing.T) {
	f, _ := identityMissionLoad(t)
	mw := f.live
	if mw == nil || f.liveMission != 30 {
		t.Fatalf("the LOAD opened mission %d", f.liveMission)
	}
	var ids []string
	for _, m := range mw.mission.party {
		ids = append(ids, m.ID)
	}
	k := slices.IndexFunc(mw.mission.party, func(m mapload.PartyMember) bool { return m.ID == "npc:22" })
	if k < 0 || k >= len(mw.mission.ids) || mw.mission.party[k].CompanionNPC != 22 {
		t.Fatalf("the mission's party is %v, want the companion npc:22", ids)
	}
	if got := carriedIdentities(f.NextParty(), f.Table); !slices.Equal(got, []int{21, 22}) {
		t.Fatalf("the campaign's party carries identities %v, want 21 and 22", got)
	}
	if err := mw.world.HeadlessKill(mw.mission.ids[k]); err != nil {
		t.Fatal(err)
	}
	missionSteps(mw, 4)
	if mw.mission.outcome != sim.OutcomeLost || mw.mission.lossReason != 3 {
		t.Fatalf("his fall: outcome %v with reason %d, want lost with reason 3", mw.mission.outcome, mw.mission.lossReason)
	}
}

// The victory of that mission carries the companion home as identity 22.
func TestSecondMissionFileVictoryCarriesTheCompanionHome(t *testing.T) {
	f, app := identityMissionLoad(t)
	secondContinuedWin(t, f, app)
	if err := app.HeadlessActivate("notice"); err != nil {
		t.Fatal(err)
	}
	if f.live != nil {
		t.Fatal("the victory left the party in the mission")
	}
	if got := carriedIdentities(f.Carried, f.Table); !slices.Equal(got, []int{21, 22}) {
		t.Fatalf("the party home carries identities %v, want 21 and 22", got)
	}
}
