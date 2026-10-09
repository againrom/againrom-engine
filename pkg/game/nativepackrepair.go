package game

import "againrom/pkg/sim"

func repairCurrentNativePacks(ms *Mission, a *currentActionData, actorIDs map[sim.EntityID]sim.EntityID) {
	var ids, blocked []sim.EntityID
	for _, p := range a.Party {
		ids = append(ids, p.Entity)
	}
	if a.Pending != nil {
		for _, p := range a.Pending.Commands {
			if p.Ignored {
				continue
			}
			switch p.Command.Kind {
			case sim.KindEquip, sim.KindReadBook, sim.KindUsePotion, sim.KindDropCarried, sim.KindUseScroll, sim.KindUseScrollAt:
				if id, ok := actorIDs[p.Command.Entity]; ok {
					blocked = append(blocked, id)
				}
			}
		}
	}
	ms.World.RepairNativePackCells(ids, blocked)
}
