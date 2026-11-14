package game

import (
	"slices"
	"testing"

	"againrom/pkg/base"
	"againrom/pkg/mod"
)

// The test-play package and every new package apply to the family of the
// game they were built on, read back from the written manifest.
func TestPackageManifestAppliesToTheGameItWasBuiltOn(t *testing.T) {
	for _, g := range []base.Game{base.GameROM1, base.GameROM2} {
		files := missionPackageFiles("Map", "Map", mod.Mission{Mod: "packagetest", Map: "scenario/20.alm"}, g, "en", nil)
		m, err := mod.ParseManifest(files[mod.ManifestName])
		if err != nil {
			t.Fatal(err)
		}
		if want := []string{base.ModFamilyOf(g)}; !slices.Equal(m.AppliesTo, want) {
			t.Fatalf("a package built on %q applies to %q, want %q", g, m.AppliesTo, want)
		}
	}
}
