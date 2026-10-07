package ui

import (
	"errors"
	"fmt"

	"againrom/pkg/render/terrain"
)

var errNilTileset = errors.New("ui: nil tileset")

// validateGrid rejects a grid the draw path could not index safely. Checking
// here means a malformed map is an error before the window opens rather than
// a panic inside Draw.
func validateGrid(g terrain.Grid) error {
	if g.Width <= 0 || g.Height <= 0 {
		return fmt.Errorf("ui: map has non-positive size %dx%d", g.Width, g.Height)
	}
	if want := g.Width * g.Height; len(g.Tiles) != want {
		return fmt.Errorf("ui: map has %d tile words, want %d for %dx%d", len(g.Tiles), want, g.Width, g.Height)
	}
	return nil
}
