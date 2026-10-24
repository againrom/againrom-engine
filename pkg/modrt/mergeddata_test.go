package modrt

import (
	"bufio"
	"crypto/sha256"
	"encoding/json"
	flagpkg "flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"againrom/pkg/mod"
)

var updateMergedData = flagpkg.Bool("update-merged-data", false, "rewrite testdata/merged-data.txt from the fixture mods")

const mergedDataFile = "testdata/merged-data.txt"

// mergedDataBases are the bases the fixture mods are read on: the text of a
// mod is read in the base's language.
var mergedDataBases = []string{"rom1-en", "rom1-ru"}

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

// mergedDataLines is what a mod set gives on a base, one line per kind of
// data it holds: the kind and the digest of its canonical form, or the
// refusal. The paths of the mods are read as their ids.
func mergedDataLines(t *testing.T, name, base string, entries []mod.Entry) []string {
	t.Helper()
	head := name + " " + base + " "
	res, err := Load(entries, base, nil, Options{})
	if err != nil {
		msg := err.Error()
		for _, e := range entries {
			msg = strings.ReplaceAll(msg, e.Dir, "<"+e.Folder+">")
		}
		return []string{head + "refused " + msg}
	}
	// Fields the layer schema adds to a kind's records that the kind did not
	// carry before; the comparison reads the records without them.
	added := map[string][]string{"companions": {"Fields", "Off", "Order"}, "bodies": {"Off", "OffLine", "Order"},
		"characters": {"Key", "Row", "RowLine", "FaceLine", "Off", "OffLine", "Order"},
		"missions":   {"Party", "PartyLine", "Off", "OffLine", "Order"},
		"items":      {"Mods", "Order"}}
	// Lists a kind gained, read without them when a mod declares none.
	addedLists := map[string][]string{"items": {"Materials", "Weapons"}}
	kinds := []struct {
		name  string
		empty bool
		v     any
	}{
		{"items", res.Items.Empty(), res.Items},
		{"characters", res.Characters.Empty(), res.Characters},
		{"screens", res.Screens.Empty(), res.Screens},
		{"companions", res.Companions.Empty(), res.Companions},
		{"spells", res.Spells.Empty(), res.Spells},
		{"bodies", res.Bodies.Empty(), res.Bodies},
		{"missions", res.Missions.Empty(), res.Missions},
		{"towns", res.Towns.Empty(), res.Towns},
		{"files", res.Files.Empty(), res.Files},
	}
	lines := []string{fmt.Sprintf("%srules skill_cap=%d", head, res.Rules.SkillCap())}
	for _, k := range kinds {
		if k.empty {
			continue
		}
		raw, err := json.Marshal(k.v)
		if err != nil {
			t.Fatal(err)
		}
		var tree any
		if err := json.Unmarshal(raw, &tree); err != nil {
			t.Fatal(err)
		}
		if top, ok := tree.(map[string]any); ok {
			for _, key := range addedLists[k.name] {
				if top[key] == nil {
					delete(top, key)
				}
			}
		}
		tree = canonicalMergedData(tree, entries, added[k.name])
		canon, err := json.Marshal(tree)
		if err != nil {
			t.Fatal(err)
		}
		if testing.Verbose() {
			t.Logf("%s%s %s", head, k.name, canon)
		}
		lines = append(lines, fmt.Sprintf("%s%s %x", head, k.name, sha256.Sum256(canon)))
	}
	lines = append(lines, fmt.Sprintf("%smark %x", head, sha256.Sum256([]byte(res.Set.Marked().Canonical()))))
	// The merge of a merged kind that needs no install gives the records
	// the mods read when no two mods name one record.
	canon := func(v any, drop []string) string {
		raw, err := json.Marshal(v)
		if err != nil {
			t.Fatal(err)
		}
		var tree any
		if err := json.Unmarshal(raw, &tree); err != nil {
			t.Fatal(err)
		}
		out, err := json.Marshal(canonicalMergedData(tree, entries, drop))
		if err != nil {
			t.Fatal(err)
		}
		return string(out)
	}
	if !res.Companions.Empty() {
		merged, _, err := mod.MergeCompanions(res.Companions)
		if err != nil || canon(merged.Joins, added["companions"]) != canon(res.Companions.Joins, added["companions"]) {
			t.Errorf("%s: the merged join conditions differ from the records read (%v)", head, err)
		}
	}
	if !res.Missions.Empty() {
		merged, overrides, err := mod.MergeMissions(res.Missions)
		if err != nil || len(overrides) == 0 && canon(merged.Missions, added["missions"]) != canon(res.Missions.Missions, added["missions"]) {
			t.Errorf("%s: the merged missions differ from the records read (%v)", head, err)
		}
	}
	if len(res.Bodies.Bodies) > 0 {
		merged, _, err := mod.MergeBodySheets(res.Bodies)
		if err != nil || canon(merged, added["bodies"]) != canon(res.Bodies.Bodies, added["bodies"]) {
			t.Errorf("%s: the merged body sheets differ from the records read (%v)", head, err)
		}
	}
	return lines
}

// canonicalMergedData is v with each mod's path read as its id and the keys
// drop removed from every object.
func canonicalMergedData(v any, entries []mod.Entry, drop []string) any {
	switch x := v.(type) {
	case map[string]any:
		for _, k := range drop {
			delete(x, k)
		}
		for k, e := range x {
			x[k] = canonicalMergedData(e, entries, drop)
		}
		return x
	case []any:
		for i, e := range x {
			x[i] = canonicalMergedData(e, entries, drop)
		}
		return x
	case string:
		for _, e := range entries {
			x = strings.ReplaceAll(x, e.Dir, "<"+e.Folder+">")
		}
		return x
	}
	return v
}

// fixtureCompletions are the files a fixture mod needs that the repository
// does not hold, which the tests that read the mod write beside it: a town
// description and a guest mission's maps.
var fixtureCompletions = map[string]map[string]string{
	"east-town": {"towns/east.json": `{"town": "east"}`},
	"guest-chain": {"mission/en/map.alm": "map en", "mission/ru/map.alm": "map ru",
		"mission/en/text/event01.txt": "event en", "mission/ru/text/event01.txt": "event ru"},
	"reniesta": {"mission/en/map.alm": "map en", "mission/ru/map.alm": "map ru",
		"mission/en/text/event01.txt": "event en", "mission/ru/text/event01.txt": "event ru"},
}

// TestFixtureModsGiveTheRecordedMergedData reads every fixture mod alone,
// as a folder and packed, the fixtures the repository leaves incomplete
// with their missing files written, and all of them together, on each base,
// and compares what each gives with the committed record: the same records
// of every kind, the same refusals and the same mod mark.
func TestFixtureModsGiveTheRecordedMergedData(t *testing.T) {
	mods := fixtureMods(t)
	packedDir := t.TempDir()
	complete := t.TempDir()
	for _, id := range mods {
		if err := os.CopyFS(filepath.Join(complete, id), os.DirFS(filepath.Join("testdata", "mods", id))); err != nil {
			t.Fatal(err)
		}
		for name, body := range fixtureCompletions[id] {
			p := filepath.Join(complete, id, filepath.FromSlash(name))
			if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
				t.Fatal(err)
			}
		}
	}
	alone := func(dir, name, id, base string) []string {
		entries, err := mod.Resolve(dir, []string{id})
		if err != nil {
			t.Fatal(err)
		}
		lines := mergedDataLines(t, name, base, entries)
		_, packed, err := mod.Pack(entries[0].Dir)
		if err != nil {
			msg := strings.ReplaceAll(err.Error(), entries[0].Dir, "<"+id+">")
			return append(lines, name+"-packed "+base+" refused "+msg)
		}
		file := filepath.Join(packedDir, mod.PackedName(id))
		if err := os.WriteFile(file, packed, 0o644); err != nil {
			t.Fatal(err)
		}
		e := mod.Load(file)
		if e.Err != nil {
			t.Fatal(e.Err)
		}
		lines = append(lines, mergedDataLines(t, name+"-packed", base, []mod.Entry{e})...)
		if err := os.Remove(file); err != nil {
			t.Fatal(err)
		}
		return lines
	}
	var got []string
	for _, base := range mergedDataBases {
		for _, id := range mods {
			got = append(got, alone(filepath.Join("testdata", "mods"), id, id, base)...)
			if fixtureCompletions[id] != nil {
				got = append(got, alone(complete, id+"-complete", id, base)...)
			}
		}
		entries, err := mod.Resolve(complete, mods)
		if err != nil {
			t.Fatal(err)
		}
		got = append(got, mergedDataLines(t, "all", base, entries)...)
	}
	if *updateMergedData {
		if err := os.WriteFile(mergedDataFile, []byte(strings.Join(got, "\n")+"\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	f, err := os.Open(mergedDataFile)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	var want []string
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1<<16), 1<<20)
	for sc.Scan() {
		want = append(want, sc.Text())
	}
	if err := sc.Err(); err != nil {
		t.Fatal(err)
	}
	if strings.Join(got, "\n") == strings.Join(want, "\n") {
		return
	}
	for i := 0; i < len(got) || i < len(want); i++ {
		var g, w string
		if i < len(got) {
			g = got[i]
		}
		if i < len(want) {
			w = want[i]
		}
		if g != w {
			t.Errorf("line %d:\n got  %s\n want %s", i+1, g, w)
		}
	}
}
