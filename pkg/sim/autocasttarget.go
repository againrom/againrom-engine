package sim

// autoCastTargetAdmitted is common to automatic target selection regardless
// of spell or support setting (TRIG-OFFMAP-041, AI-LIST-003).
func autoCastTargetAdmitted(target Entity) bool { return !target.OffMap }

// autoCastArmAdmits is the arm's own rule for an unbidden cast: an arm that
// spares mages never chooses an actor with a mana pool, the caster included.
// Bless serves a hand-held weapon, and a mage attacks with a staff (owner,
// DIV-4830, DIV-4831). A player's order does not ask it.
func autoCastArmAdmits(arm uint16, target Entity) bool {
	return !spellArmOf(arm).sparesMages || !isMage(target)
}
