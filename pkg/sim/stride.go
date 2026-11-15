package sim

import "fmt"

// NativeStride retains the actual inputs of the last accepted rated cell step.
// Direction is the native clockwise-from-north octant 0..7, not Facing: a
// cast or turn may change the body direction while this crossing still owes
// ticks. Rate and signed axis steps likewise survive later speed, group and
// terrain changes. This is native provenance, not an original SAV mover block.
//
// Present survives the final transit payment. A non-walk relocation, death or
// accepted unrated step invalidates it. Absent may coexist with an outstanding
// transit: older saves and relocated actors have no recoverable stride inputs.
type NativeStride struct {
	Present                bool
	FromX, FromY, ToX, ToY int32
	Rate                   uint8
	StepX, StepY           int8
	Direction              uint8
}

func (e *Entity) clearStride() { e.Stride = NativeStride{} }

// strideStep is one call of the step on a position record, cell*256 plus the
// sub byte per axis: it adds the signed strides and, when corrected, takes
// one off x in octant 1 or 5 if both new sub bytes are 0 (MOVE-174). Within
// one crossing from a centre that fires once, on call 128/|stride| when the
// stride divides 128 (MOVE-200).
func strideStep(x, y int32, sx, sy int8, octant uint16, corrected bool) (int32, int32) {
	x, y = x+int32(sx), y+int32(sy)
	if corrected && (octant == 1 || octant == 5) && x&255 == 0 && y&255 == 0 {
		x--
	}
	return x, y
}

// finePoint is the position record after elapsed of the crossing's total
// calls of strideStep from the start centre, and the destination centre
// once the count reaches total. corrected is the World's StrideCorrected.
func (s NativeStride) finePoint(elapsed, total int32, corrected bool) (x, y int32) {
	if elapsed >= total {
		return s.ToX*256 + 128, s.ToY*256 + 128
	}
	x, y = s.FromX*256+128, s.FromY*256+128
	for range elapsed {
		x, y = strideStep(x, y, s.StepX, s.StepY, uint16(s.Direction), corrected)
	}
	return x, y
}

// StrideCorrected reports whether w's game steps with the first game's
// stride law, the flag ProjectActorMotion and NativeBasisNow take.
func (w *World) StrideCorrected() bool { return !w.game.uncorrectedStride }

// strideAxis is the per-axis stride of rate v: v straight, trunc(0.707v)
// diagonally (MOVE-200), which is 0 at v 1 in the first game; the second
// game (floored) takes 1 there.
func strideAxis(v int32, diagonal, floored bool) int32 {
	if !diagonal {
		return v
	}
	if axis := diagonalStep(v); axis >= rateFloor || !floored {
		return axis
	}
	return rateFloor
}

// retainStride consumes the rate already computed for this accepted step.
// It does not recompute terrain, speed or duration, or move the actor again.
func (e *Entity) retainStride(from cell, rate int32, floored bool) {
	dx, dy := int64(e.X)-int64(from.x), int64(e.Y)-int64(from.y)
	d := dirOfSigns[signIndex(int32(dy))][signIndex(int32(dx))]
	axis := strideAxis(rate, dx != 0 && dy != 0, floored)
	e.Stride = NativeStride{Present: true, FromX: from.x, FromY: from.y,
		ToX: e.X, ToY: e.Y, Rate: uint8(rate), StepX: int8(dx * int64(axis)),
		StepY: int8(dy * int64(axis)), Direction: uint8(d)}
}

func strideFault(e Entity) error {
	s := e.Stride
	if !s.Present {
		if s != (NativeStride{}) {
			return fmt.Errorf("absent native stride carries state")
		}
		return nil
	}
	if !e.Alive() || e.TransitTotal == 0 || e.Transit >= e.TransitTotal {
		return fmt.Errorf("native stride lacks a living valid transit")
	}
	if e.X != s.ToX || e.Y != s.ToY {
		return fmt.Errorf("native stride destination differs from actor position")
	}
	dx, dy := int64(s.ToX)-int64(s.FromX), int64(s.ToY)-int64(s.FromY)
	if dx < -1 || dx > 1 || dy < -1 || dy > 1 || dx == 0 && dy == 0 {
		return fmt.Errorf("native stride endpoints are not adjacent")
	}
	if s.Direction >= directions || int64(stepOf[s.Direction][0]) != dx || int64(stepOf[s.Direction][1]) != dy {
		return fmt.Errorf("native stride direction differs from its endpoints")
	}
	if s.Rate < rateFloor || s.Rate > rateCeil {
		return fmt.Errorf("native stride rate %d is outside the native rate law", s.Rate)
	}
	// Either law's stride is accepted: they differ only for a rate-1
	// diagonal, which an older form and the second game step by 1.
	diagonal := dx != 0 && dy != 0
	axis := strideAxis(int32(s.Rate), diagonal, true)
	steps := func(a int32) bool { return int64(s.StepX) == dx*int64(a) && int64(s.StepY) == dy*int64(a) }
	if !steps(axis) && !steps(strideAxis(int32(s.Rate), diagonal, false)) {
		return fmt.Errorf("native stride axis steps differ from its accepted rate")
	}
	if int32(e.TransitTotal) != (subCell+axis-1)/axis {
		return fmt.Errorf("native stride duration differs from its accepted axis step")
	}
	return nil
}
