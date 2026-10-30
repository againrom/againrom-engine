package game

import (
	"crypto/sha256"
	"fmt"
	"testing"

	"againrom/pkg/sim"
)

func TestSecondGameCurseAndPoisonCloudDrawTheirHeldMarks(t *testing.T) {
	for _, tc := range []struct {
		name    string
		row     uint16
		kind    sim.EffectKind
		picture int
	}{
		{"Poison Cloud", 6, sim.EffectHealth, 20},
		{"Curse", 28, sim.EffectCurse, 64},
	} {
		t.Run(tc.name, func(t *testing.T) {
			mw := secondHeldMarkWorld(t, tc.row, tc.kind)
			mw.projectiles = emSheets(20, 24, 62, 64)
			for update, c := range []uint16{0xffff, 0xfffe} {
				mw.advanceEffectMarks()
				if len(mw.markElements) != 1 || mw.markElements[0].Count != c {
					t.Fatalf("update %d: elements %+v, want count %#x", update, mw.markElements, c)
				}
				for _, scale := range []int{1, 2} {
					draws := mw.markDraws(2, scale)
					wantCount := 1
					if tc.row == 28 {
						wantCount = 20
					}
					if len(draws) != wantCount {
						t.Fatalf("count %#x scale %d: %d records, want %d", c, scale, len(draws), wantCount)
					}
					for i, d := range draws {
						dx, depth, phase := 0, 0, int(c%6)
						if tc.row == 28 {
							quadrants := [2][5][2]int{
								{{20, 0}, {19, 6}, {16, 11}, {11, 16}, {6, 19}},
								{{6, 19}, {0, 20}, {-6, 19}, {-11, 16}, {-16, 11}},
							}
							u, v := quadrants[update][i/4][0], quadrants[update][i/4][1]
							p := [4][2]int{{u, v}, {v, -u}, {-u, -v}, {-v, u}}[i%4]
							dx, depth, phase = p[0], p[1], 4-i/4
						}
						m := d.Mark
						if m.DX != dx || m.DY != -32*scale || m.Depth != depth || m.Record != tc.picture || m.Phase != phase || d.Sheet != mw.projectiles.Sheet(tc.picture) {
							t.Errorf("count %#x scale %d record %d = %+v; want (%d,%d,%d,%d,%d) on its own sheet (R2-ENGINE-698)",
								c, scale, i, m, dx, -32*scale, depth, tc.picture, phase)
						}
					}
				}
			}
			for i := 0; i < 20000 && len(mw.world.ActiveEffects()) > 0; i++ {
				sim.Step(mw.world, nil)
			}
			mw.advanceEffectMarks()
			if len(mw.world.ActiveEffects()) != 0 || len(mw.markElements) != 0 || len(mw.markDraws(2, 1)) != 0 {
				t.Fatal("the held mark outlived its effect")
			}
		})
	}
}

func TestSecondGameExistingHeldMarksDrawAsOnTheBase(t *testing.T) {
	rows := []struct {
		row  uint16
		kind sim.EffectKind
	}{
		{4, sim.EffectProtectionFire}, {8, sim.EffectProtectionWater},
		{13, sim.EffectProtectionAir}, {19, sim.EffectProtectionEarth},
		{20, sim.EffectBless}, {27, sim.EffectAbsorption},
	}
	counts := []uint16{0xffff, 0xfffe}
	for c := uint16(0); c < 90; c++ {
		counts = append(counts, c)
	}
	h := sha256.New()
	for _, tc := range rows {
		mw := secondHeldMarkWorld(t, tc.row, tc.kind)
		mw.advanceEffectMarks()
		for _, c := range counts {
			mw.markElements[0].Count = c
			for scale := 1; scale <= 3; scale++ {
				fmt.Fprintf(h, "%d:%d:%d:%+v;", tc.row, c, scale, marksOf(mw.markDraws(2, scale)))
				for _, d := range mw.markDraws(2, scale) {
					if d.Sheet != mw.projectiles.Sheet(d.Mark.Record) {
						t.Fatal("record and sheet disagree")
					}
				}
			}
		}
	}
	got := fmt.Sprintf("%x", h.Sum(nil))
	const want = "d13d75b70e83bf9baa58a193e468732769439c3287a211b321f12db303228404"
	if got != want {
		t.Fatalf("six existing held marks digest %s, want base 76cd589e's %s", got, want)
	}
}
