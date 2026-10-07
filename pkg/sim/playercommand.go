package sim

// PlayerParameterFormation is opcode 0x46 selector 2, the authored Formation
// setting (`AI-FORM-037`). The command's value is a UI setting index, not the
// stored mode byte: the dispatch remaps it before it reaches the same raw store
// trigger instant 7 uses.
const PlayerParameterFormation PlayerParameter = 2

// PlayerParameterRetreat retains the historical native selector3 for AGS
// command compatibility. Corrected SESS-PARAM-017 places original retreat at
// selector1 and autohealing at3; DIV-1302 records this transport difference.
// The value is already the setting's mode:
// 0 is Off, 1 is Low and 2 is High.
const PlayerParameterRetreat PlayerParameter = 3

const (
	RetreatModeOff int32 = iota
	RetreatModeLow
	RetreatModeHigh
)

// applyPlayerParameter is opcode 0x46's selector dispatch. An unknown selector
// is an unknown command arm and changes nothing, on Step's existing
// forward-compatible rule.
func (w *World) applyPlayerParameter(player uint32, selector PlayerParameter, value int32) {
	switch selector {
	case PlayerParameterFormation:
		w.setCommandFormation(player, remapFormationParameter(value))
	case PlayerParameterRetreat:
		w.setRetreatMode(player, value)
	case PlayerParameterAutoHealing:
		w.SetAutoHealing(player, value)
	}
}

// setRetreatMode is parameter 3's two decoded writer calls collapsed over the
// class selector the shipped constructors actually produce (`AI-CLASS-030`).
// For that selector the first writer sets Withdraw uniformly to 0, 10 or 30
// percent of MaxHP and the second writer always clears Wimpy. The player word
// scopes both walks; entities belonging to every other roster slot are left
// byte-for-byte untouched.
//
// Values outside the decoded three modes and players outside this world's
// roster are refused. The multiplication widens before division so this
// build's int32 health domain cannot overflow while deriving the percentage.
func (w *World) setRetreatMode(player uint32, mode int32) {
	if player >= relationSlots {
		return
	}
	percent := int64(0)
	switch mode {
	case RetreatModeOff:
	case RetreatModeLow:
		percent = 10
	case RetreatModeHigh:
		percent = 30
	default:
		return
	}
	for i := range w.entities {
		e := &w.entities[i]
		if e.Owner != player {
			continue
		}
		e.Withdraw = int32(int64(e.MaxHP) * percent / 100)
		e.Wimpy = 0
	}
}

// remapFormationParameter is the command arm's exact value table from
// `AI-FORM-037`: 0 -> 0, 1 -> 2, 2 -> 1, and every other authored value takes
// the arm's default 2. The trigger-instant road does not call this function and
// keeps its raw store.
func remapFormationParameter(value int32) uint8 {
	switch value {
	case 0:
		return 0
	case 1:
		return 2
	case 2:
		return 1
	default:
		return 2
	}
}
