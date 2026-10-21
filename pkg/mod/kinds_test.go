package mod

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"againrom/pkg/formats/res"
)

// kindFixtures names, for each declared kind, the fixture mod of
// pkg/modrt/testdata/mods that carries a file of it.
var kindFixtures = map[string]string{
	"items":         "heavy-armor",
	"characters":    "archer-girl",
	"screens":       "mission-abandon",
	"companions":    "reniesta-joins-on-talk",
	"spells":        "fire-ball-tune",
	"body sheets":   "war-hammer-body",
	"weapon bodies": "war-hammer-mace",
	"missions":      "reniesta",
	"towns":         "east-town",
	"settings":      "skill-cap",
}

// Every declared kind has a name, a file a packed mod can hold, a key, a
// reader that validates it, the rules of its layer and a fixture mod that
// carries it; no file belongs to two kinds.
func TestEveryDataKindIsDeclaredWhole(t *testing.T) {
	names := map[string]bool{}
	files := map[string]string{}
	for _, k := range DataKinds() {
		if k.Name == "" || names[k.Name] {
			t.Errorf("kind %q: no name, or a name declared twice", k.Name)
		}
		names[k.Name] = true
		if len(k.Files) == 0 {
			t.Errorf("%s: no file", k.Name)
			continue
		}
		for _, seg := range strings.Split(k.Files[0], "/") {
			if len(seg) > res.MaxNameBytes {
				t.Errorf("%s: %s has the name %q of %d bytes; a packed mod holds at most %d", k.Name, k.Files[0], seg, len(seg), res.MaxNameBytes)
			}
		}
		for _, f := range k.Files {
			if other, dup := files[f]; dup {
				t.Errorf("%s: %s is a file of %s too", k.Name, f, other)
			}
			files[f] = k.Name
		}
		if k.Key == "" || !k.Validates() || k.Base == "" || k.Shared == "" {
			t.Errorf("%s: a key, a reader, layer 0 or the rule for two mods is not declared", k.Name)
		}
		switch {
		case k.Class == ConfigurationData && k.Layer != LayerOwn,
			k.Class == CatalogData && k.Layer == LayerOwn,
			k.Class != CatalogData && k.Class != ConfigurationData:
			t.Errorf("%s: class %v with layer %v", k.Name, k.Class, k.Layer)
		case k.Layer == LayerKept && k.Owner == "":
			t.Errorf("%s: a kept kind names no story that migrates it", k.Name)
		case k.Dump && k.Layer != LayerMerged:
			t.Errorf("%s: a dumped kind is not merged", k.Name)
		}
		fixture, ok := kindFixtures[k.Name]
		if !ok {
			t.Errorf("%s: no fixture mod", k.Name)
			continue
		}
		held := false
		for _, f := range k.Files {
			if _, err := os.Stat(filepath.Join("..", "modrt", "testdata", "mods", fixture, filepath.FromSlash(f))); err == nil {
				held = true
			}
		}
		if !held {
			t.Errorf("%s: fixture %s holds none of %v", k.Name, fixture, k.Files)
		}
	}
	for name := range kindFixtures {
		if !names[name] {
			t.Errorf("fixture for %q, which is not a declared kind", name)
		}
	}
}

// docsKindRow is the row docs/MODS.md gives a kind, up to its last column.
func docsKindRow(k DataKind) string {
	file := "`" + k.Files[0] + "`"
	if len(k.Files) > 1 {
		var also []string
		for _, f := range k.Files[1:] {
			also = append(also, "`"+f+"`")
		}
		file += " (a mod folder may also use " + strings.Join(also, ", ") + ")"
	}
	layer := "configuration"
	switch k.Layer {
	case LayerMerged:
		layer = "catalog, merged"
	case LayerKept:
		layer = "catalog, kept until " + k.Owner
	}
	return strings.Join([]string{k.Name, file, k.Key, layer, k.Shared}, " | ") + " |"
}

// docs/MODS.md lists exactly the declared kinds, in their order, each with
// its file, key, layer and rule for two mods.
func TestModsDocListsTheDeclaredKinds(t *testing.T) {
	doc, err := os.ReadFile(filepath.Join("..", "..", "docs", "MODS.md"))
	if err != nil {
		t.Fatal(err)
	}
	text := strings.ReplaceAll(string(doc), "\r\n", "\n")
	_, section, ok := strings.Cut(text, "\n## Data kinds\n")
	if !ok {
		t.Fatal("docs/MODS.md has no Data kinds section")
	}
	section, _, _ = strings.Cut(section, "\n## ")
	var rows []string
	for _, line := range strings.Split(section, "\n") {
		if !strings.HasPrefix(line, "| ") || strings.HasPrefix(line, "| kind |") {
			continue
		}
		cut := strings.LastIndex(strings.TrimSuffix(line, " |"), " | ")
		rows = append(rows, line[:cut]+" |")
	}
	kinds := DataKinds()
	if len(rows) != len(kinds) {
		t.Errorf("docs/MODS.md lists %d kinds; %d are declared", len(rows), len(kinds))
	}
	for i, k := range kinds {
		want := "| " + docsKindRow(k)
		if i >= len(rows) || rows[i] != want {
			got := ""
			if i < len(rows) {
				got = rows[i]
			}
			t.Errorf("row %d:\n got  %s\n want %s", i+1, got, want)
		}
	}
}
