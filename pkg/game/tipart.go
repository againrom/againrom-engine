package game

import (
	"fmt"

	"againrom/pkg/render/terrain"
	"againrom/pkg/ui"
)

// LoadTipPanelArt resolves the tip panel's frame pieces and its checkbox gem
// pair from manifest of description d, the one the tip table names. Any
// piece failing fails the whole call.
func LoadTipPanelArt(src terrain.EntrySource, d *Screens, manifest string) (*ui.TipPanelArt, error) {
	set, err := loadScreen(src, d, manifest)
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
	table := TipTable(in.profile().Edition())
	if table == nil {
		return nil
	}
	a, err := LoadTipPanelArt(in.Archives.artSource(), ScreensDescription(in.profile()), table.Art)
	if err != nil {
		return nil
	}
	return p.tipArtCache.store(a)
}
