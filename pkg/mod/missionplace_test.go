package mod

import (
	"path"
	"slices"
	"testing"
)

func TestMissionStemTakesTheSourceFileName(t *testing.T) {
	for name, want := range map[string]string{
		"C:/maps/Crypt.alm":      "Crypt",
		"loose/deep/b.c.almx":    "b.c",
		"scenario/12.alm":        "m12",
		"7":                      "m7",
		"":                       "map",
		"new/":                   "new",
		"dir/bad:name.alm":       "map",
		"dir/.hidden.alm":        "map",
		"dir/trail .alm":         "map",
		"dir/{lang}.alm":         "map",
		"dir/ends.with.dot..alm": "map",
	} {
		if got := MissionStem(name); got != want {
			t.Errorf("MissionStem(%q) = %q, want %q", name, got, want)
		}
	}
	if _, ok := missionMapNumber(path.Base(PlaceMission(MissionStem("scenario/12.alm"), false).Map)); ok {
		t.Fatal("a numbered source is placed as the file that replaces a mission by name")
	}
}

func TestMissionPlaceNamesTheTreeRows(t *testing.T) {
	for _, tc := range []struct {
		place                      MissionPlace
		mapPath, pages, en, labels string
	}{
		{PlaceMission("crypt", false), "maps/crypt.alm", "maps/crypt.text", "maps/crypt.text/en", "maps/crypt.variables"},
		{PlaceMission("crypt", true), "maps/crypt.almx", "maps/crypt.text", "maps/crypt.text/en", "maps/crypt.variables"},
		{PlaceLanguageMission("ru", "m10", false), "maps/ru/m10.alm", "maps/ru/m10.text", "maps/ru/m10.text/en", "maps/ru/m10.variables"},
		{PlaceLanguageMission("en", "m10", true), "maps/en/m10.almx", "maps/en/m10.text", "maps/en/m10.text/en", "maps/en/m10.variables"},
	} {
		p := tc.place
		if p.Map != tc.mapPath || p.Pages("") != tc.pages || p.Pages("en") != tc.en || p.Labels() != tc.labels || !p.InTree() {
			t.Fatalf("place %+v: %s %s %s %s %v", tc, p.Map, p.Pages(""), p.Pages("en"), p.Labels(), p.InTree())
		}
		for _, f := range []string{p.Map, p.Pages("") + "/event01.txt", p.Pages("ru") + "/event01.txt", p.Labels()} {
			if !IsMissionFile(f) {
				t.Errorf("%s is not a file of the tree's mission rows", f)
			}
		}
	}
	for _, f := range []string{"mission/map.alm", "mission/text/en/event01.txt", "maps/crypt.text/en/deep/x.txt", "maps/crypt.txt"} {
		if IsMissionFile(f) {
			t.Errorf("%s is taken as a file of the tree's mission rows", f)
		}
	}
	if (MissionPlace{Map: "mission/{lang}/map.alm"}).InTree() {
		t.Fatal("an old row map reads as a tree map")
	}
	maps, pages, labels := missionTreeRows()
	for name, want := range map[string][]string{missionMapsKind: maps, missionPagesKind: pages, mapLabelsKind: labels} {
		i := slices.IndexFunc(dataKinds, func(k DataKind) bool { return k.Name == name })
		if i < 0 || !slices.Equal(dataKinds[i].Paths[:len(want)], want) {
			t.Fatalf("tree row %q does not name the place rule %v", name, want)
		}
	}
}

func TestMissionPlaceFreeSkipsTakenStems(t *testing.T) {
	held := []string{"MAPS/Crypt.ALMX", "maps/crypt-2.text/en/a.txt", "maps/crypt-3.variables", "maps/crypt-5.alm"}
	if got := PlaceMission("crypt", false).Free(held).Map; got != "maps/crypt-4.alm" {
		t.Fatalf("free place %s", got)
	}
	if got := PlaceMission("other", true).Free(held).Map; got != "maps/other.almx" {
		t.Fatalf("free place %s", got)
	}
}
