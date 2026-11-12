package mod

import (
	"encoding/json"
	"strings"
	"testing"
)

const flowMissions = `[[mission]]
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
party = "main"
`

func TestMissionFlowParsesWritesAndReports(t *testing.T) {
	d, err := ParseMissions("m", MissionsFile, []byte(flowMissions), missionText, "en")
	if err != nil {
		t.Fatal(err)
	}
	if err := d.ValidateCampaign([]int{10, 20, 30}, nil, nil, map[int]int{10: 20}); err != nil {
		t.Fatal(err)
	}
	if len(d.Missions) != 2 || *d.Missions[0].AutoGetMission != 11 || *d.Missions[1].Payment != 75 || !d.Missions[1].MainParty {
		t.Fatalf("%+v", d)
	}
	if got := d.Replacements(); len(got) != 2 || !strings.Contains(got[0], "mission 10 AutoGetMission = 11") || !strings.Contains(got[1], "data/missions.toml:11") {
		t.Fatal(got)
	}
	b, err := WriteMissions(nil, nil, d.Missions)
	if err != nil {
		t.Fatal(err)
	}
	back, err := ParseMissions("m", MissionsFile, b, missionText, "en")
	if err != nil || len(back.Missions) != 2 || *back.Missions[1].AutoGetMission != 20 || *back.Missions[1].Payment != 75 {
		t.Fatalf("%s: %+v %v", b, back, err)
	}
	for _, party := range []string{`guest = 22
chapter = 30`, `party = [{ character = "npc22" }]`, `party = "main"`} {
		_, err := ParseMissions("m", MissionsFile, []byte("[[mission]]\nkind = \"campaign\"\nkey = \"guest\"\nnumber = 20\n"+party), missionText, "en")
		if err != nil {
			t.Fatal(party, err)
		}
	}
}

func TestMissionFlowRefusalsNameFileAndLine(t *testing.T) {
	for _, tc := range []struct{ src, want string }{
		{strings.Replace(flowMissions, "number = 10", "number = 12", 1), "data/missions.toml:1: unknown mission 12"},
		{strings.Replace(flowMissions, "AutoGetMission = 11", "AutoGetMission = 99", 1), "data/missions.toml:5: AutoGetMission names unknown mission 99"},
		{strings.Replace(flowMissions, "AutoGetMission = 20", "AutoGetMission = 10", 1), "data/missions.toml:5: AutoGetMission cycle"},
		{"[[mission]]\nkind = \"campaign\"\nkey = \"back\"\nnumber = 30\nAutoGetMission = 20\n", "data/missions.toml:5: AutoGetMission 20 is not above current main mission 30"},
	} {
		d, err := ParseMissions("m", MissionsFile, []byte(tc.src), missionText, "en")
		if err == nil {
			err = d.ValidateCampaign([]int{10, 20, 30}, []int{1}, nil, nil)
		}
		if err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Errorf("%v, want %s", err, tc.want)
		}
	}
}

func TestMissionFlowUsesTheMissionLoadOrder(t *testing.T) {
	a, err := ParseMissions("a", MissionsFile, []byte(flowMissions), missionText, "en")
	if err != nil {
		t.Fatal(err)
	}
	b, err := ParseMissions("b", MissionsFile, []byte(strings.Replace(flowMissions, "AutoGetMission = 11", "AutoGetMission = 30", 1)), missionText, "en")
	if err != nil {
		t.Fatal(err)
	}
	d := MissionData{Missions: append(a.Missions, b.Missions[0])}
	if _, _, err := MergeMissions(d); err == nil || !strings.Contains(err.Error(), "data/missions.toml:1") || !strings.Contains(err.Error(), "neither loads after the other") {
		t.Fatal(err)
	}
	d.Order = Precedence{"b": {"a"}}
	merged, overridden, err := MergeMissions(d)
	if err != nil || len(overridden) != 1 || *merged.Missions[0].AutoGetMission != 30 {
		t.Fatalf("%+v %+v %v", merged, overridden, err)
	}
}

func TestMissionFlowRefusesInstalledSideTargets(t *testing.T) {
	for _, source := range []string{
		"[[mission]]\nkind = \"campaign\"\nkey = \"first\"\nnumber = 20\nAutoGetMission = 31\n",
		"[[mission]]\nkind = \"campaign\"\nkey = \"first\"\nnumber = 20\nAutoGetMission = 31\n[[mission]]\nkind = \"campaign\"\nkey = \"side\"\nnumber = 31\nPayment = 5\n",
	} {
		d, err := ParseMissions("m", MissionsFile, []byte(source), nil, "en")
		if err == nil {
			err = d.ValidateCampaign([]int{10, 20, 30, 40}, []int{31}, []int{30, 31, 40}, nil)
		}
		if err == nil || !strings.Contains(err.Error(), "data/missions.toml:5:") || !strings.Contains(err.Error(), "installed side mission 31") {
			t.Fatalf("load = %v, want source line and installed side target", err)
		}
	}
}

func TestMissionFlowReportsEveryCampaignRow(t *testing.T) {
	src := "[[mission]]\nkind = \"campaign\"\nkey = \"payment\"\nnumber = 20\nPayment = 999\n" +
		"[[mission]]\nkind = \"campaign\"\nkey = \"guest\"\nnumber = 30\nguest = 22\nchapter = 40\n" +
		"[[mission]]\nkind = \"campaign\"\nkey = \"party\"\nnumber = 40\nparty = [{ character = \"npc22\" }]\n"
	d, err := ParseMissions("m", MissionsFile, []byte(src), nil, "en")
	if err != nil {
		t.Fatal(err)
	}
	off := 50
	d.Missions = append(d.Missions, Mission{Kind: MissionCampaign, Number: 50, Off: true, AutoGetMission: &off})
	got := d.Replacements()
	want := []string{
		"mission 20 Payment = 999, party = main by mod \"m\", declared at data/missions.toml:1",
		"mission 30 guest = 22, chapter = 40 by mod \"m\", declared at data/missions.toml:6",
		"mission 40 party = [\"npc22\"] by mod \"m\", declared at data/missions.toml:12",
	}
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Fatalf("check:\n%s\nwant:\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
}

func TestMissionFlowWriterKeepsCampaignFields(t *testing.T) {
	src := flowMissions +
		"[[mission]]\nkind = \"campaign\"\nkey = \"guest\"\nnumber = 20\nguest = 22\nchapter = 30\n" +
		"[[mission]]\nkind = \"campaign\"\nkey = \"party\"\nnumber = 30\nparty = [{ character = \"npc22\", level = 2 }]\nPayment = 0\n" +
		"[[mission]]\nkind = \"replace\"\nkey = \"last\"\nnumber = 40\nmap = \"last.alm\"\nAutoGetMission = -1\n"
	normalize := func(d MissionData) string {
		for i := range d.Missions {
			m := &d.Missions[i]
			m.Line, m.AutoLine, m.PartyLine = 0, 0, 0
		}
		b, err := json.Marshal(d.Missions)
		if err != nil {
			t.Fatal(err)
		}
		return string(b)
	}
	before, err := ParseMissions("m", MissionsFile, []byte(src), nil, "en")
	if err != nil {
		t.Fatal(err)
	}
	for _, source := range [][]byte{nil, []byte(src)} {
		after := make([]Mission, len(before.Missions))
		for i, m := range before.Missions {
			after[i] = CloneMission(m)
		}
		payment := 5
		after[1].Payment = &payment
		out, err := WriteMissions(source, before.Missions, after)
		if source == nil {
			out, err = WriteMissions(nil, nil, after)
		}
		if err != nil {
			t.Fatal(err)
		}
		back, err := ParseMissions("m", MissionsFile, out, nil, "en")
		if err != nil || normalize(back) != normalize(MissionData{Missions: after}) {
			t.Fatalf("campaign fields changed in write/read: %v\n%s", err, out)
		}
	}
}
