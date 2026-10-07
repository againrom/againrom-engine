package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"image/color"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/render/terrain"
	"againrom/pkg/ui"
)

// --- synthetic fixtures ---
//
// Everything below is built byte-by-byte from the documented format contracts
// (.res per 0001, .alm per 0003, BMP per 0004). No game file is read and none of
// these bytes come from an install, so this suite is green with no game present.

const bmpPixelOffset = 14 + 40 + 256*4

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

// buildBMP8 writes a standard uncompressed 8-bpp Windows BMP whose rows are
// stored bottom-up, as the terrain strips are.
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
		e[0], e[1], e[2] = byte(i*7+3), byte(255-i), byte(i)
	}
	for y := 0; y < h; y++ {
		copy(out[bmpPixelOffset+(h-1-y)*stride:], topDown[y*w:(y+1)*w])
	}
	return out
}

// solidStrip builds a 32 x (32*cells) strip whose sub-cell k is palette index
// base+k, so a drawn pixel identifies the sub-cell it came from.
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
	name string
	data []byte
}

// buildArchive assembles a .res (&YA1) archive holding one terrain directory
// node whose children are the given files (RES-HDR-002..RES-NODE-008).
func buildArchive(entries []archiveEntry) []byte {
	const headerSize, nodeSize = 24, 32

	out := make([]byte, headerSize)
	offs := make([][2]uint32, len(entries))
	for i, e := range entries {
		offs[i] = [2]uint32{uint32(len(out)), uint32(len(e.data))}
		out = append(out, e.data...)
	}
	regOffset := uint32(len(out))

	binary.LittleEndian.PutUint32(out[0x00:], 0x31415926)
	binary.LittleEndian.PutUint32(out[0x08:], 1)
	binary.LittleEndian.PutUint32(out[0x0c:], 1)
	binary.LittleEndian.PutUint32(out[0x10:], regOffset)
	binary.LittleEndian.PutUint32(out[0x14:], uint32(1+len(entries)))

	node := func(off, size, typ uint32, name string) []byte {
		n := make([]byte, nodeSize)
		binary.LittleEndian.PutUint32(n[0x04:], off)
		binary.LittleEndian.PutUint32(n[0x08:], size)
		binary.LittleEndian.PutUint32(n[0x0c:], typ)
		copy(n[0x10:0x20], name)
		return n
	}
	out = append(out, node(1, uint32(len(entries)), 1, "terrain")...)
	for i, e := range entries {
		out = append(out, node(offs[i][0], offs[i][1], 0, e.name)...)
	}
	return out
}

// objectRecord builds one 20-byte type4 placed-object record anchored at cell
// (col, row). The stored coordinates are /256 fixed-point with a deliberately
// non-zero low byte, so the decoder's >>8 truncation is genuinely exercised.
func objectRecord(col, row int) []byte {
	rec := make([]byte, 20)
	binary.LittleEndian.PutUint32(rec[0x00:], uint32(col)<<8|0x40)
	binary.LittleEndian.PutUint32(rec[0x04:], uint32(row)<<8|0x80)
	return rec // kind (+0x08) stays 0 (!= 0x21), so no extension follows
}

// unitRecord builds one 70-byte type6 placed-unit record anchored at cell
// (col, row): X at +0x00, Y at +0x04, both /256 fixed-point with a non-zero low
// byte so the decoder's >>8 truncation is exercised. The remaining 62 bytes are
// undecoded (0003 R-2) and stay zero.
func unitRecord(col, row int) []byte {
	rec := make([]byte, 70)
	binary.LittleEndian.PutUint32(rec[0x00:], uint32(col)<<8|0x80)
	binary.LittleEndian.PutUint32(rec[0x04:], uint32(row)<<8|0x7f)
	return rec
}

func buildALM(w, h int, tiles []uint16, name string, objects, units [][2]int) []byte {
	cells := w * h

	meta := make([]byte, 632)
	binary.LittleEndian.PutUint32(meta[0x00:], uint32(w))
	binary.LittleEndian.PutUint32(meta[0x04:], uint32(h))
	binary.LittleEndian.PutUint32(meta[0x1c:], 1)
	binary.LittleEndian.PutUint32(meta[0x20:], uint32(len(objects)))
	binary.LittleEndian.PutUint32(meta[0x24:], uint32(len(units)))
	copy(meta[0x30:], name) // ASCII map name, NUL-terminated by the zero fill

	tileBytes := make([]byte, 2*cells)
	for i, w16 := range tiles {
		binary.LittleEndian.PutUint16(tileBytes[i*2:], w16)
	}

	grid8 := func() []byte { return make([]byte, cells) }

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
	payloads[2] = grid8()
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

func tileWord(g, b, sub int) uint16 {
	return uint16(g<<6) | uint16(b<<4) | uint16(sub)
}

// objectCells are the fixture map's placed-object anchors. The first is off-map
// on the 4x3 grid (legal at any index under the kind==0x21 extension
// discriminator), so it is counted but marked nowhere.
var objectCells = [][2]int{{9, 9}, {1, 1}, {3, 2}}

// unitCells are the fixture map's placed-unit anchors: one off-map on the 4x3
// grid, one sharing cell (3,2) with an object, one plain. type6 is a plain
// fixed-stride table, so any record may hold any anchor.
var unitCells = [][2]int{{8, 8}, {3, 2}, {0, 2}}

// writeFixtures lays a synthetic graphics.res and map.alm into dir.
func writeFixtures(t *testing.T, dir, mapName string) (archivePath, mapPath string) {
	t.Helper()
	return writeFixturesWithUnits(t, dir, mapName, unitCells)
}

// writeFixturesWithUnits is writeFixtures with a caller-chosen placed-unit
// list, so a test can build a map with no units at all -- representable for
// type6, and still required to report "units 0".
func writeFixturesWithUnits(t *testing.T, dir, mapName string, units [][2]int) (archivePath, mapPath string) {
	t.Helper()

	archivePath = filepath.Join(dir, "graphics.res")
	archive := buildArchive([]archiveEntry{
		{"tile1-00.bmp", solidStrip(14, 1)},
		{"tile3-00.bmp", solidStrip(8, 100)},
		{"dirt.bmp", solidStrip(4, 200)},
	})
	if err := os.WriteFile(archivePath, archive, 0o644); err != nil {
		t.Fatal(err)
	}

	// A 4x3 grid: land, water, and one cell naming a slot no file fills.
	tiles := make([]uint16, 12)
	for i := range tiles {
		tiles[i] = tileWord(0, 0, i%14)
	}
	tiles[5] = tileWord(8, 0, 3)  // water -> tile3
	tiles[7] = tileWord(12, 0, 1) // road -> tile4, absent here

	mapPath = filepath.Join(dir, "map.alm")
	if err := os.WriteFile(mapPath, buildALM(4, 3, tiles, mapName, objectCells, units), 0o644); err != nil {
		t.Fatal(err)
	}
	return archivePath, mapPath
}

// AC-5: -check loads the map and tileset, prints a one-line summary and exits 0
// without opening a window.
func TestCheckLoadsHeadlessly(t *testing.T) {
	dir := t.TempDir()
	writeFixtures(t, dir, "Testmap")

	var out bytes.Buffer
	err := run([]string{"-assets", dir, "-map", filepath.Join(dir, "map.alm"), "-check"}, &out)
	if err != nil {
		t.Fatalf("run: %v", err)
	}

	got := strings.TrimSpace(out.String())
	if strings.Count(got, "\n") != 0 {
		t.Fatalf("summary is not one line: %q", got)
	}
	for _, want := range []string{"Testmap", "4x3", "12", "tile slots"} {
		if !strings.Contains(got, want) {
			t.Fatalf("summary %q missing %q", got, want)
		}
	}
}

// The asset root may come from the environment instead of the flag, and the
// archive may be named directly with -graphics.
func TestCheckAssetRootSources(t *testing.T) {
	dir := t.TempDir()
	archivePath, mapPath := writeFixtures(t, dir, "Envmap")

	t.Run("AGAINROM_ASSETS", func(t *testing.T) {
		t.Setenv("AGAINROM_ASSETS", dir)
		var out bytes.Buffer
		if err := run([]string{"-map", mapPath, "-check"}, &out); err != nil {
			t.Fatalf("run: %v", err)
		}
		if !strings.Contains(out.String(), "Envmap") {
			t.Fatalf("summary %q missing the map name", out.String())
		}
	})

	t.Run("explicit -graphics", func(t *testing.T) {
		t.Setenv("AGAINROM_ASSETS", "")
		var out bytes.Buffer
		if err := run([]string{"-graphics", archivePath, "-map", mapPath, "-check"}, &out); err != nil {
			t.Fatalf("run: %v", err)
		}
		if !strings.Contains(out.String(), "Envmap") {
			t.Fatalf("summary %q missing the map name", out.String())
		}
	})
}

func TestLoadFailuresAreReported(t *testing.T) {
	dir := t.TempDir()
	writeFixtures(t, dir, "Testmap")
	mapPath := filepath.Join(dir, "map.alm")

	badMap := filepath.Join(dir, "bad.alm")
	if err := os.WriteFile(badMap, []byte("not an alm"), 0o644); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name string
		args []string
		want string
	}{
		{"no -map", []string{"-assets", dir, "-check"}, "-map is required"},
		{"map not found", []string{"-assets", dir, "-map", filepath.Join(dir, "nope.alm"), "-check"}, ""},
		{"unparseable map", []string{"-assets", dir, "-map", badMap, "-check"}, "decode"},
		{"archive not found", []string{"-graphics", filepath.Join(dir, "nope.res"), "-map", mapPath, "-check"}, "open"},
		{"no asset root", []string{"-map", mapPath, "-check"}, "no asset root"},
		{"unknown flag", []string{"-nope", "-map", mapPath, "-check"}, ""},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("AGAINROM_ASSETS", "")
			var out bytes.Buffer
			err := run(tc.args, &out)
			if err == nil {
				t.Fatalf("run succeeded, want an error")
			}
			if tc.want != "" && !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error %q, want it to contain %q", err, tc.want)
			}
			if out.Len() != 0 {
				t.Fatalf("failed run still wrote output: %q", out.String())
			}
		})
	}
}

// An archive with no terrain entries still loads: every cell falls back to
// the placeholder rather than failing.
func TestCheckWithEmptyArchive(t *testing.T) {
	dir := t.TempDir()
	writeFixtures(t, dir, "Testmap")

	emptyPath := filepath.Join(dir, "empty.res")
	if err := os.WriteFile(emptyPath, buildArchive(nil), 0o644); err != nil {
		t.Fatal(err)
	}

	var out bytes.Buffer
	err := run([]string{"-graphics", emptyPath, "-map", filepath.Join(dir, "map.alm"), "-check"}, &out)
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if !strings.Contains(out.String(), "tile slots 0/128") {
		t.Fatalf("summary %q, want it to report 0 loaded slots", out.String())
	}
}

// A map with no name falls back to the file's base name for the window title.
func TestUnnamedMapUsesFileName(t *testing.T) {
	dir := t.TempDir()
	writeFixtures(t, dir, "")

	var out bytes.Buffer
	if err := run([]string{"-assets", dir, "-map", filepath.Join(dir, "map.alm"), "-check"}, &out); err != nil {
		t.Fatalf("run: %v", err)
	}
	if !strings.Contains(out.String(), "map.alm") {
		t.Fatalf("summary %q, want the file name as the title", out.String())
	}
}

// TestCheckReportsCadence - 0006 SC-10: -check records the water cadence it
// would have animated at, so a headless run is evidence of the configured rate.
func TestCheckReportsCadence(t *testing.T) {
	dir := t.TempDir()
	writeFixtures(t, dir, "Testmap")
	mapPath := filepath.Join(dir, "map.alm")

	// The default is the index map load pushes: 16 tps, 62 ms/tick, 992 ms/cycle.
	var out bytes.Buffer
	if err := run([]string{"-assets", dir, "-map", mapPath, "-check"}, &out); err != nil {
		t.Fatalf("run: %v", err)
	}
	got := strings.TrimSpace(out.String())
	if strings.Count(got, "\n") != 0 {
		t.Fatalf("summary is not one line: %q", got)
	}
	for _, want := range []string{"water speed 4", "16 tps", "62 ms/tick", "992 ms/cycle"} {
		if !strings.Contains(got, want) {
			t.Fatalf("summary %q missing %q", got, want)
		}
	}
}

func TestObjectsFlag(t *testing.T) {
	dir := t.TempDir()
	writeFixtures(t, dir, "Testmap")
	mapPath := filepath.Join(dir, "map.alm")

	var plain bytes.Buffer
	if err := run([]string{"-assets", dir, "-map", mapPath, "-check"}, &plain); err != nil {
		t.Fatalf("plain run: %v", err)
	}
	var marked bytes.Buffer
	if err := run([]string{"-assets", dir, "-map", mapPath, "-check", "-objects"}, &marked); err != nil {
		t.Fatalf("-objects run: %v", err)
	}

	// AC-4: the disabled summary never mentions the overlay.
	if strings.Contains(plain.String(), "objects") {
		t.Errorf("summary without -objects mentions objects: %q", plain.String())
	}

	// AC-5: the count is the decoded object total, off-map anchors included.
	token := fmt.Sprintf(", objects %d", len(objectCells))
	if !strings.Contains(marked.String(), token) {
		t.Fatalf("summary %q missing %q", strings.TrimSpace(marked.String()), token)
	}

	// The enabled summary is the disabled one with the token spliced in at the
	// end of what load() builds — i.e. after "tile slots N/M" and before the
	// ", water speed ..." run() appends. Reconstructing it from the plain summary
	// pins both the content and the position.
	slots, cadence, ok := strings.Cut(strings.TrimSpace(plain.String()), ", water speed")
	if !ok {
		t.Fatalf("plain summary %q has no cadence clause to split on", plain.String())
	}
	want := slots + token + ", water speed" + cadence
	if got := strings.TrimSpace(marked.String()); got != want {
		t.Errorf("-objects summary =\n  %q\nwant\n  %q", got, want)
	}
}

// TestAnimationFlags - 0006 SC-10: -noanimation and -speed reach the viewer, and
// an out-of-range speed clamps rather than failing.
func TestAnimationFlags(t *testing.T) {
	dir := t.TempDir()
	writeFixtures(t, dir, "Testmap")
	mapPath := filepath.Join(dir, "map.alm")

	tests := []struct {
		name string
		args []string
		want string
	}{
		{"disabled", []string{"-noanimation"}, "water static"},
		{"slowest", []string{"-speed", "0"}, "water speed 0 (8 tps, 125 ms/tick, 2000 ms/cycle)"},
		{"fastest", []string{"-speed", "8"}, "water speed 8 (32 tps, 31 ms/tick, 496 ms/cycle)"},
		{"clamped high", []string{"-speed", "99"}, "water speed 8"},
		{"clamped low", []string{"-speed", "-4"}, "water speed 0"},
		{"disabled wins over speed", []string{"-noanimation", "-speed", "7"}, "water static"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			args := append([]string{"-assets", dir, "-map", mapPath, "-check"}, tc.args...)
			var out bytes.Buffer
			if err := run(args, &out); err != nil {
				t.Fatalf("run: %v", err)
			}
			if !strings.Contains(out.String(), tc.want) {
				t.Fatalf("summary %q missing %q", strings.TrimSpace(out.String()), tc.want)
			}
		})
	}
}

func TestUnitsFlag(t *testing.T) {
	dir := t.TempDir()
	writeFixtures(t, dir, "Testmap")
	mapPath := filepath.Join(dir, "map.alm")

	summary := func(args ...string) string {
		t.Helper()
		var buf bytes.Buffer
		argv := append([]string{"-assets", dir, "-map", mapPath, "-check"}, args...)
		if err := run(argv, &buf); err != nil {
			t.Fatalf("run %v: %v", args, err)
		}
		return strings.TrimSpace(buf.String())
	}

	plain := summary()
	unitsOnly := summary("-units")
	objectsOnly := summary("-objects")
	both := summary("-objects", "-units")
	reordered := summary("-units", "-objects")

	objToken := fmt.Sprintf(", objects %d", len(objectCells))
	unitToken := fmt.Sprintf(", units %d", len(unitCells))

	// AC-4: absent the flag the summary never mentions the overlay.
	if strings.Contains(plain, "units") {
		t.Errorf("summary without -units mentions units: %q", plain)
	}
	if strings.Contains(objectsOnly, "units") {
		t.Errorf("summary with -objects but no -units mentions units: %q", objectsOnly)
	}
	if strings.Contains(unitsOnly, "objects") {
		t.Errorf("summary with -units but no -objects mentions objects: %q", unitsOnly)
	}

	// AC-5 + DD7: each enabled summary is the disabled one with its token spliced
	// in at the end of what load() builds — after "tile slots N/M" and before the
	// ", water speed ..." run() appends. Reconstructing from the plain summary
	// pins both the content and the position.
	head, cadence, ok := strings.Cut(plain, ", water speed")
	if !ok {
		t.Fatalf("plain summary %q has no cadence clause to split on", plain)
	}
	for _, tc := range []struct {
		name, got, want string
	}{
		{"-units", unitsOnly, head + unitToken + ", water speed" + cadence},
		{"-objects", objectsOnly, head + objToken + ", water speed" + cadence},
		{"-objects -units", both, head + objToken + unitToken + ", water speed" + cadence},
	} {
		if tc.got != tc.want {
			t.Errorf("%s summary =\n  %q\nwant\n  %q", tc.name, tc.got, tc.want)
		}
	}

	if oi, ui := strings.Index(both, objToken), strings.Index(both, unitToken); oi < 0 || ui < 0 || oi > ui {
		t.Errorf("FR-5: the object count must precede the unit count, got %q", both)
	}
	if reordered != both {
		t.Errorf("summary depends on CLI flag order:\n  -units -objects = %q\n  -objects -units = %q",
			reordered, both)
	}

	zeroDir := t.TempDir()
	writeFixturesWithUnits(t, zeroDir, "Empty", nil)
	var zero bytes.Buffer
	if err := run([]string{"-assets", zeroDir, "-map", filepath.Join(zeroDir, "map.alm"), "-check", "-units"}, &zero); err != nil {
		t.Fatalf("zero-unit run: %v", err)
	}
	if !strings.Contains(zero.String(), ", units 0") {
		t.Errorf("a unit-free map under -units printed %q, want a %q token", strings.TrimSpace(zero.String()), ", units 0")
	}
}

func TestUnshadedFlagHelpReadsAsADiagnostic(t *testing.T) {
	lower := strings.ToLower(unshadedUsage)
	if !strings.Contains(lower, "diagnostic") {
		t.Fatalf("-unshaded help text does not read as a diagnostic: %q", unshadedUsage)
	}
	if !strings.Contains(lower, "does not move the light") {
		t.Fatalf("-unshaded help text does not disclaim moving the light: %q", unshadedUsage)
	}
}

func TestUnshadedFlagWiresToViewerLit(t *testing.T) {
	dir := t.TempDir()
	writeFixtures(t, dir, "Testmap")
	mapPath := filepath.Join(dir, "map.alm")

	fresh := func() *ui.Viewer {
		t.Helper()
		v, _, err := load(dir, "", mapPath, false, false, false, false, terrain.AnimGateTiles)
		if err != nil {
			t.Fatalf("load: %v", err)
		}
		return v
	}

	if v := fresh(); !v.Lit() {
		t.Fatal("setup: a freshly loaded viewer over the fixture's flat altitude grid must start lit")
	}

	unshadedViewer := fresh()
	configureViewer(unshadedViewer, false, true, terrain.DefaultSpeedIndex)
	if unshadedViewer.Lit() {
		t.Fatal("configureViewer(..., unshaded=true, ...) left the viewer lit")
	}

	litViewer := fresh()
	configureViewer(litViewer, false, false, terrain.DefaultSpeedIndex)
	if !litViewer.Lit() {
		t.Fatal("configureViewer(..., unshaded=false, ...) turned the viewer unlit")
	}
}

func TestUnshadedFlagDoesNotChangeCheckOutput(t *testing.T) {
	dir := t.TempDir()
	writeFixtures(t, dir, "Testmap")
	mapPath := filepath.Join(dir, "map.alm")

	summary := func(args ...string) string {
		t.Helper()
		var buf bytes.Buffer
		argv := append([]string{"-assets", dir, "-map", mapPath, "-check"}, args...)
		if err := run(argv, &buf); err != nil {
			t.Fatalf("run %v: %v", args, err)
		}
		return buf.String()
	}

	for _, overlay := range [][]string{
		nil,
		{"-objects"},
		{"-units"},
		{"-objects", "-units"},
	} {
		without := summary(overlay...)
		with := summary(append(append([]string{}, overlay...), "-unshaded")...)
		if with != without {
			t.Errorf("overlay %v: -unshaded changed the summary:\n without = %q\n with    = %q",
				overlay, without, with)
		}
		if strings.Contains(strings.ToLower(without), "unshaded") {
			t.Errorf("overlay %v: the summary mentions \"unshaded\" even without the flag: %q", overlay, without)
		}
	}
}

// --- 0017 T12: the static-object layer's two flags ------------------------
//
// SC-8's mapview half (AC-7). The assertions live here rather than in a file
// of their own because T12's entry names three existing test files and no
// new one: main_test.go is already on that list for the widened load() call
// above, and a fourth file is not.
//
// EVERY FIXTURE BELOW IS SYNTHETIC — a .res archive, an objects registry, two
// .256 sheets and an .alm map built byte by byte by internal/synth from the
// documented format contracts. No game install is read, so nothing here is
// evidence about a shipped file; what the real registry holds is
// verification.md's business against a lawful install (AC-8), not this file's.
//
// The fixture helpers above (buildArchive, buildALM, writeFixtures, solidStrip)
// are READ and never changed: buildArchive can only lay entries under one
// terrain node and buildALM writes an all-zero overlay, so neither can
// express an object layer at all. Growing them would have edited fixtures nine
// shipped tests depend on to serve two new ones.

// The placement bytes the fixture map carries. A byte names the class whose ID
// is one less, so these are IDs plus one. staticArtless names a class whose
// sheet the archive deliberately does not hold and staticUnknown names no loaded
// class at all: both are SKIPS rather than errors, so the run succeeds and
// neither is counted as a placement.
const (
	staticArt1    = 1   // ID 0 -> objects/trees/a.256
	staticArt2    = 2   // ID 1 -> objects/trees/b.256
	staticArtless = 3   // ID 2 -> objects/missing/gone.256, absent from the archive
	staticUnknown = 200 // ID 199 -> no section carries it
)

// staticPlacements is how many of the four non-zero bytes above resolve to a
// drawable frame. Four placed bytes and TWO placements is the point: it is what
// makes "statics N" a report of the placements DRAWN rather than of the cells
// that named something.
const staticPlacements = 2

// The class canvas and centre pixel every fixture class declares. The values are
// never asserted on here — a headless -check paints nothing — but a class must
// carry them or the anchor it is placed at would be built from pkg/data's
// absent-scalar default instead of from a stated geometry.
const (
	staticCanvas = 32
	staticCentre = 16
)

// staticSheet builds a one-frame, palette-bearing .256 sheet of solid opaque
// pixels. The colours are irrelevant to every assertion below and the palette is
// present only because a palette-less sheet is one of the exclusions: it would
// leave the class artless and silently drop the placement count to a number this
// file would then be pinning for the wrong reason.
func staticSheet(w, h int, index uint8) []byte {
	pixels := make([]synth.Pixel256, w*h)
	for i := range pixels {
		pixels[i] = synth.Pixel256{Index: index, Opaque: true}
	}
	pal := make([]color.RGBA, 8)
	pal[index] = color.RGBA{R: 0x10, G: 0x20, B: 0x30}
	return synth.Sheet256(synth.Sheet256Options{
		Palette: pal,
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

// writeStaticFixtures lays a graphics.res holding the object layer's entries and
// a 4x3 map carrying the overlay, one placed type-4 object and three placed
// type-6 units, and returns the map path.
//
// The three counts the summary can report are deliberately DISTINCT — statics 2,
// objects 1, units 3 — so a token that landed in another token's place, or read
// another's count, cannot pass by coincidence.
//
// objects/missing/gone.256 is absent on purpose: the absent-sheet exclusion needs
// no builder, only an omission. No terrain strip is laid at all, since a headless
// -check composites nothing; the summary reports 0/128 loaded slots and that is
// part of the pinned line below.
func writeStaticFixtures(t *testing.T, dir string) (mapPath string) {
	t.Helper()

	archive := synth.Archive([]synth.File{
		{Path: "objects/objects.reg", Data: staticObjectRegistry()},
		{Path: "objects/trees/a.256", Data: staticSheet(20, 30, 1)},
		{Path: "objects/trees/b.256", Data: staticSheet(20, 40, 2)},
	})
	if err := os.WriteFile(filepath.Join(dir, "graphics.res"), archive, 0o644); err != nil {
		t.Fatal(err)
	}

	mapPath = filepath.Join(dir, "statics.alm")
	data := synth.ALM(synth.ALMOptions{
		Width: 4, Height: 3, Name: "Statics",
		Overlay: []uint8{
			0, staticArt1, 0, 0,
			0, staticArt2, 0, 0,
			staticArtless, 0, 0, staticUnknown,
		},
		Objects: []synth.ALMObject{{X: 1<<8 | 0x40, Y: 0<<8 | 0x80}},
		Units: []synth.ALMUnit{
			{X: 1<<8 | 0x80, Y: 1<<8 | 0x7f},
			{X: 2<<8 | 0x80, Y: 1<<8 | 0x7f},
			{X: 3<<8 | 0x80, Y: 2<<8 | 0x7f},
		},
	})
	if err := os.WriteFile(mapPath, data, 0o644); err != nil {
		t.Fatal(err)
	}
	return mapPath
}

// shippedStaticSummary and shippedStaticOverlaySummary are what the SHIPPED tool
// printed for the fixture above, captured from cmd/mapview at 919650a — the
// commit before this task — and pasted here verbatim.
const (
	shippedStaticSummary        = "mapview: Statics 4x3 cells (12), tile slots 0/128, water speed 4 (16 tps, 62 ms/tick, 992 ms/cycle)\n"
	shippedStaticOverlaySummary = "mapview: Statics 4x3 cells (12), tile slots 0/128, objects 1, units 3, water speed 4 (16 tps, 62 ms/tick, 992 ms/cycle)\n"
)

func TestStaticsFlagsSummary(t *testing.T) {
	dir := t.TempDir()
	mapPath := writeStaticFixtures(t, dir)

	summary := func(args ...string) string {
		t.Helper()
		var buf bytes.Buffer
		argv := append([]string{"-assets", dir, "-map", mapPath, "-check"}, args...)
		if err := run(argv, &buf); err != nil {
			t.Fatalf("run %v: %v", args, err)
		}
		return buf.String()
	}

	staticsToken := fmt.Sprintf(", statics %d (animated %d)", staticPlacements, 0)

	t.Run("a flagless line is the shipped line, character for character", func(t *testing.T) {
		if got := summary(); got != shippedStaticSummary {
			t.Errorf("flagless -check summary =\n  %q\nwant the shipped\n  %q", got, shippedStaticSummary)
		}
		if got := summary("-objects", "-units"); got != shippedStaticOverlaySummary {
			t.Errorf("-objects -units summary =\n  %q\nwant the shipped\n  %q", got, shippedStaticOverlaySummary)
		}
	})

	t.Run("-statics reports the placements drawn", func(t *testing.T) {
		got := summary("-statics")
		if !strings.Contains(got, staticsToken) {
			t.Fatalf("summary %q missing %q", strings.TrimSpace(got), staticsToken)
		}
		// The count is the placements, not the placed bytes: the fixture's
		// overlay names four classes and only two of them can be drawn.
		if strings.Contains(got, ", statics 4") {
			t.Errorf("the count is the non-zero overlay bytes, not the placements: %q", strings.TrimSpace(got))
		}
		// Position: the token splices into the shipped line at the end of what
		// load() builds, before the cadence run() appends.
		head, cadence, ok := strings.Cut(shippedStaticSummary, ", water speed")
		if !ok {
			t.Fatalf("the shipped summary %q has no cadence clause to split on", shippedStaticSummary)
		}
		if want := head + staticsToken + ", water speed" + cadence; got != want {
			t.Errorf("-statics summary =\n  %q\nwant\n  %q", got, want)
		}
		// -flat is a geometry diagnostic and the count is the placements
		// DRAWABLE, which is one number per map and not one per geometry, so
		// the flag must move neither the token nor its value. The fixture's
		// altitude layer is the all-zero one, on which the two geometries
		// agree numerically, so this pins the token and cannot discriminate a
		// count that varied with the geometry — that is pkg/ui's own SC-3.
		if flat := summary("-statics", "-flat"); flat != got {
			t.Errorf("-flat moved the statics summary:\n without = %q\n with    = %q", got, flat)
		}
	})

	t.Run("-staticmarkers reports nothing", func(t *testing.T) {
		// The cross needs the bundle to know which cells resolve, but the count
		// belongs to -statics: the token must not leak onto the marker flag.
		for _, args := range [][]string{{"-staticmarkers"}, {"-staticmarkers", "-objects", "-units"}} {
			want := shippedStaticSummary
			if len(args) > 1 {
				want = shippedStaticOverlaySummary
			}
			if got := summary(args...); got != want {
				t.Errorf("%v summary =\n  %q\nwant the shipped\n  %q", args, got, want)
			}
		}
	})

	t.Run("the three tokens keep their order whatever the command line says", func(t *testing.T) {
		head, cadence, _ := strings.Cut(shippedStaticSummary, ", water speed")
		want := head + staticsToken + ", objects 1, units 3, water speed" + cadence
		for _, args := range [][]string{
			{"-statics", "-objects", "-units"},
			{"-units", "-objects", "-statics"},
			{"-objects", "-statics", "-staticmarkers", "-units"},
		} {
			if got := summary(args...); got != want {
				t.Errorf("%v summary =\n  %q\nwant\n  %q", args, got, want)
			}
		}
	})
}

func TestStaticsFlagsLoadTheBundle(t *testing.T) {
	dir := t.TempDir()
	mapPath := writeStaticFixtures(t, dir)

	// In a directory of its own, under the CANONICAL NAME: the discriminator has
	// to be "this archive holds no object registry", and an archive named
	// anything else would fail the flagged runs on its identity instead — the
	// bundle's addresses carry graphics.res's own identity segment now, so a
	// renamed copy answers nothing at all (0027 R-1). Naming it graphics.res in
	// a directory the run never passes as -assets keeps -graphics the only thing
	// that could have found it, and keeps the absent registry the only reason a
	// flagged run can fail.
	bare := filepath.Join(t.TempDir(), "graphics.res")
	if err := os.WriteFile(bare, buildArchive(nil), 0o644); err != nil {
		t.Fatal(err)
	}
	argv := func(args ...string) []string {
		return append([]string{"-graphics", bare, "-map", mapPath, "-check"}, args...)
	}

	t.Run("without the flags no bundle is read", func(t *testing.T) {
		var out bytes.Buffer
		if err := run(argv(), &out); err != nil {
			t.Fatalf("a flagless run over an archive with no object registry failed: %v", err)
		}
		if got := out.String(); got != shippedStaticSummary {
			t.Errorf("summary =\n  %q\nwant the shipped\n  %q", got, shippedStaticSummary)
		}
	})

	for _, flag := range []string{"-statics", "-staticmarkers"} {
		t.Run(flag+" reads it", func(t *testing.T) {
			var out bytes.Buffer
			err := run(argv(flag), &out)
			if err == nil {
				t.Fatalf("%s over an archive with no object registry succeeded", flag)
			}
			if !strings.Contains(err.Error(), "objects/objects.reg") {
				t.Errorf("%s failed with %q, want the entry named", flag, err)
			}
			if out.Len() != 0 {
				t.Errorf("failed run still wrote output: %q", out.String())
			}
		})
	}
}

// TestStaticsFlagsWireToTheViewer is SC-8's viewer clause for this tool: the two
// parsed values must actually reach the constructed viewer's two switches and
// its placement lists, not merely move the summary. Drives load() directly — the
// same seam TestUnshadedFlagWiresToViewerLit uses — since run() exposes no
// viewer and its window path cannot be called headlessly.
//
// The load below is fully headless: no window is opened and no texture is built,
// which is what makes a -check run possible on a machine with no GPU at all.
func TestStaticsFlagsWireToTheViewer(t *testing.T) {
	dir := t.TempDir()
	mapPath := writeStaticFixtures(t, dir)

	for _, tc := range []struct {
		name             string
		statics, markers bool
		wantPlacements   int
	}{
		{"neither", false, false, 0},
		{"art alone", true, false, staticPlacements},
		// The cross alone still builds the lists: it needs them to know which
		// cells resolve, and it paints no sprite.
		{"the cross alone", false, true, staticPlacements},
		{"both", true, true, staticPlacements},
	} {
		t.Run(tc.name, func(t *testing.T) {
			v, _, err := load(dir, "", mapPath, false, false, tc.statics, tc.markers, terrain.AnimGateTiles)
			if err != nil {
				t.Fatalf("load: %v", err)
			}
			art, markers := v.StaticOverlay()
			if art != tc.statics || markers != tc.markers {
				t.Errorf("StaticOverlay() = art %v, markers %v; want %v, %v",
					art, markers, tc.statics, tc.markers)
			}
			placements, counts := v.Statics()
			if placements != tc.wantPlacements {
				t.Errorf("Statics() placed %d, want %d", placements, tc.wantPlacements)
			}
			if tc.wantPlacements == 0 {
				return
			}
			// The two skip kinds stay counted apart, so the census can say
			// whether an unresolved cell was map data pointing past the
			// registry or art that could not be decoded.
			if counts.Placed != staticPlacements || counts.NoClass != 1 || counts.NoFrame != 1 {
				t.Errorf("census = %+v, want 2 placed, 1 no-class, 1 no-frame", counts)
			}
		})
	}
}

// --- 0028 T6: the surfaces this story leaves alone -------------------------
//
// SC-10's standalone-viewer clause (AC-11), asserted through the surface
// THIS TOOL actually has and never by reaching inside pkg/ui. The selection,
// the highlight and the gesture that forms an order are unexported spellings
// on the viewer, so this package cannot name one at all; what it can say is
// that nothing here is handed a way to advance or to order a world, that the
// viewer it builds holds no entity to select in the first place, and that
// the line it prints is the line it printed.
//
// The flag set is pinned in flagset_test.go and is left to it: this story added
// no flag anywhere, and a second inventory here would be a second thing to keep
// in step with the first.

// mapviewLoad pins the signature of the load path this tool comes through.
var mapviewLoad func(string, string, string, bool, bool, bool, bool, bool) (*ui.Viewer, string, error) = load

// TestTheStandaloneViewerHasNothingToSelect is AC-11's viewer clause: the
// standalone viewer renders with no selection at all, because it holds no
// entity for one to name.
//
// A SELECTION IS AN ID OUT OF THE SNAPSHOT and nothing else — this tool never
// puts one there, so there is no id to hold, and a highlight, which is drawn
// only for a selected unit PRESENT in the most recent snapshot, has nothing to
// resolve against either. The fixture map carries three placed units, which is
// what stops the zero below being a statement about an empty map.
//
// The viewer is driven through the very configuration run() applies after
// load() — the same calls, in the same order — so the count is read off a
// viewer in the state the tool actually runs one in, not off a bare one.
func TestTheStandaloneViewerHasNothingToSelect(t *testing.T) {
	dir := t.TempDir()
	writeFixtures(t, dir, "Testmap")
	mapPath := filepath.Join(dir, "map.alm")

	v, summary, err := mapviewLoad(dir, "", mapPath, false, true, false, false, terrain.AnimGateTiles)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if want := fmt.Sprintf("units %d", len(unitCells)); !strings.Contains(summary, want) {
		t.Fatalf("the fixture's summary %q does not report %q — with no placed unit the count below says "+
			"nothing", summary, want)
	}
	if got := v.EntityMarkers(); got != 0 {
		t.Fatalf("a freshly loaded standalone viewer holds %d entities, want 0: this tool owns no world, so "+
			"there is no id for a selection to name and none for a highlight to resolve (FR-10, DD-4)", got)
	}

	configureViewer(v, false, false, terrain.DefaultSpeedIndex)
	configureFlat(v, false)
	if got := v.EntityMarkers(); got != 0 {
		t.Errorf("after the configuration run() applies the standalone viewer holds %d entities, want 0", got)
	}
}

// shippedCheckLine is the tool's -check summary over the shared fixture, as
// shipped: the one byte-level output a headless run of the standalone viewer
// has. AC-11 asks for it unchanged, and the whole line is the assertion — a
// substring check would let a token be appended to the end of it and stay green.
const shippedCheckLine = "mapview: Testmap 4x3 cells (12), tile slots 2/128, " +
	"water speed 4 (16 tps, 62 ms/tick, 992 ms/cycle)"

// TestTheStandaloneCheckLineIsWhatItWas is AC-11's remaining clause for this
// tool. It is a characterization pin: this story adds no token, drops none and
// reorders none.
func TestTheStandaloneCheckLineIsWhatItWas(t *testing.T) {
	dir := t.TempDir()
	writeFixtures(t, dir, "Testmap")

	var out bytes.Buffer
	if err := run([]string{"-assets", dir, "-map", filepath.Join(dir, "map.alm"), "-check"}, &out); err != nil {
		t.Fatalf("run: %v", err)
	}
	if got := strings.TrimSuffix(out.String(), "\n"); got != shippedCheckLine {
		t.Errorf("the standalone -check line changed:\n got %q\nwant %q", got, shippedCheckLine)
	}
}
