package game

import (
	"fmt"
	"image"

	"againrom/pkg/formats/spr256"
	"againrom/pkg/ui"
)

const (
	tipBackPath   = graphicsPrefix + "interface/t_back.bmp"
	tipFramePath  = graphicsPrefix + "interface/lm.256"
	tipGemPath    = graphicsPrefix + "interface/radiob.256"
	checkGemFrame = 2
)

// loadTipGemFrame reads one frame of a .256 sheet as an RGBA, carrying the
// sheet's own structural transparency into the alpha channel.
func loadTipGemFrame(src entrySource, addr string, frame int) (*image.RGBA, error) {
	if src == nil {
		return nil, fmt.Errorf("%s: no archive", addr)
	}
	raw, err := src.ReadFile(addr)
	if err != nil {
		return nil, err
	}
	sheet, err := spr256.Decode(raw)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", addr, err)
	}
	table, ok := sheet.Table()
	if !ok {
		return nil, fmt.Errorf("%s: no palette", addr)
	}
	if frame < 0 || frame >= len(sheet.Frames) {
		return nil, fmt.Errorf("%s: frame %d, sheet carries %d", addr, frame, len(sheet.Frames))
	}
	f := sheet.Frames[frame]
	if f.Width <= 0 || f.Height <= 0 || len(f.Pixels) < f.Width*f.Height {
		return nil, fmt.Errorf("%s: frame %d is empty", addr, frame)
	}
	return f.RGBA(table), nil
}

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
	if back, err := readInstallBMP(in.Archives.Containers, tipBackPath); err == nil && back.Bounds().Dx() == 160 && back.Bounds().Dy() == 240 {
		art.PortraitBack = back
	}
	art.Minimap, _ = readInstallBMP(in.Archives.Containers, "graphics/interface/crystalr.bmp")
	art.MinimapSeam, _ = readInstallBMP(in.Archives.Containers, "graphics/interface/crystall.bmp")
	return p.dialogArt.store(art)
}
