package terrain_test

import (
	"testing"

	terrain "againrom/pkg/render/terrain"
)

const (
	// stepCell is the number of destination columns an edge is spread over, and
	// the value the mirrored walk reflects the table about. Guarded against
	// terrain.CellSize below.
	stepCell = 32

	// stepMaxD is the widest step count an edge can carry, so the range the
	// table must cover. Both vertices of a cell edge sit in the same map row, so
	// the r*32 term of V(c,r) = r*32 - h(c,r) cancels and the step count is the
	// altitude difference alone; altitudes are signed 8-bit, so the widest
	// difference is 127 - (-128) = 255.
	stepMaxD = 255
)

// stepRecipeRow evaluates the spec's table recipe for one step count, written as
// the spec writes it. It reads nothing from the package under test.
func stepRecipeRow(d int) []int {
	s := 0x200000 / (d + 1)
	acc := 0x8000
	row := make([]int, d+1)
	for k := 0; k <= d; k++ {
		acc += s
		row[k] = acc >> 16
	}
	return row
}

// stepRecipeForward counts #{ k in [0,d) : T[d][k] <= i } as a set cardinality,
// entry by entry — the definition, not an equivalent of it.
func stepRecipeForward(row []int, d, i int) int {
	n := 0
	for k := 0; k < d; k++ {
		if row[k] <= i {
			n++
		}
	}
	return n
}

// stepRecipeMirrored counts #{ k in [0,d) : 32 - T[d][k] <= i }, likewise.
func stepRecipeMirrored(row []int, d, i int) int {
	n := 0
	for k := 0; k < d; k++ {
		if stepCell-row[k] <= i {
			n++
		}
	}
	return n
}

// stepAllColumns is every destination column of a cell, 0..31.
func stepAllColumns() []int {
	cols := make([]int, stepCell)
	for i := range cols {
		cols[i] = i
	}
	return cols
}

// stepWalkDisagreement is the COMPLETE classification of where the forward and
// mirrored walks disagree, over the whole built range d = 1..255: for each such
// step count, the difference (mirrored - forward) measured in steps, and the
// destination columns at which it holds. Every step count absent from this map
// agrees at all 32 columns.
//
//	d = 63, 127, 255 : mirrored one step AHEAD, at all 32 columns
//	d = 191          : mirrored one step BEHIND, at all 32 columns
//	d = 240          : mirrored one step BEHIND, at columns 8 and 23 only
//
// This falls out of the recipe, not out of the code. Two kinds of entry are
// asymmetric between the walks: an entry that has already reached the sentinel
// value 32 while still inside the counted range k in [0,d) never satisfies
// T[d][k] <= i for any column i <= 31, so the forward walk never counts it, while
// 32 - 32 = 0 <= i always holds, so the mirrored walk counts it at every column.
// An entry still at 0 is the same asymmetry reversed. Where the two populations
// do not cancel, the walks differ at every column by that net:
//
//	d = 63  : 0 zero entries, 1 sentinel entry inside the range  -> +1
//	d = 127 : 1 zero,  2 sentinels                               -> +1
//	d = 255 : 3 zeros, 4 sentinels                               -> +1
//	d = 191 : 3 zeros, 2 sentinels                               -> -1
//	d = 240 : 3 zeros, 3 sentinels                               ->  0 net,
//	          and the two walks still part company at columns 8 and 23, where the
//	          truncated thresholds i and 32-i fall on opposite sides of an entry
//	          boundary. It is the one step count whose disagreement is interior.
var stepWalkDisagreement = map[int]struct {
	delta int
	cols  []int
}{
	63:  {delta: +1, cols: stepAllColumns()},
	127: {delta: +1, cols: stepAllColumns()},
	191: {delta: -1, cols: stepAllColumns()},
	240: {delta: -1, cols: []int{8, 23}},
	255: {delta: +1, cols: stepAllColumns()},
}

// stepExpectedClassification expands stepWalkDisagreement into a step count ->
// column -> delta map, the shape stepClassify produces.
func stepExpectedClassification() map[int]map[int]int {
	want := make(map[int]map[int]int)
	for d, c := range stepWalkDisagreement {
		cols := make(map[int]int, len(c.cols))
		for _, i := range c.cols {
			cols[i] = c.delta
		}
		want[d] = cols
	}
	return want
}

// stepClassify sweeps the whole built range and records, for every (d, i) where
// the two step counts differ, the difference mirrored - forward.
func stepClassify(steps func(d, i int) (fwd, mir int)) map[int]map[int]int {
	got := make(map[int]map[int]int)
	for d := 1; d <= stepMaxD; d++ {
		for i := 0; i < stepCell; i++ {
			f, m := steps(d, i)
			if f == m {
				continue
			}
			if got[d] == nil {
				got[d] = make(map[int]int)
			}
			got[d][i] = m - f
		}
	}
	return got
}

// stepCompareClassification reports every (d, i) at which two classifications
// disagree, naming which side expected a disagreement and of what sign.
func stepCompareClassification(t *testing.T, what string, got, want map[int]map[int]int) {
	t.Helper()
	for d := 1; d <= stepMaxD; d++ {
		for i := 0; i < stepCell; i++ {
			g, gok := got[d][i]
			w, wok := want[d][i]
			switch {
			case gok && wok && g != w:
				t.Errorf("%s: d=%d col=%d: mirrored-forward = %+d, want %+d", what, d, i, g, w)
			case gok && !wok:
				t.Errorf("%s: d=%d col=%d: walks differ by %+d, want them to agree", what, d, i, g)
			case !gok && wok:
				t.Errorf("%s: d=%d col=%d: walks agree, want mirrored %+d from forward", what, d, i, w)
			}
		}
	}
}

// TestStepTable covers SC-2 / AC-5: the shape and sentinel of every row the table
// holds, the accessor's bounds, both walks against the recipe over the whole
// built range and in both step directions, and the complete classification of the
// step counts at which the two walks disagree, with the sign of each.
func TestStepTable(t *testing.T) {
	if terrain.CellSize != stepCell {
		t.Fatalf("CellSize = %d, want %d; the recipe and the mirrored walk are both stated over %d columns",
			terrain.CellSize, stepCell, stepCell)
	}

	t.Run("rows are the recipe's", func(t *testing.T) {
		for d := 1; d <= stepMaxD; d++ {
			want := stepRecipeRow(d)
			got := terrain.StepRow(d)
			if len(got) != len(want) {
				t.Fatalf("StepRow(%d) has %d entries, want %d (d+1)", d, len(got), len(want))
			}
			for k := range want {
				if int(got[k]) != want[k] {
					t.Fatalf("StepRow(%d)[%d] = %d, want %d (recipe)", d, k, got[k], want[k])
				}
			}
		}
	})

	t.Run("row shape and sentinel", func(t *testing.T) {
		// AC-5's shape assertions, over the whole built range rather than only
		// the engine's 1..127: non-decreasing, inside [0, 32], ending in 32.
		for d := 1; d <= stepMaxD; d++ {
			row := terrain.StepRow(d)
			for k, v := range row {
				if v > stepCell {
					t.Fatalf("StepRow(%d)[%d] = %d, want at most %d", d, k, v, stepCell)
				}
				if k > 0 && v < row[k-1] {
					t.Fatalf("StepRow(%d) decreases at k=%d: %d after %d", d, k, v, row[k-1])
				}
			}
			if last := row[len(row)-1]; last != stepCell {
				t.Fatalf("StepRow(%d) ends in %d, want the %d sentinel", d, last, stepCell)
			}
		}
	})

	t.Run("hand-computed rows", func(t *testing.T) {
		// Anchors whose arithmetic can be checked by eye, so the recipe
		// transcription above cannot drift with the implementation:
		//
		//	d=1:  s = 0x200000/2 = 0x100000; acc 0x108000, 0x208000 -> 16, 32
		//	d=2:  s = 0x200000/3 = 0x0AAAAA                         -> 11, 21, 32
		//	d=3:  s = 0x200000/4 = 0x080000                         -> 8, 16, 24, 32
		//	d=31: s = 0x200000/32 = 0x010000 exactly, so with the 0x8000 seed
		//	      T[31][k] = k+1 — the one row that steps once per column.
		rows := []struct {
			d    int
			want []int
		}{
			{1, []int{16, 32}},
			{2, []int{11, 21, 32}},
			{3, []int{8, 16, 24, 32}},
			{31, []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16,
				17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32}},
		}
		for _, r := range rows {
			got := terrain.StepRow(r.d)
			if len(got) != len(r.want) {
				t.Fatalf("StepRow(%d) has %d entries, want %d", r.d, len(got), len(r.want))
			}
			for k := range r.want {
				if int(got[k]) != r.want[k] {
					t.Errorf("StepRow(%d)[%d] = %d, want %d", r.d, k, got[k], r.want[k])
				}
			}
		}

		// Individual entries of the wide rows, each derivable in one line because
		// the divisor is exact: T[63][k]  = (k+2)/2 truncated (s = 0x8000),
		// T[127][k] = (k+3)/4 (s = 0x4000), T[255][k] = (k+5)/8 (s = 0x2000).
		entries := []struct{ d, k, want int }{
			{63, 0, 1}, {63, 1, 1}, {63, 62, 32}, {63, 63, 32},
			{127, 0, 0}, {127, 1, 1}, {127, 124, 31}, {127, 125, 32}, {127, 127, 32},
			{255, 0, 0}, {255, 2, 0}, {255, 3, 1}, {255, 251, 32}, {255, 255, 32},
		}
		for _, e := range entries {
			row := terrain.StepRow(e.d)
			if got := int(row[e.k]); got != e.want {
				t.Errorf("StepRow(%d)[%d] = %d, want %d", e.d, e.k, got, e.want)
			}
		}
	})

	t.Run("accessor bounds", func(t *testing.T) {
		for _, d := range []int{-1000, -1, 0, stepMaxD + 1, 1000} {
			if row := terrain.StepRow(d); row != nil {
				t.Errorf("StepRow(%d) = %v, want nil (outside 1..%d)", d, row, stepMaxD)
			}
		}
		for _, d := range []int{1, 2, stepMaxD - 1, stepMaxD} {
			if row := terrain.StepRow(d); len(row) != d+1 {
				t.Errorf("StepRow(%d) has %d entries, want %d", d, len(row), d+1)
			}
		}
	})

	t.Run("accessor hands out a copy", func(t *testing.T) {
		// The table is built once and never written again; a caller must not be
		// able to change every later render's geometry through the accessor.
		const d = 10
		first := terrain.StepRow(d)
		for k := range first {
			first[k] = 0xff
		}
		second := terrain.StepRow(d)
		want := stepRecipeRow(d)
		for k := range want {
			if int(second[k]) != want[k] {
				t.Fatalf("after mutating a returned row, StepRow(%d)[%d] = %d, want %d",
					d, k, second[k], want[k])
			}
		}
	})

	t.Run("walks are the recipe's, in both directions", func(t *testing.T) {
		// Each walk is checked against its set cardinality at every (d, i) the
		// format can produce, from several base vertices and with the far vertex
		// on either side of the near one — so a dropped base, a lost dir or a
		// swapped comparison all fail here.
		bases := []int{0, 7, -13, 4096}
		for d := 1; d <= stepMaxD; d++ {
			row := stepRecipeRow(d)
			for i := 0; i < stepCell; i++ {
				fwd := stepRecipeForward(row, d, i)
				mir := stepRecipeMirrored(row, d, i)
				for _, base := range bases {
					for _, dir := range []int{1, -1} {
						far := base + dir*d
						if got := terrain.EdgeForward(base, far, i); got != base+dir*fwd {
							t.Fatalf("EdgeForward(%d,%d,%d) = %d, want %d (%d steps of %+d)",
								base, far, i, got, base+dir*fwd, fwd, dir)
						}
						if got := terrain.EdgeMirrored(base, far, i); got != base+dir*mir {
							t.Fatalf("EdgeMirrored(%d,%d,%d) = %d, want %d (%d steps of %+d)",
								base, far, i, got, base+dir*mir, mir, dir)
						}
					}
				}
			}
		}
	})

	t.Run("neither walk passes its far vertex", func(t *testing.T) {
		// The spec caps both counts at d, so an edge that has reached its far
		// vertex stays there; and both counts are non-decreasing across the
		// columns, an edge never walking backwards.
		//
		// The sweep runs a little either side of the drawn columns 0..31, because
		// the cap is a statement about the counted range k in [0,d) and holds
		// wherever the column lands: the sentinel entry T[d][d] is outside that
		// range, so no column can buy a step from it.
		for d := 1; d <= stepMaxD; d++ {
			prevF, prevM := 0, 0
			for i := -4; i < stepCell+4; i++ {
				f := terrain.EdgeForward(0, d, i)
				m := terrain.EdgeMirrored(0, d, i)
				if f < 0 || f > d {
					t.Fatalf("EdgeForward(0,%d,%d) = %d, want within [0,%d]", d, i, f, d)
				}
				if m < 0 || m > d {
					t.Fatalf("EdgeMirrored(0,%d,%d) = %d, want within [0,%d]", d, i, m, d)
				}
				if f < prevF {
					t.Fatalf("EdgeForward(0,%d,·) goes backwards at col %d: %d after %d", d, i, f, prevF)
				}
				if m < prevM {
					t.Fatalf("EdgeMirrored(0,%d,·) goes backwards at col %d: %d after %d", d, i, m, prevM)
				}
				prevF, prevM = f, m
			}
		}
	})

	t.Run("equal vertices leave the edge constant", func(t *testing.T) {
		// d = 0 never consults the table.
		for _, y := range []int{0, 25, -74, 1 << 20} {
			for i := -1; i <= stepCell; i++ {
				if got := terrain.EdgeForward(y, y, i); got != y {
					t.Errorf("EdgeForward(%d,%d,%d) = %d, want %d", y, y, i, got, y)
				}
				if got := terrain.EdgeMirrored(y, y, i); got != y {
					t.Errorf("EdgeMirrored(%d,%d,%d) = %d, want %d", y, y, i, got, y)
				}
			}
		}
	})

	t.Run("a step count past the table has no row", func(t *testing.T) {
		// No pair of signed 8-bit altitudes can ask for this — the table covers
		// every step count the format produces — but the walks stay total for it
		// rather than indexing past the table: with no row there are no steps, so
		// the edge is the constant one d = 0 gives.
		for _, d := range []int{stepMaxD + 1, 512, 1 << 16} {
			for _, i := range []int{0, 15, 31} {
				if got := terrain.EdgeForward(100, 100+d, i); got != 100 {
					t.Errorf("EdgeForward(100,%d,%d) = %d, want 100 (no row for d=%d)", 100+d, i, got, d)
				}
				if got := terrain.EdgeMirrored(100, 100-d, i); got != 100 {
					t.Errorf("EdgeMirrored(100,%d,%d) = %d, want 100 (no row for d=%d)", 100-d, i, got, d)
				}
			}
		}
	})

	t.Run("the classification follows from the recipe", func(t *testing.T) {
		// The literal set above is checked against the transcribed recipe, so it
		// is pinned to the contract before the package is compared with it.
		got := stepClassify(func(d, i int) (int, int) {
			row := stepRecipeRow(d)
			return stepRecipeForward(row, d, i), stepRecipeMirrored(row, d, i)
		})
		stepCompareClassification(t, "recipe", got, stepExpectedClassification())
	})

	t.Run("the walks realise the classification", func(t *testing.T) {
		// With the near vertex at 0 and the far one at d, dir = +1 and each walk
		// returns its own step count.
		got := stepClassify(func(d, i int) (int, int) {
			return terrain.EdgeForward(0, d, i), terrain.EdgeMirrored(0, d, i)
		})
		stepCompareClassification(t, "walks", got, stepExpectedClassification())
	})

	t.Run("classification spot checks", func(t *testing.T) {
		// The disagreements and their immediate neighbourhood, spelled out, so a
		// failure names the step count and the direction rather than a map diff.
		spots := []struct {
			name            string
			d, i            int
			wantF, wantM    int
			checkDisagrees  bool
			mirroredIsAhead bool
		}{
			{name: "d=1 before its only step", d: 1, i: 15, wantF: 0, wantM: 0},
			{name: "d=1 at its only step", d: 1, i: 16, wantF: 1, wantM: 1},
			{name: "d=2 before its first step", d: 2, i: 10, wantF: 0, wantM: 0},
			{name: "d=2 at its first step", d: 2, i: 11, wantF: 1, wantM: 1},
			{name: "d=63 first column", d: 63, i: 0, wantF: 0, wantM: 1,
				checkDisagrees: true, mirroredIsAhead: true},
			{name: "d=63 last column", d: 63, i: 31, wantF: 62, wantM: 63,
				checkDisagrees: true, mirroredIsAhead: true},
			{name: "d=127 first column", d: 127, i: 0, wantF: 1, wantM: 2,
				checkDisagrees: true, mirroredIsAhead: true},
			{name: "d=127 last column", d: 127, i: 31, wantF: 125, wantM: 126,
				checkDisagrees: true, mirroredIsAhead: true},
			{name: "d=191 first column", d: 191, i: 0, wantF: 3, wantM: 2,
				checkDisagrees: true},
			{name: "d=191 last column", d: 191, i: 31, wantF: 189, wantM: 188,
				checkDisagrees: true},
			{name: "d=240 column 7 agrees", d: 240, i: 7, wantF: 56, wantM: 56},
			{name: "d=240 column 8 disagrees", d: 240, i: 8, wantF: 64, wantM: 63,
				checkDisagrees: true},
			{name: "d=240 column 23 disagrees", d: 240, i: 23, wantF: 177, wantM: 176,
				checkDisagrees: true},
			{name: "d=240 last column agrees", d: 240, i: 31, wantF: 237, wantM: 237},
			{name: "d=255 first column", d: 255, i: 0, wantF: 3, wantM: 4,
				checkDisagrees: true, mirroredIsAhead: true},
			{name: "d=255 last column", d: 255, i: 31, wantF: 251, wantM: 252,
				checkDisagrees: true, mirroredIsAhead: true},
		}
		for _, s := range spots {
			f := terrain.EdgeForward(0, s.d, s.i)
			m := terrain.EdgeMirrored(0, s.d, s.i)
			if f != s.wantF {
				t.Errorf("%s: EdgeForward(0,%d,%d) = %d, want %d", s.name, s.d, s.i, f, s.wantF)
			}
			if m != s.wantM {
				t.Errorf("%s: EdgeMirrored(0,%d,%d) = %d, want %d", s.name, s.d, s.i, m, s.wantM)
			}
			if !s.checkDisagrees {
				continue
			}
			if s.mirroredIsAhead && m != f+1 {
				t.Errorf("%s: mirrored=%d forward=%d, want mirrored one step ahead", s.name, m, f)
			}
			if !s.mirroredIsAhead && m != f-1 {
				t.Errorf("%s: mirrored=%d forward=%d, want mirrored one step behind", s.name, m, f)
			}
		}
	})
}
