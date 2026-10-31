package ui

import (
	"image"
	"image/color"
	"math"
	"sync"
)

// heightModeSide is a mode button's side in the Height tool's options: room
// for its picture at full size.
const heightModeSide = 34

// raisingHeight reports whether the Height tool's Raise or Lower row is
// chosen: the rows that read the mode and the strength.
func (e *MapEditor) raisingHeight() bool {
	return e.brush.mode == brushHeight && e.brush.sel[brushHeight] < heightSetRow
}

// levelling reports whether a Raise or Lower stroke levels its path.
func (e *MapEditor) levelling() bool {
	return e.raisingHeight() && e.brush.heightMode == HeightModeLevel
}

// brushStrength is the height change of a Raise or Lower stroke.
func (e *MapEditor) brushStrength() int {
	if e.brush.strength == 0 {
		return brushHeightStep
	}
	return e.brush.strength
}

// setBrushStrength is the one way the strength changes: the Strength row's
// minus and plus buttons come here. It holds the strength in 1..31.
func (e *MapEditor) setBrushStrength(s int) {
	e.brush.strength = min(brushMaxStrength, max(1, s))
}

// levelTarget is the height a Level stroke in progress lays: the pressed
// cell's height plus the strength for Raise, minus it for Lower, in 0..127.
func (e *MapEditor) levelTarget() (int, bool) {
	if !e.brush.down || !e.levelling() || len(e.brush.stroke) == 0 {
		return 0, false
	}
	change := e.brushStrength()
	if e.brush.sel[brushHeight] == 1 {
		change = -change
	}
	return min(max(e.cellHeight(e.brush.stroke[0])+change, 0), maxBrushHeight), true
}

// heightModeWord is the word of mode m.
func (e *MapEditor) heightModeWord(m BrushHeightMode) string {
	switch m {
	case HeightModeLevel:
		return e.word("editor.brush.mode.level")
	case HeightModeSmooth:
		return e.word("editor.brush.mode.smooth")
	}
	return e.word("editor.brush.mode.step")
}

// heightModeGridSpec is the Height tool's mode row as popup data: one grid
// row headed by the row's word, a picture button per mode, the chosen one
// pressed.
func (e *MapEditor) heightModeGridSpec() (*popupGrid, bool) {
	if e.brush.mode != brushHeight {
		return nil, false
	}
	g := &popupGrid{Side: heightModeSide, Rows: 1, Cols: int(heightModeCount), RowHeads: []string{e.word("editor.brush.mode")}, HeadW: brushLabelWidth}
	for m := range heightModeCount {
		g.Cells = append(g.Cells, popupGridCell{Icon: heightModeIcon(m), Pressed: m == e.brush.heightMode, Tip: e.heightModeWord(m)})
	}
	return g, true
}

// heightModeGridHeight is the room the mode row takes in the list popup, 0
// for another brush.
func (e *MapEditor) heightModeGridHeight() int {
	if g, ok := e.heightModeGridSpec(); ok {
		return g.height()
	}
	return 0
}

// heightModePress is a press on mode button n: it chooses that mode.
func (e *MapEditor) heightModePress(n int) {
	if n >= 0 && n < int(heightModeCount) {
		e.brush.heightMode = BrushHeightMode(n)
	}
}

// heightModeIcons are the mode buttons' pictures, drawn here: a strip of
// uneven ground with the stroke's part lifted evenly for Step, levelled to
// a flat plateau for Level and rounded into a hill for Smooth. The lifted
// part is green over the brown ground.
var heightModeIcons = sync.OnceValue(func() []*image.RGBA {
	ground, lifted := color.RGBA{150, 108, 60, 255}, color.RGBA{88, 176, 72, 255}
	uneven := [24]int{12, 12, 11, 11, 12, 13, 13, 12, 11, 11, 12, 13, 13, 12, 12, 11, 11, 12, 13, 13, 12, 12, 11, 11}
	var out []*image.RGBA
	for m := range heightModeCount {
		img := image.NewRGBA(image.Rect(0, 0, 24, 16))
		for x, base := range uneven {
			top := base
			if x >= 5 && x <= 18 {
				switch m {
				case HeightModeStep:
					top = base - 6
				case HeightModeLevel:
					top = 5
				default:
					c := math.Cos(math.Pi * (float64(x) - 11.5) / 15)
					top = base - int(math.Round(8*c*c))
				}
			}
			for y := top; y < 16; y++ {
				ink := ground
				if y < base {
					ink = lifted
				}
				img.SetRGBA(x, y, ink)
			}
		}
		out = append(out, img)
	}
	return out
})

// heightModeIcon is mode m's picture.
func heightModeIcon(m BrushHeightMode) *image.RGBA {
	if pics := heightModeIcons(); int(m) < len(pics) {
		return pics[m]
	}
	return nil
}
