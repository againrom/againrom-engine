package game

import "againrom/pkg/ui"

func (f *FrontEnd) gameMenuArt() *ui.MenuPanelArt {
	return f.Presentation.gameMenuArt(&f.InstallResources)
}

// gameMenuArt resolves the menu panel pieces from the install on first use and
// keeps them in the presentation.
func (p *Presentation) gameMenuArt(in *InstallResources) *ui.MenuPanelArt {
	if p.dialogArt.Tried() {
		return p.dialogArt.Value()
	}
	p.dialogArt.begin()
	if in.Archives == nil || in.Archives.Containers == nil {
		return nil
	}
	art := &ui.MenuPanelArt{}
	for i := range art.Pieces {
		pic, err := loadTipGemFrame(in.Archives.Containers, "graphics/interface/lm.256", i)
		if err != nil {
			return nil
		}
		art.Pieces[i] = pic
	}
	art.Portrait, _ = loadTipGemFrame(in.Archives.Containers, "graphics/interface/t_border.256", 0)
	// The dialogue pane's backdrop is the tip panel's own node (DIV-1485). A
	// missing or mis-sized one leaves the pane's black ground.
	if back, err := readChargenBMP(in.Archives.Containers, tipBackPath); err == nil && chargenSize(back, 160, 240, tipBackPath) == nil {
		art.PortraitBack = back
	}
	art.Minimap, _ = readChargenBMP(in.Archives.Containers, "graphics/interface/crystalr.bmp")
	art.MinimapSeam, _ = readChargenBMP(in.Archives.Containers, "graphics/interface/crystall.bmp")
	return p.dialogArt.store(art)
}
