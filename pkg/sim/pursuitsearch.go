package sim

// PursuitSearch is a pursuer's route-search state between passes: the victim
// its last full search or rebuild was for (`mover+0x7c`), the static count
// that search left (`mover+0x8a`), the passes counted since (`mover+0x09`),
// the route end (`mover+0x76`) and the victim's cell at the last full search
// (`mover+0x8c`). The static list itself is the world's stored route
// (AI-413, AI-415, AI-417).
//
// Held false is the one inactive shape and it is all zero, which is also what
// a fresh mover holds: its first centred pass out of reach is a full search
// (AI-414).
type PursuitSearch struct {
	Held       bool
	Victim     EntityID
	Count      uint16
	Passes     uint8
	EndX, EndY int32
	AimX, AimY int32
}

// pursue is one pursuit pass of attacker i on its unit victim ti, run for a
// mover on its cell centre and outside its stop distance (AI-414). A changed
// victim starts afresh (AI-413). Passes above a third of the static count plus
// one re-search: in full toward the victim's cell while the last count is above
// five, else the route end alone, and an empty list cancels (AI-415, AI-373).
// A full search that misses the victim's cell stamps (stampMissedApproach).
// Standing on the route end with the victim where the last full search found
// it forgets the victim (AI-415). The near search aims at the route end, the
// head or the configured lookup node. An empty search aimed at the end cancels
// (AI-373, MOVE-100). A listed step survives a turn until its age expires (AI-885).
func (w *World) pursue(s *routeScratch, i, ti int, heldFirstCall bool) {
	e := &w.entities[i]
	t := w.entities[ti]
	victim := cell{x: t.X, y: t.Y}
	p := &e.Pursuit
	e.Walk.Age++
	if !p.Held || p.Victim != t.ID {
		*p = PursuitSearch{Count: 0xff, Passes: 0xff, EndX: victim.x, EndY: victim.y, AimX: victim.x, AimY: victim.y}
		e.Walk = WalkRefresh{Age: e.Walk.Age}
		w.routes[i] = nil
	}
	if int(p.Passes) > len(w.routes[i])/3+1 {
		if int32(p.Count) > w.pathFinding().StaticIsntNeeded {
			route, ok := w.searchRoute(s, i, terrainRelation, noWindow, w.farBudgetFor(i), settleOrdered, victim.x, victim.y)
			if s.missed {
				w.stampMissedApproach(i, s.sub, victim)
			}
			if !ok {
				route = nil
			}
			w.routes[i] = route
			e.Walk = WalkRefresh{Age: e.Walk.Age}
			if len(route) > 0 {
				end := route[len(route)-1]
				p.EndX, p.EndY = end.x, end.y
			}
			p.AimX, p.AimY = victim.x, victim.y
		} else {
			w.routes[i] = []cell{{x: p.EndX, y: p.EndY}}
		}
		p.Count = uint16(len(w.routes[i]))
		if len(w.routes[i]) == 0 {
			w.cancelPursuit(s, i)
			return
		}
		p.Held, p.Victim, p.Passes = true, t.ID, 0
	}
	w.walkTo(s, i, p.EndX, p.EndY)
	if e.X == p.EndX && e.Y == p.EndY && p.AimX == victim.x && p.AimY == victim.y {
		// The pass after this one sees a changed victim, which resets every
		// field here, so the counted pass is not kept.
		*p = PursuitSearch{}
		return
	}

	route := w.routes[i]
	goal := cell{x: p.EndX, y: p.EndY}
	goalAt, aimsAtEnd := routeEndGoal(w.pathFinding(), route, cell{x: e.X, y: e.Y})
	if !aimsAtEnd {
		goal = route[goalAt]
	}
	next, listed := w.walkNext(i)
	if !listed {
		next, listed = w.nearStep(s, i, goal, settleVictim)
		p.Passes++
	}
	if !listed {
		// A human participant's order takes the stall count and the whole-map
		// check instead of the cancel, so a passing ally cannot end it, and
		// forgets a route end that served nothing, so its next pass searches
		// in full toward where the victim now stands (DIV-1316, DIV-2561).
		if e.Owner == SelfSlot {
			if aimsAtEnd {
				*p = PursuitSearch{}
			}
			e.Stall++
			if e.Stall >= stallLimit {
				refused := w.pursuitRefused(s, i)
				if w.restAt(s, i) && refused {
					w.answerRefusedPursuit(i, searchNear)
				}
			}
			return
		}
		if aimsAtEnd {
			w.cancelPursuit(s, i)
		}
		return
	}
	if w.advanceStep(s, i, next, heldFirstCall) == stepTaken {
		e.Walk = WalkRefresh{Age: e.Walk.Age}
	}
}

// cancelPursuit is the raised route-failure flag of pursuer i: the walk ends
// and the order machine answers the refusal (AI-ROUTE-045, DIV-1520). The
// pursuit forgets its victim, so an order at the same victim later starts
// with a full search (DIV-2557).
func (w *World) cancelPursuit(s *routeScratch, i int) {
	w.entities[i].Pursuit = PursuitSearch{}
	if w.restAt(s, i) {
		w.answerRefusedPursuit(i, searchNear)
	}
}

// routeEndGoal is where a near search aims along a stored route from the cell
// from: the route end while at most p.StaticIsntNeeded cells remain
// (aimsAtEnd, index of the last cell), else the head when from lies more than
// p.DynamicByStaticLookup off it, else the cell at that index. It is the one
// builder of this rule for the pursuit and for a walk that applies it. The
// index never leaves the route, whatever the file says: an empty route aims
// at its end (index -1, as the shipped value does) and the look-ahead stops
// at the last cell.
func routeEndGoal(p PathFinding, route []cell, from cell) (index int, aimsAtEnd bool) {
	switch {
	case len(route) == 0 || int64(len(route)) <= int64(p.StaticIsntNeeded):
		return len(route) - 1, true
	case from.chebyshevTo(route[0]) > int64(p.DynamicByStaticLookup):
		return 0, false
	}
	return min(max(0, int(p.DynamicByStaticLookup)), len(route)-1), false
}
