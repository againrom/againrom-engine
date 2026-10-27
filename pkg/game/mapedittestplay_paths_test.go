package game

import (
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/mod"
)

func ownedTestPlayFixture(t *testing.T) (*testPlayFixture, string) {
	t.Helper()
	f := newTestPlayFixture(t)
	if _, err := editorEdits(t, f.x, "scenario/20.alm").TestPlay(); err != nil {
		t.Fatal(err)
	}
	if len(f.started) != 1 {
		t.Fatalf("initial test play starts=%d", len(f.started))
	}
	f.started = nil
	return f, filepath.Join(f.profile, TestPlayModID, TestPlayModID)
}

func testPlayPathSnapshot(t *testing.T, dir string) map[string]string {
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

func testPlayProtectedFolder(folder, relation string) string {
	switch relation {
	case "equal":
		return folder
	case "contains":
		return filepath.Join(folder, "authored")
	default:
		return filepath.Dir(folder)
	}
}

// withoutTestPlayFolder is the snapshot without the entry rel and the entries
// below it.
func withoutTestPlayFolder(snapshot map[string]string, rel string) map[string]string {
	out := map[string]string{}
	for name, v := range snapshot {
		if name != rel && !strings.HasPrefix(name, rel+string(filepath.Separator)) {
			out[name] = v
		}
	}
	return out
}

// testPlayRefusal is the refusal of a test folder in relation to the named
// protected folder; "" where the test play goes ahead.
func testPlayRefusal(folder, relation, name string, walked bool) string {
	switch {
	case relation == "equal":
		return "the test folder " + folder + " is " + name
	case relation == "contains":
		return "the test folder " + folder + " contains " + name
	case walked:
		return "the test folder " + folder + " lies inside " + name
	}
	return ""
}

// The opened map's folder may contain the test folder: no reader walks it
// below the map's own pages. Only "equal" and "contains" refuse.
func TestTestPlayRefusesAnOwnedDestinationOverlappingTheOpenedMapFolder(t *testing.T) {
	for _, relation := range []string{"equal", "contains", "inside"} {
		t.Run(relation, func(t *testing.T) {
			f, folder := ownedTestPlayFixture(t)
			sourceFolder := testPlayProtectedFolder(folder, relation)
			opened := filepath.Join(sourceFolder, "authored.alm")
			writeTestPlayInput(t, opened, synth.ALM(synth.ALMOptions{Width: 4, Height: 4, Name: "authored"}))
			before, source := testPlayPathSnapshot(t, f.profile), testPlayPathSnapshot(t, sourceFolder)
			_, err := editorEdits(t, f.x, opened).TestPlay()
			want := testPlayRefusal(folder, relation, "the opened map's folder", false)
			if want == "" {
				if err != nil || len(f.started) != 1 {
					t.Fatalf("a test folder inside the opened map's folder must play: %v; starts=%d", err, len(f.started))
				}
				after := withoutTestPlayFolder(testPlayPathSnapshot(t, sourceFolder), TestPlayModID)
				if !reflect.DeepEqual(withoutTestPlayFolder(source, TestPlayModID), after) {
					t.Fatalf("the opened map's folder changed outside the test folder: %v", after)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), want) {
				t.Fatalf("owned destination must refuse the opened folder with %q: %v", want, err)
			}
			if len(f.started) != 0 || !reflect.DeepEqual(before, testPlayPathSnapshot(t, f.profile)) || !reflect.DeepEqual(source, testPlayPathSnapshot(t, sourceFolder)) {
				t.Fatal("overlapping opened folder was changed or started another game")
			}
		})
	}
}

// The selected mod folder refuses a test folder inside it as well: its
// digest, pack and package save walk every subfolder.
func TestTestPlayRefusesAnOwnedDestinationOverlappingTheSelectedModFolder(t *testing.T) {
	for _, relation := range []string{"equal", "contains", "inside"} {
		t.Run(relation, func(t *testing.T) {
			f, folder := ownedTestPlayFixture(t)
			selected := testPlayProtectedFolder(folder, relation)
			if relation != "equal" {
				manifest := strings.Replace(editorModManifest, `id = "ambush"`, `id = "`+filepath.Base(selected)+`"`, 1)
				writeTestPlayInput(t, filepath.Join(selected, mod.ManifestName), []byte(manifest))
				writeTestPlayInput(t, filepath.Join(selected, "main.star"), []byte(testPlayEntry))
				writeTestPlayInput(t, filepath.Join(selected, filepath.FromSlash(mod.MissionsFile)), []byte(editorModMissions))
			}
			if err := f.x.SetMod(selected); err != nil {
				t.Fatal(err)
			}
			sourceFolder := t.TempDir()
			opened := filepath.Join(sourceFolder, "authored.alm")
			writeTestPlayInput(t, opened, synth.ALM(synth.ALMOptions{Width: 4, Height: 4, Name: "authored"}))
			before, source := testPlayPathSnapshot(t, f.profile), testPlayPathSnapshot(t, sourceFolder)
			_, err := editorEdits(t, f.x, opened).TestPlay()
			want := testPlayRefusal(folder, relation, "the selected mod folder", true)
			if err == nil || !strings.Contains(err.Error(), want) {
				t.Fatalf("owned destination must refuse the selected mod with %q: %v", want, err)
			}
			if len(f.started) != 0 || !reflect.DeepEqual(before, testPlayPathSnapshot(t, f.profile)) || !reflect.DeepEqual(source, testPlayPathSnapshot(t, sourceFolder)) {
				t.Fatal("overlapping selected mod or source was changed or started another game")
			}
		})
	}
}

func TestTestPlayRefusesAnUnrecordedPageCollisionInAnOwnedDestination(t *testing.T) {
	f, folder := ownedTestPlayFixture(t)
	writeTestPlayInput(t, filepath.Join(folder, "mission", "text", "en", "event01.txt"), []byte("Author page never written by test play"))
	modDir := writeEditorMod(t, t.TempDir(), editorModMissions+"texts = \"mission/{lang}/text\"\n")
	opened := filepath.Join(modDir, "mission", "en", "map.alm")
	writeTestPlayInput(t, opened, synth.ALM(synth.ALMOptions{Width: 4, Height: 4, Name: "authored"}))
	writeTestPlayInput(t, filepath.Join(modDir, "mission", "en", "text", "event01.txt"), []byte("Current source page"))
	if err := f.x.SetMod(modDir); err != nil {
		t.Fatal(err)
	}
	before, source := testPlayPathSnapshot(t, f.profile), testPlayPathSnapshot(t, modDir)
	_, err := editorEdits(t, f.x, opened).TestPlay()
	if err == nil || !strings.Contains(err.Error(), "event01.txt") {
		t.Fatalf("unrecorded page collision must be refused: %v", err)
	}
	if len(f.started) != 0 || !reflect.DeepEqual(before, testPlayPathSnapshot(t, f.profile)) || !reflect.DeepEqual(source, testPlayPathSnapshot(t, modDir)) {
		t.Fatal("page collision changed the destination or source or started another game")
	}
}

// A map lying directly in the game folder, with the editor run from inside
// the install: the profile and its testplay/testplay lie inside the map's
// folder, and F5 plays.
func TestTestPlayPlaysAMapLyingInTheFolderThatHoldsTheProfile(t *testing.T) {
	f := newTestPlayFixture(t)
	for _, name := range []string{GraphicsArchive, WorldArchive, MoviesArchive} {
		writeTestPlayInput(t, filepath.Join(f.root, name), nil)
	}
	exe, config := filepath.Join(f.root, "mapedit.exe"), t.TempDir()
	f.x.SetTestPlay(TestPlayHost{
		Profile: func() (RuntimeProfile, error) { return ResolveRuntimeProfile("", exe, config, f.root) },
		Game:    f.game,
		Start: func(program string, args []string) error {
			f.started = append(f.started, append([]string{program}, args...))
			return nil
		},
	})
	profile, err := ResolveRuntimeProfile("", exe, config, f.root)
	if err != nil {
		t.Fatal(err)
	}
	modsDir := filepath.Join(profile.Directory, TestPlayModID)
	rel, err := filepath.Rel(f.root, modsDir)
	if err != nil || !filepath.IsLocal(rel) {
		t.Fatalf("the profile's %s is not inside the install %s: %v", modsDir, f.root, err)
	}
	opened := filepath.Join(f.root, "authored.alm")
	writeTestPlayInput(t, opened, synth.ALM(synth.ALMOptions{Width: 4, Height: 4, Name: "authored"}))
	before := withoutTestPlayFolder(testPlayPathSnapshot(t, f.root), rel)
	if _, err := editorEdits(t, f.x, opened).TestPlay(); err != nil || len(f.started) != 1 {
		t.Fatalf("a map in the folder holding the profile must play once: %v; starts=%d", err, len(f.started))
	}
	if !slices.Contains(f.started[0], modsDir) {
		t.Fatalf("the game did not start on the test mod in %s: %q", modsDir, f.started[0])
	}
	got := tree(t, filepath.Join(modsDir, TestPlayModID))
	if got[testPlayRecord] == "" || got[testPlayMap] == "" || got[mod.ManifestName] == "" {
		t.Fatalf("test mod not written: %v", keys(got))
	}
	if after := withoutTestPlayFolder(testPlayPathSnapshot(t, f.root), rel); !reflect.DeepEqual(before, after) {
		t.Fatalf("the map's folder changed outside %s: %v", rel, keys(after))
	}
}

// Save mod package keeps test play's rule through the one check: a package
// folder may lie inside the opened map's folder, not inside the selected mod
// folder, and a refusal changes nothing.
func TestSaveModPackageInsideTheOpenedMapFolderAndTheSelectedModFolder(t *testing.T) {
	t.Run("opened map's folder", func(t *testing.T) {
		_, e := workspaceFixture(t)
		s := e.Document().Edits.(*mapInspectionEdits)
		dir := t.TempDir()
		s.source = filepath.Join(dir, "authored.alm")
		writeTestPlayInput(t, s.source, synth.ALM(synth.ALMOptions{Width: 4, Height: 4, Name: "authored"}))
		before := testPlayPathSnapshot(t, dir)
		id := s.workspace.Package.ID
		if _, err := s.saveWorkspacePackage(filepath.Join(dir, id)); err != nil {
			t.Fatalf("a package folder inside the opened map's folder must save: %v", err)
		}
		if _, err := os.Stat(filepath.Join(dir, id, editorPackageRecord)); err != nil {
			t.Fatal(err)
		}
		if after := withoutTestPlayFolder(testPlayPathSnapshot(t, dir), id); !reflect.DeepEqual(before, after) {
			t.Fatalf("the opened map's folder changed outside the package: %v", keys(after))
		}
	})
	t.Run("selected mod folder", func(t *testing.T) {
		f, e := workspaceFixture(t)
		s := e.Document().Edits.(*mapInspectionEdits)
		modDir := writeEditorMod(t, t.TempDir(), editorModMissions)
		if err := f.x.SetMod(modDir); err != nil {
			t.Fatal(err)
		}
		before := testPlayPathSnapshot(t, modDir)
		_, err := s.saveWorkspacePackage(filepath.Join(modDir, s.workspace.Package.ID))
		want := " lies inside the selected mod folder " + modDir
		if err == nil || !strings.HasPrefix(err.Error(), "the mod package folder ") || !strings.Contains(err.Error(), want) {
			t.Fatalf("a package folder inside the selected mod must refuse with %q: %v", want, err)
		}
		if !reflect.DeepEqual(before, testPlayPathSnapshot(t, modDir)) {
			t.Fatal("the refused package changed the selected mod folder")
		}
	})
}
