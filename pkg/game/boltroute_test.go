package game

import (
	"image"
	"slices"
	"testing"

	"againrom/pkg/sim"
)

// One test per construction route: the call count and the phase sequence
// (MAGIC-281), read from what the drawer is handed.
func TestBoltRoutesDrawTheirPublishedPhaseSequences(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		name  string
		spawn func(mw *mapWorld)
		want  []int
	}{
		{"normal caster", func(mw *mapWorld) {
			mw.observeCasts([]sim.CastEvent{{Caster: 1, Target: 2, Spell: spLightning, FromX: 2, FromY: 2, ToX: 9, ToY: 6}})
		}, []int{4, 3, 2, 1, 0, 1, 2, 1, 0, 1, 2, 3, 4}},
		{"direct 0x8b", func(mw *mapWorld) {
			mw.observeScriptCasts([]sim.ScriptCastEvent{{Spell: spLightning, FromX: 2, FromY: 2, ToX: 9, ToY: 6}})
		}, []int{0, 4, 3, 2, 1}},
		{"source-cell 0x8c", func(mw *mapWorld) {
			mw.observeScriptCasts([]sim.ScriptCastEvent{{Spell: spPrismatic, FromX: 2, FromY: 2, ToX: 9, ToY: 6,
				Victims: []sim.CellPoint{{X: 9, Y: 6}}}})
		}, []int{0, 4, 3, 2, 1, 0, 1, 2, 1, 0, 1, 2, 3}},
	} {
		mw := spWorld(t)
		c.spawn(mw)
		var got []int
		for len(mw.bolts) > 0 {
			draws := mw.boltDraws(nil)
			if len(draws) == 0 {
				t.Fatalf("%s: call %d drew nothing", c.name, len(got)+1)
			}
			got = append(got, draws[0].Frame)
			mw.advanceBolts()
		}
		if !slices.Equal(got, c.want) {
			t.Errorf("%s: frames %v, want %v", c.name, got, c.want)
		}
	}
}

// TestPrismaticFrameAddsFiveTimesTheVictimIndex: picture 36 draws
// phase+5*(victimIndex%7); victim 9 carries tag 2.
func TestPrismaticFrameAddsFiveTimesTheVictimIndex(t *testing.T) {
	t.Parallel()
	mw := spWorld(t)
	victims := make([]sim.CellPoint, 10)
	for k := range victims {
		victims[k] = sim.CellPoint{X: int32(4 + k%5), Y: int32(6 + k/5)}
	}
	mw.observeCasts([]sim.CastEvent{{Caster: 1, Target: 2, Spell: spPrismatic, FromX: 1, FromY: 1, ToX: 4, ToY: 6, Victims: victims}})
	b := mw.bolts[9]
	if b.tag != 2 {
		t.Fatalf("victim 9 carries tag %d, want 9%%7=2", b.tag)
	}
	for range 4 {
		mw.advanceBolts() // call 5: phase 0
	}
	_, frame, points := mw.pathFigure(mw.bolts[9])
	if frame != 0+5*2 || len(points) == 0 {
		t.Fatalf("victim 9 at call 5 draws frame %d, want 10", frame)
	}
	mw.advanceBolts() // call 6: phase 1
	if _, frame, _ = mw.pathFigure(mw.bolts[9]); frame != 1+5*2 {
		t.Fatalf("victim 9 at call 6 draws frame %d, want 11", frame)
	}
}

// TestABoltStampsEachStoredPointOnceInListOrder: the drawer hands one stamp
// per stored point, in order, centred by the immediate 8 (MAGIC-280).
func TestABoltStampsEachStoredPointOnceInListOrder(t *testing.T) {
	t.Parallel()
	mw := spWorld(t)
	mw.observeCasts([]sim.CastEvent{{Caster: 1, Target: 2, Spell: spLightning, FromX: 2, FromY: 2, ToX: 9, ToY: 6}})
	b := mw.bolts[0]
	ax, ay := mw.boltDisplayPoint(castOrigin(b.from, b.launch), b.from)
	bx, by := mw.boltDisplayPoint(b.to.Mul(256), b.to)
	want := boltFigure(ax, ay, bx, by, 34, (&boltRNG{state: boltSeed(b)}).next)
	draws := mw.boltDraws(nil)
	if len(draws) != len(want) {
		t.Fatalf("%d stamps for %d stored points", len(draws), len(want))
	}
	for i, d := range draws {
		if d.Pos != image.Pt(int(want[i].X), int(want[i].Y)) || !d.Display {
			t.Fatalf("stamp %d at %v, want stored point %v", i, d.Pos, want[i])
		}
	}
}

// TestALoadedBoltContinuesItsCounterAndPhase: the visual snapshot keeps age,
// life and route, so a restored object draws the uninterrupted sequence.
func TestALoadedBoltContinuesItsCounterAndPhase(t *testing.T) {
	t.Parallel()
	run := func(cut int) []int {
		mw := spWorld(t)
		mw.observeScriptCasts([]sim.ScriptCastEvent{{Spell: spLightning, FromX: 2, FromY: 2, ToX: 9, ToY: 6}})
		var frames []int
		for i := 0; len(mw.bolts) > 0; i++ {
			if i == cut {
				var r SnapshotResidue
				mw.actionVisuals(&r)
				fresh := spWorld(t)
				fresh.restoreActionVisuals(r.SpellBolts, r.HealBursts)
				mw = fresh
			}
			_, frame, _ := mw.pathFigure(mw.bolts[0])
			frames = append(frames, frame)
			mw.advanceBolts()
		}
		return frames
	}
	whole := run(-1)
	for cut := 1; cut < 5; cut++ {
		if got := run(cut); !slices.Equal(got, whole) {
			t.Errorf("restored at call %d: frames %v, want %v", cut+1, got, whole)
		}
	}
}
