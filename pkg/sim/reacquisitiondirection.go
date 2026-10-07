package sim

// headingOf is the eight-way byte heading of a delta in 1/256 cell units: the
// axis when one magnitude exceeds twice the other, the diagonal of the two
// signs otherwise, and 224 for a coincident delta (AI-361, MAGIC-242).
func headingOf(dx, dy int64) uint8 {
	if dx == 0 && dy == 0 {
		return 224
	}
	x, y := abs64(dx), abs64(dy)
	if x > 2*y {
		if dx < 0 {
			return 192
		}
		return 64
	}
	if y > 2*x {
		if dy < 0 {
			return 0
		}
		return 128
	}
	if dx < 0 {
		if dy < 0 {
			return 224
		}
		return 160
	}
	if dy < 0 {
		return 32
	}
	return 96
}

// headingBetween is the heading from a to b measured between the footprint
// centres on the fine grid, the sub-cell bytes included (MAGIC-242).
func (w *World) headingBetween(a, b Entity) uint8 {
	ax, ay := w.moverFinePoint(a)
	bx, by := w.moverFinePoint(b)
	return headingOf(bx-ax, by-ay)
}
