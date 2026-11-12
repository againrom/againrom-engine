package sim

// programmeSupport is the Group programme byte the second game's L order
// writes (R2-ENGINE-1198).
const programmeSupport uint8 = 0x13

// supportState reports a member order this World's game admits as L's.
func (w *World) supportState(s uint32) bool {
	return w.game.orders.supportOrder && s == uint32(actorStateSupport)
}

// commandSupport is L's setter over members commandStance has put in one new
// Group at programme 0x13 and posted at their own cells: Stand Ground's reset,
// then member order 0x17 with prepared order 0 (R2-ENGINE-1199, DIV-4893).
func (w *World) commandSupport(members []int) {
	w.commandStandGround(members)
	for _, i := range members {
		w.entities[i].ActorState = actorStateSupport
		w.ensureSavedOrder(i).State = uint32(actorStateSupport)
	}
}

// supportPass is programme 0x13's update of each member: a player's member is
// left at prepared order 0 and stands, a computer's at 0xb; nothing here walks,
// turns or attacks. A caster's support runs on the automatic-cast path
// (R2-ENGINE-1200, DIV-4893, DIV-4894).
func (w *World) supportPass(g *SavedGroup) {
	w.walkSavedMembers(g, func(i int) {
		e := w.entities[i]
		o := w.savedOrder(e.ID)
		if o == nil || !e.Alive() || e.OffMap {
			return
		}
		o.Raw[8] = 0
		if w.computerControlledUnit(e) {
			o.Raw[8] = 0xb
		}
	})
}

// leaveSupport ends L's order for a unit a single-unit order has just taken:
// the unit moves into its own command Group, whose builder leaves it at state
// 0xc (R2-ENGINE-1201, DIV-4895).
func (w *World) leaveSupport(i int) {
	if w.entities[i].ActorState == actorStateSupport {
		w.commandSavedGroup([]int{i}, orderNone, cell{})
	}
}
