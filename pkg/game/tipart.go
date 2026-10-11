package game

import (
	"fmt"

	"againrom/pkg/render/terrain"
	"againrom/pkg/ui"
)

// LoadTipPanelArt resolves the tip panel's frame pieces and its checkbox gem
// pair from the tip-panel manifest of description d. Any piece failing fails
// the whole call.
func LoadTipPanelArt(src terrain.EntrySource, d *Screens) (*ui.TipPanelArt, error) {
	set, err := loadScreen(src, d, "tip-panel")
	if err != nil {
		return nil, err
	}
	a := &ui.TipPanelArt{Frame: &ui.DialogFrame{}}
	for i := range a.Frame.Pieces {
		a.Frame.Pieces[i] = rgbaOf(set.Picture(fmt.Sprintf("piece/%d", i)))
	}
	a.GemOff, a.GemOn = rgbaOf(set.Picture("gem-off")), rgbaOf(set.Picture("gem-on"))
	return a, nil
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
	a, err := LoadTipPanelArt(in.Archives.artSource(), ScreensDescription(in.profile()))
	if err != nil {
		return nil
	}
	return p.tipArtCache.store(a)
}
