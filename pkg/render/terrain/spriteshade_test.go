package terrain_test

import (
	"image/color"
	"testing"

	"againrom/pkg/render/terrain"
)

func wantSpriteChannel(ch, tint uint8, row int) (out uint8, truncated, clamped bool) {
	if row < 0 {
		row = 0
	}
	if row > 15 {
		row = 15
	}
	sum := int(ch) + int(tint) // NOT clamped: the single clamp is the last one
	gainNumerator := (16 - row) * 2
	product := sum * gainNumerator
	quotient := product / 16 // truncating toward zero
	truncated = product%16 != 0
	if quotient > 255 {
		return 255, truncated, true
	}
	return uint8(quotient), truncated, false
}

// spriteRampPalette spans the channel range: entry i carries i on red, its
// complement on green and a third, coprime walk on blue, so every one of the 256
// channel values appears on red, the extremes 0 and 255 appear on two channels
// at once, and a ramp that transposed two channels lands a colour that appears
// nowhere in the correct output.
func spriteRampPalette() [256]color.RGBA {
	var pal [256]color.RGBA
	for i := range pal {
		pal[i] = color.RGBA{R: uint8(i), G: uint8(255 - i), B: uint8(7*i + 3), A: 0x00}
	}
	return pal
}

// TestSpriteRampIsFR1AtEveryRow — AC-1, SC-1's first half.
//
// The whole 256-entry palette at zero tint, at each of the sixteen rows, against
// the hand-written contract. 12288 channel comparisons, and the two properties
// AC-1 names of its own cases — a truncating divide and a result held at 255 —
// are counted and asserted present rather than assumed to be somewhere in there.
//
// Alpha is checked on every entry too: the palette above carries alpha 0
// throughout, so a ramp that copied the entry's alpha rather than forcing it
// opaque fails here on all 4096 entries instead of on none.
func TestSpriteRampIsFR1AtEveryRow(t *testing.T) {
	pal := spriteRampPalette()

	truncations, clamps, bad := 0, 0, 0
	for row := 0; row < terrain.SpriteRowCount; row++ {
		for i, entry := range pal {
			var want color.RGBA
			want.A = 0xff
			for c, ch := range [3]uint8{entry.R, entry.G, entry.B} {
				v, truncated, clamped := wantSpriteChannel(ch, 0, row)
				if truncated {
					truncations++
				}
				if clamped {
					clamps++
				}
				switch c {
				case 0:
					want.R = v
				case 1:
					want.G = v
				case 2:
					want.B = v
				}
			}
			if got := terrain.SpriteRGBA(entry, [3]uint8{}, row); got != want {
				bad++
				if bad <= 5 {
					t.Errorf("row %d entry %d: SpriteRGBA(%v) = %v, want %v", row, i, entry, got, want)
				}
			}
		}
	}
	if bad > 5 {
		t.Errorf("%d of %d palette entries wrong in total", bad, 16*256)
	}
	if truncations == 0 {
		t.Error("no case in the fixture truncated its divide; AC-1 requires at least one")
	}
	if clamps == 0 {
		t.Error("no case in the fixture clamped at 255; AC-1 requires at least one")
	}
	t.Logf("AC-1: %d channel comparisons, %d truncating divides, %d clamped at 255",
		16*256*3, truncations, clamps)
}

func TestSpriteRampDoesNotClampTheSumBeforeTheMultiply(t *testing.T) {
	const ch, tint, row = 200, 100, 10

	want, _, _ := wantSpriteChannel(ch, tint, row)
	if want != 225 {
		t.Fatalf("the hand-written contract gives %d for (ch=%d, tint=%d, row=%d), want 225", want, ch, tint, row)
	}
	preClamped := uint8(255 * ((16 - row) * 2) / 16)
	if preClamped == want {
		t.Fatalf("the case does not discriminate: a pre-multiply clamp also gives %d", preClamped)
	}
	if got := terrain.SpriteChannel(ch, tint, row); got != want {
		t.Errorf("SpriteChannel(%d, %d, %d) = %d, want %d (a pre-multiply clamp of the sum gives %d)",
			ch, tint, row, got, want, preClamped)
	}

	// The tint reaches all three channels, in order, and none of them borrows
	// another's: three distinct tint bytes over one grey entry.
	entry := color.RGBA{R: 60, G: 60, B: 60}
	tints := [3]uint8{0, 40, 200}
	got := terrain.SpriteRGBA(entry, tints, 4)
	want3 := color.RGBA{
		R: terrain.SpriteChannel(60, tints[0], 4),
		G: terrain.SpriteChannel(60, tints[1], 4),
		B: terrain.SpriteChannel(60, tints[2], 4),
		A: 0xff,
	}
	if got != want3 || want3.R == want3.G || want3.G == want3.B {
		t.Errorf("SpriteRGBA(%v, %v, 4) = %v, want %v with three distinct channels", entry, tints, got, want3)
	}
}

func TestSpriteRowEightReproducesTheRawPalette(t *testing.T) {
	for i, entry := range spriteRampPalette() {
		want := color.RGBA{R: entry.R, G: entry.G, B: entry.B, A: 0xff}
		if got := terrain.SpriteRGBA(entry, [3]uint8{}, 8); got != want {
			t.Fatalf("entry %d: row 8 gives %v, want the raw entry %v — row 8 is gain 1.0", i, got, want)
		}
	}
}

// TestSpriteRampEqualsTheTerrainLadderSampledEveryFourthRow — AC-3, SC-1's
// second half.
//
// EXHAUSTIVE over all 4096 (row, channel) pairs, terrain side read from the
// shipped transform at level 4*row+32 rather than restated beside it. The two
// ladders are written independently — 16 rows over a divisor of 16 against 96
// rows over a divisor of 32 — and the identity is the cross-check that the new
// transform is the old one's own sampling.
func TestSpriteRampEqualsTheTerrainLadderSampledEveryFourthRow(t *testing.T) {
	bad := 0
	for row := 0; row < terrain.SpriteRowCount; row++ {
		level := 4*row + 32
		for ch := 0; ch < 256; ch++ {
			sprite := terrain.SpriteChannel(uint8(ch), 0, row)
			ground := terrain.ShadeChannel(uint8(ch), 0, level)
			if sprite != ground {
				bad++
				if bad <= 5 {
					t.Errorf("row %d channel %d: sprite %d, terrain level %d gives %d",
						row, ch, sprite, level, ground)
				}
			}
		}
	}
	if bad > 5 {
		t.Errorf("%d of 4096 pairs disagree in total", bad)
	}
	if bad == 0 {
		t.Logf("AC-3: all 4096 (row, channel) pairs agree with terrain levels 32..92")
	}
}

// TestSpriteRowIsTheAmbientByteShiftedTwice — AC-4, SC-3.
//
// All four suns AC-4 names, including both clamped ends: 0x03 is the low end
// truncating to 0 and 0xff is the high end held at 15, which is where our own
// clamp does work no published path does. Theta, Range and the sky tint are
// varied across the cases and must not move the row — the decoded row carries no
// term but the ambient byte.
func TestSpriteRowIsTheAmbientByteShiftedTwice(t *testing.T) {
	cases := []struct {
		ambient uint8
		want    int
	}{
		{0x0e, 3}, // the default daytime sun
		{0x00, 0},
		{0x03, 0}, // truncates down, not up
		{0xff, 15},
	}
	for _, c := range cases {
		lt := terrain.Light{
			Theta:   float64(c.ambient),         // deliberately non-constant and ignored
			Ambient: c.ambient,                  //
			Range:   0xff - c.ambient,           // likewise
			SkyTint: [3]uint8{c.ambient, 9, 17}, // likewise
		}
		if got := terrain.SpriteRow(lt); got != c.want {
			t.Errorf("SpriteRow(ambient %#02x) = %d, want %d", c.ambient, got, c.want)
		}
	}

	if got := terrain.SpriteRow(terrain.DefaultDaytime); got != 3 {
		t.Errorf("SpriteRow(DefaultDaytime) = %d, want 3 — the fixed sun's own row", got)
	}

	// Every ambient byte lands in range, and the ladder is monotone in the byte:
	// a brighter sun is never a darker sprite.
	prev := 0
	for a := 0; a < 256; a++ {
		row := terrain.SpriteRow(terrain.Light{Ambient: uint8(a)})
		if row < 0 || row > 15 {
			t.Fatalf("ambient %d gives row %d, outside [0,15]", a, row)
		}
		if row < prev {
			t.Fatalf("ambient %d gives row %d after %d — the row must not go backwards", a, row, prev)
		}
		prev = row
	}
}

// TestSpriteChannelHoldsAnOutOfRangeRow — the row clamp, at the level below the
// blit's (AC-10 covers the same two values through BlitStaticLit).
//
// A row is a plain int at every call site, so the ramp is total: -1 draws at row
// 0 and 99 at row 15, rather than panicking or indexing something.
func TestSpriteChannelHoldsAnOutOfRangeRow(t *testing.T) {
	for _, c := range []struct{ row, held int }{{-1, 0}, {-1000, 0}, {16, 15}, {99, 15}} {
		for ch := 0; ch < 256; ch += 17 {
			got := terrain.SpriteChannel(uint8(ch), 0, c.row)
			want := terrain.SpriteChannel(uint8(ch), 0, c.held)
			if got != want {
				t.Fatalf("row %d channel %d gives %d, want row %d's %d", c.row, ch, got, c.held, want)
			}
		}
	}
}
