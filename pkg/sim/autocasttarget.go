package sim

import "slices"

// autoCastTargetAdmitted is common to automatic target selection regardless
// of spell or support setting (TRIG-OFFMAP-041, AI-LIST-003).
func autoCastTargetAdmitted(target Entity) bool { return !target.OffMap }

// secondGameSpareMageTypes are the second game's Necromancer (99) and Druid
// (100) Units type ids (R2-SESSION-350, R2-ASSET-125). The first game's Units
// name neither.
var secondGameSpareMageTypes = []int32{99, 100}

// autoCastArmAdmits is the arm's own rule for an unbidden cast: an arm that
// spares mages never chooses a sparedMage, the caster included. Bless serves
// a hand-held weapon, and a mage attacks with a staff (owner, DIV-4830,
// DIV-4831). A player's order does not ask it.
func (w *World) autoCastArmAdmits(arm uint16, target Entity) bool {
	return !spellArmOf(arm).sparesMages || !w.sparedMage(target)
}

// sparedMage is the mage of the arms that spare mages: a human with a mana
// pool, or a creature of the profile's spareMageTypes. Any other creature
// with mana is a warrior here (owner, DIV-4830, DIV-4831).
func (w *World) sparedMage(e Entity) bool {
	return e.Humanoid && isMage(e) || slices.Contains(w.game.spareMageTypes, e.TypeID)
}
