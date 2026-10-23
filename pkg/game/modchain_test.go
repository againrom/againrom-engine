package game

import (
	"encoding/json"
	"strings"
	"testing"

	"againrom/pkg/mapload"
	"againrom/pkg/mod"
)

// chainLink is a guest mission of mod id that plays after mission after, with
// a party of the characters named.
func chainLink(id, key string, line, number, after int, party ...string) mod.Mission {
	m := guestDecl()
	m.Mod, m.Key, m.Line, m.Number, m.After = id, key, line, number, after
	m.Guest, m.Chapter, m.Carry = 0, 0, ""
	m.PartyLine = line + 1
	for _, c := range party {
		m.Party = append(m.Party, mod.GuestMember{Character: c})
	}
	return m
}

func TestSetModMissionsRefusesABrokenChain(t *testing.T) {
	for name, c := range map[string]struct {
		missions []mod.Mission
		want     string
	}{
		"missing link": {[]mod.Mission{guestDecl(), chainLink("m", "next", 9, 27, 29, "npc22")},
			`mod "m": data/missions.toml:9: after 29 is not a main mission of this install's campaign, nor a guest mission of the active mods`},
		"cycle": {[]mod.Mission{chainLink("m", "a", 9, 27, 28, "npc22"), chainLink("m", "b", 12, 28, 27, "npc22")},
			`mod "m": data/missions.toml:9: mission 27 is in a cycle of guest missions (27 after 28 after 27)`},
		"unknown character": {[]mod.Mission{guestDecl(), chainLink("m", "next", 9, 27, 25, "nobody")},
			`mod "m": data/missions.toml:10: character "nobody" is not in the character catalog`},
		"two mods on one link": {[]mod.Mission{guestDecl(), chainLink("m", "next", 9, 27, 25, "npc22"), chainLink("m2", "other", 3, 28, 25, "npc22")},
			`mod "m2": data/missions.toml:3: mod "m" already plays a mission after mission 25 (mission 27, data/missions.toml:9); one mission follows one win`},
		"two who join": {[]mod.Mission{guestDecl(), func() mod.Mission {
			m := guestDecl()
			m.Key, m.Line, m.Number, m.After = "again", 9, 27, 25
			return m
		}()}, `mod "m": data/missions.toml:9: the chain of missions 25, 27 holds 2 guests who join; one guest of a chain joins the town`},
	} {
		f := guestFixtureFront(t)
		_, err := f.SetModMissions(mod.MissionData{Missions: c.missions})
		if err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%s: %v, want %q", name, err, c.want)
		}
		if !f.Table.Mods.Missions.Empty() {
			t.Errorf("%s: a refused chain was recorded", name)
		}
	}
}

// chainGuest is the session's guest in the second link of the chain 25, 27:
// companion 22 joined from link 25, a party of two in link 27.
func chainGuest() (*guestMission, mod.MissionData) {
	first := guestDecl()
	second := chainLink("m", "next", 9, 27, 25, "npc22", "sellsword")
	d := mod.MissionData{Missions: []mod.Mission{first, second}}
	member := mapload.PartyMember{ID: "npc:22", Name: "Guest", CompanionNPC: 22, PlayerCharacter: true}
	lead := member
	lead.StartingHero = true
	sword := mapload.PartyMember{ID: "char:sellsword", Name: "Sellsword", PlayerCharacter: true}
	return &guestMission{mission: second, stash: []byte("stash"), start: lead, rest: []mapload.PartyMember{sword},
		member: &member, chain: d.Chain(second), hero: &guestHero{Mage: true, FigureDir: "x"}}, d
}

func TestGuestChainEntersItsPartyAndJoinsItsGuest(t *testing.T) {
	g, _ := chainGuest()
	if len(g.links()) != 2 || g.links()[0].Number != 25 {
		t.Fatalf("links %v", g.links())
	}
	if g.returnMission(27) != 20 || g.returnMission(25) != 20 || g.returnMission(30) != 30 {
		t.Fatal("returnMission")
	}
	party, purse, ok := g.guestEntry(27)
	if !ok || len(party) != 2 || party[0].ID != "npc:22" || party[1].ID != "char:sellsword" || purse != 0 {
		t.Fatalf("guestEntry %v %d %v", party, purse, ok)
	}
	if _, ok := g.joining(30, 22); !ok {
		t.Fatal("the joiner of the first link does not join through the town's grant")
	}
}

func TestGuestChainRecordCrossesTheSessionSupplement(t *testing.T) {
	g, d := chainGuest()
	raw, err := json.Marshal(currentSessionData{Guest: g.record()})
	if err != nil {
		t.Fatal(err)
	}
	var s currentSessionData
	if err := json.Unmarshal(raw, &s); err != nil {
		t.Fatal(err)
	}
	back, err := restoreGuest(&s, mapload.ModContext{Missions: d})
	if err != nil {
		t.Fatal(err)
	}
	if !back.playing(27) || len(back.rest) != 1 || back.rest[0].ID != "char:sellsword" || back.member == nil ||
		back.member.CompanionNPC != 22 || back.hero == nil || *back.hero != *g.hero || len(back.links()) != 2 {
		t.Fatalf("restored %+v", back)
	}
	for name, edit := range map[string]func(*currentGuest){
		"party after the chain": func(r *currentGuest) { r.Stash, r.Number, r.Key = nil, 25, "prologue" },
		"member while her link runs": func(r *currentGuest) {
			r.Key, r.Number, r.Stash, r.Party, r.Hero = "prologue", 25, []byte("s"), nil, nil
		},
	} {
		var c currentSessionData
		if err := json.Unmarshal(raw, &c); err != nil {
			t.Fatal(err)
		}
		edit(c.Guest)
		if _, err := restoreGuest(&c, mapload.ModContext{Missions: d}); err == nil || !strings.Contains(err.Error(), "invalid current guest record") {
			t.Errorf("%s: %v", name, err)
		}
	}
	// A chain of one link writes the record a save of one guest mission wrote.
	one := &guestMission{mission: guestDecl(), stash: []byte("s"), start: g.start}
	b, err := json.Marshal(one.record())
	if err != nil {
		t.Fatal(err)
	}
	var keys map[string]json.RawMessage
	if err := json.Unmarshal(b, &keys); err != nil {
		t.Fatal(err)
	}
	if _, party := keys["Party"]; party || keys["Hero"] != nil {
		t.Fatalf("a one-link record carries the chain's fields: %s", b)
	}
}
