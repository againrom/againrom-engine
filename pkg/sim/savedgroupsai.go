package sim

import (
	"encoding/binary"
	"fmt"
	"slices"
)

func savedActorStateSupported(state uint32) bool {
	return state == 0 || state == 1 || state == 3 || state == 0xa || state == 0xb || state == 0xc
}

func savedActorOrderSupported(o SavedActorOrder) bool {
	return savedActorStateSupported(o.State) || o.State == 8 || o.State == 0x11 || o.Authored && o.State == 0x16
}

func savedEscortBindingValid(o SavedActorOrder, entities []Entity, dead []originalDeadRecord) bool {
	key := binary.LittleEndian.Uint32(o.Raw[0x10:])
	i := indexOfEntity(entities, o.EscortTarget)
	if key == 0 || i < 0 || o.EscortTarget == o.Entity {
		return false
	}
	// Already-admitted late corpses have a separate source record. Their
	// current presence and exact identity matter, not MapUnitID or stage of
	// the target. The repairing actor's stage is checked by the caller.
	matches := map[EntityID]bool{}
	for _, e := range entities {
		s := e.SourceBinding
		if s.ActorClass() >= 1 && s.ActorClass() <= 3 && s.Identity == key {
			matches[e.ID] = true
		}
	}
	for _, d := range dead {
		if d.Source.Identity == key && d.Source.Class >= 1 && d.Source.Class <= 3 && indexOfEntity(entities, d.ID) >= 0 {
			matches[d.ID] = true
		}
	}
	return len(matches) == 1 && matches[o.EscortTarget]
}

// This is an operation admission check, not a whole-file LOAD refusal. In
// particular absent order payloads on dead members do not invent live state.
func (w *World) savedPrimaryIssue(g *SavedGroup) string {
	if g.AI[0x45] == 0 && !w.safeMode {
		return ""
	}
	members, complete := w.savedLiving(g)
	if !complete {
		return "primary dispatch has unmaterialized members"
	}
	order := g.AI[0x20]
	if order == orderSwarm && (int32(g.AI[10]) >= w.bounds.Width || int32(g.AI[11]) >= w.bounds.Height) {
		return "Swarm destination is outside map"
	}
	needsOrder := order == 0 || order == 1 || order == 4 || order == 5 || order == 0x11
	for _, i := range members {
		o := w.savedOrder(w.entities[i].ID)
		if needsOrder && o == nil {
			return fmt.Sprintf("actor %d has no order payload for order %d", w.entities[i].ID, order)
		}
		if o == nil {
			continue
		}
		if order == 1 || order == 0 && (o.State == 0xb || o.State == 0xa && binary.LittleEndian.Uint32(o.Raw[4:]) == 0) {
			if int32(o.Raw[0]) >= w.bounds.Width || int32(o.Raw[1]) >= w.bounds.Height {
				return fmt.Sprintf("actor %d guard post is outside map", o.Entity)
			}
		}
		if order == 0 && !savedActorOrderSupported(*o) {
			return fmt.Sprintf("actor %d state %d continuation unsupported", o.Entity, o.State)
		}
		if order == 0 && o.State == 3 && !o.Authored {
			if o.RepairStage != 0 {
				return fmt.Sprintf("actor %d engagement repair skipped by source stage %d", o.Entity, o.RepairStage)
			}
			if _, ok := w.savedEngagementTarget(*o); !ok {
				return fmt.Sprintf("actor %d engagement source-key has no distinct materialized target", o.Entity)
			}
		}
		if order == 0 && (o.State == 8 || o.State == 0x11) {
			e := w.entities[i]
			if !o.Authored && o.RepairStage != 0 {
				return fmt.Sprintf("actor %d escort repair skipped by source stage %d", o.Entity, o.RepairStage)
			}
			if !o.Authored && (!o.EscortBound || !savedEscortBindingValid(*o, w.entities, w.originalDead)) {
				return fmt.Sprintf("actor %d escort source-key has no validated materialized target", o.Entity)
			}
			if o.Authored && (!e.HasEscortTarget || indexOfEntity(w.entities, e.EscortTarget) < 0 || e.EscortTarget == e.ID) {
				return fmt.Sprintf("actor %d has no validated native escort target", o.Entity)
			}
		}
		if order == 0 && o.State == 0xa && !slices.Contains(o.Patrol, binary.LittleEndian.Uint16(o.Raw[2:])) {
			return fmt.Sprintf("actor %d patrol cursor is absent from ring", o.Entity)
		}
		if order == 4 || order == 5 || order == 0x11 || order == 0 && o.State == 1 {
			if int32(o.Raw[10]) >= w.bounds.Width || int32(o.Raw[11]) >= w.bounds.Height {
				return fmt.Sprintf("actor %d destination is outside map", o.Entity)
			}
		}
		if order == 0 && o.State == 0xa {
			for _, cell := range o.Patrol {
				if int32(cell&255) >= w.bounds.Width || int32(cell>>8) >= w.bounds.Height {
					return fmt.Sprintf("actor %d patrol cell is outside map", o.Entity)
				}
			}
		}
	}
	if order == 0x11 && w.savedRoamReroll(g, members) && len(w.savedRoamCells(g)) == 0 {
		return "Roam has no admissible 20-cell step"
	}
	return ""
}

func (w *World) savedAcquire(i int, o *SavedActorOrder) {
	w.acquireStanding(i)
	if o != nil {
		if w.entities[i].HasAttackTarget {
			o.Raw[8] = 5
		} else {
			o.Raw[8] = 0xb
		}
	}
}

// Guard's spatial query is around the POST, not the current actor cell.
// The native entity traversal/engage/scoring policies remain the build's
// existing deterministic substitutes for original actor-list chronology.
func (w *World) savedGuard(i int, o *SavedActorOrder) {
	e := &w.entities[i]
	if binary.LittleEndian.Uint16(o.Raw[:]) == 0 {
		binary.LittleEndian.PutUint16(o.Raw[:], uint16(e.X)|uint16(e.Y)<<8)
	}
	e.PostX, e.PostY = int32(o.Raw[0]), int32(o.Raw[1])
	// The block scan is postEngage (guardarm.go, 1141): one body for the two
	// sides of this routine, the original-runtime one here and the per-actor
	// one the actor layer reaches. Its filter, its ordering and its
	// engage-then-continue shape are this scan's own, lifted whole.
	if w.postEngage(i, cell{x: e.PostX, y: e.PostY}) {
		o.Raw[8] = 5
		return
	}
	if e.X != e.PostX || e.Y != e.PostY {
		binary.LittleEndian.PutUint16(o.Raw[10:], binary.LittleEndian.Uint16(o.Raw[:]))
		w.savedMove(i, o)
		return
	}
	w.savedAcquire(i, o)
}

// AI-MOVE-023's group arm has its own arrival/acquire latch. The actor-state
// move and Patrol's issue-move do not run this group-level latch protocol.
func (w *World) savedGroupMove(i int, o *SavedActorOrder) {
	e := w.entities[i]
	if e.X == int32(o.Raw[10]) && e.Y == int32(o.Raw[11]) && e.Transit == 0 && binary.LittleEndian.Uint32(o.Raw[0x50:]) == 0 {
		o.Raw[0x14] = uint8(e.Reach)
		binary.LittleEndian.PutUint32(o.Raw[0x50:], 1)
		w.clearOrder(i)
	}
	if binary.LittleEndian.Uint32(o.Raw[0x50:]) != 0 {
		w.savedAcquire(i, o)
		return
	}
	w.savedMove(i, o)
}

func (w *World) savedRoamReroll(g *SavedGroup, members []int) bool {
	var far int64
	dst := cell{x: int32(g.AI[10]), y: int32(g.AI[11])}
	for _, i := range members {
		if d := cellOf(w.entities[i]).chebyshevTo(dst); d > far {
			far = d
		}
	}
	return far < 10 || g.AI[0x15] > 50
}

var savedRoamDirections = [8]cell{{0, -1}, {1, -1}, {1, 0}, {1, 1}, {0, 1}, {-1, 1}, {-1, 0}, {-1, -1}}

func (w *World) savedRoamCells(g *SavedGroup) []cell {
	var out []cell
	for _, d := range savedRoamDirections {
		c := cell{x: int32(g.AI[10]) + 20*d.x, y: int32(g.AI[11]) + 20*d.y}
		if c.x >= 8 && c.y >= 8 && c.x <= w.bounds.Width-9 && c.y <= w.bounds.Height-9 {
			out = append(out, c)
		}
	}
	return out
}

func (w *World) savedRoam(g *SavedGroup, members []int) {
	dst, counter, ok := w.rollRoamCell(cell{int32(g.AI[10]), int32(g.AI[11])}, members, g.AI[0x15])
	if ok {
		g.AI[10], g.AI[11], g.AI[0x15] = uint8(dst.x), uint8(dst.y), counter
	}
	// Swarm2 evaluation follows; its Move fallback reads actor destinations.
}

func (w *World) savedDecision(g *SavedGroup, order uint8, obs *castObs) {
	members, complete := w.savedLiving(g)
	if !complete || len(members) == 0 {
		return
	}
	ag := aiGroup{group: g.ID, owner: w.entities[members[0]].Owner, members: members}
	cands := w.candidates(ag)
	if order == orderGuard {
		cx, cy := groupCentroid(w.entities, members)
		cands = clipToNotice(cands, w.entities, cx, cy, int64(noticeRadius(g.AI[0x38])))
	}
	cands = w.withEngagedFoes(ag.owner, members, order, cands)
	id := g.ID
	roam := g.AI[0x20] == 0x11
	w.walkSavedMembers(g, func(i int) {
		if _, using := w.structureUseIndex(w.entities[i].ID); using {
			return
		}
		if !w.entities[i].Alive() || w.entities[i].OffMap || w.stoneCursed(i) {
			return
		}
		o := w.savedOrder(w.entities[i].ID)
		// The stance decision leaves a victim a player's attack order named
		// alone, and takes a unit whose ordered victim is gone back at guard.
		ordered := &w.entities[i]
		if holdsOrderedVictim(*ordered) {
			return
		}
		if ordered.Owner == SelfSlot && ordered.ActorState == actorStateEngage && !ordered.HasAttackTarget {
			ordered.ActorState = actorStateGuard
			if o != nil {
				o.State = uint32(actorStateGuard)
			}
		}
		if order == orderMove || order == orderSwarm2 && len(cands) == 0 {
			w.savedGroupMove(i, o)
			return
		}
		best, at := scoreSeed, -1
		if uint8(len(cands)) != 0 && uint8(len(members)) != 0 {
			for _, ci := range cands {
				if ci == i {
					continue
				}
				if cost := w.candidateCost(i, ci, order); cost < best {
					best, at = cost, ci
				}
			}
		}
		if at >= 0 {
			w.orderAttack(i, w.entities[at].ID)
			if o != nil {
				o.Raw[8] = 5
			}
		} else if order == orderSwarm {
			current := w.savedGroupByID(id)
			e := &w.entities[i]
			x, y := int32(current.AI[10]), int32(current.AI[11])
			if !arrived(*e) || e.X != x || e.Y != y {
				w.cancelTurnForTargetChange(i, x, y)
				w.clearOrder(i)
				if escortState(e.ActorState) {
					w.clearEscort(i)
				}
				e.clearAttackBetweenCycles()
				e.TargetX, e.TargetY, e.HasTarget = x, y, true
				w.syncSavedDestination(i)
			}
		} else if order == orderStandGround && w.entities[i].Owner == SelfSlot && !underCommand(w.entities[i]) {
			// A participant's member idles with the pending order 0, not the idle
			// turn (AI-349, AI-353). A member walking to a cell it was sent to
			// keeps its walk, as in a native group's decision (aiGroups).
			w.standDown(i)
			if o != nil {
				o.Raw[8] = 0
			}
		} else {
			// Release is keyed to the member's actual owner, not a containing
			// Player or the possibly null/discordant saved Group owner lookup.
			if w.entities[i].Owner != SelfSlot {
				w.releaseAttack(i)
			}
			if o != nil {
				o.Raw[8] = 0xb
			}
			if order == orderGuard && o != nil {
				e := &w.entities[i]
				e.PostX, e.PostY = int32(o.Raw[0]), int32(o.Raw[1])
				w.walkHome([]int{i})
				if e.HasTarget {
					w.syncSavedDestination(i)
				}
			}
		}
		w.aiCast(i, obs)
	})
	if roam {
		w.savedGroupByID(id).AI[0x15]++
	}
}
