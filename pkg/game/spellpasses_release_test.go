package game

import (
	"testing"

	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

func TestReleaseRetainedOverlayPassesUseInstalledArt(t *testing.T) {
	f := releaseFront(t)
	mw := sbWorld(t)
	mw.projectiles = f.Projectiles
	for _, spells := range [][]uint16{{8, 19, 7, 3, 3}, {3, 7, 19, 8}} {
		var effects []sim.CellEffect
		for _, spell := range spells {
			effects = append(effects, sim.CellEffect{Spell: spell, Mode: sim.AreaModeCloud, Cells: [][2]int32{{4, 4}}})
		}
		draws := mw.areaEffectDraws(effects, nil)
		if len(draws) != 3 {
			t.Fatalf("retained cell projects %d sprites, want Fire/Earth/Freezing", len(draws))
		}
		for i, want := range []struct {
			picture int
			pass    ui.SpellPass
		}{{15, ui.SpellOverlayA}, {47, ui.SpellOverlayA}, {23, ui.SpellOverlayB}} {
			if draws[i].Sheet != f.Projectiles.Sheet(want.picture) || draws[i].Pass != want.pass || draws[i].Sheet.Frame(draws[i].Frame) == nil {
				t.Fatalf("draw %d did not carry installed picture %d in pass %d", i, want.picture, want.pass)
			}
		}
	}
	t.Log("production projection carries installed pictures 15/47 in A and 23 in B; duplicate Fire and masked Poison draw no extra sprite")
}
