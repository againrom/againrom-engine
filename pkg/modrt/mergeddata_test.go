package modrt

import (
	"os"
	"path/filepath"
	"sort"
	"testing"
)

// fixtureMods lists the fixture mods in testdata/mods.
func fixtureMods(t *testing.T) []string {
	t.Helper()
	dirs, err := os.ReadDir(filepath.Join("testdata", "mods"))
	if err != nil {
		t.Fatal(err)
	}
	var out []string
	for _, d := range dirs {
		if d.IsDir() {
			out = append(out, d.Name())
		}
	}
	sort.Strings(out)
	return out
}

// fixtureCompletions are the files a fixture mod needs that the repository
// does not hold, which the tests that read the mod write beside it: a town
// description and a guest mission's maps.
var fixtureCompletions = map[string]map[string]string{
	"east-town": {"towns/east.json": `{"town": "east"}`},
	"guest-chain": {"maps/en/m10.alm": "map en", "maps/ru/m10.alm": "map ru",
		"maps/en/m10.text/en/event01.txt": "event en", "maps/ru/m10.text/ru/event01.txt": "event ru"},
	"reniesta": {"maps/en/m10.alm": "map en", "maps/ru/m10.alm": "map ru",
		"maps/en/m10.text/en/event01.txt": "event en", "maps/ru/m10.text/ru/event01.txt": "event ru"},
}
