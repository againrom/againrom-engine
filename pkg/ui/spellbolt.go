package ui

import (
	"image"

	"github.com/hajimehoshi/ebiten/v2"

	"againrom/pkg/render/refraction"
	"againrom/pkg/render/terrain"
)

// SpellBolt is one spell object as the tier above hands it over: where it
// stands, whose cell the relief lift is keyed off, whose visibility gates it,
// and the sheet frame to draw.
//
// Pos is in ShotScale units. Cell-relative effects leave AbsolutePosition
// false; carried projectile records already include the canonical fine offset.
//
// Cell AND Owner ARE THE CASTER'S, and Cell gates the fog: an object is as much
// a statement about the caster as an archer's shot is, so it is hidden wherever
// the caster is hidden.
//
// To IS THE OBJECT'S FAR CELL, equal to Cell for anything that stands still.
// The two cells are what the RELIEF is interpolated between (spellBoltLift):
// the decoded generator builds its figure between two SCREEN points, the
// projectile drawable's own and the resolved target's, each already carrying
// its own height term (`MAGIC-BOLTSHAPE-070`).
//
// Sheet CARRIES THE CENTRING HALVES and Frame indexes it. The halves are the
// registry's own Width and Height halved and they are NOT the art's size: the
// art is centred on the point by subtracting them, and where they disagree with
// the frame the sprite shifts rather than clipping. Sprite effects with no sheet
// or an invalid frame draw nothing. Background deformation carries no sheet.
type SpellBolt struct {
	Cell             image.Point
	To               image.Point
	Pos              image.Point
	AbsolutePosition bool
	Sheet            *terrain.EffectSheet
	Frame            int
	Mirror           bool
	Owner            uint32
	Pass             SpellPass
	Effect           SpellEffect
	Phase            int
	Mapping          *refraction.Map
}

type SpellEffect uint8

const (
	SpellSprite SpellEffect = iota
	SpellBackgroundDeformation
)

// SpellPass distinguishes retained-area selectors from the projectile list.
// ANIM-CELL-085/AIRPASS-086 place A within each main cell and B after the list.
// Zero keeps ordinary cast objects in the separate projectile list. Retained
// cell overlays are tagged by their producer, not inferred from shared art.
type SpellPass uint8

const (
	SpellProjectiles SpellPass = iota
	SpellOverlayA              // Fire/Earth, after static art in each main-sweep cell
	SpellOverlayB              // Freezing/Poison, between projectiles and air bodies
)

// SetSpellBolts hands the viewer this frame's spell objects, replacing whatever
// it held — SetEntities' own per-frame shape. A viewer never given any draws
// none, which is every viewer cmd/mapview can build.
func (v *Viewer) SetSpellBolts(bolts []SpellBolt) {
	v.spellBolts = append(v.spellBolts[:0], bolts...)
}

// SpellBolts reports how many objects this frame carries, for a caller measuring
// the seam rather than the picture.
func (v *Viewer) SpellBolts() int { return len(v.spellBolts) }

// EffectGroundPoint places a cell-relative effect less its sheet's centring halves.
func EffectGroundPoint(pos image.Point, sheet *terrain.EffectSheet) (px, py int) {
	point := (SpellBolt{Pos: pos}).groundPoint()
	return point.X - sheet.CenterX, point.Y - sheet.CenterY
}

func (b SpellBolt) groundPoint() image.Point {
	point := image.Pt(b.Pos.X*terrain.CellSize/ShotScale, b.Pos.Y*terrain.CellSize/ShotScale)
	if !b.AbsolutePosition {
		point = point.Add(image.Pt(terrain.CellSize/2, terrain.CellSize/2))
	}
	return point
}

// effectScreenRect is one spell object placed in the current view: where its
// frame lands on screen and how big it is there, beside the frame itself and its
// reflection bit.
//
// It is staticScreenRect's shape one sheet format over, and separate for the one
// reason that matters: the frame is a different type, so a single rect type
// would carry two pointers of which exactly one is ever set.
type effectScreenRect struct {
	screenRect
	Cell    image.Point
	Frame   *terrain.EffectFrame
	Mirror  bool
	Pass    SpellPass
	Effect  SpellEffect
	Center  image.Point
	Mapping *refraction.Map
}

// spellArtPlacements is every drawable spell object placed in the current
// view, in the order the tier above produced them.
//
// THE FOG GATE IS THE SELECTION RIM'S OWN (fogGateEntity), as the mark's was: an
// object leaving a caster standing in the dark shows nothing.
//
// THE PLACEMENT IS placeArm's, term for term, over a lift interpolated between
// the object's two cells rather than taken whole from one of them
// (spellBoltLift): an object standing on one cell carries exactly the relief a
// mark on that cell carries, and one spanning two carries each end's.
func (v *Viewer) spellArtPlacements() []effectScreenRect {
	var out []effectScreenRect
	for _, b := range v.spellBolts {
		if !v.fogGateEntity(b.Owner, b.Cell.X, b.Cell.Y) {
			continue
		}
		if b.Effect == SpellBackgroundDeformation {
			if placement, ok := v.deformationPlacement(b); ok {
				out = append(out, placement)
			}
			continue
		}
		f := b.Sheet.Frame(b.Frame)
		if f == nil || f.Width <= 0 || f.Height <= 0 {
			continue
		}
		point := b.groundPoint()
		px, py := point.X-b.Sheet.CenterX, point.Y-b.Sheet.CenterY
		r, ok := v.placeLifted(v.spellBoltLift(b), image.Rect(px, py, px+f.Width, py+f.Height))
		if !ok {
			continue
		}
		out = append(out, effectScreenRect{screenRect: r, Cell: b.Cell, Frame: f, Mirror: b.Mirror, Pass: b.Pass})
	}
	return out
}

// spellHandLift is the departure end's own height above the ground point it
// leaves, in world pixels — HALF A CELL (16), negative because a smaller
// screen Y is higher (cellLift's own sign, `overlay.go`). No claim states a
// magnitude or that the original carries a hand-height term at all at this
// seam; `MAGIC-BOLTSHAPE-070` establishes only that the decoded figure's own
// two endpoints are screen-space fields of a drawable with that drawable's
// own height term already subtracted, which is the structure this term is
// added into, not its size. DIV-083, amended.
const spellHandLift = -terrain.CellSize / 2

// spellBoltLift is the vertical relief one stamp of one object carries: the
// TERRAIN lift of the object's own cell at its near end, the terrain lift of
// its far cell at the far end, a linear interpolation of the two in between,
// and — at the near end only — the departure's own hand height
// (spellHandLift), decaying linearly to zero by the far end.
//
// THE HAND TERM IS A PROPERTY OF THE CASTER, NOT OF THE TERRAIN, so it is
// added in BOTH display modes: cellLift already returns 0 for both ends in
// flat mode (`v.Mode() != ModeDisplaced`), which makes the terrain
// interpolation 0 there on its own, and nothing else in flat mode reads
// altitude, so adding the hand term costs no separate mode branch and changes
// no other flat-mode drawn quantity.
//
// THE TERRAIN INTERPOLATION IS OF THE LIFT AND NOT OF THE GROUND, exactly as
// it was before this term was added. The decoded figure is a straight segment
// between two screen points, each of them a drawable's position with that
// drawable's own height term already subtracted (`MAGIC-BOLTSHAPE-070`), so
// nothing in it re-reads the terrain between the two ends. A bolt therefore
// crosses a valley in a straight line rather than dipping into it, and a
// stamp's terrain height is a function of how far along the segment it stands
// and of nothing else.
//
// AN OBJECT THAT STANDS STILL TAKES ONE LIFT AND NO HAND TERM. Its two cells
// are equal — every burst, every overlay cell and every cast onto the
// caster's own cell — and both the terrain interpolation and the hand term
// are skipped: a burst has no departure end to raise, and the previous hotfix
// in this area (DIV-083) learned that lifting only one end of a same-cell
// object makes it visibly crawl, so this term takes the same early return
// that already protects against that.
//
// THE FAR END KEEPS ITS PRESENT HEIGHT. This build has no per-target
// body-height term at this seam; inventing one is not this hotfix.
//
// The projection is asked twice and the fraction is formed from the dot product
// of the stamp's own offset onto the segment, so a stamp off the segment (a
// figure's own perpendicular excursion) takes the lift of its projection onto
// it, the hand term included, and no stamp reaches past either end.
func (v *Viewer) spellBoltLift(b SpellBolt) int {
	from := v.cellLift(b.Cell)
	if b.To == b.Cell {
		return from
	}
	dx, dy := (b.To.X-b.Cell.X)*ShotScale, (b.To.Y-b.Cell.Y)*ShotScale
	px, py := b.Pos.X-b.Cell.X*ShotScale, b.Pos.Y-b.Cell.Y*ShotScale
	num, den := px*dx+py*dy, dx*dx+dy*dy
	if num <= 0 || den <= 0 {
		return from + spellHandLift
	}
	to := v.cellLift(b.To)
	if num >= den {
		return to
	}
	terrainLift := (to - from) * num / den
	hand := spellHandLift * (den - num) / den
	return from + terrainLift + hand
}

// drawSpellArt paints one pass of this frame's already placed spell objects.
//
// drawArt owns its position around shadows and bodies. All three spell passes
// remain below instruments, health bars, routes and damage numerals.
//
// The GeoM is the sprite pass's own, reflection included, so a mirrored facing
// is reflected inside its own rectangle exactly as a mirrored unit is. Filter is
// Nearest, for the sprite pass's own reason: art is blocky at high zoom by
// choice, and smoothing would bleed a transparent edge pixel into its
// neighbour's colour.
func (v *Viewer) drawSpellArt(target imageTarget, placements []effectScreenRect, pass SpellPass) {
	zoom := v.cam.Zoom
	for _, s := range placements {
		if s.Pass != pass {
			continue
		}
		if s.Effect == SpellBackgroundDeformation {
			v.drawBackgroundDeformation(target, s)
			continue
		}
		var op ebiten.DrawImageOptions
		op.Filter = ebiten.FilterNearest
		if s.Mirror {
			op.GeoM.Scale(-zoom, zoom)
			op.GeoM.Translate(s.X+s.W, s.Y)
		} else {
			op.GeoM.Scale(zoom, zoom)
			op.GeoM.Translate(s.X, s.Y)
		}
		target.DrawImage(v.effectImage(s.Frame), &op)
	}
}

// effectImage is the frame's GPU texture, built on its FIRST draw and cached
// under the frame's own pointer identity from then on — staticImage's shape, and
// for its reasons.
//
// NO TEXTURE EXISTS UNTIL A DRAW ASKS FOR ONE, and the cache map is nil
// until this is first called, so a viewer that has been constructed, queried
// and culled but never drawn holds no map at all.
//
// THE KEY IS THE FRAME POINTER. The loader puts one frame slice behind every
// picture that names one file, so two rows sharing a sheet share its textures,
// and a bolt drawn on twenty consecutive ticks uploads at most four pictures —
// one per phase of its sheet.
//
// The pixels come through the frame's own RGBA, which is already premultiplied
// at the coverage the sprite format states, so no blend is performed here and
// none may be: a second one would be the second, disagreeing walk this tier
// exists not to have.
func (v *Viewer) effectImage(f *terrain.EffectFrame) *ebiten.Image {
	if img, ok := v.effectImages[f]; ok {
		return img
	}
	img := ebiten.NewImageFromImage(f.RGBA())
	if v.effectImages == nil {
		v.effectImages = make(map[*terrain.EffectFrame]*ebiten.Image)
	}
	v.effectImages[f] = img
	return img
}
