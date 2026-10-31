package mod

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"againrom/pkg/formats/res"
	"againrom/pkg/namefold"
)

// packFixture is a mod folder with every kind of file a mod reader takes:
// the manifest, a README, scripts, settings, a data file, texts in two
// languages, a mission map with its pages, and install files, plus a dotted
// folder no reader takes.
func packFixture(id string) map[string]string {
	return map[string]string{
		"mod.toml":                      "id = \"" + id + "\"\ntitle = \"Packed\"\nversion = \"1.2\"\n",
		"README.md":                     "for people\r\n",
		"main.star":                     "load(\"lib/helper.star\", \"cap\")\ndef init(game, settings):\n    game.rules.skill_cap = cap(settings.skill_cap)\n",
		"lib/helper.star":               "def cap(v):\n    return v\n",
		"settings.toml":                 sampleSettings,
		"data/items.toml":               "# no rows\r\n",
		"text/en/strings.toml":          "\"item.cloak\" = \"Cloak\"\n",
		"text/ru/strings.toml":          "\"item.cloak\" = \"Плащ\"\n",
		"maps/east.alm":                 "not a real map",
		"maps/east.text/en/brief_1.txt": "Go east.",
		"maps/east.text/ru/brief_1.txt": "На восток.",
		"files/common/graphics/interface/Town/townmain.bmp": "BM picture bytes",
		"files/en/main/text/sites.txt":                      "East\r\n",
		"files/ru/main/text/sites.txt":                      "Восток\r\n",
		".git/config":                                       "[core]\n",
	}
}

// packInto packs the mod folder dir into mods as mod-<id>.res and returns the
// RES file's path.
func packInto(t *testing.T, dir, mods string) string {
	t.Helper()
	m, b, err := Pack(dir)
	if err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(mods, PackedName(m.ID))
	if err := os.WriteFile(p, b, 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

// readerView is what every mod reader answers for one mod.
type readerView struct {
	Manifest  Manifest
	Settings  []Setting
	Digest    string
	Files     map[string][]InstallFile
	Strings   map[string]Strings
	Pages     map[string]map[string][]byte
	Reads     map[string]string
	EntryStat bool
}

func viewOf(t *testing.T, dir string) readerView {
	t.Helper()
	e := Load(dir)
	if e.Err != nil {
		t.Fatalf("%s: %v", dir, e.Err)
	}
	v := readerView{Manifest: e.Manifest, Files: map[string][]InstallFile{}, Strings: map[string]Strings{}, Pages: map[string]map[string][]byte{}, Reads: map[string]string{}}
	var err error
	if v.Settings, err = LoadSettings(dir); err != nil {
		t.Fatal(err)
	}
	if v.Digest, err = ContentDigest(dir); err != nil {
		t.Fatal(err)
	}
	for _, lang := range []string{"en", "ru"} {
		if v.Files[lang], err = ReadInstallFiles(e.Manifest.ID, dir, lang); err != nil {
			t.Fatal(err)
		}
		if v.Strings[lang], err = LoadStrings(dir, lang); err != nil {
			t.Fatal(err)
		}
		if v.Pages[lang], _, err = (Mission{Map: "maps/east.alm"}).ReadTexts(dir, lang); err != nil {
			t.Fatal(err)
		}
	}
	for _, name := range []string{"main.star", "lib/helper.star", "data/items.toml", "./data/items.toml", "../x", "nosuch.star", "data", `data\items.toml`} {
		b, rel, err := ReadInside(dir, name)
		v.Reads[name] = rel + "|" + string(b) + "|" + errText(err)
	}
	t.Log(dir, v.Digest)
	return v
}

func errText(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

// A mod and its packed RES read the same through every reader: manifest,
// settings, digest, install files on both languages, texts, mission pages,
// script reads and their refusals. Two folders that differ only in one
// install file give different views, so the comparison can fail; a packed
// reader that skipped files/ or data/ fails it.
func TestPackedModReadsAsItsFolder(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "packed-fix")
	files := packFixture("packed-fix")
	for rel, body := range files {
		writeTreeFile(t, dir, rel, body)
	}
	if err := os.MkdirAll(filepath.Join(dir, "files", "ru", "main", "empty"), 0o755); err != nil {
		t.Fatal(err)
	}
	mods := t.TempDir()
	packed := packInto(t, dir, mods)
	folder, res := viewOf(t, dir), viewOf(t, packed)
	if !reflect.DeepEqual(folder, res) {
		t.Fatalf("the packed mod reads otherwise than its folder:\nfolder %+v\npacked %+v", folder, res)
	}
	if len(folder.Files["ru"]) != 2 || len(folder.Strings["ru"]) != 1 || len(folder.Pages["ru"]) != 1 || len(folder.Settings) != 3 {
		t.Fatalf("the fixture reads too little: %+v", folder)
	}
	if b, rel, err := ReadInside(packed, "MAIN.STAR"); err != nil || rel != "MAIN.STAR" || !strings.Contains(string(b), "def init") {
		t.Errorf("a packed name in another case: %q %v", rel, err)
	}

	other := filepath.Join(t.TempDir(), "packed-fix")
	for rel, body := range files {
		writeTreeFile(t, other, rel, body)
	}
	writeTreeFile(t, other, "files/ru/main/text/sites.txt", "Запад\r\n")
	if reflect.DeepEqual(folder, viewOf(t, other)) {
		t.Fatal("two mods with different install files read the same; the comparison proves nothing")
	}

	mark := Load(packed)
	if mark.Folder != "packed-fix" || mark.Dir != packed {
		t.Errorf("entry %+v", mark)
	}
	if tree, err := OpenTree(packed); err != nil || !tree.Packed() {
		t.Errorf("%s opens as a folder: %v", packed, err)
	}
}

func writeTreeFile(t *testing.T, dir, rel, body string) {
	t.Helper()
	p := filepath.Join(dir, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

// A folder the install-file rules refuse is refused in the same words when
// packed: an empty folder under files/ that names no language is kept in
// the RES.
func TestPackedModIsHeldToTheFolderRules(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "rules")
	writeTreeFile(t, dir, "mod.toml", good("rules"))
	if err := os.MkdirAll(filepath.Join(dir, "files", "de"), 0o755); err != nil {
		t.Fatal(err)
	}
	packed := packInto(t, dir, t.TempDir())
	_, want := ReadInstallFiles("rules", dir, "en")
	_, got := ReadInstallFiles("rules", packed, "en")
	if want == nil || got == nil || want.Error() != got.Error() {
		t.Fatalf("folder %v, packed %v", want, got)
	}
}

// Pack refuses a name the RES cannot hold, naming the file: a name over 15
// bytes and a character CP866 has no byte for. A Cyrillic name is stored in
// CP866 and reads back by its own name.
func TestPackRefusesANameTheResCannotHoldNamingTheFile(t *testing.T) {
	for _, c := range []struct{ file, want string }{
		{"data/weapon-bodies.toml", `mod "names": data/weapon-bodies.toml: cannot be packed: the name "weapon-bodies.toml" is 18 bytes; a node name is at most 15`},
		{"pictures-of-the-east/a.png", `mod "names": pictures-of-the-east: cannot be packed: the name "pictures-of-the-east" is 20 bytes`},
		{"text/日本.txt", `mod "names": text/日本.txt: cannot be packed: the name holds '日', which the archive's code page (CP866) has no byte for`},
	} {
		dir := filepath.Join(t.TempDir(), "names")
		writeTreeFile(t, dir, "mod.toml", good("names"))
		writeTreeFile(t, dir, c.file, "x")
		if _, _, err := Pack(dir); err == nil || !strings.HasPrefix(err.Error(), c.want) {
			t.Errorf("%s: %v\nwant %s", c.file, err, c.want)
		}
	}
	dir := filepath.Join(t.TempDir(), "names")
	writeTreeFile(t, dir, "mod.toml", good("names"))
	writeTreeFile(t, dir, "text/карта.txt", "map")
	packed := packInto(t, dir, t.TempDir())
	if b, rel, err := ReadInside(packed, "text/карта.txt"); err != nil || string(b) != "map" || rel != "text/карта.txt" {
		t.Errorf("Cyrillic name: %q %q %v", b, rel, err)
	}
	if _, _, err := Pack(packed); err == nil || !strings.Contains(err.Error(), "is a packed mod already") {
		t.Errorf("packing a RES: %v", err)
	}
}

// A folder and a packed file of one mod id are refused naming both, by Scan
// and by Resolve. A file named mod*.res that is not mod-<id>.res, one whose
// manifest names another id, and one that is not a RES are listed refused;
// another .res file is not listed.
func TestFolderAndPackedFileOfOneIdAreRefusedNamingBoth(t *testing.T) {
	mods := t.TempDir()
	src := filepath.Join(t.TempDir(), "twice")
	writeTreeFile(t, src, "mod.toml", good("twice"))
	packed := packInto(t, src, mods)
	writeMod(t, mods, "twice", good("twice"))
	writeMod(t, mods, "plain", good("plain"))
	for name, b := range map[string]string{"modwrong.res": "", "other.res": "", "mod-broken.res": "not a RES"} {
		if err := os.WriteFile(filepath.Join(mods, name), []byte(b), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if b, err := os.ReadFile(packed); err != nil {
		t.Fatal(err)
	} else if err := os.WriteFile(filepath.Join(mods, "mod-renamed.res"), b, 0o644); err != nil {
		t.Fatal(err)
	}

	folder := filepath.Join(mods, "twice")
	both := `mod "twice" is both ` + folder + ` and ` + packed + `; keep one of them`
	if _, err := Resolve(mods, []string{"twice"}); err == nil || err.Error() != both {
		t.Fatalf("Resolve: %v\nwant %s", err, both)
	}
	got, err := Scan(mods)
	if err != nil {
		t.Fatal(err)
	}
	refusal := map[string]string{}
	var order []string
	for _, e := range got {
		order = append(order, e.Folder+"="+filepath.Base(e.Dir))
		refusal[filepath.Base(e.Dir)] = errText(e.Err)
	}
	if want := []string{"broken=mod-broken.res", "modwrong.res=modwrong.res", "plain=plain", "renamed=mod-renamed.res", "twice=twice", "twice=mod-twice.res"}; !reflect.DeepEqual(order, want) {
		t.Fatalf("Scan lists %v, want %v", order, want)
	}
	for name, want := range map[string]string{
		"twice":           both,
		"mod-twice.res":   both,
		"plain":           "",
		"modwrong.res":    `a packed mod's file is named mod-<id>.res with a valid id, not "modwrong.res"`,
		"mod-renamed.res": `manifest id "twice" does not match the file name "mod-renamed.res"`,
		"mod-broken.res":  "res: ",
	} {
		if got := refusal[name]; want == "" && got != "" || !strings.Contains(got, want) {
			t.Errorf("%s: %q, want %q", name, got, want)
		}
	}
	if err := os.RemoveAll(folder); err != nil {
		t.Fatal(err)
	}
	if e, err := Resolve(mods, []string{"twice"}); err != nil || e[0].Dir != packed {
		t.Fatalf("the packed mod alone: %+v %v", e, err)
	}
	if _, err := Resolve(mods, []string{"absent"}); err == nil || !strings.Contains(err.Error(), "no folder "+filepath.Join(mods, "absent")+" and no packed file mod-absent.res") {
		t.Fatalf("absent: %v", err)
	}
}

// The digest is the documented form, built here by hand: each regular file
// but README.md and dotted names, in path order, as "path NUL length NUL
// bytes", with CRLF read as LF in text files only. The folder and its packed
// RES both give it, so a save's mod mark does not depend on the form.
func TestDigestIsTheDocumentedFormForFolderAndPackedFile(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "digest")
	for rel, body := range map[string]string{
		"mod.toml":       good("digest") + "\r\n",
		"README.md":      "people\r\n",
		"a/B.txt":        "l1\r\nl2",
		"pic.bmp":        "\r\n",
		".hidden/x.star": "y",
	} {
		writeTreeFile(t, dir, rel, body)
	}
	h := sha256.New()
	for _, f := range []struct{ path, data string }{
		{"a/B.txt", "l1\nl2"},
		{"mod.toml", good("digest") + "\n"},
		{"pic.bmp", "\r\n"},
	} {
		fmt.Fprintf(h, "%s\x00%d\x00%s", f.path, len(f.data), f.data)
	}
	want := hex.EncodeToString(h.Sum(nil))
	packed := packInto(t, dir, t.TempDir())
	for _, p := range []string{dir, packed} {
		if got, err := ContentDigest(p); err != nil || got != want {
			t.Errorf("%s: %s %v, want %s", p, got, err, want)
		}
	}
}

// Two names equal without case are one name to a packed mod: Pack refuses
// them naming both, and a RES that holds them is not opened. The fold covers
// every letter a RES name can hold, Cyrillic and Ё included.
func TestPackedNamesFoldEveryLetterARESCanHold(t *testing.T) {
	for _, c := range [][2]string{{"карта.star", "КАРТА.star"}, {"ёлка.star", "Ёлка.star"}, {"Map.star", "map.star"}} {
		err := refuseFoldTwins("twins", []res.WriteEntry{{Path: "a/" + c[0]}, {Path: "a/" + c[1]}})
		if err == nil || !strings.Contains(err.Error(), "a/"+c[0]) || !strings.Contains(err.Error(), "a/"+c[1]) {
			t.Errorf("%v: %v", c, err)
		}
		b, werr := res.Write([]res.WriteEntry{{Path: "a", Dir: true}, {Path: "a/" + c[0], Data: []byte("x")}, {Path: "a/" + c[1], Data: []byte("y")}})
		if werr != nil {
			continue // the writer already refuses it
		}
		if _, err := openPackedBytes("twins.res", b); err == nil || !strings.Contains(err.Error(), c[0]) || !strings.Contains(err.Error(), c[1]) {
			t.Errorf("opened a RES with %v: %v", c, err)
		}
	}
	for _, p := range []string{"карта.star", "КАРТА.star", "ЁЛКА.star", "ёлка.star", "ЄЇЎ", "єїў"} {
		if namefold.ModTree(p) != namefold.ModTree(strings.ToLower(p)) {
			t.Errorf("%q not folded as its lower case", p)
		}
	}
}

// Every reader refuses a path with a dotted component, in a folder and in a
// RES, naming the path.
func TestDottedPathsAreRefusedByEveryReader(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "dots")
	writeTreeFile(t, dir, "mod.toml", good("dots"))
	writeTreeFile(t, dir, ".cap.star", "x")
	writeTreeFile(t, dir, "d/.e/f.star", "x")
	packed := packInto(t, dir, t.TempDir())
	for _, root := range []string{dir, packed} {
		tr, err := OpenTree(root)
		if err != nil {
			t.Fatal(err)
		}
		for _, p := range []string{".cap.star", "d/.e/f.star", "d/.e"} {
			if _, _, err := tr.ReadInside(p); err == nil || !strings.Contains(err.Error(), p) {
				t.Errorf("%s ReadInside(%s): %v", root, p, err)
			}
			if _, err := tr.ReadFile(p); err == nil || !strings.Contains(err.Error(), p) {
				t.Errorf("%s ReadFile(%s): %v", root, p, err)
			}
			if _, err := tr.Stat(p); err == nil || !strings.Contains(err.Error(), p) {
				t.Errorf("%s Stat(%s): %v", root, p, err)
			}
		}
		if _, err := tr.ReadDir("d/.e"); err == nil {
			t.Errorf("%s ReadDir of a dotted folder", root)
		}
	}
}

// The weapon bodies file is read under its short and its first name; a mod
// holding both is refused by Load, naming both files, in a folder and in a
// packed file's reading alike. The short name packs.
func TestWeaponBodiesNames(t *testing.T) {
	both := filepath.Join(t.TempDir(), "wb")
	writeTreeFile(t, both, "mod.toml", good("wb"))
	writeTreeFile(t, both, WeaponBodiesFile, "x")
	if e := Load(both); e.Err != nil {
		t.Fatalf("the short name alone: %v", e.Err)
	}
	if _, _, err := Pack(both); err != nil {
		t.Fatalf("the short name did not pack: %v", err)
	}
	writeTreeFile(t, both, WeaponBodiesLongFile, "x")
	want := "the mod holds both " + WeaponBodiesFile + " and " + WeaponBodiesLongFile
	if e := Load(both); e.Err == nil || !strings.Contains(e.Err.Error(), want) {
		t.Fatalf("both names: %v", e.Err)
	}
	long := filepath.Join(t.TempDir(), "wl")
	writeTreeFile(t, long, "mod.toml", good("wl"))
	writeTreeFile(t, long, WeaponBodiesLongFile, "x")
	if e := Load(long); e.Err != nil {
		t.Fatalf("the long name alone: %v", e.Err)
	}
}
