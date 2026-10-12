package game

import (
	"os"
	"testing"

	"againrom/pkg/base"
	"againrom/pkg/mapload"
)

// TestReleaseInstalledTerrainCostTable reads the first game's installed
// map.reg into the definitions table. On the shipped roots it holds the
// compiled shipped values, so a shipped mission's cost plane is unchanged.
func TestReleaseInstalledTerrainCostTable(t *testing.T) {
	root := os.Getenv("AGAINROM_ASSETS")
	if root == "" {
		t.Skip("no AGAINROM_ASSETS: the installed cost table needs a lawful install")
	}
	archives, err := OpenArchives(root)
	if err != nil {
		t.Fatal(err)
	}
	defs, err := LoadDefinitionsFor(archives.Containers, archives.Game())
	if err != nil {
		t.Fatal(err)
	}
	if archives.Game() != base.GameROM1 {
		if defs.Table.TerrainCosts != nil {
			t.Fatalf("second game carries a first-game Terrain cost table %v", *defs.Table.TerrainCosts)
		}
		return
	}
	if defs.Table.TerrainCosts == nil {
		t.Fatal("first-game install read no Terrain cost table")
	}
	if got, want := *defs.Table.TerrainCosts, (*mapload.Table)(nil).TerrainCostTable(); got != want {
		t.Fatalf("installed Terrain cost table %v, compiled shipped values %v", got, want)
	}
}
