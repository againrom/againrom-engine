package ui

import (
	"image"
	"image/color"
)

// buttonInk is the command-label colour: disabled, at rest, or brighter
// under the pointer. The hover brightening is owner-requested (DIV-1404).
func buttonInk(enabled, hover bool) color.RGBA {
	if !enabled {
		return townShellDisabled
	}
	if hover {
		return color.RGBA{255, 255, 224, 255}
	}
	return color.RGBA{200, 184, 144, 255}
}

// buttonTextOffset moves a held command's caption and number one pixel down.
// TOWN-392 reads a one-pixel vertical displacement on the tavern's pressed
// arm; the shop and the generator share it (DIV-1404).
func buttonTextOffset(down bool) image.Point {
	if down {
		return image.Pt(0, 1)
	}
	return image.Point{}
}
