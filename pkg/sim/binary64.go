package sim

import "math/bits"

// double is an IEEE 754 binary64 value held in integers: the value is
// mant * 2^exp with mant zero or in [2^52, 2^53). Each operation rounds to
// nearest, ties to even, as binary64 does. The simulation package admits no
// floating point, and the original forms several spell operands in binary64
// before truncating them. Values stay far inside the normal range here, so
// subnormals, infinities and NaN are not represented.
type double struct {
	neg  bool
	mant uint64
	exp  int
}

// roundDouble rounds the exact magnitude (hi:lo) * 2^exp, plus a nonzero
// remainder below its last bit when sticky is set, to 53 significant bits.
func roundDouble(neg bool, hi, lo uint64, exp int, sticky bool) double {
	n := 64 + bits.Len64(hi)
	if hi == 0 {
		n = bits.Len64(lo)
	}
	if n == 0 {
		return double{}
	}
	if n <= 53 {
		// Exact: a division with a remainder always supplies more bits.
		return double{neg: neg, mant: lo << uint(53-n), exp: exp - (53 - n)}
	}
	shift := uint(n - 53)
	kept, dropHi, dropLo := shiftRight128(hi, lo, shift)
	half := shift - 1
	halfHi, halfLo := uint64(0), uint64(0)
	if half >= 64 {
		halfHi = 1 << (half - 64)
	} else {
		halfLo = 1 << half
	}
	above := dropHi > halfHi || dropHi == halfHi && dropLo > halfLo
	tie := dropHi == halfHi && dropLo == halfLo
	if above || tie && (sticky || kept&1 == 1) {
		kept++
		if kept == 1<<53 {
			kept >>= 1
			shift++
		}
	}
	return double{neg: neg, mant: kept, exp: exp + int(shift)}
}

// shiftRight128 answers (hi:lo) >> s, which must fit in 64 bits, and the
// dropped low s bits as a 128-bit value.
func shiftRight128(hi, lo uint64, s uint) (kept, dropHi, dropLo uint64) {
	switch {
	case s == 0:
		return lo, 0, 0
	case s < 64:
		return hi<<(64-s) | lo>>s, 0, lo & (1<<s - 1)
	case s == 64:
		return hi, 0, lo
	}
	return hi >> (s - 64), hi & (1<<(s-64) - 1), lo
}

func doubleFromInt(v int64) double {
	neg := v < 0
	m := uint64(v)
	if neg {
		m = -m
	}
	return roundDouble(neg, 0, m, 0, false)
}

// doubleQuo is the binary64 quotient a/b of two integers, b nonzero.
func doubleQuo(a, b int64) double {
	neg := (a < 0) != (b < 0)
	ua, ub := uint64(a), uint64(b)
	if a < 0 {
		ua = -ua
	}
	if b < 0 {
		ub = -ub
	}
	whole, rest := ua/ub, ua%ub
	frac, rem := bits.Div64(rest, 0, ub)
	if whole != 0 {
		return roundDouble(neg, whole, frac, -64, rem != 0)
	}
	// A quotient below one takes 64 more fraction bits, so at least 65
	// significant bits reach the rounding.
	low, rem := bits.Div64(rem, 0, ub)
	return roundDouble(neg, frac, low, -128, rem != 0)
}

func (x double) mul(y double) double {
	hi, lo := bits.Mul64(x.mant, y.mant)
	return roundDouble(x.neg != y.neg, hi, lo, x.exp+y.exp, false)
}

func (x double) add(y double) double {
	if x.mant == 0 {
		return y
	}
	if y.mant == 0 {
		return x
	}
	if x.exp < y.exp {
		x, y = y, x
	}
	d := uint(x.exp - y.exp)
	if d > 75 {
		// y is below 2^-74 of x and cannot move a rounded result.
		return x
	}
	// x.mant << d fits in 128 bits for d <= 75.
	xh, xl := uint64(0), x.mant
	if d >= 64 {
		xh, xl = x.mant<<(d-64), 0
	} else if d > 0 {
		xh, xl = x.mant>>(64-d), x.mant<<d
	}
	if x.neg == y.neg {
		lo, carry := bits.Add64(xl, y.mant, 0)
		return roundDouble(x.neg, xh+carry, lo, y.exp, false)
	}
	// Opposite signs: subtract the smaller magnitude from the larger.
	if xh == 0 && xl < y.mant {
		return roundDouble(y.neg, 0, y.mant-xl, y.exp, false)
	}
	lo, borrow := bits.Sub64(xl, y.mant, 0)
	return roundDouble(x.neg, xh-borrow, lo, y.exp, false)
}

func (x double) sub(y double) double {
	y.neg = !y.neg
	return x.add(y)
}

// trunc is the conversion toward zero to a signed 64-bit integer. Every
// operand here stays below 2^63.
func (x double) trunc() int64 {
	var m uint64
	switch {
	case x.exp >= 0:
		m = x.mant << uint(x.exp)
	case x.exp > -64:
		m = x.mant >> uint(-x.exp)
	}
	if x.neg {
		return -int64(m)
	}
	return int64(m)
}
