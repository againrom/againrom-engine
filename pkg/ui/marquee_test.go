package ui

import (
	"image"
	"testing"
	"time"
)

// marqueeFrozen is the one instant every camera step below is driven at, so
// elapsed time is zero and the water ticker cannot perturb anything.
var marqueeFrozen = time.Unix(1_700_000_000, 0)

// commandViewerForMarquee is stepViewer's world — far larger than the view, so
// no clamp can mask anything — with command mode set, which is the state the
// front-end's map screen puts a viewer in.
func commandViewerForMarquee(t *testing.T) *Viewer {
	t.Helper()
	v := stepViewer(t)
	v.commandMode = true
	return v
}

// marqueeRects is the outline's rectangles as this file reads them: the rects of
// the pass carrying the marquee colour, nil when no such pass was built.
//
// Reading through overlayPasses rather than a private accessor is deliberate —
// the pass slice is the draw contract, so every assertion below holds for
// exactly what Draw walks.
func marqueeRects(v *Viewer) []screenRect {
	for _, p := range v.overlayPasses() {
		if p.Color == MarqueeColor {
			return p.Rects
		}
	}
	return nil
}

// press, move and release are one frame each of a left gesture, driven through
// the SHIPPED camera step so the latch, the press point and the accumulator are
// the ones dragIntent wrote.
func marqueePress(v *Viewer, shift bool, x, y int) {
	v.step(Input{PrimaryDown: true, Shift: shift, CursorX: x, CursorY: y}, marqueeFrozen)
	v.command(appInput{PrimaryPressed: true, CursorX: x, CursorY: y})
}

func marqueeMove(v *Viewer, shift bool, x, y int) {
	v.step(Input{PrimaryDown: true, Shift: shift, CursorX: x, CursorY: y}, marqueeFrozen)
	v.command(appInput{CursorX: x, CursorY: y})
}

func marqueeRelease(v *Viewer, shift bool, x, y int) {
	v.step(Input{Shift: shift, CursorX: x, CursorY: y}, marqueeFrozen)
	v.command(appInput{PrimaryReleased: true, CursorX: x, CursorY: y})
}

func wantFrame(ax, ay, bx, by int) []screenRect {
	x0, x1 := float64(ax), float64(bx)
	if x1 < x0 {
		x0, x1 = x1, x0
	}
	y0, y1 := float64(ay), float64(by)
	if y1 < y0 {
		y0, y1 = y1, y0
	}
	w, h := x1-x0, y1-y0
	t := float64(MarqueeThickness)
	return []screenRect{
		{X: x0, Y: y0, W: w, H: t},
		{X: x0, Y: y1 - t, W: w, H: t},
		{X: x0, Y: y0, W: t, H: h},
		{X: x1 - t, Y: y0, W: t, H: h},
	}
}

// TestTheOutlineRunsFromTheFrameThePressPassesTheSlop — 0030 SC-7 (AC-7):
// no pass while the press is at or under the slop, one on the frame it
// passes it and on every frame after, and none once the button is released.
//
// THE SLOP IS v.marqueeSlop(), NOT TapSlop (F1, round 2). The outline and the
// release must read one number; command.go:1506 reads v.marqueeSlop() and this
// file reads it too, rather than writing out a second number that could drift
// from it again.
//
// The crossing is walked ONE PIXEL AT A TIME, so the frame the outline appears
// on is pinned rather than bracketed: an implementation that started drawing a
// pixel early or a pixel late fails a specific row here.
func TestTheOutlineRunsFromTheFrameThePressPassesTheSlop(t *testing.T) {
	v := commandViewerForMarquee(t)
	slop := v.marqueeSlop()

	const ax, ay = 400, 300
	marqueePress(v, false, ax, ay)
	if got := marqueeRects(v); got != nil {
		t.Fatalf("the anchor frame built %+v, want no outline — it has travelled nothing", got)
	}

	for d := 1; d <= slop; d++ {
		marqueeMove(v, false, ax+d, ay)
		if got := v.dragMoved; got != d {
			t.Fatalf("after %d pixels the accumulator reads %d — the fixture is not driving dragIntent", d, got)
		}
		if got := marqueeRects(v); got != nil {
			t.Errorf("at %d pixels of travel, at or under marqueeSlop() of %d, the frame built %+v, want no outline",
				d, slop, got)
		}
	}

	for _, d := range []int{slop + 1, slop + 2, slop + 40} {
		marqueeMove(v, false, ax+d, ay)
		got := marqueeRects(v)
		if got == nil {
			t.Fatalf("at %d pixels of travel the frame built no outline, want one", d)
		}
		assertScreenRects(t, got, wantFrame(ax, ay, ax+d, ay))
	}

	marqueeRelease(v, false, ax+slop+40, ay)
	if got := marqueeRects(v); got != nil {
		t.Errorf("after the release the frame still builds %+v, want no outline", got)
	}
}

// TestTheOutlineIsAbsentForEveryOtherPress — 0030 SC-7 (AC-7): the presses
// that draw none — one under the slop, and one on a viewer the front-end
// never put a world under.
//
// THE SHIFT CASE MOVED TO THE OTHER SIDE OF THIS TEST. Until then a drag
// begun with Shift was a PAN and drew no outline; the mission map's left
// button has no pan under any modifier (`AI-INPUT-121`), and Shift now
// selects which selection form the release performs (`AI-SELECT-122`). A
// Shift drag therefore draws the same outline a plain one does, which is
// asserted below rather than left as an absence nobody re-measured.
//
// Each runs several frames and then releases, and the outline is read on every
// one of them, so "absent" is a statement about the whole gesture rather than
// about the frame the test happened to look at.
func TestTheOutlineIsAbsentForEveryOtherPress(t *testing.T) {
	for _, tc := range []struct {
		name        string
		commandMode bool
		shift       bool
		steps       [][2]int
	}{
		{"a press that never passes the slop", true, false,
			[][2]int{{400, 300}, {401, 300}, {401, 301}, {400, 301}}},
		{"the standalone viewer, with no world under it", false, false,
			[][2]int{{400, 300}, {450, 320}, {520, 400}, {480, 360}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			v := stepViewer(t)
			v.commandMode = tc.commandMode

			marqueePress(v, tc.shift, tc.steps[0][0], tc.steps[0][1])
			for _, p := range tc.steps[1:] {
				marqueeMove(v, tc.shift, p[0], p[1])
				if got := marqueeRects(v); got != nil {
					t.Fatalf("at cursor %v the frame built %+v, want no outline", p, got)
				}
			}
			last := tc.steps[len(tc.steps)-1]
			marqueeRelease(v, tc.shift, last[0], last[1])
			if got := marqueeRects(v); got != nil {
				t.Errorf("after the release the frame builds %+v, want no outline", got)
			}
		})
	}

	// The control: the same fixture, dragged the same distance with no modifier
	// and with a world under it, DOES build one — so every "want none" above is
	// an observation of the outline's absence rather than of a fixture that
	// never draws it.
	v := commandViewerForMarquee(t)
	marqueePress(v, false, 400, 300)
	marqueeMove(v, false, 450, 320)
	if got := marqueeRects(v); got == nil {
		t.Fatalf("the probe does not discriminate: a plain drag on a map screen built no outline")
	}
}

// TestAShiftDragDrawsTheSameOutlineAPlainOneDoes is the other half of the
// case removed above: with Shift held for the whole gesture the map still
// marquees, because the left button's contract has no pan in it at all
// (`AI-INPUT-121`) and Shift reaches only the selection form the release
// performs (`AI-SELECT-122`).
//
// IT IS ASSERTED AGAINST THE PLAIN GESTURE'S OWN OUTLINE rather than against a
// rectangle written out here, so a change to how the outline is built moves both
// sides together and this stays a statement about the modifier alone.
func TestAShiftDragDrawsTheSameOutlineAPlainOneDoes(t *testing.T) {
	steps := [][2]int{{400, 300}, {450, 320}, {520, 400}}
	run := func(shift bool) []screenRect {
		v := stepViewer(t)
		v.commandMode = true
		marqueePress(v, shift, steps[0][0], steps[0][1])
		for _, p := range steps[1:] {
			marqueeMove(v, shift, p[0], p[1])
		}
		return marqueeRects(v)
	}
	plain, shifted := run(false), run(true)
	if plain == nil {
		t.Fatal("the plain drag built no outline; the comparison below would prove nothing")
	}
	assertScreenRects(t, shifted, plain)
}

// TestTheOutlineIsFourStripsLeavingItsInteriorUncovered — 0030 SC-7
// (AC-7): four strips, in every drag orientation, and not one pixel of the
// rectangle's own interior covered by any of them.
//
// The interior is the rectangle inset by MarqueeThickness on all four sides, and
// it is tested against each strip with the same half-open overlap rule the rest
// of this package's culling uses. A filled rectangle — the shape that would hide
// the very units the drag is choosing — fails this on its first strip.
func TestTheOutlineIsFourStripsLeavingItsInteriorUncovered(t *testing.T) {
	const ax, ay, bx, by = 300, 200, 500, 340

	forward := wantFrame(ax, ay, bx, by)

	for _, tc := range []struct {
		name           string
		x0, y0, x1, y1 int
	}{
		{"top-left to bottom-right", ax, ay, bx, by},
		{"bottom-right to top-left", bx, by, ax, ay},
		{"top-right to bottom-left", bx, ay, ax, by},
		{"bottom-left to top-right", ax, by, bx, ay},
	} {
		t.Run(tc.name, func(t *testing.T) {
			v := commandViewerForMarquee(t)
			marqueePress(v, false, tc.x0, tc.y0)
			marqueeMove(v, false, tc.x1, tc.y1)

			got := marqueeRects(v)
			if len(got) != 4 {
				t.Fatalf("the outline is %d strips (%+v), want 4", len(got), got)
			}
			assertScreenRects(t, got, forward)

			th := float64(MarqueeThickness)
			interior := screenRect{
				X: float64(ax) + th,
				Y: float64(ay) + th,
				W: float64(bx-ax) - 2*th,
				H: float64(by-ay) - 2*th,
			}
			if interior.W <= 0 || interior.H <= 0 {
				t.Fatalf("the fixture rectangle %+v has no interior to leave uncovered", interior)
			}
			for i, s := range got {
				if rectsOverlap(s, interior) {
					t.Errorf("strip %d %+v covers part of the interior %+v — the outline is hollow",
						i, s, interior)
				}
			}
		})
	}
}

func TestTheOutlineIsAppendedAfterEveryOtherPass(t *testing.T) {
	v := commandViewerForMarquee(t)
	// Two diagnostic cells inside this fixture's view — the camera sits at
	// (2000,2000) over an 800x600 window, so cells 62..87 by 62..81 are on
	// screen — giving the frame a pass the outline has to be appended after.
	v.SetUnits(true, []image.Point{{X: 65, Y: 65}, {X: 70, Y: 70}})

	marqueePress(v, false, 400, 300)
	before := len(v.overlayPasses())
	if before == 0 {
		t.Fatalf("the fixture builds no passes at all, so a position below says nothing")
	}

	marqueeMove(v, false, 500, 380)
	passes := v.overlayPasses()
	if len(passes) != before+1 {
		t.Fatalf("mid-drag the frame holds %d passes, want the %d it held plus the outline",
			len(passes), before)
	}
	if last := passes[len(passes)-1]; last.Color != MarqueeColor {
		t.Errorf("the last pass carries colour %+v, want the outline's %+v", last.Color, MarqueeColor)
	}
	for i := 0; i < before; i++ {
		if passes[i].Color == MarqueeColor {
			t.Errorf("pass %d carries the outline's colour, want it appended after every existing pass", i)
		}
	}
}
