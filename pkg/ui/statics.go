package ui

import (
	"image"
	"sort"

	"github.com/hajimehoshi/ebiten/v2"

	"againrom/pkg/render/camera"
	"againrom/pkg/render/terrain"
)

// staticScreenRect is one placement placed in the current view: where its frame
// lands on screen and how big it is there, beside the frame itself.
//
// It embeds screenRect so the marker overlay's own rectangle type states the
// geometry once — a sprite and a marker arm are placed by the same two camera
// expressions and differ only in what is drawn — and adds the one thing a marker
// has no equivalent of: the frame whose texture is drawn into that rectangle.
//
// Frame is the placement's own pointer into the bundle, carried through
// untouched. Two cells of one class therefore arrive here holding ONE pointer,
// which is what lets the texture cache below key on frame identity and upload a
// class's art once for the process rather than once per cell.
//
// Effect is the OTHER art a band entry can carry (1002): one frame of a
// projectile sheet, which is what an effect mark draws through. Exactly one of
// Frame and Effect is ever set. The two share this type — against
// effectScreenRect's own reason for not sharing one — because a mark is
// INTERLEAVED with the actor's own sprite in a single ordered band, so the two
// must ride one list or the two-pass split cannot be expressed at all.
type staticScreenRect struct {
	screenRect
	order              artOrder
	Frame              *terrain.StaticFrame
	Effect             *terrain.EffectFrame
	Mirror             bool
	Stone, Translucent bool
	SackOutline        bool
	Brightness         float32
	ShadeZero          bool
	Unlit              bool
	Inspection         InspectionSubject
	Tint               spriteTint
}

// staticScreenRects is the layer's visibility test and camera transform in
// one pure pass: every placement whose art meets the view, in the order the
// builder produced them, each placed at screen = (world - camera) * Zoom
// with its native size scaled by Zoom (AC-6).
//
// THE CULL IS ON THE EXACT WORLD RECTANGLE, p.Rect(), and this is the point
// of the function. An object's art routinely reaches far above the cell it
// stands on, so a sprite whose GROUND CELL sits below the view while its
// crown enters it is visible and must be kept. Culling by the ground cell
// drops exactly that one; so does culling by the camera's own tile band,
// which is the same mistake spelt with a wider net. Both were rejected, and
// both would leave a forest that pops in only once its trunks come into view
// — the failure this test is shaped to prevent rather than to detect.
//
// The transform and the cull shape are the marker overlay's, term for term:
// cam.WorldToScreen of the rectangle's top-left, the size scaled by the EXPORTED
// cam.Zoom field (the same field the tile loop and overlayScreenRects read, so
// the three agree in every camera state, including one a caller produced by
// assigning Zoom directly), and a rect kept unless it lies wholly outside
// [0,ViewW) x [0,ViewH). The cull runs AFTER the transform, on the placed
// rectangle, so a placement that would have met the view read as raw world
// coordinates is dropped and one that meets it only once the camera is applied
// is kept.
//
// A partly visible sprite is kept WHOLE and clipped by the framebuffer when
// it is drawn, exactly as a straddling marker arm is: nothing here invents
// an edge. Survivors hold their order and nothing is added, dropped or
// duplicated beyond the cull — the loop appends at most one rect per
// placement and never merges two, so a class drawn at ten cells arrives as
// ten entries.
//
// Coordinates go out unrounded. The only narrowing is the float32 the engine
// takes, at the draw call, never a pixel snap of our own.
func staticScreenRects(places []terrain.StaticPlacement, cam *camera.Camera) []staticScreenRect {
	var out []staticScreenRect
	for _, p := range places {
		r, ok := spriteScreenRect(p.Rect(), cam)
		if !ok {
			continue
		}
		out = append(out, staticScreenRect{screenRect: r, Frame: p.Frame, Mirror: p.Mirror})
	}
	return out
}

// spriteScreenRect places ONE world rectangle in the current view and reports
// whether it meets it: screen = (world - camera) * Zoom, the native size scaled
// by Zoom, kept unless the placed rectangle lies wholly outside
// [0,ViewW) x [0,ViewH).
//
// IT IS THE PACKAGE'S ONE CULL AND ITS ONE TRANSFORM, and that is why it exists
// as a function rather than as three copies of six lines: the object pass, the
// structure pass and the merged draw all place the same way, so a rectangle that
// one keeps is a rectangle all three keep. A second spelling would let the drawn
// sequence and the tested one disagree about an edge case neither could be said
// to have got wrong.
//
// The cull runs AFTER the transform, on the placed rectangle, so a placement that
// would have met the view read as raw world coordinates is dropped and one that
// meets it only once the camera is applied is kept. Coordinates go out unrounded;
// the only narrowing is the float32 the engine takes, at the draw call.
func spriteScreenRect(r image.Rectangle, cam *camera.Camera) (screenRect, bool) {
	zoom := cam.Zoom
	sx, sy := cam.WorldToScreen(float64(r.Min.X), float64(r.Min.Y))
	w := float64(r.Dx()) * zoom
	h := float64(r.Dy()) * zoom
	s := screenRect{X: sx, Y: sy, W: w, H: h}
	return s, spriteVisible(s, cam)
}

func spriteVisible(s screenRect, cam *camera.Camera) bool {
	return s.X+s.W > 0 && s.Y+s.H > 0 && s.X < float64(cam.ViewW) && s.Y < float64(cam.ViewH)
}

// imageTarget is the one method drawStatics needs out of a draw target.
// *ebiten.Image already satisfies it, so Draw passes screen through unchanged.
//
// Taking this interface, and not *ebiten.Image itself, is what makes the
// sprite pass OBSERVABLE — the same reason triangleTarget exists one file
// over. An *ebiten.Image's pixels cannot be read back before the game
// starts, so with a concrete target a test could assert only that drawing
// did not panic; through this, a plain Go struct records which texture was
// submitted, at which transform, in which order. The measured precedent is
// worth repeating: with the terrain loops fixed at *ebiten.Image an entire
// story's shading was reachable but unverified, because every assertion
// recomposed the expected value instead of asking the loop what it actually
// submitted.
type imageTarget interface {
	DrawImage(img *ebiten.Image, options *ebiten.DrawImageOptions)
}

// sackLayer is this frame's placed sack stream: one terrain.StaticPlacement
// per world sack whose cell this drawing side's own grid contains and whose
// frame index falls inside the frames it was separately given — nothing
// else joins it.
//
// lift AND originY ARE THIS SACK'S OWN CELL, read per call exactly as
// entityLayer reads them for a unit: 0 and 0 in the flat geometry, the
// cell's own AnchorHeight and the projection's MinV in the displaced one —
// live under Mode(), so a sack is lifted correctly whichever geometry a
// caller reads this in and however SetFlat has moved between two calls.
func (v *Viewer) sackLayer() []terrain.StaticPlacement {
	if len(v.sacks) == 0 {
		return nil
	}
	displaced := v.Mode() == ModeDisplaced
	out := make([]terrain.StaticPlacement, 0, len(v.sacks))
	for _, s := range v.sacks {
		if s.Cell.X < 0 || s.Cell.Y < 0 || s.Cell.X >= v.grid.Width || s.Cell.Y >= v.grid.Height {
			continue
		}
		// THE FOG GATE: a sack is drawn only while its own cell is FogVisible
		// right now, with no owner to except it the way an entity's own owner
		// does. Gated here, at the one place a sack becomes a placement, so
		// nothing downstream of sackLayer can draw one this check has already
		// refused.
		if !v.fogGateSack(s.Cell.X, s.Cell.Y) {
			continue
		}
		if s.FrameIndex < 0 || s.FrameIndex >= len(v.sackFrames) {
			continue
		}
		lift, originY := 0, 0
		if displaced {
			lift = v.proj.AnchorHeight(s.Cell.X, s.Cell.Y)
			originY = v.proj.MinV
		}
		if p, ok := terrain.SackPlace(s.Cell.X, s.Cell.Y, v.sackFrames[s.FrameIndex], lift, originY); ok {
			out = append(out, p)
		}
	}
	return out
}

// planeSprites builds the visible body/mark sequence for painting and picking.
// DepthOrder supplies the native stable corpse/sack/unit fallback order; the
// accepted category, cell and dispatch keys then arrange these admitted bodies.
// Shadows and retained areas use the same keys in drawArt (ANIM-CELL-085).
// Frame rectangles, fog gates, art switches and actor-local mark folds keep
// their existing meaning; a culled body still admits its separately culled marks.
func (v *Viewer) planeSprites() []staticScreenRect {
	objects := v.staticPlacements()
	structures := v.structurePlacements()
	entities, _, marked := v.entityLayer()
	sacks := v.sackLayer()
	v.depthOrder = terrain.DepthOrder(v.depthOrder, v.planeOrder, structures, objects, entities, sacks)

	out := make([]staticScreenRect, 0, len(v.depthOrder))
	for _, ref := range v.depthOrder {
		var f *terrain.StaticFrame
		var boundary *terrain.StaticFrame
		var rect image.Rectangle
		var mirror bool
		var stone, translucent bool
		var brightness float32
		var shadeZero bool
		var front []staticScreenRect
		var back []staticScreenRect
		var order artOrder
		var inspection InspectionSubject
		switch ref.Kind {
		case terrain.PlaneEntity:
			if ref.Index >= len(entities) {
				continue
			}
			p := entities[ref.Index]
			f, rect, mirror = p.Frame, p.Rect(), p.Mirror
			if ref.Index < len(marked) {
				order = entityArtOrder(p, marked[ref.Index], ref.Index)
				if !marked[ref.Index].Untargetable {
					inspection = InspectionSubject{Kind: InspectionUnit, ID: marked[ref.Index].ID}
				}
				stone, translucent = marked[ref.Index].Stone, marked[ref.Index].Translucent
				boundary = marked[ref.Index].Boundary
				factor := v.spellSpriteFactor(marked[ref.Index].Cell)
				if _, shadeZero = v.hoverSpriteFactor(marked[ref.Index].ID); shadeZero {
					factor = 1
				}
				if factor != 1 {
					brightness = factor
				}
			}
			// THE TWO MARK PASSES (1002; MAGIC-MARK-059). The original's
			// unit draw walks the actor's mark array once BEFORE
			// dispatching its own sprite, drawing only records with
			// `depth > 0`, and once after, drawing only the rest. Doing it
			// here, inside the band's own order, is what keeps a mark
			// behind its actor and still in front of whatever the actor
			// itself is in front of: a separate pass over the finished
			// band could only put every mark above every unit.
			if ref.Index < len(marked) {
				back, front = v.entityMarkRects(marked[ref.Index])
			}
		case terrain.PlaneSack:
			// No art switch and no bundle guard: a sack is world content, drawn
			// whenever it resolves, exactly as an entity is. The index guard mirrors
			// the other three arms even though sackLayer already excludes an
			// out-of-range frame and an ungridded cell — DepthOrder's own index is
			// built from this very slice, so the guard here is defensive rather than
			// reachable, on drawStructuresFlat's and the object arm's own precedent.
			if ref.Index >= len(sacks) {
				continue
			}
			p := sacks[ref.Index]
			order = artOrder{phase: artMainCell, cell: p.Cell, rank: 2, index: ref.Index, part: 1}
			f, rect = p.Frame, p.Rect()
		case terrain.PlaneStructure:
			if !v.showStructureArt || ref.Index >= len(structures) {
				continue
			}
			p := structures[ref.Index]
			order = artOrder{phase: artMainCell, cell: p.Cell, index: ref.Index, part: 1}
			if v.structurePickGate(InspectionSubject{Kind: InspectionStructure, ID: p.StructureID}, p.Cell.X, p.Cell.Y) {
				inspection = InspectionSubject{Kind: InspectionStructure, ID: p.StructureID}
			}
			// The EARLY pass already drew it. Skipping here, rather than
			// filtering the order, is what keeps "no structure is drawn twice" a
			// property of the two passes reading one predicate.
			if p.Class != nil && p.Class.Flat {
				continue
			}
			f, rect = p.Frame, p.Rect()
		default:
			if !v.showStaticArt || ref.Index >= len(objects) {
				continue
			}
			p := objects[ref.Index]
			order = artOrder{phase: artMainCell, cell: p.Cell, rank: 4, index: ref.Index, part: 1}
			f, rect = p.Frame, p.Rect()
		}
		// THE SECOND MARK PASS RUNS WHATEVER THE SPRITE DOES. An entry whose
		// own art is missing or culled still contributes its `depth <= 0`
		// records, each already culled on its own rectangle: the original's
		// second walk is not conditional on the sprite dispatch either, and a
		// mark that reaches the view while its actor's rectangle does not is
		// exactly the case a shared cull would drop.
		start := len(out)
		out = append(out, back...)
		if r, _ := spriteScreenRect(rect, v.cam); f != nil {
			if f.Boundary != nil {
				boundary = f.Boundary
			}
			if ref.Kind == terrain.PlaneSack {
				boundary = v.sackBoundaries[f]
			}
			out = append(out, v.spritePair(staticScreenRect{screenRect: r, Frame: f, Mirror: mirror,
				Stone: stone, Translucent: translucent, Brightness: brightness, ShadeZero: shadeZero, Inspection: inspection,
				Tint: v.tintOf(inspection)}, boundary)...)
		}
		if ref.Kind == terrain.PlaneSack && v.sackHighlight && f != nil {
			if r, ok := spriteScreenRect(rect, v.cam); ok {
				out = append(out, staticScreenRect{screenRect: r, Frame: f, SackOutline: true})
			}
		}

		out = append(out, front...)
		for i := start; i < len(out); i++ {
			out[i].order = order
			if out[i].SackOutline {
				out[i].order.phase = artOutline
			}
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].order.before(out[j].order) })
	return out
}

// drawPlane paints the content band planeSprites arranged: nearest-sampled,
// scaled by the camera's zoom, translated to where the cull put it, and
// reflected inside its own rectangle for a mirrored entry.
//
// The GeoM is the sprite pass's own, so an entity is submitted at exactly the
// transform it was submitted at before it joined this band. Filter is Nearest:
// art is blocky at high zoom by choice, and any smoothing would bleed a
// transparent edge pixel into its neighbour's colour.
//
// It draws between the terrain and every diagnostic marker.
func (v *Viewer) drawPlane(target imageTarget) {
	for _, s := range v.planeSprites() {
		v.drawPlaneSprite(target, s)
	}
}

func (v *Viewer) spritePair(base staticScreenRect, boundary *terrain.StaticFrame) []staticScreenRect {
	var out []staticScreenRect
	if spriteVisible(base.screenRect, v.cam) {
		out = append(out, base)
	}
	if !v.graphics.Smoothing || boundary == nil {
		return out
	}
	sibling := base
	sibling.Frame, sibling.Effect = boundary, nil
	sibling.Inspection = InspectionSubject{}
	sibling.W, sibling.H = float64(boundary.Width)*v.cam.Zoom, float64(boundary.Height)*v.cam.Zoom
	if base.Mirror {
		sibling.X += base.W - sibling.W
	}
	sibling.Translucent = true
	sibling.Unlit = base.Effect != nil
	if spriteVisible(sibling.screenRect, v.cam) {
		out = append(out, sibling)
	}
	return out
}

func (v *Viewer) drawPlaneSprite(target imageTarget, s staticScreenRect) {
	zoom := v.cam.Zoom
	// An effect-mark entry (1002) draws through the projectile-sheet
	// texture cache instead of the static one. Its rectangle, its
	// transform and its place in the band are the sprite entry's own.
	if s.Effect != nil {
		var op ebiten.DrawImageOptions
		op.Filter = ebiten.FilterNearest
		op.GeoM = spriteGeoM(s, zoom)
		target.DrawImage(v.effectImage(s.Effect), &op)
		return
	}
	if s.Frame.Width <= 0 || s.Frame.Height <= 0 {
		return
	}
	if s.SackOutline {
		v.drawSackOutline(target, s)
		return
	}
	var op ebiten.DrawImageOptions
	op.Filter = ebiten.FilterNearest
	op.GeoM = spriteGeoM(s, zoom)
	if s.Translucent {
		op.ColorScale.ScaleAlpha(0.5)
	}
	if s.Brightness > 0 && s.Brightness != 1 {
		op.ColorScale.Scale(s.Brightness, s.Brightness, s.Brightness, 1)
	}
	row := v.spriteRow()
	if s.ShadeZero {
		row = 0
	}
	if s.Unlit {
		row = spriteUnlit
	}
	img := v.staticImage(s.Frame, row)
	if s.Stone {
		img = v.stoneImage(s.Frame, row)
	}
	if s.Tint.On {
		img = v.tintedImage(s.Frame, s.Stone, s.Tint.Color, s.Tint.Focus, row)
	}
	target.DrawImage(img, &op)
}

func (v *Viewer) stoneImage(f *terrain.StaticFrame, rows ...int) *ebiten.Image {
	key := v.spriteKey(f, rows...)
	if img, ok := v.stoneImages[key]; ok {
		return img
	}
	img := ebiten.NewImageFromImage(v.spritePixels(f, rows...))
	if v.stoneImages == nil {
		v.stoneImages = make(map[spriteTextureKey]*ebiten.Image)
	}
	v.stoneImages[key] = img
	return img
}

// drawStructuresFlat is the EARLY pass: every entry of a class with Flat
// set, drawn BEFORE the static-object layer and therefore before everything
// else on the map (TERR-STRUCT-104).
//
// It is a pass of its own and not a branch of the merge because it is not merged
// with anything: a flat structure is ground decoration and goes under every other
// drawable, whatever row it stands on. Nothing else in this package draws before
// the object layer.
func (v *Viewer) drawStructuresFlat(target imageTarget) {
	if !v.showStructureArt {
		return
	}
	for _, p := range v.structurePlacements() {
		if p.Class == nil || !p.Class.Flat {
			continue
		}
		r, _ := spriteScreenRect(p.Rect(), v.cam)
		if p.Frame != nil {
			for _, s := range v.spritePair(staticScreenRect{screenRect: r, Frame: p.Frame, Tint: v.tintOf(InspectionSubject{Kind: InspectionStructure, ID: p.StructureID})}, p.Frame.Boundary) {
				v.drawPlaneSprite(target, s)
			}
		}
	}
}

// drawSprite paints one frame at its world rectangle, culled and transformed by
// the package's one cull, nearest-sampled and scaled by the camera's zoom.
//
// A frameless entry and a culled one both paint NOTHING and build no texture:
// there is nothing to paint, and ebiten refuses an empty image. The texture comes
// through staticImage, so object and structure art share one cache keyed on frame
// identity and one palette walk.
func (v *Viewer) drawSprite(target imageTarget, f *terrain.StaticFrame, r image.Rectangle, tint spriteTint) {
	if f == nil {
		return
	}
	s, ok := spriteScreenRect(r, v.cam)
	if !ok {
		return
	}
	zoom := v.cam.Zoom
	var op ebiten.DrawImageOptions
	op.Filter = ebiten.FilterNearest
	op.GeoM.Scale(zoom, zoom)
	op.GeoM.Translate(s.X, s.Y)
	if tint.On {
		target.DrawImage(v.tintedImage(f, false, tint.Color, tint.Focus), &op)
		return
	}
	target.DrawImage(v.staticImage(f), &op)
}

// staticImage is the frame's GPU texture, built on its FIRST draw and cached
// under the frame's own pointer identity from then on.
//
// NO TEXTURE EXISTS UNTIL A DRAW ASKS FOR ONE, and the cache map itself is
// the evidence: it is nil until this function is first called, so a viewer
// that has been constructed, queried and culled but never drawn holds no map
// at all, never mind an image. That is what keeps a headless -check run —
// which has no graphics context whatsoever — from needing a GPU to answer
// how many placements a map has. Uploading at construction was rejected for
// exactly this: it makes the one thing -check exists to avoid mandatory.
//
// The KEY IS THE FRAME POINTER AND THE ROW, not the class, not the cell and
// not a copy of the pixels. The builder puts one *StaticFrame into every
// placement of a class, so a forest of two hundred trees uploads one texture
// and looks it up two hundred times. A value key would upload per cell; a
// class key would need this package to learn a class identity it is never
// given.
//
// The pixels come through spritePixels — pkg/render/terrain's own blit
// onto a transparent canvas, lit or not — so the texture and the raster
// tool's output are one palette walk applied twice and cannot disagree per
// pixel. Transparent frame pixels arrive as alpha 0 rather than as black at
// every row, which is what lets a sprite's holes show the terrain beneath.
func (v *Viewer) staticImage(f *terrain.StaticFrame, rows ...int) *ebiten.Image {
	key := v.spriteKey(f, rows...)
	if img, ok := v.staticImages[key]; ok {
		return img
	}
	img := ebiten.NewImageFromImage(v.spritePixels(f, rows...))
	if v.staticImages == nil {
		v.staticImages = make(map[spriteTextureKey]*ebiten.Image)
	}
	v.staticImages[key] = img
	return img
}

// spriteTextureKey is what the texture cache is keyed by from 0044 on: the
// frame, the ramp row its pixels were resolved at, and — from 0100 on —
// the sky tint they were resolved with.
//
// THE ROW BELONGS IN THE KEY BECAUSE THE PIXELS GENUINELY DIFFER, which is
// exactly what separates it from the mirror: a mirrored sprite's pixels are
// identical to its unmirrored ones — the reflection is a GeoM — so 0024
// deliberately did NOT widen this key for one, and nothing here widens it
// for one either.
//
// THE TINT BELONGS IN THE KEY FOR THE SAME REASON THE ROW DOES: 0092 shipped it
// baked into spritePixels' output with no key entry at all, because every band
// carried the same (0,0,0) tint and a constant cannot desynchronise from
// itself (0092 DD-3a named the consequence and deferred it to the story that
// makes the tint vary). That story is this one, so the key widens exactly as
// DD-3a said it would have to: a band's tinted textures can no longer be
// served into a later band that resolves to the same frame and row.
//
// Dropping the cache when the row or the tint changes was rejected: it is a
// second thing to keep true, invisible at the call site, and it re-uploads
// every frame's texture on a toggle or a relight rather than handing back the
// one already there.
type spriteTextureKey struct {
	frame *terrain.StaticFrame
	row   int
	tint  [3]uint8
}

// spriteUnlit is the row slot's value under the unshaded diagnostic. It lies
// outside [0, terrain.SpriteRowCount-1], which terrain.SpriteRow never leaves,
// so a lit key and the unlit one cannot collide.
//
// It is this package's own spelling of "no row" and NEVER reaches a terrain
// call: the unshaded texture is built by RGBA(), which takes no row at all, so
// nothing here relies on how the ramp treats an out-of-range one.
const spriteUnlit = -1

// spriteRow is the lighting state EVERY sprite of this rendered frame is
// drawn at: the fixed daytime sun's own row, or spriteUnlit under the
// diagnostic.
//
// IT READS v.unshaded AND NOT Lit(). Lit() folds two conditions together — the
// diagnostic and whether the altitude grid was usable — and only one of them
// belongs here: a sprite's row comes from the ambient byte and has no dependence
// on relief whatsoever, so a map drawn flat and with unshaded terrain for want
// of altitudes must still light its sprites. Reusing Lit() would make sprite
// colour depend on data that never enters it, invisibly in every map whose
// altitudes are valid.
//
// THE SUN IS v.sun, THE DAY/NIGHT CYCLE'S CACHE, and until 0092 this comment
// read "the fixed daytime one ... this viewer holds no Light and gains no
// setter".
func (v *Viewer) spriteRow() int {
	if v.unshaded {
		return spriteUnlit
	}
	return terrain.SpriteRow(v.sun)
}

// spriteKey is the cache key one frame takes in the viewer's current state:
// the frame, the row, and the tint.
func (v *Viewer) spriteKey(f *terrain.StaticFrame, rows ...int) spriteTextureKey {
	row := v.spriteRow()
	if len(rows) != 0 {
		row = rows[0]
	}
	return spriteTextureKey{frame: f, row: row, tint: v.lightTint()}
}

// spritePixels resolves a frame at the supplied shading row, or the current
// scene row when none is supplied. The unshaded diagnostic bypasses the ramp.
func (v *Viewer) spritePixels(f *terrain.StaticFrame, rows ...int) *image.RGBA {
	row := v.spriteKey(f, rows...).row
	if row == spriteUnlit {
		return f.RGBA()
	}
	return f.RGBALit(v.lightTint(), row)
}

// --- the structure layer ---
//
// The same three pieces the object layer above has, over a different placement
// type and with NOTHING ELSE different: the cull is on the exact world rectangle,
// the transform is the camera's, and a frame becomes a texture through the very
// cache and the very palette walk the object sprites use. Structure art is the
// object layer's own *terrain.StaticFrame, so the frame-keyed cache serves it
// unchanged and a building's sheet uploads once however many cells it covers.
//
// What is NOT here is a second geometry. The entries were placed once, at
// construction, by the render tier's one builder, and this file places what it is
// handed unchanged — no shift, no re-anchoring, and above all no anchor pixel:
// a structure has none, and there is nothing in this file that could supply one.

// structureScreenRect is one structure entry placed in the current view: where
// its frame lands on screen and how big it is there, beside the frame itself.
//
// It carries no Mirror. Reflection is a unit-layer fact and no structure entry
// can hold one — the builder never sets one, because the type has no such field —
// so a mirror here would be a switch nothing can ever move.
type structureScreenRect struct {
	screenRect
	Frame *terrain.StaticFrame
}

// structureScreenRects is the structure layer's visibility test and camera
// transform in one pure pass: every entry whose frame meets the view, in the
// order the builder produced them.
//
// It is a pure function of THE ENTRY LIST AND THE CAMERA and of nothing
// else, so the whole of it is reachable in a test with no graphics context
// at all.
//
// THE CULL IS ON THE EXACT WORLD RECTANGLE, and for a structure that matters more
// than it does for an object: the overhang stacks whole cells ABOVE the back row,
// so a building whose rectangle is below the view can still have its roof inside
// it. Each entry is one tile-sized frame with its own top-left, so culling entry
// by entry keeps exactly the rows that are visible and drops the rest — which is
// what makes a 3x2 building with a five-row grid cost five draws at the top of the
// view instead of fifteen.
//
// AN ENTRY WITH NO FRAME IS SKIPPED HERE. The builder places one for a grid index
// its sheet does not hold: the entry is real and the census counts it, but there
// is nothing to draw and nothing to build a texture from.
func structureScreenRects(places []terrain.StructurePlacement, cam *camera.Camera) []structureScreenRect {
	var out []structureScreenRect
	for _, p := range places {
		if p.Frame == nil {
			continue
		}
		r, ok := spriteScreenRect(p.Rect(), cam)
		if !ok {
			continue
		}
		out = append(out, structureScreenRect{screenRect: r, Frame: p.Frame})
	}
	return out
}
