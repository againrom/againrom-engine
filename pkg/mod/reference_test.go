package mod

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"againrom/pkg/town"
)

// rowField is one row field that names a file of the mod: the text of a file
// holding it with a value, the line the value stands on, and the parse the
// kind runs. dir holds every file an accepted value names.
type rowField struct {
	name   string
	ref    Reference
	strict bool
	ext    string
	good   string
	line   int
	parse  func(dir, value string) error
}

func rowFields() []rowField {
	text := textOf(map[string]string{"n": "Name", "t": "T", "b": "B"})
	items := func(format string) func(string, string) error {
		return func(_, value string) error {
			_, err := ParseItems("m", ItemsFile, []byte(fmt.Sprintf(format, value)), text, "en")
			return err
		}
	}
	return []rowField{
		{name: "item sprite", ref: ItemSprite, strict: true, ext: ".png", good: "assets/items/a.png", line: 9,
			parse: items("[[item]]\nkey = \"gambeson\"\nname = \"n\"\nslot = \"body\"\nlayer = \"under\"\ndefence = 6\nweight = 6\nprice = 120\nsprite = %q\nstand-in = \"Soft Mail\"\nfigure = \"Soft Mail\"\n")},
		{name: "material sprite", ref: ItemSprite, strict: true, ext: ".png", good: "assets/items/a.png", line: 6,
			parse: items("[[material]]\nitem = \"Spear\"\ntier = \"Common\"\nmaterial = \"Meteoric\"\nallow = true\nsprite = %q\nname = \"n\"\n")},
		{name: "weapon sprite", ref: ItemSprite, strict: true, ext: ".png", good: "assets/items/a.png", line: 5,
			parse: items("[[weapon]]\nkey = \"star_spear\"\nname = \"n\"\nbase = \"Spear\"\nsprite = %q\ndamage-min = 9\ndamage-max = 14\n")},
		{name: "body heroes", ref: BodySheetPicture, strict: true, ext: ".png", good: "assets/bodies/a.png", line: 3,
			parse: func(_, value string) error {
				_, err := ParseBodies("m", BodiesFile, []byte(strings.Replace(goodBody, "bodies/clubman2h-heroes.png", value, 1)))
				return err
			}},
		{name: "body heroes_l", ref: BodySheetPicture, strict: true, ext: ".png", good: "assets/bodies/a.png", line: 4,
			parse: func(_, value string) error {
				_, err := ParseBodies("m", BodiesFile, []byte(strings.Replace(goodBody, "bodies/clubman2h-heroes_l.png", value, 1)))
				return err
			}},
		{name: "mission map", ref: MissionMap, good: "maps/a.alm", line: 5,
			parse: func(_, value string) error {
				_, err := ParseMissions("m", MissionsFile, []byte(strings.Replace(goodMission, "mission/{lang}/map.alm", value, 1)), text, "en")
				return err
			}},
		{name: "mission texts", ref: MissionTexts, good: "maps/a.text", line: 6,
			parse: func(_, value string) error {
				_, err := ParseMissions("m", MissionsFile, []byte(strings.Replace(goodMission, "mission/{lang}/text", value, 1)), text, "en")
				return err
			}},
		{name: "speaker picture", ref: SpeakerPicture, good: "assets/speakers/20000.png", line: 4,
			parse: func(dir, value string) error {
				_, err := ReadSpeakers("m", dir, SpeakersFile, []byte(fmt.Sprintf("[[speaker]]\nid = 20000\nname = \"S\"\npicture = %q\n", value)), nil)
				return err
			}},
		{name: "town description", ref: TownDescription, ext: ".json", good: "assets/towns/a.json", line: 3,
			parse: func(dir, value string) error {
				var d Data
				return readTowns(DataReader{Mod: "m", Dir: dir, Lang: "en"}, &d, TownsFile, []byte(fmt.Sprintf("[[town]]\nid = 9\ndescription = %q\n", value)))
			}},
	}
}

func referenceTree(t *testing.T) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "m")
	for _, p := range []string{"assets/items/a.png", "assets/bodies/a.png", "assets/speakers/20000.png", "assets/towns/a.json", "maps/a.alm", "old/a.png", "old/a.json"} {
		writeTreeFile(t, dir, p, "{}")
	}
	writeTreeFile(t, dir, ManifestName, good("m"))
	return dir
}

// Every row field that names a file takes a path inside the mod and refuses
// each other spelling at the row's own line.
func TestEveryRowFileFieldPassesTheReferenceRule(t *testing.T) {
	dir := referenceTree(t)
	long := strings.Repeat("n", maxPackedNameBytes+1)
	for _, f := range rowFields() {
		t.Run(f.name, func(t *testing.T) {
			if err := f.parse(dir, f.good); err != nil {
				t.Fatalf("%s refused: %v", f.good, err)
			}
			old := "old/a" + filepath.Ext(f.good)
			if f.ref.role == referencePages {
				old = "old"
			}
			if err := f.parse(dir, old); err != nil {
				t.Fatalf("a path outside assets is refused: %v", err)
			}
			ext := f.ext
			if ext == "" {
				ext = ".png"
			}
			refusals := map[string]string{
				"/a" + ext:          "is not a path inside the mod folder",
				"c:a" + ext:         "is not a path inside the mod folder",
				"../a" + ext:        "path inside the mod folder",
				".hidden/a" + ext:   "a name starting with a dot is not read from a mod",
				"art/.a" + ext:      "a name starting with a dot is not read from a mod",
				"art/" + long + ext: "a packed mod component is at most 255",
			}
			if f.strict {
				refusals[""] = "" // the row's own word for an empty value, at its line
				refusals["art//a"+ext] = "is not a clean path inside the mod folder"
				refusals["art/a.bmp"] = "is not a " + f.ext + " file"
			}
			if f.ref.role == referenceMap || f.ref.role == referencePages {
				refusals[""] = "is not a path inside the mod folder"
			}
			if f.name == "speaker picture" || f.name == "town description" {
				refusals["missing/none"+ext] = "no such file in the mod folder"
			}
			for value, want := range refusals {
				err := f.parse(dir, value)
				at := fmt.Sprintf(":%d: ", f.line)
				if err == nil || !strings.Contains(err.Error(), at) || !strings.Contains(err.Error(), want) {
					t.Errorf("%q: got %v, want line %d and %q", value, err, f.line, want)
				}
			}
			if f.ext != "" {
				upper := strings.TrimSuffix(f.good, f.ext) + strings.ToUpper(f.ext)
				if err := f.parse(dir, upper); err != nil {
					t.Errorf("%q: the extension matches without case, got %v", upper, err)
				}
			}
		})
	}
}

// The rule itself: what each reference takes, cleans and refuses.
func TestReferencePath(t *testing.T) {
	for _, c := range []struct {
		ref         Reference
		value, want string
		refused     string
	}{
		{ItemSprite, "assets/items/a.png", "assets/items/a.png", ""},
		{ItemSprite, "a.PNG", "a.PNG", ""},
		{ItemSprite, "./a.png", "", "not a clean path"},
		{ItemSprite, "a/../../b.png", "", "not a clean path"},
		{ItemSprite, "a.bmp", "", "is not a .png file"},
		{BodySheetPicture, "a/b.png", "a/b.png", ""},
		{BodySheetPicture, ".a/b.png", "", "starting with a dot"},
		{MissionMap, "maps//a.alm", "maps/a.alm", ""},
		{MissionMap, ".", "", "is not a path inside"},
		{MissionMap, "a/../..", "", "is not a path inside"},
		{MissionTexts, "maps/a.text/", "maps/a.text", ""},
		{TownDescription, "./towns/t.json", "towns/t.json", ""},
		{TownDescription, "towns/t.txt", "", "is not a .json file"},
		{TownArt, "./art/x.bmp", "art/x.bmp", ""},
		{TownArt, "art/y[1].bmp", "art/y[1].bmp", ""},
		{TownArt, "../x.bmp", "", "is not a path inside"},
		{SpeakerPicture, "art/{lang}.png", "art/{lang}.png", ""},
		{SpeakerPicture, "art/\xff.png", "", "not UTF-8"},
		{SpeakerPicture, "art/a\x01.png", "", "control character"},
		{SpeakerPicture, "art/" + strings.Repeat("n", maxPackedNameBytes+1), "", "at most 255"},
	} {
		got, err := c.ref.Path(c.value)
		if c.refused == "" && (err != nil || got != c.want) {
			t.Errorf("%s %q = %q, %v; want %q", c.ref.Kind, c.value, got, err, c.want)
		}
		if c.refused != "" && (err == nil || !strings.Contains(err.Error(), c.refused)) {
			t.Errorf("%s %q: %v, want %q", c.ref.Kind, c.value, err, c.refused)
		}
	}
}

// A "mod:" art key passes the rule, and a file it names that the mod lacks is
// refused by the reader of that key.
func TestTownArtKeyPassesTheReferenceRule(t *testing.T) {
	dir := referenceTree(t)
	for key, want := range map[string]string{
		"mod:.git/x.bmp":    "starting with a dot",
		"mod:/x.bmp":        "is not a path inside the mod folder",
		"mod:art/a\x02.bmp": "control character",
	} {
		_, err := TownArtFiles(artDescription(key))
		if err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("%q: %v, want %q", key, err, want)
		}
	}
	if _, _, err := TownArt.ReadIn(dir, "art/none.bmp"); err == nil || !strings.Contains(err.Error(), "no such file") {
		t.Errorf("a missing art file: %v", err)
	}
	if b, rel, err := TownArt.ReadIn(dir, "./assets/towns/a.json"); err != nil || rel != "assets/towns/a.json" || string(b) != "{}" {
		t.Errorf("read %q at %q: %v", b, rel, err)
	}
}

// The check warns about a named file outside its assets folder and about a
// file under assets no row names; a mod that keeps its files in place and
// names each of them shows neither.
func TestAssetWarnings(t *testing.T) {
	speaker := func(picture string) string {
		return fmt.Sprintf("[[speaker]]\nid = 20000\nname = \"S\"\npicture = %q\n", picture)
	}
	material := func(sprite string) string {
		return fmt.Sprintf("[[material]]\nitem = \"Pike\"\ntier = \"Common\"\nmaterial = \"Bone\"\nallow = true\nsprite = %q\n", sprite)
	}
	townRow := func(description string) string {
		return fmt.Sprintf("[[town]]\nid = 9\ndescription = %q\n", description)
	}
	art := func(key string) string {
		return fmt.Sprintf(`{"art":[{"name":"a","format":"picture","key":%q}]}`, ModArtPrefix+key)
	}
	for _, tc := range []struct {
		name  string
		files map[string]string
		want  []string
	}{
		{"clean", map[string]string{
			ItemsFile: material("assets/items/pike.png"), "assets/items/pike.png": "p",
			BodiesFile:            strings.NewReplacer("bodies/clubman2h-heroes.png", "assets/bodies/h.png", "bodies/clubman2h-heroes_l.png", "assets/bodies/sub/l.png").Replace(goodBody),
			"assets/bodies/h.png": "p", "assets/bodies/sub/l.png": "p",
			SpeakersFile: speaker("assets/speakers/20000.png"), "assets/speakers/20000.png": "p",
			TownsFile: townRow("assets/towns/east.json"), "assets/towns/east.json": art("assets/towns/east/square.bmp"), "assets/towns/east/square.bmp": "p",
		}, nil},
		{"sprite outside", map[string]string{ItemsFile: material("art/pike.png"), "art/pike.png": "p"},
			[]string{ItemsFile + ":6: sprite art/pike.png uses an old path; use assets/items/<name>.png"}},
		{"body outside", map[string]string{BodiesFile: strings.Replace(goodBody, "bodies/clubman2h-heroes_l.png", "assets/bodies/l.png", 1), "bodies/clubman2h-heroes.png": "p", "assets/bodies/l.png": "p"},
			[]string{BodiesFile + ":3: heroes bodies/clubman2h-heroes.png uses an old path; use assets/bodies/<name>.png"}},
		{"picture outside", map[string]string{SpeakersFile: speaker("speakers/20000.png"), "speakers/20000.png": "p"},
			[]string{SpeakersFile + ":4: picture speakers/20000.png uses an old path; use assets/speakers/<name>.png"}},
		{"picture in a folder below", map[string]string{SpeakersFile: speaker("assets/speakers/old/a.png"), "assets/speakers/old/a.png": "p"},
			[]string{SpeakersFile + ":4: picture assets/speakers/old/a.png uses an old path; use assets/speakers/<name>.png"}},
		{"description and art outside", map[string]string{TownsFile: townRow("towns/t.json"), "towns/t.json": art("art/x.bmp"), "art/x.bmp": "p"},
			[]string{TownsFile + ":3: description towns/t.json uses an old path; use assets/towns/<name>.json", "towns/t.json: art mod:art/x.bmp uses an old path; use mod:assets/towns/<name>.bmp"}},
		{"sprites art in the folder", map[string]string{TownsFile: townRow("assets/towns/t.json"), "assets/towns/sq.bmp": "p", "assets/towns/p.256": "p",
			"assets/towns/t.json": `{"art":[{"name":"a","format":"picture","key":"mod:assets/towns/sq.bmp"},{"name":"b","format":"sprites","key":"mod:assets/towns/p.256"}]}`}, nil},
		{"sprites art outside", map[string]string{TownsFile: townRow("assets/towns/t.json"), "art/p.256": "p",
			"assets/towns/t.json": `{"art":[{"name":"b","format":"sprites","key":"mod:art/p.256"}]}`},
			[]string{"assets/towns/t.json: art mod:art/p.256 uses an old path; use mod:assets/towns/<name>.256"}},
		{"installed description", map[string]string{TownsFile: townRow("rom1")}, nil},
		{"no row names", map[string]string{ItemsFile: material("assets/items/pike.png"), "assets/items/pike.png": "p", "assets/items/spare.png": "p", "assets/speakers/1.png": "p", "assets/towns/loose.bmp": "p", "assets/bodies/b.png": "p"},
			[]string{"assets/bodies/b.png: no row names this file of body sheets; the game does not read it", "assets/items/spare.png: no row names this file of items; the game does not read it",
				"assets/speakers/1.png: no row names this file of speakers; the game does not read it", "assets/towns/loose.bmp: no row names this file of towns; the game does not read it"}},
		{"named without case", map[string]string{ItemsFile: material("assets/items/Pike.png"), "assets/items/pike.png": "p"}, nil},
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
			if fmt.Sprint(d.Warnings) != fmt.Sprint(tc.want) {
				t.Fatalf("warnings\n%q\nwant\n%q", d.Warnings, tc.want)
			}
			// A warned file is still described, so packed and hashed.
			for p := range tc.files {
				if !d.Admits(p) {
					t.Errorf("%s is left out of the description", p)
				}
			}
		})
	}
}

// artDescription is a town description whose one art entry has key.
func artDescription(key string) *town.Description {
	data, _ := json.Marshal(map[string]any{"art": []town.ArtSpec{{Name: "a", Format: "picture", Key: key}}})
	d, err := town.Parse(data)
	if err != nil {
		panic(err)
	}
	return d
}

// A map whose file name is at the bound of a packed name loads: the text
// folder derived from it is one byte longer and is passed over as absent.
func TestMissionMapNameAtTheBoundLoads(t *testing.T) {
	for _, size := range []int{maxPackedNameBytes - 1, maxPackedNameBytes} {
		name := "maps/" + strings.Repeat("n", size-len(".alm")) + ".alm"
		dir := t.TempDir()
		writeTreeFile(t, dir, name, recordedMap(t, nil))
		if _, err := MissionMap.Path(name); err != nil {
			t.Fatalf("%d bytes: the rule refuses the map: %v", size, err)
		}
		m := Mission{Map: name, File: MissionsFile, Line: 1}
		if _, _, err := m.ReadTexts(dir, "en"); err != nil {
			t.Errorf("%d bytes: the texts answered %v", size, err)
		}
		if err := m.ReadFiles(dir); err != nil {
			t.Errorf("%d bytes: the files answered %v", size, err)
		}
	}
}

func TestAssetPlaces(t *testing.T) {
	want := map[string]string{"items": "assets/items .png", "body sheets": "assets/bodies .png", "towns": "assets/towns .json .bmp", "speakers": "assets/speakers .png"}
	for _, k := range TreeKinds() {
		place, exts := k.AssetPlace()
		got := strings.TrimSpace(place + " " + strings.Join(exts, " "))
		if got != want[k.Name] {
			t.Errorf("%s: %q, want %q", k.Name, got, want[k.Name])
		}
		if place == "" {
			continue
		}
		if found, ok := AssetKind(strings.TrimPrefix(place, AssetsFolder+"/")); !ok || found.Name != k.Name {
			t.Errorf("%s is not found by its folder", k.Name)
		}
	}
}
