package ui

import (
	"image"
	"reflect"
	"slices"
	"testing"
	"time"

	"againrom/pkg/render/camera"
	"againrom/pkg/render/terrain"
)

// commandFrozen is the one instant every camera step below is driven at, so
// elapsed time is zero and the water ticker cannot perturb anything.
var commandFrozen = time.Unix(1_700_000_000, 0)

// commandViewer is an 8x8 viewer in an 800x600 window, holding the two
// entities below. The camera is left where the clamp puts it — centred, at
// (-272, -172), since the world fills neither axis.
func commandViewer(t *testing.T) *Viewer {
	t.Helper()
	v, err := NewViewer("command", grid(8, 8), &terrain.Tileset{})
	if err != nil {
		t.Fatalf("NewViewer: %v", err)
	}
	hideBottomPanels(v)
	layoutViewport(v, 800, 600)
	if got := v.Camera().Zoom; got != 1 {
		t.Fatalf("fixture zoom = %v, want 1", got)
	}
	if got := (image.Point{X: int(v.Camera().X), Y: int(v.Camera().Y)}); got != (image.Point{X: -272, Y: -172}) {
		t.Fatalf("fixture camera at %v, want (-272,-172) — the clamp centres a world smaller than the view", got)
	}
	v.SetEntities(commandEntities())
	return v
}

// commandEntities is the snapshot the picks below run against: two entities on
// known cells, with SPARSE ids given in DESCENDING order, so nothing that
// answered with a slice position instead of an id could agree at either one.
func commandEntities() []MapEntity {
	return []MapEntity{
		{ID: 12, Cell: image.Pt(2, 2)},
		{ID: 5, Cell: image.Pt(5, 4)},
	}
}

const (
	unitAID, unitACol, unitARow = 12, 2, 2
	unitBID, unitBCol, unitBRow = 5, 5, 4
	emptyCol, emptyRow          = 0, 0 // in the extent, holding no entity
)

// cellPoint is the window position of the centre of cell (col, row), through
// the camera's own forward transform. At this fixture's camera it is
// (32*col + 288, 32*row + 188): cell (0,0) is at (288,188) and cell (7,7) at
// (512,412), both well inside the window and far from any edge margin.
func cellPoint(v *Viewer, col, row int) (int, int) {
	sx, sy := v.Camera().WorldToScreen(
		float64(col*camera.CellSize+camera.CellSize/2),
		float64(row*camera.CellSize+camera.CellSize/2))
	return int(sx), int(sy)
}

// cellSpan is the screen rectangle of the CLOSED cell range [c0,c1] x [r0,r1],
// corner to corner rather than centre to centre.
//
// IT EXISTS BECAUSE A MARQUEE IS JUDGED BY AREA (`AI-SELECT-122`): a plain
// rectangle takes a unit only where it covers strictly more than half of
// that unit's drawable rectangle, so a drag from one cell's CENTRE to
// another's covers a quarter of each end unit and takes neither. cellPoint
// is still the right point for a tap, which is a point and has no area.
func cellSpan(v *Viewer, c0, r0, c1, r1 int) (x0, y0, x1, y1 int) {
	ax, ay := v.Camera().WorldToScreen(float64(c0*camera.CellSize), float64(r0*camera.CellSize))
	bx, by := v.Camera().WorldToScreen(float64((c1+1)*camera.CellSize), float64((r1+1)*camera.CellSize))
	return int(ax), int(ay), int(bx), int(by)
}

// The two window positions that resolve OUTSIDE the 8x8 extent. Both are real
// positions inside the 800x600 window; the map occupies only [272,528) x
// [172,428) of it.
//
//	(0,0)     -> world (-272,-172) -> cell (-9,-6), off the top-left
//	(700,500) -> world ( 428, 328) -> cell (13,10), off the bottom-right
var outsideBefore = [2]int{0, 0}
var outsideAfter = [2]int{700, 500}

// tapAt taps at a window position, driven through the shipped shell: a press
// and its release in ONE frame, which is AC-2's own single-frame sequence and
// so travels zero pixels and is a tap by the slop.
//
// IT IS THE ORDERING GESTURE AND THE SELECTING ONE (`AI-INPUT-121`,
// `AI-CLICK-050`). Which of the two a tap performs is decided by the CURSOR
// it was made under, so there is one gesture here and not two, and the cases
// below that used to press the secondary button call this instead.
func tapAt(v *Viewer, x, y int) ([]order, bool) {
	return v.command(appInput{PrimaryPressed: true, PrimaryReleased: true, CursorX: x, CursorY: y})
}

// rightUpAt releases the secondary button at a window position, with no
// preceding move, so rightPanned is down and the release is `AI-INPUT-127`'s
// own CLICK: it cancels an armed mode, or deselects all when no mode is
// armed.
func rightUpAt(v *Viewer, x, y int) ([]order, bool) {
	return v.command(appInput{SecondaryReleased: true, CursorX: x, CursorY: y})
}

// selectUnit establishes a selection through the shipped path — a tap on that
// unit's own cell — rather than by writing the field, so no test below starts
// from a selection the front-end could not have reached.
func selectUnit(t *testing.T, v *Viewer, id uint32, col, row int) {
	t.Helper()
	x, y := cellPoint(v, col, row)
	if _, ok := tapAt(v, x, y); ok {
		t.Fatalf("the setup tap issued an order")
	}
	if !slices.Equal(v.sel, selection{id}) {
		t.Fatalf("the setup tap on cell (%d,%d) left selection %+v, want id %d", col, row, v.sel, id)
	}
}

func TestATapSelectsAndReplacesAndOnlyTheRightClickClears(t *testing.T) {
	v := commandViewer(t)

	ax, ay := cellPoint(v, unitACol, unitARow)
	if _, ok := tapAt(v, ax, ay); ok {
		t.Errorf("a tap under the `select` cursor issued an order")
	}
	if want := (selection{unitAID}); !slices.Equal(v.sel, want) {
		t.Fatalf("tap on unit A's cell: selection = %+v, want %+v", v.sel, want)
	}

	// Replace: a tap on the other unit's cell selects it INSTEAD, not as well.
	// The cursor over an actor with a selection standing is `select` (arm 4),
	// so this is still the selection routine and not an order.
	bx, by := cellPoint(v, unitBCol, unitBRow)
	if _, ok := tapAt(v, bx, by); ok {
		t.Errorf("a tap on the other unit issued an order; arm 4 gives `select`")
	}
	if want := (selection{unitBID}); !slices.Equal(v.sel, want) {
		t.Fatalf("tap on unit B's cell: selection = %+v, want %+v (replacing, not adding)", v.sel, want)
	}

	// Empty ground with a selection standing: an ORDER, and the selection is
	// untouched.
	ex, ey := cellPoint(v, emptyCol, emptyRow)
	ords, ok := tapAt(v, ex, ey)
	if !ok || len(ords) != 1 || ords[0].kind != orderKindMove {
		t.Fatalf("tap on an empty cell produced %+v (ok=%v), want one move order", ords, ok)
	}
	if want := (selection{unitBID}); !slices.Equal(v.sel, want) {
		t.Fatalf("tap on an empty cell: selection = %+v, want %+v -- ordering does not clear", v.sel, want)
	}

	// THE RIGHT CLICK IS THE DESELECT (`AI-INPUT-127`): a release with no
	// preceding move, with no mode armed, clears the whole selection and
	// issues nothing.
	if ords, ok := rightUpAt(v, ex, ey); ok || len(ords) != 0 {
		t.Errorf("the right click issued %+v (ok=%v), want none -- the right button never orders", ords, ok)
	}
	if len(v.sel) != 0 {
		t.Fatalf("after the right click: selection = %+v, want none", v.sel)
	}
}

func TestTapHitsTheLowerIdOnASharedCell(t *testing.T) {
	const col, row = 6, 6
	for _, tc := range []struct {
		name string
		ents []MapEntity
	}{
		{"higher id first", []MapEntity{{ID: 30, Cell: image.Pt(col, row)}, {ID: 7, Cell: image.Pt(col, row)}}},
		{"lower id first", []MapEntity{{ID: 7, Cell: image.Pt(col, row)}, {ID: 30, Cell: image.Pt(col, row)}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			v := commandViewer(t)
			v.SetEntities(tc.ents)

			x, y := cellPoint(v, col, row)
			tapAt(v, x, y)

			if want := (selection{7}); !slices.Equal(v.sel, want) {
				t.Errorf("selection = %+v, want %+v — the LOWER id is hit, in either slice order", v.sel, want)
			}
		})
	}
}

// TestALeftTapOrdersTheSelectedUnitAtTheResolvedCell is `AI-CLICK-050`'s move
// arm: a left tap with a unit selected, over empty ground inside the extent,
// issues exactly one order naming that unit and that cell; with no selection it
// issues none, because the click handler's own entry test sends a click made
// with nothing selected to the selection routine whatever the cursor is.
//
// THE UNIT-CELL CASE MOVED OUT OF THIS TEST. The cursor over an actor is
// `select` (arm 4), so that same gesture is now a selection and is asserted
// as one in TestATapSelectsAndReplacesAndOnlyTheRightClickClears.
func TestALeftTapOrdersTheSelectedUnitAtTheResolvedCell(t *testing.T) {
	t.Run("with a selection, over empty ground inside the extent", func(t *testing.T) {
		v := commandViewer(t)
		selectUnit(t, v, unitAID, unitACol, unitARow)

		x, y := cellPoint(v, emptyCol, emptyRow)
		ords, ok := tapAt(v, x, y)

		if !ok {
			t.Fatalf("no order issued")
		}
		if want := (order{kind: orderKindMove, entity: unitAID, x: emptyCol, y: emptyRow}); len(ords) != 1 || ords[0] != want {
			t.Errorf("orders = %+v, want exactly [%+v]", ords, want)
		}
		if want := (selection{unitAID}); !slices.Equal(v.sel, want) {
			t.Errorf("selection = %+v, want %+v — ordering does not move the selection", v.sel, want)
		}
	})

	t.Run("with no selection", func(t *testing.T) {
		v := commandViewer(t)
		x, y := cellPoint(v, emptyCol, emptyRow)

		if ords, ok := tapAt(v, x, y); ok {
			t.Errorf("orders %+v issued with nothing selected, want none", ords)
		}
		if len(v.sel) != 0 {
			t.Errorf("selection = %+v, want none", v.sel)
		}
	})
}

func TestOutsideTheExtentOrdersNothingAndKeepsTheSelection(t *testing.T) {
	for _, tc := range []struct {
		name string
		p    [2]int
	}{
		{"off the top-left", outsideBefore},
		{"off the bottom-right", outsideAfter},
	} {
		p := tc.p
		t.Run(tc.name, func(t *testing.T) {
			v := commandViewer(t)
			selectUnit(t, v, unitAID, unitACol, unitARow)

			if ords, ok := tapAt(v, p[0], p[1]); ok {
				t.Errorf("cursor (%d,%d): orders %+v issued outside the extent, want none", p[0], p[1], ords)
			}
			if want := (selection{unitAID}); !slices.Equal(v.sel, want) {
				t.Errorf("cursor (%d,%d): a refused order moved the selection to %+v, want %+v", p[0], p[1], v.sel, want)
			}
		})
	}
}

type outcome int

const (
	outcomeNothing outcome = iota
	outcomeSelect
	outcomeClear
	outcomeOrder
)

func (o outcome) String() string {
	return [...]string{"nothing", "select", "clear", "order"}[o]
}

// outcomesOf labels what one frame did, from the selection either side of it
// and the order it returned.
func outcomesOf(before, after selection, ok bool) []outcome {
	var got []outcome
	same := slices.Equal(before, after)
	if ok {
		got = append(got, outcomeOrder)
	}
	if !ok && len(after) != 0 && !same {
		got = append(got, outcomeSelect)
	}
	if !ok && len(after) == 0 && len(before) != 0 {
		got = append(got, outcomeClear)
	}
	if !ok && same {
		got = append(got, outcomeNothing)
	}
	return got
}

func TestEveryCellAndSelectionLandsInExactlyOneOutcome(t *testing.T) {
	cells := []struct {
		name   string
		x, y   int // resolved lazily against a viewer, below, when col >= 0
		col    int
		row    int
		inside bool
	}{
		{name: "hit", col: unitBCol, row: unitBRow, inside: true},
		{name: "miss", col: emptyCol, row: emptyRow, inside: true},
		{name: "outside", x: outsideAfter[0], y: outsideAfter[1], col: -1, row: -1},
	}

	for _, cell := range cells {
		for _, withSel := range []bool{true, false} {
			name := cell.name + "/"
			if withSel {
				name += "selected"
			} else {
				name += "none"
			}

			t.Run(name, func(t *testing.T) {
				v := commandViewer(t)
				if withSel {
					selectUnit(t, v, unitAID, unitACol, unitARow)
				}

				x, y := cell.x, cell.y
				if cell.col >= 0 {
					x, y = cellPoint(v, cell.col, cell.row)
				}

				before := v.sel
				_, ok := tapAt(v, x, y)

				want := expectedOutcome(cell.name, withSel)
				got := outcomesOf(before, v.sel, ok)
				if len(got) != 1 {
					t.Fatalf("frame landed in %v — want exactly one outcome (%v); selection %+v -> %+v, ordered %v",
						got, want, before, v.sel, ok)
				}
				if got[0] != want {
					t.Errorf("outcome = %v, want %v; selection %+v -> %+v, ordered %v",
						got[0], want, before, v.sel, ok)
				}
			})
		}
	}
}

func expectedOutcome(cell string, withSel bool) outcome {
	if cell == "hit" {
		return outcomeSelect
	}
	if cell == "miss" && withSel {
		return outcomeOrder
	}
	return outcomeNothing
}

// AC-2 (SC-2): the three sequences the contract names, driven through the
// SHIPPED dragIntent — every held tick goes through v.step, so the
// accumulator the release is judged against is the one the camera panned by,
// never a second delta of this test's own.
//
// The release position is what a tap resolves at, so each sequence releases
// inside unit A's own cell: a sequence judged the wrong way is visible as a
// selection that appeared or failed to.
func TestTheSlopSeparatesATapFromADrag(t *testing.T) {
	press := func(v *Viewer, x, y int) {
		v.step(Input{PrimaryDown: true, CursorX: x, CursorY: y}, commandFrozen)
		v.command(appInput{PrimaryPressed: true, CursorX: x, CursorY: y})
	}
	move := func(v *Viewer, x, y int) {
		v.step(Input{PrimaryDown: true, CursorX: x, CursorY: y}, commandFrozen)
		v.command(appInput{CursorX: x, CursorY: y})
	}
	release := func(v *Viewer, x, y int) {
		v.step(Input{CursorX: x, CursorY: y}, commandFrozen)
		v.command(appInput{PrimaryReleased: true, CursorX: x, CursorY: y})
	}

	// THE THRESHOLD IS THE MISSION'S OWN AND NO LONGER TapSlop: `AI-INPUT-121`
	// gives it as `screenW*10/640` -- 10, 12 and 16 pixels at the three shipped
	// screen widths -- and states the comparison as strict, "a rectangle
	// strictly beyond it goes to selection". Both boundary cases below are read
	// off `v.marqueeSlop()` rather than off a number written here, so they
	// follow the frame this fixture happens to compose at.
	t.Run("exactly the threshold is still a tap at the release position", func(t *testing.T) {
		v := commandViewer(t)
		x, y := cellPoint(v, unitACol, unitARow)
		slop := v.marqueeSlop()
		if slop < 2 {
			t.Fatalf("the fixture's marquee threshold is %d; this case needs room to move inside it", slop)
		}

		press(v, x, y)
		move(v, x+slop, y)
		release(v, x+slop, y)

		if got := v.dragMoved; got != slop {
			t.Fatalf("the gesture travelled %d screen pixels, want exactly %d — the fixture is not driving dragIntent", got, slop)
		}
		if want := (selection{unitAID}); !slices.Equal(v.sel, want) {
			t.Errorf("selection = %+v, want %+v — at the threshold it is still a click", v.sel, want)
		}
	})

	t.Run("one pixel beyond the threshold is not a tap", func(t *testing.T) {
		v := commandViewer(t)
		x, y := cellPoint(v, unitACol, unitARow)
		slop := v.marqueeSlop()

		press(v, x, y)
		move(v, x+slop+1, y)
		release(v, x+slop+1, y)

		if got := v.dragMoved; got != slop+1 {
			t.Fatalf("the gesture travelled %d screen pixels, want exactly %d", got, slop+1)
		}
		// The release still lands inside unit A's cell, so a gesture judged a
		// tap here WOULD have selected: this assertion discriminates. As a
		// marquee it is a horizontal line with no area, which qualifies nobody
		// and preserves the empty selection it started from (`AI-SELECT-122`).
		if len(v.sel) != 0 {
			t.Errorf("selection = %+v, want none — one pixel past the threshold is a marquee", v.sel)
		}
	})

	t.Run("a release with no press is nothing", func(t *testing.T) {
		v := commandViewer(t)
		x, y := cellPoint(v, unitACol, unitARow)

		release(v, x, y)

		if len(v.sel) != 0 {
			t.Errorf("selection = %+v, want none — a release nobody pressed selects nothing", v.sel)
		}
	})

	t.Run("a press and a release in one frame is a tap", func(t *testing.T) {
		// The anchor branch never runs for this sequence, so the press edge is the
		// only thing that can start the gesture.
		v := commandViewer(t)
		x, y := cellPoint(v, unitACol, unitARow)

		v.command(appInput{PrimaryPressed: true, PrimaryReleased: true, CursorX: x, CursorY: y})

		if want := (selection{unitAID}); !slices.Equal(v.sel, want) {
			t.Errorf("selection = %+v, want %+v", v.sel, want)
		}
	})
}

// TestTheOutlineAndTheReleaseAgreeOnOneThreshold is the regression witness
// for F1 (round 2): the outline shown mid-drag (overlay.go) and the order
// the release dispatches (command.go) used to read two different numbers,
// TapSlop (4) and v.marqueeSlop() (15 at this fixture's own frame width). A
// drag whose travel fell between them drew a marquee rectangle on screen,
// and its release still ordered every selected unit to the press point — a
// click under a rectangle promising a selection.
//
// EACH CASE DRIVES ONE GESTURE THROUGH BOTH HALVES OF THE SHIPPED PATH: the
// outline is read from marqueeScreenRects mid-drag, through the SAME v.step ->
// v.command sequence dragTo uses, and the order is read from the release that
// ends the same gesture. Neither half is asked what it would have done; both
// are read off what the code under test actually built.
//
// THE TRAVELS ARE COMPUTED FROM v.marqueeSlop() AND TapSlop, never written out
// as literals, so the cases still bracket the threshold if either constant or
// this fixture's own frame width ever changes.
func TestTheOutlineAndTheReleaseAgreeOnOneThreshold(t *testing.T) {
	v := boxViewer(t)
	slop := v.marqueeSlop()
	if slop < TapSlop+3 {
		t.Fatalf("fixture bug: marqueeSlop() is %d and TapSlop is %d — this case needs room "+
			"between them to bracket the threshold", slop, TapSlop)
	}

	for _, tc := range []struct {
		name    string
		travel  int
		outline bool
	}{
		{"well under TapSlop, never in question", TapSlop - 1, false},
		{"at TapSlop, the old dead band's own low end", TapSlop, false},
		{"the middle of the old dead band", (TapSlop + slop) / 2, false},
		{"the threshold itself, still a click (AI-INPUT-121's 'at or under')", slop, false},
		{"one pixel beyond, a marquee ('strictly beyond')", slop + 1, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			v := boxViewer(t)
			selectThree(t, v)
			ox, oy := cellPoint(v, 6, 6)

			v.step(Input{PrimaryDown: true, CursorX: ox, CursorY: oy}, commandFrozen)
			v.command(appInput{PrimaryPressed: true, CursorX: ox, CursorY: oy})
			v.step(Input{PrimaryDown: true, CursorX: ox + tc.travel, CursorY: oy}, commandFrozen)
			v.command(appInput{CursorX: ox + tc.travel, CursorY: oy})

			if got := v.dragMoved; got != tc.travel {
				t.Fatalf("travelled %d pixels, want exactly %d — the fixture is not driving dragIntent",
					got, tc.travel)
			}
			hasOutline := marqueeRects(v) != nil
			if hasOutline != tc.outline {
				t.Errorf("mid-drag at %d pixels of travel, outline present = %v, want %v",
					tc.travel, hasOutline, tc.outline)
			}

			v.step(Input{CursorX: ox + tc.travel, CursorY: oy}, commandFrozen)
			ords, ok := v.command(appInput{PrimaryReleased: true, CursorX: ox + tc.travel, CursorY: oy})

			if hasOutline {
				// A marquee: the release must box, and a box over empty
				// ground orders nothing (`AI-SELECT-122`).
				if ok || len(ords) != 0 {
					t.Fatalf("outline was shown, release still issued %+v (ok=%v), want none — this is F1",
						ords, ok)
				}
				return
			}
			// No outline: the release must be the click this story's own
			// contract dispatches for a selection over empty ground, one move
			// order per selected unit (`AI-INPUT-121`, `AI-SELECT-122`).
			if !ok || len(ords) != 3 {
				t.Fatalf("no outline was shown, release issued %+v (ok=%v), want 3 move orders — "+
					"the outline promised a click and the release disagreed", ords, ok)
			}
			for _, o := range ords {
				if o.kind != orderKindMove {
					t.Errorf("order %+v is not a move", o)
				}
				if o.x != 6 || o.y != 6 {
					t.Errorf("order %+v does not land on the press cell (6,6)", o)
				}
			}
		})
	}
}

func TestNoExportedMethodCarriesASelectionOrAnOrder(t *testing.T) {
	sealed := map[reflect.Type]string{
		reflect.TypeOf(selection(nil)): "selection",
		reflect.TypeOf(order{}):        "order",
	}
	vt := reflect.TypeOf((*Viewer)(nil))

	for i := 0; i < vt.NumMethod(); i++ {
		m := vt.Method(i)
		for j := 0; j < m.Type.NumIn(); j++ {
			if name, bad := sealed[m.Type.In(j)]; bad {
				t.Errorf("(*Viewer).%s takes a %s; the command machinery is package-internal (DD-4)", m.Name, name)
			}
		}
		for j := 0; j < m.Type.NumOut(); j++ {
			if name, bad := sealed[m.Type.Out(j)]; bad {
				t.Errorf("(*Viewer).%s returns a %s; the command machinery is package-internal (DD-4)", m.Name, name)
			}
		}
	}
}

// ---------------------------------------------------------------------------
// 0030 T3: the selection is a set, and a release fills it.
//
// SEPARATE CONTEXT, on the same rule as the file header: every expected cell is
// written out from the fixture's own geometry and every screen position comes
// from the CAMERA CONTRACT's forward transform (WorldToScreen, through
// cellPoint), never from a second call to ScreenToCell or ScreenToCellRange.
// ---------------------------------------------------------------------------

// boxViewer is a 12x12 viewer in an 800x600 window over a NON-IDENTITY camera:
// zoomed to 2 and sitting at (-8, 40), so neither the zoom nor either offset is
// the identity an identity fixture would hide. The world is 384x384 world
// pixels, narrower than the 400 the view covers at this zoom and taller than the
// 300 it covers, so the clamp centres x at -8 while y is the 40 it was set to.
//
// At this camera the forward transform is sx = (wx+8)*2, sy = (wy-40)*2, so cell
// (col,row)'s centre lands at (64col+48, 64row-48): cols 0..11 and rows 1..10
// are all on screen, and the screen band x < 16 resolves OUTSIDE the extent —
// which is where the off-map cases below come from without inventing a cursor
// that has left the window.
//
// Command mode is set directly. That the FLOW sets it on the transition storing
// the seam is flow_test.go's subject; what this file owes is which gesture a
// viewer in that mode resolves.
func boxViewer(t *testing.T) *Viewer {
	t.Helper()
	v, err := NewViewer("box", grid(12, 12), &terrain.Tileset{})
	if err != nil {
		t.Fatalf("NewViewer: %v", err)
	}
	hideBottomPanels(v)
	layoutViewport(v, 800, 600)
	v.commandMode = true
	v.Camera().SetZoom(2)
	v.Camera().Y = 40
	v.Camera().Clamp()

	if c := v.Camera(); c.Zoom != 2 || c.X != -8 || c.Y != 40 {
		t.Fatalf("fixture camera at (%v,%v) zoom %v, want (-8,40) at zoom 2 — every position below is "+
			"stated over a camera panned off the origin AND zoomed off 1", c.X, c.Y, c.Zoom)
	}
	v.SetEntities(boxEntities())
	return v
}

// boxEntities is the snapshot every release below runs against: four units on
// known cells, ids SPARSE and ASCENDING, which is the order the seam hands a
// snapshot over in. Three sit inside the band the cases drag; the fourth is far
// enough away that no case reaches it, so a walk that kept everything shows up
// as an extra id rather than as a coincidence.
func boxEntities() []MapEntity {
	return []MapEntity{
		{ID: 3, Cell: image.Pt(2, 2)},
		{ID: 8, Cell: image.Pt(4, 2)},
		{ID: 11, Cell: image.Pt(2, 4)},
		{ID: 20, Cell: image.Pt(9, 8)},
	}
}

// dragTo drives one left drag through the SHIPPED path: a press, one moving tick
// and the release, each frame going through v.step before v.command, so the
// latch, the press point and the accumulator are the ones dragIntent wrote and
// the release is judged against the travel that tick actually made.
func dragTo(v *Viewer, x0, y0, x1, y1 int) ([]order, bool) {
	v.step(Input{PrimaryDown: true, CursorX: x0, CursorY: y0}, commandFrozen)
	v.command(appInput{PrimaryPressed: true, CursorX: x0, CursorY: y0})
	v.step(Input{PrimaryDown: true, CursorX: x1, CursorY: y1}, commandFrozen)
	v.command(appInput{CursorX: x1, CursorY: y1})
	v.step(Input{CursorX: x1, CursorY: y1}, commandFrozen)
	return v.command(appInput{PrimaryReleased: true, CursorX: x1, CursorY: y1})
}

// dragOutAndBack is the one release whose two points COINCIDE and which is still
// a drag: the accumulator is a path length, not a displacement, so a gesture
// that wanders out and back has travelled twice the excursion and is judged a
// drag while its rectangle is a single point. It is the only way that case is
// reachable at all, and the contract asks for it by name.
func dragOutAndBack(v *Viewer, x, y int) {
	v.step(Input{PrimaryDown: true, CursorX: x, CursorY: y}, commandFrozen)
	v.command(appInput{PrimaryPressed: true, CursorX: x, CursorY: y})
	v.step(Input{PrimaryDown: true, CursorX: x + 3*TapSlop, CursorY: y}, commandFrozen)
	v.command(appInput{CursorX: x + 3*TapSlop, CursorY: y})
	v.step(Input{PrimaryDown: true, CursorX: x, CursorY: y}, commandFrozen)
	v.command(appInput{CursorX: x, CursorY: y})
	v.step(Input{CursorX: x, CursorY: y}, commandFrozen)
	v.command(appInput{PrimaryReleased: true, CursorX: x, CursorY: y})
}

// selectThree establishes the prior selection every case below replaces: the
// box over cells (2,2)..(4,4), which covers three of the fixture's four units.
//
// IT SPANS THE CELLS CORNER TO CORNER, not centre to centre. A plain
// rectangle takes a unit only where it covers strictly more than half of
// that unit's drawable rectangle (`AI-SELECT-122`), and a centre-to-centre
// drag covers a quarter of each end unit.
func selectThree(t *testing.T, v *Viewer) {
	t.Helper()
	x0, y0, x1, y1 := cellSpan(v, 2, 2, 4, 4)
	if _, ok := dragTo(v, x0, y0, x1, y1); ok {
		t.Fatalf("the setup release issued an order")
	}
	if want := (selection{3, 8, 11}); !slices.Equal(v.sel, want) {
		t.Fatalf("setup: selection %+v, want %+v", v.sel, want)
	}
}

// TestAPlainBoxTakesEveryUnitItCoversByMoreThanHalf is `AI-SELECT-122`'s
// rectangle form: a release past the slop takes every owned unit whose
// drawable rectangle the rectangle covers by STRICTLY more than half,
// answers the same whichever pair of corners it was dragged from, and
// PRESERVES the old selection when none qualifies.
//
// IT REPLACES TestABoxReplacesTheSelectionWithTheUnitsItCovers, which asserted
// the two clauses the decoded form refutes: that a box catches a unit its line
// merely meets, and that a box catching nothing clears. A zero-width rectangle
// covers no area at all, so it now qualifies nobody and preserves; the three
// cases that asserted otherwise are restated below as preserving cases, which
// is the same fixture read against the decoded rule.
//
// EVERY CASE STARTS FROM A PRIOR SELECTION OF THREE, so "replacing" is asserted
// against a set with more than one member in it — a prior selection of one could
// not tell replacement from an addition that happened to overwrite.
func TestAPlainBoxTakesEveryUnitItCoversByMoreThanHalf(t *testing.T) {
	for _, tc := range []struct {
		name               string
		c0x, c0y, c1x, c1y int
		want               selection
	}{
		{"over three of the four units", 2, 2, 4, 4, selection{3, 8, 11}},
		{"over one unit alone", 4, 2, 4, 2, selection{8}},
		{"over empty ground preserves", 6, 6, 7, 7, selection{3, 8, 11}},
		{"corner-swapped on x", 4, 2, 2, 4, selection{3, 8, 11}},
		{"corner-swapped on y", 2, 4, 4, 2, selection{3, 8, 11}},
		{"corner-swapped on both", 4, 4, 2, 2, selection{3, 8, 11}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			v := boxViewer(t)
			selectThree(t, v)

			ax, ay, bx, by := cellSpan(v, min(tc.c0x, tc.c1x), min(tc.c0y, tc.c1y),
				max(tc.c0x, tc.c1x), max(tc.c0y, tc.c1y))
			// The corner-swapped cases drag the SAME rectangle from the other
			// pair of corners, which is what orientation independence means.
			if tc.c1x < tc.c0x {
				ax, bx = bx, ax
			}
			if tc.c1y < tc.c0y {
				ay, by = by, ay
			}
			if _, ok := dragTo(v, ax, ay, bx, by); ok {
				t.Errorf("the release issued an order; a marquee never orders (`AI-SELECT-122`)")
			}
			if !slices.Equal(v.sel, tc.want) {
				t.Errorf("selection = %+v, want %+v", v.sel, tc.want)
			}
		})
	}

	// A rectangle with NO AREA takes nothing, whatever stands under its line.
	// `AI-SELECT-122`'s test is a coverage fraction of the unit's own
	// rectangle, and a line covers none of it.
	for _, tc := range []struct {
		name               string
		c0x, c0y, c1x, c1y int
	}{
		{"zero width: the line down cells (2,2)..(2,4)", 2, 2, 2, 4},
		{"zero height: the line across cells (2,2)..(4,2)", 2, 2, 4, 2},
	} {
		t.Run(tc.name+" takes nothing", func(t *testing.T) {
			v := boxViewer(t)
			selectThree(t, v)
			ax, ay, _, _ := cellSpan(v, tc.c0x, tc.c0y, tc.c1x, tc.c1y)
			_, _, bx, by := cellSpan(v, tc.c0x, tc.c0y, tc.c1x, tc.c1y)
			if tc.c0x == tc.c1x {
				bx = ax
			} else {
				by = ay
			}
			dragTo(v, ax, ay, bx, by)
			if want := (selection{3, 8, 11}); !slices.Equal(v.sel, want) {
				t.Errorf("selection = %+v, want %+v — a rectangle with no area qualifies nobody", v.sel, want)
			}
		})
	}

	// The point rectangle, which no two-position drag can reach: press, wander
	// out past the slop, come back, release where it began. It has no area
	// either, so it preserves.
	t.Run("both points coincide: no area, so the selection stands", func(t *testing.T) {
		v := boxViewer(t)
		selectThree(t, v)
		x, y := cellPoint(v, 4, 2)
		dragOutAndBack(v, x, y)
		if want := (selection{3, 8, 11}); !slices.Equal(v.sel, want) {
			t.Errorf("selection = %+v, want %+v", v.sel, want)
		}
	})

	// A release whose rectangle met no cell of the grid at all preserves, on
	// the same rule as one that met cells and covered nothing.
	t.Run("a release wholly outside the extent preserves", func(t *testing.T) {
		v := boxViewer(t)
		selectThree(t, v)
		dragTo(v, 2, 300, 10, 300) // x < 16 resolves to a negative column here
		if want := (selection{3, 8, 11}); !slices.Equal(v.sel, want) {
			t.Errorf("selection = %+v, want %+v", v.sel, want)
		}
	})

	// NON-VACUITY FOR THE PRESERVING CASES: the very same fixture, with a
	// rectangle that DOES cover a unit by more than half, replaces. Without
	// this every "preserves" above would also pass against a marquee that did
	// nothing at all.
	t.Run("non-vacuity: a covering rectangle still replaces", func(t *testing.T) {
		v := boxViewer(t)
		selectThree(t, v)
		ax, ay, bx, by := cellSpan(v, 9, 8, 9, 8)
		dragTo(v, ax, ay, bx, by)
		if want := (selection{20}); !slices.Equal(v.sel, want) {
			t.Errorf("selection = %+v, want %+v", v.sel, want)
		}
	})
}

// TestATapOverANonIdentityCameraYieldsASetOfOne — 0030 SC-3 (AC-3): the
// shipped tap outcomes, restated over the set and over a camera that is
// neither at the origin nor at native zoom — and the tie is still the
// lowest id, now as a set of exactly one member.
func TestATapOverANonIdentityCameraYieldsASetOfOne(t *testing.T) {
	t.Run("a hit selects that one, replacing a set of three", func(t *testing.T) {
		v := boxViewer(t)
		selectThree(t, v)

		tx, ty := cellPoint(v, 9, 8)
		if _, ok := tapAt(v, tx, ty); ok {
			t.Errorf("a tap issued an order")
		}
		if want := (selection{20}); !slices.Equal(v.sel, want) {
			t.Errorf("selection = %+v, want %+v — a tap replaces the whole set with one", v.sel, want)
		}
	})

	// A TAP ON AN EMPTY CELL ORDERS AND KEEPS THE SET: with a selection
	// standing, empty ground puts up `move` and the click is the move arm, so
	// the selection routine is not reached.
	t.Run("an empty cell orders and keeps the set", func(t *testing.T) {
		v := boxViewer(t)
		selectThree(t, v)
		x, y := cellPoint(v, 6, 6)
		ords, ok := tapAt(v, x, y)
		if !ok || len(ords) != 3 {
			t.Fatalf("the tap produced %+v (ok=%v), want three move orders", ords, ok)
		}
		if want := (selection{3, 8, 11}); !slices.Equal(v.sel, want) {
			t.Errorf("selection = %+v, want %+v", v.sel, want)
		}
	})

	t.Run("a point outside the extent orders nothing and keeps the set", func(t *testing.T) {
		v := boxViewer(t)
		selectThree(t, v)
		if ords, ok := tapAt(v, 8, 300); ok {
			t.Errorf("the tap issued %+v, want none", ords)
		}
		if want := (selection{3, 8, 11}); !slices.Equal(v.sel, want) {
			t.Errorf("selection = %+v, want %+v", v.sel, want)
		}
	})

	t.Run("two units on one cell: the lower id, as a set of one", func(t *testing.T) {
		for _, tc := range []struct {
			name string
			ents []MapEntity
		}{
			{"higher id first", []MapEntity{{ID: 30, Cell: image.Pt(5, 5)}, {ID: 7, Cell: image.Pt(5, 5)}}},
			{"lower id first", []MapEntity{{ID: 7, Cell: image.Pt(5, 5)}, {ID: 30, Cell: image.Pt(5, 5)}}},
		} {
			t.Run(tc.name, func(t *testing.T) {
				v := boxViewer(t)
				v.SetEntities(tc.ents)
				x, y := cellPoint(v, 5, 5)
				tapAt(v, x, y)
				if want := (selection{7}); !slices.Equal(v.sel, want) {
					t.Errorf("selection = %+v, want %+v", v.sel, want)
				}
			})
		}
	})
}

// dragToShift is dragTo with the modifier held for the whole gesture, so the
// release is a SHIFT RECTANGLE (`AI-SELECT-122`: it toggles every owned
// qualifier, under the old-summary gate).
//
// IT USED TO LATCH A PAN. The modifier is read through v.step, which is
// where the latch is written, so this drives it exactly as a played frame
// does.
func dragToShift(v *Viewer, x0, y0, x1, y1 int) ([]order, bool) {
	v.step(Input{PrimaryDown: true, Shift: true, CursorX: x0, CursorY: y0}, commandFrozen)
	v.command(appInput{PrimaryPressed: true, CursorX: x0, CursorY: y0})
	v.step(Input{PrimaryDown: true, Shift: true, CursorX: x1, CursorY: y1}, commandFrozen)
	v.command(appInput{CursorX: x1, CursorY: y1})
	v.step(Input{Shift: true, CursorX: x1, CursorY: y1}, commandFrozen)
	return v.command(appInput{PrimaryReleased: true, CursorX: x1, CursorY: y1})
}

// orderedIDs is the ids one frame's orders named, in the order they were
// emitted, and the one cell they all named. It fails the test if two orders name
// different cells, which is C-3's "every ordered unit goes to the SAME cell"
// read off the emission rather than assumed of it.
func orderedIDs(t *testing.T, ords []order) ([]uint32, image.Point) {
	t.Helper()
	var ids []uint32
	var cell image.Point
	for i, o := range ords {
		if i == 0 {
			cell = image.Pt(o.x, o.y)
		} else if got := image.Pt(o.x, o.y); got != cell {
			t.Fatalf("order %d names cell %v while order 0 names %v — one press, one cell", i, got, cell)
		}
		ids = append(ids, o.entity)
	}
	return ids, cell
}

func TestALeftTapOrdersEveryPresentMemberAscending(t *testing.T) {
	descending := []MapEntity{
		{ID: 20, Cell: image.Pt(9, 8)},
		{ID: 11, Cell: image.Pt(2, 4)},
		{ID: 8, Cell: image.Pt(4, 2)},
		{ID: 3, Cell: image.Pt(2, 2)},
	}

	t.Run("three selected, three orders, ascending, all on one cell", func(t *testing.T) {
		for _, tc := range []struct {
			name string
			ents []MapEntity
		}{
			{"snapshot ascending", boxEntities()},
			{"snapshot descending", descending},
		} {
			t.Run(tc.name, func(t *testing.T) {
				v := boxViewer(t)
				selectThree(t, v)
				v.SetEntities(tc.ents)

				ox, oy := cellPoint(v, 6, 6)
				ords, ok := tapAt(v, ox, oy)
				if !ok {
					t.Fatalf("no orders issued for a selection of three")
				}
				ids, cell := orderedIDs(t, ords)
				if want := []uint32{3, 8, 11}; !slices.Equal(ids, want) {
					t.Errorf("orders named %v, want %v in that order", ids, want)
				}
				if want := image.Pt(6, 6); cell != want {
					t.Errorf("the orders name cell %v, want %v", cell, want)
				}
				if want := (selection{3, 8, 11}); !slices.Equal(v.sel, want) {
					t.Errorf("ordering moved the selection to %+v, want %+v", v.sel, want)
				}
			})
		}
	})

	t.Run("an absent member is skipped, not dropped", func(t *testing.T) {
		v := boxViewer(t)
		selectThree(t, v)
		v.SetEntities([]MapEntity{
			{ID: 8, Cell: image.Pt(4, 2)},
			{ID: 11, Cell: image.Pt(2, 4)},
			{ID: 20, Cell: image.Pt(9, 8)},
		})

		ox, oy := cellPoint(v, 6, 6)
		ords, ok := tapAt(v, ox, oy)
		if !ok {
			t.Fatalf("no orders issued while two of the three members are present")
		}
		ids, _ := orderedIDs(t, ords)
		if want := []uint32{8, 11}; !slices.Equal(ids, want) {
			t.Errorf("orders named %v, want %v — the absent id is skipped", ids, want)
		}
		if want := (selection{3, 8, 11}); !slices.Equal(v.sel, want) {
			t.Errorf("the selection is now %+v, want %+v — a tick does not prune the set", v.sel, want)
		}
	})

	for _, tc := range []struct {
		name string
		run  func(t *testing.T, v *Viewer) ([]order, bool)
	}{
		{"nothing selected", func(t *testing.T, v *Viewer) ([]order, bool) {
			ox, oy := cellPoint(v, 6, 6)
			return tapAt(v, ox, oy)
		}},
		{"outside the extent", func(t *testing.T, v *Viewer) ([]order, bool) {
			selectThree(t, v)
			return tapAt(v, 8, 300)
		}},
		{"every selected member gone from the snapshot", func(t *testing.T, v *Viewer) ([]order, bool) {
			selectThree(t, v)
			v.SetEntities([]MapEntity{{ID: 99, Cell: image.Pt(6, 6)}})
			ox, oy := cellPoint(v, 6, 6)
			return tapAt(v, ox, oy)
		}},
		{"the press edge alone, with no release", func(t *testing.T, v *Viewer) ([]order, bool) {
			// The left button ACTS ON THE RELEASE (`AI-INPUT-121`), so a
			// press that has not come up orders nothing whatever it is over.
			selectThree(t, v)
			ox, oy := cellPoint(v, 6, 6)
			v.step(Input{PrimaryDown: true, CursorX: ox, CursorY: oy}, commandFrozen)
			return v.command(appInput{PrimaryPressed: true, CursorX: ox, CursorY: oy})
		}},
		{"a right click, which never orders", func(t *testing.T, v *Viewer) ([]order, bool) {
			selectThree(t, v)
			ox, oy := cellPoint(v, 6, 6)
			return rightUpAt(v, ox, oy)
		}},
	} {
		t.Run("no order: "+tc.name, func(t *testing.T) {
			v := boxViewer(t)
			ords, ok := tc.run(t, v)
			if ok || len(ords) != 0 {
				t.Errorf("orders %+v issued (ok=%v), want none", ords, ok)
			}
		})
	}
}

func TestNoOrderEscapesAPressThatNeverReleases(t *testing.T) {
	v := boxViewer(t)
	selectThree(t, v)

	ox, oy := cellPoint(v, 6, 6)

	v.step(Input{PrimaryDown: true, CursorX: ox, CursorY: oy}, commandFrozen)
	if ords, ok := v.command(appInput{PrimaryPressed: true, CursorX: ox, CursorY: oy}); ok {
		t.Fatalf("the press frame issued %+v, want none", ords)
	}
	for i := 1; i <= 6; i++ {
		v.step(Input{PrimaryDown: true, CursorX: ox, CursorY: oy}, commandFrozen)
		if ords, ok := v.command(appInput{CursorX: ox, CursorY: oy}); ok {
			t.Fatalf("frame %d of the held press issued %+v, want none", i, ords)
		}
	}
	if !slices.Equal(v.sel, selection{3, 8, 11}) {
		t.Fatalf("the held press moved the selection to %+v", v.sel)
	}

	// The release, at the same point and so a tap by the slop: it acts, once,
	// for each of the three.
	v.step(Input{CursorX: ox, CursorY: oy}, commandFrozen)
	ords, ok := v.command(appInput{PrimaryReleased: true, CursorX: ox, CursorY: oy})
	if !ok || len(ords) != 3 {
		t.Errorf("the release issued %+v (ok=%v), want three move orders", ords, ok)
	}
}

func TestEveryEdgeLatchAndSelectionLandsInExactlyOneOutcome(t *testing.T) {
	// Each target is a cell corner pair for the two rectangle edges and a
	// single point for the tap and the right click.
	targets := []struct {
		name               string
		c0x, c0y, c1x, c1y int
		outsideX, outsideY int
		outside            bool
	}{
		{name: "hit", c0x: 2, c0y: 2, c1x: 4, c1y: 2},
		{name: "miss", c0x: 6, c0y: 6, c1x: 7, c1y: 7},
		// BOTH ENDS OUTSIDE, and the travel vertical so that it is past the
		// marquee threshold while the release point stays off the drawn ground.
		// An 8-pixel horizontal pair used to serve here; under the decoded
		// threshold (`AI-INPUT-121`, 16 pixels on this frame) that is a click,
		// and its release landed at x=16, which resolves to column 0 and is
		// INSIDE.
		{name: "outside", outsideX: 2, outsideY: 300, outside: true},
	}

	for _, edge := range []string{"tap", "box", "shiftbox", "rightclick"} {
		for _, target := range targets {
			for _, prior := range []int{0, 1, 3} {
				name := edge + "/" + target.name + "/" + map[int]string{0: "none", 1: "one", 3: "many"}[prior]
				t.Run(name, func(t *testing.T) {
					v := boxViewer(t)
					switch prior {
					case 1:
						x, y := cellPoint(v, 9, 8)
						tapAt(v, x, y)
						if want := (selection{20}); !slices.Equal(v.sel, want) {
							t.Fatalf("setup: selection %+v, want %+v", v.sel, want)
						}
					case 3:
						selectThree(t, v)
					}

					var ax, ay, bx, by int
					if target.outside {
						ax, ay = target.outsideX, target.outsideY
						bx, by = target.outsideX, target.outsideY+80
					} else {
						ax, ay, bx, by = cellSpan(v, target.c0x, target.c0y, target.c1x, target.c1y)
					}
					// A tap and a right click are points, and the point they
					// name is the target's own first cell.
					px, py := ax, ay
					if !target.outside {
						px, py = cellPoint(v, target.c0x, target.c0y)
					}

					before := append(selection(nil), v.sel...)
					var ok bool
					switch edge {
					case "tap":
						_, ok = tapAt(v, px, py)
					case "box":
						_, ok = dragTo(v, ax, ay, bx, by)
					case "shiftbox":
						_, ok = dragToShift(v, ax, ay, bx, by)
					case "rightclick":
						_, ok = rightUpAt(v, px, py)
					}

					want := expectedGroupOutcome(edge, target.name, prior)
					got := outcomesOf(before, v.sel, ok)
					if len(got) != 1 {
						t.Fatalf("frame landed in %v — want exactly one outcome (%v); selection %+v -> %+v, "+
							"ordered %v", got, want, before, v.sel, ok)
					}
					if got[0] != want {
						t.Errorf("outcome = %v, want %v; selection %+v -> %+v, ordered %v",
							got[0], want, before, v.sel, ok)
					}
				})
			}
		}
	}
}

func expectedGroupOutcome(edge, target string, prior int) outcome {
	switch edge {
	case "rightclick":
		if prior > 0 {
			return outcomeClear
		}
		return outcomeNothing
	case "tap":
		if target == "hit" {
			return outcomeSelect
		}
		if target == "miss" && prior > 0 {
			return outcomeOrder
		}
		return outcomeNothing
	default: // box, shiftbox
		if target == "hit" {
			return outcomeSelect
		}
		return outcomeNothing
	}
}
