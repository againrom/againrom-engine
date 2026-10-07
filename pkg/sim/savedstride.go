package sim

import "encoding/binary"

// NativeStrideCompatible qualifies a serialized crossing by all retained
// operands of the native rated-step law. It does not identify who wrote the
// SAV, or establish that an original boundary callback ran. DIV-1189 lets this
// exact shape finish its accepted stride and hand the remaining dynamic route
// to the existing native mover. Other original crossings retain their named
// callback guard and original diagonal-boundary policy.
func (m SavedActorMotion) NativeStrideCompatible() bool {
	word := func(at int) uint16 { return binary.LittleEndian.Uint16(m.Mover[at:]) }
	total, elapsed := word(0xaa), word(0xac)
	from, to := word(0xa6), word(0x80)
	if m.ActorAction != 1 || len(m.StaticRoute) != 0 || len(m.DynamicRoute) == 0 || m.DynamicRoute[0] != to ||
		m.Position.Cell != m.Position.PackedCell || pendingSavedMotionTurn(m) || m.Mover[0x9d] != 0 || m.Mover[0xa4] != 0 ||
		total == 0 || elapsed == 0 || elapsed >= total || word(0xa8) > 255 || word(0xae) > 7 {
		return false
	}
	s := NativeStride{
		Present: true, FromX: int32(from & 255), FromY: int32(from >> 8),
		ToX: int32(to & 255), ToY: int32(to >> 8), Rate: uint8(word(0xa8)),
		StepX: int8(m.Mover[0xb0]), StepY: int8(m.Mover[0xb1]), Direction: uint8(word(0xae)),
	}
	// Reuse the native endpoint, direction, rate, signed-axis and duration
	// invariants without recalculating current equipment or terrain speed.
	e := Entity{X: s.ToX, Y: s.ToY, HP: 1, MaxHP: 1, Transit: total - elapsed, TransitTotal: total, Stride: s}
	if strideFault(e) != nil {
		return false
	}
	x := s.FromX*256 + 128 + int32(elapsed)*int32(s.StepX)
	y := s.FromY*256 + 128 + int32(elapsed)*int32(s.StepY)
	return x == int32(m.Position.Cell&255)*256+int32(m.Position.FineX) &&
		y == int32(m.Position.Cell>>8)*256+int32(m.Position.FineY)
}
