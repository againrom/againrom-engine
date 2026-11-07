package game

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"

	"againrom/pkg/mapload"
	"againrom/pkg/mod"
)

func partyJoinRecord(t *testing.T) string {
	t.Helper()
	var b strings.Builder
	line := func(format string, args ...any) { fmt.Fprintf(&b, format+"\n", args...) }
	state := func(key string, f *FrontEnd, second bool) {
		t.Helper()
		members, err := recordFirstEntry(f)
		if second {
			members, err = recordSecondEntry(f.Carried, f.Town, f.Table)
		}
		if err != nil {
			t.Fatal(err)
		}
		line("%s members %d", key, len(members))
		for i, p := range members {
			raw, err := json.Marshal(p)
			if err != nil {
				t.Fatal(err)
			}
			word, known := mapload.PartyIdentity(p, f.Table)
			line("%s member %d identity %d/%v failure %d %s", key, i, word, known, partyFailureReason(p, f.Table), raw)
		}
		raw, err := exportTownRecord(f)
		if err != nil {
			t.Fatalf("%s SAV: %v", key, err)
		}
		line("%s SAV %d %x", key, len(raw), sha256.Sum256(raw))
	}
	first := func(held bool) *FrontEnd {
		f := shellFrontEnd()
		f.Table = companionTable(t)
		c := Campaign{Main: []int{30}, Chapters: map[int]Chapter{30: {Mission: 30, AddHero: []int{22}, Inn: []int{30}, InnNPC: []int{22}}}}
		if !held && os.Getenv("AGAINROM_PLANT_PARTY_RECORD") == "1" {
			ch := c.Chapters[30]
			ch.AddHero = []int{23}
			c.Chapters[30] = ch
		}
		f.Campaign = resolved(c, nil)
		f.Town = NewTown(c)
		f.Town.Arrive()
		f.Carried[0].StartingHero = true
		f.Carried = mapload.OwnParty(f.Carried)
		if held {
			if err := f.SetModCompanions(mod.CompanionData{Joins: []mod.CompanionJoin{joinOf(30, 22, 22)}}); err != nil {
				t.Fatal(err)
			}
		}
		return f
	}
	f := first(false)
	state("rom1 before chapter", f, false)
	f.addChapterCompanions(30)
	state("rom1 chapter", f, false)
	f.addChapterCompanions(30)
	state("rom1 repeated chapter", f, false)
	f = first(true)
	f.addChapterCompanions(30)
	state("rom1 held chapter", f, false)
	line("rom1 wrong talk %v", f.CampaignSession.joinAtTalk(f.townInstall(), 5))
	state("rom1 wrong talk", f, false)
	line("rom1 mod talk %v", f.CampaignSession.joinAtTalk(f.townInstall(), 22))
	state("rom1 mod talk", f, false)
	line("rom1 repeated talk %v", f.CampaignSession.joinAtTalk(f.townInstall(), 22))
	state("rom1 repeated talk", f, false)

	g, _, _ := secondCampaignFixtureFiles(t, false, identityNames())
	g.Table = identityTable()
	g.Carried = []mapload.PartyMember{identityHero()}
	g.Town.second = innAt(2, 30, nil)
	state("rom2 before talk", g, true)
	g.addTalkIdentities(g.townInstall(), 5, 30)
	g.addTalkIdentities(g.townInstall(), 22, 31)
	state("rom2 wrong talk", g, true)
	g.addTalkIdentities(g.townInstall(), 22, 30)
	state("rom2 talk", g, true)
	g.addTalkIdentities(g.townInstall(), 22, 30)
	state("rom2 repeated talk", g, true)
	g.leftLocation(g.townInstall())
	state("rom2 empty leave", g, true)
	g.Town.second.bank[773] = 23
	g.leftLocation(g.townInstall())
	state("rom2 leave", g, true)
	g.leftLocation(g.townInstall())
	state("rom2 repeated leave", g, true)
	return b.String()
}

func TestPartyJoinRecord(t *testing.T) {
	got := partyJoinRecord(t)
	const path = "testdata/party_join_record.txt"
	if os.Getenv("AGAINROM_WRITE_PARTY_RECORD") == "1" {
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
				t.Fatalf("party record line %d changed:\n got %s\nwant %s", i+1, g, w)
			}
		}
	}
}
