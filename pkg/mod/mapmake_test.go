package mod

import (
	"errors"
	"reflect"
	"strings"
	"testing"
)

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

func TestMakeMissionGuestIsTheEditorsDefaultGuestNamingTheMap(t *testing.T) {
	files := []string{"mod.toml", MissionsFile, "maps/first.alm", "maps/second.alm"}
	d := MissionData{Missions: []Mission{
		{Mod: "m", File: MissionsFile, Key: "first", Kind: MissionGuest, Number: 21, After: 0, Map: "maps/first.alm"},
	}}
	e := entryOf(t, MapList(files, "m", d), "maps/second.alm")
	guest := Mission{Mod: "x", Key: "map", Kind: MissionGuest, Number: 23, After: 21, Party: []GuestMember{{Character: "c"}}}
	row, err := MakeMission("m", d, e, guest, 0, heldOnly(22))
	if err != nil {
		t.Fatalf("make guest: %v", err)
	}
	want := Mission{Mod: "m", File: MissionsFile, Key: "second", Kind: MissionGuest, Number: 23, After: 21, Map: "maps/second.alm", Party: []GuestMember{{Character: "c"}}}
	if !reflect.DeepEqual(row, want) {
		t.Fatalf("guest row:\n got %+v\nwant %+v", row, want)
	}
}

func TestMakeMissionReplaceNTakesAMissionTheInstallHolds(t *testing.T) {
	files := []string{MissionsFile, "maps/second.alm"}
	d := MissionData{}
	e := entryOf(t, MapList(files, "m", d), "maps/second.alm")
	row, err := MakeMission("m", d, e, Mission{}, 40, heldOnly(40))
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
	if _, err := MakeMission("m", d, e, Mission{}, 41, heldOnly(40)); err == nil {
		t.Fatal("replace 41 accepted; the install holds 40 only")
	}
}

func TestMakeMissionRefusesAMapThatAlreadyHasRows(t *testing.T) {
	files := []string{MissionsFile, "maps/first.alm"}
	d := MissionData{Missions: []Mission{{Mod: "m", File: MissionsFile, Key: "first", Kind: MissionGuest, Number: 21, Map: "maps/first.alm"}}}
	e := entryOf(t, MapList(files, "m", d), "maps/first.alm")
	if _, err := MakeMission("m", d, e, Mission{}, 0, heldOnly()); err == nil {
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

func TestPlaceFolderMapsTakeTheNextSuffixAgainstTheModsFiles(t *testing.T) {
	files := []string{"maps/first.alm", "maps/second-2.almx", "maps/third.variables", "maps/first-2.alm"}
	folder := FolderMaps([]string{"first.alm", "second.almx", "third.alm", "fourth.alm", "12.alm"})
	got := PlaceFolderMaps(files, folder)
	want := map[string]string{
		"first.alm":   "maps/first-3.alm",
		"second.almx": "maps/second.almx",
		"third.alm":   "maps/third-2.alm",
		"fourth.alm":  "maps/fourth.alm",
		"12.alm":      "maps/m12.alm",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("places:\n got %v\nwant %v", got, want)
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

func TestMakeRowKeyIsAValidRowKeyOfTheStem(t *testing.T) {
	files := []string{MissionsFile, "10.alm"}
	d := MissionData{Missions: []Mission{{Mod: "m", File: "10.alm", Kind: MissionReplace, Number: 10, Map: "10.alm", byName: true}}}
	e := entryOf(t, MapList(files, "m", d), "10.alm")
	row, err := MakeRow("m", d, e, heldOnly(10))
	if err != nil || row.Key != "m10" {
		t.Fatalf("key %q, %v", row.Key, err)
	}
	for stem, want := range map[string]string{"Big-Map": "big_map", "": "m"} {
		if got := mapKey(MissionData{}, stem); got != want {
			t.Errorf("mapKey(%q) = %q, want %q", stem, got, want)
		}
	}
}

func TestReplaceNumbersAreTheHeldOnesNoRowReplaces(t *testing.T) {
	d := MissionData{Missions: []Mission{{Mod: "m", File: MissionsFile, Key: "a", Kind: MissionReplace, Number: 20, Map: "a.alm"}}}
	got := d.ReplaceNumbers([]int{40, 20, 30, 50, 30}, heldOnly(20, 30, 40))
	if len(got) != 2 || got[0] != 30 || got[1] != 40 {
		t.Fatalf("numbers %v, want [30 40]", got)
	}
	if got := d.ReplaceNumbers([]int{30}, nil); len(got) != 0 {
		t.Fatalf("with no install %v", got)
	}
}
