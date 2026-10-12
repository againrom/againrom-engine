package game

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/mapload"
	"againrom/pkg/mod"
	"againrom/pkg/vfs"
)

// replaceFixtureFront is the guest fixture's campaign over an install that
// holds the map of mission 20 and of no other campaign mission.
func replaceFixtureFront(t *testing.T) *FrontEnd {
	t.Helper()
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "scenario"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "scenario", "20.alm"), synth.ALM(synth.ALMOptions{Width: 4, Height: 4, Name: "installed"}), 0o644); err != nil {
		t.Fatal(err)
	}
	fs, err := vfs.Open(nil, []string{dir})
	if err != nil {
		t.Fatal(err)
	}
	return &FrontEnd{InstallResources: InstallResources{Campaign: resolved(guestCampaign(), nil),
		Table: &mapload.Table{}, Archives: &Archives{Containers: fs}}}
}

func replaceDecl() mod.Mission {
	return mod.Mission{Mod: "r", File: mod.MissionsFile, Line: 4, Kind: mod.MissionReplace, Key: "ambush", Number: 20,
		Map: "mission/en/map.alm", Title: "Ambush", MapBytes: synth.ALM(synth.ALMOptions{Width: 4, Height: 4, Name: "modded"}),
		TextFiles: map[string][]byte{"event01.txt": []byte("E1")}}
}

func TestSetModMissionsServesAReplacingMapOverTheInstalledMission(t *testing.T) {
	f := replaceFixtureFront(t)
	shared := f.Archives.Containers
	installed, err := shared.ReadFile("scenario/20.alm")
	if err != nil {
		t.Fatal(err)
	}
	r := replaceDecl()
	if err := f.SetModMissions(mod.MissionData{Missions: []mod.Mission{r, guestDecl()}}); err != nil {
		t.Fatal(err)
	}
	for address, want := range map[string]string{
		"scenario/20.alm":                  string(r.MapBytes),
		"main/text/battle/m20/event01.txt": "E1",
		"main/text/battle/m20/title.txt":   "Ambush",
		"scenario/25.alm":                  "MAP",
	} {
		if got, err := f.Archives.Containers.ReadFile(address); err != nil || string(got) != want {
			t.Errorf("%s: %q %v", address, got, err)
		}
	}
	if got, err := shared.ReadFile("scenario/20.alm"); err != nil || string(got) != string(installed) {
		t.Error("the overlay reached the shared filesystem")
	}
	if _, err := f.Archives.Containers.ReadFile("main/text/battle/m20/briefing.txt"); err == nil {
		t.Error("a briefing the mod does not declare was served")
	}
	if got, ok := f.Table.Mods.Missions.Replacing(20); !ok || got.Key != "ambush" {
		t.Fatalf("recorded %+v", f.Table.Mods.Missions)
	}
	// The guest after mission 20 still plays: the replacement is no guest.
	if got, ok := f.Table.Mods.Missions.AfterWin(20); !ok || got.Key != "prologue" {
		t.Fatalf("AfterWin(20) = %+v", got)
	}
}

func TestSetModMissionsRefusesAReplacementTheInstallDoesNotBearOut(t *testing.T) {
	for name, c := range map[string]struct {
		edit func(*FrontEnd, *mod.Mission)
		want string
	}{
		"no campaign mission": {func(_ *FrontEnd, m *mod.Mission) { m.Number = 50 }, "number 50 is no mission of this install's campaign"},
		"no installed map":    {func(_ *FrontEnd, m *mod.Mission) { m.Number = 31 }, "the install holds no map for mission 31 (scenario/31.alm)"},
		"broken map":          {func(_ *FrontEnd, m *mod.Mission) { m.MapBytes = []byte("MAP") }, "map mission/en/map.alm: "},
	} {
		f := replaceFixtureFront(t)
		m := replaceDecl()
		c.edit(f, &m)
		err := f.SetModMissions(mod.MissionData{Missions: []mod.Mission{m}})
		if err == nil || !strings.Contains(err.Error(), `mod "r": data/missions.toml:4: `+c.want) {
			t.Errorf("%s: %v, want %q", name, err, c.want)
		}
		if !f.Table.Mods.Missions.Empty() {
			t.Errorf("%s: a refused declaration was recorded", name)
		}
	}
	// Loss control: the unedited declaration is served.
	if err := replaceFixtureFront(t).SetModMissions(mod.MissionData{Missions: []mod.Mission{replaceDecl()}}); err != nil {
		t.Fatal(err)
	}
}

func TestAnUnmarkedSaveInsideAReplacedMissionIsRefused(t *testing.T) {
	ctx := mapload.ModContext{Missions: mod.MissionData{Missions: []mod.Mission{replaceDecl()}}}
	unmarked := []byte("an unmarked save")
	if err := refuseUnmarkedReplaced(unmarked, 20, ctx); err == nil ||
		!strings.Contains(err.Error(), `mission 20 without a mod mark, and mod "r" replaces that mission's map`) {
		t.Fatalf("mission 20: %v", err)
	}
	for _, n := range []int{0, 10} {
		if err := refuseUnmarkedReplaced(unmarked, n, ctx); err != nil {
			t.Fatalf("mission %d: %v", n, err)
		}
	}
	if err := refuseUnmarkedReplaced(unmarked, 20, mapload.ModContext{}); err != nil {
		t.Fatalf("without the mod: %v", err)
	}
}
