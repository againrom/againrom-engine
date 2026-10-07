package game

import "againrom/pkg/sim"

func (mw *mapWorld) defend(entity, subject uint32) {
	mw.queueGroup(func(tag uint32) sim.Command {
		return sim.GroupDefend(sim.EntityID(entity), sim.EntityID(subject), tag)
	})
}
