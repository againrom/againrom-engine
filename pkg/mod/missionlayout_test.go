package mod

import (
	"bytes"
	"maps"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"testing"
)

// layoutCase is a mod in one layout: its mission table and its files.
type layoutCase struct {
	src      string
	files    map[string]string
	extended map[string]bool
	moves    []FileMove
	removed  []string
	rows     string // the table after, or "" when it does not change
}

func writeLayoutFiles(t *testing.T, dir string, files map[string]string) {
	t.Helper()
	for name, body := range files {
		full := filepath.Join(dir, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

// readLayoutMissions is the missions of src as the game reads them from dir
// in lang: map bytes, pages and the pages taken from the other language.
func readLayoutMissions(t *testing.T, dir string, src []byte, lang string) []Mission {
	t.Helper()
	d, err := ParseMissions("m", MissionsFile, src, missionText, lang)
	if err != nil {
		t.Fatal(err)
	}
	for i := range d.Missions {
		if err := d.Missions[i].ReadFiles(dir); err != nil {
			t.Fatalf("%s %s: %v", lang, d.Missions[i].Key, err)
		}
	}
	return d.Missions
}

func runLayoutCase(t *testing.T, tc layoutCase) {
	t.Helper()
	dir := t.TempDir()
	writeLayoutFiles(t, dir, tc.files)
	before := map[string][]Mission{}
	for _, lang := range layoutLanguages() {
		before[lang] = readLayoutMissions(t, dir, []byte(tc.src), lang)
	}
	d, err := ParseMissions("m", MissionsFile, []byte(tc.src), missionText, "en")
	if err != nil {
		t.Fatal(err)
	}
	c := TreeLayout(d.Missions, slices.Sorted(maps.Keys(tc.files)), tc.extended)
	if !slices.Equal(c.Moves, tc.moves) || !slices.Equal(c.Removed, tc.removed) {
		t.Fatalf("moves %v removed %v\nwant %v removed %v", c.Moves, c.Removed, tc.moves, tc.removed)
	}
	out, err := WriteMissions([]byte(tc.src), d.Missions, c.Missions)
	if err != nil {
		t.Fatal(err)
	}
	want := tc.rows
	if want == "" {
		want = tc.src
	}
	if string(out) != want {
		t.Fatalf("table after:\n%s\nwant:\n%s", out, want)
	}
	data := map[string][]byte{}
	for _, mv := range c.Moves {
		b, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(mv.From)))
		if err != nil {
			t.Fatal(err)
		}
		data[mv.To] = b
	}
	for _, name := range c.Removed {
		if err := os.Remove(filepath.Join(dir, filepath.FromSlash(name))); err != nil {
			t.Fatal(err)
		}
	}
	for name, b := range data {
		writeLayoutFiles(t, dir, map[string]string{name: string(b)})
	}
	for _, lang := range layoutLanguages() {
		after := readLayoutMissions(t, dir, out, lang)
		for i, m := range after {
			b := before[lang][i]
			if !bytes.Equal(m.MapBytes, b.MapBytes) || !reflect.DeepEqual(m.TextFiles, b.TextFiles) || !reflect.DeepEqual(m.OtherLanguageTexts, b.OtherLanguageTexts) {
				t.Fatalf("%s %s reads other files after the layout change: map %q pages %v other %v; before map %q pages %v other %v",
					lang, m.Key, m.MapBytes, m.TextFiles, m.OtherLanguageTexts, b.MapBytes, b.TextFiles, b.OtherLanguageTexts)
			}
		}
	}
}

func TestTreeLayoutMovesTheEditorPackageForm(t *testing.T) {
	runLayoutCase(t, layoutCase{
		src: "[[mission]]\nkind = \"replace\"\nkey = \"map\"\nnumber = 10\nmap = \"mission/map.alm\"\ntexts = \"mission/text/{lang}\"\n",
		files: map[string]string{
			"mission/map.alm": "map", "mission/map.variables": "{}", "mission/text/en/event01.txt": "en one",
			"mission/text/ru/event01.txt": "ru one", "mission/text/en/event02.txt": "en two", "mission/text/shared.txt": "shared",
		},
		moves: []FileMove{
			{"mission/map.alm", "maps/map.alm"},
			{"mission/text/en/event01.txt", "maps/map.text/en/event01.txt"}, {"mission/text/en/event02.txt", "maps/map.text/en/event02.txt"},
			{"mission/text/ru/event01.txt", "maps/map.text/ru/event01.txt"}, {"mission/text/shared.txt", "maps/map.text/shared.txt"},
			{"mission/map.variables", "maps/map.variables"},
		},
		removed: []string{"mission/map.alm", "mission/map.variables", "mission/text/en/event01.txt", "mission/text/en/event02.txt", "mission/text/ru/event01.txt", "mission/text/shared.txt"},
		rows:    "[[mission]]\nkind = \"replace\"\nkey = \"map\"\nnumber = 10\nmap = \"maps/map.alm\"\n",
	})
}

func TestTreeLayoutMovesTheAddedMapForm(t *testing.T) {
	runLayoutCase(t, layoutCase{
		src: "[[mission]]\nkind = \"replace\"\nkey = \"map\"\nnumber = 10\nmap = \"mission/map.alm\"\n" +
			"\n[[mission]]\nkind = \"replace\"\nkey = \"map22\"\nnumber = 22\nmap = \"mission/map22/map.alm\"\ntexts = \"mission/map22/text/{lang}\"\n",
		files: map[string]string{
			"mission/map.alm": "first", "maps/map.text/en/note.txt": "taken stem",
			"mission/map22/map.alm": "second", "mission/map22/text/en/event01.txt": "en", "mission/map22/map.text/en/event01.txt": "beside wins",
		},
		moves: []FileMove{
			{"mission/map.alm", "maps/map-2.alm"}, {"mission/map22/map.alm", "maps/map-3.alm"},
			{"mission/map22/map.text/en/event01.txt", "maps/map-3.text/en/event01.txt"},
		},
		removed: []string{"mission/map.alm", "mission/map22/map.alm", "mission/map22/map.text/en/event01.txt"},
		rows: "[[mission]]\nkind = \"replace\"\nkey = \"map\"\nnumber = 10\nmap = \"maps/map-2.alm\"\n" +
			"\n[[mission]]\nkind = \"replace\"\nkey = \"map22\"\nnumber = 22\nmap = \"maps/map-3.alm\"\n",
	})
}

func TestTreeLayoutMovesTheAssembledForm(t *testing.T) {
	src := "[[mission]]\nkind = \"replace\"\nkey = \"prologue\"\nnumber = 10\nmap = \"mission/{lang}/map.alm\"\ntexts = \"mission/{lang}/text\"\n" +
		"\n[[mission]]\nkind = \"replace\"\nkey = \"again\"\nnumber = 11\nmap = \"mission/{lang}/map.alm\"\ntexts = \"mission/{lang}/text\"\n"
	runLayoutCase(t, layoutCase{
		src: src,
		files: map[string]string{
			"mission/en/map.alm": "en map", "mission/ru/map.alm": "ru map",
			"mission/en/text/event01.txt": "en one", "mission/en/text/event02.txt": "en two", "mission/ru/text/event01.txt": "ru one",
		},
		moves: []FileMove{
			{"mission/en/map.alm", "maps/en/map.alm"}, {"mission/en/text/event01.txt", "maps/en/map.text/en/event01.txt"},
			{"mission/en/text/event02.txt", "maps/en/map.text/en/event02.txt"},
			{"mission/ru/map.alm", "maps/ru/map.alm"}, {"mission/ru/text/event01.txt", "maps/ru/map.text/ru/event01.txt"},
		},
		removed: []string{"mission/en/map.alm", "mission/en/text/event01.txt", "mission/en/text/event02.txt", "mission/ru/map.alm", "mission/ru/text/event01.txt"},
		rows: "[[mission]]\nkind = \"replace\"\nkey = \"prologue\"\nnumber = 10\nmap = \"maps/{lang}/map.alm\"\ntexts = \"maps/{lang}/map.text/{lang}\"\n" +
			"\n[[mission]]\nkind = \"replace\"\nkey = \"again\"\nnumber = 11\nmap = \"maps/{lang}/map.alm\"\ntexts = \"maps/{lang}/map.text/{lang}\"\n",
	})
}

func TestTreeLayoutKeepsATreeModAndRenamesAnExtendedMap(t *testing.T) {
	src := "[[mission]]\nkind = \"replace\"\nkey = \"crypt\"\nnumber = 10\nmap = \"maps/crypt.alm\"\ntexts = \"maps/crypt.text/{lang}\"\n"
	files := map[string]string{"maps/crypt.alm": "map", "maps/crypt.text/en/event01.txt": "en", "maps/crypt.variables": "{}"}
	runLayoutCase(t, layoutCase{src: src, files: files})
	runLayoutCase(t, layoutCase{
		src: src, files: files, extended: map[string]bool{"crypt": true},
		moves:   []FileMove{{"maps/crypt.alm", "maps/crypt.almx"}},
		removed: []string{"maps/crypt.alm"},
		rows:    "[[mission]]\nkind = \"replace\"\nkey = \"crypt\"\nnumber = 10\nmap = \"maps/crypt.almx\"\ntexts = \"maps/crypt.text/{lang}\"\n",
	})
}

func TestTreeLayoutLeavesRowsThatShareAMapButNotItsPages(t *testing.T) {
	runLayoutCase(t, layoutCase{
		src: "[[mission]]\nkind = \"replace\"\nkey = \"a\"\nnumber = 10\nmap = \"old/map.alm\"\ntexts = \"old/a\"\n" +
			"\n[[mission]]\nkind = \"replace\"\nkey = \"b\"\nnumber = 11\nmap = \"old/map.alm\"\ntexts = \"old/b\"\n",
		files: map[string]string{"old/map.alm": "map", "old/a/event01.txt": "a", "old/b/event01.txt": "b"},
	})
}
