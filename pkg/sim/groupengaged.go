package sim

import (
	"slices"
	"sort"
)

// engagedOrder reports whether a group stored under order goes out to fight,
// which is the population the engaged-group rule below applies to: Guard,
// Swarm, Swarm 2 and Roam. Stand Ground, Move and the order that hands a member
// to its own actor state keep their own arms' behaviour.
func engagedOrder(order uint8) bool {
	return order == orderGuard || order == orderSwarm || order == orderSwarm2 || order == orderRoam
}

// engagedFoes are the living hostile units a group is fighting, by ascending
// entity index: every unit striking at one of the members, that is holding it
// as its attack victim while mid-cycle or close enough to strike, and every
// unit at or next to a cell a member recorded as the source of a blow it took
// (AI-RETAL-056). A unit still walking up to its victim is not fighting yet,
// and a victim a member holds of its own does not count, so the group learns
// nothing from an order it cannot see and a chase the notice circle would end
// still ends once the foe stops attacking the group.
func (w *World) engagedFoes(owner uint32, members []int) []int {
	var foes []int
	add := func(fi int) {
		f := w.entities[fi]
		if f.OffMap || f.InLife(LifeDecaying) || !f.OrdinaryTargetable() || !w.relations.Hostile(owner, f.Owner) ||
			containsIndex(foes, fi) {
			return
		}
		foes = append(foes, fi)
	}
	for fi := range w.entities {
		f := w.entities[fi]
		if !f.HasAttackTarget || f.AttackTargetKind != AttackTargetUnit {
			continue
		}
		for _, mi := range members {
			if w.entities[mi].ID == f.AttackTarget {
				if f.AttackPhase != AttackReady || w.closedOn(fi, mi) {
					add(fi)
				}
				break
			}
		}
	}
	for _, mi := range members {
		n := w.attackNoticeAt(mi)
		if n.Cell == 0 {
			continue
		}
		source := cell{x: int32(n.Cell & 255), y: int32(n.Cell >> 8)}
		for fi := range w.entities {
			if cellOf(&w.entities[fi]).chebyshevTo(source) <= 1 {
				add(fi)
			}
		}
	}
	sort.Ints(foes)
	return foes
}

// withEngagedFoes adds the foes an engaged group is fighting to a decision's
// candidate list after the circle clip, so the group's own arm gives every
// member that can take one a victim. Bodies leave the list while a living foe
// is engaged, on the candidate sweep's own living-first rule. A group that is
// not engaged, or is not under an order that fights, gets its list back
// unchanged. A foe counts only inside the clipped list, in both games
// (DIV-1523).
func (w *World) withEngagedFoes(owner uint32, members []int, order uint8, cands []int) []int {
	if owner == 0 || owner == SelfSlot || !engagedOrder(order) {
		return cands
	}
	foes := inCircle(w.engagedFoes(owner, members), cands)
	if len(foes) == 0 {
		return cands
	}
	merged := make([]int, 0, len(cands)+len(foes))
	for _, ci := range cands {
		if w.entities[ci].InLife(LifeStanding) {
			merged = append(merged, ci)
		}
	}
	for _, fi := range foes {
		if !containsIndex(merged, fi) {
			merged = append(merged, fi)
		}
	}
	sort.Ints(merged)
	return merged
}

// joinsFight reports whether member mi is free to take up its group's fight:
// alive, on the map, not cursed to stone or using a structure, holding no
// victim, and holding no order but the walk back to its own post. A creature whose selector draw
// already ran in this pass is not asked again: each group handler reaches a
// member once.
func (w *World) joinsFight(mi int) bool {
	e := w.entities[mi]
	if !e.Alive() || e.OffMap || w.stoneCursed(mi) || e.HasAttackTarget || slices.Contains(w.engageDrew, e.ID) {
		return false
	}
	if _, using := w.structureUseIndex(e.ID); using {
		return false
	}
	return !e.HasTarget || (e.TargetX == e.PostX && e.TargetY == e.PostY)
}

// engagedCircle is the candidate list a group's decision clips, read without
// ageing any attack notice: the group's sight, the members' remembered
// attacker cells included, and for Guard the notice circle around the centroid
// (AI-GRPGUARD-074, AI-RADFREEZE-075). Swarm, Swarm 2 and Roam clip nothing
// past the sight.
func (w *World) engagedCircle(owner uint32, order uint8, circle noticeCircle, members []int) []int {
	return w.clipCircle(order, circle, w.groupCandidates(aiGroup{owner: owner, members: members}, false))
}

// noticeCircle is the centre cell and radius a Guard clip keeps candidates
// inside.
type noticeCircle struct {
	x, y int32
	r    int64
}

// storedCircle is a registry Group's circle as live Guard clips it: the
// centre cell +0x28 its geometry refresh stored and the working radius +0x2d
// (AI-739, AI-741; R2-ENGINE-841, R2-ENGINE-842).
func (w *World) storedCircle(g *SavedGroup) noticeCircle {
	return noticeCircle{x: int32(g.AI[0x28]), y: int32(g.AI[0x29]), r: int64(g.AI[0x2d])}
}

// clipCircle is a group decision's clip of its candidate list: Guard keeps
// the candidates inside the notice circle; every other order keeps the list.
func (w *World) clipCircle(order uint8, circle noticeCircle, cands []int) []int {
	if order != orderGuard {
		return cands
	}
	return clipToNotice(cands, w.entities, circle.x, circle.y, circle.r)
}

// inCircle keeps the foes the clipped candidate list holds.
func inCircle(foes, circle []int) []int {
	out := foes[:0:0]
	for _, fi := range foes {
		if containsIndex(circle, fi) {
			out = append(out, fi)
		}
	}
	return out
}

// joinSavedEngagedGroups is the same tail over the Group registry: each Group
// the dispatch reaches (AI +0x45 nonzero) under an order that fights, with its
// owner read from its first living member as the decision reads it. Roam
// fights as Swarm 2. A member that takes a foe holds pending order 5, as a
// decision's engage leaves it.
func (w *World) joinSavedEngagedGroups() {
	for gi := range w.savedGroups.Groups {
		g := &w.savedGroups.Groups[gi]
		if g.AI[0x45] == 0 && !w.safeMode {
			continue
		}
		members, complete := w.savedLiving(g)
		if !complete || len(members) == 0 {
			continue
		}
		order := w.savedRoute(g)
		if order == 0x11 {
			order = orderSwarm2
		}
		inWindow := func(i int) bool {
			o := w.savedOrder(w.entities[i].ID)
			return o != nil && w.game.groupRoute.runsStampWindow(g.AI[0x20]) && w.strikeWindow(i, o)
		}
		w.joinEngagedGroup(w.entities[members[0]].Owner, order, w.storedCircle(g), members, inWindow, func(i int) {
			if o := w.savedOrder(w.entities[i].ID); o != nil {
				o.Raw[8] = 5
			}
		})
	}
}

func (w *World) joinEngagedGroup(owner uint32, order uint8, circle noticeCircle, members []int, held func(int) bool, joined func(int)) {
	if owner == 0 || owner == SelfSlot || !engagedOrder(order) || len(members) == 0 {
		return
	}
	foes := w.engagedFoes(owner, members)
	if len(foes) != 0 {
		foes = inCircle(foes, w.engagedCircle(owner, order, circle, members))
	}
	if len(foes) == 0 {
		return
	}
	for _, mi := range members {
		if !w.joinsFight(mi) || held != nil && held(mi) {
			continue
		}
		at := w.chooseTarget(mi, foes, orderSwarm, targetGroupScore)
		if at >= 0 {
			w.orderAttack(mi, w.entities[at].ID)
			if joined != nil {
				joined(mi)
			}
		}
	}
}
