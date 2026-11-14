package ui

import "math"

// scaleStage is one way a final frame is scaled. The window executor
// (drawFinalFrame) and the CPU executor (referenceFinalFrame) each run the
// stage that their column of stageOf names.
type scaleStage uint8

const (
	scaleNone scaleStage = iota
	scaleNearest
	scaleCatmull
	scaleSharp
)

// scaleChoice is the stage each executor runs for one filter and scale.
type scaleChoice struct {
	window, cpu scaleStage
}

// stageOf is the one filter rule of the final frame. The window and the CPU
// reference both read it. The two columns differ only under a sharp or pixel
// filter: at a scale of 0 or less the window runs method B and the CPU
// nothing; at a whole scale (sharp from 1, pixel from 2) the window runs
// method B and the CPU nearest.
func stageOf(filter frameFilter, s float64) scaleChoice {
	if filter == filterSmooth {
		switch {
		case s <= 0:
			return scaleChoice{scaleNone, scaleNone}
		case s == 1:
			return scaleChoice{scaleNearest, scaleNearest}
		}
		return scaleChoice{scaleCatmull, scaleCatmull}
	}
	switch {
	case s <= 0:
		return scaleChoice{scaleSharp, scaleNone}
	case filter == filterPixel && s < 2:
		return scaleChoice{scaleNearest, scaleNearest}
	case s == math.Trunc(s):
		return scaleChoice{scaleSharp, scaleNearest}
	}
	return scaleChoice{scaleSharp, scaleSharp}
}
