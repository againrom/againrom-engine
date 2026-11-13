package ui

import (
	"image"
	"image/color"
	"math"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

// scaledBlit is one draw the final-frame scaler made at a scale other than 1.
type scaledBlit struct {
	kind     string
	src, dst *ebiten.Image
	scale    float64
}

// recordScaledBlits replaces the scaler's three paths with recorders for the
// test and answers the list they fill.
func recordScaledBlits(t *testing.T) *[]scaledBlit {
	t.Helper()
	var got []scaledBlit
	add := func(kind string, dst, src *ebiten.Image, op *ebiten.DrawImageOptions) {
		if s := op.GeoM.Element(0, 0); s != 1 {
			got = append(got, scaledBlit{kind, src, dst, s})
		}
	}
	n, c, b := nearestBlit, catmullRomBlit, sharpBilinearBlit
	t.Cleanup(func() { nearestBlit, catmullRomBlit, sharpBilinearBlit = n, c, b })
	nearestBlit = func(dst, src *ebiten.Image, op *ebiten.DrawImageOptions) { add("nearest", dst, src, op) }
	catmullRomBlit = func(dst, src *ebiten.Image, op *ebiten.DrawImageOptions) bool {
		add("catmull-rom", dst, src, op)
		return true
	}
	sharpBilinearBlit = func(dst, src *ebiten.Image, op *ebiten.DrawImageOptions, _ **ebiten.Image) {
		add("sharp bilinear", dst, src, op)
	}
	return &got
}

// Every picture of the mission frame is scaled at most once, straight onto
// the window: the zoom-1 map composition by the zoom times the window scale,
// the canvas by the window scale. Smoothing on scales with the Catmull-Rom
// kernel and never by another filter; Smoothing off scales only the map for
// the pixel look. The map marks are part of the zoom-1 composition.
func TestMissionFrameIsResampledOnceOntoTheWindow(t *testing.T) {
	got := recordScaledBlits(t)
	for _, win := range []image.Point{{1024, 768}, {1920, 1080}, {2560, 1440}} {
		for _, z := range []float64{1, 1.2, 1.44, 1 / 1.2} {
			for _, smoothing := range []bool{true, false} {
				*got = nil
				v, err := NewViewer("zoom", grid(60, 60), zoomTileset(false))
				if err != nil {
					t.Fatal(err)
				}
				v.SetGraphicsOptions(GraphicsOptions{Smoothing: smoothing})
				v.Layout(win.X, win.Y)
				v.cam.SetZoom(z)
				v.cam.X, v.cam.Y = 300.4, 280.7
				screen := ebiten.NewImage(win.X, win.Y)
				v.Draw(screen)
				s := v.place.Scale()
				want := map[*ebiten.Image]float64{}
				if s != 1 {
					want[v.canvas] = s
				}
				if z != 1 {
					want[v.smoothMap] = z * s
				}
				if v.marksComposed != (z != 1) {
					t.Fatalf("%v zoom %v: marks composed at zoom 1 = %v", win, z, v.marksComposed)
				}
				if len(*got) != len(want) {
					t.Fatalf("%v zoom %v smoothing %v: %d scaled blits %v, want %d", win, z, smoothing, len(*got), *got, len(want))
				}
				for _, b := range *got {
					f, ok := want[b.src]
					if !ok || b.dst != screen || math.Abs(b.scale-f) > 1e-9 {
						t.Fatalf("%v zoom %v: blit %+v, want one blit per picture onto the window at %v", win, z, b, want)
					}
					delete(want, b.src)
					wantKind := "catmull-rom"
					if !smoothing && b.src == v.smoothMap {
						wantKind = "sharp bilinear"
						if f < 2 {
							wantKind = "nearest"
						}
					}
					if b.kind != wantKind {
						t.Fatalf("%v zoom %v smoothing %v: filter %s, want %s", win, z, smoothing, b.kind, wantKind)
					}
				}
			}
		}
	}
}

// Smoothing off at scale 1.4: the map is nearest, the canvas the starter's.
func TestSmoothingOffKeepsTheStartersFilterOnTheInterface(t *testing.T) {
	got := recordScaledBlits(t)
	for _, starterOff := range []bool{false, true} {
		*got = nil
		v, err := NewViewer("zoom", grid(60, 60), zoomTileset(false))
		if err != nil {
			t.Fatal(err)
		}
		v.SetGraphicsOptions(GraphicsOptions{Smoothing: false})
		v.SetFrameSmoothing(!starterOff)
		v.Layout(1920, 1080)
		v.cam.SetZoom(1.2)
		screen := ebiten.NewImage(1920, 1080)
		v.Draw(screen)
		if s := v.place.Scale(); math.Abs(s-1.4) > 0.01 {
			t.Fatalf("window scale %v, want about 1.4", s)
		}
		canvasWant := "catmull-rom"
		if starterOff {
			canvasWant = "sharp bilinear"
		}
		kinds := map[*ebiten.Image]string{}
		for _, b := range *got {
			kinds[b.src] = b.kind
		}
		if kinds[v.smoothMap] != "nearest" || kinds[v.canvas] != canvasWant {
			t.Errorf("starter switch off %v: map %q, interface %q; want map nearest, interface %s",
				starterOff, kinds[v.smoothMap], kinds[v.canvas], canvasWant)
		}
	}
}

// The map editor keeps its own draw: with Smoothing off it composes nothing
// at zoom 1 and its marks stay out of the composition.
func TestMapEditorKeepsItsZoomedDraw(t *testing.T) {
	v := zoomViewer(t, false, true, 1.2)
	v.editorView = true
	v.SetGraphicsOptions(GraphicsOptions{})
	if _, _, ok := v.smoothMapPlacement(); ok {
		t.Fatal("the map editor composes at zoom 1 with Smoothing off")
	}
}

// The scaler's reference copies exactly at scale 1, repeats whole pixels at
// a whole factor for the pixel look, and keeps a flat picture flat under the
// Catmull-Rom kernel at a factor that is not whole.
func TestReferenceFinalFrameCopiesAndKeepsFlatColour(t *testing.T) {
	src := image.NewRGBA(image.Rect(0, 0, 5, 4))
	for i := range src.Pix {
		src.Pix[i] = byte(i*37 + 11)
	}
	for i := 3; i < len(src.Pix); i += 4 {
		src.Pix[i] = 0xff
	}
	same := image.NewRGBA(src.Rect)
	if name := referenceFinalFrame(same, src, 1, 0, 0, filterSmooth); name != "exact copy" || string(same.Pix) != string(src.Pix) {
		t.Fatalf("scale 1 ran %q and changed the picture", name)
	}
	twice := image.NewRGBA(image.Rect(0, 0, 10, 8))
	referenceFinalFrame(twice, src, 2, 0, 0, filterPixel)
	for y := range 8 {
		for x := range 10 {
			if twice.RGBAAt(x, y) != src.RGBAAt(x/2, y/2) {
				t.Fatalf("pixel look at factor 2: (%d,%d) is not source pixel (%d,%d)", x, y, x/2, y/2)
			}
		}
	}
	flat := image.NewRGBA(image.Rect(0, 0, 8, 8))
	for i := 0; i < len(flat.Pix); i += 4 {
		flat.Pix[i], flat.Pix[i+1], flat.Pix[i+2], flat.Pix[i+3] = 90, 140, 30, 255
	}
	for _, pixelLook := range []frameFilter{filterSmooth, filterSharp, filterPixel} {
		out := image.NewRGBA(image.Rect(0, 0, 15, 15))
		referenceFinalFrame(out, flat, 1.875, 0, 0, pixelLook)
		for y := range 15 {
			for x := range 15 {
				if out.RGBAAt(x, y) != (color.RGBA{90, 140, 30, 255}) {
					t.Fatalf("pixel look %v: flat colour became %v at (%d,%d)", pixelLook, out.RGBAAt(x, y), x, y)
				}
			}
		}
	}
}
