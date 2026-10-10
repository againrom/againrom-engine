package game

import (
	"image"
	"testing"

	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

func TestRetainedOverlayPassesAndCellArmPriority(t *testing.T) {
	// MAGIC-OVERLAYART-051's literal arms, independent of OverlayPicture and
	// the production pass selector. Reversing canonical input cannot reverse
	// Fire/Earth or let Poison through the Freezing arm's early return.
	for _, tc := range []struct {
		name     string
		spells   []uint16
		pictures []int
		passes   []ui.SpellPass
	}{
		{"walls", []uint16{3, 19}, []int{15, 47}, []ui.SpellPass{ui.SpellOverlayA, ui.SpellOverlayA}},
		{"walls reversed", []uint16{19, 3}, []int{15, 47}, []ui.SpellPass{ui.SpellOverlayA, ui.SpellOverlayA}},
		{"clouds", []uint16{7, 8}, []int{23}, []ui.SpellPass{ui.SpellOverlayB}},
		{"clouds reversed", []uint16{8, 7}, []int{23}, []ui.SpellPass{ui.SpellOverlayB}},
		{"all and duplicate", []uint16{8, 19, 7, 3, 3}, []int{15, 47, 23}, []ui.SpellPass{ui.SpellOverlayA, ui.SpellOverlayA, ui.SpellOverlayB}},
		{"poison alone", []uint16{8}, []int{25}, []ui.SpellPass{ui.SpellOverlayB}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			mw := aoWorld(t)
			freeze := *mw.projectiles.Sheet(25)
			mw.projectiles.Sheets[23] = &freeze
			var effects []sim.CellEffect
			for _, spell := range tc.spells {
				effects = append(effects, sim.CellEffect{Spell: spell, Mode: sim.AreaModeCloud, Cells: [][2]int32{{4, 4}}})
			}
			got := mw.areaEffectDraws(effects, nil)
			if len(got) != len(tc.pictures) {
				t.Fatalf("retained draws=%d want=%d", len(got), len(tc.pictures))
			}
			for i, draw := range got {
				if draw.Sheet != mw.projectiles.Sheet(tc.pictures[i]) || draw.Pass != tc.passes[i] || draw.Cell != image.Pt(4, 4) {
					t.Errorf("draw %d: sheet=%p pass=%d cell=%v; want picture=%d pass=%d cell=(4,4)", i, draw.Sheet, draw.Pass, draw.Cell, tc.pictures[i], tc.passes[i])
				}
			}
		})
	}
}

func TestOrdinaryObjectUsingRetainedArtStaysInProjectilePass(t *testing.T) {
	mw := aoWorld(t)
	at := int32(4*ui.ShotScale + 128)
	mw.world.ReleaseCast(sim.CastRecord{Picture: 15, Phases: 1, X: at, Y: at, AimX: at, AimY: at, Segments: 3})
	draws := mw.savedProjectileDraws()
	if len(draws) != 1 || draws[0].Pass != ui.SpellProjectiles {
		t.Fatalf("ordinary record was classified by shared picture 15: %+v", draws)
	}
}

func TestRetainedOverlayDrawsOnlyExplicitOwnedCells(t *testing.T) {
	for _, empty := range [][][2]int32{nil, {}} {
		mw := aoWorld(t)
		// An overwritten Fire area keeps its independent clock and anchor,
		// but no painted cells. Its pulse scan must not recreate a footprint.
		fire := sim.CellEffect{X: 4, Y: 4, Spell: 3, Remaining: 398, Power: 10, Mode: sim.AreaModeCloud, Cells: empty}
		if got := mw.areaEffectDraws([]sim.CellEffect{fire}, nil); len(got) != 0 {
			t.Fatalf("zero-owned area produced %d draws", len(got))
		}
		earth := sim.CellEffect{X: 4, Y: 4, Spell: 19, Remaining: 10, Mode: sim.AreaModeCloud, Cells: [][2]int32{{5, 4}}}
		got := mw.areaEffectDraws([]sim.CellEffect{fire, earth}, nil)
		if len(got) != 1 || got[0].Sheet != mw.projectiles.Sheet(47) || got[0].Cell != image.Pt(5, 4) || got[0].Pass != ui.SpellOverlayA {
			t.Fatalf("owned coverage did not preserve its cell: %+v", got)
		}
	}
}
