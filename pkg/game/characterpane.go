package game

import (
	"againrom/pkg/render/terrain"
	"againrom/pkg/ui"
)

// TownCharacterPaneArt is the shared character panel's two shipped
// presentations, mode-switched on TownCharacterView.Statistics.
type TownCharacterPaneArt struct {
	Figure ui.TownPane
	Stats  ui.TownPane
}

// LoadTownCharacterPaneArt resolves both of the shared character panel's
// mode-switched presentations, each a 160x242 body and its keyed 16x242
// seam, from the character-pane manifest of description d. Any picture
// failing fails the whole call, so a caller never receives one mode wired
// and the other not, and falls back to the authored fill.
func LoadTownCharacterPaneArt(src terrain.EntrySource, d *Screens) (TownCharacterPaneArt, error) {
	set, err := loadScreen(src, d, "character-pane")
	if err != nil {
		return TownCharacterPaneArt{}, err
	}
	return TownCharacterPaneArt{
		Figure: ui.TownPane{Body: set.Picture("figure-body"), Seam: set.Picture("figure-seam")},
		Stats:  ui.TownPane{Body: set.Picture("stats-body"), Seam: set.Picture("stats-seam")},
	}, nil
}

// LoadCharacterPaneCornerArt resolves the nine corner bitmaps of the six
// corner controls, every one colour-keyed, from the character-pane-corners
// manifest of description d.
//
// NO SIZE IS ASSERTED: a corner's blit size is an operand of the paint
// routine rather than the file's own extent (TOWN-354), so the composer draws
// each into its own decoded destination rectangle and clips there. Any node
// failing fails the whole call, so a caller never receives four corners
// wired and five not.
func LoadCharacterPaneCornerArt(src terrain.EntrySource, d *Screens) (*ui.CharacterPaneCornerArt, error) {
	set, err := loadScreen(src, d, "character-pane-corners")
	if err != nil {
		return nil, err
	}
	return &ui.CharacterPaneCornerArt{
		BackpackOpen:   set.Picture("backpack-open"),
		BackpackClosed: set.Picture("backpack-closed"),
		BookOpened:     set.Picture("book-opened"),
		BookClosed:     set.Picture("book-closed"),
		HumanMode:      set.Picture("human-mode"),
		TextMode:       set.Picture("text-mode"),
		Diskette:       set.Picture("diskette"),
		Ar1:            set.Picture("ar1"),
		Ar2:            set.Picture("ar2"),
	}, nil
}

// LoadCharacterPaneFillerArt resolves the fourth column child's own
// background strip, an opaque body and its keyed seam (TOWN-093), from the
// character-pane-filler manifest of description d.
func LoadCharacterPaneFillerArt(src terrain.EntrySource, d *Screens) (ui.TownPane, error) {
	set, err := loadScreen(src, d, "character-pane-filler")
	if err != nil {
		return ui.TownPane{}, err
	}
	return ui.TownPane{Body: set.Picture("body"), Seam: set.Picture("seam")}, nil
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
	pic, err := LoadCharacterPaneFillerArt(in.Archives.artSource(), ScreensDescription(in.profile()))
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
	a, err := LoadCharacterPaneCornerArt(in.Archives.artSource(), ScreensDescription(in.profile()))
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
	art, err := LoadTownCharacterPaneArt(in.Archives.artSource(), ScreensDescription(in.profile()))
	if err != nil {
		return TownCharacterPaneArt{}
	}
	return p.characterPaneCache.store(art)
}
