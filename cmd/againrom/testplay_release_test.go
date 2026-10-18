package main

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"againrom/pkg/formats/alm"
	"againrom/pkg/game"
	"againrom/pkg/mod"
	"againrom/pkg/modrt"
)

// testPlayEntities runs the map editor's test play of installed mission 20,
// with one creature deleted when remove is set, and then this program with
// exactly the arguments the test play started it with, in -check mode. It
// answers the started arguments and the checked mission line's number,
// address and entity count.
func testPlayEntities(t *testing.T, root string, remove bool) ([]string, string, int) {
	t.Helper()
	x, err := game.NewMapInspector(root)
	if err != nil {
		t.Fatal(err)
	}
	self, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	profile := t.TempDir()
	var started [][]string
	x.SetTestPlay(game.TestPlayHost{
		Profile: func() (game.RuntimeProfile, error) {
			return game.ResolveRuntimeProfile(filepath.Join(profile, "saves"), "", "", root)
		},
		Game: self,
		Start: func(program string, args []string) error {
			started = append(started, args)
			return nil
		},
	})
	e := x.Editor()
	if err := e.Open("scenario/20.alm"); err != nil {
		t.Fatal(err)
	}
	e.Layout(1280, 800)
	if remove {
		a, err := game.OpenArchives(root)
		if err != nil {
			t.Fatal(err)
		}
		b, err := a.Containers.ReadFile("scenario/20.alm")
		if err != nil {
			t.Fatal(err)
		}
		m, err := alm.Open(b)
		if err != nil {
			t.Fatal(err)
		}
		index := -1
		for i, u := range m.Units {
			if u.ClassID >= 0x1a {
				index = i
				break
			}
		}
		if index < 0 {
			t.Fatal("mission 20 places no creature")
		}
		if d, err := e.Document().Edits.Delete("Unit", index); err != nil || d == nil {
			t.Fatalf("delete unit %d: %v", index, err)
		}
		if !e.Dirty() {
			t.Fatal("the deletion left the map clean")
		}
	}
	if err := e.Key("testplay"); err != nil {
		t.Fatalf("test play: %v", err)
	}
	if len(started) != 1 {
		t.Fatalf("the test play started %d programs", len(started))
	}
	args := started[0]
	code, out, stderr := checkRun(t, append(append([]string{}, args...), "-check"), "")
	if code != 0 {
		t.Fatalf("-check %q: exit %d: %s", args, code, stderr)
	}
	line := regexp.MustCompile(`againrom: mission (\d+) at (\S+), \d+x\d+, (\d+) entities`).FindStringSubmatch(out)
	if line == nil {
		t.Fatalf("no mission line in\n%s", out)
	}
	n, _ := strconv.Atoi(line[3])
	return args, line[1] + " " + line[2], n
}

// TestReleaseTestPlayStartsTheEditedMapAsAGuestMission plays the editor's
// test play on the lawful install: the arguments it starts the game with load
// the test mod, and the mission they open is the edited map, one creature
// short. Loss control: the unedited map's test play opens it with that
// creature.
func TestReleaseTestPlayStartsTheEditedMapAsAGuestMission(t *testing.T) {
	root := os.Getenv("AGAINROM_ASSETS")
	if root == "" {
		t.Skip("no AGAINROM_ASSETS: the test play over a lawful install needs one")
	}
	args, mission, edited := testPlayEntities(t, root, true)
	_, plainMission, plain := testPlayEntities(t, root, false)
	n, _ := strconv.Atoi(args[len(args)-1])
	want := "scenario/" + strconv.Itoa(n) + ".alm"
	if mission != strconv.Itoa(n)+" "+want || plainMission != mission {
		t.Fatalf("the test play opened %q and %q, want mission %d at %s", mission, plainMission, n, want)
	}
	if n%10 == 0 {
		t.Fatalf("the test play started a campaign mission number %d", n)
	}
	if edited != plain-1 {
		t.Fatalf("the edited test play holds %d entities, the unedited %d; want one fewer", edited, plain)
	}
	t.Logf("test play %q: mission %s, %d entities (unedited %d)", args, mission, edited, plain)
}

func testPlayFileSnapshot(t *testing.T, dir string) map[string]string {
	t.Helper()
	files := map[string]string{}
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		name, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		var b []byte
		if !info.IsDir() {
			b, err = os.ReadFile(path)
			if err != nil {
				return err
			}
		}
		files[name] = info.Mode().String() + "\x00" + string(b)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return files
}

func TestReleaseTestPlayRefusesAnOversizedModPageBeforeWritingOrStarting(t *testing.T) {
	root := os.Getenv("AGAINROM_ASSETS")
	if root == "" {
		t.Skip("no AGAINROM_ASSETS: the test play over a lawful install needs one")
	}
	x, err := game.NewMapInspector(root)
	if err != nil {
		t.Fatal(err)
	}
	folder := filepath.Join(t.TempDir(), "authored")
	write := func(name string, b []byte) {
		t.Helper()
		path := filepath.Join(folder, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, b, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write(mod.ManifestName, []byte("id = \"authored\"\ntitle = \"Author map\"\nversion = \"1.0.0\"\napi = 1\napplies-to = [\"rom1\"]\nentry = \"main.star\"\n"))
	write("main.star", []byte("def init(game, settings):\n    game.data.add(\"data/missions.toml\")\n"))
	write(mod.MissionsFile, []byte("[[mission]]\nkind = \"replace\"\nkey = \"authored\"\nnumber = 20\nmap = \"mission/{lang}/map.alm\"\ntexts = \"mission/{lang}/text\"\n"))
	lang := modrt.LanguageFor(game.BaseID(game.InspectInstall(root)))
	if _, err := game.AssembleModMission(root, lang, 20, folder); err != nil {
		t.Fatal(err)
	}
	page := "mission/" + lang + "/text/event01.txt"
	bound := bytes.Repeat([]byte("A"), mod.MaxFileBytes)
	write(page, bound)
	if err := x.SetMod(folder); err != nil {
		t.Fatal(err)
	}
	profile := t.TempDir()
	runtimeProfile, err := game.ResolveRuntimeProfile(filepath.Join(profile, "saves"), "", "", root)
	if err != nil {
		t.Fatal(err)
	}
	self, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	var started [][]string
	x.SetTestPlay(game.TestPlayHost{
		Profile: func() (game.RuntimeProfile, error) { return runtimeProfile, nil },
		Game:    self,
		Start: func(_ string, args []string) error {
			started = append(started, append([]string(nil), args...))
			return nil
		},
	})
	e := x.Editor()
	opened := filepath.Join(folder, "mission", lang, "map.alm")
	if err := e.Open(opened); err != nil {
		t.Fatal(err)
	}
	e.Layout(1280, 800)
	sourceBefore := testPlayFileSnapshot(t, folder)
	if err := e.Key("testplay"); err != nil {
		t.Fatal(err)
	}
	if len(started) != 1 {
		t.Fatalf("in-bound page: starts=%d message=%q", len(started), e.Message())
	}
	if !reflect.DeepEqual(testPlayFileSnapshot(t, folder), sourceBefore) {
		t.Fatal("in-bound test play changed the source mod")
	}
	args := started[0]
	code, out, stderr := checkRun(t, append(append([]string(nil), args...), "-check"), "")
	if code != 0 || !strings.Contains(out, "mod 1: "+game.TestPlayModID+" ") || !strings.Contains(out, "againrom: mission ") {
		t.Fatalf("in-bound -check %q: exit=%d stdout=%q stderr=%q", args, code, out, stderr)
	}
	copied, err := os.ReadFile(filepath.Join(runtimeProfile.Directory, game.TestPlayModID, game.TestPlayModID, "mission", "text", lang, "event01.txt"))
	if err != nil || !bytes.Equal(copied, bound) {
		t.Fatalf("in-bound test page was not published intact: %v", err)
	}

	write(page, append(bound, 'A'))
	profileBefore := testPlayFileSnapshot(t, profile)
	sourceBefore = testPlayFileSnapshot(t, folder)
	documentBefore := e.Document()
	started = nil
	refusal := e.Open(opened)
	message := e.Message()
	if refusal == nil || message != refusal.Error() || len(started) != 0 || !strings.Contains(message, page) || !strings.Contains(message, "larger than "+strconv.Itoa(mod.MaxFileBytes)+" bytes") {
		t.Fatalf("oversized page: error=%v starts=%d message=%q", refusal, len(started), message)
	}
	if e.Document() != documentBefore {
		t.Fatal("refused page source replaced the editor document")
	}
	if !reflect.DeepEqual(testPlayFileSnapshot(t, profile), profileBefore) {
		t.Fatal("refused test play changed the existing owned destination or profile")
	}
	if !reflect.DeepEqual(testPlayFileSnapshot(t, folder), sourceBefore) {
		t.Fatal("refused test play changed the source mod")
	}
	t.Logf("root=%s; in-bound page=%d bytes; recorded arguments=%q; production -check exit=%d; oversized page=%d bytes; message=%q; starts=0; owned destination and source unchanged", root, len(bound), args, code, len(bound)+1, message)
}
