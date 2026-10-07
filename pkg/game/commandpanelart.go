package game

import (
	"fmt"

	"againrom/pkg/render/terrain"
	"againrom/pkg/ui"
)

const commandPanelPrefix = graphicsPrefix + "interface/"

const (
	commandPanelHeadsPath      = commandPanelPrefix + "headsr.bmp"
	commandPanelHeadsSeamPath  = commandPanelPrefix + "headsl.bmp"
	commandPanelActivePath     = commandPanelPrefix + "commandbarr.bmp"
	commandPanelActiveSeamPath = commandPanelPrefix + "commandbarl.bmp"
	commandPanelDisabledPath   = commandPanelPrefix + "commandempr.bmp"
	commandPanelSelectedPath   = commandPanelPrefix + "commanddnr.bmp"
)

// LoadCommandPanelArt reads the panel's four bitmaps out of src.
//
// ALL SIX OR NONE. The pictures are one visual unit, so an install missing
// either closing seam is treated the same as an install missing one of the
// four bodies: every failure is an error and none is a partial picture.
func LoadCommandPanelArt(src terrain.EntrySource) (*ui.CommandPanelArt, error) {
	if src == nil {
		return nil, fmt.Errorf("%s: no graphics archive", commandPanelHeadsPath)
	}
	heads, err := readChargenBMP(src, commandPanelHeadsPath)
	if err != nil {
		return nil, err
	}
	if err = chargenSize(heads, 160, 80, commandPanelHeadsPath); err != nil {
		return nil, err
	}
	headsSeam, err := readChargenBMP(src, commandPanelHeadsSeamPath)
	if err != nil {
		return nil, err
	}
	if err = chargenSize(headsSeam, 16, 80, commandPanelHeadsSeamPath); err != nil {
		return nil, err
	}
	active, err := readChargenBMP(src, commandPanelActivePath)
	if err != nil {
		return nil, err
	}
	if err = chargenSize(active, 160, 80, commandPanelActivePath); err != nil {
		return nil, err
	}
	activeSeam, err := readChargenBMP(src, commandPanelActiveSeamPath)
	if err != nil {
		return nil, err
	}
	if err = chargenSize(activeSeam, 16, 80, commandPanelActiveSeamPath); err != nil {
		return nil, err
	}
	disabled, err := readChargenBMP(src, commandPanelDisabledPath)
	if err != nil {
		return nil, err
	}
	if err = chargenSize(disabled, 160, 80, commandPanelDisabledPath); err != nil {
		return nil, err
	}
	selected, err := readChargenBMP(src, commandPanelSelectedPath)
	if err != nil {
		return nil, err
	}
	if err = chargenSize(selected, 160, 80, commandPanelSelectedPath); err != nil {
		return nil, err
	}
	return &ui.CommandPanelArt{
		Heads:      heads,
		HeadsSeam:  keyBlack(headsSeam),
		Active:     active,
		ActiveSeam: keyBlack(activeSeam),
		Disabled:   disabled,
		Selected:   selected,
	}, nil
}
