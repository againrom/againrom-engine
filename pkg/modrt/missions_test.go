package modrt

import (
	"path/filepath"
	"strings"
	"testing"

	"againrom/pkg/mod"
)

const missionsMain = "def init(game, settings):\n    game.data.add(\"data/missions.toml\")\n"

const missionSrc = "[[mission]]\nkey = \"p\"\nnumber = 25\nmap = \"m/{lang}/map.alm\"\ntexts = \"m/{lang}/text\"\ntitle = \"t\"\nafter = 20\nguest = 22\nchapter = 30\n"

func missionMod(src string) map[string]string {
	return map[string]string{"main.star": missionsMain, "data/missions.toml": src,
		"m/ru/map.alm": "RU MAP", "m/ru/text/event01.txt": "E1", "text/en/strings.toml": "\"t\" = \"Title\"\n",
		"text/ru/strings.toml": "\"t\" = \"Заголовок\"\n"}
}

func TestMissionModReadsItsMapAndTextsInTheGameLanguage(t *testing.T) {
	res, err := Load(makeMods(t, map[string]map[string]string{"m": missionMod(missionSrc)}), "rom1-ru", nil, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Missions.Missions) != 1 {
		t.Fatalf("%+v", res.Missions)
	}
	m := res.Missions.Missions[0]
	if m.Mod != "m" || m.Number != 25 || string(m.MapBytes) != "RU MAP" || string(m.TextFiles["event01.txt"]) != "E1" || m.Title != "Заголовок" {
		t.Fatalf("%+v", m)
	}
	if _, err := Load(makeMods(t, map[string]map[string]string{"m": missionMod(missionSrc)}), "rom1-en", nil, Options{}); err == nil ||
		!strings.Contains(err.Error(), `mod "m": data/missions.toml:1: map: m/en/map.alm: no such file in the mod folder`) {
		t.Fatalf("an English game read the Russian map: %v", err)
	}
}

func TestTwoModsCannotAddMissionsAfterTheSameWinOrUnderOneNumber(t *testing.T) {
	for name, c := range map[string]struct{ second, want string }{
		"number": {strings.Replace(missionSrc, "after = 20", "after = 40", 1), `data/missions.toml:1: mission 25 is added by mod "m1" and named by mod "m2", and neither loads after the other`},
		"after":  {strings.Replace(missionSrc, "number = 25", "number = 26", 1), `data/missions.toml:1: mod "m1" already plays a mission after mission 20`},
	} {
		mods := makeMods(t, map[string]map[string]string{"m1": missionMod(missionSrc), "m2": missionMod(c.second)})
		if err := mergedMissions(mods); err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%s: %v, want %q", name, err, c.want)
		}
	}
}

// mergedMissions loads mods and merges their missions, the step at which two
// mods that name one mission number or one win are refused.
func mergedMissions(mods []mod.Entry) error {
	res, err := Load(mods, "rom1-ru", nil, Options{})
	if err != nil {
		return err
	}
	_, _, err = mod.MergeMissions(res.Missions)
	return err
}

func TestReniestaFixtureNeedsItsAssembledMap(t *testing.T) {
	entries, err := mod.Resolve(filepath.Join("testdata", "mods"), []string{"reniesta"})
	if err != nil {
		t.Fatal(err)
	}
	_, err = Load(entries, "rom1-en", nil, Options{})
	if err == nil || !strings.Contains(err.Error(), `mod "reniesta": data/missions.toml:5: map: mission/en/map.alm: no such file in the mod folder`) {
		t.Fatalf("%v", err)
	}
}

const replacingSrc = "[[mission]]\nkind = \"replace\"\nkey = \"r\"\nnumber = 20\nmap = \"m/{lang}/map.alm\"\n"

func TestTwoModsCannotReplaceOneMission(t *testing.T) {
	two := strings.Replace(replacingSrc, "number = 20", "number = 10", 1)
	mods := makeMods(t, map[string]map[string]string{"m1": missionMod(replacingSrc), "m2": missionMod(two)})
	res, err := Load(mods, "rom1-ru", nil, Options{})
	if err != nil {
		t.Fatalf("two mods replacing two missions: %v", err)
	}
	if _, ok := res.Missions.Replacing(10); !ok {
		t.Fatalf("%+v", res.Missions)
	}
	mods = makeMods(t, map[string]map[string]string{"m1": missionMod(replacingSrc), "m2": missionMod(replacingSrc)})
	if err := mergedMissions(mods); err == nil ||
		!strings.Contains(err.Error(), `data/missions.toml:1: mission 20 is added by mod "m1" and named by mod "m2", and neither loads after the other`) {
		t.Fatalf("two mods replacing mission 20: %v", err)
	}
	// A guest after mission 20 and a replacement of mission 20 compose.
	mods = makeMods(t, map[string]map[string]string{"m1": missionMod(replacingSrc), "m2": missionMod(missionSrc)})
	if _, err := Load(mods, "rom1-ru", nil, Options{}); err != nil {
		t.Fatalf("a guest after the replaced mission: %v", err)
	}
}
