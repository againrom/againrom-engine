package mod

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const goodMission = `# one mission
[[mission]]
key      = "prologue"
number   = 25
map      = "mission/{lang}/map.alm"
texts    = "mission/{lang}/text"
title    = "t"
briefing = "b"
after    = 20
guest    = 22
chapter  = 30
gold     = 50
carry    = "join"
`

func missionText(key string) (string, bool) {
	v, ok := map[string]string{"t": "Title", "b": "Briefing"}[key]
	return v, ok
}

func TestParseMissionsReadsAMission(t *testing.T) {
	d, err := ParseMissions("m", MissionsFile, []byte(goodMission), missionText, "ru")
	if err != nil {
		t.Fatal(err)
	}
	if len(d.Missions) != 1 || d.Empty() {
		t.Fatalf("%+v", d)
	}
	m := d.Missions[0]
	if m.Mod != "m" || m.Line != 2 || m.Key != "prologue" || m.Number != 25 || m.Map != "mission/ru/map.alm" ||
		m.Texts != "mission/ru/text" || m.Title != "Title" || m.Brief != "Briefing" || m.After != 20 ||
		m.Guest != 22 || m.Chapter != 30 || m.Gold != 50 || m.Carry != CarryJoin {
		t.Fatalf("%+v", m)
	}
	if got, ok := d.AfterWin(20); !ok || got.Key != "prologue" {
		t.Fatalf("AfterWin: %+v %v", got, ok)
	}
	if _, ok := d.AfterWin(10); ok {
		t.Fatal("AfterWin answered for a mission no declaration names")
	}
	if got, ok := d.Find("m", "prologue"); !ok || got.Number != 25 {
		t.Fatalf("Find: %+v %v", got, ok)
	}
	if _, ok := d.Find("other", "prologue"); ok {
		t.Fatal("Find answered for another mod")
	}
}

func TestParseMissionsRefusalsNameFileAndLine(t *testing.T) {
	cases := []struct{ name, src, want string }{
		{"outside", "key = \"a\"\n", "data/missions.toml:1: key is outside a [[mission]] table"},
		{"unknown table", "[[join]]\n", `data/missions.toml:1: unknown table "join"`},
		{"unknown key", strings.Replace(goodMission, "gold     = 50", "gold     = 50\nwhen = 1", 1), `data/missions.toml:13: unknown key "when"`},
		{"zero number", strings.Replace(goodMission, "number   = 25", "number   = 0", 1), "data/missions.toml:4: number is 0; use 1 to 65535"},
		{"gold", strings.Replace(goodMission, "gold     = 50", "gold     = 2000000", 1), "data/missions.toml:12: gold is 2000000; use 0 to 1000000"},
		{"outside path", strings.Replace(goodMission, `"mission/{lang}/map.alm"`, `"../map.alm"`, 1), `data/missions.toml:5: map "../map.alm" is not a path inside the mod folder`},
		{"absolute path", strings.Replace(goodMission, `"mission/{lang}/map.alm"`, `"/map.alm"`, 1), `data/missions.toml:5: map "/map.alm" is not a path inside the mod folder`},
		{"text key", strings.Replace(goodMission, `title    = "t"`, `title    = "x"`, 1), `data/missions.toml:7: title names text key "x"`},
		{"carry", strings.Replace(goodMission, `"join"`, `"gold"`, 1), `data/missions.toml:13: carry is "gold"; only "join" is supported`},
		{"missing guest", strings.Replace(goodMission, "guest    = 22\n", "", 1), "data/missions.toml:2: [[mission]] has no guest"},
		{"after itself", strings.Replace(goodMission, "after    = 20", "after    = 25", 1), "data/missions.toml:2: mission 25 cannot play after itself"},
		{"duplicate key", goodMission + strings.Replace(strings.Replace(goodMission, "25", "26", 1), "20", "21", 1), `data/missions.toml:15: mission key "prologue" is already used at line 2`},
		{"duplicate number", goodMission + strings.Replace(strings.Replace(goodMission, "prologue", "b", 1), "20", "21", 1), "data/missions.toml:15: mission number 25 is already used at line 2"},
		{"duplicate after", goodMission + strings.Replace(strings.Replace(goodMission, "prologue", "b", 1), "25", "26", 1), "data/missions.toml:15: a mission already plays after mission 20 at line 2"},
	}
	for _, c := range cases {
		_, err := ParseMissions("m", MissionsFile, []byte(c.src), missionText, "en")
		if err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%s: %v, want %q", c.name, err, c.want)
		}
	}
}

func TestMissionReadsItsFilesFromTheModFolder(t *testing.T) {
	dir := t.TempDir()
	write := func(rel, s string) {
		p := filepath.Join(dir, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(s), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	m := Mission{File: MissionsFile, Line: 2, Map: "mission/en/map.alm", Texts: "mission/en/text"}
	if err := m.ReadFiles(dir); err == nil || !strings.Contains(err.Error(), "data/missions.toml:2: map: mission/en/map.alm: no such file in the mod folder") {
		t.Fatalf("missing map: %v", err)
	}
	write("mission/en/map.alm", "MAP")
	if err := m.ReadFiles(dir); err == nil || !strings.Contains(err.Error(), "no such folder") {
		t.Fatalf("missing texts: %v", err)
	}
	write("mission/en/text/event01.txt", "E1")
	write("mission/en/text/tips01.txt", "T1")
	if err := m.ReadFiles(dir); err != nil {
		t.Fatal(err)
	}
	if string(m.MapBytes) != "MAP" || len(m.TextFiles) != 2 || string(m.TextFiles["event01.txt"]) != "E1" {
		t.Fatalf("%q %v", m.MapBytes, m.TextFiles)
	}
	write("mission/en/text/Event02.TXT", "x")
	if err := m.ReadFiles(dir); err == nil || !strings.Contains(err.Error(), "is not a mission text file name") {
		t.Fatalf("bad name: %v", err)
	}
}
