package main

import (
	"os"
	"path/filepath"

	"againrom/pkg/game"
	"againrom/pkg/ui"
)

// loadPanelProfile gives the editor its side panels' state from the player
// profile the MCP record also lives in. A profile that cannot be resolved,
// or a missing or broken file, leaves the defaults; a save prepares the
// profile directory first and writes through the MCP record's writer.
func loadPanelProfile(e *ui.MapEditor, self, config, root string) {
	profile, err := game.ResolveRuntimeProfile("", self, config, root)
	if err != nil || profile.Directory == "" {
		e.SetPanelProfile(nil, err, nil)
		return
	}
	data, readErr := os.ReadFile(filepath.Join(profile.Directory, ui.PanelStateFile))
	e.SetPanelProfile(data, readErr, func(b []byte) error {
		dir, err := profile.PrepareDirectory(root)
		if err != nil {
			return err
		}
		return ui.WritePanelState(filepath.Join(dir, ui.PanelStateFile), b)
	})
}
