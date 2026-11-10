package mod

import (
	"fmt"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

const mapModManifest = "id = \"m\"\ntitle = \"M\"\nversion = \"1.0\"\n"

// mapMod writes a mod m holding files beside its manifest and returns its
// folder.
func mapMod(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "m")
	files["mod.toml"] = mapModManifest
	writeTree(t, dir, files)
	return dir
}

// declaredMissions reads the missions mod id at dir declares in lang as the
// launch does: the rows its data/missions.toml holds, then the missions
// kind's file form.
func declaredMissions(t *testing.T, d *Data, id, dir, lang string) (bool, error) {
	t.Helper()
	r := DataReader{Mod: id, Dir: dir, Lang: lang}
	if src, _, err := ReadInside(dir, MissionsFile); err == nil {
		if err := readMissions(r, d, MissionsFile, src); err != nil {
			return false, err
		}
	}
	tree, err := OpenTree(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, k := range DataKinds() {
		if k.Name == "missions" {
			return k.ReadNamed(r, d, tree)
		}
	}
	t.Fatal("no missions kind")
	return false, nil
}

// campaignOf answers the counterpart test of an install whose campaign holds
// the maps of numbers.
func campaignOf(numbers ...int) func(int) error {
	return func(n int) error {
		for _, held := range numbers {
			if n == held {
				return nil
			}
		}
		return fmt.Errorf("number %d is no mission of this install's campaign", n)
	}
}

// Rule 1: only the top and maps/ are read, and only names that are a mission
// number with no leading zero and .alm or .almx, without case.
func TestTheFileFormReadsNumberedMapsInTheTopAndTheMapsFolder(t *testing.T) {
	dir := mapMod(t, map[string]string{
		"10.ALM": "TEN", "maps/20.almx": "TWENTY",
		"010.alm": "x", "0.alm": "x", "10a.alm": "x", "10.alm.bak": "x", "x10.alm": "x", "70000.alm": "x",
		"50.map": "x", "60.alm/inside.txt": "x", "other/30.alm": "x", "maps/sub/40.alm": "x",
	})
	var d Data
	held, err := declaredMissions(t, &d, "m", dir, "en")
	if err != nil || !held {
		t.Fatalf("held %v: %v", held, err)
	}
	got, warnings, err := d.Missions.AdmitMaps(campaignOf(10, 20, 30, 40, 60))
	if err != nil || len(warnings) != 0 {
		t.Fatalf("%v %v", warnings, err)
	}
	if len(got.Missions) != 2 {
		t.Fatalf("%d missions: %+v", len(got.Missions), got.Missions)
	}
	for n, want := range map[int][2]string{10: {"10.ALM", "TEN"}, 20: {"maps/20.almx", "TWENTY"}} {
		m, ok := got.Replacing(n)
		if !ok || m.Mod != "m" || m.Map != want[0] || m.File != want[0] || m.Line != 0 || string(m.MapBytes) != want[1] || m.Key != "" || !m.byName {
			t.Errorf("mission %d: %+v", n, m)
		}
	}
	// Loss control: a mod without a numbered map holds nothing of the form.
	var none Data
	if held, err := declaredMissions(t, &none, "m", mapMod(t, map[string]string{"010.alm": "x"}), "en"); held || err != nil || !none.Missions.Empty() {
		t.Fatalf("held %v %v %+v", held, err, none.Missions)
	}
}

// Rule 2: a file a row names as its map is that row's map and nothing else.
func TestAMapARowNamesIsThatRowsMap(t *testing.T) {
	for name, src := range map[string]string{
		"top":     "[[mission]]\nkind = \"replace\"\nkey = \"r\"\nnumber = 10\nmap = \"10.alm\"\ntitle = \"t\"\n",
		"maps":    "[[mission]]\nkind = \"replace\"\nkey = \"r\"\nnumber = 10\nmap = \"maps/10.alm\"\ntitle = \"t\"\n",
		"guest":   "[[mission]]\nkey = \"g\"\nnumber = 25\nmap = \"maps/10.alm\"\nafter = 20\nguest = 22\nchapter = 30\n",
		"guestno": "[[mission]]\nkey = \"g\"\nnumber = 10\nmap = \"maps/10.alm\"\nafter = 20\nguest = 22\nchapter = 30\n",
	} {
		file := "maps/10.alm"
		if name == "top" {
			file = "10.alm"
		}
		dir := mapMod(t, map[string]string{file: "MAP", MissionsFile: src, "text/en/strings.toml": "\"t\" = \"Title\"\n"})
		var d Data
		held, err := declaredMissions(t, &d, "m", dir, "en")
		if err != nil || held {
			t.Fatalf("%s: held %v: %v", name, held, err)
		}
		got, warnings, err := d.Missions.AdmitMaps(campaignOf(10))
		if err != nil || len(warnings) != 0 || len(got.Missions) != 1 {
			t.Fatalf("%s: %v %v %+v", name, warnings, err, got.Missions)
		}
		m := got.Missions[0]
		if m.byName || string(m.MapBytes) != "MAP" || m.Line != 1 {
			t.Errorf("%s: %+v", name, m)
		}
		if strings.HasPrefix(name, "guest") {
			if _, ok := got.Replacing(10); ok || m.Kind != MissionGuest {
				t.Errorf("%s: the guest's map replaces mission 10", name)
			}
		} else if m.Kind != MissionReplace || m.Title != "Title" || m.Number != 10 {
			t.Errorf("%s: %+v", name, m)
		}
	}
}

// Rule 3: a row of mission N with another map beside an unnamed N file, and
// two files of one number, are refused naming both files.
func TestTwoDeclarationsOfOneReplacedMissionAreRefusedNamingBoth(t *testing.T) {
	row := "[[mission]]\nkind = \"replace\"\nkey = \"r\"\nnumber = 10\nmap = \"mission/10.alm\"\n"
	for name, c := range map[string]struct {
		files map[string]string
		want  string
	}{
		"row":    {map[string]string{"mission/10.alm": "ROW", "10.alm": "FILE", MissionsFile: row}, `mod "m": data/missions.toml:1: mission 10 is replaced with mission/10.alm here, and 10.alm replaces it by its name`},
		"maps":   {map[string]string{"10.alm": "A", "maps/10.alm": "B"}, `mod "m": maps/10.alm: 10.alm is a map of mission 10 too; keep one of the two files`},
		"almx":   {map[string]string{"10.alm": "A", "10.almx": "B"}, `mod "m": 10.almx: 10.alm is a map of mission 10 too`},
		"rowmap": {map[string]string{"mission/10.alm": "ROW", "maps/10.almx": "FILE", MissionsFile: row}, `and maps/10.almx replaces it by its name; name maps/10.almx as the map`},
	} {
		var d Data
		if _, err := declaredMissions(t, &d, "m", mapMod(t, c.files), "en"); err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if _, _, err := d.Missions.AdmitMaps(campaignOf(10)); err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%s: %v, want %q", name, err, c.want)
		}
	}
}

// Rule 4: a file whose number has no counterpart in the install replaces and
// refuses nothing and comes back as one warning naming it; a file that has
// one is refused for what its map or pages refuse.
func TestAFileWithoutACounterpartReplacesAndRefusesNothing(t *testing.T) {
	var d Data
	dir := mapMod(t, map[string]string{"99.alm": "A", "maps/99.alm": "B", "99.text/Bad Name.txt": "x"})
	if _, err := declaredMissions(t, &d, "m", dir, "en"); err != nil {
		t.Fatal(err)
	}
	got, warnings, err := d.Missions.AdmitMaps(campaignOf(10))
	want := []string{
		`mod "m": 99.alm: number 99 is no mission of this install's campaign; the file replaces nothing`,
		`mod "m": maps/99.alm: number 99 is no mission of this install's campaign; the file replaces nothing`,
	}
	if err != nil || !reflect.DeepEqual(warnings, want) || !got.Empty() {
		t.Fatalf("%q %v %+v", warnings, err, got)
	}
	// The same pages under a number the install holds refuse the mod.
	var held Data
	if _, err := declaredMissions(t, &held, "m", mapMod(t, map[string]string{"10.alm": "A", "10.text/Bad Name.txt": "x"}), "en"); err != nil {
		t.Fatal(err)
	}
	if _, _, err := held.Missions.AdmitMaps(campaignOf(10)); err == nil || !strings.Contains(err.Error(), `mod "m": 10.alm: texts: `) {
		t.Fatalf("%v", err)
	}
}

// Rule 5: one file serves the English and the Russian game.
func TestOneMapFileServesBothLanguages(t *testing.T) {
	dir := mapMod(t, map[string]string{"10.alm": "MAP"})
	var maps []Mission
	for _, lang := range []string{"en", "ru"} {
		var d Data
		if _, err := declaredMissions(t, &d, "m", dir, lang); err != nil {
			t.Fatal(err)
		}
		got, _, err := d.Missions.AdmitMaps(campaignOf(10))
		if err != nil || len(got.Missions) != 1 {
			t.Fatalf("%s: %v %+v", lang, err, got)
		}
		maps = append(maps, got.Missions[0])
	}
	if maps[0].Map != maps[1].Map || string(maps[0].MapBytes) != string(maps[1].MapBytes) || string(maps[1].MapBytes) != "MAP" {
		t.Fatalf("%+v", maps)
	}
}

// Rule 6: the check's lines name each replacement, the file form's and the
// row form's.
func TestReplacementsNameEachReplacedMission(t *testing.T) {
	var d Data
	dir := mapMod(t, map[string]string{"10.alm": "FILE", "mission/20.alm": "ROW",
		MissionsFile: "[[mission]]\nkind = \"replace\"\nkey = \"r\"\nnumber = 20\nmap = \"mission/20.alm\"\n"})
	if _, err := declaredMissions(t, &d, "m", dir, "en"); err != nil {
		t.Fatal(err)
	}
	got, _, err := d.Missions.AdmitMaps(campaignOf(10, 20))
	if err != nil {
		t.Fatal(err)
	}
	want := []string{
		`mission 20 replaced by mod "m": map mission/20.alm, declared at data/missions.toml:1`,
		`mission 10 replaced by mod "m": map 10.alm, named by its file name`,
	}
	if lines := got.Replacements(); !reflect.DeepEqual(lines, want) {
		t.Fatalf("%q", lines)
	}
}

// A packed mod declares what its folder declares.
func TestAPackedModsMapsAreItsFolders(t *testing.T) {
	dir := mapMod(t, map[string]string{"10.ALM": "TEN", "maps/20.almx": "TWENTY", "maps/99.alm": "NONE", "10.text/en/event01.txt": "E1"})
	packed := packInto(t, dir, t.TempDir())
	view := func(dir string) ([]Mission, []string) {
		var d Data
		if _, err := declaredMissions(t, &d, "m", dir, "en"); err != nil {
			t.Fatal(err)
		}
		got, warnings, err := d.Missions.AdmitMaps(campaignOf(10, 20))
		if err != nil {
			t.Fatal(err)
		}
		return got.Missions, warnings
	}
	folder, folderWarnings := view(dir)
	res, resWarnings := view(packed)
	if len(folder) != 2 || string(folder[0].TextFiles["event01.txt"]) != "E1" || len(folderWarnings) != 1 {
		t.Fatalf("folder: %+v %q", folder, folderWarnings)
	}
	if !reflect.DeepEqual(folder, res) || !reflect.DeepEqual(folderWarnings, resWarnings) {
		t.Fatalf("packed %+v %q\nfolder %+v %q", res, resWarnings, folder, folderWarnings)
	}
}

// The file form's missions keep their place in load order, so a later mod's
// row of the same mission overrides an earlier mod's file when load-after
// orders them, and the two are refused without that order.
func TestAFileAndALaterModsRowOfOneMissionMergeInLoadOrder(t *testing.T) {
	first := mapMod(t, map[string]string{"20.alm": "FILE"})
	second := filepath.Join(t.TempDir(), "r")
	writeTree(t, second, map[string]string{"mod.toml": "id = \"r\"\ntitle = \"R\"\nversion = \"1.0\"\n", "r.alm": "ROW",
		MissionsFile: "[[mission]]\nkind = \"replace\"\nkey = \"r\"\nnumber = 20\nmap = \"r.alm\"\n"})
	var d Data
	if _, err := declaredMissions(t, &d, "m", first, "en"); err != nil {
		t.Fatal(err)
	}
	if _, err := declaredMissions(t, &d, "r", second, "en"); err != nil {
		t.Fatal(err)
	}
	admitted, _, err := d.Missions.AdmitMaps(campaignOf(20))
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := MergeMissions(admitted); err == nil || !strings.Contains(err.Error(), "neither loads after the other") {
		t.Fatalf("without load-after: %v", err)
	}
	admitted.Order = Precedence{"r": {"m"}}
	merged, overrides, err := MergeMissions(admitted)
	if err != nil || len(overrides) != 1 {
		t.Fatalf("%v %v", overrides, err)
	}
	if m, ok := merged.Replacing(20); !ok || m.Mod != "r" || string(m.MapBytes) != "ROW" {
		t.Fatalf("%+v", m)
	}
}
