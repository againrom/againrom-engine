package menu

import (
	"image"
	"image/draw"

	"againrom/pkg/render/latch"
)

// State is what the menu shows in one frame: which button is selected, and
// whether it is shown pressed rather than merely hovered.
//
// Selected is 1..ButtonCount, or 0 for none — and 0 means the frame is exactly
// the base bitmap. At most one button is ever selected, so at most one overlay is
// ever composited; there is no representable state that asks for two.
type State struct {
	Selected int
	Pressed  bool
}

// Selection is the menu's pointer state machine: which button the cursor is over,
// and which button (if any) the primary mouse button is latched on.
//
// The zero value is the correct initial state — nothing hovered, nothing latched.
// It is pure: it holds no assets, reads no clock and takes no engine input, so
// the whole interaction contract is decidable without a window.
//
// Only a NON-ZERO latch suppresses hover tracking. A press that lands on no
// button latches nothing and therefore changes nothing, and the cursor goes on
// selecting buttons normally while the mouse button is held. Keeping a separate
// "is the mouse down" flag would make that case suppress hover, which is exactly
// what the specification forbids.
type Selection struct {
	hit   int         // button under the cursor, 0 for none
	press latch.Latch // the button the press latched onto
}

// Move records the button under the cursor.
func (s *Selection) Move(hit int) { s.hit = hit }

// Press records the primary mouse button going down over hit. A press on no
// button latches nothing.
func (s *Selection) Press(hit int) {
	s.hit = hit
	s.press.Press(hit, hit != 0)
}

// Release records the primary mouse button coming up over hit, and reports which
// button was activated — the latched one, and only when the release lands on it.
// A release anywhere else activates nothing. Either way the latch is cleared.
//
// The caller decides what an activation means; this package binds no button to
// any command. Only one activation has a meaning in the front-end at all, and a
// release on any of the others is consumed here and does nothing further.
func (s *Selection) Release(hit int) (activated int) {
	s.hit = hit
	if at, ok := s.press.Release(hit, true); ok {
		activated = at
	}
	return activated
}

// Clear drops the latch without activating anything, as on a lost focus.
func (s *Selection) Clear() { s.press.Clear() }

// Latched reports the button the primary mouse button is currently held on, or 0.
func (s *Selection) Latched() int {
	b, _ := s.press.Latched()
	return b
}

// State resolves the pointer state to what the frame should show:
//
//   - nothing latched: the hovered button, drawn hovered (or nothing).
//   - latched, cursor still on it: that button, drawn pressed.
//   - latched, cursor moved off it: NOTHING — not the button now under the
//     cursor, and not the latched button either. The frame equals the base until
//     the pointer comes back or the button is released.
func (s *Selection) State() State {
	if s.press.Holds() {
		if s.press.Pressed(s.hit) {
			return State{Selected: s.hit, Pressed: true}
		}
		return State{}
	}
	return State{Selected: s.hit}
}

// Overlay returns the single overlay a state asks for: the bitmap and the frame
// rectangle to draw it at, or ok == false for none.
//
// "At most one" is structural here rather than a rule the caller must keep: the
// signature cannot express two, and State cannot represent two. The rectangle is
// the button's own row of the hover or pressed table, whose size is that bitmap's
// own pixel size — so the overlay draws 1:1 in frame pixels and nothing is
// resampled inside the frame.
func (a *Assets) Overlay(s State) (img *image.RGBA, at image.Rectangle, ok bool) {
	if a == nil || s.Selected < 1 || s.Selected > ButtonCount {
		return nil, image.Rectangle{}, false
	}
	i := s.Selected - 1
	hover, pressed := HoverRects[i], PressedRects[i]
	if a.place != nil {
		hover, pressed = a.place.hover[i], a.place.pressed[i]
	}
	if s.Pressed {
		return a.Pressed[i], pressed, a.Pressed[i] != nil
	}
	return a.Hover[i], hover, a.Hover[i] != nil
}

// Compose renders one menu frame: the base brooch bitmap over the whole 640x480
// frame, then at most one button overlay at its own rectangle.
//
// The overlay is copied OPAQUELY (draw.Src, not draw.Over). The source carries no
// alpha either way, so the two would behave identically — Src is used to say in
// the code that no blend is applied, because no blend is decoded. The research
// establishes the overlays as opaque 24-bit rectangles and explicitly leaves any
// blending the original may apply outside what it has established, so a colour
// key or an alpha rule here would be invented rather than reproduced. If the real
// menu turns out to show a seam at a rectangle edge, that is a discrepancy to
// report upstream, not a licence to guess a rule.
//
// The result is a fresh image each call; the caller owns it.
func (a *Assets) Compose(s State) *image.RGBA {
	dst := image.NewRGBA(image.Rect(0, 0, FrameW, FrameH))
	if a == nil || a.Base == nil {
		return dst
	}
	draw.Draw(dst, dst.Bounds(), a.Base, a.Base.Bounds().Min, draw.Src)

	if img, at, ok := a.Overlay(s); ok {
		draw.Draw(dst, at, img, img.Bounds().Min, draw.Src)
		if a.place != nil {
			if c := a.place.caption[s.Selected-1]; c != nil {
				draw.Draw(dst, captionRect, c, c.Bounds().Min, draw.Src)
			}
		}
	}
	return dst
}
