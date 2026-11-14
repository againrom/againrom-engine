package mod

import (
	"errors"
	"reflect"
	"strings"
	"testing"
)

// heldOnly is the install's answer in these tests: nil for the numbers it
// holds as missions, an error for the rest.
func heldOnly(numbers ...int) func(int) error {
	return func(n int) error {
		for _, h := range numbers {
			if h == n {
				return nil
			}
		}
		return errors.New("not a mission of the install")
	}
}

func entryOf(t *testing.T, list []MapEntry, path string) MapEntry {
	t.Helper()
	for _, e := range list {
		if e.Path == path {
			return e
		}
	}
	t.Fatalf("no map entry %q in %v", path, list)
	return MapEntry{}
}

func TestMakeMissionGuestPlaysAfterTheLastGuestAndTakesAFreeNumber(t *testing.T) {
	files := []string{"mod.toml", MissionsFile, "maps/first.alm", "maps/second.alm"}
	d := MissionData{Missions: []Mission{
		{Mod: "m", File: MissionsFile, Key: "first", Kind: MissionGuest, Number: 21, After: 0, Map: "maps/first.alm"},
	}}
	e := entryOf(t, MapList(files, "m", d), "maps/second.alm")
	// 22 is a mission of the install, so the free number is 23.
	row, err := MakeMission("m", d, e, 0, heldOnly(22))
	if err != nil {
		t.Fatalf("make guest: %v", err)
	}
	want := Mission{Mod: "m", File: MissionsFile, Key: "second", Kind: MissionGuest, Number: 23, After: 21, Map: "maps/second.alm"}
	if !reflect.DeepEqual(row, want) {
		t.Fatalf("guest row:\n got %+v\nwant %+v", row, want)
	}
}

func TestMakeMissionReplaceNTakesAMissionTheInstallHolds(t *testing.T) {
	files := []string{MissionsFile, "maps/second.alm"}
	d := MissionData{}
	e := entryOf(t, MapList(files, "m", d), "maps/second.alm")
	row, err := MakeMission("m", d, e, 40, heldOnly(40))
	if err != nil {
		t.Fatalf("make replace: %v", err)
	}
	if row.Kind != MissionReplace || row.Number != 40 || row.Map != "maps/second.alm" {
		t.Fatalf("replace row %+v", row)
	}
}

func TestMakeMissionRefusesANumberTheInstallLacks(t *testing.T) {
	files := []string{MissionsFile, "maps/second.alm"}
	d := MissionData{}
	e := entryOf(t, MapList(files, "m", d), "maps/second.alm")
	if _, err := MakeMission("m", d, e, 41, heldOnly(40)); err == nil {
		t.Fatal("replace 41 accepted; the install holds 40 only")
	}
}

func TestMakeMissionRefusesAMapThatAlreadyHasRows(t *testing.T) {
	files := []string{MissionsFile, "maps/first.alm"}
	d := MissionData{Missions: []Mission{{Mod: "m", File: MissionsFile, Key: "first", Kind: MissionGuest, Number: 21, Map: "maps/first.alm"}}}
	e := entryOf(t, MapList(files, "m", d), "maps/first.alm")
	if _, err := MakeMission("m", d, e, 0, heldOnly()); err == nil {
		t.Fatal("make mission on a map a row names was accepted")
	}
}

func TestMakeRowNamesTheMapThatReplacesByFileName(t *testing.T) {
	files := []string{MissionsFile, "10.alm"}
	d := MissionData{Missions: []Mission{{Mod: "m", File: "10.alm", Kind: MissionReplace, Number: 10, Map: "10.alm", byName: true}}}
	e := entryOf(t, MapList(files, "m", d), "10.alm")
	row, err := MakeRow("m", d, e, heldOnly(10))
	if err != nil {
		t.Fatalf("make row: %v", err)
	}
	if row.Kind != MissionReplace || row.Number != 10 || row.Map != "10.alm" {
		t.Fatalf("row %+v", row)
	}
}

func TestPlaceMapKeepsTheStemOrTakesTheNextSuffix(t *testing.T) {
	files := []string{"maps/first.alm", "maps/second-2.almx", "maps/x/first.alm"}
	cases := []struct{ stem, ext, want string }{
		{"third", ".alm", "maps/third.alm"},
		{"first", ".alm", "maps/first-2.alm"},
		{"First", ".almx", "maps/First-2.almx"},
		{"second", ".almx", "maps/second.almx"},
		{"second", ".alm", "maps/second.alm"},
	}
	for _, c := range cases {
		if got := PlaceMap(files, c.stem, c.ext); got != c.want {
			t.Errorf("PlaceMap(%q, %q) = %q, want %q", c.stem, c.ext, got, c.want)
		}
	}
	files = append(files, "maps/first-2.alm")
	if got := PlaceMap(files, "first", ".alm"); got != "maps/first-3.alm" {
		t.Errorf("third copy = %q, want maps/first-3.alm", got)
	}
}

func TestFolderMapsTakeEachStemWithItsPagesAndLabels(t *testing.T) {
	var names []string
	for _, stem := range []string{"alpha", "beta", "gamma", "delta", "eps", "zeta", "eta"} {
		names = append(names, stem+".alm", stem+".text/en/title.txt", stem+".text/ru/title.txt", stem+".variables")
	}
	names = append(names, "readme.txt", "beta.text/en/brief.txt", "sub/other.alm")
	got := FolderMaps(names)
	if len(got) != 7 {
		t.Fatalf("maps = %d, want 7: %+v", len(got), got)
	}
	beta := got[1]
	if beta.Name != "beta.alm" || len(beta.Files) != 5 {
		t.Fatalf("beta = %+v", beta)
	}
	if !strings.Contains(strings.Join(beta.Files, ","), "beta.text/en/brief.txt") {
		t.Fatalf("beta pages %v", beta.Files)
	}
}

func TestPlaceFolderMapsGivesEachCopyItsOwnStem(t *testing.T) {
	files := []string{"maps/alpha.alm"}
	folder := FolderMaps([]string{"alpha.alm", "alpha.text/en/title.txt", "alpha.variables", "x.almx", "x.alm"})
	places := PlaceFolderMaps(files, folder)
	// x.alm sorts before x.almx: the first copy of a stem keeps it, the
	// second takes -2.
	want := map[string]string{
		"alpha.alm":               "maps/alpha-2.alm",
		"alpha.text/en/title.txt": "maps/alpha-2.text/en/title.txt",
		"alpha.variables":         "maps/alpha-2.variables",
		"x.alm":                   "maps/x.alm",
		"x.almx":                  "maps/x-2.almx",
	}
	if !reflect.DeepEqual(places, want) {
		t.Fatalf("places:\n got %v\nwant %v", places, want)
	}
}
