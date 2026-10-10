package rules

import (
	"fmt"
	"math"
)

// The school training step is stated here once. Every route that trains a
// skill in town (a member restored from an original city save, a member with a
// source actor, any other member) and the price the school displays read these
// two functions; internal/archtest refuses a second computation.
//
// Both follow the original's arithmetic: ftol of a pow(1.1, n) product, kept
// as the low dword of the signed-qword conversion (HERO-SKILLBUY-076,
// HERO-XP-077, TOWN-GENERAL-107; High). The original tests no cap here, so
// neither does the rule.

// SchoolPrice is the gold the school asks to raise a skill whose maintained
// base level is base: ftol(1.1^base * 200). From base 170 the low dword reads
// negative; a caller refuses a price below 1, as the school's Train does.
func SchoolPrice(base int32) (int32, error) {
	return schoolFTOL(math.Pow(1.1, float64(base)) * 200)
}

// SchoolTrainedXP is the experience a slot holds after the school raises it to
// level: S(level) + 1, where S(n) = ftol((1.1^n - 1) * 1000) is the curve the
// skill experience table holds. The sum wraps as the original's dword does.
func SchoolTrainedXP(level int32) (int32, error) {
	s, err := schoolFTOL((math.Pow(1.1, float64(level)) - 1) * 1000)
	if err != nil {
		return 0, err
	}
	return int32(uint32(s) + 1), nil
}

// schoolFTOL is the low dword of a truncating signed-qword conversion. Outside
// that conversion's domain the original's result is Unknown, so the caller
// refuses before any purse or slot changes.
func schoolFTOL(v float64) (int32, error) {
	if math.IsNaN(v) || math.IsInf(v, 0) || v < -0x1p63 || v >= 0x1p63 {
		return 0, fmt.Errorf("school training arithmetic exceeds the proven signed-qword domain")
	}
	return int32(int64(v)), nil
}
