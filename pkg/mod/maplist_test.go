package mod

import (
	"errors"
	"reflect"
	"strconv"
	"strings"
	"testing"
)

func TestMapListJoinsEveryMapToItsDeclarations(t *testing.T) {
	files := []string{
		"mod.toml", "data/missions.toml",
		"maps/first.alm", "maps/first.text/en/title.txt", "maps/first.text/ru/title.txt", "maps/first.variables",
		"maps/shared.almx", "maps/shared.text/title.txt",
		"maps/spare.alm",
		"10.alm", "maps/20.alm", "maps/30.alm",
		"maps/en/talk.alm",
		"maps/en/loose.alm", "maps/ru/loose.alm",
		"mission/old/map.alm",
	}
	row := func(key, kind string, n, line int, m string) Mission {
		return Mission{Mod: "m", File: MissionsFile, Line: line, Key: key, Kind: kind, Number: n, Map: m}
	}
	talk := row("talk", MissionGuest, 24, 20, "maps/en/talk.alm")
	talk.mapTemplate = "maps/{lang}/talk.alm"
	d := MissionData{Missions: []Mission{
		row("first", MissionGuest, 21, 2, "maps/first.alm"),
		row("again", MissionGuest, 22, 8, "maps/First.alm"),
		row("shared", MissionReplace, 40, 12, "maps/shared.almx"),
		row("gone", MissionGuest, 23, 16, "maps/gone.alm"),
		talk,
		row("thirty", MissionReplace, 30, 24, "maps/30.alm"),
		row("old", MissionGuest, 25, 28, "mission/old/map.alm"),
		row("other", MissionGuest, 26, 1, "maps/spare.alm"),
		{Mod: "m", Key: "off", Off: true, Number: 27},
		{Mod: "m", File: "10.alm", Kind: MissionReplace, Number: 10, Map: "10.alm", byName: true},
		{Mod: "m", File: "maps/20.alm", Kind: MissionReplace, Number: 20, Map: "maps/20.alm", byName: true},
	}}
	d.Missions[7].Mod = "another"
	list := MapList(files, "m", d)
	type line struct {
		path  string
		role  MapRole
		rows  string
		extra string
	}
	var got []line
	for _, e := range list {
		var keys []string
		for _, r := range e.Rows {
			keys = append(keys, r.Key)
		}
		extra := strings.Join(e.Languages, ",") + "|" + strings.Join(e.PageLanguages, ",") + "|" + strings.Join(e.Lacking, ",")
		if e.Labels {
			extra += "|labels"
		}
		if e.ByName {
			extra += "|" + strconv.Itoa(e.Number)
		}
		got = append(got, line{e.Path, e.Role(), strings.Join(keys, ","), extra})
	}
	want := []line{
		{"maps/first.alm", MapRows, "first,again", "|en,ru||labels"},
		{"maps/shared.almx", MapRows, "shared", "||"},
		{"maps/gone.alm", MapMissing, "gone", "||maps/gone.alm"},
		{"maps/{lang}/talk.alm", MapRows, "talk", "en||maps/ru/talk.alm"},
		{"maps/30.alm", MapRows, "thirty", "||"},
		{"mission/old/map.alm", MapRows, "old", "||"},
		{"10.alm", MapByName, "", "|||10"},
		{"maps/20.alm", MapByName, "", "|||20"},
		{"maps/{lang}/loose.alm", MapUnused, "", "en,ru||"},
		{"maps/spare.alm", MapUnused, "", "||"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("map list:\n got %v\nwant %v", got, want)
	}
	if e := list[0]; e.Stem != "first" || e.Ext != ".alm" || e.Files[0] != "maps/first.alm" {
		t.Fatalf("first entry %+v", e)
	}
	if e := list[1]; len(e.PageLanguages) != 1 || e.PageLanguages[0] != "" {
		t.Fatalf("shared pages %q", e.PageLanguages)
	}
	if got := list[3].File("ru"); got != "maps/ru/talk.alm" {
		t.Fatalf("language file %q", got)
	}
	warnings := MapWarnings(list)
	wantWarnings := []string{
		"data/missions.toml:16: map maps/gone.alm is missing",
		"maps/en/loose.alm: no mission names this map; the game does not play it",
		"maps/ru/loose.alm: no mission names this map; the game does not play it",
		"maps/spare.alm: no mission names this map; the game does not play it",
	}
	if !reflect.DeepEqual(warnings, wantWarnings) {
		t.Fatalf("warnings:\n got %q\nwant %q", warnings, wantWarnings)
	}
}

// The file form is the loader's decision: a file named by a mission number
// that the loader did not keep as a mission is an unused map, warned about.
func TestMapListTakesTheFileFormFromTheLoader(t *testing.T) {
	files := []string{"mod.toml", "maps/7.alm", "maps/900.alm"}
	file := func(n int, rel string) missionMap {
		return missionMap{Mission: Mission{Mod: "m", File: rel, Kind: MissionReplace, Number: n, Map: rel, byName: true}}
	}
	d := MissionData{maps: []missionMap{file(7, "maps/7.alm"), file(900, "maps/900.alm")}}
	admitted, _, err := d.Listed(func(n int) error {
		if n == 900 {
			return errors.New("the install holds no campaign map of this number")
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	list := MapList(files, "m", admitted)
	if len(list) != 2 || list[0].Role() != MapByName || list[0].Number != 7 || list[1].Role() != MapUnused || list[1].ByName {
		t.Fatalf("list %+v", list)
	}
	want := []string{"maps/900.alm: no mission names this map; the game does not play it"}
	if got := MapWarnings(list); !reflect.DeepEqual(got, want) {
		t.Fatalf("warnings %q", got)
	}
}

func TestMapListOfAModWithoutMaps(t *testing.T) {
	if list := MapList([]string{"mod.toml", "data/items.toml"}, "m", MissionData{}); len(list) != 0 {
		t.Fatalf("list %v", list)
	}
}
