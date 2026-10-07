package ui

// SetTextSmoothing turns owner decision method C's mission-HUD overlay
// (DIV-1385) on or off, mirroring App.SetTextSmoothing. Off reproduces
// exactly today's renderer: drawFrame never opens a capture window.
func (v *Viewer) SetTextSmoothing(enabled bool) {
	if v == nil {
		return
	}
	v.textSmoothingEnabled = enabled
}
