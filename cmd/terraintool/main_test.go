package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"againrom/pkg/formats/alm"
	"againrom/pkg/game"
	"againrom/pkg/render/terrain"
)

// --- synthetic fixtures: a .res archive of terrain strips and an .alm map ---
//
// Everything below is built byte-by-byte in test code from the documented
// format contracts (.res per 0001, .alm per 0003, BMP per 0004). No game file
// is read and none of these bytes come from an install.

const bmpPixelOffset = 14 + 40 + 256*4

func buildBMP8(w, h int, topDown []byte) []byte {
	stride := (w + 3) &^ 3
	out := make([]byte, bmpPixelOffset+stride*h)
	out[0x00], out[0x01] = 'B', 'M'
	binary.LittleEndian.PutUint32(out[0x02:], uint32(len(out)))
	binary.LittleEndian.PutUint32(out[0x0a:], uint32(bmpPixelOffset))
	binary.LittleEndian.PutUint32(out[0x0e:], 40)
	binary.LittleEndian.PutUint32(out[0x12:], uint32(w))
	binary.LittleEndian.PutUint32(out[0x16:], uint32(h))
	binary.LittleEndian.PutUint16(out[0x1a:], 1)
	binary.LittleEndian.PutUint16(out[0x1c:], 8)
	for i := 0; i < 256; i++ {
		e := out[54+i*4:]
		e[0], e[1], e[2] = byte(i*7+3), byte(255-i), byte(i) // stored B, G, R
	}
	for y := 0; y < h; y++ {
		copy(out[bmpPixelOffset+(h-1-y)*stride:], topDown[y*w:(y+1)*w])
	}
	return out
}

// rampColor is the colour palette index i decodes to, given buildBMP8's ramp.
func rampColor(i byte) color.RGBA {
	return color.RGBA{R: i, G: 255 - i, B: i*7 + 3, A: 0xff}
}

// solidStrip builds a 32 x (32*cells) tile strip whose sub-cell k is filled with
// palette index base+k.
func solidStrip(cells int, base byte) []byte {
	const size = 32
	h := size * cells
	idx := make([]byte, size*h)
	for k := 0; k < cells; k++ {
		for y := k * size; y < (k+1)*size; y++ {
			for x := 0; x < size; x++ {
				idx[y*size+x] = base + byte(k)
			}
		}
	}
	return buildBMP8(size, h, idx)
}

type archiveEntry struct {
	name string // leaf name inside the terrain directory
	data []byte
}

// buildArchive assembles a .res (&YA1) archive holding one "terrain"
// directory node whose children are the given files: a 24-byte header, the
// payloads, then a registry of 32-byte nodes (RES-HDR-002..RES-NODE-008).
func buildArchive(entries []archiveEntry) []byte {
	const headerSize, nodeSize = 24, 32

	out := make([]byte, headerSize)
	offs := make([][2]uint32, len(entries)) // {off, size}
	for i, e := range entries {
		offs[i] = [2]uint32{uint32(len(out)), uint32(len(e.data))}
		out = append(out, e.data...)
	}
	regOffset := uint32(len(out))

	binary.LittleEndian.PutUint32(out[0x00:], 0x31415926) // "&YA1"
	binary.LittleEndian.PutUint32(out[0x04:], 0)
	binary.LittleEndian.PutUint32(out[0x08:], 1) // one root
	binary.LittleEndian.PutUint32(out[0x0c:], 1) // root node is a directory
	binary.LittleEndian.PutUint32(out[0x10:], regOffset)
	binary.LittleEndian.PutUint32(out[0x14:], uint32(1+len(entries)))

	node := func(off, size, typ uint32, name string) []byte {
		n := make([]byte, nodeSize)
		binary.LittleEndian.PutUint32(n[0x04:], off)
		binary.LittleEndian.PutUint32(n[0x08:], size)
		binary.LittleEndian.PutUint32(n[0x0c:], typ)
		copy(n[0x10:0x20], name) // ASCII leaf names; NUL-padded by the zero fill
		return n
	}
	// Node 0 is the directory; its children are nodes 1..len(entries).
	out = append(out, node(1, uint32(len(entries)), 1, "terrain")...)
	for i, e := range entries {
		out = append(out, node(offs[i][0], offs[i][1], 0, e.name)...)
	}
	return out
}

func le32(v uint32) []byte {
	b := make([]byte, 4)
	binary.LittleEndian.PutUint32(b, v)
	return b
}

func concat(parts ...[]byte) []byte {
	var out []byte
	for _, p := range parts {
		out = append(out, p...)
	}
	return out
}

// objectRecord builds one 20-byte type4 placed-object record anchored at cell
// (col, row). The stored coordinates are /256 fixed-point, and the low byte is
// deliberately non-zero so the decoder's >>8 truncation is genuinely exercised
// rather than passing on exact multiples.
func objectRecord(col, row int) []byte {
	rec := make([]byte, 20)
	binary.LittleEndian.PutUint32(rec[0x00:], uint32(col)<<8|0x40)
	binary.LittleEndian.PutUint32(rec[0x04:], uint32(row)<<8|0x80)
	return rec // kind (+0x08) stays 0 (!= 0x21), so no extension follows
}

// unitRecord builds one 70-byte type6 placed-unit record anchored at cell
// (col, row). The remaining 62 bytes are undecoded (0003 R-2) and stay zero;
// nothing reads them.
func unitRecord(col, row int) []byte {
	rec := make([]byte, 70)
	binary.LittleEndian.PutUint32(rec[0x00:], uint32(col)<<8|0x80)
	binary.LittleEndian.PutUint32(rec[0x04:], uint32(row)<<8|0x7f)
	return rec
}

func buildALM(w, h int, tiles []uint16, altitudes []uint8, objects, units [][2]int) []byte {
	cells := w * h

	meta := make([]byte, 632)
	binary.LittleEndian.PutUint32(meta[0x00:], uint32(w))
	binary.LittleEndian.PutUint32(meta[0x04:], uint32(h))
	binary.LittleEndian.PutUint32(meta[0x1c:], 1)                    // #type5
	binary.LittleEndian.PutUint32(meta[0x20:], uint32(len(objects))) // #type4
	binary.LittleEndian.PutUint32(meta[0x24:], uint32(len(units)))   // #type6

	tileBytes := make([]byte, 2*cells)
	for i, w16 := range tiles {
		binary.LittleEndian.PutUint16(tileBytes[i*2:], w16)
	}

	grid8 := func() []byte { return make([]byte, cells) }

	altitudeGrid := grid8()
	if altitudes != nil {
		if len(altitudes) != cells {
			panic("buildALM: altitude grid must hold W*H bytes")
		}
		copy(altitudeGrid, altitudes)
	}

	var object []byte
	for _, o := range objects {
		object = append(object, objectRecord(o[0], o[1])...)
	}
	group := make([]byte, 76)
	copy(group[0x0c:], "Self")

	var unit []byte
	for _, u := range units {
		unit = append(unit, unitRecord(u[0], u[1])...)
	}

	var payloads [10][]byte
	payloads[0] = meta
	payloads[1] = tileBytes
	payloads[2] = altitudeGrid
	payloads[3] = grid8()
	payloads[4] = object
	payloads[5] = group
	payloads[6] = unit
	payloads[7] = le32(0) // type7: entryCount 0
	payloads[8] = nil     // type8: no records (ALM-TRIG-050; not a marker tree)
	payloads[9] = le32(0) // type9: count 0

	// 20-byte file header: magic, hdrLen 20, dataSize (ignored), recordCount 10,
	// formatVersion 990.
	out := concat(le32(0x0052374D), le32(20), le32(0), le32(10), le32(990))
	for _, typeID := range []int{0, 1, 2, 3, 5, 4, 9, 8, 6, 7} {
		p := payloads[typeID]
		// record header: tag 7, hdrLen 20, payloadSize, typeId, perMapConst 0.
		out = concat(out, le32(7), le32(20), le32(uint32(len(p))), le32(uint32(typeID)), le32(0), p)
	}
	return out
}

// tileWord assembles a type1 tile word from strip group, blend column and
// sub-cell.
func tileWord(g, b, sub int) uint16 {
	return uint16(g<<6) | uint16(b<<4) | uint16(sub)
}

// Fill bases, one per strip, so a rendered pixel identifies its source.
const (
	landBase  = 1
	waterBase = 100
	dirtBase  = 200
)

// objectCells are the fixture map's placed-object anchors. The first is off-map
// on the 4x3 grid and must therefore be marked nowhere (an off-map anchor is legal
// at any index under the kind==0x21 extension discriminator). The other two sit on
// cells whose centres the other assertions do not sample.
var objectCells = [][2]int{{9, 9}, {0, 0}, {2, 2}}

// unitCells are the fixture map's placed-unit anchors. None of them is
// (1,1), the cell the pre-existing assertions sample as untouched terrain.
var unitCells = [][2]int{{7, 7}, {2, 2}, {3, 0}}

// writeFixtures lays a synthetic graphics.res and map.alm into dir and returns
// their paths. The 4x3 grid covers land, water, an impassable land cell and one
// cell naming an absent slot.
func writeFixtures(t *testing.T, dir string) (archivePath, mapPath string) {
	t.Helper()
	return writeFixturesWithUnits(t, dir, unitCells)
}

func writeFixturesWithUnits(t *testing.T, dir string, units [][2]int) (archivePath, mapPath string) {
	t.Helper()

	archivePath = writeArchive(t, dir)
	mapPath = filepath.Join(dir, "map.alm")
	if err := os.WriteFile(mapPath, buildALM(4, 3, fixtureTiles(), nil, objectCells, units), 0o644); err != nil {
		t.Fatal(err)
	}
	return archivePath, mapPath
}

// writeArchive lays the synthetic graphics.res of terrain strips into dir. Both
// fixtures share it: the tileset is what a cell is drawn FROM, and nothing about
// it varies with the geometry a cell is drawn ON.
func writeArchive(t *testing.T, dir string) string {
	t.Helper()

	path := filepath.Join(dir, "graphics.res")
	archive := buildArchive([]archiveEntry{
		{"tile1-00.bmp", solidStrip(14, landBase)},
		{"tile3-00.bmp", solidStrip(8, waterBase)},
		{"dirt.bmp", solidStrip(4, dirtBase)},
	})
	if err := os.WriteFile(path, archive, 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

// fixtureTiles is the 4x3 tile grid both fixtures carry: cells 0..3 are plain 0
// words (tile index 0, resolving into the present tile1-00) and cells 4..7 are
// the interesting row — land, water, impassable land and one naming an absent
// slot, which is the single placeholder cell every count below expects.
func fixtureTiles() []uint16 {
	return []uint16{
		0, 0, 0, 0,
		tileWord(0, 0, 2),                         // land, sub 2
		tileWord(8, 0, 3),                         // water, sub 3
		tileWord(0, 0, 1) | terrain.ImpassableBit, // impassable land, sub 1
		tileWord(1, 0, 0),                         // tile1-04: absent -> placeholder
		0, 0, 0, 0,
	}
}

// The sloped fixture. It is a SECOND map beside the one above, not altitudes
// bolted onto it: every pre-0012 assertion here samples cell centres on a
// lattice the projection would move, and the all-zero grid is exactly what
// keeps those assertions meaning what they meant. It is also why they can
// say nothing about this story — on an all-zero grid the projected and the
// flat path agree on every pixel, both dimensions and the origin, so a tool
// that never projected at all would pass all of them.
//
// The grid, its vertices and the canvas, worked out by hand from spec 0012 so
// that no number below is read back from the code under test:
//
//	altitudes      V(c,r) = r*32 - h(c,r),  h clamped to the grid
//	 20 20  0  0     r=0:  -20 -20   0   0   0
//	  0  0  0  0     r=1:   32  32  32  32  32
//	  0  0  0 -16    r=2:   64  64  64  80  80
//	                 r=3:   96  96  96 112 112   (row 3 clamps to row 2)
//
// minV = -20, maxV = 112, so the canvas is 132 native rows and the image is
// 128x132 at scale 1 — NOT the flat raster's 128x96 — with y origin -20. Cells
// (0,0), (1,0), (2,1) and (3,1) among others have unequal corner altitudes and
// take the sloped path; (2,0) and the whole of row 1's left half are flat.
const (
	slopedOriginY      = -20 // minV
	slopedCanvasRows   = 132 // maxV - minV
	slopedPlaceholders = 1   // cell (3,1), the absent slot
)

var slopedAltitudes = []uint8{
	20, 20, 0, 0,
	0, 0, 0, 0,
	0, 0, 0, 0xf0, // 0xf0 read signed is -16
}

// writeSlopedFixtures lays the shared archive and the sloped map into dir. The
// tile grid, the object anchors and the unit anchors are the flat fixture's, so
// the ONE thing that differs between the two maps is the altitude grid.
func writeSlopedFixtures(t *testing.T, dir string) (archivePath, mapPath string) {
	t.Helper()

	archivePath = writeArchive(t, dir)
	mapPath = filepath.Join(dir, "sloped.alm")
	if err := os.WriteFile(mapPath, buildALM(4, 3, fixtureTiles(), slopedAltitudes, objectCells, unitCells), 0o644); err != nil {
		t.Fatal(err)
	}
	return archivePath, mapPath
}

func decodePNG(t *testing.T, path string) image.Image {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	img, err := png.Decode(f)
	if err != nil {
		t.Fatalf("decode %s: %v", path, err)
	}
	return img
}

// pixelAt returns the colour at the centre of the scaled cell (col, row).
func pixelAt(img image.Image, col, row, scale int) color.RGBA {
	size := terrain.CellSize * scale
	x := col*size + size/2
	y := row*size + size/2
	return color.RGBAModel.Convert(img.At(x, y)).(color.RGBA)
}

// colorAt returns the colour at one output pixel.
func colorAt(img image.Image, x, y int) color.RGBA {
	return color.RGBAModel.Convert(img.At(x, y)).(color.RGBA)
}

// markerCentre is the output pixel a marker cross is centred on for anchor cell
// (col,row), on an image whose row 0 stands for native row originY.
func markerCentre(col, row, originY, scale int) (x, y int) {
	cellpx := terrain.CellSize * scale
	return col*cellpx + cellpx/2, row*cellpx + cellpx/2 - originY*scale
}

// pngBytes is the PNG encoding of one composited image, for a byte-for-byte
// comparison against a file the tool wrote.
func pngBytes(t *testing.T, img image.Image) []byte {
	t.Helper()
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

// compositeOracle composites a fixture map directly through the render tier,
// by the entry point the tool's two independent flags select between, so a
// routing assertion compares the tool against the compositor rather than
// against another run of itself.
func compositeOracle(t *testing.T, dir, mapPath string, flat, unshaded bool, scale int) *terrain.Render {
	t.Helper()

	// Through a REAL container filesystem over the fixture's graphics.res, which is
	// the route the tool itself takes: the tile constants are addresses carrying
	// that container's identity, so an oracle handed a bare archive would resolve
	// nothing and every comparison below would pass on two empty tilesets.
	containers, err := game.OpenContainers(filepath.Join(dir, "graphics.res"))
	if err != nil {
		t.Fatal(err)
	}
	mapData, err := os.ReadFile(mapPath)
	if err != nil {
		t.Fatal(err)
	}
	m, err := alm.Open(mapData)
	if err != nil {
		t.Fatal(err)
	}

	ts := terrain.LoadTileset(containers)
	g := terrain.Grid{Width: m.Width, Height: m.Height, Tiles: m.Tiles}
	light := resolveLight(m, false, math.NaN(), -1, -1)

	var r *terrain.Render
	switch {
	case flat && unshaded:
		r, err = terrain.Composite(ts, g, scale)
	case flat:
		r, err = terrain.CompositeLit(ts, g, m.Altitudes, light, scale)
	case unshaded:
		r, err = terrain.CompositeProjected(ts, g, m.Altitudes, scale)
	default:
		r, err = terrain.CompositeProjectedLit(ts, g, m.Altitudes, light, scale)
	}
	if err != nil {
		t.Fatalf("oracle composite (flat=%v unshaded=%v scale=%d): %v", flat, unshaded, scale, err)
	}
	return r
}

func objectPixelsPerMarker(cellpx int) int {
	round := func(n int) int { return (n + terrain.CellSize/2) / terrain.CellSize }
	r := max(1, round(6*cellpx))
	t := max(1, round(3*cellpx))
	return 2*(2*r+1)*t - t*t
}

// --- tests ---

func TestRenderEndToEnd(t *testing.T) {
	dir := t.TempDir()
	_, mapPath := writeFixtures(t, dir)
	outPath := filepath.Join(dir, "out.png")

	var stdout bytes.Buffer
	if err := run([]string{"render", "-unshaded", "-assets", dir, "-map", mapPath, "-out", outPath}, &stdout); err != nil {
		t.Fatalf("run: %v", err)
	}

	// The fixture's altitudes are all zero, so the default projected geometry puts
	// every cell exactly where the flat raster does and the canvas is the flat
	// one: same pixels, same 128x96, origin 0. That is what lets every assertion
	// below stand unchanged — and equally what makes none of them evidence about
	// the projection (see writeSlopedFixtures).
	want := "terrain: 4x3 cells (12), 128x96 px at scale 1, tile slots 2/128, placeholder cells 1, unshaded, geometry projected, y origin 0\n"
	if got := stdout.String(); got != want {
		t.Fatalf("summary = %q, want %q", got, want)
	}

	img := decodePNG(t, outPath)
	if b := img.Bounds(); b.Dx() != 128 || b.Dy() != 96 {
		t.Fatalf("image is %v, want 128x96", b)
	}

	// Row 1 holds the interesting cells.
	if got, want := pixelAt(img, 0, 1, 1), rampColor(landBase+2); got != want {
		t.Errorf("land cell = %v, want %v", got, want)
	}
	if got, want := pixelAt(img, 1, 1, 1), rampColor(waterBase+3); got != want {
		t.Errorf("water cell = %v, want %v", got, want)
	}
	if got, want := pixelAt(img, 3, 1, 1), terrain.PlaceholderColor; got != want {
		t.Errorf("absent-slot cell = %v, want the placeholder %v", got, want)
	}

	// The impassable land cell at (2,1) is composited with dirt sub-cell
	// (2 + 1*5) & 3 == 3, so it must match neither the plain tile nor the
	// placeholder.
	impassable := pixelAt(img, 2, 1, 1)
	if impassable == rampColor(landBase+1) {
		t.Error("impassable cell was drawn without its dirt composite")
	}
	if impassable == terrain.PlaceholderColor {
		t.Error("impassable cell fell back to a placeholder")
	}
	// The dirt overlay is transparent-keyed (TERR-DIRT-017): every dirt sub-cell
	// in this fixture is a solid non-zero index, so it replaces the terrain
	// outright rather than mixing with it.
	tile := rampColor(landBase + 1)
	wantDirt := rampColor(byte(dirtBase + terrain.DirtSubCell(2, 1)))
	if wantDirt == tile {
		t.Fatal("fixture assumption: the dirt and tile colours must differ")
	}
	if impassable != wantDirt {
		t.Errorf("impassable cell = %v, want the dirt %v outright (sub-cell %d)",
			impassable, wantDirt, terrain.DirtSubCell(2, 1))
	}
}

func TestRenderShadedDefault(t *testing.T) {
	dir := t.TempDir()
	_, mapPath := writeFixtures(t, dir)
	shadedOut := filepath.Join(dir, "shaded.png")
	flatOut := filepath.Join(dir, "flat.png")

	var stdout bytes.Buffer
	if err := run([]string{"render", "-assets", dir, "-map", mapPath, "-out", shadedOut}, &stdout); err != nil {
		t.Fatalf("shaded run: %v", err)
	}
	if !strings.Contains(stdout.String(), "shaded (theta=0.7854 ambient=14 range=32)") {
		t.Fatalf("summary = %q, want the default daytime shaded descriptor", stdout.String())
	}
	shaded := decodePNG(t, shadedOut)

	stdout.Reset()
	if err := run([]string{"render", "-unshaded", "-assets", dir, "-map", mapPath, "-out", flatOut}, &stdout); err != nil {
		t.Fatalf("unshaded run: %v", err)
	}
	flat := decodePNG(t, flatOut)

	wantShaded := terrain.ShadeRGBA(rampColor(waterBase+3), [3]uint8{0, 0, 0}, 46)
	if wantShaded == rampColor(waterBase+3) {
		t.Fatal("fixture: level 46 must actually attenuate the colour")
	}
	if got := pixelAt(shaded, 1, 1, 1); got != wantShaded {
		t.Errorf("shaded water cell = %v, want %v (level 46, x1.5625)", got, wantShaded)
	}
	if got := pixelAt(flat, 1, 1, 1); got != rampColor(waterBase+3) {
		t.Errorf("unshaded water cell = %v, want the flat palette %v", got, rampColor(waterBase+3))
	}
	// The placeholder cell is a diagnostic fill and stays unshaded in both paths.
	if got := pixelAt(shaded, 3, 1, 1); got != terrain.PlaceholderColor {
		t.Errorf("shaded placeholder = %v, want %v", got, terrain.PlaceholderColor)
	}
}

func TestRenderLightControls(t *testing.T) {
	dir := t.TempDir()
	_, mapPath := writeFixtures(t, dir)

	base := filepath.Join(dir, "base.png")
	bright := filepath.Join(dir, "bright.png")

	var buf bytes.Buffer
	if err := run([]string{"render", "-assets", dir, "-map", mapPath, "-out", base}, &buf); err != nil {
		t.Fatalf("base: %v", err)
	}
	buf.Reset()
	// ambient 40 -> flat level 40+0x20 = 72, darker than the default 46.
	if err := run([]string{"render", "-assets", dir, "-map", mapPath, "-out", bright, "-ambient", "40"}, &buf); err != nil {
		t.Fatalf("-ambient: %v", err)
	}
	if !strings.Contains(buf.String(), "ambient=40") {
		t.Fatalf("summary = %q, want ambient=40", buf.String())
	}
	if pixelAt(decodePNG(t, base), 1, 1, 1) == pixelAt(decodePNG(t, bright), 1, 1, 1) {
		t.Error("-ambient override did not change the shaded output")
	}

	// -maplight (the fixture's stored fields are zero: a valid light) and -theta
	// and -range are accepted and produce a render with the reported light.
	buf.Reset()
	if err := run([]string{"render", "-assets", dir, "-map", mapPath, "-out", filepath.Join(dir, "ml.png"), "-maplight"}, &buf); err != nil {
		t.Fatalf("-maplight: %v", err)
	}
	buf.Reset()
	if err := run([]string{"render", "-assets", dir, "-map", mapPath, "-out", filepath.Join(dir, "th.png"), "-theta", "0.2", "-range", "80"}, &buf); err != nil {
		t.Fatalf("-theta/-range: %v", err)
	}
	if !strings.Contains(buf.String(), "theta=0.2000 ambient=14 range=80") {
		t.Fatalf("summary = %q, want theta=0.2000 ambient=14 range=80", buf.String())
	}
}

func TestRenderScaleAndEnvAssetRoot(t *testing.T) {
	dir := t.TempDir()
	_, mapPath := writeFixtures(t, dir)
	outPath := filepath.Join(dir, "out2.png")

	t.Setenv("AGAINROM_ASSETS", dir)

	var stdout bytes.Buffer
	if err := run([]string{"render", "-unshaded", "-map", mapPath, "-out", outPath, "-scale", "2"}, &stdout); err != nil {
		t.Fatalf("run: %v", err)
	}
	if !strings.Contains(stdout.String(), "256x192 px at scale 2") {
		t.Fatalf("summary = %q, want a 256x192 scale-2 render", stdout.String())
	}
	img := decodePNG(t, outPath)
	if b := img.Bounds(); b.Dx() != 256 || b.Dy() != 192 {
		t.Fatalf("image is %v, want 256x192", b)
	}
	if got, want := pixelAt(img, 1, 1, 2), rampColor(waterBase+3); got != want {
		t.Errorf("scaled water cell = %v, want %v", got, want)
	}
}

func TestRenderFailures(t *testing.T) {
	dir := t.TempDir()
	_, mapPath := writeFixtures(t, dir)
	out := filepath.Join(dir, "unused.png")

	// An empty environment must not fall back to any built-in path.
	t.Setenv("AGAINROM_ASSETS", "")

	for _, tc := range []struct {
		name string
		args []string
	}{
		{"no subcommand", []string{}},
		{"unknown subcommand", []string{"dump"}},
		{"no asset root", []string{"render", "-map", mapPath, "-out", out}},
		{"missing -map", []string{"render", "-assets", dir, "-out", out}},
		{"missing -out", []string{"render", "-assets", dir, "-map", mapPath}},
		{"unknown flag", []string{"render", "-assets", dir, "-map", mapPath, "-out", out, "-zoom", "2"}},
		{"archive not found", []string{"render", "-assets", filepath.Join(dir, "nope"), "-map", mapPath, "-out", out}},
		{"map not found", []string{"render", "-assets", dir, "-map", filepath.Join(dir, "nope.alm"), "-out", out}},
		{"scale below 1", []string{"render", "-assets", dir, "-map", mapPath, "-out", out, "-scale", "0"}},
	} {
		var stdout bytes.Buffer
		if err := run(tc.args, &stdout); err == nil {
			t.Errorf("%s: expected an error, got nil", tc.name)
		}
		if stdout.Len() != 0 {
			t.Errorf("%s: wrote a summary despite failing: %q", tc.name, stdout.String())
		}
	}

	// A malformed map is rejected by the decoder, not rendered.
	badMap := filepath.Join(dir, "bad.alm")
	if err := os.WriteFile(badMap, []byte("not a map"), 0o644); err != nil {
		t.Fatal(err)
	}
	var stdout bytes.Buffer
	if err := run([]string{"render", "-assets", dir, "-map", badMap, "-out", out}, &stdout); err == nil {
		t.Error("malformed map: expected an error, got nil")
	}
}

// countColor reports how many pixels of img hold exactly c.
func countColor(img image.Image, c color.RGBA) int {
	n := 0
	b := img.Bounds()
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			if color.RGBAModel.Convert(img.At(x, y)).(color.RGBA) == c {
				n++
			}
		}
	}
	return n
}

func TestRenderObjectsOverlay(t *testing.T) {
	dir := t.TempDir()
	_, mapPath := writeFixtures(t, dir)

	plainPath := filepath.Join(dir, "plain.png")
	markedPath := filepath.Join(dir, "marked.png")

	var plainOut bytes.Buffer
	if err := run([]string{"render", "-assets", dir, "-map", mapPath, "-out", plainPath}, &plainOut); err != nil {
		t.Fatalf("plain run: %v", err)
	}
	var markedOut bytes.Buffer
	if err := run([]string{"render", "-assets", dir, "-map", mapPath, "-out", markedPath, "-objects"}, &markedOut); err != nil {
		t.Fatalf("-objects run: %v", err)
	}

	wantSummary := strings.TrimSuffix(plainOut.String(), "\n") + fmt.Sprintf(", objects %d\n", len(objectCells))
	if got := markedOut.String(); got != wantSummary {
		t.Errorf("-objects summary = %q, want %q", got, wantSummary)
	}
	// AC-4: absent the flag the summary keeps its pre-0008 shape — no token.
	if strings.Contains(plainOut.String(), "objects") {
		t.Errorf("summary without -objects mentions objects: %q", plainOut.String())
	}

	containers, err := game.OpenContainers(filepath.Join(dir, "graphics.res"))
	if err != nil {
		t.Fatal(err)
	}
	mapData, err := os.ReadFile(mapPath)
	if err != nil {
		t.Fatal(err)
	}
	m, err := alm.Open(mapData)
	if err != nil {
		t.Fatal(err)
	}
	baseline, err := terrain.CompositeProjectedLit(
		terrain.LoadTileset(containers),
		terrain.Grid{Width: m.Width, Height: m.Height, Tiles: m.Tiles},
		m.Altitudes,
		resolveLight(m, false, math.NaN(), -1, -1),
		1,
	)
	if err != nil {
		t.Fatalf("baseline composite: %v", err)
	}
	var wantPNG bytes.Buffer
	if err := png.Encode(&wantPNG, baseline.Image); err != nil {
		t.Fatal(err)
	}
	gotPNG, err := os.ReadFile(plainPath)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(gotPNG, wantPNG.Bytes()) {
		t.Errorf("render without -objects is %d bytes, want the compositor's own %d-byte output",
			len(gotPNG), wantPNG.Len())
	}

	plain := decodePNG(t, plainPath)
	marked := decodePNG(t, markedPath)

	// The marker colour must not occur in the terrain itself, or the pixel count
	// below would not be evidence of the overlay.
	if n := countColor(plain, terrain.MarkerColor); n != 0 {
		t.Fatalf("the unmarked render already holds %d marker-coloured pixels", n)
	}

	// Two in-map anchors, each a cross of 13x3 + 3x13 pixels overlapping in a
	// 3x3 centre: 39 + 39 - 9 = 69 px per marker at the native 32 px/cell. The
	// third object is off-map and must contribute nothing.
	const perMarker = 69
	if got, want := countColor(marked, terrain.MarkerColor), 2*perMarker; got != want {
		t.Errorf("marked render holds %d marker pixels, want %d (2 in-map anchors, the off-map one drawing nothing)", got, want)
	}

	// Each in-map anchor's cell centre is the centre of its cross.
	for _, cell := range [][2]int{{0, 0}, {2, 2}} {
		if got := pixelAt(marked, cell[0], cell[1], 1); got != terrain.MarkerColor {
			t.Errorf("cell (%d,%d) = %v, want the marker %v", cell[0], cell[1], got, terrain.MarkerColor)
		}
	}
	// A cell holding no object keeps its terrain colour.
	if got, want := pixelAt(marked, 1, 1, 1), pixelAt(plain, 1, 1, 1); got != want {
		t.Errorf("unmarked cell (1,1) = %v, want the terrain colour %v", got, want)
	}
}

// TestRenderWithEmptyArchive - AC-7: an archive holding no terrain strips still
// renders, reporting every cell as a placeholder rather than failing.
func TestRenderWithEmptyArchive(t *testing.T) {
	dir := t.TempDir()
	_, mapPath := writeFixtures(t, dir)

	emptyPath := filepath.Join(dir, "empty.res")
	if err := os.WriteFile(emptyPath, buildArchive(nil), 0o644); err != nil {
		t.Fatal(err)
	}
	outPath := filepath.Join(dir, "empty.png")

	var stdout bytes.Buffer
	if err := run([]string{"render", "-graphics", emptyPath, "-map", mapPath, "-out", outPath}, &stdout); err != nil {
		t.Fatalf("run: %v", err)
	}
	if !strings.Contains(stdout.String(), "tile slots 0/128, placeholder cells 12") {
		t.Fatalf("summary = %q, want 0 slots and 12 placeholder cells", stdout.String())
	}
	img := decodePNG(t, outPath)
	if got := pixelAt(img, 0, 0, 1); got != terrain.PlaceholderColor {
		t.Errorf("cell (0,0) = %v, want the placeholder %v", got, terrain.PlaceholderColor)
	}
}

func unitPixelsPerMarker(cellpx int) int {
	round := func(n int) int { return (n + terrain.CellSize/2) / terrain.CellSize }
	r := max(1, round(4*cellpx))
	t := max(1, round(cellpx))
	return 2*(2*r+1)*t - t*t
}

func TestRenderUnitsOverlay(t *testing.T) {
	dir := t.TempDir()
	_, mapPath := writeFixtures(t, dir)

	render := func(name string, args ...string) (string, image.Image) {
		t.Helper()
		out := filepath.Join(dir, name+".png")
		argv := append([]string{"render", "-assets", dir, "-map", mapPath, "-out", out}, args...)
		var buf bytes.Buffer
		if err := run(argv, &buf); err != nil {
			t.Fatalf("%s run: %v", name, err)
		}
		return buf.String(), decodePNG(t, out)
	}

	plainSummary, plain := render("plain")
	unitsSummary, units := render("units", "-units")
	bothSummary, both := render("both", "-objects", "-units")
	objectsSummary, _ := render("objects", "-objects")

	// AC-4: the disabled summary never mentions the overlay.
	if strings.Contains(plainSummary, "units") {
		t.Errorf("summary without -units mentions units: %q", plainSummary)
	}
	if strings.Contains(objectsSummary, "units") {
		t.Errorf("summary with -objects but no -units mentions units: %q", objectsSummary)
	}

	wantUnits := strings.TrimSuffix(plainSummary, "\n") + fmt.Sprintf(", units %d\n", len(unitCells))
	if unitsSummary != wantUnits {
		t.Errorf("-units summary =\n  %q\nwant\n  %q", unitsSummary, wantUnits)
	}

	wantBoth := strings.TrimSuffix(plainSummary, "\n") +
		fmt.Sprintf(", objects %d, units %d\n", len(objectCells), len(unitCells))
	if bothSummary != wantBoth {
		t.Errorf("-objects -units summary =\n  %q\nwant\n  %q", bothSummary, wantBoth)
	}
	if oi, ui := strings.Index(bothSummary, ", objects "), strings.Index(bothSummary, ", units "); oi < 0 || ui < 0 || oi > ui {
		t.Errorf("FR-5: the object count must precede the unit count, got %q", bothSummary)
	}

	reordered, _ := render("reordered", "-units", "-objects")
	if reordered != bothSummary {
		t.Errorf("summary depends on CLI flag order:\n  -units -objects = %q\n  -objects -units = %q",
			reordered, bothSummary)
	}

	if n := countColor(plain, terrain.UnitMarkerColor); n != 0 {
		t.Fatalf("the unmarked render already holds %d unit-marker pixels", n)
	}
	inMap := 0
	for _, c := range unitCells {
		if c[0] < 4 && c[1] < 3 {
			inMap++
			if got := pixelAt(units, c[0], c[1], 1); got != terrain.UnitMarkerColor {
				t.Errorf("unit cell (%d,%d) = %v, want the unit marker %v",
					c[0], c[1], got, terrain.UnitMarkerColor)
			}
		}
	}
	if got, want := countColor(units, terrain.UnitMarkerColor), inMap*unitPixelsPerMarker(terrain.CellSize); got != want {
		t.Errorf("-units render holds %d unit-marker pixels, want %d (%d in-map anchors, the off-map one drawing nothing)",
			got, want, inMap)
	}

	if got := pixelAt(both, 2, 2, 1); got != terrain.UnitMarkerColor {
		t.Errorf("cell (2,2) holds an object and a unit; centre = %v, want the unit marker %v on top",
			got, terrain.UnitMarkerColor)
	}
	if got, want := countColor(both, terrain.UnitMarkerColor), inMap*unitPixelsPerMarker(terrain.CellSize); got != want {
		t.Errorf("composed render holds %d unit-marker pixels, want %d — the unit pass must not be occluded", got, want)
	}

	// R-6: the reachable non-native scale, where the arm thickness is even.
	_, s2 := render("units-s2", "-units", "-scale", "2")
	if got, want := countColor(s2, terrain.UnitMarkerColor), inMap*unitPixelsPerMarker(2*terrain.CellSize); got != want {
		t.Errorf("-scale 2 render holds %d unit-marker pixels, want %d", got, want)
	}
	for _, c := range unitCells {
		if c[0] < 4 && c[1] < 3 {
			if got := pixelAt(s2, c[0], c[1], 2); got != terrain.UnitMarkerColor {
				t.Errorf("at -scale 2, unit cell (%d,%d) = %v, want the unit marker", c[0], c[1], got)
			}
		}
	}

	baseline := func(withObjects bool) []byte {
		t.Helper()
		containers, err := game.OpenContainers(filepath.Join(dir, "graphics.res"))
		if err != nil {
			t.Fatal(err)
		}
		mapData, err := os.ReadFile(mapPath)
		if err != nil {
			t.Fatal(err)
		}
		m, err := alm.Open(mapData)
		if err != nil {
			t.Fatal(err)
		}
		r, err := terrain.CompositeProjectedLit(
			terrain.LoadTileset(containers),
			terrain.Grid{Width: m.Width, Height: m.Height, Tiles: m.Tiles},
			m.Altitudes,
			resolveLight(m, false, math.NaN(), -1, -1),
			1,
		)
		if err != nil {
			t.Fatalf("baseline composite: %v", err)
		}
		if withObjects {
			terrain.DrawObjectMarkersAt(r.Image, anchorCells(m), m.Width, m.Height, terrain.CellSize, -r.OriginY*1)
		}
		var buf bytes.Buffer
		if err := png.Encode(&buf, r.Image); err != nil {
			t.Fatal(err)
		}
		return buf.Bytes()
	}
	for _, tc := range []struct {
		name    string
		file    string
		objects bool
	}{
		{"objects off", "plain.png", false},
		{"objects on", "objects.png", true},
	} {
		got, err := os.ReadFile(filepath.Join(dir, tc.file))
		if err != nil {
			t.Fatal(err)
		}
		if want := baseline(tc.objects); !bytes.Equal(got, want) {
			t.Errorf("with -units absent and %s, the render is %d bytes, want the pre-0009 %d-byte output",
				tc.name, len(got), len(want))
		}
	}

	zeroDir := t.TempDir()
	_, zeroMap := writeFixturesWithUnits(t, zeroDir, nil)
	var zeroOut bytes.Buffer
	if err := run([]string{"render", "-assets", zeroDir, "-map", zeroMap,
		"-out", filepath.Join(zeroDir, "zero.png"), "-units"}, &zeroOut); err != nil {
		t.Fatalf("zero-unit run: %v", err)
	}
	if !strings.Contains(zeroOut.String(), ", units 0") {
		t.Errorf("a unit-free map under -units printed %q, want a %q token", zeroOut.String(), ", units 0")
	}
}

func TestRenderGeometrySummary(t *testing.T) {
	dir := t.TempDir()
	_, mapPath := writeSlopedFixtures(t, dir)

	const cols, rows = 4, 3

	for _, tc := range []struct {
		name     string
		args     []string
		flat     bool
		unshaded bool
		scale    int
		wantRows int // the native canvas height, before scale
		wantOrig int
	}{
		{"projected shaded", nil, false, false, 1, slopedCanvasRows, slopedOriginY},
		{"projected shaded at scale 2", []string{"-scale", "2"}, false, false, 2, slopedCanvasRows, slopedOriginY},
		{"projected unshaded", []string{"-unshaded"}, false, true, 1, slopedCanvasRows, slopedOriginY},
		{"flat shaded", []string{"-flat"}, true, false, 1, rows * terrain.CellSize, 0},
		{"flat unshaded at scale 2", []string{"-flat", "-unshaded", "-scale", "2"}, true, true, 2, rows * terrain.CellSize, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			outPath := filepath.Join(dir, strings.ReplaceAll(tc.name, " ", "_")+".png")
			argv := append([]string{"render", "-assets", dir, "-map", mapPath, "-out", outPath}, tc.args...)
			var stdout bytes.Buffer
			if err := run(argv, &stdout); err != nil {
				t.Fatalf("run: %v", err)
			}
			summary := stdout.String()

			geometry := "projected"
			if tc.flat {
				geometry = "flat"
			}
			light := "shaded (theta=0.7854 ambient=14 range=32)"
			if tc.unshaded {
				light = "unshaded"
			}
			wantTokens := fmt.Sprintf(", %s, geometry %s, y origin %d\n", light, geometry, tc.wantOrig)
			if !strings.HasSuffix(summary, wantTokens) {
				t.Errorf("summary = %q, want it to end in %q", summary, wantTokens)
			}

			// AC-12: the reported dimensions are the written image's. The
			// expectation is built from the decoded PNG, so a summary computed
			// from the cell grid — W*32 x H*32, which is the flat canvas and no
			// longer the projected one — cannot satisfy it.
			img := decodePNG(t, outPath)
			b := img.Bounds()
			if want := fmt.Sprintf("%dx%d px at scale %d", b.Dx(), b.Dy(), tc.scale); !strings.Contains(summary, want) {
				t.Errorf("summary = %q, want the written image's %q", summary, want)
			}
			// And that image is the canvas this geometry defines, worked out by
			// hand from the fixture (see writeSlopedFixtures) — 132 native rows
			// projected against the flat raster's 96.
			wantW, wantH := cols*terrain.CellSize*tc.scale, tc.wantRows*tc.scale
			if b.Dx() != wantW || b.Dy() != wantH {
				t.Errorf("image is %dx%d, want %dx%d", b.Dx(), b.Dy(), wantW, wantH)
			}

			// The geometry token is a claim about which compositor ran; this is
			// the check that it is not merely a string. The oracle is a direct
			// render-tier call, so the tool cannot satisfy it by agreeing with
			// itself.
			oracle := compositeOracle(t, dir, mapPath, tc.flat, tc.unshaded, tc.scale)
			got, err := os.ReadFile(outPath)
			if err != nil {
				t.Fatal(err)
			}
			if want := pngBytes(t, oracle.Image); !bytes.Equal(got, want) {
				t.Errorf("the written PNG is %d bytes; want the %d bytes of the flat=%v unshaded=%v entry point",
					len(got), len(want), tc.flat, tc.unshaded)
			}
			if oracle.OriginY != tc.wantOrig {
				t.Errorf("oracle OriginY = %d, want %d", oracle.OriginY, tc.wantOrig)
			}
		})
	}

	// Fixture assumption: the four entry points must disagree on this map, or the
	// byte comparison above would be satisfied by any routing at all.
	t.Run("the four combinations are distinguishable", func(t *testing.T) {
		type key struct{ flat, unshaded bool }
		seen := map[string]key{}
		for _, k := range []key{{false, false}, {false, true}, {true, false}, {true, true}} {
			enc := string(pngBytes(t, compositeOracle(t, dir, mapPath, k.flat, k.unshaded, 1).Image))
			if prev, dup := seen[enc]; dup {
				t.Fatalf("fixture: flat=%v unshaded=%v renders identically to flat=%v unshaded=%v",
					k.flat, k.unshaded, prev.flat, prev.unshaded)
			}
			seen[enc] = k
		}
	})
}

func TestRenderFlatSelection(t *testing.T) {
	dir := t.TempDir()
	_, mapPath := writeSlopedFixtures(t, dir)

	for _, tc := range []struct {
		name     string
		args     []string
		unshaded bool
	}{
		{"shaded", []string{"-flat"}, false},
		{"unshaded", []string{"-flat", "-unshaded"}, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			outPath := filepath.Join(dir, "flat_"+tc.name+".png")
			argv := append([]string{"render", "-assets", dir, "-map", mapPath, "-out", outPath}, tc.args...)
			var stdout bytes.Buffer
			if err := run(argv, &stdout); err != nil {
				t.Fatalf("run: %v", err)
			}

			// Pixels: byte-for-byte the pre-change compositor's own output. That
			// compositor is Composite/CompositeLit unchanged, so this is the
			// pre-0012 image and not a re-derivation of it.
			got, err := os.ReadFile(outPath)
			if err != nil {
				t.Fatal(err)
			}
			want := pngBytes(t, compositeOracle(t, dir, mapPath, true, tc.unshaded, 1).Image)
			if !bytes.Equal(got, want) {
				t.Errorf("-flat wrote %d bytes, want the pre-change flat raster's %d", len(got), len(want))
			}

			// Dimensions and the placeholder count, as the pre-change summary
			// reported them.
			if b := decodePNG(t, outPath).Bounds(); b.Dx() != 128 || b.Dy() != 96 {
				t.Errorf("-flat image is %v, want the pre-change 128x96", b)
			}
			if want := fmt.Sprintf("placeholder cells %d, ", slopedPlaceholders); !strings.Contains(stdout.String(), want) {
				t.Errorf("summary = %q, want %q", stdout.String(), want)
			}
		})
	}

	// And the default is genuinely something else on this map: same width, a
	// taller canvas, different bytes. Without this the two assertions above would
	// hold just as well for a tool that ignored -flat entirely.
	defaultPath := filepath.Join(dir, "default.png")
	var stdout bytes.Buffer
	if err := run([]string{"render", "-assets", dir, "-map", mapPath, "-out", defaultPath}, &stdout); err != nil {
		t.Fatalf("default run: %v", err)
	}
	if b := decodePNG(t, defaultPath).Bounds(); b.Dx() != 128 || b.Dy() != slopedCanvasRows {
		t.Errorf("the default render is %v, want 128x%d — the projected canvas, not the flat one", b, slopedCanvasRows)
	}
	flatBytes, err := os.ReadFile(filepath.Join(dir, "flat_shaded.png"))
	if err != nil {
		t.Fatal(err)
	}
	defaultBytes, err := os.ReadFile(defaultPath)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(flatBytes, defaultBytes) {
		t.Error("-flat and the default wrote the same PNG on a map with unequal altitudes")
	}
}

func TestRenderMarkersAtProjectedOrigin(t *testing.T) {
	dir := t.TempDir()
	_, mapPath := writeSlopedFixtures(t, dir)
	proj := terrain.Project(slopedAltitudes, 4, 3)

	// The in-map anchors of each kind on the 4x3 grid; the rest are off-map and
	// must draw nothing however the lattice is translated.
	inObjects := [][2]int{{0, 0}, {2, 2}}
	inUnits := [][2]int{{2, 2}, {3, 0}}

	for _, scale := range []int{1, 2} {
		t.Run(fmt.Sprintf("scale %d", scale), func(t *testing.T) {
			cellpx := terrain.CellSize * scale

			render := func(name string, args ...string) image.Image {
				t.Helper()
				out := filepath.Join(dir, fmt.Sprintf("%s-s%d.png", name, scale))
				argv := append([]string{"render", "-assets", dir, "-map", mapPath,
					"-out", out, "-scale", fmt.Sprint(scale)}, args...)
				var buf bytes.Buffer
				if err := run(argv, &buf); err != nil {
					t.Fatalf("%s run: %v", name, err)
				}
				return decodePNG(t, out)
			}

			plain := render("markers-plain")
			marked := render("markers-objects", "-objects")
			unitsImg := render("markers-units", "-units")

			// Fixture assumption: neither marker colour occurs in the terrain, or
			// the counts below would not be evidence of an overlay.
			if n := countColor(plain, terrain.MarkerColor); n != 0 {
				t.Fatalf("the unmarked render already holds %d object-marker pixels", n)
			}
			if n := countColor(plain, terrain.UnitMarkerColor); n != 0 {
				t.Fatalf("the unmarked render already holds %d unit-marker pixels", n)
			}

			for _, tc := range []struct {
				kind      string
				img       image.Image
				cells     [][2]int
				colour    color.RGBA
				perMarker int
			}{
				{"object", marked, inObjects, terrain.MarkerColor, objectPixelsPerMarker(cellpx)},
				{"unit", unitsImg, inUnits, terrain.UnitMarkerColor, unitPixelsPerMarker(cellpx)},
			} {
				for _, c := range tc.cells {
					x, y := markerCentre(c[0], c[1], slopedOriginY, scale)
					y -= proj.AnchorHeight(c[0], c[1]) * scale
					if got := colorAt(tc.img, x, y); got != tc.colour {
						t.Errorf("%s marker for cell (%d,%d): pixel (%d,%d) = %v, want %v — the lattice must be shifted by -OriginY*scale = %d and then lifted by -AnchorHeight*scale = %d",
							tc.kind, c[0], c[1], x, y, got, tc.colour, -slopedOriginY*scale, -proj.AnchorHeight(c[0], c[1])*scale)
					}
					// The UN-shifted centre is where an offset of 0, or one in
					// native rows at scale 2, would have put the cross.
					ux, uy := markerCentre(c[0], c[1], 0, scale)
					if got := colorAt(tc.img, ux, uy); got == tc.colour {
						t.Errorf("%s marker for cell (%d,%d) was drawn at the un-shifted centre (%d,%d): the overlay ignored the vertical origin",
							tc.kind, c[0], c[1], ux, uy)
					}
				}
				// Every glyph is wholly inside this canvas at the right offset, so
				// the count is exact. At the wrong SIGN the map-extent lattice runs
				// off the top of the image and the second clip eats most of the
				// (0,0) cross, which this catches even where a centre happens to
				// survive.
				if got, want := countColor(tc.img, tc.colour), len(tc.cells)*tc.perMarker; got != want {
					t.Errorf("%s overlay holds %d marker pixels, want %d (%d in-map anchors, the off-map ones drawing nothing)",
						tc.kind, got, want, len(tc.cells))
				}
			}
		})
	}

	var plainOut, bothOut bytes.Buffer
	if err := run([]string{"render", "-assets", dir, "-map", mapPath, "-out", filepath.Join(dir, "tok-plain.png")}, &plainOut); err != nil {
		t.Fatalf("plain run: %v", err)
	}
	if err := run([]string{"render", "-assets", dir, "-map", mapPath, "-out", filepath.Join(dir, "tok-both.png"), "-objects", "-units"}, &bothOut); err != nil {
		t.Fatalf("-objects -units run: %v", err)
	}
	want := strings.TrimSuffix(plainOut.String(), "\n") +
		fmt.Sprintf(", objects %d, units %d\n", len(objectCells), len(unitCells))
	if got := bothOut.String(); got != want {
		t.Errorf("composed summary =\n  %q\nwant\n  %q", got, want)
	}
}
