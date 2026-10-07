package mod

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestParseManifestReadsTheFourFields(t *testing.T) {
	src := "# a mod\nid = \"dark-rules\"   # trailing\ntitle = 'Dark \"rules\"'\nversion = \"1.2.0\"\napplies-to = [\"en\", \"ru\"]\nrequires = [\n  \"base\", # multi\n  \"other\",\n]\nflag = true\n\n[extra]\nid = \"ignored\"\nnot a pair at all\n"
	m, err := ParseManifest([]byte(src))
	if err != nil {
		t.Fatal(err)
	}
	want := Manifest{ID: "dark-rules", Title: `Dark "rules"`, Version: "1.2.0", AppliesTo: []string{"en", "ru"}, API: 1, Requires: []string{"base", "other"}}
	if !reflect.DeepEqual(m, want) {
		t.Fatalf("got %+v", m)
	}
}

func TestParseManifestDefaultsAndScalarAppliesTo(t *testing.T) {
	m, err := ParseManifest([]byte("id = \"a\"\r\ntitle = \"A\\tB\"\r\nversion = \"1\"\r\n"))
	if err != nil || m.Title != "A\tB" || !reflect.DeepEqual(m.AppliesTo, []string{"common"}) {
		t.Fatalf("%+v %v", m, err)
	}
	m, err = ParseManifest([]byte("id = \"a\"\ntitle = \"A\"\nversion = \"1\"\napplies-to = \"ru\"\n"))
	if err != nil || !reflect.DeepEqual(m.AppliesTo, []string{"ru"}) {
		t.Fatalf("%+v %v", m, err)
	}
}

func TestParseManifestRefusals(t *testing.T) {
	cases := []struct{ name, src, want string }{
		{"no id", "title = \"A\"\nversion = \"1\"\n", "id is missing"},
		{"no version", "id = \"a\"\ntitle = \"A\"\n", "version is missing"},
		{"bad id", "id = \"../x\"\ntitle = \"A\"\nversion = \"1\"\n", "not valid"},
		{"upper id", "id = \"Abc\"\ntitle = \"A\"\nversion = \"1\"\n", "not valid"},
		{"not a string", "id = 5\ntitle = \"A\"\nversion = \"1\"\n", "line 1"},
		{"unterminated", "id = \"a\ntitle = \"A\"\nversion = \"1\"\n", "line 1"},
		{"junk after value", "id = \"a\" x\ntitle = \"A\"\nversion = \"1\"\n", "unexpected text"},
		{"no equals", "id = \"a\"\nwhat\n", "line 2"},
		{"duplicate", "id = \"a\"\nid = \"b\"\n", "given twice"},
		{"bad escape", "id = \"a\\q\"\n", "unsupported escape"},
		{"empty applies", "id = \"a\"\ntitle = \"A\"\nversion = \"1\"\napplies-to = []\n", "applies-to is empty"},
		{"bad array", "id = \"a\"\ntitle = \"A\"\nversion = \"1\"\napplies-to = [\"en\" \"ru\"]\n", "expected ','"},
		{"too big", strings.Repeat("#", maxManifest+1), "larger than"},
	}
	for _, c := range cases {
		_, err := ParseManifest([]byte(c.src))
		if err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%s: err %v, want it to contain %q", c.name, err, c.want)
		}
	}
}

func writeMod(t *testing.T, root, folder, manifest string) {
	t.Helper()
	dir := filepath.Join(root, folder)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if manifest != "" {
		if err := os.WriteFile(filepath.Join(dir, ManifestName), []byte(manifest), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func good(id string) string {
	return "id = \"" + id + "\"\ntitle = \"T " + id + "\"\nversion = \"0.1\"\n"
}

func TestScanListsOnlyModFoldersSorted(t *testing.T) {
	root := t.TempDir()
	writeMod(t, root, "zeta", good("zeta"))
	writeMod(t, root, "alpha", good("alpha"))
	writeMod(t, root, "broken", "id = \n")
	writeMod(t, root, "renamed", good("other"))
	writeMod(t, root, "empty", "")
	if err := os.WriteFile(filepath.Join(root, "file.txt"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := Scan(root)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	bad := map[string]bool{}
	for _, e := range got {
		names = append(names, e.Folder)
		bad[e.Folder] = e.Err != nil
	}
	if !reflect.DeepEqual(names, []string{"alpha", "broken", "renamed", "zeta"}) {
		t.Fatalf("names %v", names)
	}
	if bad["alpha"] || bad["zeta"] || !bad["broken"] || !bad["renamed"] {
		t.Fatalf("errors %v", bad)
	}
	if none, err := Scan(filepath.Join(root, "nosuch")); err != nil || none != nil {
		t.Fatalf("missing dir: %v %v", none, err)
	}
}

func TestResolveOrderAndRefusals(t *testing.T) {
	root := t.TempDir()
	writeMod(t, root, "a", good("a"))
	writeMod(t, root, "b", good("b"))
	writeMod(t, root, "bad", "title = \"x\"\n")
	got, err := Resolve(root, []string{"b", "a"})
	if err != nil || len(got) != 2 || got[0].Manifest.ID != "b" || got[1].Manifest.ID != "a" {
		t.Fatalf("%+v %v", got, err)
	}
	for ids, want := range map[string]string{
		"a,missing": `mod "missing"`,
		"bad":       `mod "bad"`,
		"a,a":       "named twice",
		"../a":      "not a valid mod id",
		"A":         "not a valid mod id",
	} {
		_, err := Resolve(root, strings.Split(ids, ","))
		if err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("%s: %v, want %q", ids, err, want)
		}
	}
	if none, err := Resolve(root, nil); err != nil || len(none) != 0 {
		t.Fatalf("no ids: %v %v", none, err)
	}
}

func TestDefaultDir(t *testing.T) {
	if got := DefaultDir(filepath.Join("c", "game")); got != filepath.Join("c", "game", "mods") {
		t.Fatal(got)
	}
}
