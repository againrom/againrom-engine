package ui

import (
	"image"
	"image/color"
	"image/draw"
)

type RoomLayerKind uint8

const (
	RoomLayerFill RoomLayerKind = iota
	RoomLayerBox
	RoomLayerPicture
	RoomLayerBackdrop
	RoomLayerScene
	RoomLayerPanel
	RoomLayerTitle
	RoomLayerContent
	RoomLayerCells
	RoomLayerSchool
	RoomLayerButtons
)

type RoomLayer struct {
	Kind        RoomLayerKind
	Rect        image.Rectangle
	Picture     image.Image
	Scene, Text string
	Ink         color.RGBA
}

var TownRoomFill = townShellFill
var TownRoomText = townShellText
var TownRoomTitleRegion = image.Rect(176, 12, 470, 44)

type RoomPointer struct {
	At    image.Point
	Hover bool
	Press TownSurfaceControl
	Cell  int
}

type RoomSchool struct {
	Art                  *TownSchoolArt
	Class, ColumnFrame   int
	ColumnSet, IdleShine bool
	IdleSlot             int
}

func (v RoomSchool) Visible() bool {
	return v.Class >= 0 && v.Class < 2 && (!v.ColumnSet || v.ColumnFrame == v.Class*15)
}

func (v RoomSchool) draw(dst *image.RGBA, p RoomPage) {
	if v.Art != nil && v.Visible() {
		class := v.Class
		// The shown member's own column face, over whichever face the room
		// background bakes. It is opaque: the frame covers the same pixels the
		// background carries there, and a keyed blit would let the baked face
		// show through the other class's engravings.
		if face := v.Art.Faces[class]; face != nil && !v.ColumnSet {
			b := face.Bounds()
			draw.Draw(dst, b.Add(SchoolFaceOrigin.Sub(b.Min)), face, b.Min, draw.Src)
		}
		for slot := 0; slot < 5; slot++ {
			i := class*5 + slot
			if i >= len(p.Cells) || !p.Cells[i].Enabled {
				continue
			}
			state := -1 // the rest state is engraved on the class's own face
			if p.Cells[i].Selected {
				state = 0
			}
			if p.Pointer.Cell == i {
				state = 1
				if p.Cells[i].Selected {
					state = 2
				}
			}
			if state < 0 || v.Art.Skills[class][slot][state] == nil {
				continue
			}
			pic := v.Art.Skills[class][slot][state]
			r := p.Region([]string{"fighter", "mage"}[class], slot)
			b := pic.Bounds()
			at := image.Pt(r.Min.X+(r.Dx()-b.Dx())/2, r.Min.Y+(r.Dy()-b.Dy())/2)
			draw.Draw(dst, b.Add(at.Sub(b.Min)), pic, b.Min, draw.Over)
		}
		// The idle shine paints after every slot's own state: the shine picture
		// for a slot that is not selected, the shine-on picture for the
		// selected one (TOWN-500).
		if v.IdleShine && p.Pointer.Cell < 0 && v.IdleSlot >= 0 && v.IdleSlot < 5 {
			slot := v.IdleSlot
			i := class*5 + slot
			state := 1
			if i < len(p.Cells) && p.Cells[i].Selected {
				state = 2
			}
			if i < len(p.Cells) && p.Cells[i].Enabled && v.Art.Skills[class][slot][state] != nil {
				pic := v.Art.Skills[class][slot][state]
				r := p.Region([]string{"fighter", "mage"}[class], slot)
				b := pic.Bounds()
				at := image.Pt(r.Min.X+(r.Dx()-b.Dx())/2, r.Min.Y+(r.Dy()-b.Dy())/2)
				draw.Draw(dst, b.Add(at.Sub(b.Min)), pic, b.Min, draw.Over)
			}
		}
	}

}

func drawRoomCells(dst *image.RGBA, p RoomPage) {
	for i, c := range p.Cells {
		r := p.Region("cells", i)
		ink := townShellText
		if !c.Enabled {
			ink = townShellDisabled
		}
		drawTownShellBox(dst, r, c.Selected)
		if c.Picture != nil {
			b := c.Picture.Bounds()
			area := r.Inset(2)
			at := image.Pt(area.Min.X+(area.Dx()-b.Dx())/2, area.Min.Y+(area.Dy()-b.Dy())/2)
			placed := b.Add(at.Sub(b.Min))
			clip := placed.Intersect(area)
			if !clip.Empty() {
				src := b.Min.Add(clip.Min.Sub(placed.Min))
				draw.Draw(dst, clip, c.Picture, src, draw.Over)
			}
		}
		drawTownShellText(dst, p.Font, c.Label, image.Rect(r.Min.X+2, r.Min.Y+4, r.Max.X-2, r.Min.Y+34), ink)
		drawTownShellText(dst, p.Font, c.Detail, image.Rect(r.Min.X+2, r.Min.Y+34, r.Max.X-2, r.Max.Y-4), ink)
		if c.Hired {
			outline(dst, r.Inset(3), townShellText)
		}
	}

}

func (v RoomSchool) controlAt(page RoomPage, p image.Point) (RoomControl, bool) {
	if v.Art == nil || !v.Visible() {
		return RoomControl{}, false
	}
	mask := v.Art.Masks[v.Class]
	r := page.Region("panels", v.Class)
	if mask == nil || !p.In(r) {
		return RoomControl{}, false
	}
	q := p.Sub(r.Min).Add(mask.Bounds().Min)
	if !q.In(mask.Bounds()) {
		return RoomControl{}, false
	}
	code := mask.ColorIndexAt(q.X, q.Y)
	slot, ok := schoolMaskSlot(v.Class, code)
	if !ok {
		return RoomControl{}, false
	}
	i := v.Class*5 + slot
	if i < 0 || i >= len(page.Cells) || !page.Cells[i].Enabled {
		return RoomControl{}, false
	}
	return RoomControl{Surface: TownSurfaceControl{Kind: TownSurfaceControlCell, Index: i}}, true
}
