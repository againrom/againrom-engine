package sim

// heading is a facing byte one of the laws below produced (MOVE-116). Only
// this file converts to it; internal/archtest refuses a conversion elsewhere.
type heading uint8

// headingOf is AI-444's branch tree: the axis past 2:1, the diagonal at and
// inside it, 224 for the zero vector. AI-446 runs it on whole cells.
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

// actorCentre is the fine point plus (size-1)*128, truncated to 16 bits
// (AI-444).
func (w *World) actorCentre(e Entity) (int64, int64) {
	x, y := w.moverFinePoint(e)
	return int64(uint16(x)), int64(uint16(y))
}

// headingBetween is the heading from a to b between footprint centres, the
// turn before an attack or a cast at an actor (AI-443, AI-444, MAGIC-242).
func (w *World) headingBetween(a, b Entity) uint8 {
	ax, ay := w.actorCentre(a)
	bx, by := w.actorCentre(b)
	return headingOf(bx-ax, by-ay)
}

// structureHeading is headingBetween toward a structure, a size-1 token at
// its anchor cell's centre (AI-445, MOVE-184).
func (w *World) structureHeading(a Entity, s Structure) uint8 {
	ax, ay := w.actorCentre(a)
	bx, by := int64(uint16(s.Col*256+128)), int64(uint16(s.Row*256+128))
	return headingOf(bx-ax, by-ay)
}

// cellHeading is the heading from the anchor cell to (x, y) on whole cells:
// a cast at a cell, a structure use and a wall's direction (AI-446).
func cellHeading(fromX, fromY, x, y int32) uint8 {
	return headingOf(int64(x)-int64(fromX), int64(y)-int64(fromY))
}

// stepHeading is the signs of the next cell's centre minus the anchor fine
// point; the zero vector gives (facing << 5) & 0xff (MOVE-115).
func (w *World) stepHeading(e Entity, x, y int32) uint8 {
	fx, fy := w.savedProjectileTargetPoint(e)
	dx, dy := x*256+128-fx, y*256+128-fy
	if f, ok := facingToward(dx, dy); ok {
		return f
	}
	return e.Facing << 5
}

// turnToActor turns i toward ti; a cast at the caster itself does not turn.
func (w *World) turnToActor(i, ti int) bool {
	if w.entities[i].ID == w.entities[ti].ID {
		return false
	}
	return w.turnTo(i, heading(w.headingBetween(w.entities[i], w.entities[ti])))
}

func (w *World) turnToStructure(i int, s Structure) bool {
	return w.turnTo(i, heading(w.structureHeading(w.entities[i], s)))
}

func (w *World) turnToCell(i int, x, y int32) bool {
	return w.turnTo(i, heading(cellHeading(w.entities[i].X, w.entities[i].Y, x, y)))
}

func (w *World) turnToStep(i int, x, y int32) bool {
	return w.turnTo(i, heading(w.stepHeading(w.entities[i], x, y)))
}

// projectileAimPoint supplies the cached-centre geometry for actor targets.
// AI-674, AI-675
func (w *World) projectileAimPoint(e Entity) (int32, int32) {
	if w.SecondGame() {
		return w.savedProjectileTargetPoint(e)
	}
	x, y := w.moverFinePoint(e)
	return int32(x), int32(y)
}
