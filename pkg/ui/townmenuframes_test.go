package ui

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"againrom/internal/witness"
)

// The town menu's panel of each game's placement. With AGAINROM_TOWN_MENU_FRAMES
// naming a folder the frames are written there, else to a temporary folder; with
// AGAINROM_TOWN_MENU_BEFORE naming a folder of earlier frames, each must equal
// its earlier one byte for byte.
func TestTownMenuPanelFramesStayAsTheyWere(t *testing.T) {
	dir, before := os.Getenv("AGAINROM_TOWN_MENU_FRAMES"), os.Getenv("AGAINROM_TOWN_MENU_BEFORE")
	if dir == "" {
		dir = t.TempDir()
	}
	f := openTownMenu(t)
	for name, profile := range map[string]GameMenuProfile{"first": {}, "second": {NativeMissionWindows: true}} {
		pic := composeGameMenuPanel(gameMenuTestFont(), f.menuPanelSurface(), f.menuRows(), f.menuList, nil, gameMenuPointer{}, profile)
		b := witness.Bytes(t, pic)
		file := "town-menu-" + name + ".png"
		if err := os.WriteFile(filepath.Join(dir, file), b, 0o644); err != nil {
			t.Fatal(err)
		}
		if before != "" {
			if earlier, err := os.ReadFile(filepath.Join(before, file)); err != nil || !bytes.Equal(earlier, b) {
				t.Error(file, "differs from its earlier frame")
			}
		}
	}
}
