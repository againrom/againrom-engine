package sim

import "fmt"

// facingStep is 32: one direction's worth of the facing byte, so the eight
// directions are 0, 32, 64 … 224.
//
// It is one number in one place, read by both directions of the conversion, so
// the rounding that maps a byte onto a direction and the store that maps a
// direction onto a byte cannot come to disagree about how wide a direction is.
const facingStep = 32

// facingRound is half a direction, the offset the decoded rounding adds before
// it divides: a byte within 16 of a direction rounds to it.
const facingRound = facingStep / 2

// directions is how many there are. It is a mask below rather than a modulus —
// the count is a power of two — which is what the decoded expression does.
const directions = 8

// stepOf is the cell delta each direction moves by: dx and dy, clockwise from
// north, every component in {-1, 0, +1} (MOVE-DIR-034). The engine builds its
// packed delta table from exactly these two rows.
//
// It is written out as the two rows the claim states and is NOT derived from
// anything this tree already holds, which is what lets the drawing tier's own
// long-shipped octant derivation be checked AGAINST it rather than assumed to
// agree with it.
var stepOf = [directions][2]int32{
	{0, -1},  // 0 north
	{+1, -1}, // 1 north-east
	{+1, 0},  // 2 east
	{+1, +1}, // 3 south-east
	{0, +1},  // 4 south
	{-1, +1}, // 5 south-west
	{-1, 0},  // 6 west
	{-1, -1}, // 7 north-west
}

// dirOfSigns is the inverse of stepOf over the SIGNS of a delta, indexed
// [signIndex(dy)][signIndex(dx)]: the direction whose own step has those two
// signs. The centre entry is noDirection — a delta of no length points nowhere,
// and it is the one cell of this table that is not a direction.
//
// It is a sign table and not a bearing: the magnitudes never enter, so a step
// of one cell and a victim standing three cells away on the same diagonal name
// the same direction. That is exact where the deltas this package feeds it are
// one cell in each axis, which is every step a mover takes and every position a
// victim in reach can stand in.
var dirOfSigns = [3][3]int{
	{7, 0, 1},           // dy < 0, north:  NW, N, NE
	{6, noDirection, 2}, // dy == 0:         W,  —, E
	{5, 4, 3},           // dy > 0, south:  SW, S, SE
}

// noDirection is what a delta of no length names. It is out of [0,8) so that a
// caller which ignored the second return would index the table and be caught,
// rather than silently reading north.
const noDirection = -1

// FacingDir rounds any facing byte to one of eight movement headings.
// MOVE-RATE-029
func FacingDir(f uint8) int { return ((int(f) + facingRound) >> 5) & (directions - 1) }

// facingOfDir is the facing byte that names direction d: d << 5, the store the
// decoded facing derivation makes. It is the inverse of FacingDir on the eight
// directions, and it is written only with a d this file produced.
func facingOfDir(d int) uint8 { return uint8(d * facingStep) }

// facingToward is the facing that points along the delta (dx, dy), and whether
// there is one at all.
//
// A ZERO DELTA HAS NO ANSWER and says so. Every caller leaves the facing it
// found in that case, so a mover that did not move and an attacker standing on
// its victim keep pointing where they pointed — which is the behaviour, not a
// fallback: there is no direction from a cell to itself, and answering north
// would be inventing one.
func facingToward(dx, dy int32) (uint8, bool) {
	d := dirOfSigns[signIndex(dy)][signIndex(dx)]
	if d == noDirection {
		return 0, false
	}
	return facingOfDir(d), true
}

// bearingFacing is the facing of the eight headings nearest the bearing of the
// delta (dx, dy) from an actor to its target actor, and whether there is one.
// Its law is the replayed outputs of AI-361's direction helper: a heading wins
// within 22.5 degrees of the bearing, so a victim at (1,-4) lies due north and
// one at (2,-4) north-east, where facingToward's signs name north-east for both.
// The boundary tan(22.5 degrees) = sqrt(2)-1 is tested exactly in integers as
// (max-min)^2 > 2*min^2, which no integer delta can tie. A zero delta has no
// answer. The law is replayed for deltas within 4 cells; beyond them it is the
// same nearest-heading rule, unreplayed.
func bearingFacing(dx, dy int32) (uint8, bool) {
	const bound = 1 << 30
	clamp := func(v int32) int64 {
		x := int64(v)
		if x > bound {
			return bound
		}
		if x < -bound {
			return -bound
		}
		return x
	}
	x, y := clamp(dx), clamp(dy)
	if x == 0 && y == 0 {
		return 0, false
	}
	ax, ay := x, y
	if ax < 0 {
		ax = -ax
	}
	if ay < 0 {
		ay = -ay
	}
	lo, hi := ax, ay
	if lo > hi {
		lo, hi = hi, lo
	}
	if (hi-lo)*(hi-lo) <= 2*lo*lo {
		return facingToward(dx, dy)
	}
	if ax > ay {
		return facingOfDir(dirOfSigns[1][signIndex(dx)]), true
	}
	return facingOfDir(dirOfSigns[signIndex(dy)][1]), true
}

// signIndex maps a delta's sign onto its row or column of dirOfSigns: 0
// negative, 1 zero, 2 positive.
func signIndex(v int32) int {
	switch {
	case v < 0:
		return 0
	case v > 0:
		return 2
	default:
		return 1
	}
}

// Turning reports the one active shape of the canonical turn state.
func (e Entity) Turning() bool { return e.TurnRemaining != 0 }

// ANIM-136
func (e Entity) DrawnFacing() uint8 {
	if e.TurnState.Present {
		return e.TurnState.Drawn & 0xf0
	}
	if !e.Turning() || e.TurnTotal == 0 || e.TurnRemaining > e.TurnTotal {
		return e.Facing
	}
	arc := facingArc(e.Facing, e.DesiredFacing)
	if arc == 0 {
		return e.Facing
	}
	total := int32(e.TurnTotal)
	elapsed := total - int32(e.TurnRemaining)
	if elapsed <= 0 {
		return e.Facing
	}
	progressed := arc * elapsed / total
	clockwise := int32(uint8(e.DesiredFacing - e.Facing))
	counter := int32(uint8(e.Facing - e.DesiredFacing))
	if counter < clockwise {
		return uint8(int32(e.Facing) - progressed)
	}
	return uint8(int32(e.Facing) + progressed)
}

// clearTurn makes the current facing the inactive desired facing. It is used
// when an action is replaced and when an actor is felled; neither case is a
// completed turn and so neither writes the old desired direction into Facing.
func (e *Entity) clearTurn() {
	e.DesiredFacing = e.Facing
	e.TurnRemaining = 0
	e.TurnTotal = 0
	if !e.Alive() {
		e.TurnState = TurnState{}
	}
}

// facingArc is the unsigned shortest arc between two facing bytes. The tie at
// half a circle is 128 in either direction, so only its magnitude is needed by
// the decoded duration law.
func facingArc(from, to uint8) int32 {
	clockwise := int32(uint8(to - from))
	counter := int32(uint8(from - to))
	if counter < clockwise {
		return counter
	}
	return clockwise
}

// requestFacing starts or preserves the turn needed to face desired. Its
// result says the action must wait. A non-positive rate is the compatibility
// arm: it writes the direction immediately and creates no turn interval.
//
// Reissuing the target preserves progress. A new target steps from the current
// body byte and replaces the client run from its drawn sixteenth.
func (e *Entity) requestFacing(desired uint8, deferStep ...bool) bool {
	if e.Turning() && e.DesiredFacing == desired {
		return true
	}
	arc := facingArc(e.Facing, desired)
	if arc == 0 {
		e.TurnState.Active = false
		e.clearTurn()
		return false
	}
	if e.RotationSpeed <= 0 {
		e.Facing = desired
		e.TurnState.Active = false
		e.clearTurn()
		return false
	}
	drawn := e.DrawnFacing() >> 4
	active := e.TurnState.Active || e.Turning()
	e.DesiredFacing = desired
	next, count := turnStep(e.Facing, desired, uint8(e.RotationSpeed), active)
	deferred := len(deferStep) != 0 && deferStep[0]
	if !deferred {
		e.Facing = next
	}
	e.TurnRemaining = count
	e.TurnTotal = e.TurnRemaining
	counter := e.TurnState.Counter
	if !active {
		counter = 0
	}
	if !deferred {
		counter++
	}
	e.TurnState = TurnState{Present: true, Active: e.Facing != desired, Counter: counter,
		Drawn: drawn << 4, DrawTarget: uint8((int(desired)+8)>>4) & 15, DrawRemaining: e.TurnTotal}
	if !deferred {
		e.advanceDrawnTurn()
	}
	return true
}

// turnToward applies requestFacing to a cell delta. A turn does not destroy
// the stored route (MOVE-TURN-044 corrects MOVE-TURN-031). A zero delta names
// no direction and changes nothing.
func (w *World) turnToward(i int, dx, dy int32) bool {
	return w.turnFacing(i, dx, dy, facingToward)
}

// turnTowardActor is turnToward for a target actor, whose heading is the
// bearing's nearest of the eight (bearingFacing) and not the delta's signs.
func (w *World) turnTowardActor(i int, dx, dy int32) bool {
	return w.turnFacing(i, dx, dy, bearingFacing)
}

func (w *World) turnFacing(i int, dx, dy int32, heading func(dx, dy int32) (uint8, bool)) bool {
	if w.motionActive(w.entities[i].ID) {
		return true
	}
	desired, ok := heading(dx, dy)
	if !ok {
		return false
	}
	if desired != w.entities[i].Facing {
		w.invalidateActorMotion(w.entities[i].ID, "native turn supersedes original mover")
	}
	oldRemaining, oldDesired := w.entities[i].TurnRemaining, w.entities[i].DesiredFacing
	already := w.turnAlreadyStepped(w.entities[i].ID)
	wait := w.entities[i].requestFacing(desired, already)
	if wait && !already {
		w.markTurnStepped(w.entities[i].ID)
	}
	if wait && (oldRemaining == 0 || oldDesired != desired) {
		w.entities[i].startAction(w.tick, int64(w.entities[i].TurnRemaining))
	}
	return wait
}

// cancelTurnForTargetChange ends movement-owned progress when a destination is
// actually replaced. Reissuing the same destination preserves it. A pending
// book cast owns its admitted facing even when a move is attached during the
// wind-up, so that one action is the explicit exception.
func (w *World) cancelTurnForTargetChange(i int, x, y int32) {
	e := &w.entities[i]
	if e.HasTarget && e.TargetX == x && e.TargetY == y {
		return
	}
	w.clearTurnUnlessCasting(i)
}

// clearTurnUnlessCasting is the write above without its test, for the one
// caller that must take the same decision at a moment when the three fields
// the test reads have already been overwritten (armPatrol, 1141). A cast owns
// the actor's facing and is the one thing that keeps a turn.
func (w *World) clearTurnUnlessCasting(i int) {
	e := &w.entities[i]
	if _, casting := w.bookCastIndex(e.ID); !casting {
		e.clearTurn()
	}
}

// advanceTurns takes one rate step per eligible actor. The per-update guard
// shares this budget with later producers. Stone Curse and off-map presence
// freeze both the mover and the client run.
func (w *World) advanceTurns() {
	for i := range w.entities {
		e := &w.entities[i]
		if !e.Alive() || e.OffMap || w.stoneCursed(i) || w.motionActive(e.ID) {
			continue
		}
		if !e.Turning() {
			e.advanceDrawnTurn()
			continue
		}
		if w.turnAlreadyStepped(e.ID) {
			continue
		}
		if e.Facing == e.DesiredFacing {
			e.TurnState.Active = false
			e.advanceDrawnTurn()
			e.clearTurn()
			continue
		}
		if !e.TurnState.Present {
			drawn := e.DrawnFacing() & 0xf0
			counter := e.TurnTotal - e.TurnRemaining
			_, count := turnStep(e.Facing, e.DesiredFacing, uint8(e.RotationSpeed), true)
			e.TurnTotal = count
			e.TurnState = TurnState{Present: true, Active: true, Counter: counter,
				Drawn: drawn, DrawTarget: uint8((int(e.DesiredFacing)+8)>>4) & 15,
				DrawRemaining: count}
		}
		e.Facing, e.TurnRemaining = turnStep(e.Facing, e.DesiredFacing, uint8(e.RotationSpeed), e.TurnState.Active)
		w.markTurnStepped(e.ID)
		e.TurnState.Active = e.Facing != e.DesiredFacing
		e.TurnState.Counter++
		e.advanceDrawnTurn()
	}
}

// turnFault bounds persisted turn operands. Both facing bytes remain total
// over all 256 values, including targets retained from original saves.
func turnFault(e Entity) error {
	if !e.TurnState.Present && e.TurnState != (TurnState{}) || e.TurnState.DrawTarget > 15 || e.TurnState.DrawRemaining > 128 {
		return fmt.Errorf("invalid turn message state")
	}
	if !e.Turning() {
		if e.DesiredFacing != e.Facing {
			return fmt.Errorf("inactive turn desires facing %d while current facing is %d", e.DesiredFacing, e.Facing)
		}
		if e.TurnTotal != 0 {
			return fmt.Errorf("inactive turn retains total duration %d", e.TurnTotal)
		}
		return nil
	}
	if !e.Alive() {
		return fmt.Errorf("non-living entity holds %d turn tick(s)", e.TurnRemaining)
	}
	if e.RotationSpeed <= 0 {
		return fmt.Errorf("entity with rotation speed %d holds an active turn", e.RotationSpeed)
	}
	if e.TurnRemaining > 128 {
		return fmt.Errorf("active turn holds impossible remainder %d", e.TurnRemaining)
	}
	if e.TurnTotal == 0 || e.TurnTotal > 128 {
		return fmt.Errorf("active turn holds remainder %d outside total duration %d", e.TurnRemaining, e.TurnTotal)
	}
	if e.DesiredFacing == e.Facing && e.TurnRemaining != 1 {
		return fmt.Errorf("active turn holds equal facings for %d/%d ticks", e.TurnRemaining, e.TurnTotal)
	}
	return nil
}
