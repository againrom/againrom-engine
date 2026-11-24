package mod

import (
	"os"
	"path/filepath"
	"testing"
)

// Every fixture mod of the runtime tests loads, and writing each of its
// manifest's own values back leaves the manifest byte for byte.
func TestFixtureManifestsLoadAndRoundTrip(t *testing.T) {
	dirs, err := filepath.Glob(filepath.Join("..", "modrt", "testdata", "mods", "*"))
	if err != nil || len(dirs) == 0 {
		t.Fatalf("no fixture mods: %v", err)
	}
	for _, dir := range dirs {
		if e := Load(dir); e.Err != nil {
			t.Fatalf("%s: %v", dir, e.Err)
		}
		src, err := os.ReadFile(filepath.Join(dir, ManifestName))
		if err != nil {
			t.Fatal(err)
		}
		m, err := ParseManifest(src)
		if err != nil {
			t.Fatal(err)
		}
		tables, err := parseTOML(src)
		if err != nil {
			t.Fatal(err)
		}
		values := map[string]any{}
		for _, p := range tables[0].pairs {
			row, _ := ManifestKeyNamed(p.key)
			values[p.key] = row.Get(m)
		}
		out, err := WriteManifest(src, values)
		if err != nil || string(out) != string(src) {
			t.Fatalf("%s: round trip changed the manifest: %v", dir, err)
		}
	}
	t.Logf("%d fixture mods", len(dirs))
}
