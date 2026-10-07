package sim

import "testing"

// The bound is INCLUSIVE, which is the whole reason this type states its own
// range rather than borrowing a half-open convention: both ends must be
// reachable and nothing outside may be.
func TestDrawsUptoCoversItsClosedRange(t *testing.T) {
	for _, n := range []int32{1, 2, 5, 70} {
		seen := make(map[int32]bool)
		d := NewDraws(0x0123456789ABCDEF)
		for i := 0; i < 20000; i++ {
			v := d.Upto(n)
			if v < 0 || v > n {
				t.Fatalf("Upto(%d) = %d, outside [0,%d]", n, v, n)
			}
			seen[v] = true
		}
		for v := int32(0); v <= n; v++ {
			if !seen[v] {
				t.Errorf("Upto(%d) never answered %d in 20000 draws", n, v)
			}
		}
	}
}

// A bound below one has one answer, and it costs no draw to give: a caller
// picking from a one-element list must not have its sequence advanced by the
// pick, or the fallback beside it would depend on the list's length.
func TestDrawsUptoDegenerateBoundConsumesNothing(t *testing.T) {
	d := NewDraws(7)
	for _, n := range []int32{0, -1, -1000} {
		if got := d.Upto(n); got != 0 {
			t.Fatalf("Upto(%d) = %d, want 0", n, got)
		}
	}
	fresh := NewDraws(7)
	for i := 0; i < 8; i++ {
		if a, b := d.Upto(100), fresh.Upto(100); a != b {
			t.Fatalf("draw %d: %d vs %d — a degenerate bound advanced the sequence", i, a, b)
		}
	}
}

// One seed, one sequence. This is the whole of what makes a load reproducible,
// so it is asserted directly rather than inferred from a world's digest.
func TestDrawsFromOneSeedAgree(t *testing.T) {
	a, b := NewDraws(0xDEADBEEF), NewDraws(0xDEADBEEF)
	for i := 0; i < 256; i++ {
		if x, y := a.Upto(int32(i)+1), b.Upto(int32(i)+1); x != y {
			t.Fatalf("draw %d: %d vs %d", i, x, y)
		}
	}
	c := NewDraws(0xDEADBEEE)
	same := true
	for i := 0; i < 64; i++ {
		if a.Upto(1000) != c.Upto(1000) {
			same = false
			break
		}
	}
	if same {
		t.Error("two different seeds produced the same 64 draws")
	}
}
