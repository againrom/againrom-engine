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
var pointMagnitudes = func() (out [lastBookSpellID + 1]func(rule SpellRule, mag, power int32) int32) {
	for i := range out {
		out[i] = spellArmOf(uint16(i)).magnitude
	}
	return out
}()
