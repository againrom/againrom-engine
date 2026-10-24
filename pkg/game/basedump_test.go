package game

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	"againrom/pkg/mod"
	"againrom/pkg/town"
)

// dumpFront is layerFront over a plain folder standing for the install, its
// home title TOWN.
func dumpFront(t *testing.T) *FrontEnd {
	t.Helper()
	f := layerFront(t)
	f.Archives.Root = t.TempDir()
	f.Words.WorldHomeTitle = "TOWN"
	return f
}

// The base dump of the first game is a mod folder: its manifest, an entry
// script that adds the towns file, and the towns file with town 1 at its
// catalog values, named with the home title. Layered over the shipped
// catalog it changes nothing but the name the home scroll already draws;
// one edited key shows.
func TestBaseDumpLayersOverTheShippedCatalogUnchanged(t *testing.T) {
	f := dumpFront(t)
	mods := t.TempDir()
	report, err := f.WriteBaseDump(filepath.Join(mods, "base-dump"))
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"mod.toml", "main.star", "data/towns.toml"}; !slices.Equal(report.Files, want) || len(report.Notes) != 0 {
		t.Fatalf("report %+v, want files %v and no note", report, want)
	}
	towns, err := os.ReadFile(filepath.Join(mods, "base-dump", "data", "towns.toml"))
	if err != nil {
		t.Fatal(err)
	}
	script, _ := os.ReadFile(filepath.Join(mods, "base-dump", "main.star"))
	if !strings.Contains(string(script), `game.data.add("data/towns.toml")`) {
		t.Fatalf("main.star:\n%s", script)
	}
	shipped := townBooks["rom1"].catalog.Towns
	view := func(g *FrontEnd) (town.Entry, string) {
		t.Helper()
		e, _ := g.Campaign.Value().Towns.catalog.Town(1)
		g.Town.Arrive()
		s := g.TownScreen().(*townScreen)
		s.enterWorldMap()
		v := s.WorldMapView()
		if v.Problem != "" {
			t.Fatalf("world map: %s", v.Problem)
		}
		return e, v.Words.WorldHomeTitle
	}
	_, plainTitle := view(dumpFront(t))
	if plainTitle != "TOWN" {
		t.Fatalf("the home scroll reads %q without a mod", plainTitle)
	}
	layered := func(towns string) (town.Entry, string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(mods, "base-dump", "data", "towns.toml"), []byte(towns), 0o644); err != nil {
			t.Fatal(err)
		}
		g := dumpFront(t)
		overrides, err := g.SetModTowns(runTownModsFor(t, report.Base, mods, "base-dump").Towns)
		if err != nil || len(overrides) != 0 {
			t.Fatalf("the dump as a mod: %v, overrides %v", err, overrides)
		}
		return view(g)
	}
	e, title := layered(string(towns))
	want := shipped[0]
	want.Name = "TOWN"
	if len(shipped) != 1 || !sameEntryButLists(e, want) || title != plainTitle {
		t.Fatalf("town 1 under the dump: %+v titled %q, want %+v titled %q", e, title, want, plainTitle)
	}
	if _, title := layered(strings.Replace(string(towns), `name        = "TOWN"`, `name        = "Dumped"`, 1)); title != "Dumped" {
		t.Fatalf("an edited name: the home scroll reads %q", title)
	}
}

// sameEntryButLists reports whether a and b differ at most by an empty list
// standing for none.
func sameEntryButLists(a, b town.Entry) bool {
	for _, e := range []*town.Entry{&a, &b} {
		if len(e.Missions) == 0 {
			e.Missions = nil
		}
	}
	return reflect.DeepEqual(a, b)
}

// The refusals: a folder inside the install, below a folder holding an
// install's archives, a folder that is not empty, an existing file and a
// folder name that is not a mod id. Each names its reason and writes
// nothing.
func TestBaseDumpRefusals(t *testing.T) {
	f := dumpFront(t)
	other := t.TempDir()
	for _, name := range []string{MainArchive, GraphicsArchive, ScenarioArchive, WorldArchive, MoviesArchive} {
		if err := os.WriteFile(filepath.Join(other, name), nil, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	full := t.TempDir()
	if err := os.WriteFile(filepath.Join(full, "keep.txt"), []byte("keep"), 0o644); err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(t.TempDir(), "base-dump")
	if err := os.WriteFile(file, []byte("keep"), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct {
		dir  string
		want error
	}{
		{filepath.Join(f.Archives.Root, "base-dump"), ErrBaseDumpInstall},
		{f.Archives.Root, ErrBaseDumpInstall},
		{filepath.Join(other, "mods", "base-dump"), ErrBaseDumpInstall},
		{full, ErrBaseDumpNotEmpty},
		{file, ErrBaseDumpExists},
		{filepath.Join(t.TempDir(), "Base Dump"), ErrBaseDumpName},
	} {
		before := treeOf(t, filepath.Dir(c.dir))
		if _, err := f.WriteBaseDump(c.dir); !errors.Is(err, c.want) || !strings.Contains(err.Error(), c.dir) {
			t.Errorf("%s: %v, want %v naming the path", c.dir, err, c.want)
		}
		if after := treeOf(t, filepath.Dir(c.dir)); !slices.Equal(before, after) {
			t.Errorf("%s: the refusal changed %v into %v", c.dir, before, after)
		}
	}
	if b, _ := os.ReadFile(file); string(b) != "keep" {
		t.Fatal("the existing file was overwritten")
	}
}

// A file already in place when the files are written is refused, and the
// files and folders written before it are removed.
func TestBaseDumpWriteKeepsAnExistingFile(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "data"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "data", "towns.toml"), []byte("keep"), 0o644); err != nil {
		t.Fatal(err)
	}
	before := treeOf(t, dir)
	files := map[string][]byte{"mod.toml": []byte("id"), "x/y.toml": []byte("y"), "data/towns.toml": []byte("new")}
	if err := writeNewFiles(dir, false, []string{"mod.toml", "x/y.toml", "data/towns.toml"}, files); !errors.Is(err, ErrBaseDumpExists) {
		t.Fatalf("got %v, want %v", err, ErrBaseDumpExists)
	}
	if after := treeOf(t, dir); !slices.Equal(before, after) {
		t.Fatalf("the refused write left %v, want %v", after, before)
	}
	if b, _ := os.ReadFile(filepath.Join(dir, "data", "towns.toml")); string(b) != "keep" {
		t.Fatal("the existing file was overwritten")
	}
}

// treeOf lists every path under dir with its kind and size, or names dir
// missing.
func treeOf(t *testing.T, dir string) []string {
	t.Helper()
	if _, err := os.Lstat(dir); errors.Is(err, os.ErrNotExist) {
		return []string{dir + " missing"}
	}
	var out []string
	err := filepath.WalkDir(dir, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		out = append(out, fmt.Sprintf("%s dir=%v size=%d", p, d.IsDir(), info.Size()))
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return out
}

// The second game's dump lists its three towns with every key but name,
// says that the second game reads no towns layer, and its entry script adds
// no file.
func TestBaseDumpOfTheSecondGame(t *testing.T) {
	f := dumpFront(t)
	f.Archives.Base.Profile.Game = "rom2"
	layer, err := f.baseTownLayer("rom2-en")
	if err != nil {
		t.Fatal(err)
	}
	records, _, err := mod.ParseTowns("dump", mod.TownsFile, layer.data)
	if err != nil {
		t.Fatal(err)
	}
	var ids []int
	for _, r := range records {
		ids = append(ids, r.Entry.ID)
	}
	if layer.read || !slices.Equal(ids, []int{1, 2, 3}) || !strings.Contains(layer.note, "the second game reads no towns layer") ||
		!strings.Contains(string(layer.data), "# "+layer.note) {
		t.Fatalf("read %v towns %v note %q:\n%s", layer.read, ids, layer.note, layer.data)
	}
	if script := string(baseDumpEntry(nil, []string{mod.TownsFile})); strings.Contains(script, "game.data.add") || !strings.Contains(script, "pass") {
		t.Fatalf("main.star:\n%s", script)
	}
}

// decodeSaveLabel reads install text back to the UTF-8 encodeSaveLabel
// takes, and refuses a byte whose letter encodes to another byte.
func TestDecodeSaveLabelIsTheEncodersInverse(t *testing.T) {
	if s, ok := decodeSaveLabel("\x87\xa0\xaf\xa0\xa4", 1); !ok || s != "Запад" {
		t.Fatalf("got %q %v, want Запад", s, ok)
	}
	if s, ok := decodeSaveLabel("TOWN", 0); !ok || s != "TOWN" {
		t.Fatalf("got %q %v", s, ok)
	}
	for _, b := range []string{"\xc0", "\x01"} {
		if s, ok := decodeSaveLabel(b, 1); ok {
			t.Errorf("% x reads back as %q", b, s)
		}
	}
}
