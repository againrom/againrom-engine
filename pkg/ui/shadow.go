package ui

import (
	"image"

	"github.com/hajimehoshi/ebiten/v2"

	"againrom/pkg/render/terrain"
)

// Shadows retain their frame, shear, cull and blend from 0102. Each admitted
// shadow carries its category/cell key so drawArt can place it in the accepted
// composition instead of draining a single frame-wide shadow band.

type shadowDraw struct {
	screenRect
	order    artOrder
	Frame    *terrain.StaticFrame
	Alpha    float32
	Slope    float64
	PivotRow int
	Mirror   bool
	// Unit is the casting entity's ID when Cast is set.
	Unit uint32
	Cast bool
	// Second marks a caster's second silhouette; Flat one drawn by the
	// translated arm.
	Second, Flat bool
}

func shadowWiden(rect image.Rectangle, slope float64, pivotRow int) image.Rectangle {
	if rect.Dy() <= 0 {
		return rect
	}
	top := terrain.ShadowRowOffset(slope, pivotRow, 0)
	bottom := terrain.ShadowRowOffset(slope, pivotRow, rect.Dy()-1)
	lo, hi := top, bottom
	if lo > hi {
		lo, hi = hi, lo
	}
	return image.Rect(rect.Min.X+lo, rect.Min.Y, rect.Max.X+hi, rect.Max.Y)
}

func (v *Viewer) shadowDraws() []shadowDraw {
	if v.graphics.HideShadows {
		return nil
	}
	sun := v.Sun()
	theta := sun.Theta
	zoom := v.cam.Zoom
	slope := terrain.ShadowSlope(theta)
	var out []shadowDraw

	submit := func(rect image.Rectangle, frame *terrain.StaticFrame, alpha float32, mirror bool, slope float64, order artOrder) {
		pivotRow := terrain.ShadowPivotRow(frame)
		if _, ok := spriteScreenRect(shadowWiden(rect, slope, pivotRow), v.cam); !ok {
			return
		}
		originX := float64(rect.Min.X) + slope*float64(pivotRow)
		sx, sy := v.cam.WorldToScreen(originX, float64(rect.Min.Y))
		out = append(out, shadowDraw{
			order:      order,
			screenRect: screenRect{X: sx, Y: sy, W: float64(frame.Width) * zoom, H: float64(frame.Height) * zoom},
			Frame:      frame,
			Alpha:      alpha,
			Slope:      slope,
			PivotRow:   pivotRow,
			Mirror:     mirror,
		})
	}

	if v.showStaticArt {
		alpha := float32(terrain.ShadowLevel(sun, terrain.ShadowObject)) / 16
		originY := 0
		if v.Mode() == ModeDisplaced {
			originY = v.proj.MinV
		}
		for i, p := range v.staticPlacements() {
			shadow, ok := terrain.ObjectShadowPlace(p, theta, originY)
			if !ok || shadow.Frame.Width <= 0 || shadow.Frame.Height <= 0 {
				continue
			}
			submit(shadow.Rect(), shadow.Frame, alpha, shadow.Mirror, slope,
				artOrder{phase: artMainCell, cell: p.Cell, rank: 4, index: i})
		}
	}

	if v.showStructureArt {
		alpha := float32(terrain.ShadowLevel(sun, terrain.ShadowObject)) / 16
		for i, p := range v.structurePlacements() {
			shadow, ok := terrain.StructureShadowPlace(p, theta)
			if !ok || shadow.Frame.Width <= 0 || shadow.Frame.Height <= 0 {
				continue
			}
			submit(shadow.Rect(), shadow.Frame, alpha, false, slope,
				artOrder{phase: artStructureShadow, cell: p.Cell, index: i})
		}
	}

	// A unit's shadow is up to two silhouettes (TERR-191): the drawn sheet's
	// frame at the object-path level, then the paired spritesb frame at the
	// unit-path level when Smoothing is on. An invisible unit casts a shadow
	// only on its owner's client, one silhouette at the unit-path level.
	// A CAirUnit's silhouettes are translated and not sheared (TERR-193).
	sprites, _, marked := v.entityLayer()
	displaced := v.Mode() == ModeDisplaced
	shear16 := terrain.ShadowShear16(theta)
	for i, p := range sprites {
		e := marked[i]
		if e.Translucent && e.Owner != v.localOwner {
			continue
		}
		firstKind := terrain.ShadowObject
		if e.Translucent {
			firstKind = terrain.ShadowUnit
		}
		flat := e.DrawCategory == terrain.UnitAir && !e.PlayerCharacter
		pivot := 0
		if p.Frame != nil {
			pivot = terrain.ShadowPivotShift(theta, p.Frame.Height, p.Anchor.Y)
		}
		place := func(body terrain.StaticPlacement) (terrain.StaticPlacement, float64) {
			if flat {
				body.TopLeft.X += shear16 / 2000
				return body, 0
			}
			body.TopLeft.X -= pivot
			return body, slope
		}
		order := entityArtOrder(p, e, i)
		order.part = 0
		if order.phase == artAirBody {
			order.phase = artAirShadow
		}
		cast := func(shadow terrain.StaticPlacement, level terrain.ShadowKind, shadowSlope float64, second bool) {
			if shadow.Frame == nil || shadow.Frame.Width <= 0 || shadow.Frame.Height <= 0 {
				return
			}
			before := len(out)
			submit(shadow.Rect(), shadow.Frame, float32(terrain.ShadowLevel(sun, level))/16, shadow.Mirror, shadowSlope, order)
			if len(out) > before {
				d := &out[before]
				d.Unit, d.Cast, d.Second, d.Flat = e.ID, true, second, flat
			}
		}
		first, firstSlope := place(p)
		cast(first, firstKind, firstSlope, false)
		if e.Translucent || !v.graphics.Smoothing || e.Boundary == nil {
			continue
		}
		lift, originY := 0, 0
		if displaced {
			lift, originY = v.proj.AnchorHeight(e.Cell.X, e.Cell.Y), v.proj.MinV
		}
		own, ok1 := terrain.UnitPlace(e.Cell.X, e.Cell.Y, e.Art, p.Frame, p.Mirror, lift, originY)
		pair, ok2 := terrain.UnitPlace(e.Cell.X, e.Cell.Y, e.Art, e.Boundary, p.Mirror, lift, originY)
		if !ok1 || !ok2 {
			continue
		}
		second := p
		second.Frame = e.Boundary
		second.Anchor = pair.Anchor
		second.TopLeft = p.TopLeft.Add(pair.TopLeft.Sub(own.TopLeft))
		shadow, secondSlope := place(second)
		cast(shadow, terrain.ShadowUnit, secondSlope, true)
	}

	return out
}

var shadowBlend = ebiten.Blend{
	BlendFactorSourceRGB:        ebiten.BlendFactorZero,
	BlendFactorSourceAlpha:      ebiten.BlendFactorZero,
	BlendFactorDestinationRGB:   ebiten.BlendFactorOneMinusSourceAlpha,
	BlendFactorDestinationAlpha: ebiten.BlendFactorOne,
}

// shadowMask is the frame's silhouette texture, built through
// terrain.ShadowMask on first use and cached under the frame's own pointer
// identity for the life of the run — staticImage's own precedent, applied
// to a mask instead of a lit sprite. It depends on the frame alone, never on
// the level or the angle, so one hour's relight and the next serve the same
// entry.
//
// It is NIL UNTIL THE FIRST SHADOW DRAW BUILDS ONE, mirroring staticImages: a
// viewer built, queried and culled but never drawn holds no GPU state, which
// is what lets a headless -check run reach this whole layer with no graphics
// context.
func (v *Viewer) shadowMask(f *terrain.StaticFrame) *ebiten.Image {
	if img, ok := v.shadowMasks[f]; ok {
		return img
	}
	img := ebiten.NewImageFromImage(terrain.ShadowMask(f))
	if v.shadowMasks == nil {
		v.shadowMasks = make(map[*terrain.StaticFrame]*ebiten.Image)
	}
	v.shadowMasks[f] = img
	return img
}

// drawShadows paints shadowDraws' own list: the mask keyed on the frame
// alone, shadowBlend and the entry's own alpha as a colour scale — the
// level rides the draw and never the mask's pixels (0101) — and the shear
// and the mirror composed in one local matrix, before the camera's own zoom
// and position, rather than a second texture.
//
// THE SHEAR RIDES THE TRANSFORM ALONE.
//
// THE MIRROR COMPOSES INSIDE THAT SAME LOCAL MATRIX, NOT AROUND IT. A mirror
// alone would be Scale(-1,1) then Translate(width,0) — flip the local
// frame about its own origin, then slide it back over its own rectangle —
// but since a later Scale multiplies the ROW the shear element sits in,
// applying that flip as a separate ebiten Scale call after the shear was set
// would negate the shear's own sign along with the frame's, leaning a
// mirrored shadow the wrong way. Setting both elements directly instead —
// SetElement(0,0,-1) and SetElement(0,2, frame width) alongside the
// untouched SetElement(0,1,-slope) — composes to x -> (w-x) - slope*y in
// one matrix, so the mirror flips the silhouette and the shear leans it,
// independently: a mirrored shadow leans the same way as an unmirrored one
// at the same instant, matching the software walk's own w-1-col reflection
// on the source (spec D-2 owns the resulting one-pixel difference between
// the two).
//
// ebiten's GeoM composes each later call OVER what came before (Scale
// multiplies the existing rows, Translate adds to the existing translation
// untouched by any scale — geom.go's own arithmetic), so the SetElement calls
// before Scale(zoom,zoom) before Translate(d.X, d.Y) apply the local matrix to
// the LOCAL frame first and the camera's zoom and position to the whole
// sheared (and possibly mirrored) shape after, exactly as a plain sprite's
// Scale-then-Translate already does. shadowDraws' own X,Y is therefore the
// screen position of local (0,0) — the frame's row-0 left edge, already
// carrying row 0's own shear offset — and not the placement's unsheared
// TopLeft.
//
// d.W — the already-zoomed screenRect field — plays no part in the mirror
// branch any more; the local matrix's own width term is read straight off
// d.Frame.Width, unscaled, since Scale(zoom,zoom) scales it a moment later
// exactly as it scales the shear term. d.W remains only the plain rectangle a
// caller might read off the entry.
func (v *Viewer) drawShadows(target imageTarget) {
	for _, d := range v.shadowDraws() {
		v.drawShadow(target, d)
	}
}

func (v *Viewer) drawShadow(target imageTarget, d shadowDraw) {
	zoom := v.cam.Zoom
	var op ebiten.DrawImageOptions
	op.Filter = ebiten.FilterNearest
	op.Blend = shadowBlend
	op.ColorScale.ScaleAlpha(d.Alpha)
	op.GeoM = shadowGeoM(d, zoom)
	target.DrawImage(v.shadowMask(d.Frame), &op)
}

func shadowGeoM(d shadowDraw, zoom float64) ebiten.GeoM {
	var g ebiten.GeoM
	g.SetElement(0, 1, -d.Slope)
	if d.Mirror {
		g.SetElement(0, 0, -1)
		g.SetElement(0, 2, float64(d.Frame.Width))
	}
	g.Scale(zoom, zoom)
	g.Translate(d.X, d.Y)
	return g
}

// UnitShadow is one entity's shadow as the viewer will draw it: the casting
// entity, the shroud level it recolours by, the shear slope and the frame
// whose silhouette it stamps.
type UnitShadow struct {
	ID    uint32
	Level int
	Slope float64
	Frame *terrain.StaticFrame
	// Second is the caster's second silhouette, Flat a translated one.
	Second, Flat bool
}

// UnitShadows lists the shadows every drawn entity casts, in draw order.
func (v *Viewer) UnitShadows() []UnitShadow {
	var out []UnitShadow
	for _, d := range v.shadowDraws() {
		if d.Cast {
			out = append(out, UnitShadow{ID: d.Unit, Level: int(d.Alpha*16 + 0.5), Slope: d.Slope, Frame: d.Frame, Second: d.Second, Flat: d.Flat})
		}
	}
	return out
}
