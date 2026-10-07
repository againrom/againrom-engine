package sim

import "fmt"

// NormalizeMissionSurvivors enacts PARTY-ENDCULL-026 before the live actor to
// party join is discarded. The original keeps every band actor without a
// health test. A fallen band actor Heal can still raise (health 0 through -9)
// therefore rises first through the Heal transition, which doubles back the
// defence its fall halved (DIV-1448), and is reset like a living survivor; a
// body at -10 or below is not (PARTY-CULL-004).
// Effect removal uses the ordinary inverse/derive producer. Pools are restored
// after removal, so temporary maxima cannot leak.
// Inventory, progression, equipment timing and regeneration residues remain.
// The completed World is not resumed after this boundary.
func (w *World) NormalizeMissionSurvivors(owner uint32) error {
	ids := w.BoundarySurvivors(owner)
	var fallen []EntityID
	for i := range w.entities {
		e := w.entities[i]
		if e.Owner == owner && InPersistBand(e.TypeID) && e.Restorable() {
			fallen = insertAscending(fallen, e.ID)
		}
	}
	if len(ids) == 0 && len(fallen) == 0 {
		return nil
	}
	// An unsupported inverse must not publish half a return. The existing
	// source transaction owns entities, attached effects, routes and registries.
	n := w.sourceMutationCopy(0)
	// Removal can enter the ordinary death/action cleanup before admission
	// rejects that actor. Those owners must also stay inside the candidate.
	n.bookCasts = append([]bookCast(nil), w.bookCasts...)
	n.casts = append([]scriptCast(nil), w.casts...)
	n.structureUses = append([]StructureUse(nil), w.structureUses...)
	for i := range n.carried {
		n.carried[i] = cloneStacks(w.carried[i])
	}
	for _, id := range fallen {
		i := indexOfEntity(n.entities, id)
		before := n.entities[i].HP
		n.entities[i].setCurrentHealth(n.entities[i].MaxHP)
		n.restoreAfterHealthGain(i, before)
		ids = insertAscending(ids, id)
	}
	keep := make(map[EntityID]bool, len(ids))
	for _, id := range ids {
		keep[id] = true
	}
	for i := 0; i < len(n.attached); {
		if !keep[n.attached[i].Target] {
			i++
			continue
		}
		if !n.removeAttachedAt(i) {
			return fmt.Errorf("mission return cannot remove attached effect")
		}
	}
	for _, id := range ids {
		i := indexOfEntity(n.entities, id)
		if !n.entities[i].Alive() {
			return fmt.Errorf("mission return effect removal felled actor %d", id)
		}
		if !n.cancelScroll(i) {
			return fmt.Errorf("mission return cannot cancel actor %d scroll", id)
		}
	}
	for _, id := range ids {
		i := indexOfEntity(n.entities, id)
		e := &n.entities[i]
		if !e.Alive() {
			return fmt.Errorf("mission return action cleanup felled actor %d", id)
		}
		e.setCurrentHealth(e.MaxHP)
		e.setCurrentMana(e.MaxMana)
		e.clearTurn()
		e.clearTransit()
		e.clearGroupSpeed()
		e.clearAttack()
		e.clearPatrol()
		e.clearEscort()
		e.clearTarget()
		e.CommandGroup = 0
		e.ActionClock = ActionClock{}
		e.SpellFX, e.SpellFXSpell = 0, 0
		e.MapUnitID = 0
		n.routes[i] = nil
		n.clearActorCast(i)
		n.cancelStructureUse(id)
	}
	*w = n
	return nil
}
