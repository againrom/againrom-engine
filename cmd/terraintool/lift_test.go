package main

// New assertions for docs/0015-height-projected-placements/T3: the wiring in
// run() that builds a Projection and lifts each marker's own anchor cell
// onto the relief it reaches.
//
// main_test.go's own TestRenderMarkersAtProjectedOrigin is corrected in place
// to carry the lift (0015 plan SC-4: the second and last existing test file
// this story may touch); every NEW assertion lives here instead, so this
// story's T3 commit stays checkable by git show --numstat alone.
//
// Every fixture below is synthetic, built the same way every other fixture in
// this package is (buildALM/buildArchive over the documented format
// contracts, reusing writeArchive/writeSlopedFixtures/buildALM from
// main_test.go); nothing here reads a game install.

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"againrom/pkg/formats/alm"
	"againrom/pkg/render/terrain"
)

// writeClipFixture lays a synthetic 2x2-cell map — the shared archive plus a
// hand-built altitude grid — with a single placed object at objectCell. Every
// number the two clip tests below check is worked out by hand from this
// grid, never read back from the code under test.
func writeClipFixture(t *testing.T, dir, name string, altitudes []uint8, objectCell [2]int) (mapPath string) {
	t.Helper()
	writeArchive(t, dir)
	tiles := make([]uint16, 4) // 2x2 cells, all tileWord(0,0,0): present land, sub 0
	mapPath = filepath.Join(dir, name)
	data := buildALM(2, 2, tiles, altitudes, [][2]int{objectCell}, nil)
	if err := os.WriteFile(mapPath, data, 0o644); err != nil {
		t.Fatal(err)
	}
	return mapPath
}

// TestRenderMarkerLiftClipsAtUnliftedTopExtent - SC-7, AC-7: an anchor whose
// own lift carries its marker past the map's OWN top edge clips there, at
// the extent offsetY alone describes — never at one the lift itself moved.
//
// altitudes: row 0 = {100,100} (steep), row 1 = {0,0} (flat). Cell (0,0)'s
// four corners are 100,100,0,0, so AnchorHeight(0,0) = 50 (POSITIVE: the spec
// convention lifts a positive height UP the image, i.e. to a SMALLER row).
//
// By hand: Vertex(c,0) = -100 for every c (row 0's own altitude, negated);
// Vertex(c,2) clamps to row 1's all-zero altitude, giving 64. So MinV=-100,
// MaxV=64, and offsetY = -MinV*scale = 100*scale — the output row mapRect's
// OWN top edge sits at, for every scale.
//
// The flat (un-lifted) cross at row 0 centres at cellpx/2 + offsetY =
// 16*scale + 100*scale = 116*scale, comfortably inside. AnchorHeight(0,0)=50
// lifts it UP by 50*scale to 66*scale, whose r=6*scale arm reaches only to
// (66-6)*scale = 60*scale < 100*scale for every scale — never touching the
// map's own top edge, so the whole cross clips away (stage 1). A
// sign-flipped lift instead pushes the cross DOWN to 166*scale, whose arm
// partially survives at the map's own BOTTOM edge instead (a nonzero count),
// which is what tells the two apart.
func TestRenderMarkerLiftClipsAtUnliftedTopExtent(t *testing.T) {
	dir := t.TempDir()
	altitudes := []uint8{100, 100, 0, 0}
	mapPath := writeClipFixture(t, dir, "top.alm", altitudes, [2]int{0, 0})

	for _, scale := range []int{1, 2} {
		t.Run(fmt.Sprintf("scale %d", scale), func(t *testing.T) {
			out := filepath.Join(dir, fmt.Sprintf("top-s%d.png", scale))
			var buf bytes.Buffer
			if err := run([]string{"render", "-assets", dir, "-map", mapPath,
				"-out", out, "-scale", fmt.Sprint(scale), "-objects"}, &buf); err != nil {
				t.Fatalf("run: %v", err)
			}
			img := decodePNG(t, out)

			if n := countColor(img, terrain.MarkerColor); n != 0 {
				t.Errorf("scale %d: marker holds %d pixels, want 0 — an anchor lifted entirely past the map's own (unlifted) top edge must clip away completely (SC-7, AC-7)", scale, n)
			}
		})
	}
}

// TestRenderMarkerLiftClipsAtUnliftedBottomExtent - the AC-7-required mirror
// of the test above: a bottom-edge anchor lifted DOWNWARD by a NEGATIVE
// AnchorHeight, which a suite that only ever lifts upward would never catch.
//
// altitudes: row 0 = {0,0} (flat), row 1 = {-100,-100} (steep; 0x9c read
// signed). Cell (0,1) is the last row, so its "far" corners clamp back onto
// row 1 too: all four corners read -100, giving AnchorHeight(0,1) = -100.
//
// By hand: Vertex(c,0) = 0, Vertex(c,1) = 132, Vertex(c,2) = 164 (row 2
// clamps to row 1's altitude). MinV=0, so offsetY=0 and mapRect's own bottom
// edge sits at rows*cellpx = 64*scale.
//
// The flat cross at row 1 centres at cellpx + cellpx/2 = 48*scale, well
// inside. AnchorHeight=-100 lifts it DOWN by 100*scale to 148*scale, whose
// r=6*scale arm starts at (148-6)*scale = 142*scale > 64*scale for every
// scale — entirely past the map's own bottom edge, clipped away by stage 1.
func TestRenderMarkerLiftClipsAtUnliftedBottomExtent(t *testing.T) {
	dir := t.TempDir()
	altitudes := []uint8{0, 0, 0x9c, 0x9c}
	mapPath := writeClipFixture(t, dir, "bottom.alm", altitudes, [2]int{0, 1})

	for _, scale := range []int{1, 2} {
		t.Run(fmt.Sprintf("scale %d", scale), func(t *testing.T) {
			out := filepath.Join(dir, fmt.Sprintf("bottom-s%d.png", scale))
			var buf bytes.Buffer
			if err := run([]string{"render", "-assets", dir, "-map", mapPath,
				"-out", out, "-scale", fmt.Sprint(scale), "-objects"}, &buf); err != nil {
				t.Fatalf("run: %v", err)
			}
			img := decodePNG(t, out)

			if n := countColor(img, terrain.MarkerColor); n != 0 {
				t.Errorf("scale %d: marker holds %d pixels, want 0 — an anchor lifted entirely past the map's own (unlifted) bottom edge must clip away completely (SC-7 mirror, AC-7)", scale, n)
			}
		})
	}
}

// TestRenderMarkerLiftTranslatesInteriorAnchorBySignedScaledAmount is about
// the SIGN and the SCALE MULTIPLY in main.go's own liftY closure, not about
// clipping (the two tests above own that): an anchor far from every mesh edge
// so its lifted cross always stays inside both the map extent and the canvas,
// at every scale tested.
func TestRenderMarkerLiftTranslatesInteriorAnchorBySignedScaledAmount(t *testing.T) {
	dir := t.TempDir()
	writeArchive(t, dir)

	altitudes := []uint8{
		0, 0, 0, 0,
		0, 30, 30, 0,
		0, 30, 30, 0,
		0, 0, 0, 0,
	}
	tiles := make([]uint16, 16)
	mapPath := filepath.Join(dir, "interior.alm")
	if err := os.WriteFile(mapPath, buildALM(4, 4, tiles, altitudes, [][2]int{{1, 1}}, nil), 0o644); err != nil {
		t.Fatal(err)
	}

	for _, scale := range []int{1, 2} {
		t.Run(fmt.Sprintf("scale %d", scale), func(t *testing.T) {
			out := filepath.Join(dir, fmt.Sprintf("interior-s%d.png", scale))
			var buf bytes.Buffer
			if err := run([]string{"render", "-assets", dir, "-map", mapPath,
				"-out", out, "-scale", fmt.Sprint(scale), "-objects"}, &buf); err != nil {
				t.Fatalf("run: %v", err)
			}
			img := decodePNG(t, out)

			cellpx := terrain.CellSize * scale
			cx := 1*cellpx + cellpx/2
			cyFlat := 1*cellpx + cellpx/2 // offsetY is 0 on this fixture (MinV=0)
			cyLifted := cyFlat - 30*scale // -AnchorHeight(1,1)*scale = -30*scale

			if got := colorAt(img, cx, cyLifted); got != terrain.MarkerColor {
				t.Errorf("scale %d: pixel (%d,%d) = %v, want the marker %v at the lifted centre (offset -AnchorHeight*scale = %d)",
					scale, cx, cyLifted, got, terrain.MarkerColor, -30*scale)
			}
			// A dropped sign (would place the cross at cyFlat+30*scale), a
			// dropped scale multiply (would leave it at cyFlat-30, only
			// right at scale 1) or a missing lift altogether (would leave it
			// at cyFlat) must all miss this checkpoint: at scale 2 the
			// nearest of those wrong centres is still 30 native px away,
			// beyond the object cross's own r=12 reach there.
			if got := colorAt(img, cx, cyFlat); got == terrain.MarkerColor {
				t.Errorf("scale %d: pixel (%d,%d) = %v at the UN-lifted centre: the height offset was not applied", scale, cx, cyFlat, got)
			}
		})
	}
}

func TestRenderFlatOutputNeverLiftsMarkers(t *testing.T) {
	dir := t.TempDir()
	_, mapPath := writeSlopedFixtures(t, dir)

	mapData, err := os.ReadFile(mapPath)
	if err != nil {
		t.Fatal(err)
	}
	m, err := alm.Open(mapData)
	if err != nil {
		t.Fatal(err)
	}
	objCells := anchorCells(m)
	unitCellsFixture := unitAnchorCells(m)

	for _, scale := range []int{1, 2, 3} {
		for _, unshaded := range []bool{false, true} {
			t.Run(fmt.Sprintf("scale=%d unshaded=%v", scale, unshaded), func(t *testing.T) {
				out := filepath.Join(dir, fmt.Sprintf("flat-s%d-u%v.png", scale, unshaded))
				args := []string{"render", "-assets", dir, "-map", mapPath,
					"-out", out, "-scale", fmt.Sprint(scale), "-flat", "-objects", "-units"}
				if unshaded {
					args = append(args, "-unshaded")
				}
				var buf bytes.Buffer
				if err := run(args, &buf); err != nil {
					t.Fatalf("run: %v", err)
				}
				got, err := os.ReadFile(out)
				if err != nil {
					t.Fatal(err)
				}

				oracle := compositeOracle(t, dir, mapPath, true, unshaded, scale)
				cellpx := terrain.CellSize * scale
				terrain.DrawObjectMarkersAt(oracle.Image, objCells, m.Width, m.Height, cellpx, 0)
				terrain.DrawUnitMarkersAt(oracle.Image, unitCellsFixture, m.Width, m.Height, cellpx, 0)
				want := pngBytes(t, oracle.Image)

				if !bytes.Equal(got, want) {
					t.Errorf("scale=%d unshaded=%v: -flat output with -objects -units differs from the unlifted oracle (%d vs %d bytes) — AnchorHeight must never be called on the -flat path (SC-8, DD-4)",
						scale, unshaded, len(got), len(want))
				}
			})
		}
	}
}

func TestRenderComposedOverlaysStackOnLiftedMarkers(t *testing.T) {
	dir := t.TempDir()
	_, mapPath := writeSlopedFixtures(t, dir)

	out := filepath.Join(dir, "composed.png")
	var buf bytes.Buffer
	if err := run([]string{"render", "-assets", dir, "-map", mapPath,
		"-out", out, "-objects", "-units"}, &buf); err != nil {
		t.Fatalf("run: %v", err)
	}
	img := decodePNG(t, out)

	const cellpx = terrain.CellSize // scale 1
	const liftY = 8                 // -AnchorHeight(2,2)*scale = -(-8)*1
	cx := 2*cellpx + cellpx/2
	cy := 2*cellpx + cellpx/2 - slopedOriginY + liftY // -slopedOriginY == -OriginY*scale at scale 1

	if got := colorAt(img, cx, cy); got != terrain.UnitMarkerColor {
		t.Errorf("lifted overlap centre (%d,%d) = %v, want the unit colour %v (units drawn after objects, FR-5)", cx, cy, got, terrain.UnitMarkerColor)
	}
	// A point within the object's wider reach (r=6) but outside the unit's
	// narrower one (r=4) must still read the object's yellow: the unit cross
	// does not blot the object one out everywhere, only where the two
	// glyphs actually coincide.
	if got := colorAt(img, cx, cy-5); got != terrain.MarkerColor {
		t.Errorf("object-only pixel (%d,%d) = %v, want the object colour %v", cx, cy-5, got, terrain.MarkerColor)
	}
}
