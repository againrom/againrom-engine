package game

import (
	"againrom/pkg/render/terrain"
	"againrom/pkg/ui"
	"fmt"
)

// LoadMenuPanelArt binds the large window pieces and optional panel pictures.
func LoadMenuPanelArt(src terrain.EntrySource, d *Screens) (*ui.MenuPanelArt, error) {
	set, err := loadScreen(src, d, "menu-panel")
	if err != nil {
		return nil, err
	}
	art := &ui.MenuPanelArt{}
	for i := range art.Pieces {
		art.Pieces[i] = rgbaOf(set.Picture(fmt.Sprintf("piece/%d", i)))
	}
	art.Portrait = rgbaOf(set.Picture("portrait"))
	art.PortraitBack = rgbaOf(set.Picture("portrait-back"))
	minimap, _ := loadScreen(src, d, "mission-minimap", "background", "seam")
	art.Minimap = rgbaOf(minimap.Picture("background"))
	art.MinimapSeam = rgbaOf(minimap.Picture("seam"))
	return art, nil
}

func (f *FrontEnd) gameMenuArt() *ui.MenuPanelArt {
	return f.Presentation.gameMenuArt(&f.InstallResources)
}

// gameMenuArt resolves the menu panel pieces once and keeps the typed binding.
func (p *Presentation) gameMenuArt(in *InstallResources) *ui.MenuPanelArt {
	if p.dialogArt.Tried() {
		return p.dialogArt.Value()
	}
	p.dialogArt.begin()
	if in.Archives == nil || in.Archives.Containers == nil {
		return nil
	}
	art, err := LoadMenuPanelArt(in.Archives.artSource(), ScreensDescription(in.profile()))
	if err != nil {
		return nil
	}
	return p.dialogArt.store(art)
}
