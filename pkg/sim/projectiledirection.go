package sim

// ProjectileDirection is the sixteen-way direction from a point to one (dx,
// dy) away, x east and y south: 0 north, 4 east, 8 south, 12 west (ANIM-138).
// The quadrant split is by signed 32-bit slope comparisons at 1/4, 3/4, 4/3
// and 4, so the sectors are not equal angles; the products wrap as the
// original's do, and the zero vector answers 4.
func ProjectileDirection(dx, dy int32) int32 {
	a, b := dx, dy
	if a < 0 {
		a = -a
	}
	if b < 0 {
		b = -b
	}
	var q int32
	switch {
	case a >= 4*b:
		q = 0
	case 3*a >= 4*b:
		q = 1
	case b >= 4*a:
		q = 4
	case 3*b >= 4*a:
		q = 3
	default:
		q = 2
	}
	if dy > 0 {
		if dx > 0 {
			return (4 + q) & 15
		}
		return (12 - q) & 15
	}
	if dx < 0 {
		return (12 + q) & 15
	}
	return (4 - q) & 15
}
