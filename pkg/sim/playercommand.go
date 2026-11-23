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
		w.setCommandFormation(player, FormationModeOfSetting(value))
	case PlayerParameterRetreat:
		w.setRetreatMode(player, value)
	case PlayerParameterAutoHealing:
		w.SetAutoHealing(player, value)
	case PlayerParameterAutoCast:
		if value >= 0 && value <= 0xff {
			w.SetAutoCast(player, AutoCastSetting(value))
		}
	}
}

// setRetreatMode writes one player's thresholds by the rule the profile
// holds. A mode outside the three settings is percent 0, not known.
func (w *World) setRetreatMode(player uint32, mode int32) {
	if player >= relationSlots {
		return
	}
	percent, known := int64(0), true
	switch mode {
	case RetreatModeOff:
	case RetreatModeLow:
		percent = 10
	case RetreatModeHigh:
		percent = 30
	default:
		known = false
	}
	retreatModeRules[w.game.retreatMode](w, player, percent, known)
}

// retreatModeRule is the rule that turns a retreat setting into thresholds.
type retreatModeRule uint8

const (
	retreatModeUniform retreatModeRule = iota
	retreatModeByClass
)

var retreatModeRules = [...]func(w *World, player uint32, percent int64, known bool){
	retreatModeUniform: (*World).uniformRetreatMode,
	retreatModeByClass: (*World).byClassRetreatMode,
}

// uniformRetreatMode is the first game's rule: the percentage applies
// uniformly to Withdraw, Wimpy is 0, and an unknown mode changes nothing
// (AI-CLASS-030).
func (w *World) uniformRetreatMode(player uint32, percent int64, known bool) {
	if !known {
		return
	}
	for i := range w.entities {
		if e := &w.entities[i]; e.Owner == player {
			e.Withdraw, e.Wimpy = int32(int64(e.MaxHP)*percent/100), 0
		}
	}
}

// byClassRetreatMode is the second game's rule: the unit's class scales
// Withdraw and the setting's percentage is Wimpy; an unknown mode clears both
// (R2-ENGINE-1499, R2-ENGINE-1421).
func (w *World) byClassRetreatMode(player uint32, percent int64, _ bool) {
	for i := range w.entities {
		if e := &w.entities[i]; e.Owner == player {
			base := int64(e.MaxHP) * percent
			withdraw := base * [...]int64{0, 2, 3}[w.withdrawalClass(i)]
			e.Withdraw, e.Wimpy = int32(withdraw/100), int32(base/100)
		}
	}
}

// FormationRow is a UI setting index and the stored mode byte it commands.
type FormationRow struct {
	Setting int32
	Mode    uint8
}

// FormationRows is `AI-FORM-037`'s table; FormationFallback answers the rest.
var (
	FormationRows     = [...]FormationRow{{0, 0}, {1, 2}, {2, 1}}
	FormationFallback = FormationRow{Setting: 2, Mode: 2}
)

// FormationModeOfSetting is the mode a setting commands.
func FormationModeOfSetting(setting int32) uint8 {
	for _, r := range FormationRows {
		if r.Setting == setting {
			return r.Mode
		}
	}
	return FormationFallback.Mode
}

// FormationSettingOfMode is the setting whose row holds mode.
func FormationSettingOfMode(mode uint8) int32 {
	for _, r := range FormationRows {
		if r.Mode == mode {
			return r.Setting
		}
	}
	return FormationFallback.Setting
}
