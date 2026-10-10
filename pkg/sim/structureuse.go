package sim

import (
	"encoding/binary"
	"slices"
	"sort"
)

// StructureUse is one explicit command, independent of overlapping unit IDs.
type StructureUse struct {
	Entity    EntityID
	Structure StructureID
}

func (w *World) StructureUses() []StructureUse { return slices.Clone(w.structureUses) }

func (s Structure) Usable() bool {
	return s.Width > 0 && s.Height > 0 && (s.Kind == 28 || s.Kind == 29 || (s.Kind == 15 || s.Kind == 16) && s.UseAmount > 0)
}

func (w *World) pruneStructureUses() {
	w.structureUses = slices.DeleteFunc(w.structureUses, func(u StructureUse) bool {
		i, si := indexOfEntity(w.entities, u.Entity), indexOfStructure(w.structures, u.Structure)
		return i < 0 || si < 0 || !w.entities[i].Alive() || w.entities[i].OffMap || !w.structures[si].Usable()
	})
}

func (w *World) structureUseIndex(id EntityID) (int, bool) {
	return slices.BinarySearchFunc(w.structureUses, id, func(u StructureUse, id EntityID) int {
		if u.Entity < id {
			return -1
		}
		if u.Entity > id {
			return 1
		}
		return 0
	})
}

func (w *World) cancelStructureUse(id EntityID) {
	if at, ok := w.structureUseIndex(id); ok {
		w.structureUses = slices.Delete(w.structureUses, at, at+1)
	}
}

func (w *World) validStructureUse(i int, id StructureID) bool {
	si := indexOfStructure(w.structures, id)
	return i >= 0 && i < len(w.entities) && si >= 0 && w.structures[si].Usable() && w.entities[i].Alive() && !w.entities[i].OffMap && !w.stoneCursed(i)
}

// usingStructure reports whether the actor has a structure use pending.
func (w *World) usingStructure(id EntityID) bool {
	_, ok := w.structureUseIndex(id)
	return ok
}

// beginStructureUse writes the state, the building and the destination the
// actor approaches and no attack progress, so a loaded attack cycle finishes
// before the approach begins (AI-STRUCTUSE-306, AI-ORDER-039). The pending use
// takes the place of a held destination: the movement pass leaves the actor
// standing until the cycle returns to ready and the attack pass then drops the
// retained order (step.go; DIV-1577).
func (w *World) beginStructureUse(i int, id StructureID) bool {
	if !w.validStructureUse(i, id) || !w.cancelScroll(i) {
		return false
	}
	w.clearActorCast(i)
	w.invalidateActorMotion(w.entities[i].ID, "structure use supersedes original movement")
	w.clearOrder(i)
	e := &w.entities[i]
	e.clearAttackBetweenCycles()
	e.clearTurn()
	e.clearGroupSpeed()
	w.commandGroup([]int{i}, orderMove, cellOf(e))
	at, found := w.structureUseIndex(e.ID)
	use := StructureUse{Entity: e.ID, Structure: id}
	if found {
		w.structureUses[at] = use
	} else {
		w.structureUses = slices.Insert(w.structureUses, at, use)
	}
	w.syncSavedActorCommand(i)
	return true
}

func (w *World) finishStructureUse(i int) {
	e := w.entities[i]
	w.cancelStructureUse(e.ID)
	if w.savedGroups == nil {
		return
	}
	// The click origin was only a provisional movement destination. Retire
	// it at the actual stopping cell before saved Group dispatch resumes.
	dst := uint16(e.X) | uint16(e.Y)<<8
	o := w.ensureSavedOrder(i)
	binary.LittleEndian.PutUint16(o.Raw[10:], dst)
	binary.LittleEndian.PutUint32(o.Raw[0x50:], 0)
	if g := w.savedGroupFor(e.ID); g != nil {
		binary.LittleEndian.PutUint16(g.AI[10:], dst)
	}
}

func structureUseGoal(s Structure) (cell, int64) {
	goal, radius := cell{s.Col, s.Row}, int64(1)
	if s.Width == 3 || s.Width == 4 {
		goal.x++
		goal.y++
		radius = 2
	} else if s.Width != 1 && s.Width != 2 {
		goal.x += 2
		goal.y += 2
		radius = 3
	}
	return goal, radius
}

// A bounded route is selected once and reused. Failure retires the command;
// repeated unreachable clicks cannot leave a per-tick search loop behind.
func (w *World) advanceStructureUse(scratch *routeScratch, i int) bool {
	at, ok := w.structureUseIndex(w.entities[i].ID)
	if !ok {
		return false
	}
	si := indexOfStructure(w.structures, w.structureUses[at].Structure)
	if si < 0 || !w.structures[si].Usable() {
		w.finishStructureUse(i)
		return false
	}
	s := w.structures[si]
	goal, radius := structureUseGoal(s)
	a := w.entities[i]
	if cellOf(&a).chebyshevTo(goal) <= radius {
		if !w.restAt(scratch, i) {
			return true
		}
		if w.turnToward(i, goal.x-a.X, goal.y-a.Y) {
			return true
		}
		w.finishStructureUse(i)
		w.applyStructureUse(i, si)
		return true
	}
	if a.HasTarget && (cell{a.TargetX, a.TargetY}).chebyshevTo(goal) <= radius && w.open(scratch, terrainRelation, i, a.TargetX, a.TargetY) {
		return false
	}
	var candidates []cell
	for y := max(int64(0), int64(goal.y)-radius); y <= min(int64(w.bounds.Height)-1, int64(goal.y)+radius); y++ {
		for x := max(int64(0), int64(goal.x)-radius); x <= min(int64(w.bounds.Width)-1, int64(goal.x)+radius); x++ {
			if w.open(scratch, terrainRelation, i, int32(x), int32(y)) && w.restFree(scratch, i, int32(x), int32(y)) {
				candidates = append(candidates, cell{int32(x), int32(y)})
			}
		}
	}
	sort.SliceStable(candidates, func(i, j int) bool {
		return candidates[i].chebyshevTo(cellOf(&a)) < candidates[j].chebyshevTo(cellOf(&a))
	})
	for _, next := range candidates {
		if route, found := w.searchRoute(scratch, i, terrainRelation, noWindow, w.farBudgetFor(i), exactGoal, next.x, next.y); found && len(route) > 0 {
			w.walkTo(scratch, i, next.x, next.y)
			w.routes[i] = route
			return false
		}
	}
	w.finishStructureUse(i)
	w.restAt(scratch, i)
	return true
}

func (w *World) applyStructureUse(i, si int) {
	s := w.structures[si]
	if s.Kind == 28 || s.Kind == 29 {
		w.structures[si].Field42 = 0
		if s.Field42 == 0 {
			w.structures[si].Field42 = 1
		}
		return
	}
	if int16(s.Field42) <= 0 || !w.sourceMutationReady(i) {
		return
	}
	// The installed named potion supplies the amount. Use the same source
	// profile mutation and clamped pools as the carried potion path.
	n := w.sourceMutationCopy(i)
	n.structures = slices.Clone(w.structures)
	kind := uint8(6)
	if s.Kind == 16 {
		kind = 9
	}
	if n.entities[i].ActorLoad.Source.Class == 2 {
		if !n.sourcePotion(i, kind, s.UseAmount) {
			return
		}
	} else if kind == 6 {
		n.entities[i].setCurrentHealth(potionPool(n.entities[i].HP, n.entities[i].MaxHP, s.UseAmount))
	} else if isMage(n.entities[i]) {
		n.entities[i].setCurrentMana(potionPool(n.entities[i].Mana, n.entities[i].MaxMana, s.UseAmount))
	}
	n.structures[si].Field42--
	if !n.savedMutationValid() {
		return
	}
	*w = n
}

// UNIT-STRUCTZERO-080's local counter filter; callback placement in this
// scheduler is explicit integration policy, not a measured wall-clock period.
func (w *World) rechargeStructures() {
	if w.tick%60 != 0 {
		return
	}
	for i, s := range w.structures {
		if (s.Kind == 15 || s.Kind == 16) && int16(s.Field42) >= 0 && s.Field42 < s.MaxHealth {
			w.structures[i].Field42++
		}
	}
}

// RestoreStructureUseMetadata fills only fields older forms did not store.
// Existing current metadata remains part of the saved world's identity.
func (w *World) RestoreStructureUseMetadata(fresh []Structure) {
	for _, s := range fresh {
		i := indexOfStructure(w.structures, s.ID)
		if i >= 0 && w.structures[i].Kind == 0 {
			w.structures[i].Kind = s.Kind
			w.structures[i].UseAmount = s.UseAmount
		}
	}
}
