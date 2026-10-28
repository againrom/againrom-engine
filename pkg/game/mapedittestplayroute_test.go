package game

import (
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/mod"
	"againrom/pkg/ui"
)

// routeRegistry is testPlayRegistry whose town of chapter 40 also grants
// companion 23, so a package can name a guest, an after, a chapter and a gold
// that testPlayGuest never picks.
func routeRegistry() []byte {
	return synth.Reg(kindRoot, []synth.RegNode{
		{Name: "Mission10", Kind: kindDir, Children: []synth.RegNode{{Name: "AutoGetMission", Kind: kindInt, Int: 20}}},
		{Name: "Mission20", Kind: kindDir, Children: []synth.RegNode{{Name: "Mercenaries", Kind: kindInt, Int: 1}}},
		{Name: "Mission30", Kind: kindDir, Children: []synth.RegNode{
			{Name: "InnMission", Kind: kindInt, Int: 30},
			{Name: "ShopMission", Kind: kindInt, Int: 31},
			{Name: "AddHero", Kind: kindIntArray, Ints: []int32{22}},
		}},
		{Name: "Mission31", Kind: kindDir, Children: []synth.RegNode{{Name: "Payment", Kind: kindInt, Int: 700}}},
		{Name: "Mission40", Kind: kindDir, Children: []synth.RegNode{
			{Name: "InnMission", Kind: kindIntArray, Ints: []int32{0, 40, 41}},
			{Name: "AddHero", Kind: kindIntArray, Ints: []int32{23}},
		}},
		{Name: "Mission41", Kind: kindDir, Children: []synth.RegNode{{Name: "Payment", Kind: kindInt, Int: 900}}},
	})
}

// routeFixture is a new map in the W window's editor over routeRegistry.
func routeFixture(t *testing.T) (*testPlayFixture, *ui.MapEditor, *mapInspectionEdits) {
	t.Helper()
	f, e := workspaceFixture(t)
	f.x.archives.Containers.Overlay(ScenarioRegistry, func() ([]byte, error) { return routeRegistry(), nil })
	return f, e, e.Document().Edits.(*mapInspectionEdits)
}

// setRoute fills the W window's route rows by key.
func setRoute(t *testing.T, e *ui.MapEditor, rows map[string]int, kind string) {
	t.Helper()
	index := map[string]int{"number": 4, "after": 5, "guest": 6, "chapter": 7, "gold": 8}
	if kind != "" {
		if err := workspaceField(e, "package", 3, kind); err != nil {
			t.Fatal(err)
		}
	}
	for key, value := range rows {
		if err := workspaceField(e, "package", index[key], strconv.Itoa(value)); err != nil {
			t.Fatal(err)
		}
	}
}

func testModFolder(f *testPlayFixture) string {
	return filepath.Join(f.profile, TestPlayModID, TestPlayModID)
}

func TestTestPlayDeclaresTheOpenPackagesGuestRoute(t *testing.T) {
	f, e, _ := routeFixture(t)
	setRoute(t, e, map[string]int{"number": 35, "after": 30, "guest": 23, "chapter": 40, "gold": 500}, "")
	if err := pastNotice(e, "testplay"); err != nil {
		t.Fatal(err)
	}
	m := loadWorkspaceMod(t, f.root, testModFolder(f))
	if m.Kind != mod.MissionGuest || m.Number != 35 || m.After != 30 || m.Guest != 23 || m.Chapter != 40 || m.Gold != 500 || m.Carry != mod.CarryJoin {
		t.Fatalf("the test mod declares %+v", m)
	}
	toml := tree(t, testModFolder(f))[mod.MissionsFile]
	for _, want := range []string{"number  = 35", "after   = 30", "guest   = 23", "chapter = 40", "gold = 500"} {
		if !strings.Contains(toml, want) {
			t.Errorf("%s lacks %q:\n%s", mod.MissionsFile, want, toml)
		}
	}
	if len(f.started) != 1 || !reflect.DeepEqual(f.started[0][len(f.started[0])-2:], []string{"-mission", "35"}) {
		t.Fatalf("started %q", f.started)
	}
}

func TestTestPlayDeclaresTheOpenPackagesReplaceRoute(t *testing.T) {
	f, e, _ := routeFixture(t)
	original := append([]byte(nil), e.Document().SourceBytes...)
	f.x.archives.Containers.Overlay("scenario/20.alm", func() ([]byte, error) { return append([]byte(nil), original...), nil })
	setRoute(t, e, map[string]int{"number": 20}, "replace")
	if err := pastNotice(e, "testplay"); err != nil {
		t.Fatal(err)
	}
	m := loadWorkspaceMod(t, f.root, testModFolder(f))
	if m.Kind != mod.MissionReplace || m.Number != 20 || m.After != 0 || m.Guest != 0 || m.Chapter != 0 || m.Gold != 0 {
		t.Fatalf("the test mod declares %+v", m)
	}
	if len(f.started) != 1 || !reflect.DeepEqual(f.started[0][len(f.started[0])-2:], []string{"-mission", "20"}) {
		t.Fatalf("started %q", f.started)
	}
}

func TestTestPlayRefusesARouteTheSaveRefusesInTheSameWords(t *testing.T) {
	for _, c := range []struct {
		name string
		kind string
		rows map[string]int
	}{
		{"a guest the town does not grant", "", map[string]int{"number": 35, "after": 30, "guest": 22, "chapter": 40}},
		{"a chapter the win does not open", "", map[string]int{"number": 35, "after": 30, "guest": 23, "chapter": 30}},
		{"a main mission number", "", map[string]int{"number": 30, "after": 30, "guest": 23, "chapter": 40}},
		{"a replaced mission outside the campaign", "replace", map[string]int{"number": 50}},
		{"a replaced mission with no installed map", "replace", map[string]int{"number": 20}},
	} {
		t.Run(c.name, func(t *testing.T) {
			f, e, s := routeFixture(t)
			setRoute(t, e, c.rows, c.kind)
			_, saveErr := s.saveWorkspacePackage(filepath.Join(t.TempDir(), "authored-map"))
			if saveErr == nil {
				t.Fatal("the package save accepted the route")
			}
			_, err := s.TestPlay()
			if err == nil || err.Error() != "Test play: "+saveErr.Error() {
				t.Fatalf("test play: %v; the package save: %v", err, saveErr)
			}
			if got := tree(t, f.profile); len(got) != 0 || len(f.started) != 0 {
				t.Fatalf("a refused test play wrote %v and started %q", keys(got), f.started)
			}
		})
	}
}

// A hand-written package whose mission names a party opens in the W window
// with the party's mission number and after but no party: the window's route
// holds guest, chapter and gold only, so test play and the package save
// declare the mission with guest 0, which the mod loader refuses. Neither
// writes a package.
func TestTestPlayOfAPartyPackageDeclaresNoParty(t *testing.T) {
	f, e, _ := routeFixture(t)
	original := append([]byte(nil), e.Document().SourceBytes...)
	dir := writeEditorMod(t, t.TempDir(), "[[mission]]\nkey = \"party\"\nnumber = 25\nmap = \"mission/{lang}/map.alm\"\nafter = 20\nparty = [{ character = \"Monk\" }]\n")
	writeTestPlayInput(t, filepath.Join(dir, "mission/en/map.alm"), original)
	if err := workspaceField(e, "package", 0, dir); err != nil {
		t.Fatal(err)
	}
	s := e.Document().Edits.(*mapInspectionEdits)
	if err := s.prepareWorkspace(); err != nil {
		t.Fatal(err)
	}
	if p := s.workspace.Package; p.Number != 25 || p.After != 20 || p.Guest != 0 || p.Chapter != 0 {
		t.Fatalf("the window holds %+v", p)
	}
	_, saveErr := s.saveWorkspacePackage(filepath.Join(t.TempDir(), s.workspace.Package.ID))
	_, playErr := s.TestPlay()
	for name, err := range map[string]error{"package save": saveErr, "test play": playErr} {
		if err == nil || !strings.Contains(err.Error(), "guest is 0; use 1 to 65535") {
			t.Errorf("%s: %v", name, err)
		}
	}
	if got := tree(t, f.profile); len(got) != 0 || len(f.started) != 0 {
		t.Fatalf("a refused test play wrote %v and started %q", keys(got), f.started)
	}
}
