package game

import (
	"image"
	"image/color"
	"testing"

	"againrom/pkg/render/terrain"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// How an area effect is drawn.

// aoSheets adds the overlay art the four overlay spells name to the bundle the
// spell-object tests already use: pictures 15, 23, 25 and 47, which are
// `BurstPicture` of spells 3, 7, 8 and 19 (`MAGIC-OVERLAYART-051`).
//
// Picture 15 holds ELEVEN frames, the shipped `firewall` sheet's own count, so
// the frame law's five-of-eleven restriction is measurable here.
func aoSheets() *terrain.EffectSet {
	frames := func(n int) []*terrain.EffectFrame {
		out := make([]*terrain.EffectFrame, n)
		for i := range out {
			out[i] = &terrain.EffectFrame{Width: 8, Height: 8, Pixels: make([]color.RGBA, 64)}
		}
		return out
	}
	set := sbSheets()
	set.Sheets[15] = &terrain.EffectSheet{Frames: frames(11), Phases: 11, RotationPhases: 1, CenterX: 32, CenterY: 32}
	set.Sheets[25] = &terrain.EffectSheet{Frames: frames(6), Phases: 6, RotationPhases: 1, CenterX: 32, CenterY: 32}
	set.Sheets[47] = &terrain.EffectSheet{Frames: frames(4), Phases: 4, RotationPhases: 1, CenterX: 32, CenterY: 32}
	// Fire Sacrifice's burst picture, 2*4+9, and Meteor Storm's, 2*21+9.
	set.Sheets[17] = &terrain.EffectSheet{Frames: frames(8), Phases: 8, RotationPhases: 1, CenterX: 16, CenterY: 16}
	set.Sheets[51] = &terrain.EffectSheet{Frames: frames(16), Phases: 16, RotationPhases: 1, CenterX: 16, CenterY: 16}
	return set
}

func aoWorld(t *testing.T) *mapWorld {
	t.Helper()
	mw := sbWorld(t)
	mw.projectiles = aoSheets()
	return mw
}

func TestABurningCellNeverShowsItsBirthOrFadeFrames(t *testing.T) {
	t.Parallel()

	seen := map[int]bool{}
	for counter := range 400 {
		for x := range 7 {
			for y := range 7 {
				frame, ok := terrain.OverlayFrame(3, 11, counter, x, y)
				if !ok {
					t.Fatalf("the wall-of-fire arm refused a phase at counter %d", counter)
				}
				if frame < 3 || frame > 7 {
					t.Fatalf("a burning cell drew frame %d at counter %d, cell (%d,%d); "+
						"only frames 3..7 of the eleven-frame sheet are ever drawn", frame, counter, x, y)
				}
				seen[frame] = true
			}
		}
	}
	if len(seen) != 5 {
		t.Errorf("a burning cell reached %d of its five frames (%v), want all five", len(seen), seen)
	}
}

// TestTheOtherThreeOverlaysTakeTheirOwnPhaseCount is the same claim's other
// half: the three arms that are not Wall of Fire take the record's own Phases,
// with no five and no bias.
func TestTheOtherThreeOverlaysTakeTheirOwnPhaseCount(t *testing.T) {
	t.Parallel()

	for _, spell := range []int{7, 8, 19} {
		seen := map[int]bool{}
		for counter := range 200 {
			for x := range 5 {
				frame, ok := terrain.OverlayFrame(spell, 4, counter, x, 3)
				if !ok || frame < 0 || frame > 3 {
					t.Fatalf("spell %d drew frame %d (ok=%v) at counter %d, want 0..3 of its own four",
						spell, frame, ok, counter)
				}
				seen[frame] = true
			}
		}
		if len(seen) != 4 {
			t.Errorf("spell %d reached %d of its four phases, want all four", spell, len(seen))
		}
	}
	if _, ok := terrain.OverlayFrame(7, 0, 5, 1, 1); ok {
		t.Errorf("a sheet stating no positive phase count answered a frame")
	}
}

// TestNeighbouringBurningCellsDoNotShareOnePhase is what the spatial term is
// for: a wall whose cells all showed one frame would blink in unison rather than
// look like fire.
func TestNeighbouringBurningCellsDoNotShareOnePhase(t *testing.T) {
	t.Parallel()

	first, _ := terrain.OverlayFrame(3, 11, 6, 5, 5)
	differs := false
	for _, c := range [][2]int{{5, 6}, {6, 5}, {6, 6}, {5, 7}, {7, 5}} {
		if f, _ := terrain.OverlayFrame(3, 11, 6, c[0], c[1]); f != first {
			differs = true
		}
	}
	if !differs {
		t.Errorf("every cell of a wall drew frame %d on one tick; the phase does not vary by cell", first)
	}
}

// TestOnlyACloudDrawsARetainedOverlay is the three tick modes at the draw
// (`MAGIC-AREADRAW-049`): a cloud's cells are the retained overlay, a staged
// effect registers no map layer and its record must draw nothing here, and a
// cloud spell with no overlay art draws nothing either
// (`MAGIC-OVERLAYART-051`).
func TestOnlyACloudDrawsARetainedOverlay(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name  string
		spell uint16
		mode  uint8
		want  int
	}{
		{"wall of fire", 3, sim.AreaModeCloud, 2},
		{"poison cloud", 8, sim.AreaModeCloud, 2},
		{"wall of earth", 19, sim.AreaModeCloud, 2},
		// A cloud row with no overlay art: four of the six have a sprite.
		{"a cloud with no overlay art", 12, sim.AreaModeCloud, 0},
		// Fire Sacrifice is staged. Its cells are drawn as transient bursts
		// from the tick's paint report, never from the retained record.
		{"fire sacrifice", 4, sim.AreaModeRing, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			mw := aoWorld(t)
			got := mw.areaEffectDraws([]sim.CellEffect{{
				Spell: tc.spell, Mode: tc.mode,
				Cells: [][2]int32{{4, 4}, {5, 4}},
			}}, nil)
			if len(got) != tc.want {
				t.Fatalf("spell %d in mode %d drew %d overlay sprites, want %d",
					tc.spell, tc.mode, len(got), tc.want)
			}
		})
	}
}

// TestARetainedOverlayIgnoresTheRecordsOwnAge is `MAGIC-OVERLAY-050` at the
// draw: what a cloud puts on the client carries no phase per cell, so nothing
// about the drawn frame may come from how long the record has stood. The record
// is handed two different ages and answers the same frame.
func TestARetainedOverlayIgnoresTheRecordsOwnAge(t *testing.T) {
	t.Parallel()

	mw := aoWorld(t)
	cells := [][2]int32{{4, 4}}
	young := mw.areaEffectDraws([]sim.CellEffect{{Spell: 3, Mode: sim.AreaModeCloud, Phase: 0, Cells: cells}}, nil)
	old := mw.areaEffectDraws([]sim.CellEffect{{Spell: 3, Mode: sim.AreaModeCloud, Phase: 200, Cells: cells}}, nil)

	if len(young) != 1 || len(old) != 1 {
		t.Fatalf("the two reads drew %d and %d sprites, want one each", len(young), len(old))
	}
	if young[0].Frame != old[0].Frame {
		t.Errorf("a cell that has burned 200 ticks drew frame %d against a fresh cell's %d; "+
			"a cloud's cell carries no phase and cannot remember when it started burning",
			old[0].Frame, young[0].Frame)
	}
}

// TestOverlappingWallsKeepIndependentRecordsButDrawOneCellOverlay resolves the
// presentation question created by DIV-093. Unlimited stacking is a simulation
// rule: both canonical records remain live for damage and independent expiry.
// MAGIC-OVERLAY-050's client form is one bit per spell per cell, however, and
// ANIM-WALLFIREFRAME-033 derives the frame from the global counter and cell, not
// either record. The shared cell therefore draws once while each unshared cell
// still draws.
func TestOverlappingWallsKeepIndependentRecordsButDrawOneCellOverlay(t *testing.T) {
	t.Parallel()

	mw := aoWorld(t)
	effects := []sim.CellEffect{
		{Spell: 3, Mode: sim.AreaModeCloud, Phase: 4, Cells: [][2]int32{{4, 4}, {5, 4}}},
		{Spell: 3, Mode: sim.AreaModeCloud, Phase: 17, Cells: [][2]int32{{5, 4}, {6, 4}}},
	}
	got := mw.areaEffectDraws(effects, nil)
	if len(got) != 3 {
		t.Fatalf("two walls with one shared cell drew %d overlays, want 3 distinct cells", len(got))
	}
	want := map[image.Point]int{image.Pt(4, 4): 1, image.Pt(5, 4): 1, image.Pt(6, 4): 1}
	for _, draw := range got {
		want[draw.Cell]--
	}
	for cell, remaining := range want {
		if remaining != 0 {
			t.Errorf("cell %v draw balance = %d, want 0", cell, remaining)
		}
	}
}

// TestAHealStarStandsOnItsCellsCentre is the review's own counterexample, at the
// producer rather than at the seam.
//
// healSpriteDraws used to add ShotScale/2 to the base on X BY HAND,
// compensating for a consumer that centred on neither axis. When
// EffectGroundPoint began supplying the term on both axes, X received it
// twice and every heal star moved half a cell east.
//
// THE ASSERTION RUNS THE REAL PRODUCER'S OUTPUT THROUGH THE REAL CONSUMER, on
// BOTH axes, against terrain.StaticAnchor. The seam-side test in pkg/ui feeds
// EffectGroundPoint a synthetic Pos and so cannot see a producer's pre-bias at
// all; the existing heal test in this package checks only that the stars rise
// and that their frames vary. Neither could have caught this.
func TestAHealStarStandsOnItsCellsCentre(t *testing.T) {
	t.Parallel()

	mw := aoWorld(t)
	const col, row = 4, 4
	mw.healBursts = []healBurst{{at: image.Pt(col, row), picture: healingPicture, owner: 1, seed: 0}}

	stars := mw.healSpriteDraws()
	if len(stars) != 5 {
		t.Fatalf("seed-zero Heal cohort produced %d stars, want the fixed RNG witness's 5", len(stars))
	}
	sheet := mw.projectiles.Sheet(healingPicture)
	// StaticAnchor with the canvas and the frame zeroed leaves the ground point
	// less the centring halves, which is exactly what EffectGroundPoint answers.
	wantX, wantY, _, _ := terrain.StaticAnchor(col, row, 0, 0, sheet.CenterX, sheet.CenterY, 0, 0, 0, 0)

	cohort := showerCohorts(mw.healBursts[0], 0)[0]
	for i, s := range stars {
		// A fresh Heal has no vertical travel. Remove its exact ellipse offset
		// (depth projects upward on screen) to recover the producer's cell base.
		off := image.Pt(cohort[i].dx, -cohort[i].depth).Mul(boltUnitsPerPixel)
		base := s.Pos.Sub(off)
		gotX, gotY := ui.EffectGroundPoint(base, sheet)
		if gotX != wantX || gotY != wantY {
			t.Fatalf("heal star %d on cell (%d,%d) is blitted from base world pixel (%d,%d), "+
				"want the static layer's own (%d,%d) on both axes; a half cell supplied twice "+
				"on one axis is the review's counterexample",
				i, col, row, gotX, gotY, wantX, wantY)
		}
	}
}

// TestAStagedStageSpawnsOneBurstPerAcceptedCell is `MAGIC-AREADRAW-049`'s staged
// arm: one message per accepted cell, each building one transient object of 16
// ticks, 18 for Acid Stream's picture.
func TestAStagedStageSpawnsOneBurstPerAcceptedCell(t *testing.T) {
	t.Parallel()

	mw := aoWorld(t)
	mw.observeAreaPaints([]sim.AreaPaint{{
		Spell: 4, Owner: 1,
		Cells: []sim.CellPoint{{X: 3, Y: 4}, {X: 4, Y: 3}, {X: 5, Y: 4}},
	}})
	if got := len(mw.bolts); got != 3 {
		t.Fatalf("a three-cell stage spawned %d objects, want one per cell", got)
	}
	for _, b := range mw.bolts {
		if b.from != b.to {
			t.Errorf("a staged cell's object runs %v to %v, want both ends on its own cell", b.from, b.to)
		}
		if b.life != 16 {
			t.Errorf("a staged cell's object lives %d ticks, want the decoded 16", b.life)
		}
		if b.picture != 17 {
			t.Errorf("a Fire Sacrifice cell drew picture %d, want 2*4+9", b.picture)
		}
	}
	// A spell whose burst picture names no sheet spawns nothing.
	mw.bolts = nil
	mw.observeAreaPaints([]sim.AreaPaint{{Spell: 6, Cells: []sim.CellPoint{{X: 1, Y: 1}}}})
	if got := len(mw.bolts); got != 0 {
		t.Errorf("a spell with no burst sheet spawned %d objects, want none", got)
	}
}

// TestAnEarlierStageIsStillDrawnWhenALaterOneLights is the owner's Fire
// Sacrifice report (the explosion spreads outward from the caster's own cell)
// and his Meteor Storm report (rocks land at random points at a visible rate),
// which are one property: a stage runs every three ticks and an object lives
// sixteen, so the stages overlap and the effect accumulates.
//
// Drawing the retained record's own cell set instead replaced each stage with
// the next after three ticks, so exactly one stage was ever on screen.
func TestAnEarlierStageIsStillDrawnWhenALaterOneLights(t *testing.T) {
	t.Parallel()

	mw := aoWorld(t)
	mw.observeAreaPaints([]sim.AreaPaint{{Spell: 4, Cells: []sim.CellPoint{{X: 3, Y: 3}}}})
	for range 3 {
		mw.advanceBolts()
	}
	mw.observeAreaPaints([]sim.AreaPaint{{Spell: 4, Cells: []sim.CellPoint{{X: 5, Y: 5}}}})

	draws := mw.boltDraws(nil)
	if len(draws) != 2 {
		t.Fatalf("three ticks after the first stage the map holds %d objects, want both stages", len(draws))
	}
	at := map[image.Point]bool{}
	for _, d := range draws {
		at[d.Pos] = true
	}
	for _, want := range []image.Point{
		{X: 3 * ui.ShotScale, Y: 3 * ui.ShotScale},
		{X: 5 * ui.ShotScale, Y: 5 * ui.ShotScale},
	} {
		if !at[want] {
			t.Errorf("no object stands at %v; the two stages do not overlap", want)
		}
	}
}
