package game

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/mapload"
)

// hireRecord writes, for each game's hire over synthetic fixtures, the offer
// list, the party and the purse or bank words after a hire, a dismissal and
// mission entry, the members the hire producer builds and a digest of the
// bytes the SAV producer writes for each of those states.
func hireRecord(t *testing.T) string {
	t.Helper()
	var b strings.Builder
	line := func(format string, args ...any) { fmt.Fprintf(&b, format+"\n", args...) }
	party := func(key string, members []mapload.PartyMember) {
		line("%s party %d", key, len(members))
		for i, m := range members {
			raw, err := json.Marshal(m)
			if err != nil {
				t.Fatal(err)
			}
			line("%s member %d %s", key, i, raw)
		}
	}
	digest := func(key string, raw []byte, err error) {
		if err != nil {
			line("%s save refused: %v", key, err)
			return
		}
		line("%s save %d %x", key, len(raw), sha256.Sum256(raw))
	}

	// The first game: the tavern's whole-squad hire over the shell fixture.
	f := shellFrontEnd()
	f.Carried[0].StartingHero = true
	s := f.townUI
	firstOffers := func(key string) {
		for _, o := range s.tavernMercenaries() {
			line("%s offer type %d count %d capacity %d price %d hired %v affordable %v", key, o.Type, o.Count, o.Capacity, o.Price, o.Hired, o.Affordable)
		}
		for _, o := range s.tavernVisibleOffers() {
			raw, _ := json.Marshal(o)
			line("%s talk %s", key, raw)
		}
		for _, c := range s.tavernSurfaceCells() {
			line("%s cell %q %q %q enabled %v hired %v talk %v", key, c.Semantic, c.Detail, c.Price, c.Enabled, c.Hired, c.TalkOnly)
		}
	}
	firstTown := func(key string) {
		tw := f.Town
		line("%s gold %d hired %v pool %d capacity %d enabled %v", key, tw.Gold(), tw.MercenaryHired(3), tw.MercenaryPool(3), tw.MercenaryCapacity(3), tw.MercenaryEnabled(3))
		party(key, f.Carried)
		snap, label, err := f.Snapshot(false)
		if err != nil {
			line("%s snapshot refused: %v", key, err)
			return
		}
		raw, err := f.ExportCurrentSave(snap, label)
		digest(key, raw, err)
	}
	firstOffers("rom1 entry")
	firstTown("rom1 entry")
	msg, ok := s.toggleMercenary(3)
	line("rom1 hire %q %v", msg, ok)
	firstOffers("rom1 hired")
	firstTown("rom1 hired")
	entered, err := recordFirstEntry(f)
	line("rom1 mission entry err %v", err)
	party("rom1 mission entry", entered)
	msg, ok = s.toggleMercenary(3)
	line("rom1 dismiss %q %v", msg, ok)
	firstOffers("rom1 dismissed")
	firstTown("rom1 dismissed")
	f.Town.gold = 49
	msg, ok = s.toggleMercenary(3)
	line("rom1 short hire %q %v gold %d", msg, ok, f.Town.Gold())
	f.Town.gold = 1000
	for _, count := range []int{1, 2} {
		members, ok := recordSquad(f.Table, f.Town, 3, count)
		line("rom1 squad %d ok %v", count, ok)
		party(fmt.Sprintf("rom1 squad %d", count), members)
	}
	f.Town.mercHired[3], f.Town.mercPool[3] = true, 2
	if err := f.CampaignSession.restoreHiredMercenaries(f.Table); err != nil {
		line("rom1 restore err %v", err)
	}
	party("rom1 restore", f.Carried)

	// The first game's siege rows, through the hire producer.
	units := dbCollection{{}}
	for i, name := range []string{"Catapult", "Ballista"} {
		p := make([]int32, 56)
		for j := range p {
			p[j] = -1
		}
		p[0], p[4], p[29], p[30], p[31], p[32], p[55] = 20, 100, int32(40+i), 1, 1, 1, 0
		units = append(units, dbEntry{name: name, params: p})
	}
	siegeTable := &mapload.Table{Humans: f.Table.Humans, Units: units}
	for _, typ := range []int{1, 2} {
		members, ok := recordSiege(siegeTable, typ, 2)
		line("rom1 siege %d ok %v", typ, ok)
		party(fmt.Sprintf("rom1 siege %d", typ), members)
		member, ok := recordSiegeMember(units, typ)
		line("rom1 siege restore %d ok %v", typ, ok)
		party(fmt.Sprintf("rom1 siege restore %d", typ), []mapload.PartyMember{member})
	}

	// The second game: the inn's hire seats over the inn fixture bank.
	g, _, screen := secondCampaignFixtureFiles(t, false)
	g.Table = keyedHireTable(t)
	g.Carried = []mapload.PartyMember{{ID: "hero", Name: "Aldor", PlayerCharacter: true, StartingHero: true}}
	c := innAt(2, 30, map[int]int32{532: 1, 552: 2, 533: 2, 553: 2, 513: 1, 534: 1, 554: 2})
	g.Town.second, g.Town.gold = c, 500
	g.Town.settleCityGroups(g.Carried)
	c.enterInn()
	secondState := func(key string) {
		for _, o := range c.innOptions() {
			line("%s option %d %d %d %v", key, o.kind, o.topic, o.npc, o.high)
		}
		seats, hires := c.innSeats()
		for _, o := range seats {
			line("%s seat %d %d %d %v hired %v", key, o.kind, o.topic, o.npc, o.high, c.hired(o.npc))
		}
		line("%s hires %d rows %v", key, hires, c.innRows("Sleep"))
		var words []string
		for i := 500; i < 580; i++ {
			if c.bank[i] != 0 {
				words = append(words, fmt.Sprintf("%d=%d", i, c.bank[i]))
			}
		}
		line("%s bank %s stage %d gold %d", key, strings.Join(words, " "), c.bank[768], g.Town.Gold())
		for _, npc := range []int{1, 2, 3, 22} {
			member, ok := screen.campaignActor(npc)
			raw, _ := json.Marshal(member)
			line("%s actor %d %v %s", key, npc, ok, raw)
		}
		party(key, g.Carried)
		raw, err := exportTownRecord(g)
		digest(key, raw, err)
	}
	secondState("rom2 entry")
	for _, npc := range []int{1, 3, 2, 2, 1, 3, 9} {
		line("rom2 hire or fire %d %q", npc, recordHireOrFire(c, npc))
		secondState(fmt.Sprintf("rom2 after %d", npc))
	}
	line("rom2 hire or fire 1 %q", recordHireOrFire(c, 1))
	entered, err = recordSecondEntry(g.Carried, g.Town, g.Table)
	line("rom2 mission entry err %v", err)
	party("rom2 mission entry", entered)
	again, err := recordSecondEntry(append(entered, entered[len(entered)-1]), g.Town, g.Table)
	line("rom2 mission entry again err %v", err)
	party("rom2 mission entry again", again)
	c.talkTo(innOffer{kind: 0, npc: 5})
	c.talkTo(innOffer{kind: 3, npc: 22, topic: 30})
	secondState("rom2 talks")
	win := c.bank
	win[512], win[513], win[535] = 1, 1, 2
	line("rom2 complete output %d", c.completeBank(win))
	secondState("rom2 completed")

	// The second game's mission entry and its mission SAV.
	m := cellStateFront(t)
	m.Table = keyedHireTable(t)
	town := &Town{second: newSecondCampaign()}
	town.second.bank[768] = 70
	for id := 1; id <= 3; id++ {
		town.second.bank[511+id], town.second.bank[531+id] = 1, 1
	}
	town.second.bank[536] = 1
	missionParty, err := recordSecondEntry(nil, town, m.Table)
	line("rom2 mission party err %v", err)
	party("rom2 mission party", missionParty)
	app := m.App("hire record")
	app.Layout(1024, 768)
	if err := app.OpenMission(m.MissionOpenerWith(10, missionParty)); err != nil {
		line("rom2 mission open err %v", err)
	} else {
		snap, label, err := m.Snapshot(true)
		if err != nil {
			line("rom2 mission snapshot refused: %v", err)
		} else {
			raw, _, err := m.playerMissionSave(snap, label)
			digest("rom2 mission", raw, err)
		}
	}
	return b.String()
}

// keyedHireTable is the inn fixture's actor table with the keyed Humans and
// Units rows of hires 1..7 at stage 70.
func keyedHireTable(t *testing.T) *mapload.Table {
	t.Helper()
	f := cellStateFront(t)
	humans, units := f.Table.Humans.(dbCollection), f.Table.Units.(dbCollection)
	keys := []string{"2MercenaryXBowman70", "2MercenaryKnight70", "MercenaryDruid70", "MercenaryTroll70", "MercenaryOrc70", "2MercenaryMage70", "CatapultBase"}
	types := []int32{15, 7, 100, 102, 101, 24, 30}
	for i, key := range keys {
		keyID := int32(10000 + 10*(i+1))
		if i == 0 || i == 1 || i == 5 {
			p := mercenaryHumanRow(types[i], 1, 0, 10)
			p[24] = keyID
			humans = append(humans, dbEntry{name: key, params: p})
			continue
		}
		p := make([]int32, 56)
		for j := range p {
			p[j] = -1
		}
		p[0], p[4], p[29], p[30], p[31], p[32], p[55] = 20, 100, types[i], 1, 1, 1, keyID
		units = append(units, dbEntry{name: key, params: p})
	}
	f.Table.Humans, f.Table.Units, f.Table.UnitKeys = humans, units, mapload.ServerUnitKeys
	return f.Table
}

// exportTownRecord is exportTown answering its refusal.
func exportTownRecord(f *FrontEnd) ([]byte, error) {
	s, label, err := f.Snapshot(false)
	if err != nil {
		return nil, err
	}
	return f.ExportCurrentSave(s, label)
}

func recordFirstEntry(f *FrontEnd) ([]mapload.PartyMember, error) {
	return enterHires(firstCampaignRules{}, f.Carried, f.Town, f.Table)
}

func recordSecondEntry(party []mapload.PartyMember, town *Town, table *mapload.Table) ([]mapload.PartyMember, error) {
	return enterHires(secondCampaignRules{}, party, town, table)
}

func recordHireOrFire(c *secondCampaign, npc int) string { return c.seatHire(npc) }

func recordSquad(table *mapload.Table, town *Town, typ, count int) ([]mapload.PartyMember, bool) {
	return hireMembers(&tavernHires, hireState{town: town, table: table}, typ, count)
}

func recordSiege(table *mapload.Table, typ, count int) ([]mapload.PartyMember, bool) {
	return hireMembers(&tavernHires, hireState{table: table}, typ, count)
}

func recordSiegeMember(units data.Collection, typ int) (mapload.PartyMember, bool) {
	return siegeHireMember(units, typ)
}

func TestHireRecord(t *testing.T) {
	got := hireRecord(t)
	path := filepath.Join("testdata", "hire_record.txt")
	if os.Getenv("AGAINROM_WRITE_HIRE_RECORD") == "1" {
		if err := os.WriteFile(path, []byte(got), 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(want) != got {
		wl, gl := strings.Split(string(want), "\n"), strings.Split(got, "\n")
		for i := 0; i < len(wl) || i < len(gl); i++ {
			var w, g string
			if i < len(wl) {
				w = wl[i]
			}
			if i < len(gl) {
				g = gl[i]
			}
			if w != g {
				t.Fatalf("hire record line %d changed:\n got %s\nwant %s", i+1, g, w)
			}
		}
	}
}
