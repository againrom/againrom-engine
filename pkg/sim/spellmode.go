package sim

// AreaMode reports the client-object mode selected by this row's distribution
// and duration columns, or zero for a point row. It exposes areaModeFor's
// already canonical decision to presentation consumers without making them
// restate the column fork.
func (r SpellRule) AreaMode() uint8 {
	if !r.Area {
		return 0
	}
	return areaModeFor(r)
}
