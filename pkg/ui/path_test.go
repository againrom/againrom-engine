package ui

// The path overlay: which units get a line, and where the line goes.
//
// Every fixture here is a hand-built snapshot and a viewer over a synthetic
// grid; nothing reads a game install and nothing opens a window. The benchmark
// at the end builds an ebitengine image and submits strokes to it, which runs
// before the game starts the same way the static layer's own draw fixtures do.

import (
	"fmt"
	"image"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"againrom/pkg/render/camera"
	"againrom/pkg/render/terrain"
)

// pthEntity is one targeted unit as the seam delivers it: an id, a cell and the
// cells it still means to walk.
func pthEntity(id uint32, col, row int, route ...image.Point) MapEntity {
	return MapEntity{ID: id, Cell: image.Pt(col, row), Route: route}
}

// TestOnlySelectedUnitsUnderOrdersArePreviewed is 0037 AC-10: the overlay is
// scoped to the selection, and inside it to the units that actually hold a
// route.
//
// The three exclusions are asserted separately because they are three different
// reasons a unit draws no line, and a rule that answered "none of them" for one
// reason only would pass a test that lumped them together.
func TestOnlySelectedUnitsUnderOrdersArePreviewed(t *testing.T) {
	ents := []MapEntity{
		pthEntity(1, 0, 0, image.Pt(1, 0), image.Pt(2, 0)),
		pthEntity(2, 4, 4, image.Pt(5, 4)),
		pthEntity(3, 8, 8), // selected, but under no order
		{ID: 4, Cell: image.Pt(9, 9), Route: []image.Point{image.Pt(9, 8)}, Life: LifeDead},
	}

	for _, tc := range []struct {
		what string
		sel  selection
		want []uint32
	}{
		{"nothing selected previews nothing", nil, nil},
		{"one selected unit under orders", selection{1}, []uint32{1}},
		{"a targeted unit that is NOT selected is excluded", selection{2}, []uint32{2}},
		{"a selected unit holding no route is excluded", selection{3}, nil},
		{"a selected CORPSE is excluded, route and all", selection{4}, nil},
		{"a selection of several keeps only the ones under orders", selection{1, 2, 3}, []uint32{1, 2}},
		{"an id the snapshot no longer holds is skipped", selection{1, 99}, []uint32{1}},
	} {
		got := pathsToDraw(tc.sel, ents)
		if len(got) != len(tc.want) {
			t.Errorf("%s: %d previewed, want %d", tc.what, len(got), len(tc.want))
			continue
		}
		for i, e := range got {
			if e.ID != tc.want[i] {
				t.Errorf("%s: entry %d is unit %d, want %d", tc.what, i, e.ID, tc.want[i])
			}
		}
	}
}

// TestEveryDrawableUnitIsPreviewedWhateverTheCount is 0037 AC-11.
//
// The count of drawable units is not a reason to draw fewer lines and is above
// all not a reason to draw none: the cap this criterion used to assert was a
// taste judgement with no claim behind it, and its failure mode was the whole
// overlay disappearing (FR-8a).
//
// The counts below straddle the value that cap held — 31, 32, 33 — because a
// rule that had merely been RELAXED to some larger number would still answer
// correctly at all three, and a rule that still skipped whole at 32 would not.
// Two hundred and two thousand are past any number a relaxation would plausibly
// have chosen, and the last case is the same shape one order of magnitude on.
func TestEveryDrawableUnitIsPreviewedWhateverTheCount(t *testing.T) {
	build := func(targeted, idle int) ([]MapEntity, selection) {
		var ents []MapEntity
		var sel selection
		for i := 0; i < targeted; i++ {
			id := uint32(1 + i)
			ents = append(ents, pthEntity(id, i, 0, image.Pt(i, 1)))
			sel = append(sel, id)
		}
		for i := 0; i < idle; i++ {
			id := uint32(100000 + i)
			ents = append(ents, pthEntity(id, i, 5))
			sel = append(sel, id)
		}
		return ents, sel
	}

	for _, tc := range []struct {
		what     string
		targeted int
		idle     int
		want     int
	}{
		{"one under the old cap", 31, 0, 31},
		{"exactly the old cap", 32, 0, 32},
		{"one over the old cap, every line still drawn", 33, 0, 33},
		{"two hundred selected movers are two hundred previews", 200, 0, 200},
		{"two thousand of them are two thousand previews", 2000, 0, 2000},
		{"a large selection of which few are moving previews those few", 3, 500, 3},
	} {
		ents, sel := build(tc.targeted, tc.idle)
		got := pathsToDraw(sel, ents)
		if len(got) != tc.want {
			t.Errorf("%s: %d previewed, want %d", tc.what, len(got), tc.want)
			continue
		}
		// Not merely the right COUNT: the right units, in the selection's order.
		for i, e := range got {
			if e.ID != uint32(1+i) {
				t.Errorf("%s: entry %d is unit %d, want %d", tc.what, i, e.ID, 1+i)
				break
			}
		}
	}
}

// TestThePreviewedLineRunsFromTheUnitAlongItsRoute is 0037 AC-12.
//
// The route the seam carries is what is LEFT to walk, so the first leg has to be
// drawn from the unit's own cell to the route's head — a line that started at
// the head would leave the unit unattached to its own path, and one that started
// at the cell behind it would trail a stub. The whole geometry is asserted
// against the camera's own forward transform rather than against pixel values,
// so the criterion says "the same place every other glyph on that cell goes"
// rather than restating the camera's arithmetic.
func TestThePreviewedLineRunsFromTheUnitAlongItsRoute(t *testing.T) {
	v := commandViewer(t)
	route := []image.Point{image.Pt(3, 2), image.Pt(4, 3), image.Pt(5, 3)}
	v.SetEntities([]MapEntity{pthEntity(7, 2, 2, route...)})
	v.sel = selection{7}

	got := v.pathScreenSegments()
	if len(got) != len(route) {
		t.Fatalf("%d segment(s) for a route of %d cells", len(got), len(route))
	}

	centre := func(c image.Point) (float64, float64) {
		return v.Camera().WorldToScreen(
			float64(c.X*camera.CellSize+camera.CellSize/2),
			float64(c.Y*camera.CellSize+camera.CellSize/2))
	}
	want := append([]image.Point{image.Pt(2, 2)}, route...)
	for i, s := range got {
		x0, y0 := centre(want[i])
		x1, y1 := centre(want[i+1])
		if s.X0 != x0 || s.Y0 != y0 || s.X1 != x1 || s.Y1 != y1 {
			t.Errorf("segment %d is (%v,%v)-(%v,%v), want %v -> %v at (%v,%v)-(%v,%v)",
				i, s.X0, s.Y0, s.X1, s.Y1, want[i], want[i+1], x0, y0, x1, y1)
		}
	}

	// Nothing selected, nothing drawn — the same viewer, one field apart.
	v.sel = nil
	if segs := v.pathScreenSegments(); segs != nil {
		t.Errorf("an empty selection drew %d segment(s)", len(segs))
	}
}

// pthScreen is where a cell's centre lands, read back through the viewer's own
// transform so a fixture asserts its premises in the coordinates the rule uses.
func pthScreen(v *Viewer, c image.Point) (float64, float64) { return v.pathCellCentre(c) }

// TestALegThatCannotReachTheViewIsNotIssued is 0037 AC-13.
//
// It is the reduction that makes FR-8a affordable, and the only reason it is
// allowed at all is that it is EXACT: a leg whose bounding box misses the window
// contains no point inside the window, so dropping it removes no pixel.
func TestALegThatCannotReachTheViewIsNotIssued(t *testing.T) {
	v := commandViewer(t)
	// Two routes that leave the view and come back, one on each axis. The first
	// begins outside it, on the far side from where its first leg ends.
	across := []image.Point{image.Pt(40, 4), image.Pt(60, 4), image.Pt(10, 4)}
	up := []image.Point{image.Pt(2, -20), image.Pt(2, -40), image.Pt(2, 0)}
	v.SetEntities([]MapEntity{
		pthEntity(7, -30, 4, across...),
		pthEntity(8, 2, 0, up...),
	})
	v.sel = selection{7, 8}

	// The premises, asserted rather than assumed: every cell but (10,4) and (2,0)
	// is outside the view, so the kept legs are kept for crossing it and not for
	// having an end inside it.
	w, h := float64(v.Camera().ViewW), float64(v.Camera().ViewH)
	outside := func(c image.Point) bool {
		x, y := pthScreen(v, c)
		return x < 0 || x > w || y < 0 || y > h
	}
	for _, c := range []image.Point{
		image.Pt(-30, 4), image.Pt(40, 4), image.Pt(60, 4), image.Pt(2, -20), image.Pt(2, -40),
	} {
		if !outside(c) {
			t.Fatalf("fixture cell %v is inside the %vx%v view; this test then asserts nothing", c, w, h)
		}
	}
	for _, c := range []image.Point{image.Pt(10, 4), image.Pt(2, 0)} {
		if outside(c) {
			t.Fatalf("fixture cell %v is outside the %vx%v view; the return legs then assert nothing", c, w, h)
		}
	}

	want := [][2]image.Point{
		{image.Pt(-30, 4), image.Pt(40, 4)}, // BOTH ends outside, crossing the whole view
		// (40,4)-(60,4) is wholly past the right edge and is dropped
		{image.Pt(60, 4), image.Pt(10, 4)}, // crosses back in
		{image.Pt(2, 0), image.Pt(2, -20)}, // crosses the top edge
		// (2,-20)-(2,-40) is wholly above the top edge and is dropped
		{image.Pt(2, -40), image.Pt(2, 0)}, // crosses back in
	}
	got := v.pathScreenSegments()
	if len(got) != len(want) {
		t.Fatalf("%d leg(s) issued for a 6-leg pair of routes, want %d", len(got), len(want))
	}
	for i, s := range got {
		x0, y0 := pthScreen(v, want[i][0])
		x1, y1 := pthScreen(v, want[i][1])
		if s.X0 != x0 || s.Y0 != y0 || s.X1 != x1 || s.Y1 != y1 {
			t.Errorf("leg %d is (%v,%v)-(%v,%v), want %v -> %v", i, s.X0, s.Y0, s.X1, s.Y1, want[i][0], want[i][1])
		}
	}
}

// TestAnOffScreenLegIsStillDrawn is 0037 AC-12's second half, and it is the one
// place this overlay deliberately does NOT reuse placeArm.
//
// A leg whose two endpoints are both outside the window may still cross it, so
// culling on the endpoints would make a path blink out exactly when its unit
// walks past the edge of the view. The segment is emitted and the drawer clips
// it. It is also what stops AC-13's reject from being written as an endpoint
// test: this leg's endpoints are both outside and its bounding box is not.
func TestAnOffScreenLegIsStillDrawn(t *testing.T) {
	v := commandViewer(t)
	// Two cells far outside the 8x8 extent and on opposite sides of it, so the
	// leg between them crosses the whole window.
	v.SetEntities([]MapEntity{pthEntity(7, -400, 4, image.Pt(400, 4))})
	v.sel = selection{7}

	got := v.pathScreenSegments()
	if len(got) != 1 {
		t.Fatalf("%d segment(s), want the one leg", len(got))
	}
	s := got[0]
	if s.X0 >= 0 || s.X1 <= float64(v.Camera().ViewW) {
		t.Fatalf("the fixture's endpoints are at x %v and %v, and both must lie outside [0,%d) for this "+
			"to test anything", s.X0, s.X1, v.Camera().ViewW)
	}
}

// ------------------------------------------------- what drawing them all costs

// The measurement FR-8a rests on, and the reason the removed cap is not replaced
// by a bigger number: what a frame pays for is the LEGS it issues, and the count
// of selected units is not that quantity.
//
// The shape is the largest a shipped-size map can produce — every selected mover
// holding a full corner-to-corner route on 256x256, which pkg/sim's own
// TestAFullMapGroupOrderHoldsARouteOfTheMapsDiagonal fixes at 251 cells. Two
// zooms, because the reject's yield is a function of how much map the window
// holds: at the default the map is far larger than the view, and at ZoomMin the
// whole of it is nearly inside.
//
// It reads no game install and opens no window; ebitengine's draw-side calls run
// before the game starts, so the stroke arm measures the CPU half of the real
// call the frame makes — the vertex work and the queue append, never the GPU
// flush. The number that decides the contract is the COUNT, which is exact.
//
//	go test -trimpath -run '^$' -bench BenchmarkThePathOverlayOnAFullMapGroupOrder \
//	    -benchtime=30x ./pkg/ui/
const (
	pthBenchSide  = 256 // a shipped-size map
	pthBenchRoute = 251 // its corner-to-corner route, witnessed in pkg/sim
)

// pthGroupViewer is `units` selected movers on a pthBenchSide map, each walking
// the map's diagonal, at the given zoom.
func pthGroupViewer(b *testing.B, units int, zoom float64) *Viewer {
	b.Helper()
	v, err := NewViewer("paths", grid(pthBenchSide, pthBenchSide), &terrain.Tileset{})
	if err != nil {
		b.Fatalf("NewViewer: %v", err)
	}
	layoutViewport(v, 1280, 720)
	v.Camera().SetZoom(zoom)
	if v.Camera().Zoom != zoom {
		b.Fatalf("zoom clamped to %v, want %v", v.Camera().Zoom, zoom)
	}
	ents := make([]MapEntity, 0, units)
	sel := make(selection, 0, units)
	for i := 0; i < units; i++ {
		start := image.Pt(2+i%16, 2+i/16)
		route := make([]image.Point, pthBenchRoute)
		c := start
		for k := range route {
			if c.X < pthBenchSide-3 {
				c.X++
			}
			if c.Y < pthBenchSide-3 {
				c.Y++
			}
			route[k] = c
		}
		ents = append(ents, MapEntity{ID: uint32(1 + i), Cell: start, Route: route})
		sel = append(sel, uint32(1+i))
	}
	v.SetEntities(ents)
	v.sel = sel
	return v
}

// pthAllSegments is pathScreenSegments WITHOUT the view reject: the same picture,
// every leg issued, which is what the overlay did before this revision. It is the
// arm the reject is measured against.
func pthAllSegments(v *Viewer) []pathSegment {
	var out []pathSegment
	for _, e := range pathsToDraw(v.sel, v.entities) {
		x0, y0 := v.pathCellCentre(e.Cell)
		for _, c := range e.Route {
			x1, y1 := v.pathCellCentre(c)
			out = append(out, pathSegment{X0: x0, Y0: y0, X1: x1, Y1: y1})
			x0, y0 = x1, y1
		}
	}
	return out
}

func BenchmarkThePathOverlayOnAFullMapGroupOrder(b *testing.B) {
	screen := ebiten.NewImage(1280, 720)
	buildArm := func(b *testing.B, v *Viewer, f func(*Viewer) []pathSegment) {
		n := len(f(v))
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			f(v)
		}
		b.StopTimer()
		b.ReportMetric(float64(n), "legs")
		b.ReportMetric(float64(b.Elapsed().Nanoseconds())/float64(b.N)/1e6, "ms/frame")
	}
	strokeArm := func(b *testing.B, segs []pathSegment) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			for _, s := range segs {
				vector.StrokeLine(screen, float32(s.X0), float32(s.Y0), float32(s.X1), float32(s.Y1),
					PathWidth, PathColor, false)
			}
		}
		b.StopTimer()
		b.ReportMetric(float64(len(segs)), "legs")
		b.ReportMetric(float64(b.Elapsed().Nanoseconds())/float64(b.N)/1e6, "ms/frame")
	}
	for _, zoom := range []float64{1, camera.ZoomMin} {
		for _, units := range []int{40, 320} {
			v := pthGroupViewer(b, units, zoom)
			tag := fmt.Sprintf("zoom%g/%dunits", zoom, units)
			b.Run("build-every-leg/"+tag, func(b *testing.B) { buildArm(b, v, pthAllSegments) })
			b.Run("build-reject/"+tag, func(b *testing.B) { buildArm(b, v, (*Viewer).pathScreenSegments) })
			b.Run("stroke-every-leg/"+tag, func(b *testing.B) { strokeArm(b, pthAllSegments(v)) })
			b.Run("stroke-reject/"+tag, func(b *testing.B) { strokeArm(b, v.pathScreenSegments()) })
		}
	}
}
