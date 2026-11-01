package ui

// editorRowsOf is the editor popups' wheel step for a raw wheel reading.
func editorRowsOf(remainder *float64, wheel float64) int {
	return editorWheelRows(remainder, Input{WheelY: wheel})
}
