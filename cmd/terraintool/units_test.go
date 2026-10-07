package main

import (
	"bytes"
	"image/color"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/game"
	"againrom/pkg/vfs"
)

// graphicsEntry turns an address into the entry path inside graphics.res: the
// container's identity segment, which every address the loaders resolve now
// carries, stripped off the front.
//
// A container holds container-relative entries, so that is what a fixture writes;
// the tool resolves addresses through the container filesystem it opens. The two
// differ by exactly that segment, and this DERIVES it from the archive name the
// tool opens rather than writing a second prefix literal — a literal could drift
// until the fixture keyed its entry under a prefix nothing resolves, and the suite
// would still be green.
//
// It lives in this file because units_test.go and unitanim_test.go are the two
// that write a registry entry by its constant; statics_test.go names its own
// container-relative literals and needs none of this.
func graphicsEntry(t *testing.T, address string) string {
	t.Helper()
	identity, err := vfs.Identity(graphicsArchive)
	if err != nil {
		t.Fatalf("vfs.Identity(%q): %v", graphicsArchive, err)
	}
	rel := strings.TrimPrefix(address, identity+"/")
	if rel == address {
		t.Fatalf("address %q carries no %q identity segment", address, identity)
	}
	return rel
}

// The fixture registry's class ids: one whose sheet the archive holds, one
// whose [Files] entry names art the archive deliberately omits.
const (
	censusDrawableID = 3 // File 0 — units/warrior/walk.256, present
	censusArtlessID  = 7 // File 1 — units/missing/gone.256, absent on purpose
)

// censusSheet is one drawable 2x2 frame. Its pixels are irrelevant to the
// census — only that frame 0 exists and decodes with a palette.
func censusSheet() []byte {
	pal := make([]color.RGBA, 4)
	pal[1] = color.RGBA{R: 0xff}
	return synth.Sheet256(synth.Sheet256Options{
		Palette: pal,
		Frames: []synth.Frame256{{Width: 2, Height: 2, Pixels: []synth.Pixel256{
			{Index: 1, Opaque: true}, {}, {}, {Index: 1, Opaque: true},
		}}},
	})
}

// censusRegistry writes the fixture units/units.reg: two classes, one
// drawable and one whose art the archive does not hold.
func censusRegistry() []byte {
	i := func(name string, v int32) synth.RegNode {
		return synth.RegNode{Name: name, Kind: 0x02, Int: v}
	}
	return synth.UnitsReg([]string{`warrior\walk`, `missing\gone`},
		[]synth.RegNode{
			i("ID", censusDrawableID), i("File", 0),
			i("Width", 16), i("Height", 16), i("CenterX", 8), i("CenterY", 14),
		},
		[]synth.RegNode{
			i("ID", censusArtlessID), i("File", 1),
			i("Width", 16), i("Height", 16), i("CenterX", 8), i("CenterY", 14),
		},
	)
}

// censusArchiveFiles is the fixture graphics archive's content: the registry
// and the one sheet it can draw. missing\gone.256 needs no builder, only an
// omission.
func censusArchiveFiles(t *testing.T) []synth.File {
	t.Helper()
	return []synth.File{
		{Path: graphicsEntry(t, game.UnitRegistry), Data: censusRegistry()},
		{Path: "units/warrior/walk.256", Data: censusSheet()},
	}
}

// writeCensusArchive lays the fixture graphics.res into dir.
func writeCensusArchive(t *testing.T, dir string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, "graphics.res"),
		synth.Archive(censusArchiveFiles(t)), 0o644); err != nil {
		t.Fatal(err)
	}
}

// writeCensusMap lays the fixture map: six placed units whose class keys make
// the three buckets PAIRWISE DISTINCT — 3 sprites, 2 no-class, 1 no-frame —
// so a line that swapped any two counts cannot come out right. One no-class
// key is negative: a census that zero-extended the stored word would look up
// 65527 instead of -9 — still a miss here, but the sign convention is the
// world's own and pkg/game's census tests pin it discriminatingly; this map
// keeps the corpus shape, every answer occurring. The stored anchors carry
// non-zero low bytes so the >>8 cell truncation is genuinely crossed.
func writeCensusMap(t *testing.T, dir string) string {
	t.Helper()
	path := filepath.Join(dir, "census.alm")
	data := synth.ALM(synth.ALMOptions{
		Width: 4, Height: 3,
		Units: []synth.ALMUnit{
			{X: 1<<8 | 0x40, Y: 1 << 8, ClassID: censusDrawableID},
			{X: 2 << 8, Y: 1<<8 | 0x80, ClassID: censusDrawableID},
			{X: 3 << 8, Y: 1 << 8, ClassID: censusDrawableID},
			{X: 1 << 8, Y: 2 << 8, ClassID: 5},  // no section carries 5
			{X: 2 << 8, Y: 2 << 8, ClassID: -9}, // negative: a clean miss
			{X: 3 << 8, Y: 2 << 8, ClassID: censusArtlessID},
		},
	})
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

// censusWant is the ONE line the fixture must print — asserted by exact
// equality, so a second line, a changed token or a swapped count all fail.
const censusWant = "units: 6 entities, 3 sprites, 2 no-class, 1 no-frame\n"

// TestUnitsCensusLine — SC-9: the subcommand prints exactly the census line,
// through each of the three archive-locating paths render's own flags take.
func TestUnitsCensusLine(t *testing.T) {
	dir := t.TempDir()
	writeCensusArchive(t, dir)
	mapPath := writeCensusMap(t, dir)

	t.Run("-assets", func(t *testing.T) {
		var buf bytes.Buffer
		if err := run([]string{"units", "-assets", dir, "-map", mapPath}, &buf); err != nil {
			t.Fatalf("run: %v", err)
		}
		if got := buf.String(); got != censusWant {
			t.Errorf("output = %q, want %q", got, censusWant)
		}
	})

	t.Run("-graphics", func(t *testing.T) {
		// The archive in a directory no asset root names, and no -assets given:
		// only -graphics can be what found it. It keeps the canonical FILE NAME,
		// because the addresses the bundle resolves under carry that name's own
		// identity segment — a renamed copy resolves nothing, which is R-1's
		// accepted consequence and not this test's subject.
		alt := t.TempDir()
		archivePath := filepath.Join(alt, graphicsArchive)
		if err := os.WriteFile(archivePath, synth.Archive(censusArchiveFiles(t)), 0o644); err != nil {
			t.Fatal(err)
		}
		var buf bytes.Buffer
		if err := run([]string{"units", "-graphics", archivePath, "-map", mapPath}, &buf); err != nil {
			t.Fatalf("run: %v", err)
		}
		if got := buf.String(); got != censusWant {
			t.Errorf("output = %q, want %q", got, censusWant)
		}
	})

	t.Run("AGAINROM_ASSETS", func(t *testing.T) {
		t.Setenv("AGAINROM_ASSETS", dir)
		var buf bytes.Buffer
		if err := run([]string{"units", "-map", mapPath}, &buf); err != nil {
			t.Fatalf("run: %v", err)
		}
		if got := buf.String(); got != censusWant {
			t.Errorf("output = %q, want %q", got, censusWant)
		}
	})
}

// TestUnitsErrors — SC-9's broken install: every load failure is an error —
// main exits non-zero on any error run returns, the shipped wiring — and a
// failed run prints NOTHING, so a recorded census can never be a partial one.
func TestUnitsErrors(t *testing.T) {
	dir := t.TempDir()
	writeCensusArchive(t, dir)
	mapPath := writeCensusMap(t, dir)

	// The broken install: a graphics.res that opens but holds no unit
	// registry, so the archive read succeeds and the bundle load fails.
	noReg := t.TempDir()
	if err := os.WriteFile(filepath.Join(noReg, "graphics.res"), synth.Archive([]synth.File{
		{Path: "units/warrior/walk.256", Data: censusSheet()},
	}), 0o644); err != nil {
		t.Fatal(err)
	}

	// A map that is not a map.
	badMap := filepath.Join(dir, "bad.alm")
	if err := os.WriteFile(badMap, []byte("not a map"), 0o644); err != nil {
		t.Fatal(err)
	}

	for _, tc := range []struct {
		name  string
		args  []string
		names string // a substring the error must carry, "" for none pinned
	}{
		{"missing -map", []string{"units", "-assets", dir}, "-map"},
		{"no asset root", []string{"units", "-map", mapPath}, ""},
		// -out is render's flag, not this subcommand's: units takes exactly
		// its three, not render's whole set.
		{"unknown flag", []string{"units", "-assets", dir, "-map", mapPath, "-out", "x.png"}, ""},
		{"archive not found", []string{"units", "-assets", filepath.Join(dir, "nope"), "-map", mapPath}, ""},
		{"no unit registry in the archive", []string{"units", "-assets", noReg, "-map", mapPath}, game.UnitRegistry},
		{"map not found", []string{"units", "-assets", dir, "-map", filepath.Join(dir, "nope.alm")}, ""},
		{"malformed map", []string{"units", "-assets", dir, "-map", badMap}, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// Hermetic even on a machine whose environment names a real
			// install: the no-asset-root case must not find one there.
			t.Setenv("AGAINROM_ASSETS", "")
			var stdout bytes.Buffer
			err := run(tc.args, &stdout)
			if err == nil {
				t.Fatal("expected an error, got nil")
			}
			if stdout.Len() != 0 {
				t.Errorf("a failed run wrote output: %q", stdout.String())
			}
			if tc.names != "" && !strings.Contains(err.Error(), tc.names) {
				t.Errorf("error %q does not name %q", err, tc.names)
			}
		})
	}
}
