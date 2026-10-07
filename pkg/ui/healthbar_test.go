package ui

// The bar where a frame meets it: which units carry one, where its passes
// stand in the draw order, and that a bar takes exactly the relief offset the
// mark on its own cell takes.
//
// The GEOMETRY is the render tier's and is measured there, and the composed
// pixels are statusbar_test.go's. What is measured here is the selection — not
// dead and a positive maximum, read off the seam's own answer — and the
// placement, which is the one thing a copy of the transform could get wrong.

import (
	"image"
	"image/color"
	"slices"
	"testing"

	"againrom/pkg/render/terrain"
)

// statusBarScreenRects places the frame's bars of one kind as statusBarPasses
// places their runs: each whole 32x4 bar, and its filled interior when the
// pool fills any of it, in the walk's own order.
func (v *Viewer) statusBarScreenRects(kind terrain.StatusBarKind) (bars, fills []screenRect) {
	for _, b := range v.statusBars() {
		if b.Kind != kind {
			continue
		}
		lift := v.cellLift(b.Cell)
		if r, in := v.placeLifted(lift, b.Rect); in {
			bars = append(bars, r)
		}
		if b.Fill == 0 {
			continue
		}
		interior := image.Rect(b.Rect.Min.X+4, b.Rect.Min.Y, b.Rect.Min.X+4+b.Fill, b.Rect.Max.Y)
		if r, in := v.placeLifted(lift, interior); in {
			fills = append(fills, r)
		}
	}
	return bars, fills
}

// healthBarScreenRects and manaBarScreenRects are statusBarScreenRects of one
// kind each.
func (v *Viewer) healthBarScreenRects() (bars, fills []screenRect) {
	return v.statusBarScreenRects(terrain.HealthBar)
}

func (v *Viewer) manaBarScreenRects() (bars, fills []screenRect) {
	return v.statusBarScreenRects(terrain.ManaBar)
}

// hbEntities are AC-12's five pairs, one unit each on a cell of its own. Only
// the first three carry a bar: the fourth is dead and the fifth has no health
// system, and each fails a different one of the two clauses.
func hbEntities() []MapEntity {
	return []MapEntity{
		{ID: 1, Cell: image.Pt(1, 1), Life: LifeAlive, HP: 100, MaxHP: 100},
		{ID: 2, Cell: image.Pt(2, 1), Life: LifeAlive, HP: 75, MaxHP: 100},
		{ID: 3, Cell: image.Pt(3, 1), Life: LifeDowned, HP: 0, MaxHP: 100},
		{ID: 4, Cell: image.Pt(4, 1), Life: LifeDead, HP: -1, MaxHP: 100},
		{ID: 5, Cell: image.Pt(5, 1), Life: LifeAlive, HP: 3, MaxHP: 0},
	}
}

// hbViewer is a viewer over an 8x8 flat grid holding those five, with the camera
// where the clamp puts it — the same fixture the pick tests use, so a bar and a
// mark are placed by one camera in every case below.
func hbViewer(t *testing.T) *Viewer {
	t.Helper()
	v := commandViewer(t)
	v.SetEntities(hbEntities())
	return v
}

// TestOnlyTheLivingWithAHealthSystemCarryABar is AC-12's selection half: three
// bars for the five units, and only two fills, because a downed unit's bar is
// its two caps with an empty interior.
//
// The counts are compared against a hand-written expectation and the widths
// against each other: the full fill is wider than the three-quarter one, which
// is what says the fill follows the ratio rather than merely existing.
func TestOnlyTheLivingWithAHealthSystemCarryABar(t *testing.T) {
	v := hbViewer(t)
	grounds, fills := v.healthBarScreenRects()

	if len(grounds) != 3 {
		t.Fatalf("%d bar(s) over five units, want 3 — the dead one and the one with no health system "+
			"carry none", len(grounds))
	}
	if len(fills) != 2 {
		t.Fatalf("%d fill(s), want 2 — a downed unit's bar is its caps and not an absent bar",
			len(fills))
	}
	// The three bars are one size, because the picture is fixed.
	for i, g := range grounds {
		if g.W != grounds[0].W || g.H != grounds[0].H {
			t.Errorf("bar %d is %vx%v and the first is %vx%v — the bar is a fixed size",
				i, g.W, g.H, grounds[0].W, grounds[0].H)
		}
	}
	if fills[0].W*terrain.StatusBarWidth != grounds[0].W*terrain.StatusBarInterior {
		t.Errorf("the full-health fill is %v wide and its bar %v — full health fills the whole interior",
			fills[0].W, grounds[0].W)
	}
	if !(fills[1].W < fills[0].W && fills[1].W > 0) {
		t.Errorf("the three-quarter fill is %v wide against a full one of %v", fills[1].W, fills[0].W)
	}
}

// TestOnlyMagesCarryABlueManaBar covers the complete owner-directed pool
// projection: full, partial and empty mana still have a bar; only positive
// mana has a fill; a non-mage with no maximum has no meaningless zero bar.
func TestOnlyMagesCarryABlueManaBar(t *testing.T) {
	v := commandViewer(t)
	v.SetEntities([]MapEntity{
		{ID: 1, Cell: image.Pt(1, 2), Life: LifeAlive, Mana: 40, MaxMana: 40},
		{ID: 2, Cell: image.Pt(2, 2), Life: LifeAlive, Mana: 10, MaxMana: 40},
		{ID: 3, Cell: image.Pt(3, 2), Life: LifeAlive, Mana: 0, MaxMana: 40},
		{ID: 4, Cell: image.Pt(4, 2), Life: LifeAlive, Mana: 0, MaxMana: 0},
	})
	grounds, fills := v.manaBarScreenRects()
	if len(grounds) != 3 || len(fills) != 2 {
		t.Fatalf("mana bars: %d bars and %d fills, want 3 and 2", len(grounds), len(fills))
	}
	if fills[0].W*terrain.StatusBarWidth != grounds[0].W*terrain.StatusBarInterior || !(fills[1].W > 0 && fills[1].W < fills[0].W) {
		t.Fatalf("mana fill widths full=%v partial=%v bar=%v", fills[0].W, fills[1].W, grounds[0].W)
	}
	// None of the four is selected, so each filled mana interior draws its
	// brightest row as the faded #0000ff: one run per filled bar.
	blue := terrain.StatusBarFaded(color.RGBA{B: 0xff, A: 0xff})
	passes := v.overlayPasses()
	i := slices.IndexFunc(passes, func(p overlayPass) bool { return p.Color == blue })
	if i < 0 {
		t.Fatal("no blue mana row pass")
	}
	if n := len(passes[i].Rects); n != 2 {
		t.Errorf("blue row pass has %d runs, want 2", n)
	}
}

// TestHealthAndManaBarsStandAboveTheWholeUnitSquare checks final screen
// coordinates, including a multi-cell actor. Mana is immediately below health
// while neither bar crosses the top edge of the actor's square.
func TestHealthAndManaBarsStandAboveTheWholeUnitSquare(t *testing.T) {
	v := commandViewer(t)
	e := MapEntity{ID: 1, Cell: image.Pt(2, 3), Life: LifeAlive, HP: 30, MaxHP: 40,
		Mana: 20, MaxMana: 40, TokenSize: 2}
	v.SetEntities([]MapEntity{e})
	health, _ := v.healthBarScreenRects()
	mana, _ := v.manaBarScreenRects()
	if len(health) != 1 || len(mana) != 1 {
		t.Fatalf("health/mana grounds = %d/%d, want one each", len(health), len(mana))
	}
	cellTop, ok := v.placeArm(e.Cell, image.Rect(e.Cell.X*terrain.CellSize, e.Cell.Y*terrain.CellSize,
		e.Cell.X*terrain.CellSize+e.TokenSize*terrain.CellSize,
		e.Cell.Y*terrain.CellSize+e.TokenSize*terrain.CellSize))
	if !ok {
		t.Fatal("actor square is outside the viewport")
	}
	if health[0].Y+health[0].H > mana[0].Y || mana[0].Y+mana[0].H > cellTop.Y {
		t.Fatalf("health %v and mana %v must be ordered above square %v", health[0], mana[0], cellTop)
	}
	if health[0].X != mana[0].X || health[0].W != mana[0].W {
		t.Fatalf("health %v and mana %v do not share the multi-cell footprint centre", health[0], mana[0])
	}
}

func TestNoUnitAtAllMeansNoBarPasses(t *testing.T) {
	v := commandViewer(t)
	v.SetEntities([]MapEntity{
		{ID: 1, Cell: image.Pt(1, 1)},
		{ID: 2, Cell: image.Pt(2, 2)},
	})
	grounds, fills := v.healthBarScreenRects()
	if len(grounds) != 0 || len(fills) != 0 {
		t.Fatalf("%d bar(s) and %d fill(s) over entries naming no health", len(grounds), len(fills))
	}
	if n := len(v.statusBarPasses()); n != 0 {
		t.Errorf("%d bar pass(es) were built with nothing to draw", n)
	}
}

// TestTheBarPassesSitUnderTheSelectionRimAndOverTheCrosses reads the bar
// passes as statusBarPasses builds them and finds them, in that order and
// unbroken, between the unit cross and the selection rim of the frame's own
// pass slice.
func TestTheBarPassesSitUnderTheSelectionRimAndOverTheCrosses(t *testing.T) {
	v := hbViewer(t)
	v.SetUnits(true, []image.Point{{X: 1, Y: 1}})
	v.sel = selection{1}

	bars := v.statusBarPasses()
	passes := v.overlayPasses()
	first := slices.IndexFunc(passes, func(p overlayPass) bool { return len(bars) > 0 && p.Color == bars[0].Color })
	rim := slices.IndexFunc(passes, func(p overlayPass) bool { return p.Color == terrain.SelectionMarkerColor })
	cross := slices.IndexFunc(passes, func(p overlayPass) bool { return p.Color == terrain.UnitMarkerColor })
	if len(bars) == 0 || first < 0 || rim < 0 || cross < 0 {
		t.Fatalf("the frame holds %d bar pass(es) from %d, rim %d, cross %d — every one must be drawn for this "+
			"order to say anything", len(bars), first, rim, cross)
	}
	for k, b := range bars {
		if i := first + k; i >= len(passes) || passes[i].Color != b.Color || len(passes[i].Rects) != len(b.Rects) {
			t.Fatalf("bar pass %d of %d is not at slice position %d — the bar passes are not one unbroken run", k, len(bars), first+k)
		}
	}
	if !(cross < first && first+len(bars) <= rim) {
		t.Errorf("the passes run cross %d, bars %d..%d, rim %d; want the cross first, then the bars, and the rim last",
			cross, first, first+len(bars)-1, rim)
	}
}

func TestABarTakesTheReliefOffsetItsUnitsMarkTakes(t *testing.T) {
	// Row 2 and below, so every camera here keeps the bars, which stand 16
	// rows above their cells, inside the view.
	cells := []image.Point{{X: 1, Y: 2}, {X: 4, Y: 2}, {X: 6, Y: 5}}
	ents := make([]MapEntity, len(cells))
	for i, c := range cells {
		ents[i] = MapEntity{ID: uint32(i + 1), Cell: c, Life: LifeAlive, HP: 100, MaxHP: 100}
	}

	type cam struct {
		x, y, zoom float64
	}
	var base []float64
	for _, c := range []cam{{0, 0, 1}, {-40, 24, 1}, {17, -9, 2}, {-30, 12, 2}} {
		v := commandViewer(t)
		v.SetEntities(ents)
		v.SetUnits(true, cells)
		v.Camera().X, v.Camera().Y, v.Camera().Zoom = c.x, c.y, c.zoom

		grounds, _ := v.healthBarScreenRects()
		marks := v.unitScreenRects()
		if len(grounds) != len(cells) {
			t.Fatalf("camera %+v: %d ground(s) for %d cells", c, len(grounds), len(cells))
		}
		// Two arms per cross, the horizontal one first — so mark 2k is the arm
		// whose own centre is the cell's anchor on both axes.
		if len(marks) != 2*len(cells) {
			t.Fatalf("camera %+v: %d mark arm(s) for %d cells", c, len(marks), len(cells))
		}

		// The offset from a mark to its bar, in SCREEN pixels divided by the zoom
		// — which is world pixels, and is therefore the same number at every
		// camera if and only if the two went through one transform.
		got := make([]float64, 0, 2*len(cells))
		for i := range cells {
			got = append(got,
				(grounds[i].X-marks[2*i].X)/c.zoom,
				(grounds[i].Y-marks[2*i].Y)/c.zoom)
		}
		if base == nil {
			base = got
			continue
		}
		for k := range got {
			if got[k] != base[k] {
				t.Errorf("camera %+v: the bar stands %v from its mark on axis %d, and %v at the first camera",
					c, got[k], k, base[k])
			}
		}
	}
}

// TestABarIsRefusedOnTheSeamsOwnAnswerAndNotOnTheNumbers is the sharper half of
// which units carry one, and it is why the seam carries a life byte at all.
//
// The entries are deliberately INCONSISTENT: one says dead at full health, one
// says alive below zero. A viewer that re-derived the state from the pair beside
// the byte — a second copy of a rule that lives in pkg/sim — draws a bar for the
// first and refuses one for the second, which is the exact reverse of what the
// world said. Over consistent entries the two readings agree, which is precisely
// why no test built from them could see the difference.
func TestABarIsRefusedOnTheSeamsOwnAnswerAndNotOnTheNumbers(t *testing.T) {
	for _, tc := range []struct {
		name string
		e    MapEntity
		bar  bool
	}{
		{"dead at what would read as full health", MapEntity{ID: 1, Cell: image.Pt(1, 1),
			Life: LifeDead, HP: 100, MaxHP: 100}, false},
		{"alive at what would read as dead", MapEntity{ID: 2, Cell: image.Pt(2, 2),
			Life: LifeAlive, HP: -20, MaxHP: 100}, true},
		{"downed at what would read as alive", MapEntity{ID: 3, Cell: image.Pt(3, 3),
			Life: LifeDowned, HP: 50, MaxHP: 100}, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			v := commandViewer(t)
			v.SetEntities([]MapEntity{tc.e})
			grounds, _ := v.healthBarScreenRects()
			if got := len(grounds) > 0; got != tc.bar {
				t.Errorf("the entry carries a bar: %v, want %v — the state is the world's own answer, "+
					"never one re-derived from the pair beside it", got, tc.bar)
			}
		})
	}
}
