package sim

import "math/big"

// pickerRings and pickerStepCap bound picker B: eight rings, and at most 100
// steps of its two walkers on one ring (MOVE-ALT-020, MOVE-099).
const (
	pickerRings   = 8
	pickerStepCap = 100
)

// pickerDirections is the walkers' direction table, east, south, west, north
// three times over; a walker turns by moving its index (MOVE-ALT-020).
var pickerDirections = [12]cell{
	{1, 0}, {0, 1}, {-1, 0}, {0, -1},
	{1, 0}, {0, 1}, {-1, 0}, {0, -1},
	{1, 0}, {0, 1}, {-1, 0}, {0, -1},
}

// pickerB is the near search's substitute for a pursuer whose goal the wave
// left unlabelled: the contact ring around its victim, ring by ring, keeping
// the first strictly lowest label its two walkers probe; the first ring that
// finds one ends the scan. The mover's own start cell carries label 0 and may
// be returned, which leaves the near search with no step (MOVE-099,
// MOVE-100, MOVE-ALT-020).
func (w *World) pickerB(s *routeScratch, self int, start cell) (cell, bool) {
	m := w.entities[self]
	ti := indexOfEntity(w.entities, m.AttackTarget)
	if !m.HasAttackTarget || m.AttackTargetKind != AttackTargetUnit || ti < 0 {
		return cell{}, false
	}
	t := w.entities[ti]
	nM, nT := footprintSide(m.TokenSize), footprintSide(t.TokenSize)
	// Fine footprint centres, 256 to a cell.
	xm, ym := int64(m.X)*256+int64(nM)*128, int64(m.Y)*256+int64(nM)*128
	xt, yt := int64(t.X)*256+int64(nT)*128, int64(t.Y)*256+int64(nT)*128
	edge := (bearingCode(xm-xt, ym-yt)+2)>>2&3 + 4

	var best cell
	var bestLabel uint64
	found := false
	probe := func(c cell) {
		l, ok := w.label(s, start, c.x, c.y)
		if !ok || !w.restFree(s, self, c.x, c.y) || found && l >= bestLabel {
			return
		}
		best, bestLabel, found = c, l, true
	}
	for r := int32(1); r <= pickerRings && !found; r++ {
		left, right := t.X-nM-r+1, t.X+nT+r-1
		top, bottom := t.Y-nM-r+1, t.Y+nT+r-1
		var entry cell
		switch edge {
		case 4:
			entry = cell{x: pickerCrossing(xt, yt, xm, ym, top), y: top}
		case 5:
			entry = cell{x: right, y: pickerCrossing(yt, xt, ym, xm, right)}
		case 6:
			entry = cell{x: pickerCrossing(xt, yt, xm, ym, bottom), y: bottom}
		default:
			entry = cell{x: left, y: pickerCrossing(yt, xt, ym, xm, left)}
		}
		steps := 2*(right-left) + 1
		if steps > pickerStepCap {
			steps = pickerStepCap
		}
		walkers := [2]cell{entry, entry}
		dirs := [2]int{int(edge), int(edge) + 2}
		for k := int32(0); k < steps; k++ {
			probe(walkers[0])
			probe(walkers[1])
			for n := range walkers {
				if pickerAtSideEnd(walkers[n], pickerDirections[(dirs[n]%12+12)%12], left, right, top, bottom) {
					dirs[n] += 1 - 2*n
				}
				d := pickerDirections[(dirs[n]%12+12)%12]
				walkers[n].x += d.x
				walkers[n].y += d.y
			}
		}
	}
	return best, found
}

// pickerAtSideEnd reports whether a walker at c heading d stands at the end of
// its box side, where it turns before its next move. A walker that entered off
// the box never meets that end and runs straight on (MOVE-098).
func pickerAtSideEnd(c, d cell, left, right, top, bottom int32) bool {
	switch {
	case d.x > 0:
		return c.x == right
	case d.y > 0:
		return c.y == bottom
	case d.x < 0:
		return c.x == left
	}
	return c.y == top
}

// bearingCode is the 16-way bearing of a mover's fine centre from its target's
// with dx = mover - target and dy growing southward: 0 just east of north,
// clockwise, sector edges at the axes, the diagonals and the 2:1 slopes
// (MOVE-097).
func bearingCode(dx, dy int64) int32 {
	a, b := dx, dy
	if a < 0 {
		a = -a
	}
	if b < 0 {
		b = -b
	}
	// k is the sector within the quadrant, counted from the north-south axis.
	var k int32
	switch {
	case b > 2*a:
		k = 0
	case a <= b:
		k = 1
	case a <= 2*b:
		k = 2
	default:
		k = 3
	}
	switch {
	case dx > 0 && dy <= 0:
		return k
	case dx > 0:
		return 7 - k
	case dy > 0:
		return 8 + k
	}
	return 15 - k
}

// pickerCrossing is where the line through the two fine centres crosses the
// ring row (or column) line, in cells: with along the coordinate the line is
// read for and across the one line names, slope = (alongT - alongM) /
// (acrossT - acrossM) rounded to single precision (a zero divisor taken as 1),
// c = trunc(alongT - acrossT * slope) and the crossing
// trunc((line*256 + 0x80) * slope + c) >> 8. With the slope held to 24 bits
// every later product and sum is exact in the double precision the original
// evaluates it at, so integer arithmetic reproduces it (MOVE-098).
func pickerCrossing(alongT, acrossT, alongM, acrossM int64, line int32) int32 {
	q := acrossT - acrossM
	if q == 0 {
		q = 1
	}
	mant, shift := singleQuotient(alongT-alongM, q)
	if shift < 0 {
		mant.Lsh(mant, uint(-shift))
		shift = 0
	}
	scale := new(big.Int).Lsh(big.NewInt(1), uint(shift))
	// c = trunc((alongT * 2^shift - acrossT * mant) / 2^shift)
	num := new(big.Int).Mul(big.NewInt(alongT), scale)
	num.Sub(num, new(big.Int).Mul(big.NewInt(acrossT), mant))
	c := new(big.Int).Quo(num, scale)
	// trunc(((line*256 + 0x80) * mant + c * 2^shift) / 2^shift) >> 8
	v := new(big.Int).Mul(big.NewInt(int64(line)*256+0x80), mant)
	v.Add(v, new(big.Int).Mul(c, scale))
	v.Quo(v, scale)
	v.Rsh(v, 8)
	return int32(v.Int64())
}

// singleQuotient is p/q rounded to the nearest single-precision value, ties
// to even, as mant / 2^shift with mant below 2^24. Rounding to 53 bits first,
// as the original's division does before its store, cannot change this: no
// quotient of two integers below 2^26 lies within 2^-53 of a 24-bit midpoint
// unless it is that midpoint.
func singleQuotient(p, q int64) (*big.Int, int) {
	neg := (p < 0) != (q < 0)
	num, den := new(big.Int).Abs(big.NewInt(p)), new(big.Int).Abs(big.NewInt(q))
	if num.Sign() == 0 {
		return new(big.Int), 0
	}
	// 2^23 <= num * 2^shift / den < 2^24.
	shift := 23 + den.BitLen() - num.BitLen()
	scaled := func(shift int) (*big.Int, *big.Int) {
		n, d := new(big.Int).Set(num), new(big.Int).Set(den)
		if shift >= 0 {
			n.Lsh(n, uint(shift))
		} else {
			d.Lsh(d, uint(-shift))
		}
		return n, d
	}
	lo, hi := new(big.Int).Lsh(big.NewInt(1), 23), new(big.Int).Lsh(big.NewInt(1), 24)
	for {
		n, d := scaled(shift)
		m := new(big.Int).Quo(n, d)
		switch {
		case m.Cmp(lo) < 0:
			shift++
			continue
		case m.Cmp(hi) >= 0:
			shift--
			continue
		}
		rem := new(big.Int).Sub(n, new(big.Int).Mul(m, d))
		if c := rem.Lsh(rem, 1).Cmp(d); c > 0 || c == 0 && m.Bit(0) == 1 {
			m.Add(m, big.NewInt(1))
		}
		if m.Cmp(hi) == 0 {
			m.Rsh(m, 1)
			shift--
		}
		if neg {
			m.Neg(m)
		}
		return m, shift
	}
}
