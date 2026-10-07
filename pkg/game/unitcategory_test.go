package game

import (
	"testing"

	"againrom/pkg/render/terrain"
	"againrom/pkg/sim"
)

func TestEntityDrawCategorySurvivesComposedAndCorpseArt(t *testing.T) {
	for _, z := range []int{0, 96} {
		for _, stage := range []sim.DecayStage{0, 1, 2, 4} {
			e := deathUnit(1, 1, 2, 2)
			e.Decay = stage
			if stage > 0 {
				e.HP = -1
			}
			mw := deathWorld(t, nil, e)
			mw.units.Classes[1].Z = z
			// Opposite classification on both substituted classes makes either
			// accidental lookup discriminate from the placed class.
			body := deathLiveArt()
			body.Z = 96 - z
			body.Corpse = deathCorpseArt()
			body.Corpse.Z = 96 - z
			mw.art = map[sim.EntityID]*terrain.UnitClass{1: body}
			draw := deathDraw(t, mw, 1)
			want := terrain.UnitOrdinary
			if z != 0 {
				want = terrain.UnitAir
			} else if stage >= 2 {
				want = terrain.UnitAlternate
			}
			if draw.DrawCategory != want {
				t.Fatalf("placed Z=%d stage=%d substituted Z=%d: category=%d want=%d", z, stage, body.Z, draw.DrawCategory, want)
			}
			if stage == 0 && draw.Art != body || stage > 0 && draw.Art != body.Corpse {
				t.Fatalf("stage=%d did not traverse the intended art substitution", stage)
			}
		}
	}
}
