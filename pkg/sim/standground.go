package sim

import "encoding/binary"

// standDown replaces pending intent and retains this engine's loaded cycle.
// Native old-victim retention is conditional on other active-field writers
// leaving it alone (AI-352, AI-354).
//
// The cycle keeps its victim, and the destination it waits behind is the
// member's own cell. That is the shape a writer of a pending order leaves
// behind a loaded cycle (DIV-1563, DIV-1582): the movement pass leaves the
// member standing, and the attack pass drops the victim and the destination
// when the cycle returns to ready, so no new cycle loads. The next evaluation
// takes what stands in reach again (AI-353).
//
// A member holding neither a victim nor a destination already holds 0, and
// nothing is written for it: clearing an order that is not there would still
// mark a loaded route continuation as superseded.
func (w *World) standDown(i int) {
	e := &w.entities[i]
	e.clearPendingAttack()
	if !e.HasAttackTarget && !e.HasTarget {
		return
	}
	if e.HasAttackTarget && e.AttackPhase != AttackReady {
		w.clearOrder(i)
		e.TargetX, e.TargetY, e.HasTarget = e.X, e.Y, true
		return
	}
	e.clearAttack()
	w.clearOrder(i)
}

// commandStandGround is the Stand Ground setter's write to each member beside
// the group order and the post: the pending order is 0, a pending turn is
// cancelled, and the member leaves the engage state a player's attack order
// gave it, so the group decision scores it again (AI-351, AI-352, AI-CMD-054).
func (w *World) commandStandGround(members []int) {
	for _, mi := range members {
		w.clearTurnUnlessCasting(mi)
		w.holdReplacesPendingRow(mi)
		w.standDown(mi)
		if e := &w.entities[mi]; e.ActorState == actorStateEngage {
			e.ActorState = actorStateGuard
		}
		w.syncSavedStandGround(mi)
	}
}

// standScriptedMembers is the script's Stand Ground for one group's members: it
// stands each where it is, ending a walk or pursuit and a patrol or escort, and
// keeps a loaded cycle as Hold does (AI-370, DIV-1582).
func (w *World) standScriptedMembers(members []int) {
	for _, mi := range members {
		w.holdReplacesPendingRow(mi)
		w.standDown(mi)
		e := &w.entities[mi]
		e.clearGroupSpeed()
		e.clearPatrol()
		e.clearEscort()
		if w.savedGroups != nil {
			w.syncSavedStandGround(mi)
			w.ensureSavedOrder(mi).State = uint32(e.ActorState)
		}
	}
}

// syncSavedStandGround mirrors the setter's stores into the SAV order: pending
// 0, stop distance the member's reach, completion word 0 (AI-351, AI-370).
func (w *World) syncSavedStandGround(i int) {
	if w.savedGroups == nil {
		return
	}
	o := w.ensureSavedOrder(i)
	o.Raw[8], o.Raw[0x14] = 0, uint8(w.entities[i].Reach)
	binary.LittleEndian.PutUint32(o.Raw[0x50:], 0)
}

// holdReplacesPendingRow is the setter's pending order 0 for a row a running
// strike still holds back: a release, a pickup walk or a queued manual cast
// that has not been installed. The strike keeps its body (AI-351, AI-352,
// AI-354). An installed cast has written its own progress and action, which
// the setter does not touch, and a pickup completion record is separate from
// the pending order, so both stay (AI-356).
func (w *World) holdReplacesPendingRow(i int) {
	e := &w.entities[i]
	switch p := e.PendingOrder; p.Kind {
	case PendingRelease, PendingPickup:
		e.PendingOrder = PendingOrder{}
	case PendingActorCast, PendingCellCast:
		if !p.RowAdmitted {
			e.PendingOrder = PendingOrder{}
		}
	}
}
