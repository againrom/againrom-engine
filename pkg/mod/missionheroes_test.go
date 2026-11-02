package mod

import (
	"strconv"
	"strings"
	"testing"

	"againrom/pkg/herokind"
)

// heroMission is a guest mission after mission 20 for the heroes the extra
// lines name.
func heroMission(key string, number int, extra string) string {
	return "[[mission]]\nkey = \"" + key + "\"\nnumber = " + strconv.Itoa(number) +
		"\nmap = \"" + key + ".alm\"\nafter = 20\nguest = 22\nchapter = 30\n" + extra
}

func TestParseMissionsChoosesTheMissionAfterAWinByTheStartingHero(t *testing.T) {
	src := heroMission("fighter", 25, "class = \"fighter\"\n") +
		heroMission("mage_m", 27, "class = \"mage\"\nsex = \"male\"\n") +
		heroMission("mage_f", 28, "sex = \"female\"\nclass = \"mage\"\n")
	d, err := ParseMissions("m", MissionsFile, []byte(src), missionText, "en")
	if err != nil {
		t.Fatal(err)
	}
	for k, want := range map[herokind.Kind]string{
		herokind.MaleFighter: "fighter", herokind.FemaleFighter: "fighter",
		herokind.MaleMage: "mage_m", herokind.FemaleMage: "mage_f",
	} {
		if got, ok := d.AfterWin(20, k); !ok || got.Key != want {
			t.Errorf("%v: %q %v, want %q", k, got.Key, ok, want)
		}
	}
	if !d.Follows(20) || d.Follows(25) {
		t.Fatal("Follows")
	}
	if m := d.Missions[2]; m.Class != "mage" || m.Sex != "female" || m.Heroes() != herokind.SetOf(herokind.FemaleMage) {
		t.Fatalf("%+v", m)
	}
	merged, _, err := MergeMissions(d)
	if err != nil || len(merged.Missions) != 3 {
		t.Fatalf("merge of missions that do not overlap: %v", err)
	}
	// A subset: only a mage gets a mission after the win.
	d, err = ParseMissions("m", MissionsFile, []byte(heroMission("mage", 27, "class = \"mage\"\n")), missionText, "en")
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := d.AfterWin(20, herokind.FemaleFighter); ok {
		t.Fatal("a fighter got the mage's mission")
	}
	if got, ok := d.AfterWin(20, herokind.FemaleMage); !ok || got.Key != "mage" {
		t.Fatalf("a mage got %+v", got)
	}
}

func TestParseMissionsRefusesMissionsWhoseHeroesOverlap(t *testing.T) {
	fighter := heroMission("a", 25, "class = \"fighter\"\n")
	for name, c := range map[string]struct{ src, want string }{
		"overlap": {fighter + heroMission("b", 27, "sex = \"female\"\n"),
			"data/missions.toml:9: a mission already plays after mission 20 for a female fighter at line 1"},
		"every hero": {fighter + heroMission("b", 27, ""),
			"data/missions.toml:9: a mission already plays after mission 20 for a fighter at line 1"},
		"same": {fighter + heroMission("b", 27, "class = \"fighter\"\n"),
			"data/missions.toml:9: a mission already plays after mission 20 for a fighter at line 1"},
		"class word":   {heroMission("a", 25, "class = \"Mage\"\n"), `data/missions.toml:8: class "Mage" is neither "fighter" nor "mage"`},
		"sex word":     {heroMission("a", 25, "sex = \"woman\"\n"), `data/missions.toml:8: sex "woman" is neither "male" nor "female"`},
		"not a string": {heroMission("a", 25, "class = 1\n"), "data/missions.toml:8: class must be a string"},
		"replace": {"[[mission]]\nkind = \"replace\"\nkey = \"r\"\nnumber = 20\nmap = \"r.alm\"\nclass = \"mage\"\n",
			"data/missions.toml:6: class belongs to a guest mission; a replacing mission plays for every starting hero"},
		"off": {"[[mission]]\nnumber = 25\noff = true\nsex = \"male\"\n", "switches mission 25 off and sets sex"},
	} {
		if _, err := ParseMissions("m", MissionsFile, []byte(c.src), missionText, "en"); err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%s: %v, want %q", name, err, c.want)
		}
	}
}

func TestMergeMissionsRefusesTwoModsWhoseHeroesOverlapOnOneWin(t *testing.T) {
	read := func(id, src string) []Mission {
		d, err := ParseMissions(id, MissionsFile, []byte(src), missionText, "en")
		if err != nil {
			t.Fatal(err)
		}
		return d.Missions
	}
	first := read("m1", heroMission("a", 25, "class = \"mage\"\n"))
	over := read("m2", heroMission("b", 27, "sex = \"male\"\n"))
	_, _, err := MergeMissions(MissionData{Missions: append(append([]Mission{}, first...), over...)})
	want := `mod "m2": data/missions.toml:1: mod "m1" already plays a mission after mission 20 for a male mage (mission 25, data/missions.toml:1); one mission follows one win`
	if err == nil || !strings.Contains(err.Error(), want) {
		t.Fatalf("%v, want %q", err, want)
	}
	apart := read("m2", heroMission("b", 27, "class = \"fighter\"\n"))
	merged, _, err := MergeMissions(MissionData{Missions: append(append([]Mission{}, first...), apart...)})
	if err != nil {
		t.Fatal(err)
	}
	if got, ok := merged.AfterWin(20, herokind.MaleFighter); !ok || got.Mod != "m2" {
		t.Fatalf("the fighter got %+v", got)
	}
	if got, ok := merged.AfterWin(20, herokind.FemaleMage); !ok || got.Mod != "m1" {
		t.Fatalf("the mage got %+v", got)
	}
}

func TestWriteMissionsKeepsTheHeroKeys(t *testing.T) {
	src := heroMission("a", 25, "class = \"mage\"\nsex = \"female\"\n")
	d, err := ParseMissions("m", MissionsFile, []byte(src), missionText, "en")
	if err != nil {
		t.Fatal(err)
	}
	m := CloneMission(d.Missions[0])
	m.Gold = 5
	out, err := WriteMissions([]byte(src), d.Missions, []Mission{m})
	if err != nil {
		t.Fatal(err)
	}
	back, err := ParseMissions("m", MissionsFile, out, missionText, "en")
	if err != nil || back.Missions[0].Class != "mage" || back.Missions[0].Sex != "female" || back.Missions[0].Gold != 5 {
		t.Fatalf("%s: %+v %v", out, back, err)
	}
	m.Key, m.Number = "b", 27
	out, err = WriteMissions(nil, nil, []Mission{m})
	if err != nil || !strings.Contains(string(out), "class = \"mage\"\nsex = \"female\"\n") {
		t.Fatalf("a new record: %s %v", out, err)
	}
}
