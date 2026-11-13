package modrt

import (
	"strings"
	"testing"

	"againrom/pkg/mod"
)

// The check's line about a mod's maps reads the mod's one map list: a map
// file no mission names is warned about at its path, and a map a mission
// names or a file name makes a replacement gives no line. A mission whose
// map the mod does not hold stays the refusal it was.
func TestCheckWarnsOfUnusedMaps(t *testing.T) {
	used := "[[mission]]\nkind = \"replace\"\nkey = \"used\"\nnumber = 20\nmap = \"maps/used.alm\"\n"
	entries := makeMods(t, map[string]map[string]string{"maps": {
		mod.MissionsFile: used, "maps/used.alm": "map", "maps/spare.alm": "map", "maps/7.alm": "map",
	}})
	res, err := Load(entries, "rom1-en", nil, Options{})
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, w := range res.Warnings {
		if strings.Contains(w, "map") {
			got = append(got, w)
		}
	}
	if len(got) != 1 || !strings.Contains(got[0], "maps/spare.alm: no mission names this map; the game does not play it") {
		t.Fatalf("the map warnings:\n%s", strings.Join(res.Warnings, "\n"))
	}
	// A map with the language in its path may hold one language's file alone.
	talk := "[[mission]]\nkind = \"replace\"\nkey = \"talk\"\nnumber = 22\nmap = \"maps/{lang}/talk.alm\"\n"
	entries = makeMods(t, map[string]map[string]string{"maps": {mod.MissionsFile: talk, "maps/en/talk.alm": "map"}})
	res, err = Load(entries, "rom1-en", nil, Options{})
	if err != nil || len(res.Warnings) != 0 {
		t.Fatalf("a map of one language: %v: %v", res.Warnings, err)
	}
	gone := used + "[[mission]]\nkind = \"replace\"\nkey = \"gone\"\nnumber = 21\nmap = \"maps/gone.alm\"\n"
	entries = makeMods(t, map[string]map[string]string{"maps": {mod.MissionsFile: gone, "maps/used.alm": "map"}})
	if _, err := Load(entries, "rom1-en", nil, Options{}); err == nil || !strings.Contains(err.Error(), "maps/gone.alm: no such file in the mod folder") {
		t.Fatalf("a mission whose map is missing: %v", err)
	}
}
