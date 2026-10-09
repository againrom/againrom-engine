package terrain

import (
	"fmt"
	"image"
	"sort"
)

// StructureClass is one structure class as this layer draws it: the art
// rectangle it covers, the sheet grid its frames are addressed in, and the two
// flags the draw passes read (spec, "The art rectangle", "The sheet and its
// grid").
//
// TileWidth x TileHeight is the ART rectangle in cells, running right and down
// from the anchor cell — whatever any other table says about which of those cells
// a building occupies (TERR-STRUCT-101). FullHeight is the sheet grid's row
// count: the grid is TileWidth columns by FullHeight rows, and its top
// FullHeight-TileHeight rows are the OVERHANG that stacks above the back row.
//
// Frames is the class's WHOLE sheet in grid order — image cell (k, c) at index
// k*TileWidth + c — and it is nil for a class the loader resolved but could not
// draw. That is a skip and never an error (spec, error cases). A nil class in a
// StructureSet and a resolved class with no frames are different answers and are
// counted apart.
//
// VariableSize marks a structure whose rectangle comes from its placement rather
// than the registry. VariableLayout names the selectors this renderer actually
// knows; an unknown selector is still skipped and counted instead of guessed.
//
// Flat selects the EARLY pass — a class with it set draws before the static-object
// layer rather than merged with it (TERR-STRUCT-104) — and Indestructible refuses
// the ruin block to the destruction diagnostic (TERR-STRUCT-102). Both are the
// registry's own scalars carried across as booleans: this tier never sees the
// int32 and cannot come to read one of them as a count.
type StructureClass struct {
	// Inspection metadata is presentation-only. Picture has no tier axis.
	ID            int32
	Name, Picture string
	Portrait      *image.RGBA
	Selection     image.Rectangle

	TileWidth  int
	TileHeight int
	FullHeight int

	Frames []*StaticFrame

	VariableSize   bool
	VariableLayout VariableStructureLayout
	Flat           bool
	Indestructible bool
	Usable         bool

	// LightRadius and LightPulse are structures.reg's own decoded scalars
	// (REG-STR-040), carried across unconsumed until the light-source pass
	// (hotfix, owner report, DIV-1313): nonzero on exactly FIVE of the 66
	// classes in both roots — ID 12 "Tower 1", ID 13 "Tower 2", ID 15
	// "Well 2", ID 16 "Well 3" and ID 53 "Campfire" — and zero on every other
	// class, including the other tower-named ones (Guard Tower, Skrakan
	// Tower, Mage's tower). This was misstated as "exactly two" (the two
	// towers alone) before the owner's own observation of the live game
	// confirmed the well and the campfire glow as well (DIV-1313, amended).
	// What unit either scalar is stored in, and how the original consumed
	// them, remains Unknown; this tier and its one reader treat LightRadius
	// as a plain selector — nonzero means "this class glows" — and as the
	// radius that selector's own light mask dilates a structure's footprint
	// by, in whole cells (owner observation, not a decoded consumer). Reading
	// LightPulse at all records that the registry distinguishes these five
	// classes on TWO independent keys, not merely one.
	LightRadius int
	LightPulse  int

	// ShadowY is the class's own shadow pivot: the pixel height, measured from
	// a strip's own dstY, at which the shear StructureShadowShift computes is
	// zero — where the structure's silhouette actually meets the ground.
	ShadowY int

	// Timeline is the class's animation cycle already expanded — the phase
	// values a counter selects — and ITS LENGTH IS THE PERIOD. An EMPTY
	// timeline is a class that does not animate, whatever its Phases scalar
	// said: the gate is conjoined at load, so a class failing any part of it
	// arrives here indistinguishable from one that never spelled an animation
	// key.
	//
	// Rank is the mask read once: for each grid index, how many LIVE cells precede
	// it, or -1 for a cell the mask retires with a '-'. Its length is GridCells()
	// exactly when the class animates, and 0 otherwise — the mask's length is
	// validated at load and nowhere else, because a mask of the wrong length has
	// no correct interpretation and mis-ranks every live cell after the first
	// divergence while failing nowhere (seam 2).
	//
	// Live is how many cells the mask leaves live, which is the animation
	// block's stride per phase. A dead cell and a live one are one array read
	// apart at draw time and the mask string is never re-scanned per frame.
	Timeline []int
	Rank     []int
	Live     int
}

// VariableStructureLayout is a decoded selector for a variable-size sheet.
// Only the shipped vertical wooden bridge is established today. Its nine cells
// are top/middle/bottom crossed with left/centre/right.
type VariableStructureLayout uint8

const (
	VariableStructureUnsupported VariableStructureLayout = iota
	VariableStructureVerticalNinePatch
)

// GridCells is the sheet grid's own cell count, TileWidth*FullHeight — the length
// of the base block, the modulus every grid index lies under, and the length the
// animation mask is held to exactly (spec, "Animation"; seam 2).
//
// It is a method rather than an expression repeated at each site because the two
// extents it multiplies are the two a reader is most likely to transpose:
// TileHeight is the RECTANGLE's row count and never the grid's.
func (c *StructureClass) GridCells() int { return c.TileWidth * c.FullHeight }

// drawable reports whether this class can contribute an entry at all: it is not
// one of the excluded subclasses, its three extents are positive, and its sheet
// resolved to at least one frame.
//
// It is the builder's own guard and not a second copy of the loader's
// exclusion list: the loader refuses each of these already, so on a loaded
// bundle this is never the reason a class draws nothing. It is what keeps
// the builder TOTAL over a hand-built bundle, which is exactly the shape the
// tests place from.
func (c *StructureClass) drawable() bool {
	if c == nil || len(c.Frames) == 0 {
		return false
	}
	if c.VariableSize {
		return c.VariableLayout == VariableStructureVerticalNinePatch && len(c.Frames) == 9
	}
	return c.TileWidth > 0 && c.TileHeight > 0 && c.FullHeight > 0
}

// frame answers the sheet frame at a grid or block index, or nil for an
// index the sheet does not hold. Nothing in this package indexes Frames any
// other way, so no selection can leave the sheet.
func (c *StructureClass) frame(i int) *StaticFrame {
	if i < 0 || i >= len(c.Frames) {
		return nil
	}
	return c.Frames[i]
}

// StructureSet is the loaded structure-class bundle, indexed by the
// PLACEMENT KEY'S LOW BYTE — not by the registry's class ID.
//
// Classes[b] is what the low byte b draws, or nil for a byte naming no loaded
// class. Classes[0] is nil always: the registry's ID domain is 1..66, so no class
// can be reached through byte 0. Keying by the byte is what keeps the registry's
// identity convention in the package that decoded it — the loader walks 1..255
// through pkg/data's own lookup once — so this tier never learns it and the
// per-placement lookup downstream is an array read that cannot go out of bounds.
//
// A zero StructureSet is a legal, empty one, exactly as a zero StaticSet is: this
// package establishes no invariant over the field, because it is not this package
// that fills it.
type StructureSet struct {
	Classes [256]*StructureClass
}

// StructureRecord is one type-4 placement record as this layer reads it: the
// stored fixed-point anchor in 1/256 of a cell, and the key naming its class
// (spec, "The placement").
//
// It carries PLAIN UNSIGNED INTEGERS and not a map record, for AnchorCell's own
// reason: no format type crosses into the render tier. VariableWidth and
// VariableHeight are the two decoded extent bytes of the 0x21 extension; zero
// means the placement did not supply an author-sized rectangle.
//
// Key is the WHOLE stored key and is masked to its low byte at the one site
// that resolves a class. Storing the low byte here instead would put the
// registry's identity convention into the map layer, where a later reader
// could not tell a masked key from a stored one.
type StructureRecord struct {
	ID   uint32
	X, Y uint32
	Key  uint32

	VariableWidth  int
	VariableHeight int
}

// StructurePlacement is ONE IMAGE ROW of one rectangle cell, already placed:
// the rectangle cell it belongs to, the frame's top-left in world pixels,
// the grid index that row's frame is addressed by, the class, and the frame
// itself.
//
// THE UNIT OF THE LIST IS THE BLIT, NOT THE CELL. A structure is drawn once
// per rectangle cell and each of those draws is a VERTICAL STRIP of image
// rows (spec, "The strip"), so a rectangle cell of n image rows contributes
// n entries.
//
// THERE IS NO ANCHOR FIELD, and that absence is this story's central finding
// made structural.
//
// Frame is a POINTER INTO THE SET, never a copy, exactly as a
// StaticPlacement's is — and it is the object layer's own frame type, so
// the window's frame-keyed texture cache uploads structure art through the
// existing cache with no change to it. It is nil for a grid index the sheet
// does not hold, which draws nothing and is not a skip: the entry was
// placed, and the census counts it.
//
// GridIndex is the SHEET GRID's index k*TileWidth + c, not an index into Frames:
// the animation and ruin blocks are addressed FROM it (spec, "The three blocks"),
// so the per-counter pass needs the grid index the build resolved and never the
// frame it happened to select.
type StructurePlacement struct {
	StructureID uint32
	Cell        image.Point
	TopLeft     image.Point
	GridIndex   int
	Class       *StructureClass
	Frame       *StaticFrame
}

// Rect returns the placement's exact world rectangle: the half-open
// [TopLeft, TopLeft+(Width, Height)) its frame occupies.
//
// It is StaticPlacement.Rect's shape and reason: the visibility test culls
// on this rectangle and not on the cell, because a structure's overhang
// reaches whole cells above the row it is anchored on, and a cull by cell
// drops exactly the building whose crown enters the view while its ground is
// below it.
//
// A placement with no frame yields the zero rectangle, which is empty and
// therefore intersects nothing.
func (p StructurePlacement) Rect() image.Rectangle {
	if p.Frame == nil {
		return image.Rectangle{}
	}
	return image.Rectangle{
		Min: p.TopLeft,
		Max: p.TopLeft.Add(image.Point{X: p.Frame.Width, Y: p.Frame.Height}),
	}
}

// StructurePlacementCounts is the PER-PLACEMENT half of one build's census:
// the type-4 records that drew, and the three skip kinds counted apart.
type StructurePlacementCounts struct {
	Drawn        int
	NoClass      int
	Undrawable   int
	VariableSize int
}

// StructureCellCounts is the PER-CELL half: what the drawn placements'
// rectangles came to.
//
// Strips counts rectangle cells INSIDE the map extent — one strip per cell, drawn
// as a vertical run of image rows. Frames counts the entries those strips
// produced, and Overhang how many of those entries come from the grid's top
// FullHeight-TileHeight rows, which stack above the back row rather than standing
// on a cell of their own. Outside counts rectangle cells beyond the map extent,
// which contribute no entry at all.
type StructureCellCounts struct {
	Strips   int
	Frames   int
	Overhang int
	Outside  int
}

// StructureCounts is one build's whole census, the two levels kept in
// separate structs so neither can be added to the other by accident.
//
// It is filled BY THE BUILDER, beside the list it counts, so a printed number is
// the number that build produced rather than a second walk's opinion of it. No
// tool recounts a list, and the two levels answer different questions: one is
// about the map's records, the other about the cells they cover.
type StructureCounts struct {
	Placements StructurePlacementCounts
	Cells      StructureCellCounts
}

// StructurePlacements turns a map's type-4 placement records, the loaded
// classes and one geometry into the draw list: one entry per image row of
// each rectangle cell, ordered by rectangle cell — rows ascending, columns
// DESCENDING, ties in map record order — with the census of what it
// skipped.
//
// It is the layer's ONE placement path. Both renderers place from the list
// it returns, unchanged, which is what makes it impossible for them to
// disagree about where a building stands. It is pure and GPU-free: it reads
// the grid, the set and the corner accessor and writes nothing, so it runs
// under -check and in a test with neither a window nor a game install.
//
// THE ANCHOR CELL IS THE RECTANGLE'S TOP-LEFT — minimum column, minimum row —
// derived from the record's own stored coordinates by AnchorCell and from nothing
// else: no origin subtracted, no inset, no rounding (ALM-PLACE-033). The class is
// the key's LOW BYTE (ALM-CLS-036, REG-KEY-044), masked here, at the one site that
// also turns a record into a cell.
//
// corner is the terrain's CORNER-HEIGHT accessor — Projection.Altitude,
// which answers for a mesh vertex — and NOT the per-cell AnchorHeight the
// object builder takes. A bilinear sample at the rectangle's centre falls on
// a half-integer whenever an extent is odd, and a per-cell helper cannot
// express that. Nil is the flat geometry, as it is on the object side.
// originY is the render's own vertical origin (Render.OriginY,
// Projection.MinV), 0 in the flat geometry; both are SUBTRACTED, so higher
// ground raises a structure up the screen.
//
// ORDER IS ONE STABLE SORT, HERE, and neither renderer carries an opinion
// about it. Row order is load-bearing: a back row's overhang covers the rows
// in front of it, so a structure whose art reaches up must be drawn after
// everything on the rows it reaches over. Column order is free — strip
// frames are one cell wide, so two columns never overlap — and is
// reproduced because it is what the original walks. Stability is what makes
// "ties in record order" true with no tiebreaker field.
//
// The third return is the ANIMATED SUBSET: the index, in the list beside it,
// of every entry that can change with the counter — its class animates and
// its grid cell is live in the mask. Both are per-class facts no counter can
// move, so they are decided ONCE here rather than re-asked per rendered
// frame, and on a map whose structures do not animate the subset is empty
// and the per-counter pass has nothing to walk.
func StructurePlacements(g Grid, set *StructureSet, corner func(c, r int) int, originY int) ([]StructurePlacement, StructureCounts, []int) {
	var counts StructureCounts
	if set == nil || g.Width <= 0 || g.Height <= 0 {
		return nil, counts, nil
	}

	var out []StructurePlacement
	for _, rec := range g.Structures {
		// The one conversion site: the stored anchor becomes a cell and the stored
		// key becomes a low byte, together, so neither convention can be applied
		// without the other.
		anchorCol, anchorRow := AnchorCell(rec.X, rec.Y)
		c := set.Classes[byte(rec.Key)]
		switch {
		case c == nil:
			counts.Placements.NoClass++
			continue
		case c.VariableSize:
			if !c.drawable() || rec.VariableWidth < 3 || rec.VariableHeight < 2 {
				counts.Placements.VariableSize++
				continue
			}
			counts.Placements.Drawn++
			lift := structureRectLift(rec.VariableWidth, rec.VariableHeight, anchorCol, anchorRow, corner)
			out = appendVariableVerticalBridge(out, g, c, rec.ID, anchorCol, anchorRow,
				rec.VariableWidth, rec.VariableHeight, lift, originY, &counts.Cells)
			continue
		case !c.drawable():
			counts.Placements.Undrawable++
			continue
		}
		counts.Placements.Drawn++

		// ONE SAMPLE FOR THE WHOLE STRUCTURE, taken here and stamped into every
		// entry below.
		lift := structureLift(c, anchorCol, anchorRow, corner)

		out = appendStructureStrips(out, g, c, rec.ID, anchorCol, anchorRow, lift, originY, &counts.Cells)
	}

	// Stable, once, in the builder. Rows ascend and columns DESCEND; entries of
	// one cell keep the strip order the loop produced, which the contract leaves
	// free because one cell's frames do not overlap.
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Cell.Y != out[j].Cell.Y {
			return out[i].Cell.Y < out[j].Cell.Y
		}
		return out[i].Cell.X > out[j].Cell.X
	})

	// The subset is collected AFTER the sort, so it indexes the list as the
	// caller receives it. Collecting it during the walk and re-mapping it
	// afterwards would be a second place the two could come apart.
	var animated []int
	for i := range out {
		if out[i].Class.animates(out[i].GridIndex) {
			animated = append(animated, i)
		}
	}
	return out, counts, animated
}

// appendVariableVerticalBridge expands the shipped bridge1v nine-patch over the
// author-sized rectangle. Frame rows are top 0..2, repeatable middle 3..5 and
// bottom 6..8; each map cell contributes exactly one flat 32-pixel frame.
func appendVariableVerticalBridge(out []StructurePlacement, g Grid, c *StructureClass,
	id uint32, anchorCol, anchorRow, width, height, lift, originY int, cells *StructureCellCounts) []StructurePlacement {

	for row0 := 0; row0 < height; row0++ {
		frameRow := 1
		if row0 == 0 {
			frameRow = 0
		} else if row0 == height-1 {
			frameRow = 2
		}
		row := anchorRow + row0
		for col0 := 0; col0 < width; col0++ {
			frameCol := 1
			if col0 == 0 {
				frameCol = 0
			} else if col0 == width-1 {
				frameCol = 2
			}
			col := anchorCol + col0
			if col < 0 || row < 0 || col >= g.Width || row >= g.Height {
				cells.Outside++
				continue
			}
			idx := frameRow*3 + frameCol
			cells.Strips++
			cells.Frames++
			out = append(out, StructurePlacement{
				StructureID: id,
				Cell:        image.Point{X: col, Y: row},
				TopLeft:     image.Point{X: col * CellSize, Y: row*CellSize - lift - originY},
				GridIndex:   idx,
				Class:       c,
				Frame:       c.frame(idx),
			})
		}
	}
	return out
}

// animates reports whether one grid index of this class can change with the
// counter: the class carries a cycle, and the mask leaves that cell live.
//
// It is the ONE predicate the subset is built from and the same one
// SelectStructureFrame's first two arms test, so an entry outside the subset is
// an entry a selection would leave at its base frame anyway.
func (c *StructureClass) animates(gridIndex int) bool {
	return c != nil && len(c.Timeline) > 0 &&
		gridIndex >= 0 && gridIndex < len(c.Rank) && c.Rank[gridIndex] >= 0
}

// appendStructureStrips appends one drawable placement's whole rectangle — every
// image row of every cell of it — and records what those cells came to.
//
// For the rectangle cell at local position (COL0, ROW0) the strip is (spec, "The
// strip"):
//
//	rowTop = ROW0 - TileHeight + FullHeight
//	limit  = rowTop        when ROW0 != 0
//	limit  = 0             when ROW0 == 0          the back row also draws the overhang
//
//	for k = rowTop down to limit:
//	    grid index = k*TileWidth + COL0
//	    destX      = col*CellSize
//	    destY      = row*CellSize - lift - originY - (rowTop - k)*CellSize
//
// The grid's bottom TileHeight rows map one-to-one onto the rectangle's rows; its
// top FullHeight-TileHeight rows are the overhang and stack above the back row,
// which is what the ROW0 == 0 arm is for. Where FullHeight equals TileHeight the
// strip is one frame per cell and there is no overhang at all.
//
// A CELL OUTSIDE THE MAP EXTENT CONTRIBUTES NO ENTRY and is counted apart, so a
// rectangle running off the last column drops those cells and touches no cell of
// the next row — the column is tested against the map's own width rather than
// allowed to wrap through the row-major index.
//
// The lower bound is clamped at grid row 0 because a negative grid row is
// not a row of the sheet. It is unreachable wherever FullHeight >=
// TileHeight, which is every shipped class; without it a class whose grid is
// shorter than its own rectangle would address frames below the sheet's
// first, and the contract's fallback has no base frame to offer for an index
// that is not a grid index at all.
func appendStructureStrips(out []StructurePlacement, g Grid, c *StructureClass,
	id uint32, anchorCol, anchorRow, lift, originY int, cells *StructureCellCounts) []StructurePlacement {

	for row0 := 0; row0 < c.TileHeight; row0++ {
		row := anchorRow + row0
		for col0 := 0; col0 < c.TileWidth; col0++ {
			col := anchorCol + col0
			if col < 0 || row < 0 || col >= g.Width || row >= g.Height {
				cells.Outside++
				continue
			}
			cells.Strips++

			rowTop := row0 - c.TileHeight + c.FullHeight
			limit := rowTop
			if row0 == 0 {
				limit = 0
			}
			if limit < 0 {
				limit = 0
			}
			for k := rowTop; k >= limit; k-- {
				idx := k*c.TileWidth + col0
				cells.Frames++
				if k < c.FullHeight-c.TileHeight {
					cells.Overhang++
				}
				out = append(out, StructurePlacement{
					StructureID: id,
					Cell:        image.Point{X: col, Y: row},
					TopLeft:     image.Point{X: col * CellSize, Y: row*CellSize - lift - originY - (rowTop-k)*CellSize},
					GridIndex:   idx,
					Class:       c,
					Frame:       c.frame(idx),
				})
			}
		}
	}
	return out
}

// TERR-221.
func structureLift(c *StructureClass, anchorCol, anchorRow int, corner func(cc, rr int) int) int {
	return structureRectLift(c.TileWidth, c.TileHeight, anchorCol, anchorRow, corner)
}

func structureRectLift(width, height, anchorCol, anchorRow int, corner func(cc, rr int) int) int {
	if corner == nil {
		return 0
	}
	x2 := 2*anchorCol + width
	y2 := 2*anchorRow + height
	x, y := x2/2, y2/2
	a, b := corner(x, y), corner((x2+1)/2, y)
	c, d := corner(x, (y2+1)/2), corner((x2+1)/2, (y2+1)/2)
	top := a + (b-a)*(x2%2)/2
	bottom := c + (d-c)*(x2%2)/2
	return top + (bottom-top)*(y2%2)/2
}

func SelectStructureFrame(c *StructureClass, gridIndex int, counter uint32, animate, ruined bool) int {
	if c == nil {
		return gridIndex
	}
	if ruined && !c.Indestructible {
		// From the END of the sheet. A sheet carrying no ruin block at all has
		// exactly its base block, so this resolves to gridIndex and the structure
		// redraws its intact art — the fallback and the identity are the same
		// expression, not two.
		if idx := len(c.Frames) - c.GridCells() + gridIndex; idx >= 0 && idx < len(c.Frames) {
			return idx
		}
		return gridIndex
	}
	if !animate || len(c.Timeline) == 0 || gridIndex < 0 || gridIndex >= len(c.Rank) {
		return gridIndex
	}
	rank := c.Rank[gridIndex]
	if rank < 0 {
		return gridIndex
	}
	phase := c.Timeline[int(counter%uint32(len(c.Timeline)))]
	if phase <= 0 {
		return gridIndex
	}
	idx := c.GridCells() + (phase-1)*c.Live + rank
	if idx < 0 || idx >= len(c.Frames) {
		return gridIndex
	}
	return idx
}

// AnimateStructures is the per-counter pass: the entry list as it draws at
// one counter value with the animation switch in one state.
//
// IT WRITES INTO A CALLER-OWNED BUFFER and returns what to draw from, exactly as
// AnimateStatics does, so a rendered frame allocates nothing after the first.
//
// WHEN NOTHING CAN CHANGE IT RETURNS THE BUILT LIST ITSELF, untouched and
// unaliased: with the switch off, or with an empty subset, there is no entry to
// re-select, so an ordinary map draws the very slice the builder produced and
// this pass costs one comparison. A caller must therefore NOT assume the answer
// is its own buffer.
//
// THE SWITCH-OFF ARM WALKS NOTHING, and that is where this pass differs from the
// object side's. There, "off" collapses every placement to sheet frame 0, which
// is not the frame its Index selected. Here, "off" is the BASE frame at each
// entry's own grid index — which is precisely what the builder already put in
// every entry, so there is no walk that could produce a different answer.
//
// THE RUIN ARM WALKS EVERY ENTRY, as !animate does on the object side: every
// drawable structure is destroyed at once under the diagnostic, so the animated
// subset is not the set that can change. It honours Indestructible per class, so
// a class the game refuses to ruin stays intact under the switch, and it is a
// PARAMETER of the draw — no placement, class or bundle carries a destruction
// state, so nothing can leak into a simulation later mistaking a diagnostic for a
// fact.
//
// The subset is the builder's own and holds the index of every entry that CAN
// change: its class animates and its grid cell is live in the mask. Both are
// per-class facts no counter can move, so they are decided once at the build.
//
// EACH RE-SELECTED ENTRY KEEPS ITS TOP-LEFT.
func AnimateStructures(dst, places []StructurePlacement, animated []int, counter uint32, animate, ruined bool) []StructurePlacement {
	return AnimateStructureStates(dst, places, animated, counter, animate, ruined, nil)
}

// AnimateStructureStates applies the diagnostic ruin switch plus live ruin ids
// supplied by the simulation-facing viewer seam.
func AnimateStructureStates(dst, places []StructurePlacement, animated []int, counter uint32, animate, ruined bool, ruinedIDs map[uint32]bool) []StructurePlacement {
	if !ruined && len(ruinedIDs) == 0 && (!animate || len(animated) == 0) {
		return places
	}
	dst = append(dst[:0], places...)
	if ruined {
		for i := range dst {
			selectStructure(&dst[i], counter, animate, true)
		}
		return dst
	}
	for _, i := range animated {
		// The subset indexes the list it was built beside; the guard keeps the
		// pass total for a caller that pairs a subset with a different list.
		if i < 0 || i >= len(dst) {
			continue
		}
		selectStructure(&dst[i], counter, animate, false)
	}
	for i := range dst {
		if ruinedIDs[dst[i].StructureID] {
			selectStructure(&dst[i], counter, animate, true)
		}
	}
	return dst
}

// selectStructure re-selects one entry's frame in place. It writes Frame and
// nothing else: an entry's top-left cannot move when its frame changes, so there
// is no re-anchoring here and no expression that could perform one.
//
// An entry whose selection lands on no frame is left exactly as the builder made
// it, which is the intact art at a grid index the sheet does not hold.
func selectStructure(p *StructurePlacement, counter uint32, animate, ruined bool) {
	if f := p.Class.frame(SelectStructureFrame(p.Class, p.GridIndex, counter, animate, ruined)); f != nil {
		p.Frame = f
	}
}

// PlaneRef names one drawable of the merged structure/object plane: which of
// the two lists it comes from, and its index in that list.
//
// It is a REFERENCE and not a copy, so the order can be computed once and walked
// at every counter: both per-counter passes patch entries in place and preserve
// their lists' length and order, so an index stays valid for the life of the two
// builds. A merged list of values would have to be rebuilt per frame, and would
// be a third place a placement could be adjusted.
type PlaneRef struct {
	// Structure selects the list: true indexes the structure entries, false the
	// object placements. It is a BOOL and not two index fields, so a ref cannot
	// name one of each or neither.
	Structure bool
	Index     int
}

// PlaneOrder is the draw order of one map's structure and object planes: the
// EARLY pass first, then the main pass merged with the object layer by
// rectangle row.
//
// TWO PASSES, AND EACH ENTRY IS IN EXACTLY ONE. A class with Flat set draws
// before everything else on the map — the decoded early pass (TERR-STRUCT-104) —
// and every other structure entry is merged with the static-object layer.
//
// THE MERGE IS OURS BY CHOICE and it exists ONCE, here, so neither renderer
// carries an ordering opinion and no second sort exists. Nothing at this pin
// places the type-3 object plane in the original's cell walk, and both of our
// lists are cell-anchored and already row-sorted, so a two-pointer merge by ROW
// is the arrangement that keeps back-to-front order between two cell-anchored
// planes; any fixed order would put one plane permanently in front of the other
// regardless of row. At an EQUAL row the structure side goes first, which is the
// arrangement that lets an object stand in front of the building on its own row.
//
// Row order is what is load-bearing: a back row's overhang covers the rows in
// front of it, so a structure whose art reaches up must be drawn after everything
// on the rows it reaches over. Columns are compared nowhere — strip frames are one
// cell wide, so two columns never overlap, and comparing them here would break
// the row ordering the overhang depends on.
//
// It is pure: it reads the two lists' cells and classes and writes nothing.
func PlaneOrder(structures []StructurePlacement, objects []StaticPlacement) []PlaneRef {
	out := make([]PlaneRef, 0, len(structures)+len(objects))
	for i := range structures {
		if structures[i].Class != nil && structures[i].Class.Flat {
			out = append(out, PlaneRef{Structure: true, Index: i})
		}
	}

	j := 0
	for i := range structures {
		if c := structures[i].Class; c != nil && c.Flat {
			continue
		}
		for j < len(objects) && objects[j].Cell.Y < structures[i].Cell.Y {
			out = append(out, PlaneRef{Index: j})
			j++
		}
		out = append(out, PlaneRef{Structure: true, Index: i})
	}
	for ; j < len(objects); j++ {
		out = append(out, PlaneRef{Index: j})
	}
	return out
}

// PlaneKind names which of the three cell-anchored lists a DepthRef indexes.
//
// It is an enum and not two bools beside each other, so "a structure and an
// entity" is unrepresentable rather than merely never built — which is the same
// reason PlaneRef carries one bool and not two index fields.
type PlaneKind uint8

const (
	// PlaneObject is the static-object layer. It is the ZERO VALUE so that a
	// DepthRef written with no kind names the list PlaneRef's own zero value
	// names, and the two types cannot disagree about what "unspecified" means.
	PlaneObject PlaneKind = iota
	PlaneStructure
	PlaneEntity

	// PlaneSack is a ground sack — cell-anchored content exactly as an entity
	// is, merged into the same band by the same row rule. It is declared AFTER
	// PlaneEntity, and only after it, so the three existing constants and the
	// zero value keep the values every caller already compares against; nothing
	// downstream of this story's boundary reads it yet.
	PlaneSack
)

type DepthRef struct {
	Kind  PlaneKind
	Index int
}

// DepthOrder is the draw order of one frame's whole cell-anchored content
// band: the structure/object order PlaneOrder already fixed, with the ENTITY
// plane and — from 0111 — the SACK stream merged into it by cell row.
//
// WHY IT EXISTS AT ALL. Units were drawn in a later pass over the finished art
// band, so a unit standing behind a building was drawn in front of it whatever
// row it stood on — there was no depth rule between the two, and neither list
// knew about the other. A unit is cell-anchored content exactly as a tree and a
// wall are, so the answer is not a rule of its own but the rule those two already
// share, extended to a third list — and, since 0111, a fourth: a sack lying on
// the ground is cell-anchored content by the same argument.
//
// IT IS THE SECOND HALF OF ONE MERGE AND NOT A SECOND MERGE. PlaneOrder owns the
// arrangement between the two art planes and this owns only where an entity or a
// sack goes among them, so no renderer carries an ordering opinion and there is
// still one place the whole order is decided.
//
// THE FLAT PREFIX IS EMITTED UNTOUCHED AND FIRST. A class the bundle marks
// flat is ground decoration drawn before everything else on the map whatever
// row it stands on, so it is not merged with anything — and every entity
// and every sack therefore stand on top of it, which is the arrangement a
// sack lying on a paved courtyard needs exactly as a unit walking over one
// does.
//
// AT AN EQUAL ROW THE TWO ART PLANES GO FIRST, THEN THE THREE CONTENT TIERS
// IN DepthTie ORDER: corpse, then sack, then living unit. The
// art-before-content half is PlaneOrder's own choice carried forward — the
// later drawable owns the overlap, and a sack or a unit standing on a
// building's own anchor row stands in front of it rather than inside it.
// Nothing decodes this; the tiers are ours and the ruling is his.
//
// THE TIE IS THE ONLY THING THAT MOVED. Rows still decide across rows, the flat
// prefix is still emitted untouched and first, and the two streams are still
// INTERLEAVED rather than drained one after the other.
//
// THE ENTITY AND SACK LISTS ARE SORTED HERE AND NOT REQUIRED SORTED. The two
// art lists arrive row-sorted from their builders; an entity list arrives in
// world entity order and a sack list in the world's own ground-sack order,
// neither of which says anything about rows. Demanding a sorted input would
// put half of this ordering rule in every caller. Each sort is by ROW THEN
// TIE and is STABLE, and runs over its own index permutation, so neither
// caller's slice is written through or reordered: a corpse precedes a living
// unit on one row, and below that two entities keep their id order and two
// sacks keep the order the world's own list gave them. The tie term is what
// lets ONE entity stream carry two tiers — without it a corpse and a
// living unit on one cell would reach the interleave in list order, and the
// sack could only ever land on one side of the pair.
//
// THE TWO STREAMS ARE INTERLEAVED, NOT DRAINED ONE AFTER THE OTHER. Both the
// per-ref flush below and the tail drain after the loop pick whichever of
// the pending sack and the pending entity stands on the EARLIER (row, tie),
// one at a time — never every sack before the first entity or every entity
// before the first sack, which would put a near sack in front of a far
// entity or a near entity in front of a far sack.
//
// dst is a destination to reuse across frames, as the animation pass's scratch
// is: the returned slice is dst's own memory whenever it fits. Pass nil for a
// fresh one.
//
// It is pure apart from that buffer: it reads the four lists' cells and classes
// and writes none of them.
func DepthOrder(dst []DepthRef, order []PlaneRef, structures []StructurePlacement,
	objects []StaticPlacement, entities []StaticPlacement, sacks []StaticPlacement) []DepthRef {
	out := dst[:0]
	if cap(out) < len(order)+len(entities)+len(sacks) {
		out = make([]DepthRef, 0, len(order)+len(entities)+len(sacks))
	}

	// The flat prefix, in the order PlaneOrder put it. It is identified by the
	// same predicate PlaneOrder used rather than by a length remembered from
	// there, so nothing here depends on a count the two would have to keep equal.
	i := 0
	for ; i < len(order); i++ {
		r := order[i]
		if !r.Structure || r.Index >= len(structures) {
			break
		}
		if c := structures[r.Index].Class; c == nil || !c.Flat {
			break
		}
		out = append(out, DepthRef{Kind: PlaneStructure, Index: r.Index})
	}

	ents := rowOrder(entities)
	sax := rowOrder(sacks)
	j, k := 0, 0
	for ; i < len(order); i++ {
		r := order[i]
		row, ok := planeRow(r, structures, objects)
		// Flush every pending sack and entity of row < row, interleaved: at each
		// step the earlier row goes, sack first at a tie. A ref that cannot say
		// where it is (ok false) flushes nothing past it — the same guard the
		// entity-only walk already carried.
		for ok {
			eRow, eTie, eReady := rowAt(entities, ents, j)
			sRow, sTie, sReady := rowAt(sacks, sax, k)
			eReady = eReady && eRow < row
			sReady = sReady && sRow < row
			if !eReady && !sReady {
				break
			}
			if sReady && (!eReady || sackFirst(sRow, sTie, eRow, eTie)) {
				out = append(out, DepthRef{Kind: PlaneSack, Index: sax[k]})
				k++
				continue
			}
			out = append(out, DepthRef{Kind: PlaneEntity, Index: ents[j]})
			j++
		}
		kind := PlaneObject
		if r.Structure {
			kind = PlaneStructure
		}
		out = append(out, DepthRef{Kind: kind, Index: r.Index})
	}
	// The tail drain: everything left of BOTH streams once every ref is placed,
	// interleaved exactly as the walk above interleaves them — not one stream
	// exhausted before the other.
	for j < len(ents) || k < len(sax) {
		eRow, eTie, eReady := rowAt(entities, ents, j)
		sRow, sTie, sReady := rowAt(sacks, sax, k)
		if sReady && (!eReady || sackFirst(sRow, sTie, eRow, eTie)) {
			out = append(out, DepthRef{Kind: PlaneSack, Index: sax[k]})
			k++
			continue
		}
		out = append(out, DepthRef{Kind: PlaneEntity, Index: ents[j]})
		j++
	}
	return out
}

// sackFirst is the WHOLE of the interleave's choice between a pending sack and
// a pending entity, both known to exist: the earlier ROW goes first, and on one
// row the lower DepthTie goes first — corpse, sack, living unit.
//
// It is one function called from BOTH the per-ref flush and the tail drain,
// for the reason rowAt is: two hand-written comparisons are two places for
// the rule to drift, and this rule has already been wrong once in exactly
// that shape — 0111's `sRow <= eRow` was written twice and both copies put
// the sack under the body.
//
// THE FULL TIE FAVOURS THE SACK, which cannot arise between the tiers this tree
// assigns — a sack is TieGround and an entity is TieCorpse or TieUnit — and is
// named so the function is total rather than leaving one pair of inputs to the
// reader. It is also the answer 0111 gave, so a caller that names no tie at all
// gets exactly the order it used to get.
func sackFirst(sRow int, sTie DepthTie, eRow int, eTie DepthTie) bool {
	if sRow != eRow {
		return sRow < eRow
	}
	return sTie <= eTie
}

// rowAt is the row AND THE TIE the order-th entry of a row-sorted index
// permutation stands on, and whether one exists at all —
// list[order[i]].Cell.Y and .DepthTie, guarded by i against the permutation's
// own length.
//
// DepthOrder's per-ref flush and its tail drain both read the pending head
// of the entity stream and the pending head of the sack stream through this
// one expression, which is what keeps the two streams peeked at the same way
// rather than by two hand-rolled comparisons that could drift apart from
// each other. The tie rides here rather than being fetched separately for
// the same reason: the pair a comparison is made on is read in one place.
func rowAt(list []StaticPlacement, order []int, i int) (int, DepthTie, bool) {
	if i >= len(order) {
		return 0, 0, false
	}
	p := list[order[i]]
	return p.Cell.Y, p.DepthTie, true
}

// planeRow is the row one structure/object ref stands on, and whether it could
// be read at all.
//
// An index its list does not hold reports NOT ok, and the merge then emits the
// ref where it stands without flushing an entity past it: a ref that cannot say
// where it is must not move anything else.
func planeRow(r PlaneRef, structures []StructurePlacement, objects []StaticPlacement) (int, bool) {
	if r.Structure {
		if r.Index >= len(structures) {
			return 0, false
		}
		return structures[r.Index].Cell.Y, true
	}
	if r.Index >= len(objects) {
		return 0, false
	}
	return objects[r.Index].Cell.Y, true
}

// rowOrder is a cell-anchored list's indices in ascending cell row, then
// ascending DepthTie, then in the list's own order. DepthOrder calls it once
// for the entity list and once for the sack list: the same stable sort backs
// both streams, so neither can end up ordered by a rule the other does not
// share, and "two entities on one row keep their id order" and "two sacks on
// one row keep the world's own list order" are one guarantee applied twice
// rather than two.
//
// THE TIE TERM IS WHAT LETS ONE ENTITY STREAM CARRY TWO TIERS. A corpse and a
// living unit on one cell must reach the interleave corpse-first, or the sack
// between them could only ever land on one side of the pair — the interleave
// takes one head at a time and cannot reorder what is behind it. On the sack
// stream the term is inert: every sack is TieGround, so this sorts exactly as
// it did.
func rowOrder(list []StaticPlacement) []int {
	idx := make([]int, len(list))
	for i := range idx {
		idx[i] = i
	}
	sort.SliceStable(idx, func(a, b int) bool {
		pa, pb := list[idx[a]], list[idx[b]]
		if pa.Cell.Y != pb.Cell.Y {
			return pa.Cell.Y < pb.Cell.Y
		}
		return pa.DepthTie < pb.DepthTie
	})
	return idx
}

func (c StructureCounts) CensusLines() (placements, cells string) {
	return fmt.Sprintf("structures: %d drawn, %d no class, %d undrawable, %d variable-size",
			c.Placements.Drawn, c.Placements.NoClass, c.Placements.Undrawable, c.Placements.VariableSize),
		fmt.Sprintf("cells: %d strips, %d frames (%d overhang), %d outside the map",
			c.Cells.Strips, c.Cells.Frames, c.Cells.Overhang, c.Cells.Outside)
}

// VariableSizeIDs is which of a bundle's class keys carry VariableSize, ascending
// (spec AC-10's first observation).
//
// It is an OBSERVATION and never a contract clause: which ids are the bridge
// subclasses is a fact about one install's registry, so it is reported by the
// corpus harness and reconciled with nothing. The bundle is keyed by the
// placement key's low byte, so these are the keys a map would name them by.
func (s *StructureSet) VariableSizeIDs() []int {
	if s == nil {
		return nil
	}
	var out []int
	for b, c := range s.Classes {
		if c != nil && c.VariableSize {
			out = append(out, b)
		}
	}
	return out
}
