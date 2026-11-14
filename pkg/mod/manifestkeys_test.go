package mod

import (
	"slices"
	"strings"
	"testing"

	"againrom/pkg/base"
)

// A row added to the key table reads, writes and is a known key.
func TestManifestKeysAreOneTable(t *testing.T) {
	saved := ManifestKeys
	t.Cleanup(func() { ManifestKeys = saved })
	ManifestKeys = append(slices.Clone(saved), ManifestKey{Name: "launch", Value: ManifestString,
		Get: func(m Manifest) any { return m.Entry }, Set: func(m *Manifest, v any) { m.Entry = v.(string) }})
	src := []byte("id = \"k\"\ntitle = \"K\"\nversion = \"1\"\nlaunch = \"start.star\"\n")
	m, err := ParseManifest(src, true)
	if err != nil || m.Entry != "start.star" {
		t.Fatalf("parsed %+v, %v", m, err)
	}
	out, err := WriteManifest(src, map[string]any{"launch": "go.star"})
	if err != nil || !strings.Contains(string(out), `launch = "go.star"`) {
		t.Fatalf("wrote\n%s\n%v", out, err)
	}
	if _, err := ParseManifest([]byte("id = \"k\"\ntitle = \"K\"\nversion = \"1\"\nhome = \"x\"\n")); err == nil || !strings.Contains(err.Error(), "load-after, launch") {
		t.Fatalf("an unknown key: %v", err)
	}
}

// Each base table entry admits exactly the installs it names.
func TestEveryBaseOfTheTableIsAdmitted(t *testing.T) {
	games := map[base.Game]bool{}
	for _, b := range base.ModBases {
		games[b.Game] = true
		for _, p := range base.Profiles {
			if p.ID == base.ROM1Demo {
				continue
			}
			named := b.Name == base.AllBases || b.Name == p.ID || slices.Contains(p.Edition().ModFamily, b.Name)
			if got := Applies([]string{b.Name}, p.ID); got != named {
				t.Errorf("applies-to %q on %s: %v, want %v", b.Name, p.ID, got, named)
			}
			if b.Game != "" && named && p.GameOf() != b.Game {
				t.Errorf("%q of %s names %s", b.Name, b.Game, p.ID)
			}
		}
	}
	if !games[base.GameROM1] || !games[base.GameROM2] || !Applies([]string{base.ModFamilyOf(base.GameROM2)}, base.ROM2RU) {
		t.Fatal("the base table does not hold both games")
	}
}
