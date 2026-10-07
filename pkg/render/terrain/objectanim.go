package terrain

// ObjectStep is a cell's position in its class's timeline at one counter value:
//
//	step = (counter + col*(row+1)) mod period
//
// reduced EUCLIDEANLY into [0, period), so the answer indexes a timeline of
// that length at any counter and any cell. The counter enters unshifted —
// water's >>2 is water's, and nothing here rate-divides a second time.
//
// It holds no state and reads no clock: two evaluations of one input agree, and
// nothing about a placement, a phase or a previous frame reaches it. That is
// what makes the whole object arm a pure function of (counter, cell) with
// nothing stored per cell.
//
// A non-positive period answers 0 rather than dividing. The callers below gate
// on a non-empty timeline first, so the guard exists for this function's own
// totality — it is exported, and a period is a number a caller computes.
func ObjectStep(col, row int, counter uint32, period int) int {
	if period <= 0 {
		return 0
	}
	// int(counter) is the whole uint32 domain on this project's platforms; the
	// euclidean fold below is what keeps the function total where it is not,
	// and at any negative sum a caller can otherwise produce.
	s := (int(counter) + col*(row+1)) % period
	if s < 0 {
		s += period
	}
	return s
}

func SelectObjectFrame(timeline []int, index, frameCount, col, row int, counter uint32, animate, open bool) int {
	frame := index
	switch {
	case !animate:
		frame = 0
	case open && len(timeline) > 0:
		frame = index + timeline[ObjectStep(col, row, counter, len(timeline))]
	}
	if frame < 0 || frame >= frameCount {
		return 0
	}
	return frame
}
