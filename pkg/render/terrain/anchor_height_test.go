package terrain_test

import (
	"bytes"
	"math"
	"testing"

	terrain "againrom/pkg/render/terrain"
)

// anchorRecipeHeight is the spec's AnchorHeight formula, written independently
// of the implementation: col/row clamped into [0,Width-1]/[0,Height-1] with
// the min-then-max order projRecipeAltitude's own clamp uses, BEFORE the four
// corner indices are formed, summed and divided by 4 with Go's native
// truncating-toward-zero division.
func anchorRecipeHeight(alt []uint8, w, h, col, row int) int {
	col = min(col, w-1)
	row = min(row, h-1)
	col = max(col, 0)
	row = max(row, 0)

	h00 := projRecipeAltitude(alt, w, h, col, row)
	h10 := projRecipeAltitude(alt, w, h, col+1, row)
	h01 := projRecipeAltitude(alt, w, h, col, row+1)
	h11 := projRecipeAltitude(alt, w, h, col+1, row+1)
	return (h00 + h10 + h01 + h11) / 4
}

// anchorFixtures is every synthetic grid AnchorHeight is swept over: flat,
// sloped in one and both axes, negative altitudes, the widest possible
// neighbour delta, and the degenerate 1xN / Nx1 / 1x1 shapes — reusing
// project_test.go's own grid builders so the fixtures stay one set with the
// projection's own sweep.
func anchorFixtures() []projFixture {
	return []projFixture{
		{"uniform zero", 4, 3, projUniform(4, 3, 0x00)},
		{"uniform 0x01", 4, 3, projUniform(4, 3, 0x01)},
		{"uniform 0x7f", 4, 3, projUniform(4, 3, 0x7f)},
		{"uniform 0x80", 4, 3, projUniform(4, 3, 0x80)},
		{"uniform 0xff", 4, 3, projUniform(4, 3, 0xff)},
		{"ramp both axes", 5, 4, projRampAlt(5, 4)},
		{"chequered extremes (0x7f/0x80)", 4, 4, projExtremeAlt(4, 4)},
		{"every byte value", 16, 16, projSweepGrid()},
		{"single cell", 1, 1, []uint8{0x05}},
		{"one row", 5, 1, []uint8{0x00, 0x7f, 0x80, 0x40, 0xff}},
		{"one column", 1, 3, []uint8{0x00, 0x05, 0x00}},
	}
}

// TestAnchorHeightMatchesRecipe covers SC-1/AC-1: AnchorHeight against the
// hand-transcribed recipe, for an interior cell, every edge, every corner,
// and several cells at and past Width/Height, in both directions, plus
// negative indices — AnchorHeight is total over ANY col/row, not merely
// the [0,Width)x[0,Height) domain a marker anchor ordinarily lands in.
func TestAnchorHeightMatchesRecipe(t *testing.T) {
	for _, f := range anchorFixtures() {
		p := terrain.Project(f.alt, f.w, f.h)

		cols := anchorProbe(f.w)
		rows := anchorProbe(f.h)
		for _, col := range cols {
			for _, row := range rows {
				want := anchorRecipeHeight(f.alt, f.w, f.h, col, row)
				if got := p.AnchorHeight(col, row); got != want {
					t.Errorf("%s: AnchorHeight(%d,%d) = %d, want %d (recipe)",
						f.name, col, row, got, want)
				}
			}
		}
	}
}

// anchorProbe is the set of indices one dimension of size n is checked at:
// well inside the grid, on both edge cells, one and several past n, and
// negative — deduplicated is unnecessary, a redundant check costs nothing.
func anchorProbe(n int) []int {
	return []int{-5, -1, 0, 1, n / 2, n - 2, n - 1, n, n + 1, n + 7}
}

// TestAnchorHeightAtIntegerExtremes covers SC-1's "past Width/Height" clause
// at its most literal: col/row at math.MinInt and math.MaxInt (and their
// neighbours), over every fixture, checked against the same recipe.
func TestAnchorHeightAtIntegerExtremes(t *testing.T) {
	extremes := []int{math.MinInt, math.MinInt + 1, -1, 0, 1, math.MaxInt - 1, math.MaxInt}
	for _, f := range anchorFixtures() {
		p := terrain.Project(f.alt, f.w, f.h)
		for _, col := range extremes {
			for _, row := range extremes {
				want := anchorRecipeHeight(f.alt, f.w, f.h, col, row)
				if got := p.AnchorHeight(col, row); got != want {
					t.Errorf("%s: AnchorHeight(%d,%d) = %d, want %d (recipe, integer extreme)",
						f.name, col, row, got, want)
				}
			}
		}
	}
}

// wrapPastMaxInt is math.MaxInt + 1, forced through a runtime variable
// rather than written as the constant expression `math.MaxInt + 1`, which
// the compiler rejects as an overflowing untyped constant.
func wrapPastMaxInt() int {
	c := math.MaxInt
	c++
	return c
}

// TestAnchorHeightClampsBeforeFormingTheFarCorner is SC-1's own named trap:
// "a fixture built by adding 1 to an UNCLAMPED col at math.MaxInt disagrees
// with the clamped answer, so clamping after the +1 fails it" (plan SC-1).
func TestAnchorHeightClampsBeforeFormingTheFarCorner(t *testing.T) {
	const w, h = 4, 3
	alt := projRampAlt(w, h) // varies by column, so column 0 != column w-1
	p := terrain.Project(alt, w, h)

	for _, row := range []int{0, 1, h - 1} {
		want := p.AnchorHeight(w-1, row)

		// The fixture is only a trap if the wrapped read a clamp-after-+1
		// implementation would make — Altitude's own clamp applied to
		// math.MaxInt+1, which overflows to a negative number and is then
		// read as column 0 — actually disagrees with the near edge. Guard
		// against a fixture that happens not to discriminate.
		wrapped := wrapPastMaxInt() // overflows to a negative int, not Width
		wrongH10 := p.Altitude(wrapped, row)
		wrongH11 := p.Altitude(wrapped, row+1)
		if wrongH10 == p.Altitude(w-1, row) && wrongH11 == p.Altitude(w-1, row+1) {
			t.Fatalf("row %d: fixture is blind — the wrapped read and the near-edge read agree", row)
		}

		if got := p.AnchorHeight(math.MaxInt, row); got != want {
			t.Errorf("AnchorHeight(math.MaxInt,%d) = %d, want %d (the near edge cell's own mean; "+
				"col must clamp to Width-1 BEFORE +1 is formed, DD-1)", row, got, want)
		}
	}

	// The mirror on rows: math.MaxInt as row, an interior col.
	for _, col := range []int{0, 1, w - 1} {
		want := p.AnchorHeight(col, h-1)
		wrapped := wrapPastMaxInt()
		wrongH01 := p.Altitude(col, wrapped)
		wrongH11 := p.Altitude(col+1, wrapped)
		if wrongH01 == p.Altitude(col, h-1) && wrongH11 == p.Altitude(col+1, h-1) {
			t.Fatalf("col %d: fixture is blind — the wrapped row read and the near-edge read agree", col)
		}
		if got := p.AnchorHeight(col, math.MaxInt); got != want {
			t.Errorf("AnchorHeight(%d,math.MaxInt) = %d, want %d (the near edge cell's own mean; "+
				"row must clamp to Height-1 BEFORE +1 is formed, DD-1)", col, got, want)
		}
	}
}

func TestAnchorHeightFlatCellReturnsTheAltitude(t *testing.T) {
	const w, h = 4, 3
	for _, a := range []uint8{0x00, 0x01, 0x7f, 0x80, 0xff} {
		alt := projUniform(w, h, a)
		p := terrain.Project(alt, w, h)
		want := int(int8(a))
		for row := -1; row <= h+1; row++ {
			for col := -1; col <= w+1; col++ {
				if got := p.AnchorHeight(col, row); got != want {
					t.Errorf("uniform 0x%02x: AnchorHeight(%d,%d) = %d, want %d (P-1: flat cell returns its own altitude)",
						a, col, row, got, want)
				}
			}
		}
	}
}

// TestAnchorHeightTruncatesTowardZero pins the exact arithmetic the height
// lookup contract names: the sum of the four corners divided by 4 with Go's
// native truncating-toward-zero division, NOT a floor divide. Corners
// -2,-2,-1,-1 sum to -6; -6/4 truncated toward zero is -1, where a floor
// divide would give -2 — the two disagree, so this is decidable.
func TestAnchorHeightTruncatesTowardZero(t *testing.T) {
	const w, h = 2, 2
	alt := []uint8{0xFE, 0xFE, 0xFF, 0xFF} // signed: -2, -2, -1, -1
	p := terrain.Project(alt, w, h)

	if p.Altitude(0, 0) != -2 || p.Altitude(1, 0) != -2 || p.Altitude(0, 1) != -1 || p.Altitude(1, 1) != -1 {
		t.Fatalf("fixture: corners are %d,%d,%d,%d, want -2,-2,-1,-1",
			p.Altitude(0, 0), p.Altitude(1, 0), p.Altitude(0, 1), p.Altitude(1, 1))
	}
	if got, want := p.AnchorHeight(0, 0), -1; got != want {
		t.Errorf("AnchorHeight(0,0) = %d, want %d (truncating /4 of -6, not floor division's -2)", got, want)
	}
}

func TestAnchorHeightNeverMutatesTheAltitudeSlice(t *testing.T) {
	for _, f := range anchorFixtures() {
		alt := append([]uint8(nil), f.alt...) // this fixture's own copy to borrow
		original := append([]uint8(nil), alt...)
		p := terrain.Project(alt, f.w, f.h)

		probe := append(anchorProbe(f.w+f.h), math.MinInt, math.MinInt+1, -1, 0, math.MaxInt-1, math.MaxInt)
		for _, col := range probe {
			for _, row := range probe {
				_ = p.AnchorHeight(col, row)
			}
		}
		if !bytes.Equal(alt, original) {
			t.Fatalf("%s: AnchorHeight mutated the borrowed altitude slice: got %v, want %v (P-6)",
				f.name, alt, original)
		}
	}
}

func anchorBoundFixtures() []projFixture {
	return append(anchorFixtures(), projFixture{
		"negative corners, distinct (P-2's mandatory negative fixture)",
		2, 2,
		[]uint8{0xFE, 0xFE, 0xFF, 0xFF}, // signed: -2, -2, -1, -1
	})
}

func TestAnchorHeightBoundedByCornerExtremes(t *testing.T) {
	for _, f := range anchorBoundFixtures() {
		p := terrain.Project(f.alt, f.w, f.h)

		for _, col := range anchorProbe(f.w) {
			for _, row := range anchorProbe(f.h) {
				// The same pre-clamp AnchorHeight itself applies, so the four corners
				// read below are the cell AnchorHeight actually used for this col/row,
				// including every past-edge probe anchorProbe names.
				ccol := max(min(col, f.w-1), 0)
				crow := max(min(row, f.h-1), 0)

				h00 := p.Altitude(ccol, crow)
				h10 := p.Altitude(ccol+1, crow)
				h01 := p.Altitude(ccol, crow+1)
				h11 := p.Altitude(ccol+1, crow+1)

				lo, hi := h00, h00
				for _, h := range [3]int{h10, h01, h11} {
					if h < lo {
						lo = h
					}
					if h > hi {
						hi = h
					}
				}

				if got := p.AnchorHeight(col, row); got < lo || got > hi {
					t.Errorf("%s: AnchorHeight(%d,%d) = %d, outside its cell's own corner range [%d,%d] (corners %d,%d,%d,%d) — P-2",
						f.name, col, row, got, lo, hi, h00, h10, h01, h11)
				}
			}
		}
	}
}
