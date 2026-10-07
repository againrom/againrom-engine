package ui

// The mission screen's own geometry: what Draw composes on, and where it
// puts it in the window (contract B2 and B3).
//
// WHAT IS OBSERVED HERE IS THE PRODUCTION CALL, not a helper that agrees with
// it. Ebitengine refuses a pixel readback with no graphics context, so this file
// records the DrawImage call Viewer.Draw makes and reads the transform out of
// it. Everything else about Draw runs exactly as it ships: a real Viewer, a real
// Layout, a real compose onto a real canvas.
//
// EVERY EXPECTED VALUE IS WRITTEN OUT BY HAND from the window size and the
// fixed 768 logical height, with the arithmetic beside it. None of them is read
// back from frame.Fit, frame.Placement.Scale or frame.Placement.Origin, which
// are the code under test.
//
// Every fixture is synthetic, no window opens and nothing reads a game install.

import (
	"image"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"againrom/pkg/render/camera"
	"againrom/pkg/render/frame"
	"againrom/pkg/render/terrain"
)

// blitRecord is one recorded call of the blit Draw ends with.
type blitRecord struct {
	calls  int
	canvas image.Rectangle
	geom   ebiten.GeoM
	filter ebiten.Filter
}

// recordBlits replaces the package's blit for the duration of one test and
// records what Draw asks for. It deliberately does NOT forward to
// screen.DrawImage: the call itself is the observation.
func recordBlits(t *testing.T) *blitRecord {
	t.Helper()
	rec := &blitRecord{}
	prev := blitFrameCanvas
	blitFrameCanvas = func(screen, canvas *ebiten.Image, op *ebiten.DrawImageOptions) {
		rec.calls++
		rec.canvas = canvas.Bounds()
		rec.geom = op.GeoM
		rec.filter = op.Filter
	}
	t.Cleanup(func() { blitFrameCanvas = prev })
	return rec
}

// geometryViewer is a viewer over a world larger than the map view on both
// axes, so nothing about the placement below depends on a clamp.
func geometryViewer(t *testing.T) *Viewer {
	t.Helper()
	v, err := NewViewer("missiongeometry", grid(64, 64), &terrain.Tileset{})
	if err != nil {
		t.Fatalf("NewViewer: %v", err)
	}
	return v
}

// missionPlacements are the windows this file places the mission frame in. A
// 4:3 or taller window retains 1024x768. A wider window gets
// ceil(winW*768/winH) logical pixels, so width binds and no vertical bar is
// left. The ceiling can leave less than one logical pixel split top/bottom.
var missionPlacements = []struct {
	name          string
	winW, winH    int
	logicalW      int
	scale, ox, oy float64
}{
	// Both axes bind at once: the window is the frame's own size and a multiple
	// of it, so nothing is letterboxed.
	{"at the frame's own size", 1024, 768, 1024, 1, 0, 0},
	{"at twice the frame", 2048, 1536, 1024, 2, 0, 0},
	{"at half the frame", 512, 384, 1024, 0.5, 0, 0},
	// 1600/1024 and 1200/768 are both 25/16, so this one is 4:3 too.
	{"at 1600x1200, 4:3 and not a whole multiple", 1600, 1200, 1024, 1.5625, 0, 0},
	// 16:9: ceil(1920*768/1080) = 1366. Width binds at 1920/1366,
	// putting the rounding remainder on the horizontal edges, not the sides.
	{"at 1920x1080, expanded to both side edges", 1920, 1080, 1366,
		1920.0 / 1366.0, 0, (1080.0 - 768.0*1920.0/1366.0) / 2},
	// Same 16:9 geometry below 1:1.
	{"at 1280x720, expanded and below 1:1", 1280, 720, 1366,
		1280.0 / 1366.0, 0, (720.0 - 768.0*1280.0/1366.0) / 2},
	// Ultrawide: ceil(3440*768/1440) = 1835.
	{"at 3440x1440 ultrawide", 3440, 1440, 1835,
		3440.0 / 1835.0, 0, (1440.0 - 768.0*3440.0/1835.0) / 2},
	// Width binds: 1024/1024 = 1, the frame draws 768 tall, and 1200-768 = 432
	// is split into two 216s.
	{"at 1024x1200, letterboxed top and bottom", 1024, 1200, 1024, 1, 0, 216},
	// A one-pixel-wider window gets one logical pixel rather than two half-pixel
	// side bars.
	{"at 1025x768, one extra logical pixel", 1025, 768, 1025, 1, 0, 0},
}

func TestMissionDrawComposesOnTheFrameAndPlacesIt(t *testing.T) {
	for _, tc := range missionPlacements {
		t.Run(tc.name, func(t *testing.T) {
			v := geometryViewer(t)
			v.Layout(tc.winW, tc.winH)
			rec := recordBlits(t)

			v.Draw(ebiten.NewImage(tc.winW, tc.winH))

			if rec.calls != 1 {
				t.Fatalf("Draw made %d blits, want exactly 1", rec.calls)
			}
			if want := image.Rect(0, 0, tc.logicalW, MissionFrameH); rec.canvas != want {
				t.Errorf("Draw composed on %v, want %v: the frame is composed at its own size and not the window's",
					rec.canvas, want)
			}
			if rec.filter != ebiten.FilterNearest {
				t.Errorf("Draw placed the frame with filter %v, want FilterNearest: a scaled frame keeps hard pixel edges",
					rec.filter)
			}

			// GeoM after Scale(s,s) then Translate(ox,oy): the diagonal carries
			// the scale, the third column carries the origin, and the two
			// off-diagonal terms are zero because the frame is never sheared.
			for _, e := range []struct {
				i, j int
				want float64
				what string
			}{
				{0, 0, tc.scale, "horizontal scale"},
				{1, 1, tc.scale, "vertical scale"},
				{0, 1, 0, "horizontal shear"},
				{1, 0, 0, "vertical shear"},
				{0, 2, tc.ox, "origin x"},
				{1, 2, tc.oy, "origin y"},
			} {
				if got := rec.geom.Element(e.i, e.j); got != e.want {
					t.Errorf("in a %dx%d window the blit's %s = %v, want %v",
						tc.winW, tc.winH, e.what, got, e.want)
				}
			}
		})
	}

	// Nothing is drawn before the window has a size. The placement a viewer is
	// constructed with is usable, so this is the only way to reach the branch.
	t.Run("an unusable placement draws nothing", func(t *testing.T) {
		v := geometryViewer(t)
		v.place = frame.Placement{}
		rec := recordBlits(t)
		v.Draw(ebiten.NewImage(800, 600))
		if rec.calls != 0 {
			t.Fatalf("Draw made %d blits with no usable placement, want none", rec.calls)
		}
	})
}

// TestMissionCanvasFollowsLogicalBounds proves both halves of the resize
// lifecycle: a scale-only resize reuses the GPU image, while a logical-width
// change disposes it exactly once and Draw allocates the replacement bounds.
func TestMissionCanvasFollowsLogicalBounds(t *testing.T) {
	v := geometryViewer(t)
	v.Layout(1024, 768)
	rec := recordBlits(t)
	v.Draw(ebiten.NewImage(1024, 768))
	first := v.canvas
	if first == nil {
		t.Fatal("the first Draw allocated no mission canvas")
	}

	disposals := 0
	disposed := (*ebiten.Image)(nil)
	oldDispose := disposeFrameCanvas
	disposeFrameCanvas = func(canvas *ebiten.Image) {
		disposals++
		disposed = canvas
	}
	t.Cleanup(func() { disposeFrameCanvas = oldDispose })

	// Same 4:3 logical bounds at another scale.
	v.Layout(2048, 1536)
	if v.canvas != first || disposals != 0 {
		t.Fatalf("scale-only resize replaced/disposed canvas: canvas=%p first=%p disposals=%d", v.canvas, first, disposals)
	}

	// 16:9 changes the logical width from 1024 to 1366.
	v.Layout(1920, 1080)
	if v.canvas != nil || disposals != 1 || disposed != first {
		t.Fatalf("wide resize left canvas=%p, disposals=%d disposed=%p; want nil, 1, first %p", v.canvas, disposals, disposed, first)
	}
	v.Draw(ebiten.NewImage(1920, 1080))
	if v.canvas == nil || v.canvas == first || v.canvas.Bounds() != image.Rect(0, 0, 1366, 768) {
		t.Fatalf("replacement canvas=%p bounds=%v; want a new 1366x768 image", v.canvas, v.canvas.Bounds())
	}
	if rec.calls != 2 {
		t.Fatalf("two Draw calls made %d blits, want 2", rec.calls)
	}

	// Another 16:9 scale has the same logical bounds and must reuse it.
	wide := v.canvas
	v.Layout(2560, 1440)
	if v.canvas != wide || disposals != 1 {
		t.Fatalf("same-aspect resize replaced/disposed canvas: canvas=%p wide=%p disposals=%d", v.canvas, wide, disposals)
	}
}

func TestMissionFramePartitionsIntoViewportAndStrip(t *testing.T) {
	full := image.Rect(0, 0, MissionFrameW, MissionFrameH)
	view := image.Rect(0, 0, MissionViewportSize().X, MissionViewportSize().Y)
	strip := MissionPanelRect()

	if got := view.Union(strip); got != full {
		t.Errorf("map view %v and strip %v cover %v, want the whole frame %v", view, strip, got, full)
	}
	if got := view.Intersect(strip); !got.Empty() {
		t.Errorf("map view %v and strip %v overlap on %v, want no overlap", view, strip, got)
	}
	if strip.Dx() != 0xa0 {
		t.Errorf("the strip is %d frame pixels wide, want 0xa0: SESS-VIEW-028's map view rect is (0, 0, screenW - 0xa0, screenH)", strip.Dx())
	}

	// SESS-VIEW-028's spans for a 1024x768 screen, written out from the claim
	// rather than divided out of the constants above: 27 columns by 24 rows.
	if got := view.Dx() / camera.CellSize; got != 27 {
		t.Errorf("the map view spans %d columns, want 27 (SESS-VIEW-028 at 1024x768)", got)
	}
	if got := view.Dy() / camera.CellSize; got != 24 {
		t.Errorf("the map view spans %d rows, want 24 (SESS-VIEW-028 at 1024x768)", got)
	}
}
