package mod

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"againrom/pkg/screenart"
	"againrom/pkg/town"
)

func TestTownArtReferencesFollowEntryKeys(t *testing.T) {
	entries := []town.ArtSpec{
		{Name: "series", Format: "series", Key: "mod:./art/s%02d.bmp", Count: 2, First: 3},
		{Name: "indexed", Format: "picture", Key: "mod:art/i%02d.bmp", Index: []screenart.Index{{Values: []int{2, 5}}}},
		{Name: "literal", Format: "picture", Key: "mod:art/z%d.bmp"},
		{Name: "duplicate", Format: "picture", Key: "mod:art/z%d.bmp"},
		{Name: "install", Format: "picture", Key: "graphics/a.bmp"},
	}
	data, err := json.Marshal(map[string]any{"art": entries})
	if err != nil {
		t.Fatal(err)
	}
	d, err := town.Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	want := []TownArtFile{
		{"mod:./art/s03.bmp", "art/s03.bmp"}, {"mod:./art/s04.bmp", "art/s04.bmp"},
		{"mod:art/i02.bmp", "art/i02.bmp"}, {"mod:art/i05.bmp", "art/i05.bmp"},
		{"mod:art/z%d.bmp", "art/z%d.bmp"},
	}
	got, err := TownArtFiles(d)
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Fatalf("art files %+v: %v, want %+v", got, err, want)
	}
	files := map[string]string{
		ManifestName: good("townart"), TownsFile: "[[town]]\nid = 9\ndescription = \"towns/t.json\"\n",
		"towns/t.json": string(data), "art/s00.bmp": "stray", "art/s05.bmp": "stray", "art/i03.bmp": "stray",
	}
	for i, f := range want {
		files[f.Path] = fmt.Sprintf("picture %d", i)
	}
	dir := filepath.Join(t.TempDir(), "townart")
	for file, data := range files {
		writeTreeFile(t, dir, file, data)
	}
	packed := packInto(t, dir, t.TempDir())
	for _, source := range []string{dir, packed} {
		tree, err := OpenTree(source)
		if err != nil {
			t.Fatal(err)
		}
		description, err := tree.Describe()
		if err != nil {
			t.Fatal(err)
		}
		for _, f := range want {
			if !description.Admits(f.Path) {
				t.Errorf("%s excludes %s", source, f.Path)
			}
			if b, _, err := tree.ReadInside(strings.TrimPrefix(f.Key, ModArtPrefix)); err != nil || string(b) != files[f.Path] {
				t.Errorf("%s art %s: %q: %v", source, f.Key, b, err)
			}
		}
		for _, stray := range []string{"art/s00.bmp", "art/s05.bmp", "art/i03.bmp"} {
			if description.Admits(stray) {
				t.Errorf("%s admits unreferenced series/index member %s", source, stray)
			}
		}
	}
}

func TestTownArtReferencesKeepReaderRefusals(t *testing.T) {
	for _, tc := range []struct {
		entry town.ArtSpec
		want  string
	}{
		{town.ArtSpec{Name: "bound", Key: "mod:art/%d.bmp", Bound: true}, "is bound at run time"},
		{town.ArtSpec{Name: "outside", Key: "mod:../outside.bmp"}, "is not a path inside the mod folder"},
		{town.ArtSpec{Name: "hidden", Key: "mod:.hidden/a.bmp"}, "a name starting with a dot is not read from a mod"},
	} {
		t.Run(tc.entry.Name, func(t *testing.T) {
			d := &town.Description{Art: []town.ArtSpec{tc.entry}}
			_, err := TownArtFiles(d)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("refusal %v, want %s", err, tc.want)
			}
		})
	}
}
