package sim

import "slices"

// autoCastTargetAdmitted is common to automatic target selection regardless
// of spell or support setting (TRIG-OFFMAP-041, AI-LIST-003).
func autoCastTargetAdmitted(target Entity) bool { return !target.OffMap }

// autoChoiceSight is the perception question one automatic choice asks of
// its candidates, built on first use (autoCastObservers).
type autoChoiceSight struct {
	set     *observerSet
	purpose perceptionPurpose
}

// autoChoiceSees reports whether the automatic choice of caster ci may take
// the actor at ti by sight.
func (w *World) autoChoiceSees(ci, ti int, s *autoChoiceSight) bool {
	if s.set == nil {
		set, purpose := w.autoCastObservers(ci)
		s.set, s.purpose = &set, purpose
	}
	return w.perceives(perception{purpose: s.purpose, set: s.set, target: ti})
}

// autoCastObservers is the observers and purpose of caster ci's automatic
// choice: the cells the caster lights, or under autoCastSideSight the cast
// sight of the side of its player and its allies (ownOrAllied, DIV-4872).
func (w *World) autoCastObservers(ci int) (observerSet, perceptionPurpose) {
	c := w.entities[ci]
	if !w.game.autoCastSideSight {
		return w.observersOf(c.Owner, func(j int, _ *Entity) bool { return j == ci }), perceiveLit
	}
	side := func(_ int, e *Entity) bool { return e.Owner == c.Owner || w.ownOrAllied(&c, e) }
	return w.observersOf(c.Owner, side), perceiveCast
}

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
