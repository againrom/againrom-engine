package sim

import "encoding/binary"

// walkClaim is the part of an actor's Mover the walk keeps between passes
// for its arrival and for path (b) of the order 4 arm: the target of the
// static search that refused it (+0x74), the cell it claimed (+0x76) and the
// arrival flag (+0x90) (AI-658, AI-682). A static search with no route claims
// the actor's own cell and raises the flag; path (b) of the next order 4 pass
// latches a member standing on its claim with the flag raised. Only a refused
// walk of a member of a Swarm (order 2), Move, Swarm 2 or Roam Group writes
// it here; the stepper's claims are not carried.
type walkClaim struct {
	Target, Claim uint16
	Arrived       bool
}

// walkClaimOf returns actor id's claim from the Mover an original-SAV import
// holds, else from the record the World keeps.
func (w *World) walkClaimOf(id EntityID) walkClaim {
	if m := w.motionFor(id); m != nil {
		return readWalkClaim(&m.Mover)
	}
	if r := w.moverFor(id); r != nil {
		return r.Claim
	}
	return walkClaim{}
}

// setWalkClaim stores c where walkClaimOf reads it.
func (w *World) setWalkClaim(id EntityID, c walkClaim) {
	if m := w.motionFor(id); m != nil {
		putWalkClaim(&m.Mover, c)
		return
	}
	if r := w.moverFor(id); r != nil {
		next := *r
		next.Claim = c
		w.setMover(next)
	}
}

func putWalkClaim(m *[180]byte, c walkClaim) {
	binary.LittleEndian.PutUint16(m[0x74:], c.Target)
	binary.LittleEndian.PutUint16(m[0x76:], c.Claim)
	arrived := uint32(0)
	if c.Arrived {
		arrived = 1
	}
	binary.LittleEndian.PutUint32(m[0x90:], arrived)
}

func readWalkClaim(m *[180]byte) walkClaim {
	return walkClaim{
		Target:  binary.LittleEndian.Uint16(m[0x74:]),
		Claim:   binary.LittleEndian.Uint16(m[0x76:]),
		Arrived: binary.LittleEndian.Uint32(m[0x90:]) != 0,
	}
}

func packedCell(x, y int32) uint16 { return uint16(x) | uint16(y)<<8 }

// claimsWalk reports whether actor i's walk keeps a claim: a first-game
// member, not withdrawing and holding no pending order, of a Group at order
// 2, 4, 5 or 0x11, the orders whose arms re-issue a walk to their members
// (AI-662, AI-685). A walk to a sack is the pick-up's, which this build keeps
// inside the Group the original leaves (DIV-1520).
func (w *World) claimsWalk(i int) bool {
	e := w.entities[i]
	if w.SecondGame() || w.groupDialectNative() || e.ActorState == actorStateRetreat || e.PendingOrder.Kind != PendingNone || w.savedOrder(e.ID) == nil {
		return false
	}
	g := w.savedGroupFor(e.ID)
	if g == nil {
		return false
	}
	switch g.AI[0x20] {
	case orderSwarm, orderMove, orderSwarm2, orderRoam:
		return true
	}
	return false
}

// claimRefusedCell is the static search's refusal store: the walk claims the
// actor's own cell for the target it was refused, and since it stands on
// that cell the arrival flag is raised (AI-658, AI-682).
func (w *World) claimRefusedCell(i int) {
	e := w.entities[i]
	o := w.savedOrder(e.ID)
	w.setWalkClaim(e.ID, walkClaim{Target: binary.LittleEndian.Uint16(o.Raw[10:]), Claim: packedCell(e.X, e.Y), Arrived: true})
}

// heldClaim reports whether actor i's walk ends at once: its target is the
// one its claim was refused for and it stands on its claim, so no search
// runs and the arrival flag stays raised (AI-685).
func (w *World) heldClaim(i int) bool {
	if !w.claimsWalk(i) {
		return false
	}
	e := w.entities[i]
	c := w.walkClaimOf(e.ID)
	return c.Arrived && e.Transit == 0 && c.Claim == packedCell(e.X, e.Y) && c.Target == packedCell(e.TargetX, e.TargetY)
}

// clearClaimArrival lowers actor i's arrival flag as its static search
// begins (AI-658).
func (w *World) clearClaimArrival(i int) {
	if !w.claimsWalk(i) {
		return
	}
	id := w.entities[i].ID
	if c := w.walkClaimOf(id); c.Arrived {
		c.Arrived = false
		w.setWalkClaim(id, c)
	}
}

// claimLatch is path (b) of the order 4 arm (AI-658, AI-682): a member not
// on its order's cell, standing on its claim with the arrival flag raised,
// is stood down and latched, and the flag lowered. It reports the latch.
func (w *World) claimLatch(i int, o *SavedActorOrder) bool {
	e := w.entities[i]
	if !w.claimsWalk(i) || e.Transit != 0 || binary.LittleEndian.Uint32(o.Raw[0x50:]) != 0 {
		return false
	}
	c := w.walkClaimOf(e.ID)
	if !c.Arrived || c.Claim != packedCell(e.X, e.Y) {
		return false
	}
	w.clearOrder(i)
	w.standDownRefusedWalk(i)
	binary.LittleEndian.PutUint32(o.Raw[0x50:], 1)
	c.Arrived = false
	w.setWalkClaim(e.ID, c)
	return true
}
