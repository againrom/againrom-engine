package sim

import "sync"

// The canonical route search: a label-correcting wave over the enterability
// relation, and the backwards walk that reads a route out of the labels it
// leaves.
//
// It is a plain function in this package's own directory, in no sub-package and
// behind no interface. That is deliberate and it is not style: the determinism
// wall's behavioural half is a source scan over THIS directory that skips
// subdirectories, so a pkg/sim/route package would satisfy the import DAG and
// silently escape the scan that holds a search to integer arithmetic. An
// interface would be worse still — a second behaviour-steering identity beside
// the mode byte, one inside the digest and one outside.
//
// Nothing here is a receiver's to hold. The label plane and the two frontiers
// are made by the caller and passed in, so a world gains no field, the encoder
// cannot reach them, and a route exists only for as long as the call that built
// it.

// cell is one lattice position. It is a comparable value with no pointer in it,
// so a route is a slice of positions and nothing else — a caller handed one
// holds nothing that still reaches into a world.
type cell struct {
	x, y int32
}

// flatCost is what a mover OUTSIDE the ordinary ground domain is charged for
// a step, whatever ground it is crossing: 2 orthogonally and 3 diagonally,
// and its cell's cost byte is not read at all.
//
// It was every mover's cost here until this story, which is worth naming rather
// than leaving as history: the flat value was never a placeholder for the ground
// rule, it is the non-ground rule, and applying it to everybody charged ground
// movers somebody else's arm rather than undercharging them.
const flatCost = 2

// stepCost is the cost of one step: for a GROUND mover the cost byte of the cell
// the step ENTERS, for every other domain the flat value, and in both cases plus
// half of it truncated when the step is diagonal.
//
// The truncation is the contract and not a rounding to tidy away. At a cost byte
// of 1 the diagonal and the orthogonal arm come out equal, and at 0 both are
// free — neither is a cost any derivation produces, and both are states a caller
// may build.
//
// It takes the cost byte rather than a cell and a world, so the four arms stay
// testable without either, and so that WHICH cell is charged is decided at each
// call site. That is deliberate: the two searches flood in opposite directions,
// so the entered cell is a different loop variable in each, and a signature
// carrying only the step's delta could not tell them apart.
func stepCost(d Domain, cost uint8, dx, dy int32) uint64 {
	c := uint64(cost)
	if d != DomainGround {
		c = flatCost
	}
	if dx != 0 && dy != 0 {
		return c + (c >> 1)
	}
	return c
}

// routeScratch is one tick's working memory: the label plane, the two frontier
// lists and the occupancy plane, made once and reused by every search of that
// tick.
//
// The label plane holds cost+1 so that zero means unlabelled — a fresh plane then
// needs no fill, and a reset zeroes only the slots a search recorded touching
// rather than sweeping W*H of them per unit per tick. It is uint64 because a
// label is bounded by whatever budget the search runs under, and the near
// search's is still derived from an UNCLAMPED target: a target may name any cell
// however far outside the map, so the scaled rule's D is bounded only by the
// int32 range. A narrower plane would have to argue that no reachable sum
// overflows it, and the flat rule does not make that argument for it — the two
// rules share this plane.
//
// There are two frontier lists and only one label plane, and the asymmetry is the
// contract's: relaxation reads and writes the plane where it stands, so a cell
// lowered earlier in a generation is read at its new value by every frontier
// cell after it. Only the frontier is double-buffered.
//
// occ is how many entity footprints cover each in-bounds cell, and it is what makes
// enterability an O(1) question instead of a walk of the entity slice — the wave
// puts that question to all eight neighbours of every frontier cell, so the walk
// made one tick cost units x cells x 8 x units. It is a COUNT and not a set of
// occupied cells: a world whose entities already share a cell is advanced rather
// than repaired, so a cell may carry two, and a set cannot say when the second
// of them leaves.
//
// It carries BOTH LAYERS, laid out layer-major: cell i of layer L is at
// L*cells + i. A mover contends with the movers of its own layer and with no
// others, so a flyer and a ground unit share a cell freely while two ground
// units do not. One slice and not two because two would be two lengths and two
// resets to keep level, and level is the whole of what makes the pair usable.
//
// cells is that stride, and it is the ONE place the count lives: every reader
// and every writer multiplies by this field rather than reaching for the bounds
// or for a slice length of its own, which is how two layers come to be indexed
// by two notions of the same number.
//
// It lives here rather than on the world for the reason the label plane does, and
// the point is sharper for this one: an index on the world is state beside the
// digest that UnmarshalBinary would have to rebuild, and it is exactly what
// TestTheCanonicalWorldsFieldSetsArePinned refuses by name.
type routeScratch struct {
	plane    []uint64
	touched  []int
	frontier []cell
	next     []cell
	occ      []int32
	cells    int
}

// at is the count standing on cell index i of layer, and it is the only way a
// reader turns the pair (layer, cell) into a slot.
func (s *routeScratch) at(layer, i int) int32 { return s.occ[layer*s.cells+i] }

// add moves the count on cell index i of layer by d. It is the only writer, so
// the seed, the move loop and a settling flyer all reach the plane one way.
func (s *routeScratch) add(layer, i int, d int32) { s.occ[layer*s.cells+i] += d }

// addFootprint adjusts every in-bounds cell covered by the square footprint
// whose anchor is (x, y). A partially off-map authored actor contributes only
// its in-bounds cells; admission rejects such a destination separately through
// terrainOpenFootprint.
func (s *routeScratch) addFootprint(w *World, layer int, side uint8, x, y, d int32) {
	n := footprintSide(side)
	for dy := int32(0); dy < n; dy++ {
		for dx := int32(0); dx < n; dx++ {
			if i, ok := w.cellIndex(x+dx, y+dy); ok {
				s.add(layer, i, d)
			}
		}
	}
}

// newRouteScratch makes the planes a world needs — one slot per in-bounds cell,
// in the same row-major order the grid uses — with occupancy already taken from
// w's entities as they stand.
//
// It takes the WORLD and not merely its bounds, because a scratch that answers
// about occupancy answers about one world's entities and no other's. A
// constructor over bounds alone would hand back a scratch that says every cell is
// free, and every caller would have to remember to fill it.
func newRouteScratch(w *World) *routeScratch {
	n := gridCells(w.bounds)
	s := &routeScratch{plane: make([]uint64, n), occ: make([]int32, 2*n), cells: int(n)}
	s.occupy(w)
	return s
}

// tickScratches holds step scratches between ticks. A pooled scratch is not
// world state: it is reset and re-occupied before use, so a tick sees exactly
// what newRouteScratch would have built.
var tickScratches sync.Pool

// tickRouteScratch is newRouteScratch for the step, reusing a pooled scratch of
// the same grid size. releaseRouteScratch returns it once the step is done.
func tickRouteScratch(w *World) *routeScratch {
	if s, _ := tickScratches.Get().(*routeScratch); s != nil && int64(s.cells) == gridCells(w.bounds) {
		s.reset()
		s.occupy(w)
		return s
	}
	return newRouteScratch(w)
}

func releaseRouteScratch(s *routeScratch) { tickScratches.Put(s) }

// occupy re-points s at w: the counts are cleared and taken again from w's
// entities as they stand right now.
//
// An entity standing off the map has no slot and is counted nowhere, which is the
// right answer rather than a gap — an out-of-bounds cell is not enterable on
// bounds alone, so no occupant of one can change an answer.
//
// A non-living entity is counted while it dwells or while ordinary restoration
// can return it to life. The latter keeps the route plane and Heal admission on
// the same population: a mover cannot take a body's cell between Dwell expiry
// and revival. A terminal body still leaves the plane after its dwell.
//
// An entity is counted in ITS OWN LAYER and in no other, so what a mover asks
// about is the movers it actually contends with.
//
// WHO is counted is counted() below, and this loop is its only sweep.
func (s *routeScratch) occupy(w *World) {
	for i := range s.occ {
		s.occ[i] = 0
	}
	for j := range w.entities {
		e := &w.entities[j]
		if m := w.motionFor(e.ID); m != nil && m.Current {
			continue
		}
		if !counted(*e) {
			continue
		}
		s.addFootprint(w, e.Domain.layer(), e.TokenSize, e.X, e.Y, 1)
	}
	w.occupySavedMotions(s)
}

// counted reports whether e stands in its layer's plane at all.
//
// A BODY IS COUNTED WHILE IT DWELLS OR REMAINS RESTORATIVE-TARGETABLE. Dwell is
// the minimum: even a terminal corpse blocks through its fall. The restorative
// population is the longer-lived half: HP 0 through -9 with a positive maximum
// can become living through Heal and therefore keeps its cell until revival or
// until damage/decay reaches the -10 finished-body floor. At that floor a body
// releases the cell once its dwell is over, preserving terminal corpse routing.
//
// And an AIR mover is counted only while it holds NO TARGET — the soft rule: a
// moving flyer is counted nowhere, so it neither blocks another mover nor is
// blocked by one, and two flyers on crossing paths fly through each other. That
// arm is a LIVING mover's alone now, a body holding no target of any kind.
//
// It is one function with THREE readers and not a condition written out
// three times: the seed above, the self-presence term in enterable, and the
// re-seed a flyer gets when its order ends. A seed rule that disagreed with
// a subtraction would drive a count negative on the asker's own cell, and
// the predicate would then answer that a cell holding one other mover is
// free. AN OFF-MAP ENTITY IS COUNTED NOWHERE, and that one line is the whole
// of "the removal clears the actor's footprint from the occupancy grid". It
// stands FIRST, before the life ladder, because it is a statement about
// presence rather than about a stage of one: a unit the script has taken off
// the map holds no ground whether it is alive, downed, dwelling or decaying.
//
// It is done HERE and not in the seed, on the corpse's own ground stated below:
// this is the one predicate the seed, enterable's self-presence term and the
// flyer re-seed all read, and a seed rule that disagreed with a subtraction
// would drive a count negative on the asker's own cell.
func counted(e Entity) bool {
	if e.OffMap {
		return false
	}
	if !e.Alive() {
		return e.Dying() || e.restorativeTargetable()
	}
	// A flyer standing through a loaded attack cycle is not moving whatever
	// destination waits behind the cycle (DIV-1509), so it stays in the plane.
	return e.Domain != DomainAir || !e.HasTarget || e.HasAttackTarget && e.AttackPhase != AttackReady
}

// moved keeps the counts answering what a walk of the entity slice would answer:
// an entity that has just advanced is taken off every cell of the footprint it
// left and put on every cell of the footprint it entered.
//
// It is called in the same statement that moves an entity, and that is the whole
// of the rule. The predicate answers over the entities AS THEY STAND when it is
// asked, and Step rests on that — at index i the entities before i hold their
// post-move cells and those after it their pre-move ones — so an index built once
// and read all tick would answer for the top of the tick and quietly change which
// unit wins a contended cell, and with it the digest.
func (s *routeScratch) moved(w *World, e Entity, from, to cell) {
	layer := e.Domain.layer()
	s.addFootprint(w, layer, e.TokenSize, from.x, from.y, -1)
	s.addFootprint(w, layer, e.TokenSize, to.x, to.y, 1)
}

// reset returns the plane to unlabelled by walking the touch list, so the cost
// is what the last search labelled and not what the map could hold.
func (s *routeScratch) reset() {
	for _, i := range s.touched {
		s.plane[i] = 0
	}
	s.touched = s.touched[:0]
	s.frontier = s.frontier[:0]
	s.next = s.next[:0]
}

// cellIndex is (x, y)'s slot in the grid and in the label plane, and whether the
// cell is in bounds at all. The two share one indexing so that a cell's
// passability and its label are never read from two different places.
func (w *World) cellIndex(x, y int32) (int, bool) {
	if x < 0 || y < 0 || x >= w.bounds.Width || y >= w.bounds.Height {
		return 0, false
	}
	return int(y)*int(w.bounds.Width) + int(x), true
}

// relation is which of the two passability relations a search runs over. It is
// what tells the two searches apart in KIND — the mode still says which route is
// taken, and the cost model and the neighbour set are the same under both.
//
// It is no longer the only thing that differs. The two searches also run under
// different budget RULES, and the two parameters are deliberately separate: a
// relation is about which cells exist for a search, a rule about how long it may
// look for them, and folding them into one flag would make either impossible to
// move without moving the other.
//
// terrainRelation reads the MAP alone: in bounds, the asking mover's own domain
// bits clear, whatever units stand where. Both of the things it reads are
// immutable while a world is advanced, so a route over it does not depend on
// where in a tick it was asked for. unitRelation adds occupancy as it stands
// when the question is put, which is the relation every shipped search has run
// over.
//
// It arrives as a PARAMETER and is read at the offer, where a neighbour is
// tested. A second copy of each search would be two contracts to keep in
// agreement; a relation chosen per cell inside one search would be neither
// relation, so the corner test asks the same one the cell test asks.
type relation bool

const (
	terrainRelation relation = false
	unitRelation    relation = true
)

// dynamicWindow is the near search's half-width: it may label the cells within
// Chebyshev distance 8 of the mover's own cell and no others, clipped to bounds
// by the same predicate that refuses any other out-of-bounds cell.
//
// The value follows from the look-ahead rather than from taste. A sub-goal is at
// most four cells along a stored route, so 8 leaves a detour as much room again
// as the sub-goal is far — and it bounds what one near search may label at
// 17 * 17 = 289 cells, however far the order's real destination lies.
//
// noWindow is the far search's bound, which is no bound at all: a search
// carrying it sweeps the whole map in either mode. A window is safe exactly when
// something else is unbounded, and the far search is that something. It is why
// the start/target rectangle the optimised mode used to search inside is gone
// rather than kept beside this: the system holds exactly one spatial bound, and
// it is on the half that is allowed one.
const (
	dynamicWindow int32 = 8
	noWindow      int32 = -1
)

// nearSlack is the near search's budget slack, the term the scaled rule carries
// under the quarter of the Chebyshev distance: 3, the value read for the search
// aimed at a waypoint a few cells off.
//
// staticSlack is the far search's, 5, and it is read again because the flat
// budget is an OVERRIDE with a gate rather than that search's whole rule: a far
// search whose goal is not open falls through to the computed form, and the
// computed form the far arm carries is this slack's, not the near arm's.
const (
	nearSlack   int64 = 3
	staticSlack int64 = 5
)

// flatGenerations is the far search's whole budget when a human participant
// owns a one-cell mover: a thousand generations, whatever the distance to the
// target.
//
// It is not "no budget". Past a Chebyshev distance of 800 the scaled form
// exceeds it, so beyond that this is the TIGHTER of the two rules, and the two
// modes go on disagreeing about what is reachable. What it removes is the case
// that made the budget a function of the straight line, where the whole
// allowance for leaving that line was a quarter of the distance along it.
const flatGenerations int64 = 1000

// budgetRule is which of the four termination budgets a search runs under, and
// it is the whole of what tells them apart in that respect.
//
//   - scaledBudget is the near search's: max(nearSlack, D>>2) + D.
//   - flatBudget is the far search of a one-cell unit a human participant owns: a
//     thousand generations while the goal's footprint is open, and the
//     staticSlack form otherwise (MOVE-TERM-003).
//   - staticBudget is the far search of every other owner's one-cell unit: the
//     staticSlack form, max(staticSlack, D>>2) + D, with no override.
//   - largeBudget is the far search of every unit larger than one cell, whoever
//     owns it, the n x n arm: staticSlack + D, with neither a D>>2 term nor an
//     override (MOVE-TERM-003, MOVE-PARAM-006).
//
// The near search has an n x n arm too, DynamicScanAhead + D, and takes no rule
// of its own for it. Its goal is a cell of a stored route at most four cells
// along it and inside a window of eight, so D is at most 8, where D>>2 is under
// nearSlack and scaledBudget is nearSlack + D as well.
//
// It is a small NAMED TYPE, the shape settleRule already has, and deliberately
// not a func value. A function here would be a second behaviour-steering
// identity beside the mode byte — one inside the digest and one outside — which
// is the arrangement this package's own header refuses.
//
// The far search's choosers are farBudgetFor's callers and the near search's is
// Step. Neither search picks its own rule, so they cannot come to disagree about
// which is which.
type budgetRule uint8

const (
	scaledBudget budgetRule = 0
	flatBudget   budgetRule = 1
	staticBudget budgetRule = 2
	largeBudget  budgetRule = 3
)

// humanParticipantUnit reports whether a human participant owns e: the
// condition under which the far search of a one-cell unit takes the flat
// budget. The original reads the owning Player's whole dword at +0x28 for zero,
// which a campaign map authors on its roster record 0 and on no other, and the
// party takes that record's slot (MOVE-TERM-003, UNIT-OWNER-009, ALM-140,
// ALM-PLAYER-069). An entity that names no roster slot has no owning Player to
// read, so it keeps the participant's form, the one this build has always given
// it. DIV-1569.
func humanParticipantUnit(e Entity) bool {
	return e.Owner == 0 || e.Owner == SelfSlot
}

// farBudgetFor is the budget rule the far search of the entity at index i runs
// under: the n x n arm for a unit larger than one cell whoever owns it, and for a
// one-cell unit the flat rule when a human participant owns it and the
// staticSlack form otherwise. Every far search of an entity reads it here, so the
// movement pass and the searches that mirror it agree on a mover's budget.
//
// The footprint is asked before the owner because the claim gives the flat
// thousand to the static 1x1 arm alone, and the search reads the mover's
// footprint side and no other size (MOVE-TERM-003, MOVE-SPEED-011). The party's
// own hired Catapult and Ballista, footprint 2, take the n x n arm with every
// other unit larger than one cell. DIV-1570.
func (w *World) farBudgetFor(i int) budgetRule {
	e := &w.entities[i]
	switch {
	case footprintSide(e.TokenSize) > 1:
		return largeBudget
	case humanParticipantUnit(*e):
		return flatBudget
	}
	return staticBudget
}

// settleRule is whether a search that cannot label its goal may settle for a
// substitute near it, or must answer that there is no route — and, where it may,
// how far from the goal it may look.
//
// It is the fourth of the parameters that tell one search from another, and it
// is separate from the other three for the reason they are separate from each
// other: a relation is which cells exist for a search, a window where it may
// look, a budget rule how long, and this one what it may come back with. Folding
// any pair into one flag makes either impossible to move without moving the
// other.
//
// BOTH SEARCHES SETTLE, and they differ in the RING BOUND alone. That is why
// this is a three-valued type and not the bool it was: the bound belongs to the
// question "what may this search come back with" and to no other parameter, so a
// fifth parameter carrying it would be a second thing to keep in agreement with
// this one. The two callers in Step are again the only choosers.
//
// settleOrdered is the far search's, whose goal is the order's own destination:
// the bound is shaped on the distance ordered, so a long order looks further for
// a substitute than a short one. settleStep is the near search's, whose goal is
// a WAYPOINT four cells along a route the far search chose over terrain alone —
// so the bound is flat, and a step aside is the whole of what it can come back
// with.
type settleRule uint8

const (
	exactGoal     settleRule = 0
	settleOrdered settleRule = 1
	settleStep    settleRule = 2
)

// settles reports whether a search under s may come back with a cell other than
// the one it was aimed at. It is one predicate rather than two comparisons
// written out at each site, so adding a third settling rule cannot leave one
// site behind.
func (s settleRule) settles() bool { return s != exactGoal }

// window is a search's spatial bound: the cells within Chebyshev distance half
// of centre, and every cell there is when half is noWindow.
//
// The centre is the mover's own cell and never the target's, which is the whole
// difference between this and the rectangle it replaces: what a near search may
// label depends on where the MOVER stands and not on how far away it was sent,
// so one search's cost is statable without knowing the order.
type window struct {
	centre cell
	half   int32
}

// holds reports whether (x, y) is a cell win permits a search to label.
//
// The arithmetic is int64 because a target may name any cell: the difference of
// two int32 coordinates does not fit in one.
func (win window) holds(x, y int32) bool {
	if win.half < 0 {
		return true
	}
	dx := int64(x) - int64(win.centre.x)
	if dx < 0 {
		dx = -dx
	}
	dy := int64(y) - int64(win.centre.y)
	if dy < 0 {
		dy = -dy
	}
	if dy > dx {
		dx = dy
	}
	return dx <= int64(win.half)
}

// terrainOpen reports whether the map itself permits a mover of domain d to
// stand on (x, y): in bounds, with that domain's own block bits clear.
//
// It takes a DOMAIN and no scratch and no asker, because it is the per-cell
// primitive used by terrainOpenFootprint. The grid, bounds and mover domain do
// not change while a world is advanced.
//
// The domain arrives as a value rather than as an entity index for the same
// reason: an index would tie the terrain question to a world's entity slice,
// where what it actually depends on is one byte of one mover.
func (w *World) terrainOpen(d Domain, x, y int32) bool {
	i, ok := w.cellIndex(x, y)
	if key, inside := savedPlaneKey(x, y); ok && inside && w.savedCellPlanes != nil {
		return w.savedCellPlanes.Static[key]&savedStaticMask(d) == 0 && (d == DomainAir || w.grid[i]&blockMagicWall == 0)
	}
	return ok && w.grid[i]&d.blocksForSaved(w.hasSavedStructures) == 0
}

// footprintSide normalises the table's zero value to the one-cell footprint
// every pre-footprint Entity literal has always meant.
func footprintSide(side uint8) int32 {
	if side == 0 {
		return 1
	}
	return int32(side)
}

// terrainOpenFootprint reports whether every cell of e's square footprint is
// in bounds and open to its movement domain at anchor (x, y).
func (w *World) terrainOpenFootprint(e Entity, x, y int32) bool {
	return w.terrainOpenSize(e.Domain, e.TokenSize, x, y)
}

// terrainOpenSize reads only the two actor fields that affect static admission.
// Route relaxation asks this for every neighbour; passing the complete Entity
// at that site copies its unrelated combat and inventory state for each offer.
func (w *World) terrainOpenSize(domain Domain, size uint8, x, y int32) bool {
	if w.savedCellPlanes != nil {
		for dy := int32(0); dy < footprintSide(size); dy++ {
			for dx := int32(0); dx < footprintSide(size); dx++ {
				if !w.terrainOpen(domain, x+dx, y+dy) {
					return false
				}
			}
		}
		return true
	}
	return terrainOpenSizeIn(w.bounds, w.grid, domain, size, x, y, w.hasSavedStructures)
}

// terrainOpenFootprintIn is the same predicate over the decoded bounds and grid
// before a World exists. decodeRoutes uses it so a stored route cannot carry an
// anchor that the runtime's full-footprint search would never produce.
func terrainOpenFootprintIn(b Bounds, grid []byte, e Entity, x, y int32, savedMode ...bool) bool {
	saved := len(savedMode) != 0 && savedMode[0]
	return terrainOpenSizeIn(b, grid, e.Domain, e.TokenSize, x, y, saved)
}

func terrainOpenSizeIn(b Bounds, grid []byte, domain Domain, size uint8, x, y int32, saved bool) bool {
	n := footprintSide(size)
	for dy := int32(0); dy < n; dy++ {
		for dx := int32(0); dx < n; dx++ {
			i, ok := cellIndexIn(b, x+dx, y+dy)
			if !ok || grid[i]&domain.blocksForSaved(saved) != 0 {
				return false
			}
		}
	}
	return true
}

// routeTerrainOpen reports whether the mover can put its complete footprint at
// every anchor in route. Runtime route construction already guarantees this;
// the separate read exists for a same-version save written by an earlier build,
// whose decoder admitted route terrain at the anchor alone.
func (w *World) routeTerrainOpen(e Entity, route []cell) bool {
	for _, c := range route {
		if !w.terrainOpenFootprint(e, c.x, c.y) {
			return false
		}
	}
	return true
}

// footprintsOverlap reports whether the two square footprints share a cell.
func footprintsOverlap(ax, ay int32, as uint8, bx, by int32, bs uint8) bool {
	an, bn := footprintSide(as), footprintSide(bs)
	return ax < bx+bn && bx < ax+an && ay < by+bn && by < ay+an
}

// placementOpen is the linear fit test used before the tick's route scratch
// exists: every destination cell is terrain-open and no other counted actor on
// the mover's occupancy layer overlaps any part of the destination footprint.
func (w *World) placementOpen(self int, x, y int32) bool {
	e := w.entities[self]
	if !w.terrainOpenFootprint(e, x, y) {
		return false
	}
	if w.savedMotion != nil {
		return w.occupancyOpenFootprint(newRouteScratch(w), self, x, y)
	}
	layer := e.Domain.layer()
	for j := range w.entities {
		if j == self {
			continue
		}
		o := w.entities[j]
		if counted(o) && o.Domain.layer() == layer &&
			footprintsOverlap(x, y, e.TokenSize, o.X, o.Y, o.TokenSize) {
			return false
		}
	}
	return true
}

// occupancyOpenFootprint is placementOpen's scratch-backed occupancy half.
// The asker's own current footprint is subtracted cell by cell so an adjacent
// step may keep the cells shared by its old and new footprints.
func (w *World) occupancyOpenFootprint(s *routeScratch, self int, x, y int32) bool {
	e := w.entities[self]
	n := footprintSide(e.TokenSize)
	layer := e.Domain.layer()
	m := w.motionFor(e.ID)
	for dy := int32(0); dy < n; dy++ {
		for dx := int32(0); dx < n; dx++ {
			cx, cy := x+dx, y+dy
			i, ok := w.cellIndex(cx, cy)
			if !ok {
				return false
			}
			occupants := s.at(layer, i)
			if m != nil && m.Current && w.motionOwnsCell(e.ID, layer, cx, cy) || (m == nil || !m.Current) && counted(e) && entityCoversCell(e, cx, cy) {
				occupants--
			}
			if occupants != 0 {
				return false
			}
		}
	}
	return true
}

// enterable reports whether the entity at index self may stand on (x, y): every
// cell of its footprint is open to the terrain, and no OTHER same-layer entity
// covers any of them as the entities stand right now.
//
// "No other" is the count on the scratch less the asker's own presence, which is
// the same predicate a walk of the entity slice computes and is why the count is
// kept rather than a flag: two entities on one cell must still refuse a third,
// and refuse each other.
//
// The asker's presence is subtracted only from destination cells its current
// footprint covers and only when counted() put it in the plane. This is what
// permits adjacent large-actor steps without hiding a second actor on one of
// the shared cells. A dead unit never asks: the move loop skips it before it
// reads a target.
//
// BOTH HALVES ARE THE ASKER'S OWN DOMAIN'S. The terrain half refuses a flyer a
// cell the border closes to it and permits it one water closes to a ground
// mover; the occupancy half reads the asker's own LAYER, so a mover contends
// with the movers of that layer and with no others. Ground and ghost share a
// layer, which is why a ghost blocks a ground mover and a flyer does not.
func (w *World) enterable(s *routeScratch, self int, x, y int32) bool {
	e := &w.entities[self]
	return w.terrainOpenSize(e.Domain, e.TokenSize, x, y) && w.occupancyOpenFootprint(s, self, x, y)
}

// restFree reports whether the entity at index self may come to REST with its
// footprint anchored on (x, y): no OTHER mover of its own layer covers it.
//
// It answers TRUE for a ground and a ghost mover unconditionally, and that is
// the whole of "this test is asked of an air mover and of no other". Their
// occupancy is resolved by the near search, tick by tick, and a rule that
// reached them here would be a second opinion about where they may stop.
//
// It is a second ANSWER and not a third relation. A relation is honoured by both
// searches at every offer a wave makes, which is precisely the fly-through a
// flyer's near search exists to have: the step is blind, and only the cell an
// order ENDS on is asked about.
//
// The asker's own presence is taken off only where counted() wrote it. An air
// mover under orders is absent from the plane, so no subtraction occurs; this
// keeps its current footprint eligible when settleFor returns it.
func (w *World) restFree(s *routeScratch, self int, x, y int32) bool {
	e := &w.entities[self]
	if e.Domain != DomainAir {
		return true
	}
	return w.occupancyOpenFootprint(s, self, x, y)
}

// open is the offer both searches make about a cell: may a route over r put the
// entity at index self on (x, y)?
//
// This is the ONE place a relation is turned into an answer, so the two searches
// cannot come to read it differently, and the two relations are nested rather
// than parallel — the unit-aware one is the terrain one plus a count, so a cell
// the terrain refuses is refused under both.
func (w *World) open(s *routeScratch, r relation, self int, x, y int32) bool {
	if r == terrainRelation {
		e := &w.entities[self]
		return w.terrainOpenSize(e.Domain, e.TokenSize, x, y)
	}
	return w.enterable(s, self, x, y)
}

// label is (x, y)'s label as it stands, and whether it has one.
//
// The start is the one cell that may lie outside the plane: a search begins from
// the entity's own cell whether or not that cell is enterable, and whether or
// not it is even on the map, so that a unit standing on a blocked cell — or off
// the grid entirely — can still be routed off it. An in-bounds start is written
// into the plane like any other cell; an out-of-bounds one has no slot, and its
// label of 0 is answered here.
func (w *World) label(s *routeScratch, start cell, x, y int32) (uint64, bool) {
	if i, ok := w.cellIndex(x, y); ok {
		if s.plane[i] == 0 {
			return 0, false
		}
		return s.plane[i] - 1, true
	}
	if x == start.x && y == start.y {
		return 0, true
	}
	return 0, false
}

// generationBudget is how many generations a search from start toward target may
// run under the rule it is given.
//
// Under flatBudget with an open goal it is a thousand, whatever the two cells
// are. Under scaledBudget it is max(nearSlack, D>>2) + D, D being the Chebyshev
// distance between them — so the whole allowance for leaving the straight line
// is a quarter of the distance along it, which is why that rule is the NEAR
// search's alone. The near search is aimed at a sub-goal at most four cells off
// and is windowed besides, so a bound shaped like the straight line is the right
// shape there and the wrong one for an order sent across a map. A far search of a
// one-cell mover that is not a human participant's runs staticBudget, the same
// shape with the far search's own slack, and flatBudget falls back to that form
// when its goal is not open. A far search of a mover larger than one cell runs
// largeBudget whoever owns it, the far slack and D with no quarter of D between
// them: it agrees with the one-cell form for D under 24 and parts from there.
//
// The RULE is what differs between the two searches, where a slack used to, and
// it arrives as a parameter for the same reason the relation does: one place
// that turns a rule into a number, rather than two that could come to disagree
// about the shift or the sum.
//
// It is COMPUTED on every call that gets past the pre-sweep refusal and CONSULTED
// only after the goal-labelled test — see canonicalRoute, where the order of the
// stop tests is what discharges the decoded override's goal-free half.
//
// The subtraction is in int64 because a target may name any cell, so the
// difference of two int32 coordinates does not fit in one.
func generationBudget(start, target cell, rule budgetRule, goalOpen bool) int64 {
	if rule == flatBudget && goalOpen {
		return flatGenerations
	}
	d := start.chebyshevTo(target)
	if rule == largeBudget {
		return staticSlack + d
	}
	slack := nearSlack
	if rule != scaledBudget {
		slack = staticSlack
	}
	scalar := d >> 2
	if scalar < slack {
		scalar = slack
	}
	return scalar + d
}

// chebyshevTo is the Chebyshev distance between two cells: the larger of the two
// absolute coordinate differences.
//
// It is int64 because a target may name any cell, so the difference of two int32
// coordinates does not fit in one. It is one function because the budget and the
// settle bound are both shaped on it and a second copy is how the two come to
// disagree about which coordinate is subtracted from which.
func (a cell) chebyshevTo(b cell) int64 {
	dx := int64(b.x) - int64(a.x)
	if dx < 0 {
		dx = -dx
	}
	dy := int64(b.y) - int64(a.y)
	if dy < 0 {
		dy = -dy
	}
	if dy > dx {
		return dy
	}
	return dx
}

// canonicalRoute is the route the reconstructed procedure yields from the entity
// at index self to (tx, ty) over the relation r, or ok=false when that procedure
// finds none. The procedure IS the contract, so what follows is written to be
// read against it.
//
// Every cell begins unlabelled. The start's label is 0 and it is the whole first
// frontier. A generation relaxes each frontier cell in turn: for each of its
// eight enterable neighbours, the frontier cell's label plus the cost of the step
// into the neighbour replaces the neighbour's label when strictly less, and the
// neighbour joins the next frontier. A cell may therefore be relabelled any
// number of times and may sit in one frontier twice, and a diagonal is taken with
// no test at all on the two cells it passes between — refusing to cut a corner is
// the OTHER mode's rule, not a shared one.
//
// Before each generation the search stops if the target is labelled, if the
// frontier is empty, or once the budget is spent. A target unlabelled then is no
// route: the wave stops in the first generation that puts ANY label on the
// destination, so the label it reads is a minimum over routes of at most that
// many steps and not a minimum over all routes. That is the whole reason this
// mode and the optimised one are two searches and not one with a quality knob.
//
// A fourth stop comes before all three, and only for a search that may not
// settle: a target the mover may not enter at all is answered without a wave. A
// settling search runs the wave precisely there — the labels it leaves are what
// a substitute is chosen from — so the refusal and the settle are the two halves
// of one question. See the comment at the test itself.
//
// A start that equals the target yields an empty route with ok true. Nothing here
// calls it that way — an entity already standing on its target neither searches
// nor steps — but the answer is the consistent one rather than an error.
//
// The plane is left holding this search's labels. Nothing downstream reads them;
// the next search's reset is what clears them, which is what makes the reset cost
// the labels written rather than the map's size.
func (w *World) canonicalRoute(s *routeScratch, self int, r relation, half int32, rule budgetRule, settle settleRule, tx, ty int32) ([]cell, bool) {
	start := cell{x: w.entities[self].X, y: w.entities[self].Y}
	target := cell{x: tx, y: ty}
	win := window{centre: start, half: half}
	// The mover's domain, read once, from the same record the terrain test reads
	// it from — so which mover reads the ground and which pays flat cannot come
	// to differ between the two questions.
	dom := w.entities[self].Domain

	s.reset()
	goalOpen := win.holds(target.x, target.y) && w.open(s, r, self, target.x, target.y)
	// A SETTLING search must be able to STOP on its goal, and stopping is resting.
	// For a ground or a ghost mover restFree is true and this term changes
	// nothing; for a flyer it is what makes a goal another flyer is resting on
	// count as unreachable, so the wave runs on and the substitute path takes it.
	if settle == settleOrdered {
		goalOpen = goalOpen && w.restFree(s, self, target.x, target.y)
	}
	if start != target && !goalOpen && settle == exactGoal {
		// Nothing but the start is ever labelled without being open under r, so a
		// target that is not open cannot end up labelled however long the
		// wave runs — and the wave would run its whole budget to establish it.
		// The answer is the one the spent sweep reaches, at the same point and
		// with the stall raised by the same rule; a search that could not have
		// succeeded is not made to succeed by being cut short.
		//
		// It returns having labelled nothing, and that is the only difference an
		// observer can find: the touch list and the frontier are both empty here,
		// where a budget-spent refusal leaves both full and a sealed map leaves
		// the touch list full.
		return nil, false
	}
	if i, ok := w.cellIndex(start.x, start.y); ok {
		s.plane[i] = 1 // label 0, held as cost+1
		s.touched = append(s.touched, i)
	}
	s.frontier = append(s.frontier, start)

	budget := generationBudget(start, target, rule, goalOpen)
	// Whether a labelled goal ENDS the wave. A start that is its own target
	// answers immediately, and a search that may not settle stops on the label
	// alone -- which is every search this package ran before the term above
	// existed. Only a settling search asks the rest question, and only a flyer
	// can answer it no: for every other mover this is the bare label test it was.
	stopOnGoal := start == target || settle == exactGoal || goalOpen
	for gens := int64(0); ; gens++ {
		if _, ok := w.label(s, start, target.x, target.y); ok && stopOnGoal {
			break
		}
		if len(s.frontier) == 0 || gens >= budget {
			// The wave is over with the goal unlabelled, which is the one state
			// a substitute is chosen in. A search that may not settle answers
			// that there is no route, which is what every search did before
			// there was a second answer.
			if settle.settles() {
				if sub, ok := w.settleFor(s, self, settle, start, target); ok {
					return w.walkBack(s, dom, win, start, sub)
				}
			}
			return nil, false
		}

		s.next = s.next[:0]
		for _, c := range s.frontier {
			// Read as it stands: a cell that a neighbour lowered earlier in this
			// same generation is relaxed from its NEW label, and a duplicate
			// entry relaxes from the same value twice rather than from the one
			// it was appended with.
			from, ok := w.label(s, start, c.x, c.y)
			if !ok {
				continue
			}
			for dx := int32(-1); dx <= 1; dx++ {
				for dy := int32(-1); dy <= 1; dy++ {
					if dx == 0 && dy == 0 {
						continue
					}
					nx, ny := c.x+dx, c.y+dy
					if !win.holds(nx, ny) || !w.open(s, r, self, nx, ny) {
						continue
					}
					i, _ := w.cellIndex(nx, ny) // open implies in bounds
					// This wave floods FORWARD from the mover, so the cell the
					// step enters is the neighbour, and the charge is its own.
					held := from + stepCost(dom, w.searchCostAt(i, cell{x: nx, y: ny}), dx, dy) + 1
					if s.plane[i] != 0 && held >= s.plane[i] {
						continue
					}
					if s.plane[i] == 0 {
						s.touched = append(s.touched, i)
					}
					s.plane[i] = held
					s.next = append(s.next, cell{x: nx, y: ny})
				}
			}
		}
		s.frontier, s.next = s.next, s.frontier
	}

	return w.walkBack(s, dom, win, start, target)
}

// walkBack reads the route out of the labels, backwards from the target.
//
// At each cell it takes, among that cell's eight labelled neighbours, the one
// minimising the neighbour's label plus the cost of the step from it into the
// current cell. The scan is dx = -1, 0, +1 outer and dy = -1, 0, +1 inner, and
// the accept is ASYMMETRIC: an orthogonal candidate displaces the best on less
// than or equal, a diagonal one only on strictly less. So which of two equal
// candidates is taken depends on where each sits in that scan, and the order is
// as much of the contract as the comparison is.
//
// The centre is not scanned. Nine neighbours would make a cell its own
// predecessor reachable, and a self-step needs a step cap to terminate; eight
// removes the loop instead of bounding it.
//
// It ends by comparing coordinates with the start rather than by reading a zero
// out of the plane. An out-of-bounds start has no slot to read, and leaning on
// "no label can fall below the start's 0" would make a stated rule a consequence
// of arithmetic that a cost plane could falsify.
func (w *World) walkBack(s *routeScratch, dom Domain, win window, start, target cell) ([]cell, bool) {
	var route []cell
	// The walk is BOUNDED, and the bound is the world's own cell count. It
	// cannot fire while every cost on the walk is positive: the label then
	// falls strictly at each step, so the walk visits a strictly decreasing
	// sequence and cannot outlast the cells. A cost byte of ZERO breaks that
	// — every candidate ties, the accept below picks by scan order alone, and
	// a labelled region of three cells cycles forever. No derivation produces a
	// zero, a caller may pass one and a decoder must take one, and a hang is a
	// worse failure than a wrong route.
	limit := gridCells(w.bounds)
	for cur := target; cur != start; {
		if int64(len(route)) > limit {
			return nil, false
		}
		route = append(route, cur)

		var best cell
		var bestCost uint64
		found := false
		for dx := int32(-1); dx <= 1; dx++ {
			for dy := int32(-1); dy <= 1; dy++ {
				if dx == 0 && dy == 0 {
					continue
				}
				nx, ny := cur.x+dx, cur.y+dy
				if !win.holds(nx, ny) {
					continue
				}
				l, ok := w.label(s, start, nx, ny)
				if !ok {
					continue
				}
				// The labels measure cost FROM the mover, so this walk runs
				// backwards and the cell each step ENTERS is the one being left
				// — cur, not the candidate.
				c := l + stepCost(dom, w.costAt(cur), dx, dy)
				switch {
				case !found:
				case dx != 0 && dy != 0:
					if c >= bestCost {
						continue
					}
				default:
					if c > bestCost {
						continue
					}
				}
				best, bestCost, found = cell{x: nx, y: ny}, c, true
			}
		}
		if !found {
			return nil, false
		}
		cur = best
	}

	for i, j := 0, len(route)-1; i < j; i, j = i+1, j-1 {
		route[i], route[j] = route[j], route[i]
	}
	return route, true
}

// stepRings is how far from its WAYPOINT the near search may look for a
// substitute: eight, flat, whatever the waypoint's distance.
//
// It is flat where the far search's bound is shaped on the distance ordered, and
// the asymmetry is the two goals'. The far search's goal is the cell a player
// named, so how far it is worth looking for something near it grows with how far
// away it was; the near search's goal is a waypoint four cells along a route the
// far search chose over TERRAIN ALONE, so what a substitute has to do is get the
// mover past whatever body the far search could not see — a job whose size does
// not depend on how far the order pointed.
const stepRings int64 = 8

// settleRings is how far from the requested cell a settling search may look for
// a substitute, under the rule it runs: eight for the near search, and for the
// far one a quarter of the Chebyshev distance the mover was sent, plus four.
//
// The far arm is shaped on the same distance the budget is, and the pair is what
// makes the two agree: a wave allowed a quarter of the distance in slack over
// the straight line reaches about that far past the goal, and this is the ring at
// which looking for what it left stops being worth it. Read as a bound on r
// rather than as a count of rings — see settleFor, where the growth test is the
// decoded one and not a loop over 1..limit.
//
// exactGoal never reaches here: a search that may not settle returns before a
// bound is asked for. It shares the far arm rather than being an arm of its own,
// because a third arm would be a number nothing reads.
func settleRings(settle settleRule, start, target cell) int64 {
	if settle == settleStep {
		return stepRings
	}
	return (start.chebyshevTo(target) >> 2) + 4
}

// The int32 range, as the bound a ring probe is held to. A ring around a cell
// near the edge of that range runs off it, and a coordinate that cannot be an
// int32 names no cell of any world — so it is refused here rather than wrapped
// into one that looks like a cell and is not.
//
// They are written out rather than taken from math's own names because pkg/sim
// imports nothing but what it must: the determinism wall's scan is lexical, and
// the cheapest way to keep it saying something is to give it less to read.
const (
	minCoord int64 = -1 << 31
	maxCoord int64 = 1<<31 - 1
)

// settleFor is the substitute a failed wave settles for: among the cells the
// wave LABELLED, the cheapest to reach from the mover in the first expanding
// square ring around the REQUESTED cell that holds any labelled cell at all.
//
// The two halves are measured from different places and that is the whole rule.
// The RING ORDER is Chebyshev outward from the cell the order named, so a
// substitute is looked for near what was asked for; the CHOICE INSIDE A RING is
// the smallest label, and a label is the accumulated step cost from the mover's
// own start cell — so the cell taken is the one cheapest to reach FROM THE
// MOVER, not the one nearest the request. The obvious rival — "the nearest free
// cell to the clicked cell" — is half of this and no more.
//
// "FREE" IS "LABELLED", and it is not passability. The plane this reads is the
// one the wave that has just failed wrote, so it already carries that search's
// relation, its window and its budget: a perfectly enterable cell the wave never
// reached carries no label and is invisible here, and no second predicate is
// asked. That is why this takes the scratch and no relation — there is nothing
// left for one to answer.
//
// The whole ring is scanned before its best is taken, so the exit test sits
// after the ring and not inside it: the first ring holding ANY labelled cell
// decides, and within it the label decides. The centre is never probed — r
// starts at 1 — which costs nothing, since this runs only where the requested
// cell is unlabelled.
//
// The scan order is a walk of i from -r to +r taking the ring's four sides in
// turn, and the accept is STRICTLY less, so among cells of equal label the one
// reached earliest in that walk wins. The order is as much of the contract as
// the comparison is: it is what makes two identical worlds settle on the same
// cell, and it is reproduced here rather than replaced by a coordinate
// tie-break, which would answer differently.
//
// The mover's own start cell is labelled like any other and may be returned. The
// caller reads that as "it can get no nearer", because a route from a cell to
// itself is empty.
//
// A candidate must also be one the mover may REST on, which narrows membership
// and nothing else: the ring order, the strict accept and the growth bound are
// untouched, so a ring whose labelled cells the mover may not rest on does not
// stop the growth -- the search goes to the next ring and answers no route at
// the bound, which is the answer a wave that labelled nothing already gives.
func (w *World) settleFor(s *routeScratch, self int, settle settleRule, start, target cell) (cell, bool) {
	limit := settleRings(settle, start, target)
	if start.chebyshevTo(target) <= pursuitRings && w.pursuitGoalIsVictim(self, settle, target) && limit < pursuitRings+1 {
		limit = pursuitRings + 1
	}
	victim, towardVictim := w.pursuitVictimCell(self, settle)

	var best cell
	var bestLabel uint64
	var bestGap int64
	for r := int64(1); ; r++ {
		found := false
		probe := func(x, y int64) {
			if x < minCoord || x > maxCoord || y < minCoord || y > maxCoord {
				return
			}
			l, ok := w.label(s, start, int32(x), int32(y))
			if !ok || !w.restFree(s, self, int32(x), int32(y)) {
				return
			}
			var gap int64
			if towardVictim {
				dx, dy := x-int64(victim.x), y-int64(victim.y)
				gap = dx*dx + dy*dy
			}
			if found && (gap > bestGap || gap == bestGap && l >= bestLabel) {
				return
			}
			best, bestLabel, bestGap, found = cell{x: int32(x), y: int32(y)}, l, gap, true
		}
		tx, ty := int64(target.x), int64(target.y)
		for i := -r; i <= r; i++ {
			probe(tx+i, ty+r)
			probe(tx+i, ty-r)
			probe(tx+r, ty+i)
			probe(tx-r, ty+i)
		}
		if found {
			return best, true
		}
		if r+1 >= limit {
			return cell{}, false
		}
	}
}

// pursuitVictimCell is where an attacker's near search looks for a free cell
// beside a taken one: the cell of the unit victim it pursues, so that a crowd
// closing on one victim fills the free cells nearest it before it stands behind
// a taken one.
func (w *World) pursuitVictimCell(self int, settle settleRule) (cell, bool) {
	e := w.entities[self]
	if settle != settleStep || !e.HasAttackTarget || e.AttackTargetKind != AttackTargetUnit {
		return cell{}, false
	}
	ti := indexOfEntity(w.entities, e.AttackTarget)
	if ti < 0 {
		return cell{}, false
	}
	return cell{x: w.entities[ti].X, y: w.entities[ti].Y}, true
}

// pursuitRings is how many rings around its victim a pursuer's far search looks
// for a free cell once the victim is that many cells away (DIV-2456).
const pursuitRings int64 = 8

// pursuitGoalIsVictim reports whether a settling far search of the entity at
// index self is aimed at the cell of the unit victim it pursues.
func (w *World) pursuitGoalIsVictim(self int, settle settleRule, target cell) bool {
	if settle != settleOrdered {
		return false
	}
	v, ok := w.pursuitVictimCell(self, settleStep)
	return ok && v == target
}
