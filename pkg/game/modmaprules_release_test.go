package game

import (
	"encoding/binary"
	"fmt"
	"os"
	"slices"
	"strconv"
	"strings"
	"testing"

	"againrom/internal/firstgame"
	"againrom/pkg/mod"
	"againrom/pkg/modrt"
)

// TestReleaseModMapRuleCensusRefusesNoInstalledMap runs the rule table the mod
// load runs over every installed scenario map as a replacing mod mission would
// ship it unchanged, with no texts of its own, and counts the findings per rule
// and level. An installed map is never refused.
func TestReleaseModMapRuleCensusRefusesNoInstalledMap(t *testing.T) {
	firstgame.SkipOnSecondGame(t)
	f := releaseFront(t)
	x, err := f.modMapInspector()
	if err != nil {
		t.Fatal(err)
	}
	maps := ArchiveMaps(f.Archives.Containers)
	count := map[string]int{}
	checked := 0
	for _, name := range maps.Names() {
		n, err := strconv.Atoi(strings.TrimSuffix(name, ".alm"))
		if err != nil {
			continue
		}
		b, err := maps.Read(name)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		checked++
		m := mod.Mission{Mod: "census", Kind: mod.MissionReplace, Number: n, MapBytes: b}
		address, _ := MissionMap(n)
		for _, finding := range x.CheckModMap(m, address) {
			level := "warning"
			if finding.Error {
				level = "error"
				t.Errorf("%s refused: %s", name, finding.Line())
			}
			count[level+" "+finding.Rule]++
		}
	}
	if checked == 0 {
		t.Fatal("the install holds no numbered scenario map")
	}
	keys := make([]string, 0, len(count))
	for k := range count {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	lines := []string{fmt.Sprintf("maps checked %d", checked)}
	for _, k := range keys {
		lines = append(lines, fmt.Sprintf("%s %d", k, count[k]))
	}
	t.Log("census: " + strings.Join(lines, "; "))
}

// outsideUnit is the map with its first unit moved outside the map.
func outsideUnit(t *testing.T, b []byte) []byte {
	t.Helper()
	out := slices.Clone(b)
	binary.LittleEndian.PutUint32(out[payloadOffsets(t, out)[6]:], 4000*256)
	return out
}

// TestReleaseModMissionRoutesRefuseAMapTheRulesReport loads a guest mission and
// a replacing mission through the production load on the install under test: a
// unit outside the map is refused with the rule named, the unedited map loads.
func TestReleaseModMissionRoutesRefuseAMapTheRulesReport(t *testing.T) {
	firstgame.SkipOnSecondGame(t)
	dir := guestMods(t, guestModID)
	entries, err := mod.Resolve(dir, []string{guestModID})
	if err != nil {
		t.Fatal(err)
	}
	install := os.Getenv("AGAINROM_ASSETS")
	res, err := modrt.Load(entries, BaseID(InspectInstall(install)), nil, modrt.Options{})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Missions.Missions) != 1 {
		t.Fatalf("the fixture mod declares %d missions", len(res.Missions.Missions))
	}
	guest := res.Missions.Missions[0]
	base := releaseFront(t)
	installed, err := base.Archives.Containers.ReadFile("scenario/20.alm")
	if err != nil {
		t.Fatal(err)
	}
	replacing := mod.Mission{Mod: "census", File: mod.MissionsFile, Line: 3, Kind: mod.MissionReplace, Key: "r", Number: 20, Map: "map.alm", MapBytes: installed}
	for name, m := range map[string]mod.Mission{"guest": guest, "replacing": replacing} {
		t.Run(name, func(t *testing.T) {
			bad := m
			bad.MapBytes = outsideUnit(t, m.MapBytes)
			_, err := releaseFront(t).SetModMissions(mod.MissionData{Missions: []mod.Mission{bad}})
			if err == nil || !strings.Contains(err.Error(), "error record-outside Unit #0") || !strings.Contains(err.Error(), fmt.Sprintf("mod %q: %s:%d: ", m.Mod, m.File, m.Line)) {
				t.Fatalf("outside unit: %v", err)
			}
			if _, err := releaseFront(t).SetModMissions(mod.MissionData{Missions: []mod.Mission{m}}); err != nil {
				t.Fatalf("control: %v", err)
			}
		})
	}
}
