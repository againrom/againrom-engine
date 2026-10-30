package game

import (
	"againrom/pkg/render/terrain"
	"againrom/pkg/sim"
)

// casterClass is the caster's class at the observation: equipment-selected
// for a hero (MAGIC-262, DIV-2657). nil when absent.
func (mw *mapWorld) casterClass(id sim.EntityID) *terrain.UnitClass {
	if mw.units == nil {
		return nil
	}
	e, ok := mw.entity(id)
	if !ok {
		return nil
	}
	return mw.units.Classes[mw.spellClientClass(id, e.Class)]
}
