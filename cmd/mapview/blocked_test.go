package main

import (
	"bytes"
	"image"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// AC-11 — the tint's cells are exactly the plane's ground-closed ones, in
// row-major order, and a cell closed to an AIR mover alone is not among them.
//
// The plane is built by hand here, from the two bit meanings the contract gives
// and not from a derivation of ours: what this pins is the reduction, and a plane
// read out of the deriver would make the assertion agree with whatever that
// deriver did.
func TestBlockedCells(t *testing.T) {
	t.Parallel()

	const w, h = 4, 3
	// (0,0) ground, (2,0) ground+air, (1,1) AIR ONLY, (3,2) ground. The rest open.
	plane := make([]byte, w*h)
	plane[0*w+0] = 0x01
	plane[0*w+2] = 0x03
	plane[1*w+1] = 0x02
	plane[2*w+3] = 0x01

	want := []image.Point{{X: 0, Y: 0}, {X: 2, Y: 0}, {X: 3, Y: 2}}
	got := blockedCells(plane, w, h)
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("cell %d is %v, want %v — the list must be row-major", i, got[i], want[i])
		}
	}
	for _, c := range got {
		if c == (image.Point{X: 1, Y: 1}) {
			t.Error("the air-only cell is in the list; the tint must read bit 0 alone")
		}
	}
}

// The reduction is total: no extent, a short plane and an over-long one each
// yield an answer rather than a panic.
func TestBlockedCellsIsTotal(t *testing.T) {
	t.Parallel()

	full := []byte{1, 1, 1, 1, 1, 1}
	for _, tc := range []struct {
		name  string
		plane []byte
		w, h  int
		want  int
	}{
		{"no width", full, 0, 3, 0},
		{"no height", full, 3, 0, 0},
		{"a negative extent", full, -2, 3, 0},
		{"a plane shorter than the extent", full[:4], 3, 2, 4},
		{"a plane longer than the extent", full, 2, 2, 4},
		{"a nil plane", nil, 3, 3, 0},
	} {
		if got := blockedCells(tc.plane, tc.w, tc.h); len(got) != tc.want {
			t.Errorf("%s: %d cells, want %d", tc.name, len(got), tc.want)
		}
	}
}

// -blocked needs a definition table and says so before anything is opened: no
// asset root and no -databin is a failure, and the message names the way out
// rather than a path of ours (golden rule 3).
func TestBlockedNeedsATable(t *testing.T) {
	dir := t.TempDir()
	writeFixtures(t, dir, "Testmap")
	mapPath := filepath.Join(dir, "map.alm")

	t.Setenv("AGAINROM_ASSETS", "")
	var out bytes.Buffer
	err := run([]string{"-graphics", filepath.Join(dir, "graphics.res"), "-map", mapPath, "-check", "-blocked"}, &out)
	if err == nil {
		t.Fatal("-blocked without a table succeeded")
	}
	for _, want := range []string{"-blocked", "-databin", "AGAINROM_ASSETS"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("the failure %q does not mention %q", err, want)
		}
	}
	if out.Len() != 0 {
		t.Errorf("a failing -blocked run printed a summary: %q", out.String())
	}

	// And an asset root with no world.res in it fails on the archive rather
	// than silently drawing the arms-only plane.
	out.Reset()
	if err := run([]string{"-assets", dir, "-map", mapPath, "-check", "-blocked"}, &out); err == nil {
		t.Error("-blocked with no world.res in the asset root succeeded")
	}
}

func TestBlockedIsOffByDefault(t *testing.T) {
	dir := t.TempDir()
	writeFixtures(t, dir, "Testmap")
	mapPath := filepath.Join(dir, "map.alm")

	// No world.res exists in the fixture dir at all, so a run that reached for
	// a table would fail rather than quietly succeed.
	if _, err := os.Stat(filepath.Join(dir, worldArchive)); !os.IsNotExist(err) {
		t.Fatalf("the fixture dir already holds a %s", worldArchive)
	}

	var out bytes.Buffer
	if err := run([]string{"-assets", dir, "-map", mapPath, "-check"}, &out); err != nil {
		t.Fatalf("plain run: %v", err)
	}
	if strings.Contains(out.String(), "blocked") {
		t.Errorf("the unflagged summary mentions the tint: %q", out.String())
	}

	v, _, err := load(dir, "", mapPath, false, false, false, false, false)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if on, n := v.BlockedOverlay(); on || n != 0 {
		t.Errorf("an unflagged load set the tint to %v/%d", on, n)
	}
}
