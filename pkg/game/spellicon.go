package game

import (
	"image"

	"againrom/pkg/data"
	"againrom/pkg/formats/bmp"
)

// The spell book's icons: one strip, cut by a fixed grid, at the position the
// engine's own slot table gives each spell (`MAGIC-ICON-024`).
//
// AN ICON IS CUT, NOT LOADED. There is no per-spell file and no indexed cell in
// the archive: `graphics\interface\SpellBook.bmp` is ONE pre-composited 480x85
// bitmap that the original blits whole, and what is per-spell is where in it a
// spell's picture sits. So this file reads one node per mission and takes 36x36
// squares out of it.
//
// FOUR OF THE TWENTY-EIGHT SHIPPED SPELLS HAVE NO ICON AT ALL — ids 11, 17, 27
// and 28 are in no slot, so 24 cells serve 28 spells. That is a fact about the
// shipped game and not a gap here; those four reach the drawing tier with no
// picture and it draws them the way it drew every spell before this story.

// spellAtlas is the decoded strip, or nil for an install that does not carry it
// or carries one this decoder refuses.
//
// IT IS TRIED ONCE PER MISSION AND THE FAILURE IS CACHED WITH THE SUCCESS —
// classPortrait's own convention (portrait.go), for the same reason: an install
// missing the node would otherwise be re-read on every frame that recomposes a
// book.
func loadSpellAtlas(src entrySource) *bmp.Image {
	if src == nil {
		return nil
	}
	b, err := src.ReadFile(graphicsPrefix + data.SpellIconAtlasPath)
	if err != nil {
		return nil
	}
	im, err := bmp.Decode(b)
	if err != nil {
		return nil
	}
	// THE EXTENT IS CHECKED AND NOT ASSUMED. The grid below was measured
	// against a 480x85 strip and its bottom-right cell ends two pixels inside
	// it; cutting that grid out of a differently sized picture would hand the
	// window somebody else's pixels rather than fail. An install whose atlas is
	// not the measured one draws no icons, which is what an unrecognised file
	// should cost.
	if im.Width != data.SpellIconAtlasW || im.Height != data.SpellIconAtlasH {
		return nil
	}
	return im
}

func (mw *mapWorld) spellAtlas() *bmp.Image {
	if mw.spellAtlasTried {
		return mw.spellAtlasImg
	}
	mw.spellAtlasTried = true
	mw.spellAtlasImg = loadSpellAtlas(mw.archive())
	return mw.spellAtlasImg
}

// cutSpellIcon is one slot's 36x36 cell as the drawing tier's own surface,
// FULLY OPAQUE — the bitmap decoder's own reading of a format with no fourth
// channel, applied to a sub-rectangle instead of a whole picture.
//
// The atlas is pre-composited: a cell already carries whatever ground the
// original's panel shows behind that icon, so there is nothing here to key out
// and nothing that could be keyed out without inventing a transparent colour.
func cutSpellIcon(im *bmp.Image, slot int) *image.RGBA {
	x0, y0, ok := data.SpellIconCell(slot)
	if !ok || im == nil {
		return nil
	}
	n := data.SpellIconSide
	return im.SubRGBA(image.Rect(x0, y0, x0+n, y0+n))
}

// spellIcon is the picture for one spell id, or nil for a spell that has none.
//
// THE CACHE IS KEYED BY SPELL ID AND HOLDS ITS MISSES, portrait.go's own
// convention: a spell with no slot, and every spell at all when the atlas did
// not read, answers nil once and is not looked at again.
func (mw *mapWorld) spellIcon(id uint16) *image.RGBA {
	if pic, tried := mw.spellIcons[id]; tried {
		return pic
	}
	var pic *image.RGBA
	if slot, ok := data.SpellIconSlot(int32(id)); ok {
		pic = cutSpellIcon(mw.spellAtlas(), slot)
	}
	if mw.spellIcons == nil {
		mw.spellIcons = make(map[uint16]*image.RGBA)
	}
	mw.spellIcons[id] = pic
	return pic
}

// shopSpellIcon is the town shop's reuse of the mission spellbook atlas.
// Both views read the same installed strip and the same id-to-slot table;
// only their caches differ because a townScreen has no mapWorld.
func (t *townScreen) shopSpellIcon(id uint16) *image.RGBA {
	if t == nil {
		return nil
	}
	if pic, tried := t.shopSpellIcons[id]; tried {
		return pic
	}
	if !t.shopSpellAtlasTried {
		t.shopSpellAtlasTried = true
		if t.sess != nil && t.in.Archives != nil {
			t.shopSpellAtlasImg = loadSpellAtlas(t.in.Archives.Containers)
		}
	}
	var pic *image.RGBA
	if slot, ok := data.SpellIconSlot(int32(id)); ok {
		pic = cutSpellIcon(t.shopSpellAtlasImg, slot)
	}
	if t.shopSpellIcons == nil {
		t.shopSpellIcons = make(map[uint16]*image.RGBA)
	}
	t.shopSpellIcons[id] = pic
	return pic
}
