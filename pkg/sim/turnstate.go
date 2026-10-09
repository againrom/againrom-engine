package sim

// TurnState separates the mover flag and counter from the client message run.
// Client fields never choose a simulation step or an action gate.
type TurnState struct {
	Present, Active, DrawComplete             bool
	Counter, Drawn, DrawTarget, DrawRemaining uint8
}

// MOVE-106, DIV-1421
func (e *Entity) refreshHumanTurnRate() {
	if e.Humanoid && e.ActorLoad.Source.Class == 0 {
		e.RotationSpeed = int32(uint8(aloneSpeed(*e)))
		if e.RotationSpeed == 0 && e.Turning() {
			e.Facing = e.DesiredFacing
			e.TurnState.Active = false
			e.clearTurn()
		}
	}
}

func (e Entity) DrawingTurn() bool {
	if e.TurnState.Present {
		return e.TurnState.DrawRemaining > 0 || e.TurnState.DrawComplete
	}
	return e.Turning()
}

func (e *Entity) replaceDrawnTurn() {
	e.TurnState.DrawRemaining, e.TurnState.DrawComplete = 0, false
	if !e.TurnState.Active && !e.Turning() {
		e.TurnState = TurnState{}
	}
}

// MOVE-105
func turnStep(current, desired, rate uint8, active bool) (uint8, uint8) {
	arc := facingArc(current, desired)
	if arc == 0 && active {
		return desired, 0
	}
	if rate == 0 || !active && arc <= 32 {
		return desired, 1
	}
	step := uint8(min(arc, int32(rate)))
	if uint8(desired-current) <= 128 {
		current += step
	} else {
		current -= step
	}
	return current, uint8((arc + int32(rate) - 1) / int32(rate))
}

// ANIM-136
func (e *Entity) advanceDrawnTurn() {
	s := &e.TurnState
	s.DrawComplete = false
	if s.DrawRemaining == 0 {
		return
	}
	d := int(s.DrawTarget)*16 - int(s.Drawn)
	if d <= -128 {
		d += 256
	} else if d > 128 {
		d -= 256
	}
	s.Drawn = uint8(int(s.Drawn) + d/int(s.DrawRemaining))
	s.DrawRemaining--
	s.DrawComplete = s.DrawRemaining == 0
}
