package ui

import (
	"image"

	"againrom/pkg/render/terrain"
)

// THE UNIT DRAW'S TWO MARK PASSES (1002; MAGIC-MARK-059).
//
// The original's unit draw walks the actor's mark-record array TWICE: once
// before dispatching the actor's own sprite, drawing a record only when
// `depth > 0`, and once after, drawing a record only when `depth <= 0`. Both
// passes compute the same position. This file is that walk.
//
// IT DERIVES NOTHING. Which effects an actor carries, what countdown each is
// at, which records a kind builds and which sheet a record index names are all
// decided on the far side of this seam. What arrives is a sheet and a record.

// UnitMark is one built mark record beside the art it draws through: the record
// the render tier's builder produced, and the sheet its record index named.
//
// Sheet CARRIES THE CENTRING HALVES, exactly as SpellBolt's does — the
// registry's own Width and Height halved, which the position expression
// subtracts. MAGIC-MARK-059 states that subtraction as `- Width/2` and
// `- Height/2` off the sprite record's own fields, so the two are the same
// numbers reached by the same route.
//
// A nil Sheet, and a Phase the sheet does not hold, both draw nothing: the same
// three refusals a spell object already takes.
type UnitMark struct {
	Sheet *terrain.EffectSheet
	Mark  terrain.EffectMark
}

// entityMarkRects places one entity's marks in the current view, split into the
// two passes by the sign of each record's depth (MAGIC-MARK-059).
//
// THE POSITION IS THE CLAIM'S, TERM FOR TERM:
//
//	x = anchorX + dx - Width/2
//	y = anchorY - lift - dy'... where dy' folds dy and depth
//
// spelled here as the actor's own cell anchor in world pixels — the very point
// StaticAnchor puts a unit sprite's anchor at — plus the record's dx and dy,
// minus the record's depth, minus the sheet's two centring halves. The relief
// lift and the displaced origin are subtracted here rather than in placeArm
// because the band's other entries take them the same way, through UnitPlace:
// a mark and the actor under it must be lifted by one expression or they part
// company on a slope.
//
// THE ENTITY'S OWN DISPLACEMENT IS ADDED, so a mark rides a crossing with the
// actor rather than standing on the cell it started from.
//
// back is every record with `depth > 0` and front every record with
// `depth <= 0`, each in the builder's own order, so the two lists together are
// the record array read twice and nothing is drawn twice or dropped.
func (v *Viewer) entityMarkRects(e MapEntity) (back, front []staticScreenRect) {
	if len(e.Marks) == 0 {
		return nil, nil
	}
	lift, originY := 0, 0
	if v.Mode() == ModeDisplaced {
		lift = v.proj.AnchorHeight(e.Cell.X, e.Cell.Y)
		originY = v.proj.MinV
	}
	shift := v.entityShift(e)
	anchorX := e.Cell.X*terrain.CellSize + terrain.CellSize/2 + shift.X
	anchorY := e.Cell.Y*terrain.CellSize + terrain.CellSize/2 + shift.Y - lift - originY

	for _, m := range e.Marks {
		f := m.Sheet.Frame(m.Mark.Phase)
		if f == nil || f.Width <= 0 || f.Height <= 0 {
			continue
		}
		px := anchorX + m.Mark.DX - m.Sheet.CenterX
		py := anchorY + m.Mark.DY - m.Mark.Depth - m.Sheet.CenterY
		r, ok := spriteScreenRect(image.Rect(px, py, px+f.Width, py+f.Height), v.cam)
		if !ok {
			continue
		}
		entry := staticScreenRect{screenRect: r, Effect: f}
		if m.Mark.Depth > 0 {
			back = append(back, entry)
		} else {
			front = append(front, entry)
		}
	}
	return back, front
}
