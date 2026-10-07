package ui

import (
	"image"
	"reflect"
	"testing"

	"againrom/pkg/render/camera"
	"againrom/pkg/render/terrain"
)

func nativeSelectionRim(col, row int) []image.Rectangle {
	x0, y0 := 32*col, 32*row
	x1, y1 := x0+32, y0+32
	return []image.Rectangle{
		image.Rect(x0, y0, x1, y0+2),     // top, full width
		image.Rect(x0, y1-2, x1, y1),     // bottom, full width
		image.Rect(x0, y0+2, x0+2, y1-2), // left, between the two
		image.Rect(x1-2, y0+2, x1, y1-2), // right, between the two
	}
}

// wantSelectionScreenRects is the flat oracle: the cell's native rim placed by
// exactly the transform Draw gives a terrain tile — top-left
// cam.WorldToScreen(strip.Min), size the native size scaled by the exported
// cam.Zoom field. It applies no culling, so callers keep the cell inside the
// view or assert the culling themselves.
func wantSelectionScreenRects(cam *camera.Camera, cells []image.Point) []screenRect {
	var want []screenRect
	for _, c := range cells {
		for _, s := range nativeSelectionRim(c.X, c.Y) {
			sx, sy := cam.WorldToScreen(float64(s.Min.X), float64(s.Min.Y))
			want = append(want, screenRect{
				X: sx,
				Y: sy,
				W: float64(s.Dx()) * cam.Zoom,
				H: float64(s.Dy()) * cam.Zoom,
			})
		}
	}
	return want
}

// selectionRects is the highlight pass's rectangles as this file reads them:
// the rects of the pass carrying the selection colour, nil when no such pass was
// built. Reading through overlayPasses rather than a private accessor is
// deliberate — the pass slice is the draw contract, so every assertion below
// holds for exactly what Draw walks.
func selectionRects(v *Viewer) []screenRect {
	for _, p := range v.overlayPasses() {
		if p.Color == terrain.SelectionMarkerColor {
			return p.Rects
		}
	}
	return nil
}

// selectID writes the stored selection directly. See the file header: the tap
// that would produce it is T2's contract, and the last test in this file is the
// one that drives it.
func selectID(v *Viewer, id uint32) { v.sel = selection{id} }

// selectionEntities is the snapshot every highlight below is resolved against:
// three entities on three known cells, with SPARSE ids in an order that is
// neither ascending nor a permutation of their cells, and the id this file
// mostly selects sitting SECOND. Nothing that answered with a slice position
// instead of an id could agree at more than one of them.
//
// The middle one carries art, because AC-9 asks for the unit's art to still
// draw under its own highlight; the other two are art-less squares.
func selectionEntities(artA *terrain.UnitClass) []MapEntity {
	return []MapEntity{
		{ID: 12, Cell: image.Pt(2, 2)},
		{ID: 5, Cell: image.Pt(0, 1), Art: artA, Frame: artA.Frames[0]},
		{ID: 9, Cell: image.Pt(1, 0)},
	}
}

// TestSelectionHighlightIsThatUnitsFootprint — SC-4 (AC-9): a selected id
// present in the snapshot yields ITS OWN cell's footprint rim, displaced and
// flat; the displaced rim carries exactly the -AnchorHeight(cell)-MinV
// offset a marker on that cell carries, applied before the camera transform;
// and the cell is found BY ID, so selecting each of the three ids in turn
// moves the rim to three different cells.
//
// The id half is what makes this more than a transform check. Which entry of the
// snapshot a unit occupies is not stable — the world's order is by id and an
// entity that leaves shifts every later one — so an implementation that took the
// first entity, or one remembered by index, would answer the same cell for all
// three selections here.
func TestSelectionHighlightIsThatUnitsFootprint(t *testing.T) {
	if terrain.CellSize != 32 {
		t.Fatalf("CellSize = %d, want 32; every literal in this file is stated over 32-pixel cells", terrain.CellSize)
	}

	// cliffCanvasH / cliffW*CellSize matches the world on both axes exactly, so
	// the camera sits at the identity and every screen coordinate is a world one.
	v := identityViewer(t, cliffGrid(), cliffW*terrain.CellSize, cliffCanvasH)
	v.SetUnits(true, nil) // These are the opt-in diagnostic rim checks.
	cam := v.Camera()
	if v.Mode() != ModeDisplaced {
		t.Fatalf("Mode() = %v, want displaced — the fixture must select displaced for the lift half to say anything", v.Mode())
	}
	v.SetEntities(selectionEntities(entityArtA()))

	proj := cliffProjection()
	byID := []struct {
		id   uint32
		cell image.Point
	}{
		{5, image.Pt(0, 1)},
		{12, image.Pt(2, 2)},
		{9, image.Pt(1, 0)},
	}

	// Guard the fixture: three distinct cells, or "the rim followed the id"
	// could be satisfied by a rim that never moved.
	seen := map[image.Point]bool{}
	for _, b := range byID {
		if seen[b.cell] {
			t.Fatalf("fixture does not discriminate: cell %v is claimed by two ids", b.cell)
		}
		seen[b.cell] = true
	}

	for _, b := range byID {
		selectID(v, b.id)
		cells := []image.Point{b.cell}

		got := selectionRects(v)
		assertScreenRects(t, got, wantDisplacedRects(cam, proj, cells, nativeSelectionRim))

		// The lift stated explicitly rather than only implied by the match: only
		// Y moves, and it moves by exactly the AnchorHeight/MinV offset the four
		// shipped glyphs take for the same cell.
		dy := -proj.AnchorHeight(b.cell.X, b.cell.Y) - proj.MinV
		flat := wantSelectionScreenRects(cam, cells)
		if len(got) != len(flat) {
			t.Fatalf("id %d: rectangle count changed under displacement: %d -> %d", b.id, len(flat), len(got))
		}
		for i := range got {
			if got[i].X != flat[i].X || got[i].W != flat[i].W || got[i].H != flat[i].H {
				t.Fatalf("id %d strip %d: displaced %+v, flat %+v — only Y may move", b.id, i, got[i], flat[i])
			}
			if offset := got[i].Y - flat[i].Y; offset != float64(dy) {
				t.Fatalf("id %d strip %d: Y offset = %v, want the AnchorHeight/MinV offset %d", b.id, i, offset, dy)
			}
		}
		if dy == 0 {
			t.Fatalf("fixture does not discriminate: cell %v lifts by 0, so the displaced half of this case "+
				"is indistinguishable from the flat one", b.cell)
		}
	}

	// The flat half: over the SAME valid, sloped altitude grid, flat mode carries
	// no lift at all. v.proj stays non-nil throughout, which is why the guard
	// inside the shared transform must be Mode() and not v.proj != nil.
	v.SetFlat(true)
	if v.Mode() != ModeFlat {
		t.Fatalf("SetFlat(true): Mode() = %v, want flat", v.Mode())
	}
	if v.proj == nil {
		t.Fatal("SetFlat(true): v.proj is nil — the projection must still exist; only the mode changed")
	}
	for _, b := range byID {
		selectID(v, b.id)
		assertScreenRects(t, selectionRects(v), wantSelectionScreenRects(v.Camera(), []image.Point{b.cell}))
	}
}

// TestSelectionHighlightAbsentOrNoneDrawsNothing — SC-4 (AC-9): the other
// two of AC-9's three states, plus the one the world produces on its own.
//
// Nothing selected draws no highlight; a selected id the most recent snapshot no
// longer holds draws none either; and a selected unit standing OFF the map draws
// none, because the render tier's shared off-map rejection answers for the rim
// exactly as it answers for the four glyphs beside it. In every case the pass is
// omitted entirely rather than appended empty, so the frame is the one that
// would have been drawn with no selection at all.
//
// The control at the end is what makes each of these an observation rather than
// a tautology: the SAME viewer, given a present id, does produce the pass.
func TestSelectionHighlightAbsentOrNoneDrawsNothing(t *testing.T) {
	v := identityViewer(t, cliffGrid(), cliffW*terrain.CellSize, cliffCanvasH)
	v.SetUnits(true, nil) // These are the opt-in diagnostic rim checks.
	v.SetEntities(selectionEntities(entityArtA()))

	for _, tc := range []struct {
		name string
		sel  selection
	}{
		// "Nothing selected while still carrying an id" was a case here while the
		// selection was an id beside a bool. It is not a state a set can hold:
		// absence is the length, so the id and the "is anything selected"
		// question cannot come apart. What is left of it is the empty non-nil
		// set beside the nil one — two spellings of nothing, one answer.
		{"a fresh viewer's zero value: nothing selected", nil},
		{"an emptied set, non-nil", selection{}},
		{"a selected id the snapshot does not hold", selection{7}},
		{"a selected id no snapshot could hold", selection{1 << 31}},
	} {
		v.sel = tc.sel
		if got := selectionRects(v); got != nil {
			t.Errorf("%s: the highlight pass holds %+v, want none", tc.name, got)
		}
		for i, p := range v.overlayPasses() {
			if p.Color == terrain.SelectionMarkerColor {
				t.Errorf("%s: pass %d carries the selection colour, want no such pass at all", tc.name, i)
			}
		}
	}

	// The world may walk a selected unit off the map — a normally resolved
	// outcome, not an error — and the rim contributes no geometry there.
	t.Run("a selected unit standing off the map", func(t *testing.T) {
		v := identityViewer(t, cliffGrid(), cliffW*terrain.CellSize, cliffCanvasH)
		v.SetUnits(true, nil) // These are the opt-in diagnostic rim checks.
		v.SetEntities([]MapEntity{
			{ID: 5, Cell: image.Pt(-1, 0)},
			{ID: 9, Cell: image.Pt(cliffW, cliffH)},
		})
		for _, id := range []uint32{5, 9} {
			selectID(v, id)
			if got := selectionRects(v); got != nil {
				t.Errorf("id %d off the map: the highlight pass holds %+v, want none", id, got)
			}
		}
	})

	// The control: a present, in-map id on the same viewer DOES draw, so every
	// "want none" above is a real observation of the pass's absence.
	selectID(v, 5)
	if got := selectionRects(v); len(got) != 4 {
		t.Fatalf("the probe does not discriminate: a present selected id yielded %d rects (%+v), want the "+
			"four strips of its cell's rim — none of the cases above could have seen a highlight", len(got), got)
	}
}

// TestSelectionPassIsAppendedLastAndMovesNothing — SC-4's pass-list clause
// (AC-9): with every layer on, the highlight is appended AFTER the three
// diagnostic crosses and the entity layer's two content passes, and the
// passes it was appended to are BYTE-IDENTICAL to the ones the same viewer
// built with nothing selected.
//
// That identity is the whole of the entry's fence stated as an observation: the
// four shipped glyph builders keep their arms, their colours and their places,
// and the new pass is appended rather than inserted. It is also AC-9's last
// clause — the selected unit's art and the marker on its cell both still draw —
// in the strongest form available without reading pixels back: the sprite pass
// and the unit-cross pass are the identical values they were before anything was
// selected.
//
// The second half measures the covering rather than assuming it. The unit cross
// on the highlighted cell must lie INSIDE the rim's own bounding box — or
// "disjoint" would be a statement about two glyphs on different cells and would
// say nothing — and it must share no pixel with any strip of it.
func TestSelectionPassIsAppendedLastAndMovesNothing(t *testing.T) {
	const selCol, selRow = 0, 1

	// ONE art value for both builds: the sprite pass carries the frame's own
	// pointer identity, so two calls to entityArtA would differ there for a
	// reason that has nothing to do with the selection.
	artA := entityArtA()

	viewer := func(t *testing.T, sel bool) *Viewer {
		t.Helper()
		v := staticsIdentityViewer(t, true, true)
		v.SetObjects(true, []image.Point{{X: selCol, Y: selRow}})
		v.SetUnits(true, []image.Point{{X: selCol, Y: selRow}})
		v.SetEntities(selectionEntities(artA))
		if sel {
			selectID(v, 5) // the entity on (selCol, selRow), carrying art
		}
		return v
	}
	build := func(t *testing.T, sel bool) []overlayPass {
		t.Helper()
		return viewer(t, sel).overlayPasses()
	}
	// bandOf is the same frame's CONTENT band, where the selected unit's own
	// art lives since 0068.
	bandOf := func(t *testing.T, sel bool) []staticScreenRect {
		t.Helper()
		return viewer(t, sel).planeSprites()
	}

	before := build(t, false)
	after := build(t, true)

	if len(before) != 4 {
		t.Fatalf("with every layer on and nothing selected the viewer built %d passes, want 4 "+
			"(squares, objects, units, statics)", len(before))
	}
	if len(after) != 5 {
		t.Fatalf("with a unit selected the viewer built %d passes, want 5 — the highlight is APPENDED, "+
			"so it is one more pass and not a replacement", len(after))
	}
	for i := range before {
		if after[i].Color != before[i].Color {
			t.Fatalf("pass %d changed colour from %+v to %+v when a unit was selected; no existing pass moves",
				i, before[i].Color, after[i].Color)
		}
		assertScreenRects(t, after[i].Rects, before[i].Rects)
	}
	// The content band is where the selected unit's own art lives since 0068, and
	// a selection may not move it either (AC-9).
	bandBefore, bandAfter := bandOf(t, false), bandOf(t, true)
	if !reflect.DeepEqual(bandBefore, bandAfter) {
		t.Fatalf("the content band is %+v with a selection and %+v without — the selected unit's own art "+
			"must draw exactly as it did (AC-9)", bandAfter, bandBefore)
	}
	// The selected unit's own art must actually be in the band, or AC-9's clause
	// would be observed over a frame that draws none. The band carries the map's
	// object art too, so this counts the entity's own frame rather than the band.
	var selectedArt []staticScreenRect
	for _, sp := range bandAfter {
		if sp.Frame == artA.Frames[0] {
			selectedArt = append(selectedArt, sp)
		}
	}
	if len(selectedArt) != 1 {
		t.Fatalf("the content band holds %d entries carrying the selected unit's frame, want 1 — the fixture "+
			"must actually draw art on the highlighted cell for AC-9's clause to be observed", len(selectedArt))
	}

	rim := after[4]
	if rim.Color != terrain.SelectionMarkerColor {
		t.Fatalf("the last pass is %+v, want the highlight pass in %+v",
			rim.Color, terrain.SelectionMarkerColor)
	}
	cam := staticsIdentityViewer(t, true, true).Camera()
	assertScreenRects(t, rim.Rects, wantDisplacedRects(cam, cliffProjection(),
		[]image.Point{{X: selCol, Y: selRow}}, nativeSelectionRim))

	// The covering, measured. after[2] is the unit-cross pass, over the very cell
	// the highlight stands on.
	cross := after[2]
	if cross.Color != terrain.UnitMarkerColor || len(cross.Rects) == 0 {
		t.Fatalf("pass 2 is %+v with %d rects, want the unit cross on the highlighted cell",
			cross.Color, len(cross.Rects))
	}
	box := rim.Rects[0]
	for _, r := range rim.Rects[1:] {
		lo, hi := min(box.X, r.X), max(box.X+box.W, r.X+r.W)
		loY, hiY := min(box.Y, r.Y), max(box.Y+box.H, r.Y+r.H)
		box = screenRect{X: lo, Y: loY, W: hi - lo, H: hiY - loY}
	}
	for i, c := range cross.Rects {
		if c.X < box.X || c.Y < box.Y || c.X+c.W > box.X+box.W || c.Y+c.H > box.Y+box.H {
			t.Fatalf("fixture does not discriminate: unit-cross arm %d %+v is not inside the rim's footprint %+v, "+
				"so its disjointness from the rim says nothing about covering", i, c, box)
		}
		for j, s := range rim.Rects {
			if rectsOverlap(c, s) {
				t.Errorf("unit-cross arm %d %+v meets rim strip %d %+v — the highlight is drawn LAST and must "+
					"leave the marker on its own cell visible (FR-2, DD-8)", i, c, j, s)
			}
		}
	}
	for i, sp := range selectedArt {
		for j, s := range rim.Rects {
			if rectsOverlap(sp.screenRect, s) {
				t.Errorf("sprite %d %+v meets rim strip %d %+v — the highlight must leave the selected unit's "+
					"art visible (FR-2)", i, sp.screenRect, j, s)
			}
		}
	}
}

// TestSelectionHighlightFollowsTheTapThatMadeIt closes the seam between T2's
// store and this entry's read: a tap driven through the shipped shell selects a
// unit, and the very next pass slice carries that unit's rim — no parameter, no
// second copy of the selection anywhere.
//
// It runs over command_test.go's own flat 8x8 fixture, whose entities are
// art-less squares with sparse descending ids, so the highlight's cell can only
// have come from the id the tap resolved. Tapping empty ground then clears the
// selection and the pass goes with it, which is the same clause
// TestSelectionHighlightAbsentOrNoneDrawsNothing states over the field written
// directly.
func TestSelectionHighlightFollowsTheTapThatMadeIt(t *testing.T) {
	v := commandViewer(t)
	v.SetUnits(true, nil) // These are the opt-in diagnostic rim checks.
	cam := v.Camera()

	if got := selectionRects(v); got != nil {
		t.Fatalf("a viewer that has been tapped nowhere holds the highlight %+v, want none", got)
	}

	for _, tc := range []struct {
		id       uint32
		col, row int
	}{
		{unitBID, unitBCol, unitBRow},
		{unitAID, unitACol, unitARow},
	} {
		selectUnit(t, v, tc.id, tc.col, tc.row)
		assertScreenRects(t, selectionRects(v), wantSelectionScreenRects(cam, []image.Point{{X: tc.col, Y: tc.row}}))
	}

	// THE RIGHT CLICK CLEARS THE SELECTION, and the pass goes with it.
	x, y := cellPoint(v, emptyCol, emptyRow)
	if _, ok := rightUpAt(v, x, y); ok {
		t.Fatalf("the right click issued an order")
	}
	if len(v.sel) != 0 {
		t.Fatalf("the right click left selection %+v, want none", v.sel)
	}
	if got := selectionRects(v); got != nil {
		t.Fatalf("the highlight pass holds %+v after the selection was cleared, want none — nothing survives a frame", got)
	}
}

// selectIDs writes a stored selection of several ids directly, on the same
// grounds selectID writes one: which units the marks below are for is what this
// file is about, and how the set came to hold them is command_test.go's.
func selectIDs(v *Viewer, ids ...uint32) { v.sel = selection(ids) }

// TestEverySelectedPresentUnitIsMarkedOnItsOwnCell — 0030 SC-6 (AC-6): k
// selected units present in the snapshot yield k marks, each on ITS OWN cell
// and each carrying that cell's own relief lift; an id the snapshot does not
// hold contributes none and does not disturb the others.
//
// THE THREE CELLS LIFT BY THREE DIFFERENT AMOUNTS on this fixture, which is what
// makes the displaced half discriminating: one offset applied to the whole set —
// the shape a group mark would take if it were lifted once rather than per cell
// — would land two of the three rims in the wrong place.
func TestEverySelectedPresentUnitIsMarkedOnItsOwnCell(t *testing.T) {
	v := identityViewer(t, cliffGrid(), cliffW*terrain.CellSize, cliffCanvasH)
	v.SetUnits(true, nil) // These are the opt-in diagnostic rim checks.
	cam := v.Camera()
	if v.Mode() != ModeDisplaced {
		t.Fatalf("Mode() = %v, want displaced", v.Mode())
	}
	art := entityArtA()
	// Three units on cells this fixture lifts by THREE DIFFERENT amounts, in a
	// slice order that is neither ascending id nor cell order; the middle one
	// carries art, because AC-6 asks for the units' art still to draw under the
	// marks.
	v.SetEntities([]MapEntity{
		{ID: 12, Cell: image.Pt(0, 0)},
		{ID: 5, Cell: image.Pt(1, 1), Art: art, Frame: art.Frames[0]},
		{ID: 9, Cell: image.Pt(0, 3)},
	})
	proj := cliffProjection()

	// The expectation is stated in ASCENDING ID — the order the set is held in
	// and the order presentSelected walks — and the cells are transcribed here
	// rather than read back off the snapshot.
	allCells := []image.Point{
		{X: 1, Y: 1}, // id 5
		{X: 0, Y: 3}, // id 9
		{X: 0, Y: 0}, // id 12
	}

	lifts := map[image.Point]int{}
	for _, c := range allCells {
		lifts[c] = -proj.AnchorHeight(c.X, c.Y) - proj.MinV
	}
	if lifts[allCells[0]] == lifts[allCells[1]] || lifts[allCells[1]] == lifts[allCells[2]] {
		t.Fatalf("fixture does not discriminate: the three cells lift by %v, and a per-set lift would "+
			"be indistinguishable from a per-cell one", lifts)
	}

	for _, tc := range []struct {
		name  string
		ids   []uint32
		cells []image.Point
	}{
		{"all three present", []uint32{5, 9, 12}, allCells},
		{"two of the three", []uint32{5, 12}, []image.Point{allCells[0], allCells[2]}},
		{"one, as before", []uint32{9}, []image.Point{allCells[1]}},
		{"an absent id beside two present ones", []uint32{5, 7, 12},
			[]image.Point{allCells[0], allCells[2]}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			selectIDs(v, tc.ids...)
			assertScreenRects(t, selectionRects(v), wantDisplacedRects(cam, proj, tc.cells, nativeSelectionRim))
		})
	}

	// The flat half, over the SAME valid sloped grid: no lift at all, k marks
	// still one per present unit.
	v.SetFlat(true)
	if v.Mode() != ModeFlat {
		t.Fatalf("SetFlat(true): Mode() = %v, want flat", v.Mode())
	}
	if v.proj == nil {
		t.Fatal("SetFlat(true): v.proj is nil — the projection must still exist; only the mode changed")
	}
	selectIDs(v, 5, 9, 12)
	assertScreenRects(t, selectionRects(v), wantSelectionScreenRects(v.Camera(), allCells))

	// AC-6's last clause: the marks do not replace what they mark. With all three
	// selected the frame still carries the art of the one unit that resolves and
	// the entity square of the two that do not.
	sprites, squares := len(v.planeSprites()), 0
	for _, p := range v.overlayPasses() {
		if p.Color == terrain.EntityMarkerColor {
			squares += len(p.Rects)
		}
	}
	if sprites != 1 {
		t.Errorf("the frame draws %d unit sprites with three units marked, want 1", sprites)
	}
	if squares != 2 {
		t.Errorf("the frame draws %d entity squares with three units marked, want 2", squares)
	}
}

// TestASelectionWithNoPresentMemberAppendsNoPass — 0030 SC-6 (AC-6): the
// empty selection and the selection every member of which is gone from the
// snapshot both leave the pass slice EXACTLY as a viewer with nothing
// selected leaves it.
//
// It is asserted on the pass slice itself and not on an empty rect list, which
// is the difference between "no pass was appended" and "a pass was appended
// carrying nothing": the second draws no pixel either, and would pass a test
// written the other way while moving every pass count this package pins.
func TestASelectionWithNoPresentMemberAppendsNoPass(t *testing.T) {
	base := func(t *testing.T) *Viewer {
		t.Helper()
		v := identityViewer(t, cliffGrid(), cliffW*terrain.CellSize, cliffCanvasH)
		v.SetUnits(true, nil) // These are the opt-in diagnostic rim checks.
		v.SetEntities(selectionEntities(entityArtA()))
		return v
	}

	unselected := len(base(t).overlayPasses())
	if unselected == 0 {
		t.Fatalf("the fixture builds no passes at all with nothing selected, so a count below says nothing")
	}

	for _, tc := range []struct {
		name string
		sel  selection
	}{
		{"nothing selected", nil},
		{"an emptied set, non-nil", selection{}},
		{"every member absent from the snapshot", selection{7, 8, 1 << 31}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			v := base(t)
			v.sel = tc.sel

			passes := v.overlayPasses()
			if len(passes) != unselected {
				t.Errorf("the frame holds %d passes, want the %d a viewer with nothing selected holds",
					len(passes), unselected)
			}
			for i, p := range passes {
				if p.Color == terrain.SelectionMarkerColor {
					t.Errorf("pass %d carries the selection colour, want no such pass at all", i)
				}
			}
		})
	}

	// The control: the same viewer, given ids the snapshot DOES hold, appends
	// one pass carrying three cells' worth of rim — so every "no pass" above is
	// an observation of the pass's absence rather than of a fixture that never
	// builds one.
	v := base(t)
	selectIDs(v, 5, 9, 12)
	if got, want := len(v.overlayPasses()), unselected+1; got != want {
		t.Fatalf("with three present ids selected the frame holds %d passes, want %d", got, want)
	}
	if got := selectionRects(v); len(got) != 12 {
		t.Fatalf("three marks yielded %d rects, want 12 — four strips each", len(got))
	}
}
