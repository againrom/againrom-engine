package ui

import (
	"image"
	"sort"

	"againrom/pkg/render/terrain"
)

// HoverProfile describes rectangle sources and registration phases.
type HoverProfile struct {
	UnitRectangle      func(*terrain.UnitClass) image.Rectangle
	StructureRectangle func(*terrain.StructureClass) image.Rectangle
	UnitPhases         [3]int
	StructurePhases    [2]int
}

// KindRectangleHover follows AI-768/769 and R2-ENGINE-849/850.
var KindRectangleHover = HoverProfile{
	UnitRectangle:      func(c *terrain.UnitClass) image.Rectangle { return c.Selection },
	StructureRectangle: func(c *terrain.StructureClass) image.Rectangle { return c.Selection },
	UnitPhases:         [3]int{3, 0, 4},
	StructurePhases:    [2]int{1, 2},
}

func (v *Viewer) SetHoverProfile(p HoverProfile) { v.hoverProfile = p; v.hover.valid = false }

type hoverHit struct {
	subject   InspectionSubject
	entity    *MapEntity
	structure *MapStructure
	rect      screenRect
}

type hoverCache struct {
	hit                    hoverHit
	x, y                   int
	valid                  bool
	bodySpell              bool
	fogReveal              bool
	cameraX, cameraY, zoom float64
	viewW, viewH           int
}

type hoverRegistration struct {
	hit   hoverHit
	order artOrder
}

func (v *Viewer) hoverEntityRect(e MapEntity) (screenRect, bool) {
	p := v.hoverProfile
	if p.UnitRectangle == nil {
		p = KindRectangleHover
	}
	var r screenRect
	var ok bool
	if c := e.Art; c != nil {
		cx, cy := v.cellWorldCentre(e.Cell)
		x, y := int(cx)-c.CenterX, int(cy)-c.CenterY
		bounds := p.UnitRectangle(c).Canon()
		if bounds.Empty() && !c.SelectionSet {
			bounds = image.Rect(0, 0, c.Width, c.Height)
			if bounds.Empty() {
				return v.entityPickRect(e)
			}
		}
		r, ok = spriteScreenRect(bounds.Add(image.Pt(x, y)).Add(v.entityShift(e)).Add(e.DamageJolt), v.cam)
	} else {
		r, ok = v.entityPickRect(e)
	}
	return r, ok
}

func (v *Viewer) hoverRegistrations() []hoverRegistration {
	p := v.hoverProfile
	if p.UnitRectangle == nil || p.StructureRectangle == nil {
		p = KindRectangleHover
	}
	var out []hoverRegistration
	for i := range v.entities {
		e := &v.entities[i]
		if e.Untargetable && !v.hoverAllowsBody() || !v.fogGateEntity(e.Owner, e.Cell.X, e.Cell.Y) {
			continue
		}
		r, ok := v.hoverEntityRect(*e)
		if !ok {
			continue
		}
		phase := p.UnitPhases[0]
		if int(e.DrawCategory) < len(p.UnitPhases) {
			phase = p.UnitPhases[e.DrawCategory]
		}
		out = append(out, hoverRegistration{hit: hoverHit{subject: InspectionSubject{InspectionUnit, e.ID}, entity: e, rect: r}, order: artOrder{phase: phase, cell: e.Cell, index: i}})
	}
	if v.showStructureArt {
		for i, record := range v.grid.Structures {
			ref := InspectionSubject{InspectionStructure, record.ID}
			c := v.structureInfo[record.ID]
			col, row := terrain.AnchorCell(record.X, record.Y)
			cell := image.Pt(col, row)
			_, admitted := v.inspectionPanel(ref)
			if c == nil || !admitted || !v.structurePickGate(ref, cell.X, cell.Y) {
				continue
			}
			bounds := p.StructureRectangle(c).Canon()
			if bounds.Empty() && (!c.SelectionSet || v.usableStructure(ref)) {
				w, h := (cellSpace{}).corner(c.TileWidth, c.FullHeight)
				bounds = image.Rect(0, 0, int(w), int(h))
			}
			cx, cy := v.cellWorldCentre(cell)
			_, height := (cellSpace{}).corner(0, c.TileHeight-c.FullHeight)
			at := image.Pt(int(cx)-terrain.CellSize/2, int(cy)-terrain.CellSize/2+int(height))
			r, ok := spriteScreenRect(bounds.Add(at), v.cam)
			if !ok {
				continue
			}
			phase := p.StructurePhases[1]
			if c.Flat {
				phase = p.StructurePhases[0]
			}
			state, _ := v.structureState(ref)
			out = append(out, hoverRegistration{hit: hoverHit{subject: ref, structure: &state, rect: r}, order: artOrder{phase: phase, cell: cell, index: i}})
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].order.before(out[j].order) })
	return out
}

// hoverAt stores the accepted object and ID together. Pointer queries use the
// same registration walk as the frame's cursor, card, highlight and release.
func (v *Viewer) hoverAt(x, y int) (InspectionSubject, bool) {
	body := v.hoverAllowsBody()
	if v.hover.valid && v.hover.x == x && v.hover.y == y && v.hover.bodySpell == body && v.hover.fogReveal == v.fogReveal &&
		v.hover.cameraX == v.cam.X && v.hover.cameraY == v.cam.Y && v.hover.zoom == v.cam.Zoom && v.hover.viewW == v.cam.ViewW && v.hover.viewH == v.cam.ViewH {
		return v.hover.hit.subject, v.hover.hit.subject.Kind != InspectionNone
	}
	v.hover = hoverCache{x: x, y: y, valid: true, bodySpell: body, fogReveal: v.fogReveal, cameraX: v.cam.X, cameraY: v.cam.Y, zoom: v.cam.Zoom, viewW: v.cam.ViewW, viewH: v.cam.ViewH}
	if !v.gameAreaAt(x, y) {
		return InspectionSubject{}, false
	}
	entries := v.hoverRegistrations()
	v.hover.hit = registeredHover(entries, x, y)
	return v.hover.hit.subject, v.hover.hit.subject.Kind != InspectionNone
}

func (v *Viewer) hoverAllowsBody() bool {
	return v.spellModeLive() && v.spellNamesBody(v.selectedSpell)
}

func registeredHover(entries []hoverRegistration, x, y int) hoverHit {
	for i := len(entries) - 1; i >= 0; i-- {
		r := entries[i].hit.rect
		if r.W > 0 && r.H > 0 && r.X < float64(x+1) && r.X+r.W > float64(x-1) && r.Y < float64(y+1) && r.Y+r.H > float64(y-1) {
			return entries[i].hit
		}
	}
	return hoverHit{}
}
