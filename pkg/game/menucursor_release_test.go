package game

import (
	"image"
	"math"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"againrom/pkg/render/frame"
)

// TestReleaseMainMenuCursorFollowsThePointer moves the pointer over the
// installed main menu through App input, one step per move, and reads where the
// next frame draws the installed `select` cursor.
//
// The picture stands on the pixel lattice of the 640x480 frame the menu is
// drawn in (AI-CURSOR-218), on the pixel the hit tests resolve for the pointer
// (MENU-MASK-004), and on the very next frame: the placement read after a move
// is that move's, with no earlier position in it.
func TestReleaseMainMenuCursorFollowsThePointer(t *testing.T) {
	f := releaseFront(t)
	reg := f.CursorRegistry.Value()
	if reg == nil {
		t.Fatalf("the production cursor registry did not resolve: %v", f.CursorRegistry.Err())
	}
	slot, ok := reg.Slot("select")
	if !ok || len(slot.Frames) != 1 {
		t.Fatalf("the installed registry has no one-frame `select` slot: %+v", slot)
	}
	a := f.App("main menu cursor")
	defer a.StopAudio()

	// A route across the frame that crosses the brooch, ending on two
	// neighbouring pixels.
	route := []image.Point{
		{40, 40}, {160, 120}, {280, 220}, {400, 300}, {520, 380}, {600, 440}, {300, 240}, {301, 240}, {302, 241},
	}
	const eps = 1e-9
	for _, win := range []image.Point{{640, 480}, {1000, 700}, {1280, 960}, {1920, 1080}, {2560, 1440}} {
		a.Layout(win.X, win.Y)
		place := frame.Fit(frame.W, frame.H, win.X, win.Y)
		ox, oy := place.Origin()
		scale := place.Scale()

		var prev image.Point
		for i, fp := range route {
			wx, wy, ok := place.FrameToWindow(fp)
			if !ok {
				t.Fatalf("window %v: frame pixel %v has no window pixel", win, fp)
			}
			// The first window pixel of the frame pixel, and, where the frame
			// pixel spans more than one, its second: both name the same pixel.
			points := []image.Point{{wx, wy}}
			if scale >= 2 {
				points = append(points, image.Pt(wx+1, wy+1))
			}
			var firstX, firstY float64
			for k, p := range points {
				if err := a.HeadlessPointer("hover", p.X, p.Y); err != nil {
					t.Fatal(err)
				}
				c, ok := a.HeadlessCursor()
				if !ok {
					t.Fatalf("window %v, pointer %v: the main menu draws no cursor", win, p)
				}
				if c.Pic != slot.Frames[0] || c.Hot != slot.Hotspot {
					t.Fatalf("window %v, pointer %v: drew picture %p with hotspot %v, want the installed `select` picture %p with hotspot %v",
						win, p, c.Pic, c.Hot, slot.Frames[0], slot.Hotspot)
				}
				if c.Tip != fp {
					t.Fatalf("window %v, pointer %v: the hotspot is on frame pixel %v the next frame, want %v (the previous move was %v)",
						win, p, c.Tip, fp, prev)
				}
				wantX := ox + float64(fp.X-slot.Hotspot.X)*scale
				wantY := oy + float64(fp.Y-slot.Hotspot.Y)*scale
				if c.Scale != scale || math.Abs(c.X-wantX) > eps || math.Abs(c.Y-wantY) > eps {
					t.Fatalf("window %v, pointer %v: picture top-left (%.4f,%.4f) scale %v, want (%.4f,%.4f) scale %v",
						win, p, c.X, c.Y, c.Scale, wantX, wantY, scale)
				}
				if got, in := place.WindowToFrame(p.X, p.Y); !in || got != fp {
					t.Fatalf("window %v: the pointer %v is on frame pixel %v (inside %v), the route named %v", win, p, got, in, fp)
				}
				if k == 0 {
					firstX, firstY = c.X, c.Y
				} else if math.Abs(c.X-firstX) > eps || math.Abs(c.Y-firstY) > eps {
					t.Fatalf("window %v: a second window pixel of frame pixel %v moved the picture from (%.4f,%.4f) to (%.4f,%.4f)",
						win, fp, firstX, firstY, c.X, c.Y)
				}
				if i == len(route)-1 && k == len(points)-1 {
					t.Logf("window %v scale %.4f: last step frame %v, picture top-left (%.3f,%.3f)", win, scale, fp, c.X, c.Y)
				}
			}
			prev = fp
		}

		// The frame the placement is for is drawn through the production Draw,
		// which reads the same placement.
		screen := ebiten.NewImage(win.X, win.Y)
		a.Draw(screen)
		screen.Dispose()
	}
}
