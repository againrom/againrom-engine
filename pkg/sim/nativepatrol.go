package sim

import "encoding/binary"

// Authored patrols keep the actor phase and current two-point ring. Original
// order blocks retain their own dispatch. Both remain owned by the Group.
func (w *World) nativePatrol(i int) bool {
	e := w.entities[i]
	o := w.savedOrder(e.ID)
	if g := w.savedGroupFor(e.ID); g != nil && g.AI[0x20] == orderRoam {
		return false
	}
	return e.ActorState == actorStatePatrol && o != nil && o.Authored && o.State == uint32(actorStatePatrol)
}

func (w *World) syncNativePatrol(i int) {
	if !w.nativePatrol(i) {
		return
	}
	e := w.entities[i]
	o := w.savedOrder(e.ID)
	patrolOrderFields(o, e, w.patrolInterrupted(i))
	if e.HasTarget {
		w.syncSavedDestination(i)
	}
	if e.HasAttackTarget {
		o.Raw[8] = 5
	}
}

// patrolOrderFields writes a patroller's ring, post, cursor, re-anchor latch
// and idle-turn order: the ring's two nodes in ring order, and the cursor at
// the node it is walking to, which is a ring member (SAV-PATROLCURSOR-571,
// AI-PATROL-018).
func patrolOrderFields(o *SavedActorOrder, e Entity, interrupted bool) {
	o.Patrol = []uint16{uint16(e.PatrolHeadX) | uint16(e.PatrolHeadY)<<8, uint16(e.PatrolTailX) | uint16(e.PatrolTailY)<<8}
	binary.LittleEndian.PutUint16(o.Raw[:], uint16(e.PostX)|uint16(e.PostY)<<8)
	binary.LittleEndian.PutUint16(o.Raw[2:], uint16(e.legX())|uint16(e.legY())<<8)
	binary.LittleEndian.PutUint32(o.Raw[4:], 0)
	if !interrupted {
		binary.LittleEndian.PutUint32(o.Raw[4:], 1)
	}
	o.Raw[8] = 0xb
}

// PatrolOrder is the actor order a living patroller holds: state 0x0a and
// the fields patrolOrderFields writes, its current move as destination and
// order 1, and order 5 while it pursues a victim. raw supplies every other
// byte. It reports false for an actor not in patrol.
func PatrolOrder(e Entity, raw [144]byte) (SavedActorOrder, bool) {
	if !e.Alive() || e.ActorState != actorStatePatrol {
		return SavedActorOrder{}, false
	}
	o := SavedActorOrder{Entity: e.ID, State: uint32(actorStatePatrol), Raw: raw}
	patrolOrderFields(&o, e, e.patrolInterrupted())
	if e.HasTarget {
		binary.LittleEndian.PutUint16(o.Raw[10:], uint16(e.TargetX)|uint16(e.TargetY)<<8)
		o.Raw[8] = 1
	}
	if e.HasAttackTarget {
		o.Raw[8] = 5
	}
	return o, true
}
