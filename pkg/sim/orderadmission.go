package sim

// primaryArm is the executor arm that owns an actor, numbered as the
// original's primary byte: zero admits a prepared action, and strike, cast,
// step and hold each own the actor (R2-ENGINE-542, R2-ENGINE-553).
type primaryArm uint8

const (
	armPrepared primaryArm = iota
	armStrike
	armCast
	armStep
	armHold
	armClear
	armKeep
)

// primaryCell is the executor's table over +9 = 1..255: 1..4 name their
// arms, 5..254 the clear arm and 255 the arm that keeps +9 (R2-ENGINE-553).
func primaryCell(b uint8) primaryArm {
	switch {
	case b <= uint8(armHold):
		return primaryArm(b)
	case b == 255:
		return armKeep
	}
	return armClear
}

// executor names a producer that asks the admission before it acts.
type executor uint8

const (
	executeCast executor = iota
	executeMove
	executeStrike
	executePending
)

// admission is the one answer to which arm owns actor i on this call. Cast,
// movement, strike and a pending order each read it instead of testing the
// carriers themselves, so a row added here holds all four at once.
type admission struct {
	arm      primaryArm
	cell     primaryArm
	progress uint8
	hold     bool
	motion   bool
	transit  bool
}

func (w *World) admission(i int) admission {
	e := w.entities[i]
	a := admission{hold: w.stoneCursed(i), motion: w.motionActive(e.ID), transit: e.Transit != 0}
	switch {
	case a.hold:
		a.arm = armHold
	case a.transit || a.motion:
		a.arm = armStep
	case w.castHolds(i, waitCastAction):
		a.arm = armCast
	case e.AttackPhase != AttackReady:
		a.arm = armStrike
	}
	// A saved byte no typed carrier holds takes the profile's table cell; an
	// imported hold byte without its effect clears as the hold arm does.
	if o := w.savedOrder(e.ID); w.game.orders.primaryCells && o != nil && !a.hold && !e.Retreat.Known {
		switch primaryCell(o.Raw[9]) {
		case armHold, armClear:
			a.cell = armClear
		case armKeep:
			a.cell = armKeep
		}
	}
	a.progress = w.admittedProgress(i, a)
	return a
}

// admittedProgress is the order's progress byte a pending order waits on:
// a retreat's own continuation, the saved order's byte with a cycle the World
// loaded or finished since it was written, else the typed carriers.
func (w *World) admittedProgress(i int, a admission) uint8 {
	e := w.entities[i]
	if e.Retreat.Known {
		return e.Retreat.Progress
	}
	if o := w.savedOrder(e.ID); o != nil {
		if o.Raw[9] == 1 && e.AttackPhase == AttackReady && e.AttackCountdown == 0 {
			return 0
		}
		// A cycle the World loaded after the order's progress byte was last
		// written is in progress until it is ready again.
		if o.Raw[9] == 0 && e.cycleLoaded() {
			return 1
		}
		return o.Raw[9]
	}
	if a.transit || a.motion {
		return 3
	}
	if e.AttackPhase != AttackReady || e.AttackCountdown != 0 {
		return 1
	}
	return 0
}

// runPrimaryCell runs a raw cell on the executor call: the clear arm stores
// +9 = 0 and owns the call, so nothing else acts on it. The keep arm owns no
// call: its byte stays and only a pending order waits on it (waits).
func (w *World) runPrimaryCell(i int) bool {
	o := w.savedOrder(w.entities[i].ID)
	if !w.game.orders.primaryCells || o == nil || o.Raw[9] < uint8(armHold) {
		return false
	}
	if w.admission(i).cell != armClear {
		return false
	}
	o.Raw[9] = 0
	return true
}

// waits reports whether an arm other than k's own holds executor k on this
// call. The clear cell holds every executor; the keep cell holds only a
// pending order. The imported step is the move executor's own arm; a pending
// order already admitted to its row waits for any step.
func (a admission) waits(k executor) bool {
	switch {
	case a.cell == armClear:
		return true
	case a.cell == armKeep && k == executePending:
		return true
	}
	switch k {
	case executeCast:
		return a.hold || a.motion
	case executePending:
		return a.hold || a.motion || a.transit
	}
	return a.hold
}
