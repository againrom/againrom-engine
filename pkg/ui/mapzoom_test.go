package ui

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"image"
	"image/color"
	"math"
	"testing"
	"time"

	"github.com/hajimehoshi/ebiten/v2"

	"againrom/pkg/render/catmullrom"
	"againrom/pkg/render/terrain"
	"againrom/pkg/words"
)

// zoomBackground is the colour the CPU witness prefills a frame with, so a
// pixel the map did not cover, or covered only in part, shows it.
var zoomBackground = color.RGBA{R: 255, B: 255, A: 255}

// zoomTileset holds two opaque cells: with uniform false a striped green one
// and a striped brown one, so a seam between them has two colours; with
// uniform true both are one green.
func zoomTileset(uniform bool) *terrain.Tileset {
	pal := color.Palette{
		color.RGBA{R: 40, G: 120, B: 60, A: 255},
		color.RGBA{R: 70, G: 150, B: 90, A: 255},
		color.RGBA{R: 150, G: 110, B: 70, A: 255},
		color.RGBA{R: 180, G: 140, B: 100, A: 255},
	}
	if uniform {
		pal = color.Palette{color.RGBA{R: 30, G: 140, B: 40, A: 255}}
	}
	cells := make([]*image.Paletted, 2)
	for i := range cells {
		cells[i] = image.NewPaletted(image.Rect(0, 0, terrain.CellSize, terrain.CellSize), pal)
		if uniform {
			continue
		}
		for y := 0; y < terrain.CellSize; y++ {
			for x := 0; x < terrain.CellSize; x++ {
				cells[i].SetColorIndex(x, y, uint8(2*i+(x/4+y/8)%2))
			}
		}
	}
	ts := &terrain.Tileset{}
	ts.Slots[0] = &terrain.Strip{SubCells: cells}
	return ts
}

// zoomDisc is a 32 by 32 structure frame holding an opaque red disc of
// radius 10 and transparent pixels around it.
func zoomDisc() *terrain.StaticFrame {
	f := &terrain.StaticFrame{Width: 32, Height: 32, Pixels: make([]terrain.StaticPixel, 32*32)}
	f.Palette[1] = color.RGBA{R: 200, G: 30, B: 30, A: 255}
	for y := 0; y < 32; y++ {
		for x := 0; x < 32; x++ {
			dx, dy := float64(x)-15.5, float64(y)-15.5
			if dx*dx+dy*dy <= 100 {
				f.Pixels[y*32+x] = terrain.StaticPixel{Index: 1, Opaque: true}
			}
		}
	}
	return f
}

// zoomViewer is a flat, unshaded 24 by 24 cell map, its tiles a checkerboard
// of the tileset's two cells, with one disc structure anchored on cell
// (8, 8), seen through a 320 by 240 view at zoom z from a fractional origin.
func zoomViewer(t *testing.T, uniform, smoothing bool, z float64) *Viewer {
	t.Helper()
	words := make([]uint16, 24*24)
	for i := range words {
		words[i] = uint16((i%24 + i/24) % 2)
	}
	g := grid(24, 24, words...)
	g.Structures = []terrain.StructureRecord{{X: 8 << 8, Y: 8 << 8, Key: 1}}
	set := &terrain.StructureSet{}
	disc := zoomDisc()
	set.Classes[1] = &terrain.StructureClass{TileWidth: 1, TileHeight: 1, FullHeight: 2, Frames: []*terrain.StaticFrame{disc, disc}}
	v, err := NewViewerWithStatics("zoom", g, zoomTileset(uniform), nil, false, false, terrain.AnimGateTiles, set, true)
	if err != nil {
		t.Fatalf("NewViewerWithStatics: %v", err)
	}
	v.SetFlat(true)
	v.SetUnshaded(true)
	v.SetGraphicsOptions(GraphicsOptions{Smoothing: smoothing, HideShadows: true})
	v.cam.ViewW, v.cam.ViewH = 320, 240
	v.cam.X, v.cam.Y, v.cam.Zoom = 100.3, 120.6, z
	return v
}

// cpuMapLayers composes the map's terrain and art under the viewer's camera
// on the CPU, through the headless terrain and art witnesses.
func cpuMapLayers(t *testing.T, v *Viewer) *image.RGBA {
	t.Helper()
	r := &headlessTerrainTarget{v: v, pix: image.NewRGBA(image.Rect(0, 0, v.cam.ViewW, v.cam.ViewH)), sources: map[*ebiten.Image]*image.RGBA{}}
	v.drawFlat(r)
	if r.err != nil {
		t.Fatal(r.err)
	}
	draws, err := v.HeadlessArtDraws()
	if err != nil {
		t.Fatal(err)
	}
	for _, d := range draws {
		compositeHeadlessArt(r.pix, d)
	}
	return r.pix
}

// cpuMapFrame is the map surface drawMap leaves, on the CPU: the layers at
// the camera's zoom where smoothMapPlacement declines, otherwise the zoom-1
// composition resampled by the Catmull-Rom reference at the placement's op
// over zoomBackground.
func cpuMapFrame(t *testing.T, v *Viewer) *image.RGBA {
	t.Helper()
	unit, op, ok := v.smoothMapPlacement()
	if !ok {
		return cpuMapLayers(t, v)
	}
	var comp *image.RGBA
	v.underCamera(unit, func() { comp = cpuMapLayers(t, v) })
	out := image.NewRGBA(image.Rect(0, 0, v.cam.ViewW, v.cam.ViewH))
	for i := 0; i < len(out.Pix); i += 4 {
		out.Pix[i], out.Pix[i+1], out.Pix[i+2], out.Pix[i+3] = zoomBackground.R, zoomBackground.G, zoomBackground.B, zoomBackground.A
	}
	catmullrom.Resample(out, comp, op.GeoM.Element(0, 0), op.GeoM.Element(0, 2), op.GeoM.Element(1, 2))
	return out
}

func zoomFrameHash(img *image.RGBA) string {
	sum := sha256.Sum256(img.Pix)
	return hex.EncodeToString(sum[:8])
}

// The rule answers only with Smoothing on, at a zoom other than 1, and within
// the composition's size cap; its camera is the view's at zoom 1 from a whole
// world pixel less the apron, its op the zoom and the origin's remainder.
func TestSmoothMapPlacementRule(t *testing.T) {
	for _, tc := range []struct {
		name      string
		smoothing bool
		z         float64
		viewW     int
		ok        bool
	}{
		{"off", false, 1.2, 320, false},
		{"native", true, 1, 320, false},
		{"zoomed in", true, 1.2, 320, true},
		{"zoomed out", true, 1 / 1.2, 320, true},
		{"past the cap", true, 0.2, 1000, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			v := zoomViewer(t, false, tc.smoothing, tc.z)
			v.cam.ViewW = tc.viewW
			unit, op, ok := v.smoothMapPlacement()
			if ok != tc.ok {
				t.Fatalf("ok = %v, want %v", ok, tc.ok)
			}
			if !ok {
				return
			}
			wantW := int(math.Ceil(float64(tc.viewW)/tc.z)) + 2*smoothMapApron + 1
			wantH := int(math.Ceil(240/tc.z)) + 2*smoothMapApron + 1
			if unit.Zoom != 1 || unit.X != 100-smoothMapApron || unit.Y != 120-smoothMapApron || unit.ViewW != wantW || unit.ViewH != wantH {
				t.Fatalf("unit camera = zoom %v at (%v,%v) size %dx%d, want zoom 1 at (98,118) size %dx%d",
					unit.Zoom, unit.X, unit.Y, unit.ViewW, unit.ViewH, wantW, wantH)
			}
			g := op.GeoM
			wantTX, wantTY := (unit.X-100.3)*tc.z, (unit.Y-120.6)*tc.z
			if g.Element(0, 0) != tc.z || g.Element(1, 1) != tc.z || g.Element(0, 1) != 0 || g.Element(1, 0) != 0 ||
				math.Abs(g.Element(0, 2)-wantTX) > 1e-9 || math.Abs(g.Element(1, 2)-wantTY) > 1e-9 {
				t.Fatalf("op = %v, want Scale(%v) then Translate(%v,%v)", g.String(), tc.z, wantTX, wantTY)
			}
		})
	}
}

// Draw sends the map through blitSmoothMap only where the rule answers, with
// the rule's op and a composition the rule's size; elsewhere the map is drawn
// straight onto the map surface and no composition exists.
func TestDrawBlitsTheSmoothMapOnlyWhereTheRuleAnswers(t *testing.T) {
	type blit struct {
		size image.Point
		op   ebiten.GeoM
	}
	var got []blit
	saved := blitSmoothMap
	t.Cleanup(func() { blitSmoothMap = saved })
	blitSmoothMap = func(_, src *ebiten.Image, op *ebiten.DrawImageOptions) {
		got = append(got, blit{src.Bounds().Size(), op.GeoM})
	}
	for _, tc := range []struct {
		name      string
		smoothing bool
		z         float64
		blits     bool
	}{
		{"off zoomed", false, 1.2, false},
		{"on native", true, 1, false},
		{"off native", false, 1, false},
		{"on zoomed in", true, 1.2, true},
		{"on zoomed out", true, 1 / 1.2, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got = nil
			v, err := NewViewer("zoom", grid(60, 60), zoomTileset(false))
			if err != nil {
				t.Fatal(err)
			}
			layoutViewport(v, 320, 240)
			v.SetGraphicsOptions(GraphicsOptions{Smoothing: tc.smoothing})
			v.cam.SetZoom(tc.z)
			v.cam.X, v.cam.Y = 300.4, 280.7
			v.Draw(ebiten.NewImage(v.frameW, v.frameH))
			if !tc.blits {
				if len(got) != 0 || v.smoothMap != nil {
					t.Fatalf("%d smooth blits, composition %v; want none", len(got), v.smoothMap != nil)
				}
				return
			}
			unit, op, ok := v.smoothMapPlacement()
			if !ok || len(got) != 1 {
				t.Fatalf("rule ok %v, %d smooth blits; want one", ok, len(got))
			}
			if got[0].size != image.Pt(unit.ViewW, unit.ViewH) || got[0].op != op.GeoM {
				t.Fatalf("blit %v at %v, want %dx%d at %v", got[0].size, got[0].op.String(), unit.ViewW, unit.ViewH, op.GeoM.String())
			}
			if v.cam.Zoom != tc.z {
				t.Fatalf("camera zoom after Draw = %v, want %v restored", v.cam.Zoom, tc.z)
			}
		})
	}
}

// zoomNativeHash is the CPU frame of zoomViewer's checkerboard at zoom 1,
// computed on the base commit's renderer.
const zoomNativeHash = "1ac987a21597882b"

// At zoom 1 the frame with Smoothing on equals the frame with it off and the
// base commit's frame.
func TestSmoothMapLeavesTheNativeFrameUnchanged(t *testing.T) {
	on := zoomFrameHash(cpuMapFrame(t, zoomViewer(t, false, true, 1)))
	off := zoomFrameHash(cpuMapFrame(t, zoomViewer(t, false, false, 1)))
	if on != off || on != zoomNativeHash {
		t.Fatalf("zoom-1 frame on %s, off %s, want both %s", on, off, zoomNativeHash)
	}
}

// At 1.2 and 1/1.2 the smooth frame differs from today's, and every pixel of
// it is opaque map content: none is transparent or the background colour,
// on the tile seams (the columns and rows a cell boundary crosses, and their
// neighbours) or anywhere else in the view.
func TestSmoothMapHasNoSeamOrGap(t *testing.T) {
	for _, z := range []float64{1.2, 1 / 1.2} {
		t.Run(fmt.Sprint(z), func(t *testing.T) {
			v := zoomViewer(t, false, true, z)
			on := cpuMapFrame(t, v)
			off := cpuMapFrame(t, zoomViewer(t, false, false, z))
			if zoomFrameHash(on) == zoomFrameHash(off) {
				t.Fatal("the smooth frame equals today's frame")
			}
			bad := func(x, y int) bool {
				p := on.RGBAAt(x, y)
				return p.A != 255 || p == zoomBackground
			}
			seams := 0
			for k := 0; k <= 24; k++ {
				sx := (float64(k*terrain.CellSize)-v.cam.X)*z - 0.5
				sy := (float64(k*terrain.CellSize)-v.cam.Y)*z - 0.5
				for d := -1; d <= 1; d++ {
					if x := int(math.Floor(sx)) + d; x >= 0 && x < v.cam.ViewW {
						seams++
						for y := 0; y < v.cam.ViewH; y++ {
							if bad(x, y) {
								t.Fatalf("seam column %d row %d is %v", x, y, on.RGBAAt(x, y))
							}
						}
					}
					if y := int(math.Floor(sy)) + d; y >= 0 && y < v.cam.ViewH {
						seams++
						for x := 0; x < v.cam.ViewW; x++ {
							if bad(x, y) {
								t.Fatalf("seam row %d column %d is %v", y, x, on.RGBAAt(x, y))
							}
						}
					}
				}
			}
			if seams < 12 {
				t.Fatalf("only %d seam lines in view", seams)
			}
			for y := 0; y < v.cam.ViewH; y++ {
				for x := 0; x < v.cam.ViewW; x++ {
					if bad(x, y) {
						t.Fatalf("pixel (%d,%d) is %v", x, y, on.RGBAAt(x, y))
					}
				}
			}
		})
	}
}

// The disc is composed over the terrain before the scale, so no pixel of the
// smooth frame mixes in the sprite's transparent surround: on the uniform
// green map (channel sum 210) with the red disc (260), every pixel keeps a
// channel sum of at least 85 percent of the darker of the two.
func TestSmoothMapHasNoSpriteHalo(t *testing.T) {
	for _, z := range []float64{1.2, 1 / 1.2, 2.5} {
		t.Run(fmt.Sprint(z), func(t *testing.T) {
			v := zoomViewer(t, true, true, z)
			v.cam.X, v.cam.Y = 272.3-160/z, 256.6-120/z
			on := cpuMapFrame(t, v)
			red := 0
			for y := 0; y < on.Rect.Dy(); y++ {
				for x := 0; x < on.Rect.Dx(); x++ {
					p := on.RGBAAt(x, y)
					if p.R > 150 {
						red++
					}
					if sum := int(p.R) + int(p.G) + int(p.B); p.A != 255 || float64(sum) < 0.85*210 {
						t.Fatalf("pixel (%d,%d) is %v", x, y, p)
					}
				}
			}
			if red == 0 {
				t.Fatal("the disc is not in view")
			}
		})
	}
}

// zoomRedCentroid is the x centroid of the frame's red weight, R above the
// green ground's 30, clamped to the disc's own 170.
func zoomRedCentroid(img *image.RGBA) float64 {
	var sum, moment float64
	for y := 0; y < img.Rect.Dy(); y++ {
		for x := 0; x < img.Rect.Dx(); x++ {
			w := min(max(float64(img.RGBAAt(x, y).R)-30, 0), 170)
			sum += w
			moment += w * (float64(x) + 0.5)
		}
	}
	return moment / sum
}

// Panning by a quarter world pixel at a time moves the smooth frame's disc by
// a quarter of the zoom in screen pixels each step, to within 0.05 pixels:
// the map glides instead of jumping a whole pixel at a time. Today's frame
// misses by at least 0.1 pixels on the same pans.
func TestSmoothMapPansWithoutShimmer(t *testing.T) {
	const z, step = 1.2, 0.25
	worst := func(smoothing bool) float64 {
		v := zoomViewer(t, true, smoothing, z)
		x0 := v.cam.X
		c0 := zoomRedCentroid(cpuMapFrame(t, v))
		var err float64
		for k := 1; k <= 8; k++ {
			v.cam.X = x0 + step*float64(k)
			c := zoomRedCentroid(cpuMapFrame(t, v))
			err = max(err, math.Abs(c-c0+step*z*float64(k)))
		}
		return err
	}
	on, off := worst(true), worst(false)
	t.Logf("worst centroid error: smooth %.4f px, today %.4f px", on, off)
	if on >= 0.05 || off < 0.1 {
		t.Fatalf("pan centroid error smooth %.4f px, today %.4f px; want the smooth one below 0.05 and today's, the fixture's witness, at least 0.1", on, off)
	}
}

// zoomNoticeViewer is stepViewer at zoom 1 holding format as its zoom notice.
func zoomNoticeViewer(t *testing.T, format string) *Viewer {
	t.Helper()
	v := stepViewer(t)
	v.SetFont(messageFont())
	w := AuthoredWords()
	w.ZoomNotice = format
	v.SetWords(w)
	if v.cam.Zoom != 1 {
		t.Fatalf("setup zoom = %v, want 1", v.cam.Zoom)
	}
	return v
}

// A wheel notch posts the zoom in percent of native; the next notch replaces
// that line and leaves other lines standing (DIV-4877).
func TestWheelNotchPostsTheZoomAndTheNextReplacesIt(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	v := zoomNoticeViewer(t, AuthoredWords().ZoomNotice)
	notch := func(dir float64) {
		v.step(Input{CursorX: centre.x, CursorY: centre.y, WheelY: dir}, now)
	}
	notch(+1)
	if got := noticeTexts(v); len(got) != 1 || got[0] != "Zoom: 120%" {
		t.Fatalf("after one notch in: %q, want [Zoom: 120%%]", got)
	}
	v.PostMessage("other", MessageWhite, time.Second)
	notch(+1)
	if got := noticeTexts(v); len(got) != 2 || got[0] != "other" || got[1] != "Zoom: 144%" {
		t.Fatalf("after a second notch in: %q, want [other Zoom: 144%%]", got)
	}
	notch(-1)
	notch(-1)
	notch(-1)
	if got := noticeTexts(v); len(got) != 2 || got[1] != "Zoom: 83%" {
		t.Fatalf("after three notches out: %q, want [other Zoom: 83%%]", got)
	}
}

// A notch at the camera's limit changes nothing and posts nothing new; the
// notch that reached the limit posted the limit.
func TestWheelNotchAtTheLimitPostsNothing(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	for _, tc := range []struct {
		dir   float64
		limit string
	}{{+1, "Zoom: 800%"}, {-1, "Zoom: 13%"}} {
		v := zoomNoticeViewer(t, AuthoredWords().ZoomNotice)
		for i := 0; i < 60; i++ {
			v.step(Input{CursorX: centre.x, CursorY: centre.y, WheelY: tc.dir}, now)
		}
		got := noticeTexts(v)
		if len(got) != 1 || got[0] != tc.limit {
			t.Fatalf("at the limit: %q, want [%s]", got, tc.limit)
		}
		serial := v.messages.serial
		v.step(Input{CursorX: centre.x, CursorY: centre.y, WheelY: tc.dir}, now)
		if v.messages.serial != serial {
			t.Fatalf("a notch at the %s limit posted again", tc.limit)
		}
	}
}

// The notice's words are the engine's own: English and Russian.
func TestZoomNoticeWords(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	for _, tc := range []struct {
		lang, want string
	}{{"english", "Zoom: 120%"}, {"russian", "Масштаб: 120%"}} {
		v := zoomNoticeViewer(t, words.For(tc.lang).Text("map.zoom"))
		v.step(Input{CursorX: centre.x, CursorY: centre.y, WheelY: +1}, now)
		if got := noticeTexts(v); len(got) != 1 || got[0] != tc.want {
			t.Fatalf("%s: %q, want [%s]", tc.lang, got, tc.want)
		}
	}
	if got := AuthoredWords().ZoomNotice; got != words.For("english").Text("map.zoom") {
		t.Fatalf("authored ZoomNotice %q is not the English word", got)
	}
}

// The world draw under the unit camera shades the unit under the pointer, and
// no other, exactly as the draw without Smoothing does.
func TestSmoothMapKeepsTheHoveredUnitHighlighted(t *testing.T) {
	shade := func(v *Viewer) map[uint32]bool {
		out := map[uint32]bool{}
		for _, s := range v.planeSprites() {
			if s.Inspection.Kind == InspectionUnit {
				out[s.Inspection.ID] = s.ShadeZero
			}
		}
		return out
	}
	for _, z := range []float64{1.2, 1 / 1.2, 2} {
		a, v := inspectionFixture(t, image.Pt(1024, 768))
		giveEntitiesArt(v)
		v.cam.Zoom = z
		v.cam.X, v.cam.Y = 40.3, 30.6
		inspectionHover(t, a, v, InspectionSubject{InspectionUnit, 7})
		v.refreshHoverLighting()
		want := shade(v)
		if !want[7] || want[1] {
			t.Fatalf("z=%v: native hover lighting %v does not light unit 7 alone", z, want)
		}
		v.SetGraphicsOptions(GraphicsOptions{Smoothing: true})
		unit, _, ok := v.smoothMapPlacement()
		if !ok {
			t.Fatalf("z=%v: rule declined", z)
		}
		var got map[uint32]bool
		v.underCamera(unit, func() { got = shade(v) })
		if len(got) != len(want) || got[7] != want[7] || got[1] != want[1] {
			t.Errorf("z=%v: under the unit camera %v, native %v", z, got, want)
		}
		if h, ok := v.hoverAt(v.cursorX, v.cursorY); !ok || h != (InspectionSubject{InspectionUnit, 7}) {
			t.Errorf("z=%v: hover after the draw %v/%v", z, h, ok)
		}
	}
}
