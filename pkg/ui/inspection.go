package ui

import (
	"fmt"
	"image"
	"math"

	"againrom/pkg/render/terrain"
)

// InspectionPanel reports the same subject the mission card composes.
func (v *Viewer) InspectionPanel() (PanelSubject, bool) { return v.panelSubject() }

// InspectionPoint finds a currently exposed pixel for a headless pointer drive.
// It never moves the camera, reveals fog, changes selection or injects a subject.
func (v *Viewer) InspectionPoint(ref InspectionSubject) (int, int, error) {
	var rects []screenRect
	best, point, found := math.Inf(1), image.Point{}, false
	for _, entry := range v.hoverRegistrations() {
		if entry.hit.subject == ref {
			rects = append(rects, entry.hit.rect)
		}
	}
	for _, r := range rects {
		for y := max(0, int(math.Floor(r.Y))); y <= min(v.frameH-1, int(math.Ceil(r.Y+r.H))); y++ {
			for x := max(0, int(math.Floor(r.X))); x <= min(v.frameW-1, int(math.Ceil(r.X+r.W))); x++ {
				if !v.mapSurfaceCaptures(x, y) || v.groundSurfaceCaptures(x, y) {
					continue
				}
				if got, ok := v.hoverAt(x, y); !ok || got != ref {
					continue
				}
				wx, wy, err := v.frameToWindow(image.Pt(x, y), "inspection point")
				if err != nil {
					continue
				}
				win := v.place.WindowSize()
				if wx < EdgeMargin || wy < EdgeMargin || wx >= win.X-EdgeMargin || wy >= win.Y-EdgeMargin {
					continue
				}
				// Downscaling can map adjacent frame pixels to the same window
				// pixel. Validate the delivered coordinate, not the source one.
				fx, fy := v.windowToFrame(wx, wy)
				if _, blocked := TipPanelEventAt(v.MissionTip(), image.Pt(fx, fy), false); blocked {
					continue
				}
				if _, blocked := TipPanelEventAt(v.MissionTip(), image.Pt(fx, fy), true); blocked {
					continue
				}
				if got, ok := v.hoverAt(fx, fy); ok && got == ref {
					dx, dy := float64(fx)-r.X-r.W/2, float64(fy)-r.Y-r.H/2
					if distance := dx*dx + dy*dy; distance < best {
						best, point, found = distance, image.Pt(wx, wy), true
					}
				}
			}
		}
	}
	if found {
		return point.X, point.Y, nil
	}
	return 0, 0, fmt.Errorf("inspection %d:%d has no exposed pixel in the current view", ref.Kind, ref.ID)
}

// InspectionSubject tags object identity where unit and structure IDs overlap.
// Entity and structure numbers overlap, including structure zero.
type InspectionKind uint8

const (
	InspectionNone InspectionKind = iota
	InspectionUnit
	InspectionStructure
)

type InspectionSubject struct {
	Kind InspectionKind
	ID   uint32
}

// SetUnitInspectionPictureSource supplies read-only live equipment/portrait art.
// It does not replace InventorySubject or expose an equipment slot mask.
func (v *Viewer) SetUnitInspectionPictureSource(source func(uint32) *image.RGBA) {
	v.unitInspectionPicture = source
}

// Inspection returns the hovered subject, or the visible selected primary.
// Hover switching and selected fallback are owner-directed (DIV-536), not an
// inference from AI-CURSOR-231's capability-mask decode.
func (v *Viewer) Inspection() (InspectionSubject, bool) {
	if ref, ok := v.hoverInspection(); ok {
		return ref, true
	}
	present := v.visiblePanelSelection(presentSelected(v.sel, v.entities))
	if len(present) == 0 {
		return InspectionSubject{}, false
	}
	return InspectionSubject{Kind: InspectionUnit, ID: present[0].ID}, true
}

func (v *Viewer) visiblePanelSelection(present []MapEntity) []MapEntity {
	// Do not copy or reorder the real selection. A hidden primary makes the
	// panel empty; it must not expose newly changed stats through the fallback.
	if len(present) > 0 && !v.fogGateEntity(present[0].Owner, present[0].Cell.X, present[0].Cell.Y) {
		return nil
	}
	return present
}

func (v *Viewer) hoverInspection() (InspectionSubject, bool) {
	if !v.hasCursor || v.popupOpen() || v.primaryDown || v.secondaryDown ||
		v.dragging || v.rightDragging || v.middleDragging || v.held || v.invGrab || v.dragCandKind != dragNone || v.dragActive ||
		!v.mapSurfaceCaptures(v.cursorX, v.cursorY) || v.groundSurfaceCaptures(v.cursorX, v.cursorY) {
		return InspectionSubject{}, false
	}
	if v.hasWinCursor {
		win := v.place.WindowSize()
		if !image.Pt(v.winCursorX, v.winCursorY).In(image.Rectangle{Max: win}) {
			return InspectionSubject{}, false
		}
	}
	return v.hoverAt(v.cursorX, v.cursorY)
}

func (v *Viewer) inspectionPanel(ref InspectionSubject) (PanelSubject, bool) {
	switch ref.Kind {
	case InspectionUnit:
		for _, e := range v.entities {
			if e.ID == ref.ID && v.fogGateEntity(e.Owner, e.Cell.X, e.Cell.Y) {
				s := v.panelSubjectFromPresent([]MapEntity{e})
				s.Selected = 0
				return s, true
			}
		}
	case InspectionStructure:
		c := v.structureInfo[ref.ID]
		if c == nil {
			break
		}
		if s, ok := v.structureState(ref); ok {
			if s.MaxHealth != 0 || v.usableStructure(ref) {
				hp, maximum := int(int16(s.Health)), int(s.MaxHealth)
				if maximum == 0 {
					hp, maximum = 1000, 1000
				}
				return PanelSubject{Kind: InspectionStructure, ID: s.ID, ClassID: c.ID, Name: structureCardName(v.words, c),
					HP: hp, MaxHP: maximum, Cell: s.Cell,
					DetailLevel: 7, DetailSet: true, Words: v.words}, true
			}
		}
	}
	return PanelSubject{}, false
}

func (v *Viewer) inspectionFigure(ref InspectionSubject) *image.RGBA {
	if ref.Kind == InspectionStructure {
		if c := v.structureInfo[ref.ID]; c != nil {
			return c.Portrait
		}
	}
	if ref.Kind == InspectionUnit && v.unitInspectionPicture != nil {
		return v.unitInspectionPicture(ref.ID)
	}
	return nil
}

func (v *Viewer) refreshHoverLighting() {
	if v.hasCursor {
		v.hoverAt(v.cursorX, v.cursorY)
	}
}

// hoverSpriteFactor selects shading zero for the stored hovered unit.
// AI-770 and R2-ENGINE-851 establish the argument; brightening is Medium.
func (v *Viewer) hoverSpriteFactor(id uint32) (float32, bool) {
	if v.graphics.DisableLighting || v.unshaded {
		return 1, false
	}
	if v.pinnedHover.set {
		ref, ok := v.pinnedHover.ref, v.pinnedHover.ok
		return 1, ok && ref.Kind == InspectionUnit && ref.ID == id
	}
	ref, ok := v.hoverInspection()
	return 1, ok && ref.Kind == InspectionUnit && ref.ID == id
}

// structureCardName is the installed building name for the class when the
// install states one, and the registry's own name text otherwise.
func structureCardName(words Words, c *terrain.StructureClass) string {
	if id := int(c.ID); id > 0 && id < len(words.BuildingNames) && words.BuildingNames[id] != "" {
		return words.BuildingNames[id]
	}
	return c.Name
}
