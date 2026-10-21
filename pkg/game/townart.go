package game

import (
	"sync"

	"againrom/pkg/town"
	"againrom/pkg/ui"
)

// townArtLoads are one town description's install art: its square, its
// tavern, school and shop pages, each read on first use.
type townArtLoads struct {
	square *lazyLoad[*town.Art]
	tavern *lazyLoad[*ui.TownTavernArt]
	school *lazyLoad[*ui.TownSchoolArt]
	shop   *lazyLoad[*ui.ShopScreenArt]
}

// townArtMu guards every Archives' townArt map.
var townArtMu sync.Mutex

// townArtOf is description d's art over the archives, built once per
// description. The archives are shared by every front end of one install,
// so the shop page is read without a front end's book.
func (a *Archives) townArtOf(d *town.Description) *townArtLoads {
	townArtMu.Lock()
	defer townArtMu.Unlock()
	if l := a.townArt[d]; l != nil {
		return l
	}
	src := a.artSource()
	l := &townArtLoads{
		square: deferredRead(func() (*town.Art, error) { return LoadTownSquareArtFor(d, src) }),
		tavern: deferredRead(func() (*ui.TownTavernArt, error) { return LoadTownTavernArt(d, src) }),
		school: deferredRead(func() (*ui.TownSchoolArt, error) { return LoadTownSchoolArt(d, src) }),
		shop: deferredRead(func() (*ui.ShopScreenArt, error) {
			var from entrySource
			if src != nil {
				from = src
			}
			return loadShopArt(d, from), nil
		}),
	}
	if a.townArt == nil {
		a.townArt = map[*town.Description]*townArtLoads{}
	}
	a.townArt[d] = l
	return l
}

// partyTownArt is the art of the party's town's description when it is not
// the starting town's, whose art the install resources hold; nil for the
// starting town's or a screen with no archives.
func (t *townScreen) partyTownArt() *townArtLoads {
	if t == nil || t.in == nil || t.in.Archives == nil {
		return nil
	}
	d := t.townDescription()
	if d == nil || d == TownDescription(t.profile()) {
		return nil
	}
	return t.in.Archives.townArtOf(d)
}

// squareArt, tavernArt, schoolArt and shopArt are the party's town's art.
func (t *townScreen) squareArt() *town.Art {
	if l := t.partyTownArt(); l != nil {
		return onFirstRead(l.square).Value()
	}
	return t.in.TownSquareArt.Value()
}

func (t *townScreen) tavernArt() *ui.TownTavernArt {
	if l := t.partyTownArt(); l != nil {
		return onFirstRead(l.tavern).Value()
	}
	return t.in.TownTavernArt.Value()
}

func (t *townScreen) schoolArt() *ui.TownSchoolArt {
	if l := t.partyTownArt(); l != nil {
		return onFirstRead(l.school).Value()
	}
	return t.in.TownSchoolArt.Value()
}

func (t *townScreen) shopArt() *ui.ShopScreenArt {
	if l := t.partyTownArt(); l != nil {
		shared := onFirstRead(l.shop).Value()
		if shared == nil {
			return nil
		}
		art := *shared
		art.Book = t.in.BottomHUDArt.Value()
		return &art
	}
	return t.art.shopScreen()
}
