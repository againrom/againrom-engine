package ui

import (
	"image"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

// markViewer is a mission viewer on a 320 by 240 map surface at zoom z with
// Smoothing on, its camera off the map's corner.
func markViewer(t *testing.T, z float64) *Viewer {
	t.Helper()
	v, err := NewViewer("zoom", grid(60, 60), zoomTileset(false))
	if err != nil {
		t.Fatal(err)
	}
	layoutViewport(v, 320, 240)
	v.SetGraphicsOptions(GraphicsOptions{Smoothing: true})
	v.cam.SetZoom(z)
	v.cam.X, v.cam.Y = 300.4, 280.7
	return v
}

// The drag rectangle is in frame pixels of the pointer. At every zoom it is
// the one pass that stays out of the zoom-1 composition and is drawn on the
// canvas where the pointer pressed, at the size the pointer travelled.
func TestDragRectangleStaysOnTheCanvasInFramePixels(t *testing.T) {
	for _, z := range []float64{1, 1.2, 1.44, 1 / 1.2} {
		v := markViewer(t, z)
		v.dragging, v.boxing, v.dragMoved = true, true, 1000
		v.pressX, v.pressY, v.dragX, v.dragY = 100, 80, 220, 200
		v.Draw(ebiten.NewImage(v.frameW, v.frameH))
		if v.marksComposed != (z != 1) {
			t.Fatalf("zoom %v: marks composed = %v", z, v.marksComposed)
		}
		frame := 0
		for _, pass := range v.overlayPasses() {
			if !pass.Frame {
				continue
			}
			frame++
			if pass.Color != MarqueeColor || marksMap.holds(pass) || !marksFrame.holds(pass) || !marksAll.holds(pass) {
				t.Fatalf("zoom %v: frame pass %+v is not the drag rectangle kept for the canvas", z, pass.Color)
			}
		}
		if frame != 1 {
			t.Fatalf("zoom %v: %d passes in frame pixels, want the drag rectangle alone", z, frame)
		}
		canvas := image.NewRGBA(image.Rect(0, 0, v.cam.ViewW, v.cam.ViewH))
		v.headlessOverlayRects(canvas, marksFrame)
		top := v.marqueeScreenRects()[0]
		if top.X != 100 || top.Y != 80 || top.W != 120 {
			t.Errorf("zoom %v: the drag rectangle's top strip is at (%v,%v) width %v, want (100,80) width 120", z, top.X, top.Y, top.W)
		}
		if canvas.RGBAAt(100, 80) != MarqueeColor || canvas.RGBAAt(160, 80) != MarqueeColor ||
			canvas.RGBAAt(160, 78) == MarqueeColor || canvas.RGBAAt(160, 140) == MarqueeColor {
			t.Errorf("zoom %v: the drag rectangle's top strip is not on the canvas row the pointer pressed", z)
		}
		unit, _, ok := v.smoothMapPlacement()
		if !ok {
			continue
		}
		comp := image.NewRGBA(image.Rect(0, 0, unit.ViewW, unit.ViewH))
		v.underCamera(unit, func() { v.headlessOverlayRects(comp, marksMap) })
		for i := 0; i < len(comp.Pix); i += 4 {
			if comp.Pix[i+3] != 0 {
				t.Fatalf("zoom %v: the zoom-1 composition holds a mark of the drag rectangle", z)
			}
		}
	}
}

// A mission dialogue remaps the canvas pixels under its backdrop and frame
// shadows. While it is open the map is drawn on the canvas, so the backdrop
// reaches it; without it the zoomed map goes to the window apart.
func TestDialogueBackdropReachesTheZoomedMap(t *testing.T) {
	savedBlit, savedSubmit := blitSmoothMap, submitDialogueBackdrop
	t.Cleanup(func() { blitSmoothMap, submitDialogueBackdrop = savedBlit, savedSubmit })
	for _, dialogue := range []bool{false, true} {
		var mapDst, backdropDst *ebiten.Image
		blitSmoothMap = func(dst, _ *ebiten.Image, _ *ebiten.DrawImageOptions, _ frameFilter) { mapDst = dst }
		submitDialogueBackdrop = func(dst *ebiten.Image, _ []ebiten.Vertex, _ *ebiten.Shader, _ *ebiten.DrawTrianglesShaderOptions) {
			backdropDst = dst
		}
		v := markViewer(t, 1.2)
		v.SetFont(panelFont())
		if dialogue {
			v.SetNotice("words", NoticeDialogue)
		}
		if v.canvasRemapsMap() != dialogue {
			t.Fatalf("dialogue %v: canvasRemapsMap = %v", dialogue, v.canvasRemapsMap())
		}
		screen := ebiten.NewImage(v.frameW, v.frameH)
		v.Draw(screen)
		if mapDst == nil {
			t.Fatalf("dialogue %v: the zoomed map was not placed", dialogue)
		}
		if !dialogue {
			if mapDst != screen || backdropDst != nil {
				t.Errorf("no dialogue: the map must go to the window once and no backdrop is submitted")
			}
			continue
		}
		if backdropDst == nil {
			t.Fatal("dialogue: no backdrop submitted")
		}
		if mapDst == screen || mapDst.Bounds().Dx() != v.cam.ViewW {
			t.Errorf("dialogue: the map was placed on %v, not on the canvas's map surface the backdrop remaps", mapDst.Bounds())
		}
		if !v.marksComposed {
			t.Error("dialogue: the map marks left the zoom-1 composition")
		}
	}
}

// The cell grid hides where a cell is under GridMinCellPixels at the
// player's zoom, also inside the zoom-1 composition, whose camera has zoom 1.
func TestGridFloorReadsThePlayersZoomInsideTheComposition(t *testing.T) {
	for _, tc := range []struct {
		zoom  float64
		drawn bool
	}{{0.4, false}, {0.49, false}, {0.6, true}, {1.2, true}} {
		v := markViewer(t, tc.zoom)
		v.showGrid = true
		unit, _, ok := v.smoothMapPlacement()
		if !ok {
			t.Fatalf("zoom %v: no composition", tc.zoom)
		}
		var segments []gridSegment
		v.underCamera(unit, func() {
			if v.playerZoom() != tc.zoom {
				t.Fatalf("zoom %v: the composition reads zoom %v", tc.zoom, v.playerZoom())
			}
			segments = v.gridScreenSegments()
		})
		if (len(segments) > 0) != tc.drawn {
			t.Errorf("zoom %v: %d grid segments in the composition, drawn want %v", tc.zoom, len(segments), tc.drawn)
		}
		if v.playerZoom() != v.cam.Zoom {
			t.Errorf("zoom %v: the player's zoom reads %v after the composition", tc.zoom, v.playerZoom())
		}
	}
}

// The pixel filter draws through nearest neighbour below a scale of 2 and
// through method B from 2 up; method B itself is unchanged below 2. With
// Smoothing off the mission's map, canvas and pointer use the pixel filter;
// with it on the map is Catmull-Rom and the canvas follows the starter.
func TestPixelFilterIsNearestBelowTwo(t *testing.T) {
	got := recordScaledBlits(t)
	dst, src := ebiten.NewImage(64, 64), ebiten.NewImage(8, 8)
	for _, tc := range []struct {
		filter frameFilter
		scale  float64
		want   string
	}{
		{filterPixel, 1 / 1.2, "nearest"}, {filterPixel, 1.2, "nearest"}, {filterPixel, 1.999, "nearest"},
		{filterPixel, 2, "sharp bilinear"}, {filterPixel, 2.25, "sharp bilinear"},
		{filterSharp, 1.2, "sharp bilinear"}, {filterSmooth, 1.2, "catmull-rom"}, {filterSmooth, 2.25, "catmull-rom"},
	} {
		*got = nil
		var op ebiten.DrawImageOptions
		op.GeoM.Scale(tc.scale, tc.scale)
		var buf *ebiten.Image
		drawFinalFrame(dst, src, &op, &buf, tc.filter)
		if len(*got) != 1 || (*got)[0].kind != tc.want {
			t.Errorf("filter %d at %v: %v, want one %s draw", tc.filter, tc.scale, *got, tc.want)
		}
	}
	for _, tc := range []struct {
		smoothing           bool
		mapWant, canvasWant frameFilter
	}{
		{true, filterSmooth, filterSmooth},
		{false, filterPixel, filterSharp},
	} {
		v := markViewer(t, 1.2)
		v.SetGraphicsOptions(GraphicsOptions{Smoothing: tc.smoothing})
		if v.mapFilter() != tc.mapWant || v.canvasFilter() != tc.canvasWant {
			t.Errorf("%+v: map filter %d, canvas filter %d", tc, v.mapFilter(), v.canvasFilter())
		}
	}
}

// The scenario key "home" reaches the mission's zoom reset.
func TestHeadlessHomeKeyResetsTheZoom(t *testing.T) {
	a, v := qolApp(t)
	v.cam.SetZoom(defaultMapZoom() * 1.44)
	if err := a.HeadlessKey("home"); err != nil {
		t.Fatal(err)
	}
	if v.cam.Zoom != defaultMapZoom() {
		t.Errorf("the scenario's home key left zoom %v, want %v", v.cam.Zoom, defaultMapZoom())
	}
}
