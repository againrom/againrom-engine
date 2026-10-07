package sim

// A GROUP ORDER: one order over a set of entities.
//
// Every order this package had before named one entity, and a selection sent to
// one cell was that many separate orders that happened to agree. That is not
// what is being reconstructed: a player order there builds a group out of the
// selection and issues to it, and what the group knows about itself — where its
// members stand relative to one another, and which of them is slowest — is read
// exactly once, at the order, and reaches every member.
//
// So a group order here is a SET OF COMMANDS sharing a tag, gathered inside one
// advance and applied once. The tag is a correlation key for that advance and is
// stored nowhere: no world field, no byte-form field, nothing in the digest. A
// world cannot be asked which groups it holds, because it holds none — what
// survives an order is the per-member state the order wrote, and nothing else.

// clamp is (x, y) brought inside b, per axis independently.
//
// The destination of every member of a group order goes through it, on BOTH
// arms, because the routine that issues one member's destination in the original
// clamps into the map's playable rectangle whether or not a formation offset
// moved it. That rectangle's own four bytes are not something this tree has; our
// bounds are its analogue, and the clamp is the behaviour.
//
// An extent that is not positive has no cell to clamp into, so the low end wins
// and the answer is zero. Such a world holds no in-bounds cell at all — its grid
// is empty by gridCells' own rule — so every destination is equally unreachable
// and none of them is more wrong than another.
func (b Bounds) clamp(x, y int32) (int32, int32) {
	return clampAxis(x, b.Width), clampAxis(y, b.Height)
}

// clampAxis is one axis of the clamp above: into [0, extent-1], and zero when the
// extent names no cell.
func clampAxis(v, extent int32) int32 {
	if v < 0 || extent <= 0 {
		return 0
	}
	if v > extent-1 {
		return extent - 1
	}
	return v
}

// formationSpread is how far a member may stand from its group's centroid, in
// whole cells and by Chebyshev distance, for the group to be in formation. One
// member past it takes the WHOLE GROUP out of formation, which is why the test
// that computes it does not stop at the first member it likes.
//
// It is a compile-time constant in the original too — carried by no shipped
// file, so lifting it changes no file's bytes — and it lives on the AI manager
// rather than on the world, which is a distinction with nothing here to hang it
// from. What it means is measurable: two members four cells apart are in
// formation and five cells apart are not.
const formationSpread = 2

// groupSpeedInit is the running minimum's starting value in the original's own
// per-member loop, and it is a BOUND rather than a sentinel: a group all of
// whose members are faster than it keeps it, so no group term can exceed 249.
//
// It is a customisation limit and it is written here rather than smoothed away.
// The shipped speed column runs 8 to 35, sixteen times inside it.
const groupSpeedInit = 250

// groupOrder applies the one order that begins at cmds[first], and marks every
// command it consumed.
//
// MEMBERSHIP is every command of this kind in cmds carrying the same tag, in
// slice order, resolved to the entities the world still holds that are ALIVE,
// each counted once. A command naming an absent or felled entity is ignored,
// which is what every command naming one already is, and a duplicate naming is
// taken once so that it moves neither the centroid's divisor nor the minimum.
//
// The ORDERED CELL is the first command's, and no other member's coordinates are
// read. A group order carries one destination — the cell the player clicked —
// and where each member goes from there is this file's business, not the
// caller's; reading each command's own cell would let a caller hand in an order
// no click could produce and get a formation computed against it.
//
// THE WRITE IS TWO CALLS NOW: commandGroup first, taking every member out of
// whatever group — placed, or a command group an earlier order already
// built — it stood in and putting all of them into ONE fresh command group
// at the move order; then issueGroupDestination, unchanged, over the same
// members and the same cell. This function's own job stays what it was
// before 0117 — resolving WHO the order reaches and WHERE it is aimed —
// and the write is still that pair handed both, one call longer than it used
// to be.
//
// IT DOES NOT WRITE ON THE PLACED GROUP'S OWN RECORD: the law's own scenario
// command allocates a fresh group whose byte the setter moves, this tree
// cannot allocate THAT one, and the placed group's order — if it has a
// record at all — is left exactly as it stood. commandGroup's write is a
// second, player-owned group word sitting beside that record rather than
// inside it — see CommandGroup, world.go. THE KIND IS THE FOURTH THING IT
// RESOLVES NOW. Four command kinds reach here — the move, the two aimed
// orders and the stance — and every one of them resolves membership and
// aim by the identical scan above. What the kind decides is only the WRITE
// at the foot: which order byte the fresh group is held at, whether the
// formation distribution runs, whether the members' posts are anchored, and
// whether they are handed to the actor layer instead of being decided for.
// Three sibling functions would have been three copies of the scan and three
// places for "an order names a group, not a unit" to drift.
//
// MEMBERSHIP IS BY KIND AND TAG TOGETHER. A stance press and a move press in
// one advance carry their own tags already, but a tag is a small counter and
// the kinds are what keep two orders of different sorts from ever merging on
// one.
func (w *World) groupOrder(cmds []Command, first int, consumed []bool) {
	kind, tag := cmds[first].Kind, cmds[first].Group
	if kind == KindGroupRetreat {
		w.retreatOrder(cmds, first, consumed)
		return
	}

	// The membership scan starts at first and not at 0: any earlier command
	// carrying this tag would have made THAT one the first, and consumed this
	// one along with it.
	var members []int
	for k := first; k < len(cmds); k++ {
		c := cmds[k]
		if c.Kind != kind || c.Group != tag {
			continue
		}
		consumed[k] = true
		i := indexOfEntity(w.entities, c.Entity)
		// Liveness alone. A member that is fighting, approaching, charging,
		// relaxing or casting is still an addressed member of this order: the
		// centroid, the formation and the rate below are all computed over the
		// set the player addressed, and dropping members from it changes where
		// the survivors are sent and how fast they walk.
		if i < 0 || !w.entities[i].Alive() || w.stoneCursed(i) {
			continue
		}
		if kind == KindGroupDefend && (w.entities[i].OffMap || w.entities[i].Owner == 0) {
			continue
		}
		if !containsIndex(members, i) {
			members = append(members, i)
		}
	}
	ordered := cell{x: cmds[first].X, y: cmds[first].Y}
	if kind == KindGroupStance && ordered.x != int32(orderGuard) && ordered.x != int32(orderStandGround) {
		return
	}
	if kind == KindGroupDefend {
		// The subject must be admitted before even an in-flight scroll is
		// canceled. Missing/off-map subjects leave the entire order intact.
		w.commandDefend(members, EntityID(uint32(ordered.x)))
		return
	}
	for _, i := range members {
		w.cancelScroll(i)
	}
	switch kind {
	case KindGroupMoveTo:
		w.commandGroup(members, orderMove, ordered)
		w.issueGroupDestination(members, ordered)
	case KindGroupSwarmTo:
		w.commandGroup(members, orderSwarm2, ordered)
		w.issueGroupDestination(members, ordered)
	case KindGroupStance:
		w.commandStance(members, ordered.x)
	case KindGroupPatrolTo:
		w.commandPatrol(members, ordered)
	}
}

// isGroupKind reports whether a command kind is one of the four the group
// arm takes. It is a function and not a set literal at the call site so that
// the four kinds and the switch at groupOrder's foot are read together: a
// kind admitted here with no arm there would consume its commands and change
// nothing, which is a silent drop rather than an unimplemented order.
func isGroupKind(kind uint8) bool {
	switch kind {
	case KindGroupMoveTo, KindGroupSwarmTo, KindGroupStance, KindGroupPatrolTo, KindGroupDefend, KindGroupRetreat:
		return true
	}
	return false
}

func (w *World) commandStance(members []int, order int32) {
	if order != int32(orderGuard) && order != int32(orderStandGround) {
		return
	}
	defer func() {
		for _, i := range members {
			w.syncSavedPost(i)
		}
	}()
	w.commandGroup(members, uint8(order), cell{})
	for _, mi := range members {
		w.entities[mi].PostX, w.entities[mi].PostY = w.entities[mi].X, w.entities[mi].Y
	}
	if order == int32(orderStandGround) {
		w.commandStandGround(members)
	}
}

func (w *World) commandPatrol(members []int, ordered cell) {
	if w.savedGroups != nil {
		w.commandSavedGroup(members, 0, ordered)
		w.setSavedPatrol(members, ordered)
		return
	}
	w.commandGroup(members, orderNone, ordered)
	for _, mi := range members {
		e := &w.entities[mi]
		w.clearOrder(mi)          // destination, stall count and stored route
		w.retainCycleForState(mi) // victim, and the attack cycle unless one is loaded
		e.clearGroupSpeed()       // the group rate term
		e.ActorState = actorStatePatrol
		e.PatrolHeadX, e.PatrolHeadY = e.X, e.Y
		e.PatrolTailX, e.PatrolTailY = w.bounds.clamp(ordered.x, ordered.y)
		e.PatrolLeg = patrolLegTail
		// The post, on the script sibling's own ground (script.go,
		// `AI-PATROL-018` `L00415`): the ring and the post are one write
		// in the law's setter and are one write in both of ours.
		e.PostX, e.PostY = e.X, e.Y
	}
}

// issueGroupDestination is the distribution itself, lifted out of groupOrder
// so the player's own group move and the script's Move and Swarm 2 setters
// share one body rather than three drifting copies of it: the formation gate
// on the members' spread about their centroid, each member offset by its own
// displacement from it when in formation and sent to the bare cell when not,
// the rate term from the slowest member on the formation arm alone, every
// destination clamped into the map and every fight ended.
//
// It writes NO group order: the law's own scenario command has nothing here
// to allocate and the player path leaves the placed group's order untouched
// on the same grounds (D-3), so a caller that wants one written writes it
// itself, beside the call — which is what the script's two setters do.
//
// An order with no member does nothing at all, and needs no guard beyond the
// loops below running zero times: the centroid is never asked for, because
// the divisor it would need is the member count.
func (w *World) issueGroupDestination(members []int, ordered cell) {
	var g *SavedGroup
	if members != nil && w.hasSavedFormations() {
		// The ordinary-command prologue has already created this exact Group.
		g = w.savedFormationGroup(members)
	}
	w.issueSavedGroupDestination(g, members, ordered)
}

func (w *World) issueSavedGroupDestination(g *SavedGroup, members []int, ordered cell) {
	defer func() {
		for _, i := range members {
			w.syncSavedDestination(i)
		}
	}()
	if len(members) == 0 {
		return
	}
	cx, cy := groupCentroid(w.entities, members)

	// ONE FLAG, COMPUTED ONCE, read at exactly two places below. It is the whole
	// of why the distribution and the rate term are one story: they are two
	// effects of one decision, and a build that computed either without the
	// other would have a behaviour the thing being reconstructed does not.
	//
	// AI-FORMOWNER-314 follows Group+44, even when its members were stamped
	// with the enclosing Player. Null/unknown ownership has no fallback.
	mode := w.FormationMode(w.entities[members[0]].Owner)
	if w.hasSavedFormations() {
		var found bool
		mode, found = w.savedGroupFormation(g)
		if !found {
			return
		}
	}
	formation := w.modeInFormation(mode, members, cx, cy)

	var term uint8
	if formation {
		term = groupMinSpeed(w.entities, members)
	}

	for _, i := range members {
		e := &w.entities[i]
		// EVERY GROUP ORDER ALLOCATES A FRESH GROUP, whose rate term is zero
		// until the formation arm sets one. So the zeroing is unconditional and
		// stands before the arms rather than inside the one that does not set a
		// term — an order that leaves the group out of formation leaves every
		// member with no term, which is the same statement.
		e.clearGroupSpeed()
		// AND IT ENDS THE FIGHT THIS MEMBER WAS IN BETWEEN CYCLES, by the rule the
		// plain move arm follows: walking and attacking are one state, so a member
		// sent somewhere holds a destination and no victim, except behind a loaded
		// cycle (MOVE-FORM-036, MOVE-GATE-035; DIV-1563).
		e.clearAttackBetweenCycles()

		x, y := ordered.x, ordered.y
		if formation {
			// THE DISTRIBUTION: the ordered cell offset by this member's own
			// displacement from the centroid, so the shape the player selected
			// is the shape that is sent.
			//
			// Each offset is stored as a 16-bit field and read back as a BYTE,
			// and both narrowings are written out rather than folded away: the
			// original stores and reads exactly that pair of widths, and a
			// consumer that kept the full displacement would part company with
			// it the moment a forced-formation member stood more than 127
			// cells from the centroid. formationOffset retains the existing
			// signed-byte interpretation; this story changes owner selection,
			// not the offset arithmetic.
			x += formationOffset(e.X - cx)
			y += formationOffset(e.Y - cy)
			e.GroupSpeed = term
		}
		tx, ty := w.bounds.clamp(x, y)
		// Same rule and same reason as the plain move writer in step.go
		// (R3-C1, round3-review.md): a stored route must end at the
		// entity's target or be empty, and this write can move the target
		// without the walk rebuilding the route on this tick for a member a
		// book cast owns or one still paying transit. Drop the stale route
		// here rather than leave a mismatch the byte form refuses.
		if !e.HasTarget || tx != e.TargetX || ty != e.TargetY {
			w.routes[i] = nil
		}
		w.cancelTurnForTargetChange(i, tx, ty)
		e.TargetX, e.TargetY = tx, ty
		e.HasTarget = true
	}
}

func (w *World) commandFloor() uint32 {
	floor := uint32(1)
	if w.script == nil {
		return floor
	}
	for _, c := range w.script.checks {
		if c.HasGroup {
			if above := commandFloorAbove(c.Group); above > floor {
				floor = above
			}
		}
	}
	for _, in := range w.script.instants {
		if in.HasGroup {
			if above := commandFloorAbove(in.Group); above > floor {
				floor = above
			}
		}
	}
	return floor
}

func commandFloorAbove(group uint32) uint32 {
	if group == ^uint32(0) {
		return group
	}
	return group + 1
}

func (w *World) freeCommandGroup() uint32 {
	id := w.commandFloor()
	for {
		taken := false
		for i := range w.entities {
			if effectiveGroup(w.entities[i]) == id {
				taken = true
				break
			}
		}
		if !taken {
			return id
		}
		if id == ^uint32(0) {
			return 0
		}
		id++
	}
}

func (w *World) commandGroup(members []int, order uint8, ordered cell) {
	for _, i := range members {
		w.cancelStructureUse(w.entities[i].ID)
		w.noteActorMotionOrder(w.entities[i].ID)
	}
	if w.savedGroups != nil {
		w.commandSavedGroup(members, order, ordered)
		return
	}
	if len(members) == 0 {
		return
	}
	for _, mi := range members {
		w.entities[mi].CommandGroup = 0
		// AND THE RING GOES OUT WITH THE MEMBERSHIP. A patrol is a standing
		// arrangement exactly as a group is, and this is the one statement that
		// takes a member out of whatever it stood in — so every player order
		// there is, and every one there will be, ends a patrol without a rule
		// anyone has to keep.
		//
		// IT IS NOT DEFENSIVE. Without it the actor pass re-issues the
		// patroller's ring on the very tick the order arrives, one phase after
		// the command was applied: the order is written, then overwritten,
		// inside one advance. The player's own Patrol arm calls this function
		// FIRST and installs its ring after, so it is not undone by its own
		// release.
		w.entities[mi].clearPatrol()
		w.entities[mi].clearEscort()
	}
	id := w.freeCommandGroup()
	if id == 0 {
		// freeCommandGroup's own unreachable edge (see its doc): nothing is
		// left to allocate. Rather than write a command group under the
		// sentinel that means "none", this command builds no group at all —
		// every named member stays released to its placed group, which the
		// drop above already left it in, and is the closest this state has
		// to "unchanged".
		return
	}

	var owners []uint32
	for _, mi := range members {
		e := &w.entities[mi]
		if e.Owner == 0 {
			continue //
		}
		e.CommandGroup = id
		if !containsOwner(owners, e.Owner) {
			owners = append(owners, e.Owner)
		}
	}

	for _, owner := range owners {
		var share []int
		for _, mi := range members {
			if w.entities[mi].Owner == owner {
				share = append(share, mi)
			}
		}
		cx, cy := groupCentroid(w.entities, share)
		w.upsertGroup(groupAI{
			owner:      owner,
			group:      id,
			base:       noticeBase(w.entities, share, cx, cy),
			order:      order,
			commandedX: ordered.x,
			commandedY: ordered.y,
		})
	}
}

// upsertGroup writes rec into w.groups at (rec.owner, rec.group):
// overwritten in place if a record already stands there, otherwise inserted
// at the position that keeps the slice ascending by (owner, group) —
// decodeGroups' own requirement of it — rather than appended and sorted
// afterward, so the slice is never observed out of order even transiently.
func (w *World) upsertGroup(rec groupAI) {
	for i := range w.groups {
		if w.groups[i].owner == rec.owner && w.groups[i].group == rec.group {
			w.groups[i] = rec
			return
		}
	}
	at := len(w.groups)
	for i := range w.groups {
		if w.groups[i].owner > rec.owner ||
			(w.groups[i].owner == rec.owner && w.groups[i].group > rec.group) {
			at = i
			break
		}
	}
	w.groups = append(w.groups, groupAI{})
	copy(w.groups[at+1:], w.groups[at:])
	w.groups[at] = rec
}

// containsOwner is containsIndex's own twin over owner ids, for the same
// reason: a group is what a player's click named, small enough that a scan
// costs nothing, and a map here would put Go's randomised iteration order on
// a path that touches a world.
func containsOwner(owners []uint32, owner uint32) bool {
	for _, o := range owners {
		if o == owner {
			return true
		}
	}
	return false
}

// formationOffset is one axis of a member's displacement from its group's
// centroid as the order carries it: stored as a 16-bit field and read back as a
// SIGNED byte.
//
// It is its own function because the pair of narrowings is the whole content and
// nothing in this build can reach past them: the spread gate admits only
// displacements in [-2, +2], so an order this package can be given never sees
// either width bite. They are here for the formation mode this tree does not
// model — where the gate is skipped and a member may stand anywhere — and being
// a function is what lets a test exercise the widths directly instead of
// asserting a behaviour no order can produce.
//
// The byte is read SIGNED, and that is OURS: the rows this is built from say
// "as bytes" and not which extension. Over every displacement the modelled mode
// admits the two readings agree, so nothing measurable here separates them.
func formationOffset(delta int32) int32 { return int32(int8(int16(delta))) }

// groupCentroid is the group's centre cell, per axis, over the members named.
//
// IT IS COMPUTED IN SUB-CELL UNITS AND THAT IS THE WHOLE POINT. A position in
// the original is a cell byte and a fraction byte, and a mover standing still
// sits at the CENTRE of its cell — half the sub-cell grid — so the summands
// already carry a half-cell each. Sum them, divide by the member count, take the
// whole cells of the quotient, and the answer is the mean rounded to the NEAREST
// cell, TIES UP. Nobody chose that rounding and nothing here applies one
// afterwards; it falls out of where a resting unit stands. The independent check
// on it is the spread threshold's own worked example — two members four cells
// apart are in formation and five cells apart are not — which comes out that way
// under this reading and under no other.
//
// The half is spelled subCell/2 rather than as a number so that the centred
// fraction and the sub-cell grid cannot drift apart, and the arithmetic is int64
// so that a member count times a coordinate cannot wrap into a plausible mean.
// The result always fits an int32: the quotient is bounded by the largest
// coordinate among the members plus a half-cell, and the second division takes
// the half back off.
//
// BOTH DIVISIONS ARE FLOOR. The original divides unsigned and shifts, over
// coordinates that are unsigned bytes; ours may be negative, where Go's own
// division truncates toward zero instead. Floor agrees with the unsigned divide
// over every position the original can hold and is defined outside it, which is
// the narrower of the two ways to be wrong.
func groupCentroid(ents []Entity, members []int) (int32, int32) {
	var sx, sy int64
	for _, i := range members {
		sx += int64(ents[i].X)*subCell + subCell/2
		sy += int64(ents[i].Y)*subCell + subCell/2
	}
	n := int64(len(members))
	return int32(floorDiv(floorDiv(sx, n), subCell)), int32(floorDiv(floorDiv(sy, n), subCell))
}

// floorDiv is a/b rounded toward negative infinity, for a positive b. Go's own
// division truncates toward zero, and the two differ exactly where a is
// negative — which is where the centroid's own arithmetic would otherwise stop
// agreeing with an unsigned shift.
func floorDiv(a, b int64) int64 {
	q := a / b
	if a%b != 0 && a < 0 {
		q--
	}
	return q
}

// inFormation reports whether the group is in formation: no member standing
// further from the centroid than formationSpread, by Chebyshev distance in whole
// cells.
//
// ONE MEMBER OVER THE LINE TAKES THE WHOLE GROUP OUT, which is why the answer is
// a property of the group and not of each member — a straggler does not walk to
// the ordered cell while its companions keep their shape; nobody keeps their
// shape.
//
// The differences are taken in int64: two members at opposite ends of the
// coordinate range are a difference no int32 holds, and a wrap there would read
// as a tight formation.
func inFormation(ents []Entity, members []int, cx, cy int32) bool {
	for _, i := range members {
		dx, dy := abs64(int64(ents[i].X)-int64(cx)), abs64(int64(ents[i].Y)-int64(cy))
		if dx > dy {
			dy = dx
		}
		if dy > formationSpread {
			return false
		}
	}
	return true
}

// abs64 is the magnitude of v. The one value it cannot represent is the smallest
// int64, which no coordinate difference reaches: both operands are int32-wide,
// so their difference is bounded well inside the range.
func abs64(v int64) int64 {
	if v < 0 {
		return -v
	}
	return v
}

// groupMinSpeed is the rate term a formation order stores: the minimum over the
// members of the speed each would move at alone (aloneSpeed, which carries its
// own overload penalty), in the original's own widths.
//
// The loop is written out rather than replaced by a plain minimum because the
// two narrowings are real and are customisation limits, not noise. The running
// value is a BYTE starting at groupSpeedInit, and each comparison is against
// that byte zero-extended; the member's speed is compared as a SIGNED 16-BIT
// value — the width of the class field it comes from — and stored as that
// value's LOW BYTE. So a group of members all faster than groupSpeedInit keeps
// it, and a class speed past a byte folds onto its low half. Both are far
// outside the shipped speed column and both are reachable by customising it.
func groupMinSpeed(ents []Entity, members []int) uint8 {
	min := uint8(groupSpeedInit)
	for _, i := range members {
		if speed := int16(aloneSpeed(ents[i])); speed < int16(min) {
			min = uint8(speed)
		}
	}
	return min
}

// containsIndex reports whether is holds i. It is a scan and not a set, because
// a group is the units a player has selected: the walk costs nothing at that
// size, and a map here would put Go's randomised iteration order on a path that
// touches a world.
func containsIndex(is []int, i int) bool {
	for _, k := range is {
		if k == i {
			return true
		}
	}
	return false
}
