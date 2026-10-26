package ui

import (
	"image"

	"github.com/hajimehoshi/ebiten/v2"

	"againrom/pkg/render/camera"
	"againrom/pkg/render/terrain"
)

// cellSpace is the one projection of map cells into the viewer's world: a
// cell's corners, its centre, its quad and its four screen vertices, in flat
// mode (proj nil, the un-displaced CellSize lattice) and in displaced mode
// (proj, the height mesh). The game and the map editor take every cell corner
// and centre from here; internal/archtest fails a new site that computes one
// in pkg/ui or pkg/render outside this file.
//
// The displaced centre stands at the cell's anchor height, the mean of its
// four corner altitudes truncated toward zero (TERR-SPR-039,
// terrain.Projection.AnchorHeight): the height a sprite, a path leg and an
// editor marker on that cell share.
type cellSpace struct {
	proj *terrain.Projection
}

// cellSpace is this viewer's cell projection in its current mode. The guard is
// Mode() and not v.proj != nil: SetFlat may select flat while a projection
// exists, and nothing may carry a height offset there.
func (v *Viewer) cellSpace() cellSpace {
	if v.Mode() == ModeDisplaced {
		return cellSpace{proj: v.proj}
	}
	return cellSpace{}
}

// corner is mesh vertex (c,r) in world pixels.
func (s cellSpace) corner(c, r int) (float64, float64) {
	if s.proj == nil {
		return float64(c * terrain.CellSize), float64(r * terrain.CellSize)
	}
	x, y := s.proj.WorldCorner(c, r)
	return float64(x), float64(y)
}

// lift is the vertical world offset a point anchored in cell carries over the
// flat lattice: -AnchorHeight - MinV when displaced, 0 when flat.
func (s cellSpace) lift(cell image.Point) int {
	if s.proj == nil {
		return 0
	}
	return -s.proj.AnchorHeight(cell.X, cell.Y) - s.proj.MinV
}

// centre is the world point in the middle of cell: the flat lattice's centre
// plus the cell's lift.
func (s cellSpace) centre(cell image.Point) (float64, float64) {
	x := cell.X*terrain.CellSize + terrain.CellSize/2
	y := cell.Y*terrain.CellSize + terrain.CellSize/2 + s.lift(cell)
	return float64(x), float64(y)
}

// quad is cell (col,row)'s four world corners in outline order: TL, TR, BR, BL.
func (s cellSpace) quad(col, row int) [4][2]float64 {
	var out [4][2]float64
	for i, d := range [4][2]int{{0, 0}, {1, 0}, {1, 1}, {0, 1}} {
		x, y := s.corner(col+d[0], row+d[1])
		out[i] = [2]float64{x, y}
	}
	return out
}

// vertices is cell (tx,ty)'s four screen corners in the order quadIndices
// assumes: TL, TR, BL, BR. Dst is WorldToScreen of corner, unrounded until the
// float32 the vertex takes; Src spans 0..CellSize; the colour is 1,1,1,1,
// because ebiten draws ColorA == 0 fully transparent.
func (s cellSpace) vertices(cam *camera.Camera, tx, ty int) [4]ebiten.Vertex {
	var out [4]ebiten.Vertex
	for i, d := range [4][2]int{{0, 0}, {1, 0}, {0, 1}, {1, 1}} {
		sx, sy := cam.WorldToScreen(s.corner(tx+d[0], ty+d[1]))
		out[i] = ebiten.Vertex{
			DstX:   float32(sx),
			DstY:   float32(sy),
			SrcX:   float32(d[0] * terrain.CellSize),
			SrcY:   float32(d[1] * terrain.CellSize),
			ColorR: 1, ColorG: 1, ColorB: 1, ColorA: 1,
		}
	}
	return out
}

// cellWorldCentre is the world point at the middle of cell as this viewer
// draws it. A path leg, a start view, an effect mark and every editor marker
// read it, so they sit on one point of a sloped cell.
func (v *Viewer) cellWorldCentre(cell image.Point) (float64, float64) {
	return v.cellSpace().centre(cell)
}

// cellScreenCentre is cellWorldCentre through the camera.
func (v *Viewer) cellScreenCentre(cell image.Point) (float64, float64) {
	return v.cam.WorldToScreen(v.cellWorldCentre(cell))
}

// cellScreenCorner is mesh vertex (c,r) through the camera.
func (v *Viewer) cellScreenCorner(c, r int) (float64, float64) {
	return v.cam.WorldToScreen(v.cellSpace().corner(c, r))
}

// cellLift is the vertical world offset every glyph on one cell carries. A
// glyph spanning two cells interpolates between their two lifts
// (spellBoltLift).
func (v *Viewer) cellLift(cell image.Point) int {
	return v.cellSpace().lift(cell)
}

// footprintCentre is the offset from the centre of an actor's stored cell to
// the centre of its side-by-side footprint. The stored cell is the
// footprint's top-left (MOVE-087) and the drawable centre is that cell plus
// 128 sub-cells per extra cell (ANIM-TELEPORT-167), so a side n adds
// (n-1)*CellSize/2 on each axis. A side of one or less adds nothing.
func footprintCentre(side int) image.Point {
	if side <= 1 {
		return image.Point{}
	}
	d := (side - 1) * terrain.CellSize / 2
	return image.Point{X: d, Y: d}
}

// tileVertices is tile (tx,ty)'s displaced screen vertices over proj.
func tileVertices(cam *camera.Camera, proj *terrain.Projection, tx, ty int) [4]ebiten.Vertex {
	return cellSpace{proj: proj}.vertices(cam, tx, ty)
}

// flatTileVertices is tile (tx,ty)'s screen vertices over the flat lattice.
func flatTileVertices(cam *camera.Camera, tx, ty int) [4]ebiten.Vertex {
	return cellSpace{}.vertices(cam, tx, ty)
}
