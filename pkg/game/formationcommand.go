package game

import "againrom/pkg/sim"

// cycleFormation is the ui.MapFormation installed on a live mission viewer. It
// reads the existing canonical byte only to choose the next authored UI setting
// and appends opcode 0x46 selector 2 to the same pending command stream every
// order uses. It does not mutate the world, mark an entity commanded or step a
// stopped world.
func (mw *mapWorld) cycleFormation() {
	player := uint32(sim.SelfSlot)
	mode, found := mw.world.CommandFormationMode(player)
	if !found {
		return
	}
	value := nextFormationParameter(mode)
	// A stopped world deliberately keeps ordinary commands pending. Preserve
	// each distinct press in that state: the most recent queued formation
	// command is the state the next press follows, even though the canonical
	// byte must remain untouched until Step drains the queue.
	for _, cmd := range mw.pending {
		if cmd.Kind == sim.KindPlayerParameter && cmd.Player == player &&
			sim.PlayerParameter(cmd.X) == sim.PlayerParameterFormation {
			value = nextQueuedFormationParameter(cmd.Y)
		}
	}
	mw.enqueueFormation(value)
}

func (mw *mapWorld) enqueueFormation(value int32) {
	mw.pending = append(mw.pending, sim.SetPlayerParameter(sim.SelfSlot, sim.PlayerParameterFormation, value))
	mw.rememberApplicationFormation(value)
}

// nextQueuedFormationParameter advances one already-authored label. The
// default arm mirrors what the simulation command will observe: any value
// outside 0..2 stores Auto (mode 2), whose successor is authored On (2).
func nextQueuedFormationParameter(value int32) int32 {
	switch value {
	case 0: // off -> auto
		return 1
	case 1: // auto -> on
		return 2
	case 2: // on -> off
		return 0
	default: // command default -> auto -> on
		return 2
	}
}

// nextFormationParameter translates the stored byte back to the three authored
// labels and advances off -> auto -> on -> off. `AI-FORM-037`'s command arm
// maps those setting indices 0,1,2 to stored modes 0,2,1 respectively. Any
// trigger-authored nonzero value other than 2 has the same "on" behaviour as 1
// under `MOVE-GATE-035`, so it cycles to off on the next player press.
func nextFormationParameter(mode uint8) int32 {
	switch mode {
	case 0: // off -> auto
		return 1
	case 2: // auto -> on
		return 2
	default: // on (all other nonzero modes) -> off
		return 0
	}
}
