package rules

// SightDivisor and SightBase are the sight law's two constants
// (HERO-SIGHT-007): the divisor the sum of Mind and Reaction is taken over,
// and the term added after it.
const (
	SightDivisor int32 = 25
	SightBase    int32 = 4
)

// SightSubCells is HERO-SIGHT-007's sight word, `((mind + reaction)/25 + 4)`
// in 1/256 cell, whose high byte is the whole-cell radius.
func SightSubCells(mind, reaction int32) int32 {
	return (mind+reaction)*256/SightDivisor + SightBase*256
}

// SightWord is a Human's sight word in 1/256 cell: the fractional part
// ((mind+reaction)/25 + 4) carries, and the whole-cell byte is replaced by
// cells, the value the derive reports after equipment and effects.
func SightWord(mind, reaction int32, cells int32) uint16 {
	raw := SightSubCells(mind, reaction)
	raw += cells<<8 - raw&^0xff
	if raw <= 0 || raw > 0xffff {
		return uint16(max(0, min(cells, 255)) << 8)
	}
	return uint16(raw)
}
