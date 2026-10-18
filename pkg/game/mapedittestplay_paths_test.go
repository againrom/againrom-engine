package game

import (
	"os"
	"path/filepath"
	"reflect"
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

func TestTestPlayRefusesAnOwnedDestinationOverlappingTheOpenedMapFolder(t *testing.T) {
	for _, relation := range []string{"equal", "contains", "inside"} {
		t.Run(relation, func(t *testing.T) {
			f, folder := ownedTestPlayFixture(t)
			sourceFolder := testPlayProtectedFolder(folder, relation)
			opened := filepath.Join(sourceFolder, "authored.alm")
			writeTestPlayInput(t, opened, synth.ALM(synth.ALMOptions{Width: 4, Height: 4, Name: "authored"}))
			before, source := testPlayPathSnapshot(t, f.profile), testPlayPathSnapshot(t, sourceFolder)
			_, err := editorEdits(t, f.x, opened).TestPlay()
			if err == nil || !strings.Contains(err.Error(), folder) {
				t.Fatalf("owned destination must refuse the opened folder and name %s: %v", folder, err)
			}
			if len(f.started) != 0 || !reflect.DeepEqual(before, testPlayPathSnapshot(t, f.profile)) || !reflect.DeepEqual(source, testPlayPathSnapshot(t, sourceFolder)) {
				t.Fatal("overlapping opened folder was changed or started another game")
			}
		})
	}
}

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
			if err == nil || !strings.Contains(err.Error(), folder) {
				t.Fatalf("owned destination must refuse the selected mod and name %s: %v", folder, err)
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
