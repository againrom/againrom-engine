package sim

// The first-game ids tests name their fixtures by. Production code asks the
// spell-arm table instead.
const (
	fireBallSpell        = 2
	prismaticSpellID     = 14
	controlSpiritSpellID = 25
	teleportSpellID      = 26
)

// pointMagnitudes is each arm's MAGIC-EFFECT-015 magnitude, indexed by arm,
// read from the spell-arm table.
var pointMagnitudes = func() (out [lastBookSpellID + 1]func(row *spellRow, mag, power int32) int32) {
	for i := range out {
		arm := spellArmOf(uint16(i))
		switch {
		case arm.poisonScaled:
			out[i] = func(row *spellRow, mag, power int32) int32 { return row.poison(mag, power) }
		case arm.magnitude != nil:
			out[i] = func(_ *spellRow, mag, power int32) int32 { return arm.magnitude(mag, power) }
		}
	}
	return out
}()
