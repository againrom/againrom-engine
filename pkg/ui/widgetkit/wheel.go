package widgetkit

// WheelStep is the signed step a wheel reading asks of a list, positive towards
// the end: a turn away from the user goes towards the first unit. step is the
// units one notch moves; whole moves by whole notches, a fraction below one
// notch moving nothing, otherwise the wheel moves by its sign alone. pkg/ui's
// list model reads this step for every list row.
func WheelStep(w float64, step int, whole bool) int {
	if whole {
		return -int(w) * step
	}
	switch {
	case w > 0:
		return -step
	case w < 0:
		return step
	}
	return 0
}

// ListWheelLines is the lines a wheel reading moves the launcher window's
// lists: negative towards the first line, zero for no turn. It is the list row
// of the launcher window: one line a notch.
func ListWheelLines(wheel float64) int {
	return WheelStep(wheel, 1, false)
}
