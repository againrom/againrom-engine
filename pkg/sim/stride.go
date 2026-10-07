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

// retainStride consumes the rate already computed for this accepted step.
// It does not recompute terrain, speed or duration, or move the actor again.
func (e *Entity) retainStride(from cell, rate int32) {
	dx, dy := int64(e.X)-int64(from.x), int64(e.Y)-int64(from.y)
	d := dirOfSigns[signIndex(int32(dy))][signIndex(int32(dx))]
	axis := rate
	if dx != 0 && dy != 0 {
		axis = diagonalStep(rate)
	}
	if axis < rateFloor {
		axis = rateFloor
	}
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
	axis := int32(s.Rate)
	if dx != 0 && dy != 0 {
		axis = diagonalStep(axis)
	}
	if axis < rateFloor {
		axis = rateFloor
	}
	if int64(s.StepX) != dx*int64(axis) || int64(s.StepY) != dy*int64(axis) {
		return fmt.Errorf("native stride axis steps differ from its accepted rate")
	}
	if int32(e.TransitTotal) != (subCell+axis-1)/axis {
		return fmt.Errorf("native stride duration differs from its accepted axis step")
	}
	return nil
}
