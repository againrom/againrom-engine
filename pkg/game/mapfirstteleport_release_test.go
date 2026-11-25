package game

import (
	"reflect"
	"strings"
	"testing"

	"againrom/internal/firstgame"
	"againrom/pkg/formats/alm"
)

// crossTeleports are the type-9 cell records of spell 26 the English
// install's loose Cross.ALM holds: power 100, source at the cell.
var crossTeleports = []alm.CellBinding{
	{X: 106, Y: 13, Spell: firstCellMove, Power: 100, SourceX: 106, SourceY: 13, LastX: 168, LastY: 24},
	{X: 245, Y: 66, Spell: firstCellMove, Power: 100, SourceX: 245, SourceY: 66, LastX: 231, LastY: 57},
	{X: 238, Y: 164, Spell: firstCellMove, Power: 100, SourceX: 238, SourceY: 164, LastX: 239, LastY: 241},
}

// The installed first-game maps' cell records of spell 26: no campaign map
// in the archives holds one (TERR-229 for its five maps), and the loose maps
// hold only Cross.ALM's three, where the cell move plays.
func TestReleaseInstalledFirstMapTeleportTrapCensus(t *testing.T) {
	firstgame.SkipOnSecondGame(t)
	x := installedInspector(t)
	keys, maps := installedMapBytes(t, x)
	cells, loose := 0, map[string][]alm.CellBinding{}
	for n, key := range keys {
		m, err := alm.Open(maps[n])
		if err != nil {
			t.Fatalf("%s: %v", key, err)
		}
		bindings, err := m.CellBindings()
		if err != nil {
			t.Fatalf("%s: %v", key, err)
		}
		for _, b := range bindings {
			cells++
			if b.Spell != firstCellMove {
				continue
			}
			if !strings.HasPrefix(key, "loose/") {
				t.Errorf("%s: cell %d,%d holds spell 26: %+v", key, b.X, b.Y, b)
			}
			loose[key] = append(loose[key], b)
		}
	}
	if len(keys) == 0 {
		t.Fatal("no installed map")
	}
	for key, got := range loose {
		if key != "loose/Cross.ALM" || !reflect.DeepEqual(got, crossTeleports) {
			t.Errorf("%s holds the spell-26 records %+v", key, got)
		}
	}
	t.Logf("%d maps, %d cell records, spell 26 on %v", len(keys), cells, loose)
}
