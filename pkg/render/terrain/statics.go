package terrain

import (
	"image"
	"image/color"
)

// StaticPixel is one pixel of a decoded object frame: a palette index that is
// meaningful only when Opaque is true, and a see-through hole otherwise.
//
// Transparency is STRUCTURAL — the decoded frame carries it per pixel and no
// palette index is reserved to mean it — so an opaque pixel may legitimately
// hold index 0 (spec, "Class to sprite frame"). This mirrors spr256.Pixel
// rather than reusing it because that package is a tier this one may not
// import; the loader converts.
type StaticPixel struct {
	Index  uint8
	Opaque bool
}

// StaticFrame is one drawable object frame: a Width x Height grid of pixels in
// row-major order, plus the palette those indices select in. Pixels holds
// exactly Width*Height elements.
//
// The palette rides on the FRAME rather than on the class or the set because
// a blit takes a frame alone: an opaque pixel is drawn in the colour the
// SHEET the frame came out of selects (spec, "Blit"), so a frame that could
// be blitted without carrying its own palette would be a frame that could be
// blitted with somebody else's.
//
// The palette is a fixed 256-entry array, not a slice: an index is a uint8, so
// every index a pixel can carry indexes it, and a consumer needs neither a
// bounds test nor a fallback colour. Its entries are the sheet's colours at
// full opacity — the alpha channel here is not a transparency channel and
// nothing may read it as one, StaticPixel.Opaque being the only answer to which
// pixels are holes.
type StaticFrame struct {
	Width   int
	Height  int
	Pixels  []StaticPixel
	Palette [256]color.RGBA
}

// StaticClass is one object class as this layer draws it: the canvas geometry
// its anchor is measured in, and the one frame the class's Index selects
// (TERR-SPR-042).
//
// Width/Height and CenterX/CenterY are the registry's own key spellings carried
// across as plain ints — the canvas the class declares, and the pixel of it
// that lands on a cell's ground point. They are NOT the frame's size: a frame
// that does not fill its class's canvas is the ordinary case rather than the
// exotic one (TERR-SPR-043, SPR256-FRAME-023), which is why StaticAnchor takes
// both and why this type carries neither of the frame's dimensions itself.
//
// Frame is nil for a class the loader resolved but could not draw — an absent,
// undecodable or palette-less sheet, or an Index outside it. That is a skip and
// never an error (spec, error cases): the layer draws what it can. A nil class
// in a StaticSet and a resolved class with a nil Frame are therefore different
// answers, and are counted apart rather than together.
//
// Frames is the class's WHOLE sheet, and Index the frame of it that Frame
// is. A cycle names its frames as offsets from Index, so a class that draws
// at all must carry every frame its timeline can reach — and since the
// loader's per-path memo is shared, two classes naming one sheet receive ONE
// slice, pointer for pointer, and the frame-keyed texture cache uploads it
// once.
//
// FRAME STAYS THE DRAWABILITY ANSWER, and the two agree by the loader's own
// arithmetic rather than by a rule stated here: the whole-sheet conversion fails
// only on the exclusions that exclude every frame, and the single-frame one adds
// only the range test, so Frame is non-nil exactly when Frames is non-nil and
// Index lies inside it. The census that counts an artless class apart from an
// unnamed one is therefore untouched.
//
// Timeline is the class's animation cycle already expanded — the values a
// step selects, added to Index — and ITS LENGTH IS THE PERIOD. An empty
// timeline is a class with no cycle. The expansion happens in the tier that
// owns registries; nothing here re-derives it, and this tier never learns
// that a timeline came from two arrays.
type StaticClass struct {
	// Dead is the resolved DeadObject class drawn at sheet frame0 on bit13.
	// It is a detached nonanimated variant; nil means no admitted dead form.
	Dead    *StaticClass
	Width   int
	Height  int
	CenterX int
	CenterY int
	// FireObject is the ambient selector carried by the object registry. It is
	// compared only: -2 is the crow population, >=0 the potential bird arm.
	FireObject int32
	Frame      *StaticFrame

	Frames   []*StaticFrame
	Index    int
	Timeline []int
}

// StaticSet is the loaded object-class bundle, indexed by the PLACEMENT BYTE
// a map's object grid holds — not by class ID.
//
// Classes[b] is what byte b draws, or nil for a byte naming no loaded class;
// Classes[0] is nil always, 0 being *no object*. Keying by the byte is what
// keeps the b-1 offset in the one place that decoded it — pkg/data's own
// lookup, which the loader walks 1..255 through once — so this tier never
// learns the registry's identity convention and the per-cell path is an array
// read.
//
// A zero StaticSet is a legal, empty one: this package establishes no invariant
// over the field, because it is not this package that fills it.
type StaticSet struct {
	Classes [256]*StaticClass
}

func StaticAnchor(col, row, canvasW, canvasH, centerX, centerY, frameW, frameH, lift, originY int) (destX, destY, anchorX, anchorY int) {
	anchorX = centerX - canvasW/2 + frameW/2
	anchorY = centerY - canvasH/2 + frameH/2

	destX = col*CellSize + CellSize/2 - anchorX
	destY = row*CellSize + CellSize/2 - anchorY - lift - originY
	return destX, destY, anchorX, anchorY
}

// StaticPlacement is one cell's object sprite, already placed: the cell it
// came from, the frame's top-left in world pixels, the frame's own anchor
// pixel, and the frame itself.
//
// Cell is (col, row) in the X/Y spelling pkg/game.MarkerCells already uses
// for a marker's anchor cell, so the two layers name a cell the same way
// even though neither derives its position from the other. TopLeft and
// Anchor are StaticAnchor's own four returns carried across as two points
// — the frame's anchor pixel is stored rather than recomputed, so the
// halving that produces it exists once in the tree.
//
// Frame is a POINTER INTO THE SET, never a copy. Two cells holding the same
// placement byte therefore carry the same pointer, which is what lets the
// window cache one GPU texture per frame under frame identity instead of one
// per cell. A placement consequently does not own its pixels: everything
// downstream of the builder reads through this pointer and nothing writes
// through it.
//
// World pixels are native pixels — one cell is CellSize of them in both
// renderers — so a placement built once places the same art at the same
// point in the raster and in the window.
//
// Class is the object class this placement drew from, or nil for a placement
// the object layer did not make. It is what lets a per-counter pass
// re-select this cell's frame WITHOUT rebuilding the list: the class carries
// the sheet, the Index and the timeline, and the placement carries the
// ground point they are measured against.
//
// THE ZERO VALUE IS THE GROUND TIER, which is what makes this field free: a
// sack placement, and every placement built before this existed, says nothing
// and lands where a sack lands. Only an entity's builder names one.
type DepthTie int8

// The three tiers, top to bottom on one cell, and they are the OWNER'S OWN
// RULING: «на клетке стоит герой или юнит — он
// выше всех. если есть мешок, то он под
// героем. если есть труп, то он под
// мешочком.» A living unit above everything, the sack under it, the
// corpse under the sack.
//
// IT IS AUTHORED AND IS NOT A REPRODUCTION. His ruling is the spec; it is
// his as author, and the tiers are ours.
//
// ONLY THE DEAD TAKE TieCorpse. A downed unit — exactly zero health, "a body
// in the way", not yet dead and having dropped nothing — takes TieUnit with
// the living, because the picture the ruling describes is a body that has
// already dropped its sack, and the narrowest reading that satisfies it is the
// right one for a tie-break nothing decodes.
const (
	TieCorpse DepthTie = -1
	TieGround DepthTie = 0
	TieUnit   DepthTie = 1
)

type StaticPlacement struct {
	Cell     image.Point
	TopLeft  image.Point
	Anchor   image.Point
	Frame    *StaticFrame
	Class    *StaticClass
	Mirror   bool
	DepthTie DepthTie
}

// Rect returns the placement's exact world rectangle: the half-open
// [TopLeft, TopLeft+(Width, Height)) its frame occupies.
//
// EXACT is the load-bearing word. The visibility test culls on this
// rectangle and not on the ground cell, because an object's art routinely
// reaches far above the cell it stands on: a tall sprite whose ground cell
// is below the view but whose crown enters it must be kept, and a cull by
// cell — or by the camera's tile band — drops exactly that one.
//
// A placement with no frame yields the zero rectangle, which is empty and
// therefore intersects nothing. The accessor is total on a zero value rather
// than a defence against one: the builder never produces a frameless placement.
func (p StaticPlacement) Rect() image.Rectangle {
	if p.Frame == nil {
		return image.Rectangle{}
	}
	return image.Rectangle{
		Min: p.TopLeft,
		Max: p.TopLeft.Add(image.Point{X: p.Frame.Width, Y: p.Frame.Height}),
	}
}

// Ground returns the world point this sprite's own geometry puts the cell's
// ground point at: TopLeft + Anchor, the two values the placement already
// carries.
//
// It SUMS and recomputes nothing — no class field, no frame size, no cell
// arithmetic, and above all no marker geometry. What is compared has to be
// what was DRAWN, or the comparison measures one derivation twice instead of
// two once.
func (p StaticPlacement) Ground() image.Point {
	return p.TopLeft.Add(p.Anchor)
}

// StaticCounts is one build's census: the placements made, and the two skip
// kinds counted apart.
//
// Placed equals the length of the list built beside it. NoClass counts cells
// whose non-zero byte named no loaded class; NoFrame counts cells whose byte
// named a class the loader resolved but could not draw — an absent, undecodable
// or palette-less sheet, or an Index outside it. Neither is an error: the layer
// draws what it can and never fails a run on data it cannot use (spec, error
// cases).
//
// The two are kept apart because they are different facts about an install. A
// byte naming no class is MAP data pointing at a class this registry does not
// hold; a class with no frame is ART we could not decode. A corpus census that
// added them could not say which it had found, and the corpus criterion (AC-8)
// exists to record exactly that.
//
// Byte 0 is counted nowhere. It is *no object* — the absence of a placement, not
// the skipping of one — so the three counters together are a census of the cells
// that named something, and never of the map.
//
// The record is geometry-independent: which cells resolve is a question
// about bytes and classes alone, and altitudes reach only the anchors. So a
// viewer that builds a flat and a displaced list of the same map gets one
// census, and one record serves both. Animated is how many of those
// placements have an OPEN CYCLE — the length of the animated subset the
// same build returned. It is filled by the builder and by nothing else, so
// the number a front-end prints is the number the build actually produced
// rather than a second walk's opinion of it.
//
// It is a subset of Placed and never a fourth skip kind: an animated placement
// was placed. On shipped data it is 0, because no cell opens the gate, and that
// is exactly what makes it worth reporting — it and the diagnostic are the only
// witnesses that the cycle exists at all.
type StaticCounts struct {
	Placed   int
	NoClass  int
	NoFrame  int
	Animated int
}

func StaticPlacements(g Grid, set *StaticSet, lift func(col, row int) int, originY int, gateAll bool) ([]StaticPlacement, StaticCounts, []int) {
	var counts StaticCounts
	if set == nil || g.Width <= 0 || g.Height <= 0 || len(g.Overlay) != g.Width*g.Height {
		return nil, counts, nil
	}

	var out []StaticPlacement
	var animated []int
	for row := 0; row < g.Height; row++ {
		for col := 0; col < g.Width; col++ {
			b := g.Overlay[row*g.Width+col]
			if b == 0 {
				continue
			}
			// The set is keyed by the placement byte itself, so this is an array read
			// over the whole uint8 domain: the b-1 offset stayed in pkg/data, where
			// the registry's identity convention is decoded, and no bounds test is
			// possible here.
			c := set.Classes[b]
			if len(g.Tiles) == g.Width*g.Height && g.Tiles[row*g.Width+col]&0x2000 != 0 && c != nil && c.Dead != nil {
				c = c.Dead
			}
			if c == nil {
				counts.NoClass++
				continue
			}
			f := c.Frame
			if f == nil {
				counts.NoFrame++
				continue
			}

			h := 0
			if lift != nil {
				h = lift(col, row)
			}
			destX, destY, anchorX, anchorY := StaticAnchor(
				col, row, c.Width, c.Height, c.CenterX, c.CenterY, f.Width, f.Height, h, originY)

			if ObjectCycleOpen(g, col, row, len(c.Timeline), gateAll) {
				animated = append(animated, len(out))
			}
			out = append(out, StaticPlacement{
				Cell:    image.Point{X: col, Y: row},
				TopLeft: image.Point{X: destX, Y: destY},
				Anchor:  image.Point{X: anchorX, Y: anchorY},
				Frame:   f,
				Class:   c,
			})
			counts.Placed++
		}
	}
	counts.Animated = len(animated)
	return out, counts, animated
}

// animGateBits are the two top bits of a tile word the cycle's gate reads —
// bits 15 and 14, which splitTileWord masks off and no other reader consults
// (TERR-TILE-044).
//
// BOTH must be set in the OR of four words for a cell's cycle to open. The map
// files set neither bit in 880,704 shipped cells; `TERR-TILE-079` identifies
// them as runtime fog state and names the reveal writer. This implementation
// keeps the equivalent three states in the viewer's fog plane, so this decoded
// tile-word reader remains for the direct instrument path.
const animGateBits = 0xc000

// ObjectCycleOpen reports whether the cell (col, row) may cycle: the class's
// period is non-zero AND the cell's own tile word ORed with its three
// neighbours to the east, south and south-east has both of animGateBits set.
//
// THE GATE IS A PROPERTY OF THE CELL, not of the class alone. A class with a
// cycle standing on a cell that does not open it draws its Index at every
// counter, which is what lets a map decide whether its own foliage moves.
//
// A CELL ON THE LAST COLUMN OR THE LAST ROW IS CLOSED, and that is OURS by
// choice rather than decoded: the engine reads its three neighbours unguarded
// and so reads past its own grid at the far edge, where there is no defined
// behaviour to reproduce. Closing is the answer that adds no cycle nobody asked
// for. A grid whose tile layer is absent or the wrong length is closed for the
// same reason, and so is a non-positive period, whatever the tile words hold.
//
// AnimGateAll opens every non-zero period whatever the tile words hold;
// AnimGateTiles asks the decoded test. The game uses the former to build the
// stable candidate list before applying live fog. Neither opens period 0: a
// class with no cycle has nothing to step through.
//
// It is pure: it reads the grid's dimensions and tile words and writes nothing,
// so two evaluations of one cell agree and no build order can move it.
func ObjectCycleOpen(g Grid, col, row, period int, gateAll bool) bool {
	if period <= 0 {
		return false
	}
	if gateAll {
		return true
	}
	if col < 0 || row < 0 || col+1 >= g.Width || row+1 >= g.Height {
		return false
	}
	if len(g.Tiles) != g.Width*g.Height {
		return false
	}
	i := row*g.Width + col
	word := g.Tiles[i] | g.Tiles[i+1] | g.Tiles[i+g.Width] | g.Tiles[i+g.Width+1]
	return word&animGateBits == animGateBits
}

// The two values StaticPlacements' gateAll parameter takes, so a call site
// says which gate it is asking for rather than passing a bare boolean —
// the withHeights/withoutHeights shape one file over.
const (
	// AnimGateTiles is the decoded gate: the four-corner tile-word test.
	AnimGateTiles = false

	// AnimGateAll builds every non-zero-period candidate. The game then applies
	// its live FogVisible gate on each draw; diagnostics can use the same full
	// population without manufacturing frames for a zero-period class.
	AnimGateAll = true
)

// AnimateStatics is the per-counter pass: the placement list as it draws at
// one counter value with the animation switch in one state.
//
// IT WRITES INTO A CALLER-OWNED BUFFER and returns what to draw from. dst is
// reused between calls — the window keeps one scratch slice for the life of the
// viewer — so a rendered frame allocates nothing after the first. A fresh slice
// per frame would be hundreds of kilobytes a second to move a handful of
// entries.
//
// WHEN NOTHING CAN CHANGE IT RETURNS THE BUILT LIST ITSELF, untouched and
// unaliased: with the switch on and an empty subset there is no cell to
// re-select, so an ordinary map draws the very slice the builder produced
// and this pass costs one comparison. A caller must therefore NOT assume the
// answer is its own buffer.
//
// The two walks are different sizes because the two arms are:
//
//   - switch ON: only the open-cycle subset can move, so only it is walked, and
//     a map with thousands of trees and no open cell pays nothing.
//   - switch OFF: EVERY placement collapses to sheet frame 0 whatever its Index
//     says, so every placement is walked. That arm is counter-independent — it
//     is the same picture at every counter — but it is decided at the DRAW and
//     never at the build, so turning the switch changes what is painted and
//     never what was placed.
//
// EACH RE-SELECTED PLACEMENT IS RE-ANCHORED FROM ITS OWN GROUND POINT: the
// new anchor pixel is computed from the class canvas and the DRAWN frame's
// size, and TopLeft becomes Ground() - Anchor. So a cycle whose frames
// differ in size stands each of them on the same world point the build
// produced, and Rect() is the drawn frame's own rectangle at every counter.
// It needs no lift, no origin and no projection and takes none —
// StaticAnchor's signature requires all three, so calling it here would put
// a projection into a per-frame pass and let it disagree with the build
// about a cell's height.
//
// Draw order is the built list's order throughout: entries are patched in place,
// never appended, moved or dropped.
func AnimateStatics(dst, places []StaticPlacement, animated []int, counter uint32, animate bool) []StaticPlacement {
	return animateStatics(dst, places, animated, counter, animate, nil)
}

// AnimateVisibleStatics is AnimateStatics with a live per-cell visibility
// gate. A cycle-capable placement changes frame only while visible reports
// true for its cell. A closed placement keeps the class's built Index frame;
// the placement itself is never dropped here.
//
// A nil visible function opens every cycle and is equivalent to
// AnimateStatics. If no candidate is visible, the built slice is returned
// unchanged and no scratch buffer is populated.
func AnimateVisibleStatics(dst, places []StaticPlacement, animated []int, counter uint32, animate bool,
	visible func(col, row int) bool) []StaticPlacement {
	return animateStatics(dst, places, animated, counter, animate, visible)
}

func animateStatics(dst, places []StaticPlacement, animated []int, counter uint32, animate bool,
	visible func(col, row int) bool) []StaticPlacement {
	if animate && len(animated) == 0 {
		return places
	}
	if animate && visible != nil {
		open := false
		for _, i := range animated {
			if i >= 0 && i < len(places) && visible(places[i].Cell.X, places[i].Cell.Y) {
				open = true
				break
			}
		}
		if !open {
			return places
		}
	}
	dst = append(dst[:0], places...)
	if !animate {
		for i := range dst {
			animateStatic(&dst[i], counter, false, false)
		}
		return dst
	}
	for _, i := range animated {
		// The subset is the builder's own and indexes the list it was built
		// beside; the guard keeps the pass total for a caller that pairs a
		// subset with a different list.
		if i < 0 || i >= len(dst) {
			continue
		}
		if visible != nil && !visible(dst[i].Cell.X, dst[i].Cell.Y) {
			continue
		}
		animateStatic(&dst[i], counter, true, true)
	}
	return dst
}

// animateStatic re-selects and re-anchors one placement in place. A placement
// with no class or no sheet is left exactly as the builder made it: the unit
// layer's own placements carry neither, and nothing here may move one.
//
// The two halvings below are StaticAnchor's first two lines, deliberately
// not a call into it — its signature requires the lift and the vertical
// origin, which are the build's business and must not enter a per-frame
// pass.
func animateStatic(p *StaticPlacement, counter uint32, animate, open bool) {
	c := p.Class
	if c == nil || len(c.Frames) == 0 {
		return
	}
	f := c.Frames[SelectObjectFrame(c.Timeline, c.Index, len(c.Frames), p.Cell.X, p.Cell.Y, counter, animate, open)]
	if f == nil {
		return
	}
	ground := p.Ground()
	p.Anchor = image.Point{
		X: c.CenterX - c.Width/2 + f.Width/2,
		Y: c.CenterY - c.Height/2 + f.Height/2,
	}
	p.TopLeft = ground.Sub(p.Anchor)
	p.Frame = f
}
