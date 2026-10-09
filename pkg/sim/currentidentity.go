package sim

import (
	"cmp"
	"fmt"
	"slices"
)

// ActorIdentityReferences includes live, retired and absent endpoints. Source
// archive keys, structure IDs and optional fields without presence are excluded.
func (w *World) ActorIdentityReferences() []EntityID {
	var ids []EntityID
	w.actorIdentityFields(func(id *EntityID) { ids = append(ids, *id) })
	slices.Sort(ids)
	return slices.Compact(ids)
}

// RestoreActorIdentities moves every typed reference together before LOAD is
// adopted. Numeric actor order is execution order, including future actions.
func (w *World) RestoreActorIdentities(ids map[EntityID]EntityID, floor *uint64) error {
	refs := w.ActorIdentityReferences()
	used := map[EntityID]bool{}
	for _, old := range refs {
		id, ok := ids[old]
		if !ok || used[id] {
			return fmt.Errorf("sim: incomplete or aliased current actor identities")
		}
		used[id] = true
	}
	if floor != nil && *floor > entityIDLimit {
		return fmt.Errorf("sim: current actor identity floor exceeds namespace")
	}
	raw, err := w.MarshalBinary()
	if err != nil {
		return err
	}
	n := *w
	if err := n.UnmarshalBinary(raw); err != nil {
		return err
	}
	n.actorTraversal = append([]EntityID{}, n.actorTraversalIDs()...)
	n.actorIdentityFields(func(id *EntityID) { *id = ids[*id] })
	n.sortRemovedNativeBases()
	order := make([]int, len(n.entities))
	for i := range order {
		order[i] = i
	}
	slices.SortFunc(order, func(a, b int) int { return cmp.Compare(n.entities[a].ID, n.entities[b].ID) })
	entities, routes := slices.Clone(n.entities), slices.Clone(n.routes)
	carried, equipment := slices.Clone(n.carried), slices.Clone(n.equipment)
	for i, from := range order {
		n.entities[i], n.routes[i] = entities[from], routes[from]
		n.carried[i], n.equipment[i] = carried[from], equipment[from]
	}
	slices.SortFunc(n.bookCasts, func(a, b bookCast) int { return cmp.Compare(a.Caster, b.Caster) })
	slices.SortFunc(n.scrollCasts, func(a, b ScrollCast) int { return cmp.Compare(a.Caster, b.Caster) })
	slices.SortFunc(n.structureUses, func(a, b StructureUse) int { return cmp.Compare(a.Entity, b.Entity) })
	slices.SortFunc(n.attached, func(a, b attachedEffect) int {
		if c := cmp.Compare(a.Target, b.Target); c != 0 {
			return c
		}
		return cmp.Compare(a.Spell, b.Spell)
	})
	if n.savedMotion != nil {
		slices.SortFunc(n.savedMotion.Motions, func(a, b SavedActorMotion) int { return cmp.Compare(a.Entity, b.Entity) })
	}
	if floor != nil {
		n.entityIDFloor = *floor
	}
	var resolved []EntityID
	for _, id := range ids {
		resolved = append(resolved, id)
	}
	n.reserveAbsentEntityIDs(resolved)
	encoded, err := n.MarshalBinary()
	if err != nil {
		return err
	}
	if err := n.UnmarshalBinary(encoded); err != nil {
		return err
	}
	*w = n
	return nil
}

// Extant actors and retained dead bodies already bound NextEntityID. Only
// endpoints without such a body need an extra persistent reservation here.
func (w *World) reserveAbsentEntityIDs(ids []EntityID) {
	present := make(map[EntityID]bool, len(w.entities)+len(w.originalDead))
	for _, e := range w.entities {
		present[e.ID] = true
	}
	for _, d := range w.originalDead {
		present[d.ID] = true
	}
	for _, d := range w.currentTerminalActors {
		present[d.ID] = true
	}
	for _, id := range ids {
		if !present[id] {
			w.ReserveEntityIDs([]EntityID{id})
		}
	}
}

func (w *World) actorIdentityFields(ref func(*EntityID)) {
	optional := func(id *EntityID, present bool) {
		if present {
			ref(id)
		}
	}
	for i := range w.actorTraversal {
		ref(&w.actorTraversal[i])
	}
	for i := range w.entities {
		e := &w.entities[i]
		ref(&e.ID)
		optional(&e.KillCreditSource, e.HasKillCredit)
		optional(&e.AttackTarget, e.HasAttackTarget && e.AttackTargetKind == AttackTargetUnit)
		optional(&e.PendingAttackTarget, e.HasPendingAttackTarget && e.PendingAttackTargetKind == AttackTargetUnit)
		optional(&e.PendingOrder.Target, e.PendingOrder.Kind == PendingActorCast)
		optional(&e.EscortTarget, e.HasEscortTarget)
		optional(&e.HeldOrder.Target, e.HeldOrder.Kind == HeldOrderBody)
	}
	if w.script != nil {
		for i := range w.script.checks {
			c := &w.script.checks[i]
			optional(&c.Unit, c.HasUnit)
			optional(&c.Unit2, c.HasUnit2)
		}
		for i := range w.script.instants {
			in := &w.script.instants[i]
			optional(&in.Unit, in.HasUnit)
			optional(&in.Unit2, in.HasUnit2)
		}
	}
	for i := range w.bookCasts {
		c := &w.bookCasts[i]
		ref(&c.Caster)
		optional(&c.Target, !c.AtCell)
	}
	for i := range w.scrollCasts {
		c := &w.scrollCasts[i]
		ref(&c.Caster)
		optional(&c.Target, !c.AtCell)
	}
	for i := range w.casts {
		c := &w.casts[i]
		optional(&c.Target, c.AtUnit)
	}
	for i := range w.deliveries {
		d := &w.deliveries[i]
		optional(&d.Caster, d.HasCaster)
		optional(&d.Target, !d.AtCell)
	}
	for i := range w.effects {
		e := &w.effects[i]
		optional(&e.Caster, e.HasCaster)
	}
	for i := range w.attached {
		e := &w.attached[i]
		ref(&e.Target)
		optional(&e.Caster, e.HasCaster)
	}
	for i := range w.structureUses {
		ref(&w.structureUses[i].Entity)
	}
	for i := range w.originalDead {
		ref(&w.originalDead[i].ID)
	}
	for i := range w.currentTerminalActors {
		ref(&w.currentTerminalActors[i].ID)
	}
	for i := range w.removedNativeBases {
		ref(&w.removedNativeBases[i].ID)
	}
	if w.savedGroups != nil {
		for i := range w.savedGroups.Groups {
			for j := range w.savedGroups.Groups[i].Members {
				m := &w.savedGroups.Groups[i].Members[j]
				optional(&m.Entity, m.Bound)
			}
		}
		for i := range w.savedGroups.Orders {
			o := &w.savedGroups.Orders[i]
			ref(&o.Entity)
			optional(&o.EscortTarget, o.EscortBound)
		}
	}
	if w.savedMotion != nil {
		for i := range w.savedMotion.Motions {
			ref(&w.savedMotion.Motions[i].Entity)
		}
		for i := range w.savedMotion.Cells {
			c := &w.savedMotion.Cells[i]
			optional(&c.Ground.Entity, c.Ground.Bound)
			optional(&c.Air.Entity, c.Air.Bound)
		}
	}
	for i := range w.savedCellRecords {
		c := &w.savedCellRecords[i]
		optional(&c.Ground.Entity, c.Ground.Bound)
		optional(&c.Air.Entity, c.Air.Bound)
	}
	if w.savedObjects != nil {
		owner := func(o *SavedObjectOwner) {
			optional(&o.Entity, o.Kind == SavedOwnerActorPack || o.Kind == SavedOwnerActorWorn)
		}
		for i := range w.savedObjects.ItemRoots {
			owner(&w.savedObjects.ItemRoots[i].Owner)
		}
		slices.SortFunc(w.savedObjects.ItemRoots, savedRootCompare)
		for i := range w.savedObjects.Containers {
			owner(&w.savedObjects.Containers[i].Owner)
		}
		for i := range w.savedObjects.BookRoots {
			ref(&w.savedObjects.BookRoots[i].Entity)
		}
		slices.SortFunc(w.savedObjects.BookRoots, func(a, b SavedBookRoot) int {
			if a.Entity < b.Entity {
				return -1
			}
			if a.Entity > b.Entity {
				return 1
			}
			return 0
		})
	}
	for i := range w.savedDiaries {
		o := &w.savedDiaries[i].Owner
		optional(&o.Actor, !o.Player)
	}
	if w.savedWorldEffects != nil {
		for i := range w.savedWorldEffects.Projectiles {
			p := &w.savedWorldEffects.Projectiles[i]
			optional(&p.Target, p.HasTarget)
		}
	}
	if w.savedSpellGraph != nil {
		for i := range w.savedSpellGraph.Nodes {
			n := &w.savedSpellGraph.Nodes[i]
			optional(&n.Target, n.HasTarget)
			optional(&n.Caster, n.HasCaster)
		}
	}
}
