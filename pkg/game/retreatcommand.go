package game

import "againrom/pkg/sim"

// cycleRetreat advances the process preference Off -> Low -> High -> Off,
// persists the new label and appends opcode 0x46 selector 3 to the live world's
// ordinary pending-command stream. It does not mutate entity thresholds or
// step a stopped world; sim.Step remains the only command executor.
func (p *PersistenceContext) cycleRetreat(mw *mapWorld) {
	if p == nil || mw == nil {
		return
	}
	p.wimpyMode = nextWimpyMode(p.wimpyMode)
	mw.enqueueRetreat(p.wimpyMode)
	_ = p.Options.SetWimpyMode(p.wimpyMode)
}

// nextWimpyMode is total over a damaged or hand-built cached value. The safe
// fallback treats it as Off and advances to Low, matching OptionsStore's own
// read fallback and never selecting High from malformed state.
func nextWimpyMode(mode int) int {
	switch mode {
	case wimpyModeOff:
		return wimpyModeLow
	case wimpyModeLow:
		return wimpyModeHigh
	case wimpyModeHigh:
		return wimpyModeOff
	default:
		return wimpyModeLow
	}
}

func (mw *mapWorld) enqueueRetreat(mode int) {
	mw.pending = append(mw.pending, sim.SetPlayerParameter(sim.SelfSlot, sim.PlayerParameterRetreat, int32(mode)))
}
