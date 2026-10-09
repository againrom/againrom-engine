package game

import (
	"image"
	"image/draw"

	"againrom/pkg/data"
	"againrom/pkg/formats/bmp"
	"againrom/pkg/formats/spr256"
	"againrom/pkg/mapload"
	"againrom/pkg/render/terrain"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

func memberHasHorse(m mapload.PartyMember) bool {
	return m.Body == "" && data.FigureHasHorse(m.Class)
}

func memberFigureID(m mapload.PartyMember) figureID {
	dir, face := memberFigure(m)
	return figureID{Dir: dir, Face: face, Horse: memberHasHorse(m), Hero: m.PlayerCharacter && !m.Hired()}
}

// HERO-FIGURE-144: a warrior hero has a sex-selected background behind the
// face and equipment. Its own palette is used without owner or sky tinting.
// Horse and hero background do not participate in either equipment mask.
func addFigureBackground(src entrySource, figure *image.RGBA, fig figureID) *image.RGBA {
	if figure == nil || src == nil {
		return figure
	}
	if fig.Hero && !fig.Dir.Mage() {
		path := "interface/heroback/backm.256"
		if fig.Dir.Female() {
			path = "interface/heroback/backf.256"
		}
		sheets := sheetCache{src: src, decoded: map[string]*spr256.Sprite{}, converted: map[string][]*terrain.StaticFrame{}}
		if back := figureRGBA(&sheets, path); back != nil {
			pic := image.NewRGBA(figure.Bounds())
			draw.Draw(pic, pic.Bounds(), back, back.Bounds().Min, draw.Src)
			draw.Draw(pic, pic.Bounds(), figure, figure.Bounds().Min, draw.Over)
			figure = pic
		}
	}
	if fig.Horse && !fig.Hero {
		figure = addFigureHorse(src, figure)
	}
	return figure
}

func composeInventoryPortrait(src entrySource, id uint32, eq data.Equipment, fig figureID) (ui.InventorySubject, []string) {
	return composeInventoryPortraitLayered(src, id, eq, nil, fig)
}

func composeInventoryPortraitLayered(src entrySource, id uint32, eq data.Equipment, layers []figureLayer, fig figureID) (ui.InventorySubject, []string) {
	out, unread := composeInventorySubjectLayered(src, id, eq, layers, fig.Dir, fig.Face)
	out.Figure = addFigureBackground(src, out.Figure, fig)
	return out, unread
}

// The horse is a background layer on the same 160x240 canvas. It carries no
// equipment slot; the rider's existing slot and hover masks stay unchanged.
func addFigureHorse(src entrySource, rider *image.RGBA) *image.RGBA {
	if rider == nil {
		return nil
	}
	horse := loadPortrait(src, graphicsPrefix+"infowindow/horse.bmp")
	if horse == nil {
		return rider
	}
	pic := image.NewRGBA(rider.Bounds())
	draw.Draw(pic, pic.Bounds(), horse, horse.Bounds().Min, draw.Src)
	draw.Draw(pic, pic.Bounds(), rider, rider.Bounds().Min, draw.Over)
	return pic
}

// Candidate and town party inspection share the actor's picture category.
// Siege engines have no human face fallback or equipment hit mask.
func composeMemberPortrait(src entrySource, units *terrain.UnitSet, id uint32, eq data.Equipment, m mapload.PartyMember) ui.InventorySubject {
	return composeMemberPortraitLayered(src, units, id, eq, nil, m)
}

// composeMemberPortraitLayered is composeMemberPortrait with the member's worn
// clothing layers.
func composeMemberPortraitLayered(src entrySource, units *terrain.UnitSet, id uint32, eq data.Equipment, layers []figureLayer, m mapload.PartyMember) ui.InventorySubject {
	if !data.ComposesFigure(m.Class) {
		out := ui.InventorySubject{ID: id}
		if units != nil {
			out.Figure = classPortrait(src, units.Classes[m.Class], m.FigureFace, make(map[string]*image.RGBA))
		}
		return out
	}
	out, _ := composeInventoryPortraitLayered(src, id, eq, layers, memberFigureID(m))
	return out
}

// The picture the info column shows for whatever is selected: a COMPOSED figure
// for an actor that has one, and a flat portrait bitmap for one that does not
// (`UNIT-PICT-035`, `UNIT-PICT-036`).
//
// THIS FILE CLOSES A SUBSTITUTION 0140 SHIPPED AND DISCLOSED. That story's doll
// box showed the party character's composed figure and, for anything else, the
// unit's own drawn world sprite — because the four figure directories this
// project could name are the four human ones and nothing published said where a
// monster's picture lived. It now does, and it is not a figure at all: a
// non-composing actor has a flat 24-bit bitmap under `graphics\infowindow`, one
// per class, at the same 160x240 an equipment sheet is. So the box's fallback
// stops being a placeholder and becomes the shipped art.
//
// THE DRAWING TIER STILL RECEIVES A PICTURE AND NOTHING ELSE. This file
// opens the archive, decodes a bitmap and hands over an *image.RGBA; pkg/ui
// learns no class, no registry key and no address.

// loadPortrait reads one flat portrait at addr — the FULL, prefixed address,
// like loadItemIcon's — and answers nil for every way it can fail to produce
// one.
//
// EVERY FAILURE IS AN ABSENCE AND NONE IS AN ERROR, which is loadItemIcon's own
// contract at a different leaf and is what the window above needs: a class whose
// picture the install does not carry draws no portrait and everything else about
// it works. That case is not exotic — the RU root ships thirteen structure
// sections naming a node `GRAPHICS.RES` does not have (`REG-PICT-083`) — so an
// error return here would be an invitation to log a shipped condition.
func loadPortrait(src entrySource, addr string) *image.RGBA {
	if src == nil || addr == "" {
		return nil
	}
	b, err := src.ReadFile(addr)
	if err != nil {
		return nil
	}
	pic, err := bmp.DecodeRGBA(b)
	if err != nil {
		return nil
	}
	// The 24-bit BMP has no stored alpha (SPR256-PICT-043), but the owner
	// requests a transparent black portrait background. Preserve near-black
	// artwork and leave opaque interface backgrounds on their separate loader.
	return keyBlack(pic)
}

// classPortrait is the picture for one unit class AT ONE TIER, or nil for a
// class that has none — resolved through the cache the caller holds, keyed by
// the ADDRESS rather than by the class id.
//
// KEYED BY THE ADDRESS BECAUSE THE ADDRESS IS WHAT IS READ. Two classes naming
// one picture are two reads under a class-id key and one under this one, and the
// shipped registries do exactly that — `Bee` is named by ids 1 and 2, `Mage` by
// 23 and 24. One class at two tiers is two addresses and two pictures, which the
// same key separates without a second map beside it.
//
// THE TIER IS THE OWNER'S RULING AND IT DIVERGES FROM THE ORIGINAL,
// disclosed here rather than at the call site because this is where the
// address is formed. `UNIT-PICT-036` reads the original's own info panel as
// formatting its name with THREE arguments — no tier reaches that leaf, so
// a red orc's info panel shows the green picture — and it is the DIALOGUE
// panel that appends the tier digit. So this build's info column shows the
// tier's own picture, through the dialogue panel's own decoded formatter.
// What is reproduced is the ADDRESS; which window uses it is ours.
//
// A MISS AND A FAILURE ARE CACHED ALIKE, sheetCache's own "tried" convention:
// presence in the map says the address has been resolved once, and a nil value
// is the answer for a node that is not there. So an install missing a picture
// costs one read per address for the whole session rather than one per frame.
func classPortrait(src entrySource, c *terrain.UnitClass, tier int,
	cache map[string]*image.RGBA) *image.RGBA {
	if c == nil || c.Portrait == "" {
		return nil
	}
	// A TIER BELOW ONE IS THE FIRST TIER, which is terrain.TierFrames' own
	// reading of the same number one tier over ("tier 0, which is what a
	// placement stating no tier carries" answers the sheet's own colours). It is
	// normalised HERE rather than left to the fallback below, so the ordinary
	// case — a placement with no tier at all — costs one read and not two.
	if tier < 1 {
		tier = 1
	}
	addr := graphicsPrefix + data.PortraitTierPath(c.Portrait, int32(tier))
	if pic, tried := cache[addr]; tried {
		return pic
	}
	pic := loadPortrait(src, addr)
	// A TIER THIS CLASS HAS NO FILE FOR FALLS BACK TO ITS FIRST, which is the
	// one place this build is more forgiving than the address it borrowed.
	// Three of the sixteen shipped portrait classes carry a single picture and
	// declare `Palette 1` (`REG-PICT-083`), so a placement stating a tier on one
	// of them addresses a node that does not exist; showing that class's own
	// first picture is better than showing none, and it cannot show the WRONG
	// creature, the name being the same either way.
	if pic == nil && addr != graphicsPrefix+data.PortraitPath(c.Portrait) {
		pic = classPortrait(src, c, 1, cache)
	}
	cache[addr] = pic
	return pic
}

// pushPortrait hands the viewer the flat portrait of whatever is selected, or
// none — pushSpellbook's own per-tick shape and its own reasons (spell.go): the
// selection lives in pkg/ui, it can change on a frame carrying no tick, so this
// asks v.SelectedUnit() fresh on every call.
//
// AN ACTOR THAT COMPOSES A FIGURE IS PUSHED NO PORTRAIT AT ALL, and that is the
// whole of `UNIT-PICT-035` read from this side: the two pictures are exclusive,
// the test is the class id, and the composed one already reaches the viewer by
// its own seam (SetInventorySubject). Pushing both would leave the drawing tier
// deciding which to prefer with no rule to decide it by.
//
// A SELECTION THIS WORLD NO LONGER HOLDS, A CLASS THIS BUNDLE DOES NOT KNOW AND
// A PICTURE THE INSTALL DOES NOT CARRY ALL PUSH nil, which the viewer reads as
// "no portrait for this unit" and falls back exactly as it did before this file
// existed.
func (mw *mapWorld) pushPortrait() {
	id, ok := mw.view.SelectedUnit()
	if !ok {
		mw.view.SetUnitPortrait(0, nil)
		return
	}
	e, held := mw.entity(sim.EntityID(id))
	if !held {
		mw.view.SetUnitPortrait(0, nil)
		return
	}
	pic := mw.unitPicture(sim.EntityID(id), e.Class)
	if pic == nil {
		// OWNER 0 RATHER THAN THE ID WITH A NIL PICTURE, so "this unit has no
		// picture of its own" and "no unit is selected" are one state at the
		// viewer rather than two it would have to tell apart.
		mw.view.SetUnitPortrait(0, nil)
		return
	}
	mw.view.SetUnitPortrait(id, pic)
}

// archive is the mission's own entry source, or nil for a map that has no
// mission at all.
//
// IT EXISTS BECAUSE mw.mission IS A NIL POINTER ON EVERY MAP THE PICKER OPENS,
// which its own field doc says in as many words ("IT IS NIL FOR EVERY MAP THE
// PICKER OPENS"). Every earlier reader of mw.mission.src sits behind a guard
// that happens to exclude that case — refreshPack and refreshEquipment behind
// invSubjectSet, the announcer behind the mission driver itself — so nothing
// needed this until a picture had to be read on a seam that runs for every map.
// A test found the dereference before a player did; this is the guard, in one
// place, so the next reader inherits it.
func (mw *mapWorld) archive() entrySource {
	if mw.mission == nil {
		return nil
	}
	return mw.mission.src
}

// unitPicture is the whole DECISION behind the push: which of the two kinds of
// picture an actor has, and which one of that kind. It is split out from
// pushPortrait because that method reads a selection this package cannot set
// from a test, and the decision is the part worth criteria — spellbookOf's own
// split from pushSpellbook (spell.go), for the same reason.
//
// THE CLASS DECIDES THE KIND AND NOTHING ELSE DOES (`UNIT-PICT-035`). Below the
// composing limit an actor is a DOLL — a base sheet with its worn layers, from
// the figure directory and face its own definition carries (figures.go) — and at
// or above it an actor is a FLAT BITMAP at its own tier. The two are exclusive,
// so exactly one arm ever opens the archive, and reading an InfoPicture for a
// composing class would be opening a node for a value the engine can never
// format: dead data, on thirteen shipped classes (`REG-PICT-083`).
//
// THE PARTY'S OWN SUBJECT ANSWERS nil ON THE COMPOSING ARM, and that is not a
// gap: its figure already reaches the viewer through SetInventorySubject, which
// composes the worn ICONS beside it and refreshes on every equipment change, and
// the box prefers that picture anyway (pkg/ui's dollSubject). Composing a second
// copy here would be twelve more archive reads for a picture the window already
// has, and two caches to keep in step.
//
// A RECORDED FIGURE WINS OVER THE CLASS TEST, inspectionUnitPicture's own
// reading applied here: a human restored from an original SAV carries its
// overwritten type id (gender + 0x21 or + 0x23, DAT-HUMANS-008), which is
// above the compose limit although the actor still has a doll in mw.figures.
func (mw *mapWorld) unitPicture(id sim.EntityID, class int32) *image.RGBA {
	_, composed := mw.figures[id]
	if composed || data.ComposesFigure(class) {
		if mw.invSubjectSet && uint32(id) == mw.invSubject.ID {
			return nil
		}
		return mw.unitFigure(id)
	}
	if mw.units == nil {
		return nil
	}
	return classPortrait(mw.archive(), mw.units.Classes[class], mw.tiers[id], mw.portraits)
}

// inspectionUnitPicture reads current equipment, never the inventory interaction
// subject. The viewer calls it after this frame's pointer update so the figure
// and card cannot lag one another by a frame.
//
// A recorded figure wins over the class test. A human restored from an original
// SAV carries its overwritten type id (gender + 0x21 or + 0x23, DAT-HUMANS-008),
// which is above the compose limit although the actor still has a doll.
func (mw *mapWorld) inspectionUnitPicture(id uint32) *image.RGBA {
	e, ok := mw.entity(sim.EntityID(id))
	if !ok {
		return nil
	}
	if _, composed := mw.figures[e.ID]; composed || data.ComposesFigure(e.Class) {
		return mw.unitFigure(e.ID)
	}
	return mw.unitPicture(e.ID, e.Class)
}
