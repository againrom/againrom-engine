package game

import (
	"fmt"
	"image"

	"againrom/pkg/formats/spr256"
	"againrom/pkg/render/terrain"
	"againrom/pkg/ui"
)

const (
	tipBackPath    = graphicsPrefix + "interface/t_back.bmp"
	tipFramePath   = graphicsPrefix + "interface/lm.256"
	tipGemPath     = graphicsPrefix + "interface/radiob.256"
	tipGemOffFrame = 4
	tipGemOnFrame  = 5
	checkGemFrame  = 2
)

func LoadTipPanelArt(src terrain.EntrySource) (*ui.TipPanelArt, error) {
	a := &ui.TipPanelArt{Frame: &ui.DialogFrame{}}
	for i, size := range []image.Point{{48, 32}, {32, 32}, {48, 32}, {32, 32}, {32, 32}, {32, 32}, {32, 32}, {48, 32}, {32, 32}} {
		pic, err := loadTipGemFrame(src, tipFramePath, 9+i)
		if err != nil {
			return nil, err
		}
		if err = chargenSize(pic, size.X, size.Y, tipFramePath); err != nil {
			return nil, err
		}
		a.Frame.Pieces[i] = pic
	}
	var err error
	if a.GemOff, err = loadTipGemFrame(src, tipGemPath, tipGemOffFrame); err != nil {
		return nil, err
	}
	if a.GemOn, err = loadTipGemFrame(src, tipGemPath, tipGemOnFrame); err != nil {
		return nil, err
	}
	return a, nil
}

// loadTipGemFrame reads one frame of a .256 sheet as an RGBA, carrying the
// sheet's own structural transparency into the alpha channel — loadShopSprite
// (shopart.go)'s own reader, generalized to a frame index instead of always
// frame 0, since radiob.256 packs six gem pictures in one stream and this
// story needs two of them.
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
	if !sheet.HasPalette {
		return nil, fmt.Errorf("%s: no palette", addr)
	}
	if frame < 0 || frame >= len(sheet.Frames) {
		return nil, fmt.Errorf("%s: frame %d, sheet carries %d", addr, frame, len(sheet.Frames))
	}
	f := sheet.Frames[frame]
	if f.Width <= 0 || f.Height <= 0 || len(f.Pixels) < f.Width*f.Height {
		return nil, fmt.Errorf("%s: frame %d is empty", addr, frame)
	}
	pic := image.NewRGBA(image.Rect(0, 0, f.Width, f.Height))
	for i := 0; i < f.Width*f.Height; i++ {
		p := f.Pixels[i]
		if !p.Opaque || int(p.Index) >= len(sheet.Palette) {
			continue
		}
		e := sheet.Palette[p.Index]
		o := i * 4
		pic.Pix[o], pic.Pix[o+1], pic.Pix[o+2], pic.Pix[o+3] = e.R, e.G, e.B, 0xff
	}
	return pic, nil
}

// tipArt is the front end's one copy, loaded on first use and cached the way
// shopArt already is.
func (f *FrontEnd) tipArt() *ui.TipPanelArt {
	if f == nil {
		return nil
	}
	return f.Presentation.tipArt(&f.InstallResources)
}

func (p *Presentation) tipArt(in *InstallResources) *ui.TipPanelArt {
	if p.tipArtCache.Tried() {
		return p.tipArtCache.Value()
	}
	p.tipArtCache.begin()
	if in.Archives == nil {
		return nil
	}
	a, err := LoadTipPanelArt(in.Archives.Containers)
	if err != nil {
		return nil
	}
	return p.tipArtCache.store(a)
}
