package game

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"againrom/pkg/base"
	"againrom/pkg/formats/alm"
	"againrom/pkg/render/terrain"
)

// installedMapsOfEitherGame are the installed maps' bytes by name: the first
// game's catalogue, or the second game's loose map files at the install root.
func installedMapsOfEitherGame(t *testing.T) (game base.Game, keys []string, maps [][]byte) {
	t.Helper()
	root := os.Getenv("AGAINROM_ASSETS")
	if root == "" {
		t.Skip("no AGAINROM_ASSETS: needs lawful install")
	}
	m, err := base.Detect(root, nil)
	if err != nil {
		t.Fatal(err)
	}
	if m.Profile.HasFirstData() {
		x := installedInspector(t)
		keys, maps = installedMapBytes(t, x)
		return x.archives.Game(), keys, maps
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if e.IsDir() || !strings.EqualFold(filepath.Ext(e.Name()), ".alm") {
			continue
		}
		b, err := os.ReadFile(filepath.Join(root, e.Name()))
		if err != nil {
			t.Fatal(err)
		}
		keys, maps = append(keys, e.Name()), append(maps, b)
	}
	return base.GameROM2, keys, maps
}

// Every cell of every installed map names a tile group its own metadata mask
// enables, so admitting slots by the mask leaves the installed maps' frames
// unchanged.
func TestReleaseInstalledMapMasksEnableTheGroupsTheyDraw(t *testing.T) {
	game, keys, maps := installedMapsOfEitherGame(t)
	for i, data := range maps {
		m, err := MapOpener(game)(data)
		if err != nil {
			t.Fatal(keys[i], err)
		}
		for c, w := range m.Tiles {
			g := alm.TileGroup(w)
			if terrain.IsWaterGroup(g) {
				for wg := 8; wg < 12; wg++ {
					if !terrain.SlotAdmitted(m.Meta.Bitmask, wg*4) {
						t.Errorf("%s: cell %d draws water but mask %#x excludes group %d", keys[i], c, m.Meta.Bitmask, wg)
						break
					}
				}
				continue
			}
			if !terrain.SlotAdmitted(m.Meta.Bitmask, g*4) {
				t.Errorf("%s: cell %d draws group %d but mask %#x excludes it", keys[i], c, g, m.Meta.Bitmask)
				break
			}
		}
	}
	t.Logf("%d installed maps checked (%s)", len(maps), string(game))
}
