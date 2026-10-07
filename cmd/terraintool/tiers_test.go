package main

import (
	"bytes"
	"image/color"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/formats/pal"
	"againrom/pkg/game"
)

const (
	tierCensusFull  = 4  // 3 declared tiers, all three loaded
	tierCensusShort = 11 // 2 declared, one absent
	tierCensusNone  = 19 // no tiers at all
)

// tierCensusPalette is the fixture sheet's own table, and tierCensusOther one
// that differs at every entry. Generated, never taken from anything.
func tierCensusPalette(shift uint8) []color.RGBA {
	out := make([]color.RGBA, 256)
	for i := range out {
		out[i] = color.RGBA{R: uint8(i) ^ shift, G: uint8(255 - i), B: uint8(i * 5)}
	}
	return out
}

// tierPalFile writes one synthetic colour-table file: the magic, filler to the
// table offset, the 1024-byte table in the same [B, G, R, reserved] order a
// sheet's own palette block uses, and pixel data after it that is never read.
func tierPalFile(colors []color.RGBA) []byte {
	out := make([]byte, pal.TableOffset)
	out[0], out[1] = 'B', 'M'
	out = append(out, synth.Palette256(colors)...)
	return append(out, make([]byte, 32)...)
}

func tierCensusFiles(t *testing.T) []synth.File {
	t.Helper()
	i := func(name string, v int32) synth.RegNode {
		return synth.RegNode{Name: name, Kind: 0x02, Int: v}
	}
	sheet := synth.Sheet256(synth.Sheet256Options{
		Palette: tierCensusPalette(0),
		Frames: []synth.Frame256{
			{Width: 2, Height: 3, Pixels: []synth.Pixel256{
				{Index: 40, Opaque: true}, {Index: 41, Opaque: true},
				{Index: 42, Opaque: true}, {}, {Index: 44, Opaque: true}, {},
			}},
		},
	})
	return []synth.File{
		{Path: graphicsEntry(t, game.UnitRegistry), Data: synth.UnitsReg(
			[]string{`full\walk`, `short\walk`, `plain\walk`},
			[]synth.RegNode{i("ID", tierCensusFull), i("File", 0), i("Palette", 3),
				i("Width", 4), i("Height", 4), i("CenterX", 2), i("CenterY", 3)},
			[]synth.RegNode{i("ID", tierCensusShort), i("File", 1), i("Palette", 2),
				i("Width", 4), i("Height", 4), i("CenterX", 2), i("CenterY", 3)},
			[]synth.RegNode{i("ID", tierCensusNone), i("File", 2), i("Palette", 0),
				i("Width", 4), i("Height", 4), i("CenterX", 2), i("CenterY", 3)},
		)},
		{Path: "units/full/walk.256", Data: sheet},
		{Path: "units/short/walk.256", Data: sheet},
		{Path: "units/plain/walk.256", Data: sheet},
		{Path: "units/full/palette.pal", Data: tierPalFile(tierCensusPalette(0))},
		{Path: "units/full/palette2.pal", Data: tierPalFile(tierCensusPalette(0x33))},
		{Path: "units/full/palette3.pal", Data: tierPalFile(tierCensusPalette(0x7f))},
		{Path: "units/short/palette.pal", Data: tierPalFile(tierCensusPalette(0))},
		// units/short/palette2.pal is ABSENT: the class declares two tiers and
		// only one of them is there.
	}
}

func writeTierArchive(t *testing.T, dir string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, graphicsArchive),
		synth.Archive(tierCensusFiles(t)), 0o644); err != nil {
		t.Fatal(err)
	}
}

// tierCensusWant is the EXACT output the fixture must print, asserted by whole-
// string equality so a missing line, a changed token or a swapped count all
// fail. The figures are the fixture's own by hand: three tiers all present on
// the first class, two declared with one absent on the second, and the third
// class carrying none is not a row at all — the summary's 2-of-3 is where it
// appears.
const tierCensusWant = "class 4 \"\": 3 tier(s), 3 loaded, 0 fell back, 1 frames, tier 1 the sheet's own\n" +
	"class 11 \"\": 2 tier(s), 1 loaded, 1 fell back, 1 frames, tier 1 the sheet's own\n" +
	"tiers: 2 of 3 classes carry tiers, 5 declared, 4 loaded, 1 fell back, " +
	"2 take the sheet's own table at tier 1\n"

// The census, and that a fall-back is DATA: the run reporting one still exits
// zero.
func TestTiersCensusLines(t *testing.T) {
	dir := t.TempDir()
	writeTierArchive(t, dir)

	t.Run("-assets", func(t *testing.T) {
		var buf bytes.Buffer
		if err := run([]string{"tiers", "-assets", dir}, &buf); err != nil {
			t.Fatalf("run: %v", err)
		}
		if got := buf.String(); got != tierCensusWant {
			t.Errorf("output = %q, want %q", got, tierCensusWant)
		}
	})

	// The archive in a directory no asset root names, and no -assets given:
	// only -graphics can be what found it.
	t.Run("-graphics", func(t *testing.T) {
		alt := t.TempDir()
		writeTierArchive(t, alt)
		var buf bytes.Buffer
		if err := run([]string{"tiers", "-graphics", filepath.Join(alt, graphicsArchive)}, &buf); err != nil {
			t.Fatalf("run: %v", err)
		}
		if got := buf.String(); got != tierCensusWant {
			t.Errorf("output = %q, want %q", got, tierCensusWant)
		}
	})
}

// A run naming no class writes no file: the census is the subcommand's own
// output and the picture is a second thing it can be asked for, so neither flag
// alone produces one and neither alone is an error.
func TestTiersWritesNoStillWithoutBothFlags(t *testing.T) {
	dir := t.TempDir()
	writeTierArchive(t, dir)
	still := filepath.Join(t.TempDir(), "still.png")

	for _, args := range [][]string{
		{"tiers", "-assets", dir},
		{"tiers", "-assets", dir, "-out", still},
		{"tiers", "-assets", dir, "-class", "4"},
	} {
		var buf bytes.Buffer
		if err := run(args, &buf); err != nil {
			t.Fatalf("run %v: %v", args, err)
		}
		if _, err := os.Stat(still); err == nil {
			t.Fatalf("run %v wrote a still", args)
		}
	}
}

// The still: one panel per tier, ascending, on a canvas whose width is the sum
// of the panels and the gutters. The frame is 2x3, so three tiers make a
// 3*2 + 4*8 = 38 wide by 3 + 2*8 = 19 high image, and the panels differ from
// each other because their tables do.
func TestTiersWritesTheStill(t *testing.T) {
	dir := t.TempDir()
	writeTierArchive(t, dir)
	still := filepath.Join(t.TempDir(), "still.png")

	var buf bytes.Buffer
	if err := run([]string{"tiers", "-assets", dir, "-class", "4", "-out", still}, &buf); err != nil {
		t.Fatalf("run: %v", err)
	}
	img := decodePNG(t, still)
	if got, want := img.Bounds().Dx(), 3*2+4*tierStillGutter; got != want {
		t.Errorf("still is %d wide, want %d", got, want)
	}
	if got, want := img.Bounds().Dy(), 3+2*tierStillGutter; got != want {
		t.Errorf("still is %d high, want %d", got, want)
	}

	// The three panels' top-left drawn pixel, which is the frame's own (0,0) —
	// index 40 of each tier's table, shaded. They must differ pairwise: the
	// three tables differ at that entry, and a still built from one slice three
	// times would print one colour three times.
	var seen []color.Color
	for i := 0; i < 3; i++ {
		x := tierStillGutter + i*(2+tierStillGutter)
		seen = append(seen, img.At(x, tierStillGutter))
	}
	for a := 0; a < len(seen); a++ {
		for b := a + 1; b < len(seen); b++ {
			if seen[a] == seen[b] {
				t.Errorf("panels %d and %d drew the same colour %v", a+1, b+1, seen[a])
			}
		}
	}
}

// The three refusals the still has, each naming what it could not do. A class
// the registry does not hold, one declaring no tiers, and a run over an archive
// that will not open are errors; a fall-back never is.
func TestTiersStillRefusals(t *testing.T) {
	dir := t.TempDir()
	writeTierArchive(t, dir)
	still := filepath.Join(t.TempDir(), "still.png")

	for _, tc := range []struct {
		name, class, want string
	}{
		{"a class the registry does not hold", "99", "no unit class"},
		{"a class declaring no tiers", "19", "declares no tiers"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer
			err := run([]string{"tiers", "-assets", dir, "-class", tc.class, "-out", still}, &buf)
			if err == nil {
				t.Fatal("no error")
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Errorf("error %q does not name %q", err, tc.want)
			}
		})
	}
}

// No install present at all: the subcommand fails, prints nothing and does not
// panic — the census is a recorded measurement, and lines over a half-loaded
// install would be wrong figures that look like right ones.
func TestTiersRefusesWithNoInstall(t *testing.T) {
	var buf bytes.Buffer
	if err := run([]string{"tiers", "-assets", filepath.Join(t.TempDir(), "nothing")}, &buf); err == nil {
		t.Fatal("no error")
	}
	if buf.Len() != 0 {
		t.Errorf("printed %q over a failed load", buf.String())
	}
}

// The dispatch: the usage the front door prints names every subcommand, so a
// caller who typed nothing is told the whole surface.
func TestTiersIsInTheUsage(t *testing.T) {
	var buf bytes.Buffer
	err := run(nil, &buf)
	if err == nil {
		t.Fatal("no error")
	}
	if !strings.Contains(err.Error(), tiersUsage) {
		t.Errorf("usage %q does not name the tiers subcommand", err)
	}
}
