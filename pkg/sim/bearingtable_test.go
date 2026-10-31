package sim

import (
	"testing"

	"againrom/pkg/intmath"
)

// centreBearingTable is MOVE-175's half-sector table and its fold, written
// out from the claim and not from headingOf: s from the signs and the 2:1
// tests, then ((s + (s != 0)) & 0xfe) << 4 in a byte.
func centreBearingTable(dx, dy int64) uint8 {
	x, y := intmath.Abs(dx), intmath.Abs(dy)
	b := func(c bool) int64 {
		if c {
			return 1
		}
		return 0
	}
	var s int64
	switch {
	case dx > 0 && dy > 0:
		if x > y {
			s = 4 + b(x <= 2*y)
		} else {
			s = 6 + b(y > 2*x)
		}
	case dx > 0:
		if x > y {
			s = 2 + b(x > 2*y)
		} else {
			s = b(y <= 2*x)
		}
	case dy > 0:
		if x > y {
			s = 0xa + b(x > 2*y)
		} else {
			s = 8 + b(y <= 2*x)
		}
	default:
		if x > y {
			s = 0xc + b(x <= 2*y)
		} else {
			s = 0xe + b(y > 2*x)
		}
	}
	return uint8(((s + b(s != 0)) & 0xfe) << 4)
}

// The actor heading law gives MOVE-175's facing for every centre delta: the
// whole square of deltas up to 3 cells and 32 fine units on each axis, and a
// sweep of the rest of the range two 16-bit centres can differ by.
func TestHeadingOfMatchesTheCentreBearingTableForEveryDelta(t *testing.T) {
	check := func(dx, dy int64) bool {
		if got, want := headingOf(dx, dy), centreBearingTable(dx, dy); got != want {
			t.Errorf("delta (%d,%d): heading %#x, want %#x", dx, dy, got, want)
			return false
		}
		return true
	}
	for dy := int64(-800); dy <= 800; dy++ {
		for dx := int64(-800); dx <= 800; dx++ {
			if !check(dx, dy) {
				return
			}
		}
	}
	for dy := int64(-65535); dy <= 65535; dy += 251 {
		for dx := int64(-65535); dx <= 65535; dx += 241 {
			if !check(dx, dy) {
				return
			}
		}
	}
	for _, d := range [][2]int64{{65535, 0}, {0, -65535}, {-65535, 65535}, {65535, 32767}, {65535, 32768}, {-32768, -65535}} {
		check(d[0], d[1])
	}
}

// The step heading is MOVE-176's sign table toward the node centre, and at a
// zero delta the low three bits of the current facing byte times 0x20, for
// every facing byte.
func TestStepHeadingMatchesTheStepFacingTable(t *testing.T) {
	want := [3][3]uint8{ // [sign dy][sign dx], sign index 0 negative, 1 zero, 2 positive
		{0xe0, 0x00, 0x20},
		{0xc0, 0x00, 0x40},
		{0xa0, 0x80, 0x60},
	}
	w := fineWorld(map[EntityID][2]uint8{1: {0x80, 0x80}, 2: {0x30, 0xd0}})
	for _, fine := range []struct {
		id     EntityID
		fx, fy int32
	}{{1, 0x80, 0x80}, {2, 0x30, 0xd0}} {
		e := Entity{ID: fine.id, X: 20, Y: 20}
		for ny := int32(18); ny <= 22; ny++ {
			for nx := int32(18); nx <= 22; nx++ {
				dx, dy := (nx-20)*256+128-fine.fx, (ny-20)*256+128-fine.fy
				if dx == 0 && dy == 0 {
					continue
				}
				if got, wantH := w.stepHeading(e, nx, ny), want[signIndex(dy)][signIndex(dx)]; got != wantH {
					t.Errorf("entity %d node (%d,%d): heading %#x, want %#x", fine.id, nx, ny, got, wantH)
				}
			}
		}
	}
	for f := 0; f < 256; f++ {
		e := Entity{ID: 1, X: 20, Y: 20, Facing: uint8(f)}
		if got, want := w.stepHeading(e, 20, 20), uint8((f&7)<<5); got != want {
			t.Errorf("zero delta, facing %#x: heading %#x, want %#x", f, got, want)
		}
	}
}
