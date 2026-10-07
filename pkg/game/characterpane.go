package game

import (
	"image"

	"againrom/pkg/render/terrain"
	"againrom/pkg/ui"
)

const (
	characterPaneFigureBodyPath = graphicsPrefix + "interface/humanbackr.bmp"
	characterPaneFigureSeamPath = graphicsPrefix + "interface/humanbackl.bmp"
	characterPaneStatsBodyPath  = graphicsPrefix + "interface/textbackr.bmp"
	characterPaneStatsSeamPath  = graphicsPrefix + "interface/textbackl.bmp"
)

// TownCharacterPaneArt is the shared character panel's two shipped
// presentations, mode-switched on TownCharacterView.Statistics.
type TownCharacterPaneArt struct {
	Figure ui.TownPane
	Stats  ui.TownPane
}

// loadCharacterPanePair resolves one 160x242 body and its 16x242 seam.
// Cosmetic on LoadTipPanelArt's own rule: a missing or mis-sized node
// carries its own address in the returned error and every caller falls
// back to the authored fill DrawTownCharacterRegion drew before this story.
func loadCharacterPanePair(src terrain.EntrySource, bodyPath, seamPath string) (ui.TownPane, error) {
	var p ui.TownPane
	body, err := readChargenBMP(src, bodyPath)
	if err != nil {
		return ui.TownPane{}, err
	}
	if err = chargenSize(body, 160, 242, bodyPath); err != nil {
		return ui.TownPane{}, err
	}
	seam, err := readChargenBMP(src, seamPath)
	if err != nil {
		return ui.TownPane{}, err
	}
	if err = chargenSize(seam, 16, 242, seamPath); err != nil {
		return ui.TownPane{}, err
	}
	p.Body, p.Seam = body, keyBlack(seam)
	return p, nil
}

// LoadTownCharacterPaneArt resolves both of the shared character panel's
// mode-switched presentations. Either failing fails the whole call, so a
// caller never receives one mode wired and the other not.
func LoadTownCharacterPaneArt(src terrain.EntrySource) (TownCharacterPaneArt, error) {
	var a TownCharacterPaneArt
	var err error
	if a.Figure, err = loadCharacterPanePair(src, characterPaneFigureBodyPath, characterPaneFigureSeamPath); err != nil {
		return TownCharacterPaneArt{}, err
	}
	if a.Stats, err = loadCharacterPanePair(src, characterPaneStatsBodyPath, characterPaneStatsSeamPath); err != nil {
		return TownCharacterPaneArt{}, err
	}
	return a, nil
}

// The six corner controls' own nine shipped bitmaps.
//
// EVERY PATH IS `TOWN-356`'s (High).
//
// THE SAME ROW CONFIRMS DIV-175's AUTHORED PAIRING.
//
// EVERY ONE IS COLOUR-KEYED. `TOWN-354` reads all six corner blits through
// `obj->vt+0x38`, the primitive `TOWN-282` identifies as colour-keyed — the
// same distinction loadCharacterPanePair's own doc already draws between the
// keyed seam (`vt+0x38`) and the opaque body (`vt+0x18`). keyBlack is applied
// to each on the load-time-keying convention this front end already uses.
const (
	cornerBackpackOpenPath   = graphicsPrefix + "interface/backpackop.bmp"
	cornerBackpackClosedPath = graphicsPrefix + "interface/backpackcl.bmp"
	cornerBookOpenedPath     = graphicsPrefix + "interface/bookopened.bmp"
	cornerBookClosedPath     = graphicsPrefix + "interface/bookclosed.bmp"
	cornerHumanModePath      = graphicsPrefix + "interface/humanmode.bmp"
	cornerTextModePath       = graphicsPrefix + "interface/textmode.bmp"
	cornerDiskettePath       = graphicsPrefix + "interface/diskette.bmp"
	cornerAr1Path            = graphicsPrefix + "interface/ar1.bmp"
	cornerAr2Path            = graphicsPrefix + "interface/ar2.bmp"
)

// LoadCharacterPaneCornerArt resolves the nine corner bitmaps.
//
// NO SIZE IS ASSERTED, deliberately, and that is a departure from
// loadCharacterPanePair's own rule one function up. The body and the seam have
// exactly one size each and a wrong one would mis-tile the pane; a corner's
// own BLIT SIZE is an operand of the paint routine rather than the file's own
// extent (`TOWN-354` gives rect A 32x31 out of a 32-row file and rect B 28x38
// out of a 40-row one), so asserting the file size here would fail on nodes
// that are correct. The composer draws each into its own decoded destination
// rectangle and clips there.
//
// Any node failing fails the whole call, so a caller never receives four
// corners wired and five not.
func LoadCharacterPaneCornerArt(src terrain.EntrySource) (*ui.CharacterPaneCornerArt, error) {
	var a ui.CharacterPaneCornerArt
	for _, bind := range []struct {
		path string
		dst  *image.Image
	}{
		{cornerBackpackOpenPath, &a.BackpackOpen},
		{cornerBackpackClosedPath, &a.BackpackClosed},
		{cornerBookOpenedPath, &a.BookOpened},
		{cornerBookClosedPath, &a.BookClosed},
		{cornerHumanModePath, &a.HumanMode},
		{cornerTextModePath, &a.TextMode},
		{cornerDiskettePath, &a.Diskette},
		{cornerAr1Path, &a.Ar1},
		{cornerAr2Path, &a.Ar2},
	} {
		pic, err := readChargenBMP(src, bind.path)
		if err != nil {
			return nil, err
		}
		*bind.dst = keyBlack(pic)
	}
	return &a, nil
}

const (
	characterPaneFillerBodyPath = graphicsPrefix + "interface/extra1024r.bmp"
	characterPaneFillerSeamPath = graphicsPrefix + "interface/extra1024l.bmp"
)

// LoadCharacterPaneFillerArt resolves the fourth column child's own
// background strip. `TOWN-093`'s own shared block (`L08007`) blits it
// opaque, not colour-keyed — the same distinction loadCharacterPanePair's
// own doc draws for the third child's body, and unlike the corner bitmaps
// (colour-keyed, `TOWN-354`) this is not composited over anything.
func LoadCharacterPaneFillerArt(src terrain.EntrySource) (ui.TownPane, error) {
	body, err := readChargenBMP(src, characterPaneFillerBodyPath)
	if err != nil {
		return ui.TownPane{}, err
	}
	if err = chargenSize(body, 160, 46, characterPaneFillerBodyPath); err != nil {
		return ui.TownPane{}, err
	}
	seam, err := readChargenBMP(src, characterPaneFillerSeamPath)
	if err != nil {
		return ui.TownPane{}, err
	}
	if err = chargenSize(seam, 16, 46, characterPaneFillerSeamPath); err != nil {
		return ui.TownPane{}, err
	}
	return ui.TownPane{Body: body, Seam: keyBlack(seam)}, nil
}

// characterPaneFiller is the front end's one copy of the fourth column
// child's background, on characterPaneCorners' own cache rule: tried once,
// not once a frame, with a flag rather than a nil answer saying so.
func (f *FrontEnd) characterPaneFiller() ui.TownPane {
	if f == nil {
		return ui.TownPane{}
	}
	return f.Presentation.characterPaneFiller(&f.InstallResources)
}

// characterPaneFiller is characterPaneFiller on a presentation and an install.
func (p *Presentation) characterPaneFiller(in *InstallResources) ui.TownPane {
	if p.characterFillerCache.Tried() {
		return p.characterFillerCache.Value()
	}
	p.characterFillerCache.begin()
	if in.Archives == nil || in.Archives.Containers == nil {
		return ui.TownPane{}
	}
	pic, err := LoadCharacterPaneFillerArt(in.Archives.Containers)
	if err != nil {
		return ui.TownPane{}
	}
	return p.characterFillerCache.store(pic)
}

// characterPaneCorners is the front end's one copy of the nine corner
// bitmaps, on characterPanes' own cache rule below: tried once, not once a
// frame, with a flag rather than a nil answer saying so.
func (f *FrontEnd) characterPaneCorners() *ui.CharacterPaneCornerArt {
	if f == nil {
		return nil
	}
	return f.Presentation.characterPaneCorners(&f.InstallResources)
}

// characterPaneCorners is characterPaneCorners on a presentation and an install.
func (p *Presentation) characterPaneCorners(in *InstallResources) *ui.CharacterPaneCornerArt {
	if p.characterCornerCache.Tried() {
		return p.characterCornerCache.Value()
	}
	p.characterCornerCache.begin()
	// A FrontEnd with no container filesystem is the ordinary developer and
	// test case, and the nil is a typed one: Archives.Containers is a *vfs.FS,
	// so it must be tested here rather than after the terrain.EntrySource
	// conversion, where the interface is non-nil and every read faults.
	if in.Archives == nil || in.Archives.Containers == nil {
		return nil
	}
	a, err := LoadCharacterPaneCornerArt(in.Archives.Containers)
	if err != nil {
		return nil
	}
	return p.characterCornerCache.store(a)
}

// characterPanes is the front end's one copy, loaded on first use and
// cached on tipArt's own precedent: tried once, not once a frame, and the
// flag rather than the pane's own zero value says so (a zero
// TownCharacterPaneArt, both Body fields nil, is itself a valid "nothing
// loaded" answer).
func (f *FrontEnd) characterPanes() TownCharacterPaneArt {
	if f == nil {
		return TownCharacterPaneArt{}
	}
	return f.Presentation.characterPanes(&f.InstallResources)
}

// characterPanes is characterPanes on a presentation and an install.
func (p *Presentation) characterPanes(in *InstallResources) TownCharacterPaneArt {
	if p.characterPaneCache.Tried() {
		return p.characterPaneCache.Value()
	}
	p.characterPaneCache.begin()
	// The typed-nil test characterPaneCorners' own doc explains, above.
	if in.Archives == nil || in.Archives.Containers == nil {
		return TownCharacterPaneArt{}
	}
	art, err := LoadTownCharacterPaneArt(in.Archives.Containers)
	if err != nil {
		return TownCharacterPaneArt{}
	}
	return p.characterPaneCache.store(art)
}
