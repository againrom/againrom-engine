package mod

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// The description holds rows and number-named maps; Listed keeps every map
// with no answer and only the held ones with the install's.
func TestDescribeMissionsHoldsTheMapsThatReplaceMissionsByName(t *testing.T) {
	root := t.TempDir()
	for _, tc := range []struct {
		id    string
		files map[string]string
		rows  []int
	}{
		{"byname", map[string]string{ManifestName: good("byname"), "7.alm": "map", "maps/12.alm": "map", "maps/notes.alm": "map"}, nil},
		{"mixed", map[string]string{ManifestName: good("mixed"), "7.alm": "map", "maps/12.alm": "map", "maps/30.alm": "map",
			MissionsFile: "[[mission]]\nkey = \"thirty\"\nkind = \"replace\"\nnumber = 30\nmap = \"maps/30.alm\"\n"}, []int{30}},
	} {
		dir := filepath.Join(root, tc.id)
		for name, body := range tc.files {
			path := filepath.Join(dir, filepath.FromSlash(name))
			if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
				t.Fatal(err)
			}
		}
		_, packed, err := Pack(dir)
		if err != nil {
			t.Fatal(err)
		}
		file := filepath.Join(t.TempDir(), PackedName(tc.id))
		if err := os.WriteFile(file, packed, 0o644); err != nil {
			t.Fatal(err)
		}
		for _, at := range []string{dir, file} {
			d, _, err := DescribeMissions(at, tc.id, "en")
			if err != nil {
				t.Fatalf("%s: %v", at, err)
			}
			if got := numbersOf(d.Missions); !reflect.DeepEqual(got, tc.rows) {
				t.Fatalf("%s: rows %v, want %v", at, got, tc.rows)
			}
			all, warnings, err := d.Listed(nil)
			if want := append(append([]int(nil), tc.rows...), 7, 12); err != nil || len(warnings) != 0 || !reflect.DeepEqual(numbersOf(all.Playable()), want) {
				t.Fatalf("%s: listed %v, %q, %v; want %v", at, numbersOf(all.Playable()), warnings, err, want)
			}
			for _, m := range all.Missions[len(tc.rows):] {
				if !m.ByFileName() || m.Kind != MissionReplace || m.Map != map[int]string{7: "7.alm", 12: "maps/12.alm"}[m.Number] || len(m.MapBytes) != 0 {
					t.Fatalf("%s: a file form mission %+v", at, m)
				}
			}
			held, warnings, err := d.Listed(func(n int) error {
				if n == 12 {
					return nil
				}
				return errors.New("no such mission")
			})
			if want := append(append([]int(nil), tc.rows...), 12); err != nil || len(warnings) != 1 || !strings.Contains(warnings[0], "7.alm") || !reflect.DeepEqual(numbersOf(held.Missions), want) {
				t.Fatalf("%s: answered %v, %q, %v; want %v", at, numbersOf(held.Missions), warnings, err, want)
			}
			row := Mission{Mod: tc.id, File: MissionsFile, Key: "twelve", Kind: MissionReplace, Number: 12, Map: "maps/12.alm"}
			joined, _, err := d.WithMissions(append(append([]Mission(nil), d.Missions...), row)).Listed(nil)
			if want := append(append([]int(nil), tc.rows...), 7, 12); err != nil || !reflect.DeepEqual(numbersOf(joined.Missions), want) || joined.Missions[len(tc.rows)+1].ByFileName() {
				t.Fatalf("%s: with a row naming maps/12.alm %v, %v; want %v", at, numbersOf(joined.Missions), err, want)
			}
			again, _, err := d.WithMissions(all.Missions).Listed(nil)
			if err != nil || !reflect.DeepEqual(numbersOf(again.Missions), numbersOf(all.Missions)) {
				t.Fatalf("%s: listed twice %v, %v", at, numbersOf(again.Missions), err)
			}
		}
	}
}

func numbersOf(list []Mission) []int {
	var out []int
	for _, m := range list {
		out = append(out, m.Number)
	}
	return out
}
