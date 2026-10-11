package game

import (
	"math"
	"testing"

	"againrom/pkg/sim"
)

// The simulation package forms the first game's binary64 spell operands in
// integers. These tests compute the same operations in float64 here, outside
// the simulation package, and require equality over the whole input domain.
// Each explicit float64 conversion rounds, so no operation is fused.

func recordDamageBinary64(dmin, dmax, power int32) (uint8, uint8) {
	f := float64(float64(uint8(power))/30.0) + 1.0
	var base, spread uint8
	if dmin > 0 {
		base = uint8(int64(float64(float64(dmin) * f)))
	}
	if dmax > 0 {
		spread = uint8(int64(float64(float64(float64(dmax)*f) - float64(base))))
	}
	return base, spread
}

// TestSpellRecordDamageEqualsBinary64 covers every damage column byte against
// every power byte, which contains every clamped cast power 0..100.
func TestSpellRecordDamageEqualsBinary64(t *testing.T) {
	failures := 0
	for power := int32(0); power <= 255; power++ {
		for dmin := int32(0); dmin <= 255; dmin++ {
			for dmax := int32(0); dmax <= 255; dmax++ {
				gb, gs := sim.SpellColumnDamage(dmin, dmax, power)
				wb, ws := recordDamageBinary64(dmin, dmax, power)
				if gb != int64(wb) || gs != int64(ws) {
					failures++
					if failures <= 10 {
						t.Errorf("power %d columns %d/%d: integer %d/%d, binary64 %d/%d", power, dmin, dmax, gb, gs, wb, ws)
					}
				}
			}
		}
	}
	if failures != 0 {
		t.Fatalf("%d of %d inputs differ", failures, 256*256*256)
	}
	for _, c := range [][3]int32{{-1, -1, 100}, {-5, 20, 50}, {12, -3, 100}, {300, 400, 30}, {1000, 70000, 255}} {
		gb, gs := sim.SpellRecordDamage(c[0], c[1], c[2])
		wb, ws := recordDamageBinary64(c[0], c[1], c[2])
		if gb != wb || gs != ws {
			t.Errorf("columns %d/%d power %d: integer %d/%d, binary64 %d/%d", c[0], c[1], c[2], gb, gs, wb, ws)
		}
	}
}

func TestModDamageColumnsKeepBinary64Rounding(t *testing.T) {
	for power := int32(0); power <= 255; power++ {
		f := float64(float64(power)/30.0) + 1.0
		for _, columns := range [][2]int32{{300, 600}, {45, 300}, {0, 300}, {255, 256}, {256, 256}, {6898, 6898}, {0, 6898}} {
			b, s := sim.SpellColumnDamage(columns[0], columns[1], power)
			wb := int64(float64(float64(columns[0]) * f))
			ws := int64(float64(float64(float64(columns[1])*f) - float64(wb)))
			if b != wb || s != ws {
				t.Fatalf("columns %v power %d: %d/%d, want binary64 %d/%d", columns, power, b, s, wb, ws)
			}
		}
	}
	if b, _ := sim.SpellColumnDamage(45, 300, 12); b != 62 {
		t.Fatalf("whole row rounded base %d, want 62 instead of integer 63", b)
	}
}

// TestSpellRecordDamageLeavesTheIntegerFormula is the loss control: the
// former 64-bit integer formula differs from binary64 on these inputs.
func TestSpellRecordDamageLeavesTheIntegerFormula(t *testing.T) {
	differs := 0
	for power := int32(0); power <= 100; power++ {
		for dmin := int32(1); dmin <= 255; dmin++ {
			old := int64(dmin) * int64(power+30) / 30
			b, _ := sim.SpellRecordDamage(dmin, 0, power)
			if int64(b) != old {
				differs++
			}
		}
	}
	if differs == 0 {
		t.Fatal("the integer formula agrees everywhere; the binary64 form is not exercised")
	}
	// 45 * (12/30 + 1) is 62.99999999999999 in binary64: the record keeps 62.
	if b, _ := sim.SpellRecordDamage(45, 0, 12); b != 62 {
		t.Fatalf("columns 45 at power 12: base %d, want 62", b)
	}
	// 200 * (100/30 + 1) is 866, whose low byte is 98.
	if b, _ := sim.SpellRecordDamage(200, 0, 100); b != 98 {
		t.Fatalf("column 200 at power 100: base %d, want the low byte 98", b)
	}
}

func TestPoisonMagnitudeEqualsBinary64(t *testing.T) {
	want := func(m, power int32) int32 {
		f := float64(float64(power)/30.0) + 1.0
		return int32(int16(int64(float64(float64(m) * f))))
	}
	failures := 0
	check := func(m, power int32) {
		if got, w := sim.PoisonMagnitude(m, power), want(m, power); got != w {
			failures++
			if failures <= 10 {
				t.Errorf("magnitude %d power %d: integer %d, binary64 %d", m, power, got, w)
			}
		}
	}
	for power := int32(0); power <= 255; power++ {
		for m := int32(math.MinInt16); m <= math.MaxInt16; m++ {
			check(m, power)
		}
	}
	for power := int32(math.MinInt16); power <= math.MaxInt16; power++ {
		for _, m := range []int32{-32768, -300, -2, -1, 1, 2, 7, 32767} {
			check(m, power)
		}
	}
	if failures != 0 {
		t.Fatalf("%d inputs differ", failures)
	}
}

func TestResistedSpellDamageEqualsBinary64(t *testing.T) {
	failures := 0
	for r := int32(math.MinInt16); r <= math.MaxInt16; r++ {
		for v := int64(0); v <= 510; v++ {
			want := v
			if r != 0 {
				want = int64(int32(int64(float64(float64(float64(v*(100-int64(r)))/100.0) + 0.75))))
			}
			if got := sim.ResistedSpellDamage(v, int16(r)); got != want {
				failures++
				if failures <= 10 {
					t.Errorf("v %d resistance %d: integer %d, binary64 %d", v, r, got, want)
				}
			}
		}
	}
	if failures != 0 {
		t.Fatalf("%d inputs differ", failures)
	}
}
