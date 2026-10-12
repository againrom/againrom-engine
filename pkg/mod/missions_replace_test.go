package mod

import (
	"strings"
	"testing"
)

const replacingMission = `[[mission]]
kind     = "replace"
key      = "ambush"
number   = 20
map      = "mission/{lang}/map.alm"
title    = "t"
`

func TestParseMissionsReadsAReplacingMission(t *testing.T) {
	src := replacingMission + strings.NewReplacer(`"ambush"`, `"second"`, "= 20", "= 10").Replace(replacingMission)
	d, err := ParseMissions("m", MissionsFile, []byte(src), missionText, "ru")
	if err != nil {
		t.Fatal(err)
	}
	if len(d.Missions) != 2 {
		t.Fatalf("%+v", d)
	}
	m := d.Missions[0]
	if m.Kind != MissionReplace || m.Number != 20 || m.Map != "mission/ru/map.alm" || m.Title != "Title" || m.Carry != "" || m.After != 0 {
		t.Fatalf("%+v", m)
	}
	if got, ok := d.Replacing(20); !ok || got.Key != "ambush" {
		t.Fatalf("Replacing(20) = %+v %v", got, ok)
	}
	// A replacing mission is no guest: neither guest lookup finds it.
	if _, ok := d.ByNumber(20); ok {
		t.Fatal("ByNumber found a replacing mission")
	}
	if _, ok := d.AfterWin(0); ok {
		t.Fatal("AfterWin found a replacing mission")
	}
	// Loss control: the guest declaration is found by the guest lookups and
	// not by Replacing.
	g, err := ParseMissions("m", MissionsFile, []byte(goodMission), missionText, "en")
	if err != nil {
		t.Fatal(err)
	}
	if g.Missions[0].Kind != MissionGuest {
		t.Fatalf("a declaration without kind is %q", g.Missions[0].Kind)
	}
	if _, ok := g.ByNumber(25); !ok {
		t.Fatal("ByNumber lost the guest")
	}
	if _, ok := g.Replacing(25); ok {
		t.Fatal("Replacing found a guest")
	}
}

func TestParseMissionsReplacingRefusals(t *testing.T) {
	cases := []struct{ name, src, want string }{
		{"unknown kind", strings.Replace(replacingMission, `"replace"`, `"swap"`, 1), `data/missions.toml:2: kind is "swap"; use "guest" or "replace"`},
		{"after", replacingMission + "after = 10\n", "data/missions.toml:7: after belongs to a guest mission"},
		{"guest", replacingMission + "guest = 22\n", "data/missions.toml:7: guest belongs to a guest mission"},
		{"gold", replacingMission + "gold = 5\n", "data/missions.toml:7: gold belongs to a guest mission"},
		{"no map", strings.Replace(replacingMission, "map      = \"mission/{lang}/map.alm\"\n", "", 1), "data/missions.toml:1: [[mission]] has no map"},
		{"one number twice", replacingMission + strings.Replace(replacingMission, `"ambush"`, `"b"`, 1), "data/missions.toml:7: mission number 20 is already used at line 1"},
		// Loss control: the same table without kind is a guest and needs its keys.
		{"guest without after", strings.Replace(replacingMission, "kind     = \"replace\"\n", "", 1), "data/missions.toml:1: [[mission]] has no after"},
	}
	for _, c := range cases {
		_, err := ParseMissions("m", MissionsFile, []byte(c.src), missionText, "en")
		if err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%s: %v, want %q", c.name, err, c.want)
		}
	}
}
