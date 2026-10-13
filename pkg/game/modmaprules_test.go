package game

import (
	"strings"
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/mapedit"
	"againrom/pkg/mod"
)

// ruleFront is a front end whose tables hold what the rule fixtures read, and
// whose campaign takes the guest and the replacing mission of the fixtures.
func ruleFront(t *testing.T) *FrontEnd {
	t.Helper()
	f := replaceFixtureFront(t)
	f.Table = checkInspector().table
	return f
}

func ruleMapBytes(t *testing.T, o synth.ALMOptions, edit func(*mapedit.Editor)) []byte {
	t.Helper()
	return built(t, o, edit, nil).Bytes()
}

func ruleMissions(t *testing.T, mapBytes []byte) map[string]mod.Mission {
	t.Helper()
	guest := guestDecl()
	guest.MapBytes = mapBytes
	replace := replaceDecl()
	replace.MapBytes = mapBytes
	return map[string]mod.Mission{"guest": guest, "replacing": replace}
}

func TestSetModMissionsRunsTheMapRuleTable(t *testing.T) {
	outside := cleanOptions()
	outside.Units[1].X = 40*256 + 5
	twoSacks := func(m *mapedit.Editor) {
		for i := 0; i < 2; i++ {
			if _, err := m.Place(mapedit.KindLoot, m.GroundSack(14*256+128, 14*256+128, 0x0101)); err != nil {
				t.Fatal(err)
			}
		}
	}
	for route := range ruleMissions(t, nil) {
		t.Run(route, func(t *testing.T) {
			load := func(b []byte) ([]string, error) {
				return ruleFront(t).SetModMissions(mod.MissionData{Missions: []mod.Mission{ruleMissions(t, b)[route]}})
			}
			// An error finding refuses the mission with the mod, the file, the
			// line and the rule.
			_, err := load(ruleMapBytes(t, outside, nil))
			want := ruleMissions(t, nil)[route]
			if err == nil || !strings.Contains(err.Error(), `mod "`+want.Mod+`": data/missions.toml:`) ||
				!strings.Contains(err.Error(), "error record-outside Unit #1 cell 40,18: ") {
				t.Fatalf("error map: %v", err)
			}
			// Loss control: the same map with the unit inside loads, silently.
			if warnings, err := load(ruleMapBytes(t, cleanOptions(), nil)); err != nil || len(warnings) != 0 {
				t.Fatalf("clean map: %v %v", warnings, err)
			}
			// A warning refuses nothing and comes back as a line of the mod.
			warnings, err := load(ruleMapBytes(t, cleanOptions(), twoSacks))
			if err != nil || len(warnings) != 1 || !strings.Contains(warnings[0], `mod "`+want.Mod+`": data/missions.toml:`) ||
				!strings.Contains(warnings[0], "warning sack-duplicate Sack #1") {
				t.Fatalf("warning map: %q %v", warnings, err)
			}
		})
	}
}

func TestSetModMissionsRuleTableReadsTheTextsTheModShips(t *testing.T) {
	m := guestDecl()
	m.MapBytes = ruleMapBytes(t, cleanOptions(), nil)
	m.Texts = "texts"
	m.TextFiles = map[string][]byte{"event01.txt": []byte("ok \u4e2d\r\n")}
	warnings, err := ruleFront(t).SetModMissions(mod.MissionData{Missions: []mod.Mission{m}})
	if err != nil || len(warnings) != 1 || !strings.Contains(warnings[0], "warning mod-message-text map") ||
		!strings.Contains(warnings[0], "texts/event01.txt holds the character") {
		t.Fatalf("%q %v", warnings, err)
	}
	m.TextFiles = map[string][]byte{"event01.txt": []byte("ok\r\n")}
	if warnings, err = ruleFront(t).SetModMissions(mod.MissionData{Missions: []mod.Mission{m}}); err != nil || len(warnings) != 0 {
		t.Fatalf("control: %q %v", warnings, err)
	}
}
