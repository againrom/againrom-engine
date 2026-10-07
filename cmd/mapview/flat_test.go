package main

// T6: -flat wires SetFlat into the standalone viewer.
//
// Everything below is a NEW test file — SC-4's budget for editing an existing
// test file is fully spent by T3 and T4 (cmd/terraintool/main_test.go,
// pkg/ui/mode_test.go, pkg/ui/light_test.go). cmd/mapview/main_test.go and
// flagset_test.go are untouched.
//
// The fixture below is its own tiny .alm builder (buildSlopedALM), not
// main_test.go's buildALM: that helper hardcodes an all-zero altitude
// payload, which cannot discriminate flat mode's world height from a
// genuinely displaced one. A borrowed offset into buildALM's output would be
// fragile against a change to its record order; a small, self-contained
// builder mirroring its framing is not.

import (
	"bytes"
	"encoding/binary"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"againrom/pkg/render/terrain"
	"againrom/pkg/ui"
)

// buildSlopedALM builds a minimal valid .alm (the same 10-record framing
// buildALM in main_test.go uses) carrying a caller-chosen altitude layer, and
// no objects, groups or units at all (their type-0 counts default to zero,
// which alm.go's decodeObjects/decodeGroups/decodeUnits accept only alongside
// an equally empty payload).
func buildSlopedALM(w, h int, tiles []uint16, alts []uint8, name string) []byte {
	cells := w * h
	if len(alts) != cells {
		panic("buildSlopedALM: alts must have exactly w*h entries")
	}
	if len(tiles) != cells {
		panic("buildSlopedALM: tiles must have exactly w*h entries")
	}

	meta := make([]byte, 632)
	binary.LittleEndian.PutUint32(meta[0x00:], uint32(w))
	binary.LittleEndian.PutUint32(meta[0x04:], uint32(h))
	copy(meta[0x30:], name)

	tileBytes := make([]byte, 2*cells)
	for i, w16 := range tiles {
		binary.LittleEndian.PutUint16(tileBytes[i*2:], w16)
	}

	var payloads [10][]byte
	payloads[0] = meta
	payloads[1] = tileBytes
	payloads[2] = append([]byte(nil), alts...)
	payloads[3] = make([]byte, cells)
	// payloads[4] (objects), [5] (groups), [6] (units) stay nil: the meta
	// counts they are checked against (Count4/Count5/Count6) are zero too, so
	// an empty payload is the one alm.go accepts.
	payloads[7] = le32(0)
	payloads[8] = nil
	payloads[9] = le32(0)

	out := concat(le32(0x0052374D), le32(20), le32(0), le32(10), le32(990))
	for _, typeID := range []int{0, 1, 2, 3, 5, 4, 9, 8, 6, 7} {
		p := payloads[typeID]
		out = concat(out, le32(7), le32(20), le32(uint32(len(p))), le32(uint32(typeID)), le32(0), p)
	}
	return out
}

// slopeW/slopeH/slopeAlts is a genuinely non-uniform 2x2 altitude grid: row 0
// flat at 0, row 1 flat at 40. Its displaced canvas height is computed from
// terrain.Project itself, never hand-derived, and self-checked against the
// flat height below so a degenerate fixture fails loudly rather than passing
// vacuously.
const slopeW, slopeH = 2, 2

var slopeAlts = []uint8{0, 0, 40, 40}

// writeSlopedFixture lays a synthetic (empty) graphics.res -- no terrain
// texture is read by a headless load -- and a sloped map.alm into dir.
func writeSlopedFixture(t *testing.T, dir string) (archivePath, mapPath string) {
	t.Helper()
	archivePath = filepath.Join(dir, "graphics.res")
	if err := os.WriteFile(archivePath, buildArchive(nil), 0o644); err != nil {
		t.Fatal(err)
	}
	tiles := make([]uint16, slopeW*slopeH)
	mapPath = filepath.Join(dir, "slope.alm")
	if err := os.WriteFile(mapPath, buildSlopedALM(slopeW, slopeH, tiles, slopeAlts, "Slope"), 0o644); err != nil {
		t.Fatal(err)
	}
	return archivePath, mapPath
}

// TestFlatFlagIsDefined guards the tests below from a typo in the flag's own
// name: if -flat were not registered, flag.Parse would reject it as unknown
// before either could reach any assertion (mirroring flagset_test.go's own
// "not defined" check, without touching that pinned file).
func TestFlatFlagIsDefined(t *testing.T) {
	err := run([]string{"-flat"}, io.Discard)
	if err != nil && strings.Contains(err.Error(), "not defined") {
		t.Fatalf("-flat is not a defined flag: %v", err)
	}
}

func TestFlatFlagSelectsFlatModeWhileStayingLit(t *testing.T) {
	dir := t.TempDir()
	_, mapPath := writeSlopedFixture(t, dir)

	proj := terrain.Project(slopeAlts, slopeW, slopeH)
	wantDisplacedH := float64(proj.CanvasHeight())
	wantFlatH := float64(slopeH) * terrain.CellSize
	if wantDisplacedH == wantFlatH {
		t.Fatalf("fixture does not discriminate: displaced and flat world heights are both %v", wantFlatH)
	}

	v, _, err := load(dir, "", mapPath, false, false, false, false, terrain.AnimGateTiles)
	if err != nil {
		t.Fatalf("load: %v", err)
	}

	if v.Mode() != ui.ModeDisplaced || !v.Lit() || v.Camera().WorldH() != wantDisplacedH {
		t.Fatalf("setup: Mode()=%v Lit()=%v WorldH()=%v, want displaced, lit, %v",
			v.Mode(), v.Lit(), v.Camera().WorldH(), wantDisplacedH)
	}

	configureFlat(v, true)
	if got := v.Mode(); got != ui.ModeFlat {
		t.Errorf("configureFlat(v, true): Mode() = %v, want flat", got)
	}
	if !v.Lit() {
		t.Error("configureFlat(v, true): Lit() = false, want true -- -flat must not suppress lighting")
	}
	if got := v.Camera().WorldH(); got != wantFlatH {
		t.Errorf("configureFlat(v, true): WorldH() = %v, want the flat %v", got, wantFlatH)
	}

	configureFlat(v, false)
	if got := v.Mode(); got != ui.ModeDisplaced {
		t.Errorf("configureFlat(v, false): Mode() = %v, want displaced", got)
	}
	if !v.Lit() {
		t.Error("configureFlat(v, false): Lit() = false, want true")
	}
	if got := v.Camera().WorldH(); got != wantDisplacedH {
		t.Errorf("configureFlat(v, false): WorldH() = %v, want the displaced %v", got, wantDisplacedH)
	}
}

func TestFlatFlagCheckOutputByteIdentical(t *testing.T) {
	dir := t.TempDir()
	_, mapPath := writeSlopedFixture(t, dir)

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
		with := summary(append(append([]string{}, overlay...), "-flat")...)
		if with != without {
			t.Errorf("overlay %v: -flat changed the summary:\n without = %q\n with    = %q",
				overlay, without, with)
		}
		if strings.Contains(strings.ToLower(without), "flat") {
			t.Errorf("overlay %v: the summary mentions \"flat\" even without the flag: %q", overlay, without)
		}
	}
}
