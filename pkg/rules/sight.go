package rules

// SightDivisor and SightBase are HERO-SIGHT-007's divisor and added term.
const (
	SightDivisor int32 = 25
	SightBase    int32 = 4
)

// SightSubCells is HERO-SIGHT-007's sight word, `((mind + reaction)/25 + 4)`
// in 1/256 cell, whose high byte is the whole-cell radius.
func SightSubCells(mind, reaction int32) int32 {
	return (mind+reaction)*256/SightDivisor + SightBase*256
}

// SightWordIn is a Human's sight word: SightSubCells with its whole-cell byte
// replaced by cells. ok is false outside 1..0xffff.
func SightWordIn(mind, reaction int32, cells int32) (word uint16, ok bool) {
	raw := SightSubCells(mind, reaction)
	raw += cells<<8 - raw&^0xff
	if raw <= 0 || raw > 0xffff {
		return 0, false
	}
	return uint16(raw), true
}

// SightWord is SightWordIn's word, or cells alone, clamped, when out of range.
func SightWord(mind, reaction int32, cells int32) uint16 {
	if word, ok := SightWordIn(mind, reaction, cells); ok {
		return word
	}
	return uint16(max(0, min(cells, 255)) << 8)
}
