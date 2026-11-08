package sav

import "encoding/binary"

// Order block offsets of the two withdrawal thresholds: Withdraw at +0x40
// and Wimpy at +0x44, signed absolute health (AI-WITHDRAW-027). The original
// restores the Order block raw on LOAD, so these bytes are what its
// withdrawal test reads (AI-WITHDRAW-026).
const (
	OrderWithdrawOffset = 0x40
	OrderWimpyOffset    = 0x44
)

// WithdrawalThresholds reads Withdraw and Wimpy from the record's Order
// block, or from the second game's command block at the same offsets
// (R2-ENGINE-1005). ok is false when the record carries neither.
func (a ActorRecord) WithdrawalThresholds() (withdraw, wimpy int32, ok bool) {
	if a.HasOrder {
		return OrderWithdrawalThresholds(a.Order[:])
	}
	if a.HasCommand {
		return a.CommandWithdraw, a.CommandWimpy, true
	}
	return 0, 0, false
}

// OrderWithdrawalThresholds reads both thresholds from a raw Order block.
func OrderWithdrawalThresholds(order []byte) (withdraw, wimpy int32, ok bool) {
	if len(order) < OrderWimpyOffset+4 {
		return 0, 0, false
	}
	return int32(binary.LittleEndian.Uint32(order[OrderWithdrawOffset:])), int32(binary.LittleEndian.Uint32(order[OrderWimpyOffset:])), true
}

// PutOrderWithdrawalThresholds writes both thresholds into a raw Order block.
func PutOrderWithdrawalThresholds(order []byte, withdraw, wimpy int32) bool {
	if len(order) < OrderWimpyOffset+4 {
		return false
	}
	binary.LittleEndian.PutUint32(order[OrderWithdrawOffset:], uint32(withdraw))
	binary.LittleEndian.PutUint32(order[OrderWimpyOffset:], uint32(wimpy))
	return true
}
