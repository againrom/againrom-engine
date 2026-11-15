package game

import (
	"bytes"
	"errors"
	"image/color"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/mod"
	"againrom/pkg/modrt"
	"againrom/pkg/ui"
	"againrom/pkg/vfs"
)

// The files the fixture install holds that a mod may replace.
const (
	filesPicture = "graphics/pic/a.bmp"
	filesText    = "main/text/battle/m20/event02.txt"
	filesSound   = "sfx/x.wav"
)

// filesPage is W open on a new mod made in mods/authored-map, on an install
// whose archives hold a picture, a text and a sound, and whose starter
// enables the ids enabled names.
func filesPage(t *testing.T, enabled *[]string) (*mapInspectionEdits, string) {
	t.Helper()
	x, s := contextEditor(t)
	root := x.archives.Root
	writeFileAll(t, filepath.Join(root, GraphicsArchive), synth.Archive([]synth.File{{Path: "pic/a.bmp", Data: filesBMP(4, 3)}}))
	writeFileAll(t, filepath.Join(root, SfxArchive), synth.Archive([]synth.File{{Path: "x.wav", Data: modToneWAV()}}))
	containers, err := vfs.Open([]string{filepath.Join(root, ScenarioArchive), filepath.Join(root, MainArchive), filepath.Join(root, GraphicsArchive)}, nil)
	if err != nil {
		t.Fatal(err)
	}
	x.archives.Containers = containers
	withInstallTable(x)
	mods := t.TempDir()
	host := *x.testPlay
	host.ModsFolder, host.EnabledMods = mods, func() []string { return *enabled }
	x.SetTestPlay(host)
	dir := filepath.Join(mods, "authored-map")
	d, err := s.SetWorkspaceField("package", "folder", dir)
	if err != nil || d == nil {
		t.Fatalf("make %s: %v", dir, err)
	}
	return d.Edits.(*mapInspectionEdits), dir
}

// filesBMP is a 24-bit picture of w by h pixels.
func filesBMP(w, h int) []byte {
	return synth.BMP24(solidPicture(w, h, color.RGBA{R: 10, G: 20, B: 30, A: 255}))
}

// filesField is the files page's field key; false when the page has none.
func filesField(s *mapInspectionEdits, key string) (ui.InspectionField, bool) {
	for _, f := range s.WorkspaceFields(ui.ModFilesScope) {
		if f.Key == key {
			return f, true
		}
	}
	return ui.InspectionField{}, false
}

// importFile imports data as the replacement of address in language folder
// lang through the page's field.
func importFile(s *mapInspectionEdits, lang, address string, data []byte, t *testing.T) error {
	t.Helper()
	source, _ := assetFile(t, "replacement"+filepath.Ext(address), data)
	_, err := s.SetWorkspaceField(ui.ModFilesScope, lang+":"+address, source)
	return err
}

// The page lists the install's paths by the filter with an import per
// language folder, takes a replacement the game serves and refuses one by
// each rule in that rule's words, leaving the project as it was.
func TestFilesPageImportsByTheRulesTheGameServesBy(t *testing.T) {
	enabled := []string{}
	s, _ := filesPage(t, &enabled)
	for _, key := range []string{"common:" + filesPicture, "en:" + filesText, "ru:" + filesSound} {
		f, ok := filesField(s, key)
		if !ok || f.Group != key[strings.Index(key, ":")+1:] || len(f.Pick) != 1 {
			t.Fatalf("field %s: %+v %v", key, f, ok)
		}
	}
	if _, err := s.SetWorkspaceField(ui.ModFilesScope, "filter", ".WAV"); err != nil {
		t.Fatal(err)
	}
	if _, ok := filesField(s, "common:"+filesPicture); ok {
		t.Fatal("the filter .WAV lists the picture")
	}
	if _, ok := filesField(s, "common:"+filesSound); !ok {
		t.Fatal("the filter .WAV hides the sound")
	}
	if _, err := s.SetWorkspaceField(ui.ModFilesScope, "filter", ""); err != nil {
		t.Fatal(err)
	}
	big := filepath.Join(t.TempDir(), "big.bmp")
	if err := os.WriteFile(big, make([]byte, mod.MaxInstallFileBytes+1), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct {
		name, lang, address string
		data                []byte
		want                string
	}{
		{"picture size", "common", filesPicture, filesBMP(5, 3), "the picture is 5x3; the install's is 4x3"},
		{"picture bytes", "common", filesPicture, []byte("not a picture"), "the game cannot decode this picture"},
		{"sound", "common", filesSound, []byte("not a sound"), "the game cannot decode this sound"},
		{"alphabet", "en", filesText, []byte("日本"), "the install's alphabet has no"},
		{"utf-8", "en", filesText, []byte{0xff, 0xfe}, "is not UTF-8 text"},
		{"absent path", "common", "graphics/pic/none.bmp", filesBMP(4, 3), "the install holds no file graphics/pic/none.bmp"},
		{"kind", "common", "graphics/pic/a.png", []byte("png"), "the game takes no install file of this kind"},
		{"folder", "xx", filesPicture, filesBMP(4, 3), "the files page has no field xx:" + filesPicture},
	} {
		err := importFile(s, c.lang, c.address, c.data, t)
		if err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%s: %v, want %q", c.name, err, c.want)
		}
	}
	if _, err := s.SetWorkspaceField(ui.ModFilesScope, "common:"+filesPicture, big); err == nil || !strings.Contains(err.Error(), strconv.Itoa(mod.MaxInstallFileBytes)) {
		t.Errorf("size bound: %v", err)
	}
	if p := s.workspace.Project; p != nil && slices.ContainsFunc(s.projectFiles(p), func(n string) bool { return strings.HasPrefix(n, mod.FilesDir+"/") }) {
		t.Fatalf("a refused import entered the project: %v", s.projectFiles(p))
	}
	if err := importFile(s, "common", filesPicture, filesBMP(4, 3), t); err != nil {
		t.Fatal(err)
	}
	if f, _ := filesField(s, "common:"+filesPicture); !strings.Contains(f.Meaning, "files/common/"+filesPicture) {
		t.Fatalf("the replaced path's field says %q", f.Meaning)
	}
	if _, ok := filesField(s, filesDropKey+"files/common/"+filesPicture); !ok {
		t.Fatal("the page has no removal of the replacement")
	}
	var ask *ui.WorkspaceConfirmation
	if err := importFile(s, "common", filesPicture, filesBMP(4, 3), t); !errors.As(err, &ask) {
		t.Fatalf("a second import of the path: %v, want the replace question", err)
	}
}

// A replacement goes to its language folder, the common folder's and each
// language's apart; one for a language not the install's is held to the
// folder's rules and says so. Removal, Undo and Redo are draft steps, and
// an Undo past a save is saved as a removal.
func TestFilesPagePlacesByLanguageAndUndoesPastASave(t *testing.T) {
	enabled := []string{}
	s, dir := filesPage(t, &enabled)
	if err := importFile(s, "en", filesText, []byte("Page two, English"), t); err != nil {
		t.Fatal(err)
	}
	if err := importFile(s, "ru", filesText, []byte("Страница два"), t); err != nil {
		t.Fatal(err)
	}
	if err := importFile(s, "common", filesSound, modToneWAV(), t); err != nil {
		t.Fatal(err)
	}
	if f, _ := filesField(s, "ru:"+filesText); !strings.Contains(f.Meaning, "RU install checks it") {
		t.Fatalf("the Russian field says %q", f.Meaning)
	}
	if _, err := s.SaveIntoMod(); err != nil {
		t.Fatal(err)
	}
	saved := tree(t, dir)
	for rel, want := range map[string]string{"files/en/" + filesText: "Page two, English", "files/ru/" + filesText: "Страница два", "files/common/" + filesSound: string(modToneWAV())} {
		if saved[rel] != want {
			t.Fatalf("saved %s = %q, want %q", rel, saved[rel], want)
		}
	}
	ruFile := "files/ru/" + filesText
	if _, err := s.SetWorkspaceField(ui.ModFilesScope, filesDropKey+ruFile, ""); err != nil {
		t.Fatal(err)
	}
	if _, ok := filesField(s, filesDropKey+ruFile); ok {
		t.Fatal("the removed file is still listed as held")
	}
	if _, err := s.Undo(); err != nil {
		t.Fatal(err)
	}
	if _, ok := filesField(s, filesDropKey+ruFile); !ok {
		t.Fatal("Undo did not bring the removed file back")
	}
	if _, err := s.Redo(); err != nil {
		t.Fatal(err)
	}
	// Undo past the save takes the sound and the texts out again; the next
	// save removes them from the folder.
	for range 4 {
		if _, err := s.Undo(); err != nil {
			t.Fatal(err)
		}
	}
	if _, ok := filesField(s, filesDropKey+"files/common/"+filesSound); ok {
		t.Fatal("Undo past the save kept the sound")
	}
	if _, err := s.SaveIntoMod(); err != nil {
		t.Fatal(err)
	}
	for rel := range tree(t, dir) {
		if strings.HasPrefix(rel, mod.FilesDir+"/") {
			t.Fatalf("the save after Undo kept %s", rel)
		}
	}
}

// The page's override lines are the lines the mod check prints, the
// install files of the enabled mods served by the check's rule over the
// install: a path two mods replace names both.
func TestFilesPageOverridesAreTheChecksLines(t *testing.T) {
	enabled := []string{"authored-map", "later-art"}
	s, dir := filesPage(t, &enabled)
	mods := filepath.Dir(dir)
	writeModText(t, filepath.Join(mods, "later-art"), "mod.toml", "id = \"later-art\"\ntitle = \"later-art\"\nversion = \"1.0\"\nload-after = [\"authored-map\"]\n")
	writeFileAll(t, filepath.Join(mods, "later-art", "files", "common", filepath.FromSlash(filesPicture)), filesBMP(4, 3))
	if err := importFile(s, "common", filesPicture, filesBMP(4, 3), t); err != nil {
		t.Fatal(err)
	}
	want := `override: install file graphics/pic/a.bmp: mod "later-art" (files/common/graphics/pic/a.bmp) overrides mod "authored-map" (files/common/graphics/pic/a.bmp)`
	if got := pageOverrides(s); !slices.Equal(got, []string{want}) {
		t.Fatalf("the draft's overrides %q, want %q", got, want)
	}
	if _, err := s.SaveIntoMod(); err != nil {
		t.Fatal(err)
	}
	x := s.inspector
	entries, err := mod.Resolve(mods, enabled)
	if err != nil {
		t.Fatal(err)
	}
	res, err := modrt.Load(entries, BaseID(InspectInstall(x.archives.Root)), nil, modrt.Options{})
	if err != nil {
		t.Fatal(err)
	}
	b, _ := checkServes(t, x, res)
	var check []string
	for _, o := range b.overrides {
		check = append(check, "override: "+o)
	}
	if got := pageOverrides(s); len(check) != 1 || !slices.Equal(got, check) {
		t.Fatalf("the page's overrides %q, the check's %q", got, check)
	}
}

// pageOverrides is the override lines the files page shows.
func pageOverrides(s *mapInspectionEdits) []string {
	var out []string
	for _, f := range s.WorkspaceFields(ui.ModFilesScope) {
		if strings.HasPrefix(f.Key, "override.") {
			out = append(out, f.TextValue)
		}
	}
	return out
}

// checkServes serves res's install files over the install of x as the
// launch's share serves them: its own layer over the archives and the sound
// archive opened from the root.
func checkServes(t *testing.T, x *MapInspector, res modrt.Result) (*modFiles, *Archives) {
	t.Helper()
	a := *x.archives
	a.Containers = a.Containers.Fork()
	sounds, err := vfs.Open([]string{filepath.Join(a.Root, SfxArchive)}, nil)
	if err != nil {
		t.Fatal(err)
	}
	b := &modFiles{fs: a.Containers, sounds: sounds, order: res.Files.Order, owners: map[string]modFileEntry{}}
	a.mods = b
	if err := b.serveInstallFiles(&a, res.Files.Files); err != nil {
		t.Fatalf("the check refused the mod: %v", err)
	}
	return b, &a
}

// A mod made only through the page saves, packs, passes the check and has
// the game's file layer serve its replacements.
func TestFilesPageModSavesPublishesAndServes(t *testing.T) {
	enabled := []string{}
	s, _ := filesPage(t, &enabled)
	picture, sound, text := filesBMP(4, 3), modToneWAV(), []byte("Page two, replaced")
	for _, c := range []struct {
		lang, address string
		data          []byte
	}{{"common", filesPicture, picture}, {"common", filesSound, sound}, {"en", filesText, text}} {
		if err := importFile(s, c.lang, c.address, c.data, t); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := s.SaveIntoMod(); err != nil {
		t.Fatal(err)
	}
	out := t.TempDir()
	if _, err := s.SetWorkspaceField("package", "output", out); err != nil {
		t.Fatal(err)
	}
	if _, err := s.SetWorkspaceField("package", "publish", ""); err != nil {
		t.Fatal(err)
	}
	entries, err := mod.Resolve(out, []string{"authored-map"})
	if err != nil || len(entries) != 1 || filepath.Ext(entries[0].Dir) != ".res" {
		t.Fatal(entries, err)
	}
	x := s.inspector
	res, err := modrt.Load(entries, BaseID(InspectInstall(x.archives.Root)), nil, modrt.Options{})
	if err != nil || len(res.Files.Files) != 3 {
		t.Fatalf("the packed mod: %v, %d files", err, len(res.Files.Files))
	}
	b, a := checkServes(t, x, res)
	for address, want := range map[string][]byte{filesPicture: picture, filesText: text} {
		if got, err := a.Containers.ReadFile(address); err != nil || !bytes.Equal(got, want) {
			t.Errorf("the layer serves %s as %q, %v", address, got, err)
		}
	}
	if got, err := b.sounds.ReadFile(filesSound); err != nil || !bytes.Equal(got, sound) {
		t.Errorf("the layer serves %s as %d bytes, %v", filesSound, len(got), err)
	}
}
