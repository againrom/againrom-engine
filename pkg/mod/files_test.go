package mod

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
)

// writeFiles lays files, by slash path under the mod folder, into a new mod
// folder and returns it.
func writeFiles(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	for rel, body := range files {
		p := filepath.Join(dir, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func addresses(files []InstallFile) string {
	var out []string
	for _, f := range files {
		out = append(out, f.Lang+":"+f.Address)
	}
	return strings.Join(out, " ")
}

// On each install the language folder's file wins over the common folder's
// for one install path; the common folder's other files apply on every
// install, and another language's files on none.
func TestInstallFilesTakeTheInstallLanguageOverCommon(t *testing.T) {
	dir := writeFiles(t, map[string]string{
		"files/common/main/text/sites.txt":           "Home\r\n",
		"files/common/graphics/interface/a.bmp":      "BM",
		"files/en/main/text/sites.txt":               "East\r\n",
		"files/ru/main/text/sites.txt":               "Восток\r\n",
		"files/ru/sfx/ui/click.wav":                  "RIFF",
		"files/.hidden/x.256":                        "skipped",
		"files/common/graphics/interface/.notes.txt": "skipped",
	})
	for lang, want := range map[string]string{
		"en":        "common:graphics/interface/a.bmp en:main/text/sites.txt",
		"ru":        "common:graphics/interface/a.bmp ru:main/text/sites.txt ru:sfx/ui/click.wav",
		CommonFiles: "common:graphics/interface/a.bmp common:main/text/sites.txt",
	} {
		files, err := ReadInstallFiles("m", dir, lang)
		if err != nil {
			t.Fatalf("%s: %v", lang, err)
		}
		if got := addresses(files); got != want {
			t.Errorf("%s: %s, want %s", lang, got, want)
		}
	}
	files, _ := ReadInstallFiles("m", dir, "ru")
	for _, f := range files {
		if f.Address == "main/text/sites.txt" && (string(f.Bytes) != "Восток\r\n" || f.Kind != FileText || f.File != "files/ru/main/text/sites.txt" || f.Mod != "m") {
			t.Errorf("the Russian file reads %+v", f)
		}
	}
	none, err := ReadInstallFiles("m", t.TempDir(), "en")
	if err != nil || len(none) != 0 {
		t.Fatalf("a mod with no files folder: %v %v", none, err)
	}
}

// Every file of every language folder is held to the folder's rules, and
// a refusal names the mod and the file.
func TestInstallFilesRefuseWhatTheGameCannotUse(t *testing.T) {
	for _, c := range []struct {
		files map[string]string
		want  string
	}{
		{map[string]string{"files/readme.txt": "x"}, `mod "m": files/readme.txt: stands outside a language folder`},
		{map[string]string{"files/de/main/text/a.txt": "x"}, `mod "m": files/de: "de" is not a language folder: the folders are common, en, ru`},
		{map[string]string{"files/en/a.txt": "x"}, `mod "m": files/en/a.txt: names no archive`},
		{map[string]string{"files/ru/graphics/units/a.256": "x"}, `mod "m": files/ru/graphics/units/a.256: the game takes no install file of this kind`},
		{map[string]string{"files/ru/main/text/a.txt": "\xff\xfe"}, `mod "m": files/ru/main/text/a.txt: is not UTF-8 text`},
	} {
		_, err := ReadInstallFiles("m", writeFiles(t, c.files), "en")
		if err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%v: %v, want %q", c.files, err, c.want)
		}
	}
	dir := writeFiles(t, map[string]string{"files": "not a folder"})
	if _, err := ReadInstallFiles("m", dir, "en"); err == nil || !strings.Contains(err.Error(), `mod "m": files: is not a folder`) {
		t.Errorf("a files file: %v", err)
	}
	big := writeFiles(t, map[string]string{"files/en/main/text/a.txt": ""})
	if err := os.Truncate(filepath.Join(big, "files", "en", "main", "text", "a.txt"), MaxInstallFileBytes+1); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadInstallFiles("m", big, "en"); err == nil || !strings.Contains(err.Error(), "is larger than") {
		t.Errorf("a large file: %v", err)
	}
}

// The mark set is the set without its state-free mods, in load order; the
// flag is not part of the canonical form, so a set's digest does not depend
// on it.
func TestMarkedSetLeavesOutStateFreeMods(t *testing.T) {
	s := Set{Base: "rom1-en", Mods: []SetEntry{{ID: "a", Version: "1"}, {ID: "pic", Version: "1", StateFree: true}, {ID: "b", Version: "2"}}}
	m := s.Marked()
	if m.Base != "rom1-en" || len(m.Mods) != 2 || m.Mods[0].ID != "a" || m.Mods[1].ID != "b" {
		t.Fatalf("marked %+v", m)
	}
	plain := Set{Base: s.Base, Mods: []SetEntry{{ID: "a", Version: "1"}, {ID: "pic", Version: "1"}, {ID: "b", Version: "2"}}}
	if s.Digest() != plain.Digest() {
		t.Fatal("the state-free flag changes the digest")
	}
	if !(Set{Base: "rom1-en", Mods: []SetEntry{{ID: "pic", StateFree: true}}}).Marked().Empty() {
		t.Fatal("a set of state-free mods marks a mod")
	}
}

// A language folder is named in any case, as the install path is; a file in
// files/EN applies on an English install.
func TestInstallFilesTakeALanguageFolderInAnyCase(t *testing.T) {
	dir := writeFiles(t, map[string]string{
		"files/Common/graphics/interface/a.bmp": "BM",
		"files/EN/main/text/sites.txt":          "East\r\n",
		"files/Ru/main/text/sites.txt":          "x\r\n",
	})
	files, err := ReadInstallFiles("m", dir, "en")
	if err != nil {
		t.Fatal(err)
	}
	if got, want := addresses(files), "common:graphics/interface/a.bmp en:main/text/sites.txt"; got != want {
		t.Errorf("%s, want %s", got, want)
	}
}

// Only a text can change game state: a picture and a sound do not.
func TestOnlyATextChangesState(t *testing.T) {
	art := []InstallFile{{Kind: FilePicture}, {Kind: FileSound}}
	if FilesChangeState(art) || FilesChangeState(nil) {
		t.Fatal("a picture or a sound changes state")
	}
	if !FilesChangeState(append(art, InstallFile{Kind: FileText})) {
		t.Fatal("a text does not change state")
	}
}

// Two spellings of one language folder in one mod are refused, naming both.
// The mod folder is a map file system, since a case-insensitive disk cannot
// hold both.
func TestInstallFilesRefuseTwoSpellingsOfALanguageFolder(t *testing.T) {
	fsys := fstest.MapFS{
		"files/EN/main/text/a.txt": {Data: []byte("x")},
		"files/en/main/text/b.txt": {Data: []byte("y")},
	}
	_, err := readInstallFiles("m", fsys, []string{CommonFiles, "en", "ru"}, "en")
	if err == nil || !strings.Contains(err.Error(), `is the language folder "en"`) || !strings.Contains(err.Error(), "files/EN") || !strings.Contains(err.Error(), "files/en") {
		t.Fatalf("two spellings: %v", err)
	}
}
