package menu_test

import (
	"fmt"
	"image"
	"image/color"
	"sync"
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/render/menu"
)

// ---------------------------------------------------------------------------
// The contract, transcribed from docs/0010-main-menu/spec.md
// ---------------------------------------------------------------------------

var stFrameRect = image.Rect(0, 0, 640, 480)

// The graphics/mainmenu/ subtree of main.res — spec "Asset set" —
// addressed under main.res's own identity segment.
const stPrefix = "main/graphics/mainmenu/"

// stRect converts a {x, y, w, h} table row to a half-open rectangle.
func stRect(x, y, w, h int) image.Rectangle { return image.Rect(x, y, x+w, y+h) }

// Overlay placement (MENU-GEOM-005, MENU-GEOM-006), {x, y, w, h} in the frame,
// drawn 1:1. Transcribed from the spec's table:
//
//	btn | hover (x, y, w, h)  | pressed (x, y, w, h)
//	  1 | 112,  64, 212, 136  | 116,  64, 208, 138
//	  2 |  84,  88, 236, 148  |  88,  88, 236, 152
//	  3 |  84, 236, 236, 152  |  88, 236, 236, 152
//	  4 | 112, 276, 212, 136  | 116, 272, 208, 140
//	  5 | 320,  60, 212, 140  | 320,  64, 212, 140
//	  6 | 324,  88, 236, 148  | 324,  88, 232, 152
//	  7 | 324, 236, 236, 152  | 324, 236, 236, 152
//	  8 | 324, 276, 208, 136  | 320, 272, 212, 140
var (
	stHover = [8]image.Rectangle{
		stRect(112, 64, 212, 136),
		stRect(84, 88, 236, 148),
		stRect(84, 236, 236, 152),
		stRect(112, 276, 212, 136),
		stRect(320, 60, 212, 140),
		stRect(324, 88, 236, 148),
		stRect(324, 236, 236, 152),
		stRect(324, 276, 208, 136),
	}
	stPressedRects = [8]image.Rectangle{
		stRect(116, 64, 208, 138),
		stRect(88, 88, 236, 152),
		stRect(88, 236, 236, 152),
		stRect(116, 272, 208, 140),
		stRect(320, 64, 212, 140),
		stRect(324, 88, 232, 152),
		stRect(324, 236, 236, 152),
		stRect(320, 272, 212, 140),
	}
)

// The eight hot mask indices (MENU-MASK-003). They matter here only because the
// fixture builder paints a mask; nothing this file asserts reads it.
var stHotIndices = [8]byte{0x80, 0x90, 0xa0, 0xb0, 0xc0, 0xd0, 0xe0, 0xf0}

// ---------------------------------------------------------------------------
// One shared, full-size, valid asset set
// ---------------------------------------------------------------------------

// stLoad builds the eighteen 640x480-scale bitmaps once: they are the expensive
// part, and every case here derives from the same set.
var stLoad = sync.OnceValues(func() (*menu.Assets, error) {
	return menu.Load(mapSource(synth.MenuFiles(synth.MenuOptions{
		Prefix:    stPrefix,
		Hover:     stHover,
		Pressed:   stPressedRects,
		MaskIndex: stHotIndices,
	})))
})

func stAssets(t *testing.T) *menu.Assets {
	t.Helper()
	a, err := stLoad()
	if err != nil {
		t.Fatalf("Load of a complete, consistent synthetic set failed: %v", err)
	}
	if a == nil {
		t.Fatal("Load returned a nil *Assets and a nil error")
	}
	return a
}

// ---------------------------------------------------------------------------
// Pixel helpers — the expected frame is built from the decoded inputs
// (Assets.Base / Hover / Pressed) and the transcribed tables above, never from
// Compose or Overlay.
// ---------------------------------------------------------------------------

func stPix(img *image.RGBA, x, y int) color.RGBA {
	i := img.PixOffset(x, y)
	return color.RGBA{R: img.Pix[i], G: img.Pix[i+1], B: img.Pix[i+2], A: img.Pix[i+3]}
}

func stExpectOverlay(a *menu.Assets, sel int, pressed bool) (img *image.RGBA, at image.Rectangle, ok bool) {
	if sel < 1 || sel > 8 {
		return nil, image.Rectangle{}, false
	}
	if pressed {
		return a.Pressed[sel-1], stPressedRects[sel-1], true
	}
	return a.Hover[sel-1], stHover[sel-1], true
}

// stCheckFrame asserts, pixel for pixel over the whole 640x480 frame, that got
// is the base with at most one overlay copied opaquely at that state's own
// rectangle, the overlay's pixel (0,0) landing at the rectangle's Min.
func stCheckFrame(t *testing.T, ctx string, a *menu.Assets, got *image.RGBA, sel int, pressed bool) {
	t.Helper()
	if got == nil {
		t.Fatalf("%s: Compose returned nil", ctx)
	}
	if got.Bounds() != stFrameRect {
		t.Fatalf("%s: Compose bounds = %v, want the 640x480 frame %v", ctx, got.Bounds(), stFrameRect)
	}

	ov, at, hasOverlay := stExpectOverlay(a, sel, pressed)
	var ovMin image.Point
	if hasOverlay {
		ovMin = ov.Bounds().Min
	}

	bad := 0
	for y := stFrameRect.Min.Y; y < stFrameRect.Max.Y; y++ {
		for x := stFrameRect.Min.X; x < stFrameRect.Max.X; x++ {
			var want color.RGBA
			var from string
			if hasOverlay && (image.Point{X: x, Y: y}).In(at) {
				want = stPix(ov, ovMin.X+x-at.Min.X, ovMin.Y+y-at.Min.Y)
				from = "overlay"
			} else {
				want = stPix(a.Base, x, y)
				from = "base"
			}
			if g := stPix(got, x, y); g != want {
				bad++
				if bad <= 5 {
					t.Errorf("%s: frame pixel (%d,%d) = %v, want %v (from the %s)", ctx, x, y, g, want, from)
				}
			}
		}
	}
	if bad > 5 {
		t.Errorf("%s: %d frame pixels differ from the required composition in total", ctx, bad)
	}
}

// stDiffFromBase returns the number of pixels of got that differ from the base,
// split into those inside r and those outside it, plus one witness of each.
func stDiffFromBase(a *menu.Assets, got *image.RGBA, r image.Rectangle) (inside, outside int, outsideWitness image.Point) {
	outsideWitness = image.Point{X: -1, Y: -1}
	for y := stFrameRect.Min.Y; y < stFrameRect.Max.Y; y++ {
		for x := stFrameRect.Min.X; x < stFrameRect.Max.X; x++ {
			if stPix(got, x, y) == stPix(a.Base, x, y) {
				continue
			}
			if (image.Point{X: x, Y: y}).In(r) {
				inside++
				continue
			}
			if outside == 0 {
				outsideWitness = image.Point{X: x, Y: y}
			}
			outside++
		}
	}
	return inside, outside, outsideWitness
}

func stState(sel int, pressed bool) menu.State {
	return menu.State{Selected: sel, Pressed: pressed}
}

func stFresh() *menu.Selection { return &menu.Selection{} }

func stWantState(t *testing.T, ctx string, s *menu.Selection, wantSel int, wantPressed bool) {
	t.Helper()
	got := s.State()
	if got.Selected != wantSel || got.Pressed != wantPressed {
		t.Errorf("%s: State() = {Selected:%d Pressed:%v}, want {Selected:%d Pressed:%v}",
			ctx, got.Selected, got.Pressed, wantSel, wantPressed)
	}
}

func TestSelectionAndCompose(t *testing.T) {
	a := stAssets(t)

	// Snapshot the base up front: DD10's "copy the base, then blit" must leave
	// the Assets' own bitmap untouched however many frames are composed.
	baseBefore := append([]byte(nil), a.Base.Pix...)

	if menu.ButtonCount != 8 {
		t.Fatalf("ButtonCount = %d, want 8 (spec: exactly eight buttons)", menu.ButtonCount)
	}

	// -----------------------------------------------------------------------
	// The latch state machine (DD9)
	// -----------------------------------------------------------------------

	t.Run("zero value selects nothing", func(t *testing.T) {
		var s menu.Selection
		stWantState(t, "zero value", &s, 0, false)
		if got := s.Latched(); got != 0 {
			t.Errorf("zero value: Latched() = %d, want 0", got)
		}
	})

	t.Run("move tracks hover", func(t *testing.T) {
		for n := 1; n <= 8; n++ {
			s := stFresh()
			s.Move(n)
			stWantState(t, "after Move(n)", s, n, false)
			if got := s.Latched(); got != 0 {
				t.Errorf("after Move(%d): Latched() = %d, want 0 — a move latches nothing", n, got)
			}
			s.Move(0)
			stWantState(t, "after Move(n) then Move(0)", s, 0, false)
		}
	})

	t.Run("press selects that button pressed", func(t *testing.T) {
		for n := 1; n <= 8; n++ {
			// The realistic order: the pointer is over the button, then the
			// primary button goes down.
			s := stFresh()
			s.Move(n)
			s.Press(n)
			stWantState(t, "after Move(n), Press(n)", s, n, true)
			if got := s.Latched(); got != n {
				t.Errorf("after Move(%d), Press(%d): Latched() = %d, want %d", n, n, got, n)
			}

			b := stFresh()
			b.Press(n)
			stWantState(t, "after a bare Press(n)", b, n, true)
			if got := b.Latched(); got != n {
				t.Errorf("after a bare Press(%d): Latched() = %d, want %d", n, got, n)
			}
		}
	})

	t.Run("held and moved off draws nothing", func(t *testing.T) {
		for n := 1; n <= 8; n++ {
			for m := 0; m <= 8; m++ {
				if m == n {
					continue
				}
				s := stFresh()
				s.Move(n)
				s.Press(n)
				s.Move(m)
				got := s.State()
				if got.Selected != 0 || got.Pressed {
					t.Errorf("Press(%d) then Move(%d): State() = {Selected:%d Pressed:%v}, want {Selected:0 Pressed:false}",
						n, m, got.Selected, got.Pressed)
				}
				if m != 0 && got.Selected == m {
					t.Errorf("Press(%d) then Move(%d): the new button became selected while held", n, m)
				}
				if got.Selected == n {
					t.Errorf("Press(%d) then Move(%d): the latched button stayed selected after the pointer left it", n, m)
				}
				if l := s.Latched(); l != n {
					t.Errorf("Press(%d) then Move(%d): Latched() = %d, want %d — the latch survives leaving", n, m, l, n)
				}
			}
		}
	})

	t.Run("latch survives re-entering", func(t *testing.T) {
		for n := 1; n <= 8; n++ {
			for _, m := range []int{0, 1 + n%8} {
				if m == n {
					continue
				}
				s := stFresh()
				s.Move(n)
				s.Press(n)
				s.Move(m)
				s.Move(n)
				stWantState(t, "after Press(n), Move(off), Move(n)", s, n, true)
			}
		}
	})

	t.Run("release on the latched button activates it", func(t *testing.T) {
		for n := 1; n <= 8; n++ {
			s := stFresh()
			s.Move(n)
			s.Press(n)
			if got := s.Release(n); got != n {
				t.Errorf("Press(%d) then Release(%d) = %d, want %d", n, n, got, n)
			}
			if l := s.Latched(); l != 0 {
				t.Errorf("after Release(%d): Latched() = %d, want 0", n, l)
			}
			if p := s.State().Pressed; p {
				t.Errorf("after Release(%d): State().Pressed = true, want false", n)
			}
			// Hover tracks normally again.
			k := 1 + n%8
			s.Move(k)
			stWantState(t, "after a completed click, Move(k)", s, k, false)
			s.Move(0)
			stWantState(t, "after a completed click, Move(0)", s, 0, false)
		}
	})

	t.Run("release elsewhere activates nothing but clears the latch", func(t *testing.T) {
		for n := 1; n <= 8; n++ {
			for m := 0; m <= 8; m++ {
				if m == n {
					continue
				}
				s := stFresh()
				s.Move(n)
				s.Press(n)
				if got := s.Release(m); got != 0 {
					t.Errorf("Press(%d) then Release(%d) = %d, want 0", n, m, got)
				}
				if l := s.Latched(); l != 0 {
					t.Errorf("Press(%d) then Release(%d): Latched() = %d, want 0 — the latch clears anyway", n, m, l)
				}
				k := 1 + n%8
				s.Move(k)
				stWantState(t, "after a cancelled click, Move(k)", s, k, false)
			}
		}
	})

	t.Run("press on no button latches nothing", func(t *testing.T) {
		s := stFresh()
		s.Press(0)
		if l := s.Latched(); l != 0 {
			t.Errorf("Press(0): Latched() = %d, want 0", l)
		}
		stWantState(t, "after Press(0)", s, 0, false)
		for k := 1; k <= 8; k++ {
			s.Move(k)
			stWantState(t, "after Press(0), Move(k)", s, k, false)
		}
		k := 3
		s.Move(k)
		if got := s.Release(k); got != 0 {
			t.Errorf("Press(0) then Move(%d) then Release(%d) = %d, want 0 — nothing was latched", k, k, got)
		}

		// The same with the pointer already over a button when it is pressed
		// "on nothing" (the pointer having left first).
		u := stFresh()
		u.Move(5)
		u.Press(0)
		if l := u.Latched(); l != 0 {
			t.Errorf("Move(5) then Press(0): Latched() = %d, want 0", l)
		}
		u.Move(6)
		stWantState(t, "after Press(0), Move(6)", u, 6, false)
		if got := u.Release(6); got != 0 {
			t.Errorf("Press(0) then Move(6) then Release(6) = %d, want 0", got)
		}
	})

	t.Run("release without a press activates nothing", func(t *testing.T) {
		for n := 0; n <= 8; n++ {
			s := stFresh()
			if got := s.Release(n); got != 0 {
				t.Errorf("Release(%d) on a fresh Selection = %d, want 0", n, got)
			}
			if l := s.Latched(); l != 0 {
				t.Errorf("Release(%d) on a fresh Selection: Latched() = %d, want 0", n, l)
			}
		}
		// And after a completed click, a second release activates nothing.
		s := stFresh()
		s.Move(2)
		s.Press(2)
		s.Release(2)
		if got := s.Release(2); got != 0 {
			t.Errorf("a second Release(2) = %d, want 0 — the latch was already cleared", got)
		}
	})

	t.Run("activation reports which button", func(t *testing.T) {
		if menu.NewGameButton < 1 || menu.NewGameButton > 8 {
			t.Fatalf("NewGameButton = %d, want a button in 1..8", menu.NewGameButton)
		}
		newGameHits := 0
		for n := 1; n <= 8; n++ {
			s := stFresh()
			s.Move(n)
			s.Press(n)
			got := s.Release(n)
			if got != n {
				t.Errorf("a full click on button %d activated %d, want %d", n, got, n)
			}
			if got == menu.NewGameButton {
				newGameHits++
			}
		}
		if newGameHits != 1 {
			t.Errorf("exactly one of the eight activations must be NewGameButton (=%d); got %d",
				menu.NewGameButton, newGameHits)
		}
	})

	// -----------------------------------------------------------------------
	// The overlay (DD10)
	// -----------------------------------------------------------------------

	t.Run("overlay picks the right bitmap and rectangle", func(t *testing.T) {
		for n := 1; n <= 8; n++ {
			for _, pressed := range []bool{false, true} {
				wantRect := stHover[n-1]
				wantImg := a.Hover[n-1]
				role := "hover"
				if pressed {
					wantRect = stPressedRects[n-1]
					wantImg = a.Pressed[n-1]
					role = "pressed"
				}

				img, at, ok := a.Overlay(stState(n, pressed))
				if !ok {
					t.Errorf("Overlay({%d, %v}): ok = false, want an overlay", n, pressed)
					continue
				}
				if at != wantRect {
					t.Errorf("Overlay({%d, %v}): rect = %v, want the %s rect %v", n, pressed, at, role, wantRect)
				}
				if img == nil {
					t.Errorf("Overlay({%d, %v}): image is nil with ok = true", n, pressed)
					continue
				}
				// 1:1 draw rule: the bitmap's own size is the (w,h) of its row.
				if img.Bounds().Dx() != at.Dx() || img.Bounds().Dy() != at.Dy() {
					t.Errorf("Overlay({%d, %v}): image size %dx%d, want %dx%d — the placement (w,h)",
						n, pressed, img.Bounds().Dx(), img.Bounds().Dy(), at.Dx(), at.Dy())
				}
				// Which bitmap it is. Buttons 3 and 7 have hover and pressed
				// rectangles of equal size (7's are identical), so only the
				// pixels tell a swap apart there.
				if !stSameImage(img, wantImg) {
					t.Errorf("Overlay({%d, %v}): returned a bitmap that is not the %s overlay of button %d",
						n, pressed, role, n)
				}
			}
		}
	})

	t.Run("overlay declines when nothing is selected", func(t *testing.T) {
		for _, sel := range []int{0, -1, 9, 100} {
			for _, pressed := range []bool{false, true} {
				img, at, ok := a.Overlay(stState(sel, pressed))
				if ok {
					t.Errorf("Overlay({%d, %v}): ok = true (rect %v), want false", sel, pressed, at)
				}
				if img != nil {
					t.Errorf("Overlay({%d, %v}): returned a non-nil image with ok = false", sel, pressed)
				}
			}
		}
	})

	// -----------------------------------------------------------------------
	// Composition (AC-10) — pixel level
	// -----------------------------------------------------------------------

	t.Run("nothing selected equals the base", func(t *testing.T) {
		stCheckFrame(t, "Compose({0,false})", a, a.Compose(stState(0, false)), 0, false)
	})

	t.Run("the fixture makes a blend detectable", func(t *testing.T) {
		// The assertions below compare composed pixels with the overlay's own
		// values. That is only evidence of an opaque copy if the base and the
		// overlay disagree everywhere inside the rectangle — otherwise a blend
		// (or a missing blit) could pass. Assert the precondition.
		for n := 1; n <= 8; n++ {
			for _, pressed := range []bool{false, true} {
				ov, at, _ := stExpectOverlay(a, n, pressed)
				ovMin := ov.Bounds().Min
				same := 0
				for y := at.Min.Y; y < at.Max.Y; y++ {
					for x := at.Min.X; x < at.Max.X; x++ {
						o := stPix(ov, ovMin.X+x-at.Min.X, ovMin.Y+y-at.Min.Y)
						if o == stPix(a.Base, x, y) {
							same++
						}
					}
				}
				if same != 0 {
					t.Errorf("fixture: button %d (pressed=%v) overlay agrees with the base at %d of %d pixels; "+
						"the opaque-copy assertions would be that much weaker",
						n, pressed, same, at.Dx()*at.Dy())
				}
			}
		}
	})

	t.Run("hover composites at the hover rect", func(t *testing.T) {
		// One from each column: button 1 (left) and button 5 (right).
		for _, n := range []int{1, 5} {
			stCheckFrame(t, stCtx("Compose hover", n), a, a.Compose(stState(n, false)), n, false)
		}
	})

	t.Run("pressed composites at the pressed rect", func(t *testing.T) {
		for _, n := range []int{1, 5} {
			stCheckFrame(t, stCtx("Compose pressed", n), a, a.Compose(stState(n, true)), n, true)
		}
	})

	t.Run("the overlay is copied, not blended", func(t *testing.T) {
		// Every pixel of the rectangle must equal the overlay's own value
		// exactly; the base differs from it at every one of them (asserted
		// above), so anything between the two is caught.
		for _, tc := range []struct {
			n       int
			pressed bool
		}{{1, false}, {1, true}, {6, false}, {8, true}} {
			ov, at, _ := stExpectOverlay(a, tc.n, tc.pressed)
			ovMin := ov.Bounds().Min
			got := a.Compose(stState(tc.n, tc.pressed))
			bad := 0
			for y := at.Min.Y; y < at.Max.Y; y++ {
				for x := at.Min.X; x < at.Max.X; x++ {
					want := stPix(ov, ovMin.X+x-at.Min.X, ovMin.Y+y-at.Min.Y)
					if g := stPix(got, x, y); g != want {
						bad++
						if bad <= 3 {
							t.Errorf("button %d (pressed=%v): composed (%d,%d) = %v, want the overlay's %v (base is %v)",
								tc.n, tc.pressed, x, y, g, want, stPix(a.Base, x, y))
						}
					}
				}
			}
			if bad > 3 {
				t.Errorf("button %d (pressed=%v): %d of %d rectangle pixels are not the overlay's own value",
					tc.n, tc.pressed, bad, at.Dx()*at.Dy())
			}
		}
	})

	t.Run("never two overlays at once", func(t *testing.T) {
		check := func(sel int, pressed bool) {
			got := a.Compose(stState(sel, pressed))
			_, at, ok := stExpectOverlay(a, sel, pressed)
			if !ok {
				at = image.Rectangle{}
			}
			inside, outside, w := stDiffFromBase(a, got, at)
			if outside != 0 {
				t.Errorf("state {%d, %v}: %d pixels outside %v differ from the base (first at %v) — more than one rectangle changed",
					sel, pressed, outside, at, w)
			}
			want := at.Dx() * at.Dy()
			if inside != want {
				t.Errorf("state {%d, %v}: %d of %d pixels inside %v differ from the base, want all of them",
					sel, pressed, inside, want, at)
			}
		}
		check(0, false)
		check(0, true) // not reachable from the machine, but must still draw nothing
		for n := 1; n <= 8; n++ {
			check(n, false)
			check(n, true)
		}
	})

	t.Run("compose returns a fresh image", func(t *testing.T) {
		first := a.Compose(stState(4, false))
		orig := stPix(first, 0, 0)
		mutated := color.RGBA{R: ^orig.R, G: ^orig.G, B: ^orig.B, A: 0xff}
		i := first.PixOffset(0, 0)
		first.Pix[i], first.Pix[i+1], first.Pix[i+2], first.Pix[i+3] = mutated.R, mutated.G, mutated.B, mutated.A

		second := a.Compose(stState(4, false))
		if second == first {
			t.Fatal("Compose returned the same *image.RGBA twice")
		}
		if got := stPix(second, 0, 0); got != orig {
			t.Errorf("a second Compose saw the first result's mutation: (0,0) = %v, want %v", got, orig)
		}
		stCheckFrame(t, "Compose after mutating an earlier result", a, second, 4, false)
	})

	// -----------------------------------------------------------------------
	// AC-10's end-to-end sequence, in order
	// -----------------------------------------------------------------------

	t.Run("AC-10 sequence", func(t *testing.T) {
		const (
			first  = 1 // left column
			second = 5 // right column
		)
		s := stFresh()

		// 1. Hover the first button: its hover bitmap at its normal rectangle.
		s.Move(first)
		stWantState(t, "AC-10 hover", s, first, false)
		stCheckFrame(t, "AC-10 hover", a, a.Compose(s.State()), first, false)

		// 2. Press it: its pressed bitmap at its pressed rectangle.
		s.Press(first)
		stWantState(t, "AC-10 press", s, first, true)
		stCheckFrame(t, "AC-10 press", a, a.Compose(s.State()), first, true)
		if stHover[first-1] == stPressedRects[first-1] {
			t.Fatalf("AC-10: button %d's hover and pressed rectangles are equal in the transcribed tables, "+
				"so this step could not tell them apart", first)
		}

		// 3. Move onto a different button while still held: the frame equals
		//    the base and the other button does not light.
		s.Move(second)
		got := s.State()
		if got.Selected != 0 || got.Pressed {
			t.Errorf("AC-10 dragged off: State() = {Selected:%d Pressed:%v}, want {Selected:0 Pressed:false}",
				got.Selected, got.Pressed)
		}
		if got.Selected == second {
			t.Errorf("AC-10 dragged off: button %d lit while the pointer was held from button %d", second, first)
		}
		frame := a.Compose(got)
		stCheckFrame(t, "AC-10 dragged off", a, frame, 0, false)
		if inside, outside, _ := stDiffFromBase(a, frame, image.Rectangle{}); inside+outside != 0 {
			t.Errorf("AC-10 dragged off: %d pixels differ from the base, want 0", inside+outside)
		}
		if _, _, ok := a.Overlay(got); ok {
			t.Error("AC-10 dragged off: Overlay reports an overlay to draw, want none")
		}

		// 4. Release there: activates nothing. Hover then resumes on that
		//    button, un-pressed.
		if act := s.Release(second); act != 0 {
			t.Errorf("AC-10 released off: activated %d, want 0", act)
		}
		stWantState(t, "AC-10 after releasing off", s, second, false)
		stCheckFrame(t, "AC-10 after releasing off", a, a.Compose(s.State()), second, false)

		// 5. A fresh press-and-release back on the original: it activates.
		s.Move(first)
		stWantState(t, "AC-10 back on the original", s, first, false)
		s.Press(first)
		stWantState(t, "AC-10 pressed again", s, first, true)
		stCheckFrame(t, "AC-10 pressed again", a, a.Compose(s.State()), first, true)
		if act := s.Release(first); act != first {
			t.Errorf("AC-10 released on the original: activated %d, want %d", act, first)
		}
		if l := s.Latched(); l != 0 {
			t.Errorf("AC-10 released on the original: Latched() = %d, want 0", l)
		}
	})

	// -----------------------------------------------------------------------
	// The Assets' own base is never written to (DD10: copy, then blit).
	// -----------------------------------------------------------------------

	t.Run("the base bitmap is untouched", func(t *testing.T) {
		if len(a.Base.Pix) != len(baseBefore) {
			t.Fatalf("Assets.Base.Pix changed length: %d, want %d", len(a.Base.Pix), len(baseBefore))
		}
		for i := range baseBefore {
			if a.Base.Pix[i] != baseBefore[i] {
				t.Fatalf("Assets.Base was modified at Pix[%d]: %d, want %d", i, a.Base.Pix[i], baseBefore[i])
			}
		}
	})
}

// stSameImage reports whether two bitmaps hold the same pixels at the same size.
func stSameImage(a, b *image.RGBA) bool {
	if a == nil || b == nil {
		return a == b
	}
	ab, bb := a.Bounds(), b.Bounds()
	if ab.Dx() != bb.Dx() || ab.Dy() != bb.Dy() {
		return false
	}
	for y := 0; y < ab.Dy(); y++ {
		for x := 0; x < ab.Dx(); x++ {
			if stPix(a, ab.Min.X+x, ab.Min.Y+y) != stPix(b, bb.Min.X+x, bb.Min.Y+y) {
				return false
			}
		}
	}
	return true
}

func stCtx(what string, n int) string {
	return fmt.Sprintf("%s button %d", what, n)
}
