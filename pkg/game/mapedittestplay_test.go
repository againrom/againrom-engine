package game

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/mapedit"
	"againrom/pkg/mapload"
	"againrom/pkg/mod"
	"againrom/pkg/modrt"
	"againrom/pkg/ui"
	"againrom/pkg/vfs"
)

// testPlayRegistry is a scenario registry whose mission 10 leads to mission
// 20 and whose mission 20 is won into the town of chapter 30, which grants
// companion 22; buildings offer 30, 31, 40 and 41.
func testPlayRegistry() []byte {
	return synth.Reg(kindRoot, []synth.RegNode{
		{Name: "Mission10", Kind: kindDir, Children: []synth.RegNode{{Name: "AutoGetMission", Kind: kindInt, Int: 20}}},
		{Name: "Mission20", Kind: kindDir, Children: []synth.RegNode{{Name: "Mercenaries", Kind: kindInt, Int: 1}}},
		{Name: "Mission30", Kind: kindDir, Children: []synth.RegNode{
			{Name: "InnMission", Kind: kindInt, Int: 30},
			{Name: "ShopMission", Kind: kindInt, Int: 31},
			{Name: "AddHero", Kind: kindIntArray, Ints: []int32{22}},
		}},
		{Name: "Mission31", Kind: kindDir, Children: []synth.RegNode{{Name: "Payment", Kind: kindInt, Int: 700}}},
		{Name: "Mission40", Kind: kindDir, Children: []synth.RegNode{{Name: "InnMission", Kind: kindIntArray, Ints: []int32{0, 40, 41}}}},
		{Name: "Mission41", Kind: kindDir, Children: []synth.RegNode{{Name: "Payment", Kind: kindInt, Int: 900}}},
	})
}

// testPlayFixture is an editor over a synthetic first-game install holding the
// registry and mission 20's message page, a game program, a profile folder
// and a recording launch.
type testPlayFixture struct {
	x       *MapInspector
	root    string
	profile string
	game    string
	started [][]string
}

func newTestPlayFixture(t *testing.T) *testPlayFixture {
	t.Helper()
	root := t.TempDir()
	for name, data := range map[string][]byte{
		ScenarioArchive: synth.Archive([]synth.File{{Path: "scenario.reg", Data: testPlayRegistry()}}),
		MainArchive: synth.Archive([]synth.File{
			{Path: "text/battle/m20/event02.txt", Data: []byte("Page two")},
			{Path: "text/battle/m20/notes.txt", Data: []byte("not a page the mission reads")},
		}),
	} {
		if err := os.WriteFile(filepath.Join(root, name), data, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	containers, err := vfs.Open([]string{filepath.Join(root, ScenarioArchive), filepath.Join(root, MainArchive)}, nil)
	if err != nil {
		t.Fatal(err)
	}
	f := &testPlayFixture{root: root, profile: t.TempDir()}
	f.game = filepath.Join(t.TempDir(), "againrom.exe")
	if err := os.WriteFile(f.game, []byte("program"), 0o644); err != nil {
		t.Fatal(err)
	}
	f.x = &MapInspector{archives: &Archives{Root: root, Containers: containers}}
	f.x.SetTestPlay(TestPlayHost{
		Profile: func() (RuntimeProfile, error) {
			return ResolveRuntimeProfile(filepath.Join(f.profile, "saves"), "", "", root)
		},
		Game: f.game,
		Start: func(program string, args []string) error {
			f.started = append(f.started, append([]string{program}, args...))
			return nil
		},
	})
	return f
}

// tree is every file under dir, by slash path.
func tree(t *testing.T, dir string) map[string]string {
	t.Helper()
	out := map[string]string{}
	err := filepath.WalkDir(dir, func(p string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		b, err := os.ReadFile(p)
		rel, _ := filepath.Rel(dir, p)
		out[filepath.ToSlash(rel)] = string(b)
		return err
	})
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		t.Fatal(err)
	}
	return out
}

func TestTestPlayWritesTheEditedMapIntoTheTestModAndStartsTheGame(t *testing.T) {
	f := newTestPlayFixture(t)
	// The map is opened from a file a mod's replacing entry names, so Save
	// would write into that mod; the test play writes neither.
	modDir := writeEditorMod(t, t.TempDir(), editorModMissions+"texts = \"mission/{lang}/text\"\n")
	opened := filepath.Join(modDir, "mission", "en", "map.alm")
	original := synth.ALM(synth.ALMOptions{Width: 4, Height: 4, Name: "edited"})
	for name, b := range map[string][]byte{opened: original, filepath.Join(modDir, "mission", "en", "text", "event01.txt"): []byte("Page one")} {
		if err := os.MkdirAll(filepath.Dir(name), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(name, b, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := f.x.SetMod(modDir); err != nil {
		t.Fatal(err)
	}
	modBefore := tree(t, modDir)
	s := editorEdits(t, f.x, opened)
	s.saved = s.model.Bytes()
	if err := s.model.SetCell(mapedit.GridRaw, mapedit.GridAltitude, 1, 1, 7); err != nil {
		t.Fatal(err)
	}
	edited, err := s.model.SaveBytes()
	if err != nil {
		t.Fatal(err)
	}

	line, err := s.TestPlay()
	if err != nil {
		t.Fatal(err)
	}
	modsDir := filepath.Join(f.profile, TestPlayModID)
	folder := filepath.Join(modsDir, TestPlayModID)
	got := tree(t, folder)
	if got[testPlayMap] != string(edited) || bytes.Equal(edited, original) {
		t.Fatal("the test mod does not hold the edited map with its unsaved edit")
	}
	if got["mission/text/en/event01.txt"] != "Page one" || got["mission/text/en/event02.txt"] != "Page two" || len(got) != 7 || got[testPlayRecord] == "" {
		t.Fatalf("the test mod holds %v", keys(got))
	}
	// The declaration is a guest mission the game's own mod load and
	// resolver accept, under a number of its own.
	res, err := modrt.Load([]mod.Entry{mod.Load(folder)}, "rom1-en", nil, modrt.Options{})
	if err != nil {
		t.Fatal(err)
	}
	m := res.Missions.Missions
	if len(m) != 1 || m[0].Kind != mod.MissionGuest || m[0].Number != 21 || m[0].After != 20 || m[0].Chapter != 30 || m[0].Guest != 22 ||
		!bytes.Equal(m[0].MapBytes, edited) || string(m[0].TextFiles["event01.txt"]) != "Page one" {
		t.Fatalf("the mod load read %+v", m)
	}
	c, err := LoadCampaign(f.x.archives.Containers)
	if err != nil {
		t.Fatal(err)
	}
	front := &FrontEnd{InstallResources: InstallResources{Campaign: resolved(c, nil), Table: &mapload.Table{}, Archives: f.x.archives}}
	if _, err := front.SetModMissions(res.Missions); err != nil {
		t.Fatalf("the game refuses the test mod: %v", err)
	}
	want := []string{f.game, "-assets", f.root, "-mods", TestPlayModID, "-mods-dir", modsDir, "-mission", "21"}
	if len(f.started) != 1 || !reflect.DeepEqual(f.started[0], want) {
		t.Fatalf("started %q, want %q", f.started, want)
	}
	if line != "Test play: againrom.exe opens mission 21" {
		t.Fatalf("line %q", line)
	}
	// Nothing the author saved changes, and the window keeps the edit unsaved.
	if after := tree(t, modDir); !reflect.DeepEqual(after, modBefore) {
		t.Fatal("the test play changed the mod Save writes into")
	}
	if !s.Dirty() || s.source != opened {
		t.Fatalf("dirty %v source %q", s.Dirty(), s.source)
	}

	// A second test play of an installed campaign map replaces the map and
	// takes that mission's installed pages; the earlier page is gone.
	f.x.mod = nil
	s2 := editorEdits(t, f.x, "scenario/20.alm")
	if _, err := s2.TestPlay(); err != nil {
		t.Fatal(err)
	}
	got = tree(t, folder)
	if got[testPlayMap] != string(mustSaveBytes(t, s2)) || got["mission/text/en/event02.txt"] != "Page two" {
		t.Fatalf("the second test play holds %v", keys(got))
	}
	if _, ok := got["mission/text/en/event01.txt"]; ok {
		t.Fatal("a page of the earlier test play is still served")
	}
	if _, ok := got["mission/text/en/notes.txt"]; ok {
		t.Fatal("a file the mission does not read was copied")
	}
	if entries, _ := os.ReadDir(f.profile); len(entries) != 1 {
		t.Fatalf("the profile holds %d entries; only the test mods folder is written", len(entries))
	}
}

func mustSaveBytes(t *testing.T, s *mapInspectionEdits) []byte {
	t.Helper()
	b, err := s.model.SaveBytes()
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func keys(m map[string]string) []string {
	var out []string
	for k := range m {
		out = append(out, k)
	}
	return out
}

func TestTestPlayOfAMapWithAnErrorStartsNothing(t *testing.T) {
	f := newTestPlayFixture(t)
	f.x.rules = []mapRule{{ID: "always", Error: true, run: func(*mapCheck) []mapHit { return []mapHit{{Text: "an error"}} }}}
	s := editorEdits(t, f.x, "scenario/20.alm")
	_, err := s.TestPlay()
	var r *ui.ReportRefusal
	if !errors.As(err, &r) || len(r.Findings) != 1 || r.Findings[0].Rule != "always" {
		t.Fatalf("err %v, want the report refusal", err)
	}
	if len(f.started) != 0 || len(tree(t, f.profile)) != 0 {
		t.Fatalf("a refused test play started %q or wrote %v", f.started, keys(tree(t, f.profile)))
	}
	// Loss control: a warning does not stop the test play.
	f.x.rules = []mapRule{{ID: "warn", run: func(*mapCheck) []mapHit { return []mapHit{{Text: "a warning"}} }}}
	s = editorEdits(t, f.x, "scenario/20.alm")
	if _, err := s.TestPlay(); err != nil || len(f.started) != 1 {
		t.Fatalf("a warning stopped the test play: %v", err)
	}
}

func TestTestPlayNamesAMissingOrFailedGameProgram(t *testing.T) {
	f := newTestPlayFixture(t)
	f.x.testPlay.Game = filepath.Join(t.TempDir(), "againrom.exe")
	_, err := editorEdits(t, f.x, "scenario/20.alm").TestPlay()
	if err == nil || !strings.Contains(err.Error(), "is not beside the map editor") || len(f.started) != 0 {
		t.Fatalf("missing game: %v, started %q", err, f.started)
	}
	if len(tree(t, f.profile)) != 0 {
		t.Fatal("a test play without the game wrote the test mod")
	}
	f = newTestPlayFixture(t)
	f.x.testPlay.Start = func(string, []string) error { return errors.New("access is denied") }
	_, err = editorEdits(t, f.x, "scenario/20.alm").TestPlay()
	if err == nil || err.Error() != "Test play: againrom.exe did not start: access is denied" {
		t.Fatalf("failed start: %v", err)
	}
}

func TestTestPlayWritesNothingIntoTheInstall(t *testing.T) {
	f := newTestPlayFixture(t)
	f.profile = filepath.Join(f.root, "Againrom")
	if err := os.Mkdir(f.profile, 0o755); err != nil {
		t.Fatal(err)
	}
	// The fixture root is no install by its archives, so the fence that holds
	// here is the editor's own: the root the editor reads is never written.
	_, err := editorEdits(t, f.x, "scenario/20.alm").TestPlay()
	if err == nil || len(f.started) != 0 {
		t.Fatalf("a profile inside the edited install: %v, started %q", err, f.started)
	}
	if entries, _ := os.ReadDir(f.profile); len(entries) != 0 {
		t.Fatalf("wrote %d entries inside the install", len(entries))
	}
}
