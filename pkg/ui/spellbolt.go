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
// Display marks a path stamp (pictures 34 and 36): Pos is a native display
// pixel, terrain height already subtracted, and the stamp is centred by the
// immediate 8 rather than the sheet's halves (ANIM-BOLTDRAW-034).
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
	Display          bool
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

// GroundPixel is a cell-relative ShotScale point's ground pixel: the point
// every non-display spell object is centred on before relief.
func GroundPixel(pos image.Point) image.Point {
	return (SpellBolt{Pos: pos}).groundPoint()
}

// boltStampCentre is the path drawer's centring immediate (ANIM-BOLTDRAW-034).
const boltStampCentre = 8

// DisplayHeight is the terrain height a cell's display point subtracts:
// AnchorHeight in the displaced view, zero in the flat view.
func (v *Viewer) DisplayHeight(cell image.Point) int {
	if v.Mode() != ModeDisplaced {
		return 0
	}
	return v.proj.AnchorHeight(cell.X, cell.Y)
}

// displayLift moves a native display pixel into the camera world.
func (v *Viewer) displayLift() int {
	if v.Mode() != ModeDisplaced {
		return 0
	}
	return -v.proj.MinV
}

// DisplayRow is the ground row under a native display point: the first row,
// in ascending order, whose corner-edge bounds at x contain y (the ground
// picker's edge model, TERR-GEOM-036). The flat view answers y>>5.
func (v *Viewer) DisplayRow(x, y int) (int, bool) {
	if v.Mode() != ModeDisplaced {
		return y >> 5, true
	}
	col, wy := x>>5, y-v.proj.MinV
	if col < 0 || col >= v.proj.Width {
		return 0, false
	}
	for row := 0; row < v.proj.Height; row++ {
		if top, bottom := v.proj.CellColumnBounds(col, row, x); top <= wy && wy <= bottom {
			return row, true
		}
	}
	return 0, false
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
		point, lift := b.groundPoint(), v.spellBoltLift(b)
		px, py := point.X-b.Sheet.CenterX, point.Y-b.Sheet.CenterY
		if b.Display {
			px, py, lift = b.Pos.X-boltStampCentre, b.Pos.Y-boltStampCentre, v.displayLift()
		}
		r, ok := v.placeLifted(lift, image.Rect(px, py, px+f.Width, py+f.Height))
		if !ok {
			continue
		}
		out = append(out, effectScreenRect{screenRect: r, Cell: b.Cell, Frame: f, Mirror: b.Mirror, Pass: b.Pass})
	}
	return out
}

// spellBoltLift is one stamp's relief: the terrain lift of the near cell, of
// the far cell, or the interpolation by the stamp's projection onto the
// segment (`MAGIC-BOLTSHAPE-070`). The launch point already carries its
// height above the caster's ground point (MAGIC-261). A still object takes
// its one cell's lift.
func (v *Viewer) spellBoltLift(b SpellBolt) int {
	from := v.cellLift(b.Cell)
	if b.To == b.Cell {
		return from
	}
	dx, dy := (b.To.X-b.Cell.X)*ShotScale, (b.To.Y-b.Cell.Y)*ShotScale
	px, py := b.Pos.X-b.Cell.X*ShotScale, b.Pos.Y-b.Cell.Y*ShotScale
	num, den := px*dx+py*dy, dx*dx+dy*dy
	if num <= 0 || den <= 0 {
		return from
	}
	to := v.cellLift(b.To)
	if num >= den {
		return to
	}
	return from + (to-from)*num/den
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
