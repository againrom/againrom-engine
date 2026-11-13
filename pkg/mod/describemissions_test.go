package mod

import (
	"os"
	"path/filepath"
	"testing"
)

// DescribeMissions reads a mod's missions from its folder and from its packed
// file alike, worded in the language asked, with no map file there to open;
// a mod with no missions file declares none.
func TestDescribeMissionsReadsAFolderAndAPackedMod(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "ambush")
	files := map[string]string{
		ManifestName:           good("ambush"),
		MissionsFile:           "[[mission]]\nkey = \"first\"\nnumber = 25\nmap = \"maps/first.alm\"\ntitle = \"first.title\"\nafter = 20\nguest = 22\nchapter = 30\n[[mission]]\nnumber = 26\noff = true\n",
		"text/en/strings.toml": "\"first.title\" = \"Ambush\"\n",
		"text/ru/strings.toml": "\"first.title\" = \"Засада\"\n",
	}
	for name, body := range files {
		path := filepath.Join(dir, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	_, packed, err := Pack(dir)
	if err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(t.TempDir(), PackedName("ambush"))
	if err := os.WriteFile(file, packed, 0o644); err != nil {
		t.Fatal(err)
	}
	for _, at := range []string{dir, file} {
		for lang, title := range map[string]string{"en": "Ambush", "ru": "Засада"} {
			d, source, err := DescribeMissions(at, "ambush", lang)
			if err != nil {
				t.Fatalf("%s %s: %v", at, lang, err)
			}
			if string(source) != files[MissionsFile] {
				t.Errorf("%s %s: the source is not the missions file", at, lang)
			}
			listed := d.Listed()
			if len(d.Missions) != 2 || len(listed) != 1 || listed[0].Number != 25 || listed[0].Title != title {
				t.Errorf("%s %s: listed %+v of %d rows, want mission 25 %q", at, lang, listed, len(d.Missions), title)
			}
		}
	}

	plain := filepath.Join(root, "plain")
	writeMod(t, root, "plain", good("plain"))
	d, source, err := DescribeMissions(plain, "plain", "en")
	if err != nil || source != nil || !d.Empty() || len(d.Listed()) != 0 {
		t.Fatalf("a mod with no missions file: %+v %q %v", d, source, err)
	}
	if err := os.MkdirAll(filepath.Join(plain, "data"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(plain, filepath.FromSlash(MissionsFile)), []byte("[[mission]]\nbogus = 1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, _, err := DescribeMissions(plain, "plain", "en"); err == nil {
		t.Fatal("a broken missions file was described")
	}
}
