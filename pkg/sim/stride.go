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
// sub byte per axis: it adds the signed strides and takes one off x in octant
// 1 or 5 if both new sub bytes are 0, in both games (MOVE-174, R2-ENGINE-1213).
// Within one crossing from a centre that fires once, on call 128/|stride| when
// the stride divides 128 (MOVE-200).
func strideStep(x, y int32, sx, sy int8, octant uint16) (int32, int32) {
	x, y = x+int32(sx), y+int32(sy)
	if (octant == 1 || octant == 5) && x&255 == 0 && y&255 == 0 {
		x--
	}
	return x, y
}

// finePoint is the position record after elapsed of the crossing's total
// calls of strideStep from the start centre, and the destination centre
// once the count reaches total.
func (s NativeStride) finePoint(elapsed, total int32) (x, y int32) {
	if elapsed >= total {
		return s.ToX*256 + 128, s.ToY*256 + 128
	}
	x, y = s.FromX*256+128, s.FromY*256+128
	for range elapsed {
		x, y = strideStep(x, y, s.StepX, s.StepY, uint16(s.Direction))
	}
	return x, y
}

// strideAxis is the per-axis stride of rate v: v straight, trunc(0.707v)
// diagonally (MOVE-200, R2-ENGINE-1470), which is 0 at v 1 in both games; the
// advance count then divides by 1 (transitOf).
func strideAxis(v int32, diagonal bool) int32 {
	if !diagonal {
		return v
	}
	return diagonalStep(v)
}

// retainStride consumes the rate already computed for this accepted step.
// It does not recompute terrain, speed or duration, or move the actor again.
func (e *Entity) retainStride(from, to cell, rate int32) {
	dx, dy := int64(to.x)-int64(from.x), int64(to.y)-int64(from.y)
	d := dirOfSigns[signIndex(int32(dy))][signIndex(int32(dx))]
	axis := strideAxis(rate, dx != 0 && dy != 0)
	e.Stride = NativeStride{Present: true, FromX: from.x, FromY: from.y,
		ToX: to.x, ToY: to.y, Rate: uint8(rate), StepX: int8(dx * int64(axis)),
		StepY: int8(dy * int64(axis)), Direction: uint8(d)}
}

// committed reports whether a stride's position has crossed into its
// destination after elapsed of its total calls: the y cell byte has changed,
// or the x byte for a step with no y motion (strideEntersStep). The step's
// logical cell is its origin until then and its destination from then on
// (MOVE-088, MOVE-194).
func (s NativeStride) committed(elapsed, total int32) bool {
	c := strideCell(s, elapsed, total)
	if s.StepY != 0 {
		return c.y != s.FromY
	}
	return c.x != s.FromX
}

// stepping reports whether e is inside a rated step: a retained stride with
// transit still owed.
func (e Entity) stepping() bool { return e.Stride.Present && e.Transit > 0 }

// StepCell is the cell e stands on, or during a rated step the cell the step
// enters. A view draws the step toward it while the logical cell X, Y stays
// the origin until the crossing.
func (e Entity) StepCell() (x, y int32) {
	if e.stepping() {
		return e.Stride.ToX, e.Stride.ToY
	}
	return e.X, e.Y
}

// heldCell is the second anchor a stepping mover holds on its layer's plane:
// the destination it claimed at the step's start until the crossing, then
// the origin it vacates until it lands centred (MOVE-CLAIM-007).
func (e Entity) heldCell() (cell, bool) {
	if !e.stepping() {
		return cell{}, false
	}
	if e.X == e.Stride.ToX && e.Y == e.Stride.ToY {
		return cell{x: e.Stride.FromX, y: e.Stride.FromY}, true
	}
	return cell{x: e.Stride.ToX, y: e.Stride.ToY}, true
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
	// The logical cell is the origin before the crossing and the destination
	// after it. A form an earlier build wrote holds the destination from the
	// start, and that step finishes as it began.
	atTo, atFrom := e.X == s.ToX && e.Y == s.ToY, e.X == s.FromX && e.Y == s.FromY
	if !atTo && !(atFrom && !s.committed(int32(e.TransitTotal-e.Transit), int32(e.TransitTotal))) {
		return fmt.Errorf("native stride logical cell differs from actor position")
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
	// A rate-1 diagonal strides 0; a form an earlier build wrote strides 1.
	diagonal := dx != 0 && dy != 0
	axis := strideAxis(int32(s.Rate), diagonal)
	steps := func(a int32) bool { return int64(s.StepX) == dx*int64(a) && int64(s.StepY) == dy*int64(a) }
	if !steps(axis) && !(diagonal && s.Rate == 1 && steps(1)) {
		return fmt.Errorf("native stride axis steps differ from its accepted rate")
	}
	if int32(e.TransitTotal) != transitOf(int32(s.Rate), diagonal) {
		return fmt.Errorf("native stride duration differs from its accepted axis step")
	}
	return nil
}
