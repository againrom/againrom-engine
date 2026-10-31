package intmath

import (
	"math"
	"testing"
)

type pixels int32

func TestAbs(t *testing.T) {
	if Abs(-5) != 5 || Abs(5) != 5 || Abs(0) != 0 {
		t.Error("Abs(int)")
	}
	if Abs(int32(-7)) != 7 || Abs(int64(-7)) != 7 || Abs(int8(-7)) != 7 || Abs(int16(-7)) != 7 {
		t.Error("Abs(sized)")
	}
	if Abs(pixels(-3)) != pixels(3) {
		t.Error("Abs(named)")
	}
}

func TestAbsOfTheSmallestValueIsItself(t *testing.T) {
	if got := Abs(math.MinInt); got != math.MinInt {
		t.Errorf("int: %d", got)
	}
	if got := Abs(int32(math.MinInt32)); got != math.MinInt32 {
		t.Errorf("int32: %d", got)
	}
	if got := Abs(int64(math.MinInt64)); got != math.MinInt64 {
		t.Errorf("int64: %d", got)
	}
	if got := Abs(int8(math.MinInt8)); got != math.MinInt8 {
		t.Errorf("int8: %d", got)
	}
	if got := Abs(int16(math.MinInt16)); got != math.MinInt16 {
		t.Errorf("int16: %d", got)
	}
	if Abs(int32(math.MaxInt32)) != math.MaxInt32 || Abs(int32(-math.MaxInt32)) != math.MaxInt32 {
		t.Error("int32 extremes")
	}
}
