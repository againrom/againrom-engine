package sim

// Map presence: the mission script's power to take a unit off the map and put
// it back (0164). Five instant opcodes reach the two arms below — 16 and 17 for
// one unit, 32 and 33 for every member of a group, and 18 for the pair of a
// removal and a placement.
//
// AI-362: traversal unlink preserves survivor order; a return appends at the
// tail. Identity-sorted entity storage remains independent of that sequence.

// takeOffMap is instant 16's arm: it takes the entity at index i off the map
// and does nothing else.
//
// IT IS IDEMPOTENT, and that is the decoded arm's own first three instructions:
// an actor already carrying the bit returns immediately. So this reports whether
// it changed anything, which is what lets a caller — and a test — tell a removal
// from a re-removal without reading the world twice.
//
// WHAT IT DOES NOT TOUCH is the whole of the rest of the entity: group,
// command group, owner, position, container, equipment, health, order and
// attack target. A removal that reached into another entity's order — an
// attacker's victim, say — would be a write the arm does not make; that victim
// is dropped by the ATTACKER instead, on the tick it next advances
// (advanceAttack, combat.go), which is where a dead victim is already dropped.
func (w *World) takeOffMap(i int) bool {
	if w.entities[i].OffMap {
		return false
	}
	w.unlinkActorTraversal(w.entities[i].ID)
	w.entities[i].OffMap = true
	w.invalidateActorMotion(w.entities[i].ID, "native removal supersedes original cell membership")
	return true
}

// The two search bounds, both engine search bounds rather than stored fields
// (`TRIG-INSTCENSUS-046`): widening either changes no byte of any shipped file.
//
// placeRadius is the radius every placement search runs at, and placeExact the
// degenerate radius instant 17 tries the retained cell at first. attemptsAt is
// how many random attempts a radius makes: the original's counter starts at 0,
// is raised after a FAILED attempt and is compared JLE against (r*r)/2 + 1, so
// the body runs (r*r)/2 + 2 times — 2 at radius 0 and 6 at radius 3.
const (
	placeExact  int32 = 0
	placeRadius int32 = 3
)

func attemptsAt(r int32) int { return int((r*r)/2 + 2) }

// returnToMap is instant 17's arm: it puts the entity at index i back on the
// map at the cell it retained, through the bounded search.
//
// THE CELL IS NOT AUTHORED ANYWHERE. No parameter of the node names one; the
// entity kept its coordinates when it was removed, and those are the cell. That
// is the whole reason the off-map bit is state and not a position.
//
// THE SEARCH IS TWO STAGES AND THEY ARE THE ORIGINAL'S TWO CALLS: one at radius
// 0, which is the exact cell twice because the half-radius and the draws are all
// zero there, and then one at radius 3. It reports whether the unit is back on
// the map.
//
// ON FAILURE NOTHING CHANGES AT ALL — the unit stays off the map, at the cell it
// retained, and there is no retry. This build commits no cell until an attempt
// succeeds, where the original writes the position object at each attempt and so
// leaves a failed search's last attempt standing (spec SC-4). The difference is
// unobservable on the shipped corpus: every trigger naming one of these nodes
// carries once = 1, so no shipped map ever searches from a drifted cell.
//
// AN ENTITY ALREADY ON THE MAP IS LEFT ALONE. The original's arm has no such
// test — it would append an already-listed actor a second time — and that is
// undefined behaviour rather than behaviour, refused here on registerAt's and
// the give-all self-transfer's own precedent.
func (w *World) returnToMap(i int) bool {
	if !w.entities[i].OffMap {
		return false
	}
	x, y := w.entities[i].X, w.entities[i].Y
	if w.tryAttempts(i, x, y, placeExact) {
		return true
	}
	return w.placeNear(i, x, y)
}

// placeNear is the placement half alone: the six random attempts around (x,
// y) and then the exhaustive square, with the cell supplied by the caller
// rather than read off the entity. It is what instant 18 reaches, and what
// returnToMap falls through to once the exact cell has been tried twice.
func (w *World) placeNear(i int, x, y int32) bool {
	if w.tryAttempts(i, x, y, placeRadius) {
		return true
	}
	return w.trySquare(i, x, y, placeRadius)
}

// tryAttempts is the random stage at one radius: attemptsAt(r) draws of
// (x - r/2 + uniform(r), y - r/2 + uniform(r)), each offered to the fit test,
// stopping at the first that fits.
//
// EVERY ATTEMPT DRAWS TWICE, radius 0 included. The original's helper calls
// its random routine at every attempt whatever the radius, and this
// package's uniform draws even where the answer is fixed, for the same
// reason: how many draws an advance makes must follow from the world's state
// and never from a value a draw produced. At radius 0 both draws return 0
// and both are consumed, so the two attempts are the exact cell twice.
//
// THE WINDOW IS THE CLAIM'S OWN ARITHMETIC and not its prose summary (spec
// SC-5). `TRIG-RETURN-042` quotes the attempt as (x - r/2 + rand(r), ...), which
// at r = 3 reaches x - 1 through x + 2, and separately summarises the six
// attempts as falling inside the 3x3 square. The quoted expression is the
// instruction-level fact and is what is implemented; the exhaustive stage below
// is 3x3 in both readings.
func (w *World) tryAttempts(i int, x, y int32, r int32) bool {
	half := r / 2
	for n := attemptsAt(r); n > 0; n-- {
		cx := x - half + w.rng.uniform(r)
		cy := y - half + w.rng.uniform(r)
		if w.placeAt(i, cx, cy) {
			return true
		}
	}
	return false
}

// trySquare is the exhaustive stage: every cell of [x-r/2, x+r/2] x [y-r/2,
// y+r/2] in order, stopping at the first that fits. It draws nothing.
//
// THE AXIS ORDER IS AUTHORED (spec SC-6). The claim says the square is scanned
// "in order" and does not say which axis is outer; this is x outer, y inner.
func (w *World) trySquare(i int, x, y int32, r int32) bool {
	half := r / 2
	for cx := x - half; cx <= x+half; cx++ {
		for cy := y - half; cy <= y+half; cy++ {
			if w.placeAt(i, cx, cy) {
				return true
			}
		}
	}
	return false
}

// placeAt offers one anchor: if the entity at index i fits there with its whole
// footprint, it is put there and put back on the map, and this reports true.
// Otherwise nothing is written.
//
// THE TWO WRITES ARE ONE ACT. A cell written without the bit cleared would move
// a unit that is still off the map, and a bit cleared without a cell would put
// one back wherever it happened to be standing.
func (w *World) placeAt(i int, x, y int32) bool {
	if !w.attachFootprint(i, x, y) {
		return false
	}
	e := &w.entities[i]
	e.X, e.Y = x, y
	w.invalidateActorMotion(e.ID, "native placement supersedes original movement")
	e.clearStride()
	w.appendActorTraversal(e.ID)
	e.OffMap = false
	return true
}

// placeFree is the fit test: may the entity at index i stand with its complete
// footprint anchored on (x, y) as the world stands right now?
//
// IT ASKS THE SAME TWO RELATIONS THE MOVEMENT PREDICATE ASKS — every footprint
// cell is open to the asker's OWN domain's terrain, and no OTHER counted entity
// of that domain's layer overlaps one — so this is not a second movement law.
// What the original's own fit test rejects a cell for beyond occupancy is
// Unknown (`TRIG-RETURN-042`): the routine was not read. This build's answer is
// authored on that seam and provenance.md names it.
//
// IT IS A LINEAR SCAN AND NOT A routeScratch. placementOpen owns that scan
// because the script pass runs before the tick's scratch plane is built. It
// reads counted, which is the seed's own predicate, and the shared footprint
// geometry, so the two cannot disagree about who covers a cell.
//
// THE ASKER IS SKIPPED. It is off the map at every call this package makes, so
// counted already answers false for it; skipping by index says so once rather
// than relying on that, which keeps the test right if it is ever offered an
// on-map asker.
func (w *World) placeFree(i int, x, y int32) bool {
	return w.placementOpen(i, x, y)
}

// groupMembers is every entity carrying the given group, in ascending id,
// and it is what instants 32 and 33 walk.
//
// IT READS Group AND NOT effectiveGroup. Every other script arm over a group
// — the count check, the hand-over, the group order — reads the map's
// own group word; reading the command group here would make a script's
// removal of a group depend on whether the player had moved it, which no
// claim supports.
//
// THE DEAD ARE INCLUDED, on the group hand-over's own ground: membership and
// identity outlive death everywhere in this runtime, and the decoded arms test
// no member before calling their helper.
//
// THE MEMBERSHIP IS READ ONCE, BEFORE THE FIRST WRITE, on the hand-over's own
// ground again — though neither arm here writes the field this selects on, so
// unlike the hand-over it would still be correct done in one pass. Collecting
// first is what makes the answer the membership as the node found it, whatever
// a later arm comes to write.
func (w *World) groupMembers(group uint32) []int {
	if w.savedGroups != nil {
		groups, err := w.resolveSavedGroups(group)
		if err != nil {
			return nil
		}
		var out []int
		for _, g := range groups {
			for _, m := range g.Members {
				out = append(out, indexOfEntity(w.entities, m.Entity))
			}
		}
		return out
	}
	owner, ok := w.scriptGroupOwner(group)
	if !ok {
		return nil
	}
	var members []int
	for i := range w.entities {
		if w.entities[i].Group == group && w.entities[i].Owner == owner {
			members = append(members, i)
		}
	}
	return members
}
