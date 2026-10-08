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

// staticIsntNeeded is the shipped StaticIsntNeeded value: a previous static
// count above it re-searches in full, at or below it rebuilds the route end
// as one node, and a near search aims at the route end while the static list
// holds at most this many nodes (MOVE-PARAM-006, AI-384).
const staticIsntNeeded = 5

// dynamicByStaticLookup is the shipped DynamicByStaticLookup value: a head
// node farther than it from the mover is the near search's goal, and one
// within it is removed after the search (MOVE-PARAM-006, AI-373).
const dynamicByStaticLookup = 3

// pursue is one pursuit pass of attacker i on its unit victim ti, run for a
// mover on its cell centre and outside its stop distance (AI-414). A changed
// victim starts afresh (AI-413). Passes above a third of the static count plus
// one re-search: in full toward the victim's cell while the last count is above
// five, else the route end alone, and an empty list cancels (AI-415, AI-373).
// Standing on the route end with the victim where the last full search found
// it forgets the victim (AI-415). The near search aims at the route end, the
// head or the node three after it and settles by picker B; empty and aimed at
// the route end, it cancels (AI-373, MOVE-100). A pass ending in a turn is not
// counted, since the original turns on the list it already holds (DIV-2556).
func (w *World) pursue(s *routeScratch, i, ti int, heldFirstCall bool) {
	e := &w.entities[i]
	t := w.entities[ti]
	victim := cell{x: t.X, y: t.Y}
	p := &e.Pursuit
	if !p.Held || p.Victim != t.ID {
		*p = PursuitSearch{Count: 0xff, Passes: 0xff, EndX: victim.x, EndY: victim.y, AimX: victim.x, AimY: victim.y}
		w.routes[i] = nil
	}
	if int(p.Passes) > len(w.routes[i])/3+1 {
		if p.Count > staticIsntNeeded {
			route, ok := w.searchRoute(s, i, terrainRelation, noWindow, w.farBudgetFor(i), settleOrdered, victim.x, victim.y)
			if !ok {
				route = nil
			}
			w.routes[i] = route
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
	goal, aimsAtEnd := cell{x: p.EndX, y: p.EndY}, true
	if len(route) > staticIsntNeeded {
		aimsAtEnd = false
		goal = route[3]
		if (cell{x: e.X, y: e.Y}).chebyshevTo(route[0]) > dynamicByStaticLookup {
			goal = route[0]
		}
	}
	near := unitRelation
	if e.Domain == DomainAir {
		near = terrainRelation
	}
	step, ok := w.searchRoute(s, i, near, dynamicWindow, scaledBudget, settleVictim, goal.x, goal.y)
	if !ok || len(step) == 0 {
		w.countPursuitPass(i)
		// A human participant's order takes the stall count and the whole-map
		// check instead of the cancel, so a passing ally cannot end it
		// (DIV-1316).
		if e.Owner == SelfSlot {
			e.Stall++
			if e.Stall >= stallLimit {
				refused := w.pursuitRefused(s, i)
				if w.restAt(s, i) && refused {
					w.answerRefusedPursuit(i)
				}
			}
			return
		}
		if aimsAtEnd {
			w.cancelPursuit(s, i)
		}
		return
	}
	if w.advanceStep(s, i, step[0], heldFirstCall) != stepTurned {
		w.countPursuitPass(i)
	}
}

// countPursuitPass counts one near search of pursuer i and removes the head
// node of its static list when the head lies within dynamicByStaticLookup of
// the mover (AI-373, AI-417).
func (w *World) countPursuitPass(i int) {
	e := &w.entities[i]
	e.Pursuit.Passes++
	if route := w.routes[i]; len(route) > 0 && (cell{x: e.X, y: e.Y}).chebyshevTo(route[0]) <= dynamicByStaticLookup {
		w.routes[i] = route[1:]
	}
}

// cancelPursuit is the raised route-failure flag of pursuer i: the walk ends
// and the order machine answers the refusal (AI-ROUTE-045, DIV-1520). The
// pursuit forgets its victim, so an order at the same victim later starts
// with a full search (DIV-2557).
func (w *World) cancelPursuit(s *routeScratch, i int) {
	w.entities[i].Pursuit = PursuitSearch{}
	if w.restAt(s, i) {
		w.answerRefusedPursuit(i)
	}
}
