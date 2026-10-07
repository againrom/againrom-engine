package ui

import (
	"image"
	"math"
	"testing"
)

// hitViewer is the cliff fixture at the identity camera, in command mode, with
// the entities it is handed. Screen coordinates equal world coordinates here,
// so every number below can be read as the world position it names.
func hitViewer(t *testing.T, ents []MapEntity) *Viewer {
	t.Helper()
	v := identityViewer(t, cliffGrid(), cliffW*32, cliffCanvasH)
	if v.Mode() != ModeDisplaced {
		t.Fatalf("the cliff fixture came up %v, want displaced — the defect is invisible flat", v.Mode())
	}
	v.commandMode = true
	v.SetEntities(ents)
	return v
}

// liftedCellY is the world Y of the top edge of cell (col,row)'s footprint as
// the DRAWING places it, from the story's own formula and an independent
// projection. It is the oracle; nothing in the pick path is consulted.
func liftedCellY(col, row int) int {
	p := cliffProjection()
	return row*32 - p.AnchorHeight(col, row) - p.MinV
}

// boxRelease drives one complete box gesture: a press at (x0,y0), a drag past
// the slop to (x1,y1), and the release that is judged. It goes through the
// viewer's own shell rather than calling decide, so what is exercised is the
// path the window takes.
func boxRelease(v *Viewer, x0, y0, x1, y1 int) {
	v.pressX, v.pressY = x0, y0
	v.held, v.boxing = true, true
	v.dragMoved = TapSlop + 1
	v.command(appInput{CursorX: x1, CursorY: y1, PrimaryReleased: true})
}

// hitTapAt drives one complete tap gesture at (x,y): under the slop, so the
// release is judged a tap.
func hitTapAt(v *Viewer, x, y int) {
	v.pressX, v.pressY = x, y
	v.held, v.boxing = true, false
	v.dragMoved = 0
	v.command(appInput{CursorX: x, CursorY: y, PrimaryReleased: true})
}

// TestABoxTakesTheUnitWhereItIsDrawn — 0058 AC-1 and AC-2, the owner's report
// as a test. On a cell whose lift is a whole cell, a box over the rectangle the
// unit's rim occupies selects it, and a box over the cells its stored cell
// flatly resolves to does not.
//
// The two rectangles are DISJOINT here, which is what makes the assertion
// discriminating: cell (1,1) is drawn at world Y 0..32 and its flat lattice
// position is 32..64.
func TestABoxTakesTheUnitWhereItIsDrawn(t *testing.T) {
	const col, row = 1, 1
	drawnY := liftedCellY(col, row)
	if drawnY != 0 {
		t.Fatalf("the fixture's cell (%d,%d) is drawn at world Y %d, want 0 — re-derive the "+
			"numbers below before trusting this test", col, row, drawnY)
	}
	flatY := row * 32
	if drawnY == flatY {
		t.Fatalf("drawn and flat coincide at Y %d; this fixture cannot show the defect", drawnY)
	}

	t.Run("a box over where it is drawn selects it", func(t *testing.T) {
		v := hitViewer(t, []MapEntity{{ID: 7, Cell: image.Pt(col, row)}})
		boxRelease(v, col*32+4, drawnY+4, col*32+28, drawnY+28)
		if got := v.sel; len(got) != 1 || got[0] != 7 {
			t.Errorf("selection %v, want [7] — the box was drawn over the unit's own rim", got)
		}
	})

	t.Run("a box over the flat cell selects nothing", func(t *testing.T) {
		v := hitViewer(t, []MapEntity{{ID: 7, Cell: image.Pt(col, row)}})
		boxRelease(v, col*32+4, flatY+4, col*32+28, flatY+28)
		if got := v.sel; len(got) != 0 {
			t.Errorf("selection %v, want none — the flat lattice is not where the unit is drawn", got)
		}
	})

	t.Run("the unit two cells up is no longer taken instead", func(t *testing.T) {
		// The defect's own signature: a box aimed at the unit on (1,1) used to
		// catch whatever stood on the cell the flat lattice named. Put a second
		// unit there and require that it is NOT what comes back.
		v := hitViewer(t, []MapEntity{
			{ID: 7, Cell: image.Pt(col, row)},
			{ID: 9, Cell: image.Pt(col, 0)},
		})
		boxRelease(v, col*32+4, drawnY+4, col*32+28, drawnY+28)
		got := v.sel
		for _, id := range got {
			if id == 9 {
				t.Errorf("selection %v holds the unit on the neighbouring cell — the pick is "+
					"still answering about the flat lattice", got)
			}
		}
		if len(got) != 1 || got[0] != 7 {
			t.Errorf("selection %v, want [7]", got)
		}
	})
}

func TestATapAndAOnePointBoxAskOneQuestion(t *testing.T) {
	ents := []MapEntity{
		{ID: 20, Cell: image.Pt(0, 0)},
		{ID: 6, Cell: image.Pt(0, 1)},
		{ID: 7, Cell: image.Pt(1, 1)},
		{ID: 11, Cell: image.Pt(2, 3)},
	}
	if liftedCellY(0, 0) != liftedCellY(0, 1) {
		t.Fatal("the fixture no longer contains an overlap; it would pass vacuously")
	}

	sawOverlap := false
	for y := -8; y < cliffCanvasH+8; y += 3 {
		for x := -8; x < cliffW*32+8; x += 3 {
			vt := hitViewer(t, ents)
			hitTapAt(vt, x, y)

			// THE BOX SIDE IS READ OFF THE HIT TEST AND NOT OFF A RELEASE. A
			// one-point rectangle covers no AREA, and the decoded rectangle form
			// takes a unit only where it covers strictly more than half of that
			// unit's own rectangle (`AI-SELECT-122`), so driving one through
			// boxRelease now takes nothing whatever it meets. What C-1 claims is
			// still true and is what is asserted here: both forms ask ONE hit test of
			// the same pixel, and picked is that test.
			vb := hitViewer(t, ents)
			var box selection
			for _, e := range picked(vb.entities, vb.entityPickRect, float64(x), float64(y), float64(x), float64(y)) {
				box = append(box, e.ID)
			}

			tap := vt.sel
			if len(box) > 1 {
				sawOverlap = true
			}

			// The tap is empty exactly when the box is.
			if (len(tap) == 0) != (len(box) == 0) {
				t.Fatalf("at (%d,%d) the tap selected %v and the one-point box %v — one hit test, "+
					"so they agree about whether anything was hit at all", x, y, tap, box)
			}
			if len(tap) == 0 {
				continue
			}
			// And it is the lowest id of exactly the box's own set.
			if len(tap) != 1 {
				t.Fatalf("at (%d,%d) the tap selected %v; a tap selects at most one", x, y, tap)
			}
			want := box[0]
			for _, id := range box {
				if id < want {
					want = id
				}
			}
			if tap[0] != want {
				t.Fatalf("at (%d,%d) the tap selected %v and the one-point box %v — the tap must be "+
					"the lowest id of the box's own set", x, y, tap, box)
			}
		}
	}
	if !sawOverlap {
		t.Fatal("the sweep never found a point two hit targets hold; it discriminated nothing")
	}
}

// TestAnEmptyOrInvertedHitTargetIsNeverHit — 0058 error cases: the half-open
// reading is only correct for a rectangle with area, and a viewer whose zoom was
// assigned directly can produce one without.
func TestAnEmptyOrInvertedHitTargetIsNeverHit(t *testing.T) {
	for _, r := range []screenRect{
		{X: 10, Y: 10, W: 0, H: 8},
		{X: 10, Y: 10, W: 8, H: 0},
		{X: 10, Y: 10, W: -8, H: -8},
		{X: 10, Y: 10, W: math.NaN(), H: 8},
	} {
		if r.meets(0, 0, 100, 100) {
			t.Errorf("%+v was hit by a box that covers it; a target with no area is not a target", r)
		}
		if r.meets(10, 10, 10, 10) {
			t.Errorf("%+v was hit by a tap on its own corner", r)
		}
	}
}

func TestThePickRectIsTheRimRect(t *testing.T) {
	for _, cell := range []image.Point{{X: 0, Y: 0}, {X: 1, Y: 1}, {X: 2, Y: 2}, {X: 1, Y: 3}} {
		v := hitViewer(t, []MapEntity{{ID: 4, Cell: cell}})
		v.sel = selection{4}

		pick, ok := v.entityPickRect(v.entities[0])
		if !ok {
			t.Fatalf("cell %v has no pick rectangle", cell)
		}
		strips := v.selectionScreenRects()
		if len(strips) == 0 {
			t.Fatalf("cell %v drew no rim", cell)
		}

		union := strips[0]
		for _, s := range strips[1:] {
			union = screenRect{
				X: math.Min(union.X, s.X),
				Y: math.Min(union.Y, s.Y),
				W: math.Max(union.X+union.W, s.X+s.W),
				H: math.Max(union.Y+union.H, s.Y+s.H),
			}
			union.W -= union.X
			union.H -= union.Y
		}
		if union != pick {
			t.Errorf("cell %v: the rim spans %+v and the pick tests %+v — one rectangle, or the "+
				"two can drift again", cell, union, pick)
		}
	}
}

// TestThePickCarriesTheEntitysOwnDisplacement — 0058 AC-7: a unit drawn
// part-way between two cells is caught by a box over where it is DRAWN, and not
// by one over the cell it is standing in.
func TestThePickCarriesTheEntitysOwnDisplacement(t *testing.T) {
	// Half-way through a tick, having stepped one cell down: the drawing places
	// it 16 world pixels back toward the cell it left, plus the relief term.
	e := MapEntity{ID: 5, Cell: image.Pt(1, 1), Step: image.Pt(0, 1)}
	v := hitViewer(t, []MapEntity{e})
	v.SetPhase(500, 1000)

	shift := v.entityShift(e)
	if shift == (image.Point{}) {
		t.Fatal("the fixture produced no displacement; it cannot show what it claims to")
	}
	drawnY := liftedCellY(1, 1) + shift.Y

	boxRelease(v, 1*32+4, drawnY+4, 1*32+28, drawnY+28)
	if got := v.sel; len(got) != 1 || got[0] != 5 {
		t.Errorf("selection %v, want [5] — a box over where a walking unit is drawn must take it", got)
	}
}

// TestTheLowerIdWinsWhereTwoPlacedRectsHoldOnePoint — 0058 AC-5. Relief makes
// this reachable between DIFFERENT cells, where before it needed two units on
// one cell: the cliff's own face puts cells (0,0) and (0,1) at the same world Y
// in the same column, so their footprints coincide exactly and a point in
// either is a point in both.
func TestTheLowerIdWinsWhereTwoPlacedRectsHoldOnePoint(t *testing.T) {
	a, b := image.Pt(0, 0), image.Pt(0, 1)
	ya, yb := liftedCellY(a.X, a.Y), liftedCellY(b.X, b.Y)
	if ya != yb {
		t.Fatalf("the fixture draws %v at %d and %v at %d; without a coincidence this test "+
			"discriminates nothing — re-derive the pair before changing it", a, ya, b, yb)
	}
	y := ya

	for _, order := range [][]MapEntity{
		{{ID: 20, Cell: a}, {ID: 6, Cell: b}},
		{{ID: 6, Cell: b}, {ID: 20, Cell: a}},
	} {
		v := hitViewer(t, order)
		hitTapAt(v, 16, y+16)
		if got := v.sel; len(got) != 1 || got[0] != 6 {
			t.Errorf("with the snapshot holding %d then %d, the tap selected %v, want [6] — the "+
				"tie is settled over ids, never over slice position", order[0].ID, order[1].ID, got)
		}
	}
}

// TestABoxIsOrientationIndependentOnRelief — 0058 AC-6: a rectangle dragged
// right-to-left and bottom-to-top selects exactly what its corner-swapped twin
// selects, on a fixture where the placement is doing real work.
func TestABoxIsOrientationIndependentOnRelief(t *testing.T) {
	ents := []MapEntity{
		{ID: 7, Cell: image.Pt(1, 1)},
		{ID: 3, Cell: image.Pt(0, 0)},
		{ID: 11, Cell: image.Pt(2, 2)},
	}
	x0, y0, x1, y1 := 4, 4, 90, 150

	base := hitViewer(t, ents)
	boxRelease(base, x0, y0, x1, y1)
	want := base.sel
	if len(want) < 2 {
		t.Fatalf("the reference box caught %v; it must catch several to discriminate", want)
	}

	for _, tc := range [][4]int{{x1, y0, x0, y1}, {x0, y1, x1, y0}, {x1, y1, x0, y0}} {
		v := hitViewer(t, ents)
		boxRelease(v, tc[0], tc[1], tc[2], tc[3])
		got := v.sel
		if len(got) != len(want) {
			t.Fatalf("box %v selected %v, want %v", tc, got, want)
		}
		for i := range got {
			if got[i] != want[i] {
				t.Fatalf("box %v selected %v, want %v", tc, got, want)
			}
		}
	}
}

func TestTheOrderDestinationFollowsTheDrawnGround(t *testing.T) {
	const col, row = 1, 1
	drawnY := liftedCellY(col, row)
	if drawnY != 0 {
		t.Fatalf("the fixture's cell (%d,%d) is drawn at world Y %d, want 0 — re-derive the "+
			"numbers below before trusting this test", col, row, drawnY)
	}

	// THE SELECTED UNIT STANDS AWAY FROM THE CLICK. An actor under the pointer
	// puts up the `select` cursor (`AI-CURSOR-226` arm 4) and a click under it
	// selects rather than orders, so a fixture that selected the unit standing
	// on the clicked cell would witness a selection and not a destination. The
	// premise below refuses the fixture if that happens again.
	v := hitViewer(t, []MapEntity{{ID: 7, Cell: image.Pt(3, 0)}})
	v.sel = selection{7}

	x, y := col*32+16, drawnY+16 // the drawn footprint's own centre
	if id, hit := topAt(v.entities, v.entityPickRect, float64(x), float64(y)); hit {
		t.Fatalf("the fixture point (%d,%d) hits entity %d, so the click is a selection and not an "+
			"order; move the unit before trusting this test", x, y, id)
	}

	// THE DEFECT'S OWN SIGNATURE, asserted directly and BEFORE the press: the
	// camera's flat inverse alone — the whole of what the ground pick answered
	// before the first DIV-044 correction — names a DIFFERENT cell at this exact
	// point, which is what "the click is not bound to the point the drawing
	// corresponds to" meant.
	flatCol, flatRow, insideFlat := v.Camera().ScreenToCell(float64(x), float64(y))
	if !insideFlat {
		t.Fatalf("the fixture point (%d,%d) is off the flat extent; pick another", x, y)
	}
	if flatCol == col && flatRow == row {
		t.Fatalf("the flat inverse already names (%d,%d) at this point; the fixture does not "+
			"discriminate the defect — re-derive it before trusting this test", flatCol, flatRow)
	}

	ords, ok := tapAt(v, x, y)
	if !ok || len(ords) != 1 {
		t.Fatalf("the tap issued %v (ok=%v), want one order", ords, ok)
	}
	if ords[0].x != col || ords[0].y != row {
		t.Errorf("the order names cell (%d,%d), want the drawn cell (%d,%d) — the flat inverse "+
			"names (%d,%d) at this point, which is the owner's report reproduced",
			ords[0].x, ords[0].y, col, row, flatCol, flatRow)
	}
}

func TestTheOrderDestinationIsTheFlatInverseOnFlatGround(t *testing.T) {
	v := identityViewer(t, grid(4, 4), 4*32, 4*32)
	if v.Mode() != ModeFlat {
		t.Fatalf("the fixture came up %v, want flat", v.Mode())
	}
	v.commandMode = true
	v.SetEntities([]MapEntity{{ID: 7, Cell: image.Pt(0, 0)}})
	v.sel = selection{7}

	const sx, sy = 40, 40
	wantCol, wantRow, inside := v.Camera().ScreenToCell(sx, sy)
	if !inside {
		t.Fatalf("the fixture point (%d,%d) is off the extent; pick another", sx, sy)
	}

	ords, ok := tapAt(v, sx, sy)
	if !ok || len(ords) != 1 {
		t.Fatalf("the tap issued %v (ok=%v), want one order", ords, ok)
	}
	if ords[0].x != wantCol || ords[0].y != wantRow {
		t.Errorf("the order names cell (%d,%d), want the flat inverse's (%d,%d)",
			ords[0].x, ords[0].y, wantCol, wantRow)
	}
}

func TestOnAViewerWithNoAltitudesEveryPickRectIsTheFlatCell(t *testing.T) {
	v := identityViewer(t, grid(4, 4), 4*32, 4*32)
	if v.Mode() != ModeFlat {
		t.Fatalf("a grid with no altitude layer came up %v, want flat", v.Mode())
	}
	for row := 0; row < 4; row++ {
		for col := 0; col < 4; col++ {
			e := MapEntity{ID: uint32(row*4 + col), Cell: image.Pt(col, row)}
			v.SetEntities([]MapEntity{e})
			got, ok := v.entityPickRect(e)
			if !ok {
				t.Fatalf("cell (%d,%d) has no pick rectangle", col, row)
			}
			want := screenRect{X: float64(col * 32), Y: float64(row * 32), W: 32, H: 32}
			if got != want {
				t.Errorf("cell (%d,%d) picks on %+v, want the flat cell %+v", col, row, got, want)
			}
		}
	}
}

func TestOnFlatGroundOnlyAMidStepUnitMoves(t *testing.T) {
	e := MapEntity{ID: 5, Cell: image.Pt(2, 2), Step: image.Pt(0, 1)}
	v := identityViewer(t, grid(4, 4), 4*32, 4*32)
	if v.Mode() != ModeFlat {
		t.Fatalf("the fixture came up %v, want flat", v.Mode())
	}
	v.commandMode = true
	v.SetEntities([]MapEntity{e})

	t.Run("standing still it is picked on its own cell", func(t *testing.T) {
		got, ok := v.entityPickRect(e)
		if !ok {
			t.Fatal("no hit target")
		}
		want := screenRect{X: 2 * 32, Y: 2 * 32, W: 32, H: 32}
		if got != want {
			t.Errorf("hit target %+v, want the flat cell %+v", got, want)
		}
	})

	t.Run("mid-step it is picked where it is drawn", func(t *testing.T) {
		v.SetPhase(500, 1000)
		shift := v.entityShift(e)
		if shift.Y == 0 {
			t.Fatal("flat mode produced no displacement; the case cannot be shown")
		}
		got, ok := v.entityPickRect(e)
		if !ok {
			t.Fatal("no hit target")
		}
		want := screenRect{X: 2 * 32, Y: float64(2*32 + shift.Y), W: 32, H: 32}
		if got != want {
			t.Errorf("hit target %+v, want the drawn position %+v — the displacement is not "+
				"gated on displaced mode and the pick carries it", got, want)
		}
		v.SetPhase(0, 0)
	})
}

func TestTwoEntitiesOnOneCellSharePickRect(t *testing.T) {
	cell := image.Pt(1, 1)
	a := MapEntity{ID: 1, Cell: cell, Life: LifeAlive, HP: 10, MaxHP: 10}
	b := MapEntity{ID: 999, Cell: cell, Life: LifeDowned, Name: "x"}
	v := hitViewer(t, []MapEntity{a, b})

	ra, oka := v.entityPickRect(a)
	rb, okb := v.entityPickRect(b)
	if !oka || !okb {
		t.Fatal("one of two entities on one cell has no pick rectangle")
	}
	if ra != rb {
		t.Errorf("two entities on cell %v pick on %+v and %+v", cell, ra, rb)
	}
}
