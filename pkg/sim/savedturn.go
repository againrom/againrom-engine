package sim

import "encoding/binary"

func pendingSavedMotionTurn(m SavedActorMotion) bool {
	return m.Mover[0] != m.Mover[1] || binary.LittleEndian.Uint32(m.Mover[0xa0:]) != 0
}

// Older native saves retain the old admission refusal verbatim. Reconsider
// only those two turn-specific refusals, on an executed body update; LOAD
// itself does not rewrite their saved state or hash. Other issues stay deferred.
func savedTurnQueued(m *SavedActorMotion) bool {
	if m == nil || !m.Current || m.Active || m.Position.FineX != 128 || m.Position.FineY != 128 || !pendingSavedMotionTurn(*m) || m.Mover[10] == 0 {
		return false
	}
	switch m.Issue {
	case "", "centered original turn continuation is not executed", "original pending turn after crossing is not executed":
		return true
	}
	return false
}

// MOVE-TURN-044: the active flag is a DWORD, the counter/estimate are bytes,
// and the estimate uses the pre-step gap. It is never a countdown. The local
// leaf writes current facing, not a route. DIV-1019 owns first-LOAD scheduling
// and the bounded handoff to the native mover; SAV-TURNLOAD-822 proves only
// the serializer's local raw transfer.
func (w *World) advanceSavedTurn(i int, m *SavedActorMotion) bool {
	if !savedTurnQueued(m) {
		return false
	}
	e := &w.entities[i]
	if m.Issue != "" {
		m.Issue = w.motionAdmissionIssue(*m, *e)
		if m.Issue != "" {
			return true
		}
	}
	active := binary.LittleEndian.Uint32(m.Mover[0xa0:]) != 0
	if !active {
		m.Mover[0x9d] = 0
	}
	drawn := e.DrawnFacing() & 0xf0
	desired := m.Mover[1]
	newDesired := e.DesiredFacing != desired
	freshMessage := !e.TurnState.Present || newDesired
	m.Mover[0], m.Mover[0xa4] = turnStep(m.Mover[0], desired, m.Mover[10], active)
	w.markTurnStepped(e.ID)
	m.Mover[0x9d]++
	flag := uint32(0)
	if m.Mover[0] != m.Mover[1] {
		flag = 1
	}
	binary.LittleEndian.PutUint32(m.Mover[0xa0:], flag)
	e.Facing = m.Mover[0]
	e.RotationSpeed = int32(m.Mover[10])
	e.DesiredFacing, e.TurnRemaining = desired, m.Mover[0xa4]
	if freshMessage {
		e.TurnTotal = e.TurnRemaining
		e.TurnState = TurnState{Present: true, Drawn: drawn, DrawTarget: uint8((int(desired)+8)>>4) & 15,
			DrawRemaining: e.TurnTotal}
	}
	e.TurnState.Active, e.TurnState.Counter = flag != 0, m.Mover[0x9d]
	e.advanceDrawnTurn()
	if newDesired {
		e.startAction(w.tick, int64(m.Mover[0xa4]))
	}
	if flag == 0 {
		// A saved standing/attack facing can carry an old route. Only an
		// ordinary move order authorizes handing that route to the body.
		if o := w.savedOrder(e.ID); o != nil && o.RepairStage == 0 && o.Raw[8] == 1 && o.Raw[9] == 0 {
			w.beginSavedRouteContinuation(i, m)
		}
	}
	// The movement loop continues after this return, so even a completed
	// turn cannot also take a position step on this body update.
	return true
}
