package game

import (
	"fmt"

	"againrom/pkg/render/terrain"
	"againrom/pkg/ui"
)

// LoadCommandPanelArt reads the panel's four bodies and two closing seams
// from the command-panel manifest of description d.
//
// ALL SIX OR NONE. The pictures are one visual unit, so an install missing
// either closing seam is treated the same as an install missing one of the
// four bodies: every failure is an error and none is a partial picture.
func LoadCommandPanelArt(src terrain.EntrySource, d *Screens) (*ui.CommandPanelArt, error) {
	if src == nil {
		return nil, fmt.Errorf("command panel: no graphics archive")
	}
	set, err := loadScreen(src, d, "command-panel")
	if err != nil {
		return nil, err
	}
	return &ui.CommandPanelArt{
		Heads:      set.Picture("heads"),
		HeadsSeam:  set.Picture("heads-seam"),
		Active:     set.Picture("active"),
		ActiveSeam: set.Picture("active-seam"),
		Disabled:   set.Picture("disabled"),
		Selected:   set.Picture("selected"),
	}, nil
}
