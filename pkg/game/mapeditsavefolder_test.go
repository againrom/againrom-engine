package game

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/render/terrain"
	"againrom/pkg/vfs"
)

// saveFolderFixture is an editor over a synthetic install whose executable
// sits inside it, so its player profile is Againrom/ in the install.
type saveFolderFixture struct {
	install string
	maps    string
	x       *MapInspector
	edits   *mapInspectionEdits
}

func newSaveFolderFixture(t *testing.T, language string) *saveFolderFixture {
	t.Helper()
	install := profileInstall(t)
	if resolved, err := filepath.EvalSymlinks(install); err == nil {
		install = resolved
	}
	exe := filepath.Join(install, "mapedit.exe")
	f := &saveFolderFixture{install: install, maps: filepath.Join(install, "Againrom", "maps")}
	for name, data := range map[string][]byte{
		ScenarioArchive: synth.Archive([]synth.File{{Path: "scenario.reg", Data: testPlayRegistry()}}),
		MainArchive:     synth.Archive([]synth.File{{Path: "text/battle/m20/event02.txt", Data: []byte("Page two")}}),
	} {
		if err := os.WriteFile(filepath.Join(install, name), data, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	containers, err := vfs.Open([]string{filepath.Join(install, ScenarioArchive), filepath.Join(install, MainArchive)}, nil)
	if err != nil {
		t.Fatal(err)
	}
	archives := &Archives{Root: install, Containers: containers}
	archives.Base.Profile.Language = language
	f.x = &MapInspector{tiles: &terrain.Tileset{}, archives: archives}
	f.x.SetTestPlay(TestPlayHost{Profile: func() (RuntimeProfile, error) {
		return ResolveRuntimeProfile("", exe, t.TempDir(), install)
	}})
	d, err := f.x.Open("new/32x32")
	if err != nil {
		t.Fatal(err)
	}
	f.edits = d.Edits.(*mapInspectionEdits)
	return f
}

// installFiles is every file under the install with its bytes, by path.
func installFiles(t *testing.T, dir string) map[string]string {
	t.Helper()
	out := map[string]string{}
	err := filepath.WalkDir(dir, func(p string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		b, err := os.ReadFile(p)
		out[p] = string(b)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	return out
}

// Save As into the profile's maps folder writes the map, though the profile
// sits inside the install; a relative path from the fallback prompt lands in
// the same folder.
func TestSaveAsWritesIntoTheProfileMapsFolderInsideTheInstall(t *testing.T) {
	f := newSaveFolderFixture(t, "")
	want, err := f.edits.model.SaveBytes()
	if err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(f.maps, "new.alm")
	got, err := f.edits.SaveAs(target)
	if err != nil {
		t.Fatalf("Save As into the maps folder: %v", err)
	}
	if b, err := os.ReadFile(target); err != nil || !bytes.Equal(b, want) || filepath.Clean(got) != target {
		t.Fatalf("maps/new.alm = %d bytes, %v, answered %s; want %d bytes", len(b), err, got, len(want))
	}
	if _, err := f.edits.SaveAs("typed.alm"); err != nil {
		t.Fatalf("a relative path: %v", err)
	}
	if _, err := os.Stat(filepath.Join(f.maps, "typed.alm")); err != nil {
		t.Fatalf("a relative path did not resolve against the maps folder: %v", err)
	}
	if _, ok := f.edits.OwnFile(); !ok {
		t.Fatal("a map saved in the maps folder has no file of its own to save over")
	}
	if _, err := f.edits.SaveInPlace(); err != nil {
		t.Fatalf("Save over a map in the maps folder: %v", err)
	}
}

// Every other folder in the install stays refused, with nothing written and no
// install file changed: the install, the profile beside the maps folder, the
// saves folder and a folder the install holds.
func TestSaveAsRefusesTheRestOfTheInstall(t *testing.T) {
	f := newSaveFolderFixture(t, "")
	if dir, _ := f.x.mapsFolder(); dir == "" {
		t.Fatal("no maps folder")
	}
	if err := os.Mkdir(filepath.Join(f.install, "sub"), 0o700); err != nil {
		t.Fatal(err)
	}
	before := installFiles(t, f.install)
	for _, dir := range []string{f.install, filepath.Join(f.install, "Againrom"), filepath.Join(f.install, "Againrom", "saves"), filepath.Join(f.install, "sub")} {
		if _, err := f.edits.SaveAs(filepath.Join(dir, "new.alm")); err == nil {
			t.Fatalf("Save As into %s was admitted", dir)
		}
	}
	if after := installFiles(t, f.install); len(after) != len(before) {
		t.Fatalf("install files %d before, %d after the refusals", len(before), len(after))
	} else {
		for p, b := range before {
			if after[p] != b {
				t.Fatalf("install file changed: %s", p)
			}
		}
	}
}

// A link from the maps folder into the install does not carry a write there.
func TestSaveAsRefusesALinkFromTheMapsFolderIntoTheInstall(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("the fixture link is a Windows junction")
	}
	f := newSaveFolderFixture(t, "")
	if dir, _ := f.x.mapsFolder(); dir == "" {
		t.Fatal("no maps folder")
	}
	sub := filepath.Join(f.install, "sub")
	if err := os.Mkdir(sub, 0o700); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(f.maps, "link")
	// Only joins two owned temporary fixture paths.
	if out, err := exec.Command("cmd", "/c", "mklink", "/J", link, sub).CombinedOutput(); err != nil {
		t.Fatalf("create temporary junction: %v %s", err, out)
	}
	defer os.Remove(link)
	if _, err := f.edits.SaveAs(filepath.Join(link, "new.alm")); err == nil {
		t.Fatal("a write through a link from the maps folder into the install was admitted")
	}
	if _, err := os.Stat(filepath.Join(sub, "new.alm")); !os.IsNotExist(err) {
		t.Fatalf("the refusal wrote into the install: %v", err)
	}
}

// The Save As window starts in the map's own folder when the editor may write
// there, else in the maps folder.
func TestSaveFolderForEachKindOfMap(t *testing.T) {
	f := newSaveFolderFixture(t, "")
	outside := t.TempDir()
	if resolved, err := filepath.EvalSymlinks(outside); err == nil {
		outside = resolved
	}
	for _, c := range []struct{ name, source, want string }{
		{"new map", "new/32x32", f.maps},
		{"archive map", "scenario/m10/map.alm", f.maps},
		{"loose map", "loose/maps/a.alm", f.maps},
		{"outside map", filepath.Join(outside, "a.alm"), outside},
		{"map in the install", filepath.Join(f.install, "a.alm"), f.maps},
		{"map in the maps folder", filepath.Join(f.maps, "a.alm"), f.maps},
	} {
		f.edits.source = c.source
		if got := f.edits.SaveFolder(); got != c.want {
			t.Errorf("%s: starts in %q, want %q", c.name, got, c.want)
		}
	}
	if _, err := os.Stat(f.maps); err != nil {
		t.Fatalf("the window did not make the maps folder: %v", err)
	}
}

// The refusal names the folder and the maps folder in the editor's language.
func TestSaveAsInstallRefusalReadsInTheEditorsLanguage(t *testing.T) {
	for _, c := range []struct{ language, head, mid string }{
		{"en", "Cannot save here: ", "Save the map in another folder, such as "},
		{"ru", "Сюда сохранить нельзя: ", "Сохраните карту в другую папку, например "},
	} {
		f := newSaveFolderFixture(t, c.language)
		_, err := f.edits.SaveAs(filepath.Join(f.install, "new.alm"))
		if err == nil {
			t.Fatal("Save As into the install was admitted")
		}
		msg := err.Error()
		if !strings.HasPrefix(msg, c.head+f.install) || !strings.Contains(msg, c.mid+f.maps+".") || strings.Contains(msg, "refusing to write") {
			t.Errorf("%s refusal = %q", c.language, msg)
		}
	}
}
