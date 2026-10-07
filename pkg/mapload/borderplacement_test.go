package mapload_test

import (
	"testing"

	"againrom/pkg/formats/alm"
	"againrom/pkg/mapload"
)

func cellAt(x, y int) (fx, fy uint32) { return uint32(x) << 8, uint32(y) << 8 }

// Ring placements go, first interior cells stay, survivors keep their order.
func TestWithdrawBorderPlacementsKeepsInteriorUnitsInOrder(t *testing.T) {
	t.Parallel()
	const w, h = 40, 36
	cells := [][2]int{
		{20, 20},          // interior
		{0, 20},           // ring, west edge
		{7, 20},           // ring, innermost west column
		{8, 20},           // interior
		{20, 7},           // ring, innermost north row
		{20, 8},           // interior, first row after the ring
		{w - 8, 20},       // ring, innermost east column
		{w - 9, 20},       // interior, last column before the ring
		{20, h - 8},       // ring, innermost south row
		{20, h - 9},       // interior, last row before the ring
		{43 % w, 136 % h}, // ring on the south side
	}
	var units []alm.Unit
	for i, c := range cells {
		x, y := cellAt(c[0], c[1])
		units = append(units, alm.Unit{X: x, Y: y, UnitID: uint16(100 + i)})
	}
	m := &alm.Map{Width: w, Height: h, Units: units}

	if got := mapload.WithdrawBorderPlacements(m); got != 6 {
		t.Fatalf("withdrew %d placements, want 6", got)
	}
	var ids []uint16
	for _, u := range m.Units {
		ids = append(ids, u.UnitID)
	}
	want := []uint16{100, 103, 105, 107, 109}
	if len(ids) != len(want) {
		t.Fatalf("survivors %v, want %v", ids, want)
	}
	for i := range want {
		if ids[i] != want[i] {
			t.Fatalf("survivors %v, want %v", ids, want)
		}
	}
	if again := mapload.WithdrawBorderPlacements(m); again != 0 {
		t.Fatalf("a second pass withdrew %d placements, want 0", again)
	}
	if mapload.WithdrawBorderPlacements(nil) != 0 {
		t.Fatal("nil map withdrew placements")
	}
}
