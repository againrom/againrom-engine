package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"image"
	"image/color"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/render/terrain"
)

// --- the fixture --------------------------------------------------------

// The placement bytes the fixture map carries. A byte names the class whose ID
// is one less, so these are the classes' IDs plus one; staticUnknown names no
// loaded class at all, and staticArtless names one whose sheet the archive
// deliberately does not hold. Both are SKIPS and not errors, so a run over this
// map must succeed and must not count either as a placement.
const (
	staticA       = 1   // ID 0: the 20x30 sheet, on cell (1,0)
	staticB       = 2   // ID 1: the 20x40 sheet, on cell (1,1)
	staticArtless = 3   // ID 2: names objects/missing/gone.256, which is absent
	staticUnknown = 200 // ID 199: no section carries it
)

// The two placed cells, and the class geometry every expected coordinate below
// is derived from. Both classes declare a 32x32 canvas with its centre pixel at
// (16,16); only the FRAME sizes differ, which is what makes the two anchors
// differ and the two sprites overlap.
const (
	staticACol, staticARow = 1, 0
	staticBCol, staticBRow = 1, 1

	staticCanvas  = 32 // Width and Height
	staticCentre  = 16 // CenterX and CenterY
	staticAFrameW = 20
	staticAFrameH = 30
	staticBFrameW = 20
	staticBFrameH = 40
)

// The two sprite colours, and the palette indices that select them. Neither
// colour occurs in the terrain the fixture composites (asserted, not assumed),
// and neither is any of the three marker colours.
var (
	staticColorA = color.RGBA{R: 0x10, G: 0x20, B: 0x30, A: 0xff}
	staticColorB = color.RGBA{R: 0x40, G: 0x50, B: 0x60, A: 0xff}
)

const (
	staticIndexA = 1
	staticIndexB = 2
)

// staticSpritePalette is the palette both fixture sheets carry. Entry 0 is left
// black and unused: transparency here is structural, so nothing depends on index
// 0 meaning a hole, and these frames are wholly opaque in any case.
func staticSpritePalette() []color.RGBA {
	pal := make([]color.RGBA, 8)
	pal[staticIndexA] = color.RGBA{R: staticColorA.R, G: staticColorA.G, B: staticColorA.B}
	pal[staticIndexB] = color.RGBA{R: staticColorB.R, G: staticColorB.G, B: staticColorB.B}
	return pal
}

// solidSheet builds a one-frame .256 sheet whose every pixel is an opaque index.
// A wholly opaque frame is what lets a pixel assertion be about PLACEMENT alone:
// the frame's rectangle and the pixels it paints are the same set, so a wrong
// top-left cannot be masked by a hole.
func solidSheet(w, h int, index uint8) []byte {
	pixels := make([]synth.Pixel256, w*h)
	for i := range pixels {
		pixels[i] = synth.Pixel256{Index: index, Opaque: true}
	}
	return synth.Sheet256(synth.Sheet256Options{
		Palette: staticSpritePalette(),
		Frames:  []synth.Frame256{{Width: w, Height: h, Pixels: pixels}},
	})
}

// staticObjectRegistry writes the fixture objects/objects.reg: three dense
// sections sharing one canvas and one centre pixel, differing only in which
// [Files] entry they name.
func staticObjectRegistry() []byte {
	key := func(name string, v int32) synth.RegNode {
		return synth.RegNode{Name: name, Kind: 0x02, Int: v}
	}
	class := func(id, file int32) []synth.RegNode {
		return []synth.RegNode{
			key("ID", id), key("File", file), key("Index", 0),
			key("Width", staticCanvas), key("Height", staticCanvas),
			key("CenterX", staticCentre), key("CenterY", staticCentre),
		}
	}
	return synth.ObjectsReg(
		[]string{`trees\a`, `trees\b`, `missing\gone`},
		class(0, 0),
		class(1, 1),
		class(2, 2), // the art this archive does not hold
	)
}

// writeStaticArchive lays a graphics.res holding BOTH the terrain strips and the
// object layer's own entries — which is the shipped arrangement and the reason
// the tool loads the bundle off the archive it already opened.
//
// objects/missing/gone.256 is absent on purpose: the absent-sheet exclusion needs
// no builder, only an omission.
func writeStaticArchive(t *testing.T, dir string) string {
	t.Helper()

	path := filepath.Join(dir, "graphics.res")
	archive := synth.Archive([]synth.File{
		{Path: "terrain/tile1-00.bmp", Data: solidStrip(14, landBase)},
		{Path: "terrain/tile3-00.bmp", Data: solidStrip(8, waterBase)},
		{Path: "terrain/dirt.bmp", Data: solidStrip(4, dirtBase)},
		{Path: "objects/objects.reg", Data: staticObjectRegistry()},
		{Path: "objects/trees/a.256", Data: solidSheet(staticAFrameW, staticAFrameH, staticIndexA)},
		{Path: "objects/trees/b.256", Data: solidSheet(staticBFrameW, staticBFrameH, staticIndexB)},
	})
	if err := os.WriteFile(path, archive, 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

// staticOverlay is the fixture map's type-3 grid on the 4x3 lattice: two cells
// that resolve to drawable art, one naming a class with no art, one naming no
// class at all, and eight empty cells.
//
// Four non-zero bytes and TWO placements is the point — the summary's count is
// the placements drawn, not the cells that named something.
func staticOverlay() []uint8 {
	return []uint8{
		0, staticA, 0, 0,
		0, staticB, 0, 0,
		staticArtless, 0, 0, staticUnknown,
	}
}

// tilePayload serialises a tile grid to the type-1 record's little-endian words.
func tilePayload(tiles []uint16) []byte {
	out := make([]byte, 2*len(tiles))
	for i, w := range tiles {
		binary.LittleEndian.PutUint16(out[i*2:], w)
	}
	return out
}

// writeStaticMap lays a 4x3 map carrying the overlay above, one placed type-4
// object and one placed type-6 unit — BOTH anchored on cell (1,0), the same cell
// staticA stands on, so a composed render exercises all three glyphs at one
// point. The stored anchors carry a non-zero low byte so the decoder's >>8
// truncation is genuinely exercised.
//
// altitudes nil is the flat map (an all-zero grid, on which the projected and
// the flat geometry agree on every pixel); slopedAltitudes is the relief one.
func writeStaticMap(t *testing.T, dir, name string, altitudes []uint8) string {
	t.Helper()

	path := filepath.Join(dir, name)
	data := synth.ALM(synth.ALMOptions{
		Width: 4, Height: 3,
		Type1Payload: tilePayload(fixtureTiles()),
		Altitudes:    altitudes,
		Overlay:      staticOverlay(),
		Objects:      []synth.ALMObject{{X: staticACol<<8 | 0x40, Y: staticARow<<8 | 0x80}},
		Units:        []synth.ALMUnit{{X: staticACol<<8 | 0x80, Y: staticARow<<8 | 0x7f}},
	})
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

// writeStaticFixtures lays the shared archive and both maps into dir.
func writeStaticFixtures(t *testing.T, dir string) (flatMap, slopedMap string) {
	t.Helper()
	writeStaticArchive(t, dir)
	return writeStaticMap(t, dir, "statics.alm", nil),
		writeStaticMap(t, dir, "statics-sloped.alm", slopedAltitudes)
}

// --- the hand-worked geometry -------------------------------------------
//
// From the spec's "Sprite anchor", with every /2 truncating toward zero:
//
//	anchorX = (CenterX - Width/2)  + frameW/2 = (16 - 16) + frameW/2
//	anchorY = (CenterY - Height/2) + frameH/2 = (16 - 16) + frameH/2
//	destX   = col*32 + 16 - anchorX
//	destY   = row*32 + 16 - anchorY - lift - originY
//
// so A's anchor is (10, 15) and B's is (10, 20), and the ground point a sprite
// stands on — its top-left plus its own anchor — is
//
//	(col*32 + 16, row*32 + 16 - lift - originY).
//
// ON THE FLAT MAP lift and originY are both 0:
//
//	A at (1,0): top-left (38,  1), rect [38,58) x [ 1,31), ground (48,16)
//	B at (1,1): top-left (38, 28), rect [38,58) x [28,68), ground (48,48)
//
// so the two sprites OVERLAP on [38,58) x [28,31) — three rows the later cell,
// B, must own.
//
// ON THE SLOPED MAP originY is slopedOriginY = -20 and the lifts are the cells'
// four-corner altitude means over slopedAltitudes: cell (1,0)'s corners are
// 20,0,0,0 so lift 5, and cell (1,1)'s are all 0 so lift 0. Hence
//
//	A at (1,0): top-left (38, 16), rect [38,58) x [16,46), ground (48,31)
//	B at (1,1): top-left (38, 48), rect [38,58) x [48,88), ground (48,68)
//
// with no overlap, on a canvas 128 x slopedCanvasRows.
var (
	staticFlatRectA = image.Rect(38, 1, 58, 31)
	staticFlatRectB = image.Rect(38, 28, 58, 68)

	staticFlatGroundA = image.Pt(48, 16)
	staticFlatGroundB = image.Pt(48, 48)

	staticSlopedGroundA = image.Pt(48, 31)
	staticSlopedGroundB = image.Pt(48, 68)
)

// Sample points on the flat map, each chosen so that no marker glyph of any kind
// can reach it (the three crosses are centred on (48,16) and (48,48) and reach at
// most 6 px along an arm and 1 px across it away from those centres).
var (
	staticOnlyA  = image.Pt(40, 5)  // inside A alone
	staticOnlyB  = image.Pt(40, 60) // inside B alone
	staticShared = image.Pt(40, 29) // inside both: B's, or the draw order is wrong
)

// staticPlacementCount is what the summary must report for this fixture: two of
// the four non-zero overlay bytes resolve to a drawable frame.
const staticPlacementCount = 2

// --- helpers ------------------------------------------------------------

// runTool renders one invocation and returns its summary, failing the test on
// any error.
func runTool(t *testing.T, dir, mapPath, name string, args ...string) (summary string, img image.Image) {
	t.Helper()
	out := filepath.Join(dir, name+".png")
	argv := append([]string{"render", "-assets", dir, "-map", mapPath, "-out", out}, args...)
	var buf bytes.Buffer
	if err := run(argv, &buf); err != nil {
		t.Fatalf("%s run (%v): %v", name, args, err)
	}
	return buf.String(), decodePNG(t, out)
}

// differsOutside reports the first pixel outside every rect in keep at which got
// and want disagree, or ok == true when there is none.
func differsOutside(got, want image.Image, keep ...image.Rectangle) (at image.Point, ok bool) {
	b := want.Bounds()
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			p := image.Pt(x, y)
			inside := false
			for _, r := range keep {
				if p.In(r) {
					inside = true
					break
				}
			}
			if inside {
				continue
			}
			if color.RGBAModel.Convert(got.At(x, y)) != color.RGBAModel.Convert(want.At(x, y)) {
				return p, false
			}
		}
	}
	return image.Point{}, true
}

// --- tests --------------------------------------------------------------

func TestRenderStaticsScaleRefusal(t *testing.T) {
	dir := t.TempDir()
	mapPath, _ := writeStaticFixtures(t, dir)

	// An output file that already holds a real render. Its bytes are the
	// baseline every refusal below must leave untouched.
	existing := filepath.Join(dir, "existing.png")
	var buf bytes.Buffer
	if err := run([]string{"render", "-assets", dir, "-map", mapPath, "-out", existing}, &buf); err != nil {
		t.Fatalf("baseline run: %v", err)
	}
	before, err := os.ReadFile(existing)
	if err != nil {
		t.Fatal(err)
	}
	if len(before) == 0 {
		t.Fatal("fixture: the baseline output is empty, so byte-identity would prove nothing")
	}

	for _, scale := range []string{"2", "3", "0"} {
		t.Run("scale "+scale, func(t *testing.T) {
			// (a) an existing file stays byte-identical.
			var stdout bytes.Buffer
			err := run([]string{"render", "-assets", dir, "-map", mapPath,
				"-out", existing, "-statics", "-scale", scale}, &stdout)
			if err == nil {
				t.Fatalf("-statics -scale %s was accepted", scale)
			}
			if stdout.Len() != 0 {
				t.Errorf("a refused run wrote a summary: %q", stdout.String())
			}
			after, readErr := os.ReadFile(existing)
			if readErr != nil {
				t.Fatalf("the refused run removed the existing output: %v", readErr)
			}
			if !bytes.Equal(before, after) {
				t.Errorf("the refused run rewrote the existing output: %d bytes, want the original %d",
					len(after), len(before))
			}

			// (b) a missing file is not created.
			missing := filepath.Join(dir, "missing-s"+scale+".png")
			stdout.Reset()
			if err := run([]string{"render", "-assets", dir, "-map", mapPath,
				"-out", missing, "-statics", "-scale", scale}, &stdout); err == nil {
				t.Fatalf("-statics -scale %s was accepted", scale)
			}
			if _, statErr := os.Stat(missing); !os.IsNotExist(statErr) {
				t.Errorf("the refused run created %s (stat error %v, want IsNotExist)", missing, statErr)
			}

			// The error still has to name the conflict, so a caller can act on it.
			if msg := err.Error(); !strings.Contains(msg, "-statics") || !strings.Contains(msg, "-scale 1") {
				t.Errorf("error %q does not name the -statics/-scale conflict", msg)
			}
		})
	}

	// The refusal is about -statics ALONE. -staticmarkers renders at scale 2 and
	// reports nothing, and at that scale the glyph is drawn on the SCALED lattice
	// while the placement list stays native: cell (1,0)'s cross centres at
	// (1*64 + 32, 0*64 + 32) = (96,32) — worked out from the marker contract, not
	// from the list.
	summary, img := runTool(t, dir, mapPath, "marks-s2", "-staticmarkers", "-scale", "2")
	if strings.Contains(summary, "statics") {
		t.Errorf("-staticmarkers alone reported a statics token: %q", summary)
	}
	if got := colorAt(img, 96, 32); got != terrain.StaticMarkerColor {
		t.Errorf("at -scale 2 the static cross for cell (1,0) is %v at (96,32), want %v",
			got, terrain.StaticMarkerColor)
	}
	// And -statics at the one scale it allows is accepted, or every assertion
	// above would be satisfied by a tool that refused the flag outright.
	if summary, _ := runTool(t, dir, mapPath, "statics-s1", "-statics", "-scale", "1"); !strings.Contains(summary, ", statics ") {
		t.Errorf("-statics -scale 1 produced no statics token: %q", summary)
	}
}

func TestRenderStaticsLayer(t *testing.T) {
	dir := t.TempDir()
	mapPath, _ := writeStaticFixtures(t, dir)

	plainSummary, plain := runTool(t, dir, mapPath, "plain", "-unshaded")

	// Fixture assumptions. Neither sprite colour may occur in the terrain, or a
	// pixel assertion below would not be evidence that anything was drawn.
	for _, tc := range []struct {
		label string
		c     color.RGBA
	}{{"A", staticColorA}, {"B", staticColorB}} {
		if n := countColor(plain, tc.c); n != 0 {
			t.Fatalf("the unrendered terrain already holds %d pixels of sprite %s's colour", n, tc.label)
		}
	}
	if strings.Contains(plainSummary, "statics") {
		t.Errorf("a flagless summary mentions statics: %q", plainSummary)
	}

	t.Run("the art lands at its anchors and the later row owns the overlap", func(t *testing.T) {
		summary, img := runTool(t, dir, mapPath, "statics", "-unshaded", "-statics")

		want := strings.TrimSuffix(plainSummary, "\n") + fmt.Sprintf(", statics %d (animated 0)\n", staticPlacementCount)
		if summary != want {
			t.Errorf("summary =\n  %q\nwant\n  %q", summary, want)
		}

		for _, tc := range []struct {
			label string
			at    image.Point
			want  color.RGBA
		}{
			{"inside A alone", staticOnlyA, staticColorA},
			{"inside B alone", staticOnlyB, staticColorB},
			{"the overlap, owned by the later row", staticShared, staticColorB},
			{"A's own ground point", staticFlatGroundA, staticColorA},
			{"B's own ground point", staticFlatGroundB, staticColorB},
		} {
			if got := colorAt(img, tc.at.X, tc.at.Y); got != tc.want {
				t.Errorf("%s: pixel %v = %v, want %v", tc.label, tc.at, got, tc.want)
			}
		}

		if at, ok := differsOutside(img, plain, staticFlatRectA, staticFlatRectB); !ok {
			t.Errorf("pixel %v changed although it lies outside both sprite rectangles %v and %v",
				at, staticFlatRectA, staticFlatRectB)
		}
		// And the flag alone draws no cross.
		if n := countColor(img, terrain.StaticMarkerColor); n != 0 {
			t.Errorf("-statics alone drew %d static-marker pixels", n)
		}
	})

	t.Run("the cross is drawn without the art, and reports nothing", func(t *testing.T) {
		summary, img := runTool(t, dir, mapPath, "marks", "-unshaded", "-staticmarkers")

		if summary != plainSummary {
			t.Errorf("-staticmarkers changed the summary:\n  %q\nwant\n  %q", summary, plainSummary)
		}
		for _, tc := range []struct {
			label string
			c     color.RGBA
		}{{"A", staticColorA}, {"B", staticColorB}} {
			if n := countColor(img, tc.c); n != 0 {
				t.Errorf("-staticmarkers alone painted %d pixels of sprite %s", n, tc.label)
			}
		}
		// Both resolving cells are marked, and neither of the two skipped ones is.
		for _, at := range []image.Point{staticFlatGroundA, staticFlatGroundB} {
			if got := colorAt(img, at.X, at.Y); got != terrain.StaticMarkerColor {
				t.Errorf("cell cross at %v = %v, want %v", at, got, terrain.StaticMarkerColor)
			}
		}
		// A radius-3, thickness-1 cross is 2*(2*3+1)*1 - 1 = 13 px at native
		// scale, so exactly two crosses is exactly 26 pixels: the artless class
		// and the byte naming no class are marked NOWHERE.
		if got, want := countColor(img, terrain.StaticMarkerColor), 2*13; got != want {
			t.Errorf("%d static-marker pixels, want %d — only the two cells that resolve are marked", got, want)
		}
	})

	t.Run("all three glyphs draw over the art, in their shipped order", func(t *testing.T) {
		shippedSummary, _ := runTool(t, dir, mapPath, "shipped-two", "-unshaded", "-objects", "-units")
		summary, img := runTool(t, dir, mapPath, "all", "-unshaded",
			"-statics", "-objects", "-units", "-staticmarkers")

		want := strings.TrimSuffix(plainSummary, "\n") +
			fmt.Sprintf(", statics %d (animated 0), objects 1, units 1\n", staticPlacementCount)
		if summary != want {
			t.Errorf("summary =\n  %q\nwant\n  %q", summary, want)
		}

		// Cell (1,0) carries an object record, a unit record and a static
		// placement, so its three crosses are concentric on (48,16) — which is
		// also sprite A's own ground point. The static cross is a strict subset
		// of the unit cross and that of the object cross, so a correct order
		// leaves one ring of each visible:
		//
		//	(42,16) only the object arm reaches   -> yellow
		//	(44,16) the unit arm reaches too      -> cyan
		//	(48,16) all three                     -> red, drawn last
		for _, tc := range []struct {
			label string
			at    image.Point
			want  color.RGBA
		}{
			{"the object arm over the sprite", image.Pt(42, 16), terrain.MarkerColor},
			{"the unit arm over the object arm", image.Pt(44, 16), terrain.UnitMarkerColor},
			{"the static cross over both", staticFlatGroundA, terrain.StaticMarkerColor},
			{"the static cross on B's cell", staticFlatGroundB, terrain.StaticMarkerColor},
			{"the overlap, still the later row's", staticShared, staticColorB},
		} {
			if got := colorAt(img, tc.at.X, tc.at.Y); got != tc.want {
				t.Errorf("%s: pixel %v = %v, want %v", tc.label, tc.at, got, tc.want)
			}
		}

		_, markersOnly := runTool(t, dir, mapPath, "markers-only", "-unshaded",
			"-objects", "-units", "-staticmarkers")
		for _, tc := range []struct {
			label string
			c     color.RGBA
		}{
			{"object", terrain.MarkerColor},
			{"unit", terrain.UnitMarkerColor},
			{"static", terrain.StaticMarkerColor},
		} {
			got, want := countColor(img, tc.c), countColor(markersOnly, tc.c)
			if want == 0 {
				t.Fatalf("fixture: the %s glyph covers no pixel at all", tc.label)
			}
			if got != want {
				t.Errorf("%d %s-marker pixels with the art on, want the %d the same glyphs drew without it",
					got, tc.label, want)
			}
		}
		if strings.Contains(shippedSummary, "statics") {
			t.Errorf("-objects -units without the new flags mentions statics: %q", shippedSummary)
		}
	})
}

func TestRenderStaticsCrossStandsOnItsOwnBase(t *testing.T) {
	dir := t.TempDir()
	flatMap, slopedMap := writeStaticFixtures(t, dir)

	for _, tc := range []struct {
		name    string
		mapPath string
		args    []string
		groundA image.Point
		groundB image.Point
	}{
		{
			name: "flat geometry", mapPath: flatMap, args: []string{"-flat"},
			groundA: staticFlatGroundA, groundB: staticFlatGroundB,
		},
		{
			// slopedAltitudes gives cell (1,0) a four-corner mean of 5 and cell
			// (1,1) one of 0, on a canvas whose origin is slopedOriginY = -20 —
			// so the two sprites move by different amounts and a run that
			// ignored either term lands on neither point.
			name: "projected geometry", mapPath: slopedMap, args: nil,
			groundA: staticSlopedGroundA, groundB: staticSlopedGroundB,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			args := append([]string{"-unshaded"}, tc.args...)
			_, art := runTool(t, dir, tc.mapPath, "base-art-"+strings.ReplaceAll(tc.name, " ", "_"),
				append(append([]string{}, args...), "-statics")...)
			_, cross := runTool(t, dir, tc.mapPath, "base-cross-"+strings.ReplaceAll(tc.name, " ", "_"),
				append(append([]string{}, args...), "-staticmarkers")...)

			for _, p := range []struct {
				label  string
				at     image.Point
				sprite color.RGBA
			}{
				{"A", tc.groundA, staticColorA},
				{"B", tc.groundB, staticColorB},
			} {
				if got := colorAt(art, p.at.X, p.at.Y); got != p.sprite {
					t.Errorf("sprite %s does not cover its own ground point %v: %v, want %v",
						p.label, p.at, got, p.sprite)
				}
				if got := colorAt(cross, p.at.X, p.at.Y); got != terrain.StaticMarkerColor {
					t.Errorf("the cross for %s's cell is not on %v: %v, want %v",
						p.label, p.at, got, terrain.StaticMarkerColor)
				}
			}
		})
	}
}

func TestRenderStaticsFlaglessOutputUnchanged(t *testing.T) {
	dir := t.TempDir()
	flatMap, slopedMap := writeStaticFixtures(t, dir)

	for _, m := range []struct {
		name string
		path string
	}{
		{"flat map", flatMap},
		{"sloped map", slopedMap},
	} {
		for _, scale := range []int{1, 2, 3} {
			for _, geometry := range []bool{false, true} {
				for _, unshaded := range []bool{false, true} {
					name := fmt.Sprintf("%s flat=%v unshaded=%v scale=%d", m.name, geometry, unshaded, scale)
					t.Run(name, func(t *testing.T) {
						out := filepath.Join(dir, strings.NewReplacer(" ", "_", "=", "-").Replace(name)+".png")
						argv := []string{"render", "-assets", dir, "-map", m.path, "-out", out,
							"-scale", fmt.Sprint(scale)}
						if geometry {
							argv = append(argv, "-flat")
						}
						if unshaded {
							argv = append(argv, "-unshaded")
						}
						var stdout bytes.Buffer
						if err := run(argv, &stdout); err != nil {
							t.Fatalf("run: %v", err)
						}
						if strings.Contains(stdout.String(), "statics") {
							t.Errorf("flagless summary mentions statics: %q", stdout.String())
						}

						got, err := os.ReadFile(out)
						if err != nil {
							t.Fatal(err)
						}
						oracle := compositeOracle(t, dir, m.path, geometry, unshaded, scale)
						if want := pngBytes(t, oracle.Image); !bytes.Equal(got, want) {
							t.Errorf("the flagless render is %d bytes, want the compositor's own %d",
								len(got), len(want))
						}
					})
				}
			}
		}
	}
}

// --- 0044: the resolved sun reaches the sprites --------------------------
//
// New assertions for docs/0044-sprite-lighting/T3 (AC-8, SC-6).

type spriteLitWant struct {
	row  int
	a, b color.RGBA
}

var spriteLitRows = map[int]spriteLitWant{
	3:  {3, color.RGBA{R: 26, G: 52, B: 78, A: 0xff}, color.RGBA{R: 104, G: 130, B: 156, A: 0xff}},
	8:  {8, staticColorA, staticColorB},
	15: {15, color.RGBA{R: 2, G: 4, B: 6, A: 0xff}, color.RGBA{R: 8, G: 10, B: 12, A: 0xff}},
}

// TestRenderStaticsTakeTheResolvedSunsRow — AC-8, SC-6.
//
// Four renders of one map: the default sun (ambient 0x0e, row 3), the same
// render under -unshaded, and two more at -ambient 32 and -ambient 60 (rows 8
// and 15). Each lit render is PINNED at its own row's hand-computed colours
// rather than compared only for inequality, which is the assertion a tool that
// lit at some other row would fail while an inequality test passed.
//
// AC-8 also asks that "the two lit renders differ from it and from each
// other". The second half holds; the first does not, and cannot: at -ambient
// 32 the row is 8, whose gain is exactly 1.0, so those sprite pixels ARE the
// raw palette and are byte-identical to the unshaded render's. The -ambient
// 60 render is what carries the criterion's discriminating content: a lit
// render that differs both from the raw palette and from the other lit rows.
func TestRenderStaticsTakeTheResolvedSunsRow(t *testing.T) {
	dir := t.TempDir()
	mapPath, _ := writeStaticFixtures(t, dir)

	// The three sample points 0017 already pins, each inside a sprite and out of
	// every marker glyph's reach.
	type sample struct {
		label string
		at    image.Point
		which func(spriteLitWant) color.RGBA
	}
	samples := []sample{
		{"inside A alone", staticOnlyA, func(w spriteLitWant) color.RGBA { return w.a }},
		{"inside B alone", staticOnlyB, func(w spriteLitWant) color.RGBA { return w.b }},
		{"the overlap, owned by the later row", staticShared, func(w spriteLitWant) color.RGBA { return w.b }},
	}

	// The unshaded render: raw palette, and the summary's own token.
	unshadedSummary, unshaded := runTool(t, dir, mapPath, "unshaded", "-statics", "-unshaded")
	if !strings.Contains(unshadedSummary, "unshaded") {
		t.Errorf("the unshaded summary lost its token: %q", unshadedSummary)
	}
	for _, s := range samples {
		want := s.which(spriteLitWant{a: staticColorA, b: staticColorB})
		if got := colorAt(unshaded, s.at.X, s.at.Y); got != want {
			t.Errorf("-unshaded: %s at %v = %v, want the raw palette's %v", s.label, s.at, got, want)
		}
	}

	// The three lit renders. -ambient 14 is the default sun spelt out, so the
	// flagless shaded run and this one must agree; the descriptor keeps its own
	// spelling in every case.
	lit := map[int]image.Image{}
	for _, tc := range []struct {
		name    string
		ambient int
		row     int
	}{
		{"default", 14, 3},
		{"identity", 32, 8},
		{"darkest", 60, 15},
	} {
		summary, img := runTool(t, dir, mapPath, tc.name, "-statics", "-ambient", fmt.Sprint(tc.ambient))
		if want := fmt.Sprintf("shaded (theta=0.7854 ambient=%d range=32)", tc.ambient); !strings.Contains(summary, want) {
			t.Errorf("%s: summary %q does not carry %q", tc.name, summary, want)
		}
		if !strings.Contains(summary, fmt.Sprintf(", statics %d (animated 0)", staticPlacementCount)) {
			t.Errorf("%s: summary %q lost its statics token", tc.name, summary)
		}
		w := spriteLitRows[tc.row]
		for _, s := range samples {
			if got := colorAt(img, s.at.X, s.at.Y); got != s.which(w) {
				t.Errorf("%s (ambient %d, row %d): %s at %v = %v, want %v",
					tc.name, tc.ambient, tc.row, s.label, s.at, got, s.which(w))
			}
		}
		lit[tc.row] = img
	}

	// The relations AC-8 states, each named for what it is.
	sameSprites := func(a, b image.Image) bool {
		for _, s := range samples {
			if colorAt(a, s.at.X, s.at.Y) != colorAt(b, s.at.X, s.at.Y) {
				return false
			}
		}
		return true
	}
	if sameSprites(lit[3], unshaded) {
		t.Error("the default sun's sprites equal the raw palette; row 3 is gain 1.625, not 1.0")
	}
	if sameSprites(lit[15], unshaded) {
		t.Error("the darkest row's sprites equal the raw palette")
	}
	if sameSprites(lit[3], lit[8]) || sameSprites(lit[3], lit[15]) || sameSprites(lit[8], lit[15]) {
		t.Error("two lit renders at different rows drew the same sprite pixels")
	}
	if !sameSprites(lit[8], unshaded) {
		t.Error("row 8 does not reproduce the raw palette; gain 1.0 must be the identity")
	}

	_, byDefault := runTool(t, dir, mapPath, "bydefault", "-statics")
	if !sameSprites(byDefault, lit[3]) {
		t.Error("the flagless shaded run's sprites differ from -ambient 14's; the default sun is 0x0e")
	}
}

func TestRenderStaticsLightingIsNotGatedOnGeometry(t *testing.T) {
	dir := t.TempDir()
	flatMap, slopedMap := writeStaticFixtures(t, dir)

	w := spriteLitRows[3]
	for _, tc := range []struct {
		name string
		path string
		at   image.Point
		args []string
	}{
		{"flat geometry, flat map", flatMap, staticOnlyA, []string{"-flat"}},
		{"projected geometry, flat map", flatMap, staticOnlyA, nil},
		{"flat geometry, sloped map", slopedMap, staticOnlyA, []string{"-flat"}},
	} {
		args := append([]string{"-statics"}, tc.args...)
		_, img := runTool(t, dir, tc.path, "geom-"+strings.ReplaceAll(tc.name, " ", "-"), args...)
		if got := colorAt(img, tc.at.X, tc.at.Y); got != w.a {
			t.Errorf("%s: sprite A at %v = %v, want row 3's %v — relief must not move a sprite's row",
				tc.name, tc.at, got, w.a)
		}
	}
}
