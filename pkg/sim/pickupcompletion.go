package sim

import (
	"fmt"
	"sort"
)

// CompleteSackPickup records a successful gameplay pickup for completion on the
// next Step. The caller must first finish TakeSack and any quest-item routing.
// TakeSack itself remains a pure inventory transfer, including for tools.
//
// AI-356 separates completion state from any still-running physical body.
func (w *World) CompleteSackPickup(id EntityID) bool {
	i := indexOfEntity(w.entities, id)
	if i < 0 || !w.entities[i].Alive() || w.entities[i].OffMap || w.entities[i].Owner >= relationSlots {
		return false
	}
	w.cancelScroll(i)
	w.commandGroup([]int{i}, orderNone, cell{})
	w.clearOrder(i)
	e := &w.entities[i]
	admitted := e.PendingOrder.RowAdmitted
	if e.SourceBinding.GroupIndex != 0 && e.SourceBinding.GroupSelector != 0 && w.savedGroups != nil {
		if g := w.savedGroupFor(id); g != nil && g.Authored {
			e.SourceBinding.GroupSelector = g.Selector
		}
	}
	e.releaseBetweenCycles(PendingPickupComplete)
	if admitted && e.PendingOrder.Kind == PendingPickupComplete {
		e.PendingOrder.RowAdmitted = true
	}
	if e.PendingOrder.Kind == PendingNone {
		e.Retreat = RetreatContinuation{}
	}
	e.clearGroupSpeed()
	if !w.actorCastBusy(i) {
		e.clearTurn()
	}
	e.ActorState = actorStatePickupComplete
	w.syncCurrentActorOrder(i)
	return true
}

func (w *World) CancelSackPickup(id EntityID) {
	if i := indexOfEntity(w.entities, id); i >= 0 && w.entities[i].PendingOrder.Kind == PendingPickup {
		w.entities[i].PendingOrder = PendingOrder{}
	}
}

// stepPickupCompletions is actor order progress, not the phase-6 AI decision.
// It runs after that decision: completing on phase 6 cannot acquire a target
// retroactively on the same pass. An off-map actor keeps its pending order,
// just as takeOffMap preserves movement, attack and escort orders.
func (w *World) stepPickupCompletions() {
	for i := range w.entities {
		e := &w.entities[i]
		if e.ActorState != actorStatePickupComplete || !e.Alive() {
			continue
		}
		if e.PendingOrder.Kind == PendingPickupComplete {
			continue
		}
		// Script orders can change an existing command group without going
		// through commandGroup. That replacement wins as player orders do.
		//
		// A SAVED-GROUP WORLD HOLDS THE SAME FACT ON A DIFFERENT RECORD
		// (1113): CompleteSackPickup's own commandGroup call reroutes to
		// commandSavedGroup the instant w.savedGroups is non-nil, and that
		// write lands on a SavedGroup, never on w.groups. groupState scans
		// only w.groups, so it answered "no record names this pair" for
		// every saved-Group world and this arm forced every completion to
		// Guard, never to Acquire. This reads the record commandSavedGroup
		// actually wrote: whichever SavedGroup presently binds the actor as
		// a member, by its own AI[0x20] order byte — the saved form's
		// groupAI.order (newSavedCommandGroup, cmdSavedGroupOrder).
		if e.Owner != 0 {
			if w.savedGroups != nil {
				g := w.savedGroupFor(e.ID)
				if g == nil || g.AI[0x20] != orderNone {
					e.ActorState = actorStateGuard
					continue
				}
			} else if order, _, ok := w.groupState(e.Owner, effectiveGroup(e)); !ok || order != orderNone {
				e.ActorState = actorStateGuard
				continue
			}
		}
		if e.OffMap {
			continue
		}
		e.ActorState = actorStateAcquire
	}
}

func pickupCompletionGroupsFault(ents []Entity, groups []groupAI, saved *savedGroupState) error {
	for _, e := range ents {
		if e.ActorState != actorStatePickupComplete || e.Owner == 0 {
			continue
		}
		// SAVED GROUPS HOLD THE OWN GROUP RECORD THIS ENTITY'S WRITE
		// LANDED ON (1113): decode-time saved Group state is already split
		// out of data before this runs (UnmarshalBinary's own splitSavedGroups),
		// so a continued-from-original save is checked against the SAME
		// record commandSavedGroup wrote, not against w.groups, which a
		// saved-Group world's own commandGroup reroute never populates.
		if saved != nil {
			g := savedGroupForIn(saved, e.ID)
			if e.Owner >= relationSlots || e.CommandGroup == 0 || g == nil || g.AI[0x20] != orderNone {
				return fmt.Errorf("sim: entity %d pickup completion requires group order zero", e.ID)
			}
			continue
		}
		id := effectiveGroup(&e)
		i := sort.Search(len(groups), func(i int) bool {
			return groups[i].owner > e.Owner || groups[i].owner == e.Owner && groups[i].group >= id
		})
		if e.Owner >= relationSlots || e.CommandGroup == 0 || i == len(groups) || groups[i].owner != e.Owner || groups[i].group != id || groups[i].order != orderNone {
			return fmt.Errorf("sim: entity %d pickup completion requires group order zero", e.ID)
		}
	}
	return nil
}
