package game

import (
	"againrom/pkg/base"
	"againrom/pkg/render/menu"
	"againrom/pkg/render/terrain"
)

// LoadMainMenuArt binds the edition's main-menu manifest to its menu layout.
func LoadMainMenuArt(src terrain.EntrySource, p base.Profile) (*menu.Assets, error) {
	set, err := loadScreen(src, ScreensDescription(p), "main-menu")
	if err != nil {
		return nil, err
	}
	return filesOf(p.GameOf()).loadMenu(set)
}
