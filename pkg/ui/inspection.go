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
	for _, s := range v.planeSprites() {
		if s.Inspection == ref {
			rects = append(rects, s.screenRect)
		}
	}
	if ref.Kind == InspectionUnit {
		_, squares, _ := v.entityLayer()
		for _, e := range squares {
			if e.ID == ref.ID {
				if r, ok := v.entityPickRect(e); ok {
					rects = append(rects, r)
				}
			}
		}
	}
	if ref.Kind == InspectionStructure && v.showStructureArt {
		for _, p := range v.structurePlacements() {
			if p.StructureID == ref.ID && p.Class != nil && p.Class.Flat {
				if r, ok := spriteScreenRect(p.Rect(), v.cam); ok {
					rects = append(rects, r)
				}
			}
		}
	}
	for _, r := range rects {
		for y := max(0, int(r.Y)); y < min(v.frameH, int(math.Ceil(r.Y+r.H))); y++ {
			for x := max(0, int(r.X)); x < min(v.frameW, int(math.Ceil(r.X+r.W))); x++ {
				if !v.mapSurfaceCaptures(x, y) || v.groundSurfaceCaptures(x, y) {
					continue
				}
				if got, ok := v.inspectionAt(x, y); !ok || got != ref {
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
				if got, ok := v.inspectionAt(fx, fy); ok && got == ref {
					return wx, wy, nil
				}
			}
		}
	}
	return 0, 0, fmt.Errorf("inspection %d:%d has no exposed pixel in the current view", ref.Kind, ref.ID)
}

// InspectionSubject is presentation identity, not a selection or command target.
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
		v.dragging || v.rightDragging || v.held || v.invGrab || v.dragCandKind != dragNone || v.dragActive ||
		!v.mapSurfaceCaptures(v.cursorX, v.cursorY) || v.groundSurfaceCaptures(v.cursorX, v.cursorY) {
		return InspectionSubject{}, false
	}
	if v.hasWinCursor {
		win := v.place.WindowSize()
		if !image.Pt(v.winCursorX, v.winCursorY).In(image.Rectangle{Max: win}) {
			return InspectionSubject{}, false
		}
	}
	return v.inspectionAt(v.cursorX, v.cursorY)
}

// The top opaque sprite owns the pointer. This uses the production draw order,
// mirror and camera transform; a tree or a noninspectable corpse occludes the
// object behind it too. Effect overlays are not independent subjects.
func (v *Viewer) inspectionAt(x, y int) (InspectionSubject, bool) {
	_, squares, _ := v.entityLayer()
	for _, e := range squares {
		if e.Untargetable {
			continue
		}
		if r, ok := v.entityPickRect(e); ok && r.meets(float64(x), float64(y), float64(x), float64(y)) {
			return InspectionSubject{Kind: InspectionUnit, ID: e.ID}, true
		}
	}
	sprites := v.planeSprites()
	for i := len(sprites) - 1; i >= 0; i-- {
		if inspectionPixel(sprites[i], x, y) {
			ref := sprites[i].Inspection
			_, exists := v.inspectionPanel(ref)
			return ref, exists
		}
	}
	// Flat structures belong to the early ground pass, behind every sprite.
	if v.showStructureArt {
		places := v.structurePlacements()
		for i := len(places) - 1; i >= 0; i-- {
			p := places[i]
			if p.Class == nil || !p.Class.Flat {
				continue
			}
			r, ok := spriteScreenRect(p.Rect(), v.cam)
			if ok && inspectionPixel(staticScreenRect{screenRect: r, Frame: p.Frame}, x, y) {
				ref := InspectionSubject{Kind: InspectionStructure, ID: p.StructureID}
				_, exists := v.inspectionPanel(ref)
				return ref, exists && v.fogGateSack(p.Cell.X, p.Cell.Y)
			}
		}
	}
	return InspectionSubject{}, false
}

func inspectionPixel(s staticScreenRect, x, y int) bool {
	if s.Frame == nil || s.W <= 0 || s.H <= 0 || float64(x) < s.X || float64(y) < s.Y ||
		float64(x) >= s.X+s.W || float64(y) >= s.Y+s.H {
		return false
	}
	px := int(math.Floor((float64(x) - s.X) * float64(s.Frame.Width) / s.W))
	py := int(math.Floor((float64(y) - s.Y) * float64(s.Frame.Height) / s.H))
	if s.Mirror {
		px = s.Frame.Width - 1 - px
	}
	i := py*s.Frame.Width + px
	return i >= 0 && i < len(s.Frame.Pixels) && s.Frame.Pixels[i].Opaque
}

func (v *Viewer) inspectionPanel(ref InspectionSubject) (PanelSubject, bool) {
	switch ref.Kind {
	case InspectionUnit:
		for _, e := range v.entities {
			if e.ID == ref.ID && !e.Untargetable && v.fogGateEntity(e.Owner, e.Cell.X, e.Cell.Y) {
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
		for _, s := range v.structureStates {
			// Bridges and mission decorations have no health pool. They draw
			// normally but must not become inspection or command subjects.
			// A destroyed building still has a maximum and keeps its card.
			if s.ID == ref.ID && s.MaxHealth != 0 {
				return PanelSubject{Kind: InspectionStructure, ID: s.ID, ClassID: c.ID, Name: structureCardName(v.words, c),
					HP: int(int16(s.Health)), MaxHP: int(s.MaxHealth), Cell: s.Cell,
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

// hoverSpriteBrightness is the absolute sprite-ladder scale a hovered unit's
// own sprite is pushed to — authored equal to spellLightingCells' own Light
// spell peak (pkg/game/world.go's lightSpriteBrightness), so a hovered unit
// reads as "this unit is lit" at the same magnitude the spell itself uses
// rather than a second, unrelated brightening rule (owner report 1, hotfix
// DIV-1349: "яркость выше, как будто спрайт освещается").
//
// NO LOCATED CLAIM ESTABLISHES A HOVER-BRIGHTENING MECHANISM IN THE
// ORIGINAL. This is a bounded search over the claim corpus this package's
// callers already cite (terrain.md, magic.md), not a universal claim that
// no such mechanism exists anywhere in rom.exe; the value and the whole
// behaviour are AUTHORED CHOICE.
const hoverSpriteBrightness = float32(2)

// refreshHoverLighting caches this frame's hovered UNIT subject for
// hoverSpriteFactor. It must run once per frame, from drawFrame, before
// planeSprites is built — see hoverLitID's own doc on why reading
// hoverInspection from inside planeSprites itself would recurse.
func (v *Viewer) refreshHoverLighting() {
	v.hoverLitID, v.hoverLitOK = 0, false
	if ref, ok := v.hoverInspection(); ok && ref.Kind == InspectionUnit {
		v.hoverLitID, v.hoverLitOK = ref.ID, true
	}
}

// hoverSpriteFactor is the ColorScale gain the hovered unit's OWN sprite
// takes, id-keyed and not cell-keyed deliberately: spellSpriteFactor's plane
// is per-cell, but the owner's own words are "его спрайт" — that unit's own
// sprite — and a cell-keyed highlight would also brighten a second unit, a
// corpse or a sack sharing the hovered unit's cell.
//
// THE SECOND RESULT IS WHETHER HOVER IS ENGAGED FOR THIS ID AT ALL — never
// under DisableLighting, mirroring spellSpriteFactor's own gate — and it is
// what a caller composing this with spellSpriteFactor by MAX must branch on,
// not a comparison against the plain float. A caster's Darkness can legally
// put spellSpriteFactor BELOW 1 (an ordinary GAIN of exactly 1 is not
// special on that scale), so testing the numeric VALUE for "hover had
// nothing to say" would silently overwrite a real darkening with a false
// "no gain" whenever this unit is not the one hovered.
func (v *Viewer) hoverSpriteFactor(id uint32) (float32, bool) {
	if v.graphics.DisableLighting || !v.hoverLitOK || id != v.hoverLitID {
		return 1, false
	}
	normal := terrain.ShadeScale(4*v.spriteRow() + 32)
	return hoverSpriteBrightness / normal, true
}

// structureCardName is the installed building name for the class when the
// install states one, and the registry's own name text otherwise.
func structureCardName(words Words, c *terrain.StructureClass) string {
	if id := int(c.ID); id > 0 && id < len(words.BuildingNames) && words.BuildingNames[id] != "" {
		return words.BuildingNames[id]
	}
	return c.Name
}
