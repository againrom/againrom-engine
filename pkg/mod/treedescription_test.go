package mod

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"againrom/pkg/formats/res"
	"againrom/pkg/locale"
)

func TestFixedTreeLanguagesFollowTheLocaleTable(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "languages")
	writeTreeFile(t, dir, ManifestName, good("languages"))
	writeTreeFile(t, dir, MissionsFile, "[[mission]]\nmap = \"old/{lang}/map.alm\"\n")
	var admitted []string
	for _, l := range locale.All() {
		for _, p := range []string{
			StringsFile(l.Code), "maps/" + l.Code + "/a.alm", "files/" + l.Code + "/main/title.txt", "old/" + l.Code + "/map.alm",
		} {
			writeTreeFile(t, dir, p, "held")
			admitted = append(admitted, p)
		}
	}
	unknown := "unknown-language"
	if locale.Known(unknown) {
		t.Fatal("the negative language control is known")
	}
	rejected := []string{StringsFile(unknown), "maps/" + unknown + "/a.alm", "files/" + unknown + "/main/title.txt"}
	for _, p := range rejected {
		writeTreeFile(t, dir, p, "stray")
	}
	common := "files/" + CommonFiles + "/main/title.txt"
	writeTreeFile(t, dir, common, "common")
	admitted = append(admitted, common)
	tree, _ := OpenTree(dir)
	d, err := tree.Describe()
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range admitted {
		if !d.Admits(p) {
			t.Errorf("excluded %s", p)
		}
	}
	for _, p := range rejected {
		if d.Admits(p) {
			t.Errorf("admitted %s", p)
		}
	}
	if len(d.Warnings) != len(rejected)+1 {
		t.Fatalf("warnings = %v, want unknown language files and the legacy map path", d.Warnings)
	}
}

func TestMapLabelFileKeepsTheMapsStem(t *testing.T) {
	for _, tc := range []struct{ mapPath, want string }{
		{"maps/a.alm", "maps/a.variables"},
		{"maps/sub.dir/a.almx", "maps/sub.dir/a.variables"},
		{`C:\maps\sub.dir\a.almx`, `C:\maps\sub.dir\a.variables`},
		{"maps/no-extension", "maps/no-extension.variables"},
	} {
		if got := MapLabelFile(tc.mapPath); got != tc.want {
			t.Errorf("MapLabelFile(%q) = %q, want %q", tc.mapPath, got, tc.want)
		}
	}
}

func TestMissionPathWarningsUseResolvedPaths(t *testing.T) {
	for _, tc := range []struct {
		name, mapPath, texts, want string
	}{
		{"dot pages", "maps/a.alm", "./maps/a.text", ""},
		{"dot map", "./maps/a.almx", "./maps/a.text", ""},
		{"language map", "mission/{lang}/a.alm", "", "map mission/{lang}/a.alm uses an old path; use maps/{lang}/<stem>.alm"},
		{"language extended map", "mission/{lang}/a.almx", "", "map mission/{lang}/a.almx uses an old path; use maps/{lang}/<stem>.almx"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := writeFiles(t, map[string]string{
				ManifestName: good("paths"),
				MissionsFile: fmt.Sprintf("[[mission]]\nmap = %q\n", tc.mapPath),
			})
			if tc.texts != "" {
				writeTreeFile(t, dir, MissionsFile, fmt.Sprintf("[[mission]]\nmap = %q\ntexts = %q\n", tc.mapPath, tc.texts))
			}
			tree, err := OpenTree(dir)
			if err != nil {
				t.Fatal(err)
			}
			d, err := tree.Describe()
			if err != nil {
				t.Fatal(err)
			}
			if tc.want == "" {
				if len(d.Warnings) != 0 {
					t.Fatalf("warnings on canonical paths: %v", d.Warnings)
				}
			} else if len(d.Warnings) != 1 || !strings.HasSuffix(d.Warnings[0], tc.want) {
				t.Fatalf("warnings %v, want %s", d.Warnings, tc.want)
			}
		})
	}
}

// The old texts warning names the pages folder the place rule gives the map:
// a numbered map's stem is m<N>.
func TestOldTextsWarningNamesThePlaceRulePages(t *testing.T) {
	for mapPath, want := range map[string]string{
		"mission/10.alm":        "texts mission/text/{lang} uses an old path; use maps/m10.text or its language folder",
		"mission/{lang}/10.alm": "texts mission/text/{lang} uses an old path; use maps/{lang}/m10.text or its language folder",
		"maps/a.alm":            "texts mission/text/{lang} uses an old path; use maps/a.text or its language folder",
	} {
		dir := writeFiles(t, map[string]string{ManifestName: good("paths"),
			MissionsFile: fmt.Sprintf("[[mission]]\nmap = %q\ntexts = \"mission/text/{lang}\"\n", mapPath)})
		tree, err := OpenTree(dir)
		if err != nil {
			t.Fatal(err)
		}
		d, err := tree.Describe()
		if err != nil {
			t.Fatal(err)
		}
		if !slices.ContainsFunc(d.Warnings, func(w string) bool { return strings.HasSuffix(w, want) }) {
			t.Errorf("%s: warnings %q, want %s", mapPath, d.Warnings, want)
		}
	}
}

func TestFixedTreeWarnings(t *testing.T) {
	for _, tc := range []struct {
		name  string
		files map[string]string
		want  []string
	}{
		{"outside", map[string]string{"tools/build.py": "ignored", "data/typo.toml": "ignored", ".git/config": "ignored", "stray.text/title.txt": "ignored"}, []string{"data/typo.toml: no fixed-tree kind", "stray.text/title.txt: no fixed-tree kind", "tools/build.py: no fixed-tree kind"}},
		{"second name", map[string]string{WeaponBodiesLongFile: "# empty"}, []string{WeaponBodiesLongFile + ": use " + WeaponBodiesFile}},
		{"old map", map[string]string{MissionsFile: "[[mission]]\nmap = \"mission/en/map.alm\"\n", "mission/en/map.alm": "map"}, []string{MissionsFile + ":2: map mission/en/map.alm uses an old path"}},
		{"old pages", map[string]string{MissionsFile: "[[mission]]\nmap = \"maps/a.alm\"\ntexts = \"mission/text/{lang}\"\n", "mission/text/en/title.txt": "title"}, []string{MissionsFile + ":3: texts mission/text/{lang} uses an old path"}},
		{"title page", map[string]string{MissionsFile: "[[mission]]\nmap = \"maps/a.alm\"\ntitle = \"key\"\n", "maps/a.text/en/title.txt": "Title"}, []string{MissionsFile + ":3: title key overrides title.txt for en", MissionsFile + ":3: title key overrides title.txt for ru"}},
		{"briefing page", map[string]string{MissionsFile: "[[mission]]\nmap = \"maps/a.alm\"\nbriefing = \"key\"\n", "maps/a.text/ru/briefing.txt": "Briefing"}, []string{MissionsFile + ":3: briefing key overrides briefing.txt for en", MissionsFile + ":3: briefing key overrides briefing.txt for ru"}},
		{"long names", map[string]string{"maps/long-map-name-over-fifteen.alm": "map"}, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := filepath.Join(t.TempDir(), "warnings")
			writeTreeFile(t, dir, ManifestName, good("warnings"))
			for p, b := range tc.files {
				writeTreeFile(t, dir, p, b)
			}
			tree, _ := OpenTree(dir)
			d, err := tree.Describe()
			if err != nil {
				t.Fatal(err)
			}
			if len(d.Warnings) != len(tc.want) {
				t.Fatalf("warnings %q, want %q", d.Warnings, tc.want)
			}
			for i, w := range tc.want {
				if !strings.HasPrefix(d.Warnings[i], w) {
					t.Errorf("%q, want prefix %q", d.Warnings[i], w)
				}
			}
		})
	}
}

func TestFixedTreeEveryKindPacksAndKeepsBytesAndDigest(t *testing.T) {
	files := map[string]string{
		ManifestName: good("whole"), EntryName: "def init(game, settings):\n    pass\n", "lib/a.star": "x = 1\n",
		SettingsName: "# no settings\n", "text/en/strings.toml": "\"a\" = \"A\"\n",
		"maps/a.alm": "map", "maps/a.text/en/title.txt": "Title\r\n", "maps/a.variables": "{}",
		"assets/items/a.png": "PNG", "assets/bodies/a.png": "PNG", "assets/speakers/20000.png": "PNG", "assets/towns/a.json": "{}",
		"files/common/graphics/a.bmp": "BM\r\n", "files/en/sounds/a.wav": "WAV", "files/ru/main/a.txt": "text\r\n",
		"10.alm": "map", ReadmeName: "people\r\n",
	}
	for _, k := range DataKinds() {
		files[k.Files[0]] = "# no records\n"
	}
	dir := filepath.Join(t.TempDir(), "whole")
	for p, b := range files {
		writeTreeFile(t, dir, p, b)
	}
	tree, _ := OpenTree(dir)
	d, err := tree.Describe()
	if err != nil || len(d.Warnings) != 0 || len(d.Files) != len(files) {
		t.Fatalf("description %+v: %v", d, err)
	}
	held := map[string]bool{}
	for _, f := range d.Files {
		held[f.Kind] = true
	}
	for _, k := range TreeKinds() {
		if !held[k.Name] {
			t.Errorf("no file of %s", k.Name)
		}
	}
	var entries []res.WriteEntry
	var paths []string
	err = tree.WalkDir(func(p string, e os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if p == "." {
			return nil
		}
		entries = append(entries, res.WriteEntry{Path: p, Dir: e.IsDir(), Data: []byte(files[p])})
		if !e.IsDir() && p != ReadmeName {
			paths = append(paths, p)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	wantPack, err := res.Write(entries)
	if err != nil {
		t.Fatal(err)
	}
	_, gotPack, err := Pack(dir)
	if err != nil || !bytes.Equal(gotPack, wantPack) {
		t.Fatalf("pack bytes moved: %v", err)
	}
	h := sha256.New()
	slices.Sort(paths)
	for _, p := range paths {
		body := []byte(files[p])
		switch filepath.Ext(p) {
		case ".toml", ".star", ".txt", ".md":
			body = bytes.ReplaceAll(body, []byte("\r\n"), []byte("\n"))
		}
		fmt.Fprintf(h, "%s\x00%d\x00", p, len(body))
		h.Write(body)
	}
	wantDigest := hex.EncodeToString(h.Sum(nil))
	packed, err := openPackedBytes(PackedName("whole"), gotPack)
	if err != nil {
		t.Fatal(err)
	}
	for _, ttree := range []Tree{tree, packed} {
		got, err := ttree.Digest()
		if err != nil || got != wantDigest {
			t.Errorf("digest %s: %v, want %s", got, err, wantDigest)
		}
		for p, b := range files {
			got, err := ttree.ReadFile(p)
			if err != nil || string(got) != b {
				t.Errorf("read %s: %q %v", p, got, err)
			}
		}
	}
}

func TestFixedTreeExcludesStraysAndKeepsReferencedOldPaths(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "old")
	for p, b := range map[string]string{
		ManifestName: good("old") + "entry = \"logic/entry.rule\"\n", "logic/entry.rule": "def init(game, settings):\n    pass\n",
		ItemsFile: "[[material]]\nsprite = \"old/picture.png\"\n", "old/picture.png": "PNG",
		TownsFile: "[[town]]\ndescription = \"old/town.json\"\n", "old/town.json": "{\"art\":[{\"name\":\"picture\",\"format\":\"picture\",\"key\":\"mod:old/art.bmp\"}]}", "old/art.bmp": "BM",
		MissionsFile:  "[[mission]]\nmap = \"old/map.alm\"\ntexts = \"pages/{lang}\"\n",
		"old/map.alm": "map", "old/map.text/en/title.txt": "Title", "old/map.variables": "{}", "pages/ru/event01.txt": "Event",
		"stray.txt": "stray", "old/unused.png": "stray", "data/typo.toml": "stray",
	} {
		writeTreeFile(t, dir, p, b)
	}
	tree, _ := OpenTree(dir)
	d, err := tree.Describe()
	if err != nil {
		t.Fatal(err)
	}
	before, err := tree.Digest()
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range []string{"stray.txt", "old/unused.png", "data/typo.toml"} {
		if d.Admits(p) {
			t.Errorf("admitted %s", p)
		}
		writeTreeFile(t, dir, p, "changed stray")
	}
	after, _ := tree.Digest()
	if before != after {
		t.Fatal("stray moved digest")
	}
	packedPath := packInto(t, dir, t.TempDir())
	packed, _ := OpenTree(packedPath)
	for _, f := range d.Files {
		want, _ := tree.ReadFile(f.Path)
		got, err := packed.ReadFile(f.Path)
		if err != nil || !bytes.Equal(got, want) {
			t.Errorf("lost %s: %v", f.Path, err)
		}
	}
	if _, err := packed.ReadFile("stray.txt"); err == nil {
		t.Fatal("packed stray")
	}
	writeTreeFile(t, dir, "old/picture.png", "changed picture")
	changed, _ := tree.Digest()
	if changed == after {
		t.Fatal("referenced asset did not move digest")
	}
}

func TestModsDocListsFixedTree(t *testing.T) {
	doc, err := os.ReadFile(filepath.Join("..", "..", "docs", "MODS.md"))
	if err != nil {
		t.Fatal(err)
	}
	text := strings.ReplaceAll(string(doc), "\r\n", "\n")
	_, part, ok := strings.Cut(text, "<!-- fixed-tree:start -->\n")
	if !ok {
		part = ""
	}
	part, _, _ = strings.Cut(part, "<!-- fixed-tree:end -->")
	rows := []string{"| kind | paths | formats | found by | language |", "|---|---|---|---|---|"}
	if len(TreeKinds()) != 24 {
		t.Fatalf("%d tree kinds, want 24", len(TreeKinds()))
	}
	for _, k := range TreeKinds() {
		paths := append(slices.Clone(k.Files), k.Paths...)
		if k.Format == "" || k.Found == "" || k.Language == "" {
			t.Fatalf("incomplete row %s", k.Name)
		}
		row := "| " + strings.Join([]string{k.Name, "`" + strings.Join(paths, "`, `") + "`", k.Format, k.Found, k.Language}, " | ") + " |"
		rows = append(rows, row)
		if os.Getenv("AGAINROM_TREE_DOC_ROWS") != "" {
			t.Log("TREE_ROW" + row)
		}
	}
	want := strings.Join(rows, "\n") + "\n"
	if part != want {
		t.Errorf("fixed-tree section differs from the table")
	}
}
