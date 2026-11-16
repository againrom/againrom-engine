package sim

import (
	"math"
	"math/big"
	"testing"
)

// wholeShare is the share by its definition, at any width.
func wholeShare(p regenPart, gain *big.Int) *big.Int {
	var hi, lo big.Int
	hi.Quo(hi.Mul(gain, big.NewInt(p.j+1)), big.NewInt(p.k))
	lo.Quo(lo.Mul(gain, big.NewInt(p.j)), big.NewInt(p.k))
	return hi.Sub(&hi, &lo)
}

// TestPartShareMatchesWholeFormula pins both widths of the share to its definition.
func TestPartShareMatchesWholeFormula(t *testing.T) {
	gains := []int64{0, 1, 15, 33, 100, 12345, 1 << 40, math.MaxInt64 / 64, math.MaxInt64/64 + 1, math.MaxInt64 - 1, math.MaxInt64}
	var wide big.Int
	wide.SetString("1000000000000000000000007", 10)
	for _, k := range []int64{1, 2, 16, 64} {
		for j := int64(0); j < k; j++ {
			p := regenPart{j: j, k: k}
			for _, gain := range gains {
				got, apply := p.step(gain, 0)
				if want := wholeShare(p, big.NewInt(gain)); !apply || !want.IsInt64() || got != want.Int64() {
					t.Fatalf("part %d of %d, gain %d: share %d, definition %s", j, k, gain, got, want)
				}
			}
			var q, r big.Int
			q.QuoRem(&wide, big.NewInt(k), &r)
			q.Add(&q, big.NewInt(partShare(p, r.Int64())))
			if want := wholeShare(p, &wide); q.Cmp(want) != 0 {
				t.Fatalf("part %d of %d of a wide gain: share %s, definition %s", j, k, &q, want)
			}
		}
	}
}

// TestARegenerationPartAllocatesNothing keeps the per-tick share in integers.
func TestARegenerationPartAllocatesNothing(t *testing.T) {
	cur, rest := int32(10), uint8(0)
	allocs := testing.AllocsPerRun(200, func() {
		cur = 10
		regenerate(&cur, 1000, 30, &rest, 2, 0, 1, regenPart{j: 3, k: 16})
	})
	if allocs != 0 {
		t.Fatalf("a regeneration part allocates %v times", allocs)
	}
}
