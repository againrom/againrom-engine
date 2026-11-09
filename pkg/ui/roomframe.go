package ui

import "image"

// RoomControl carries a room action or a shared pane corner.
type RoomControl struct {
	Surface TownSurfaceControl
	Square  TownSquareControl
	Corner  CharacterPaneCorner
	Pane    bool
}

type RoomHit struct {
	Rect    image.Rectangle
	Control RoomControl
	Enabled bool
}

func roomHitAt(rows []RoomHit, p image.Point, panel RoomPanel) (RoomControl, bool, bool) {
	for _, row := range rows {
		enabled := row.Enabled
		if panel.TwoLines && row.Control.Surface.Kind == TownSurfaceControlButton {
			i := row.Control.Surface.Index
			if i < 0 || i >= len(panel.Values) {
				continue
			}
			enabled = panel.Values[i].Enabled
		}
		if p.In(row.Rect) {
			if !enabled {
				return RoomControl{}, true, false
			}
			return row.Control, true, enabled
		}
	}
	return RoomControl{}, false, false
}

// ControlAt resolves the page's ordered controls, shared pane and scene.
func (v RoomPage) ControlAt(p image.Point) (RoomControl, bool) {
	if v.Book && p.In(shopTableRegion) {
		return RoomControl{Surface: TownSurfaceControl{Kind: TownSurfaceControlBookPage}}, true
	}
	if c, hit, enabled := roomHitAt(v.BeforePane, p, v.Panel); hit {
		return c, enabled
	}
	if v.Pane != nil {
		if c, hit, acts := characterPaneCornerControl(*v.Pane, p); hit {
			if !acts && v.PaneBlocks {
				return RoomControl{}, false
			}
			if acts {
				kind := map[CharacterPaneCorner]TownSurfaceControlKind{CharacterPanePrev: TownSurfaceControlPrevious,
					CharacterPaneNext: TownSurfaceControlNext, CharacterPaneMode: TownSurfaceControlMode, CharacterPaneBook: TownSurfaceControlBook}[c]
				return RoomControl{Surface: TownSurfaceControl{Kind: kind}, Corner: c, Pane: true}, true
			}
		}
	}
	if v.School != nil && v.School.Art != nil {
		return v.School.controlAt(v, p)
	}
	if c, hit, enabled := roomHitAt(v.AfterPane, p, v.Panel); hit {
		return c, enabled
	}
	if v.Scene != nil && v.HitScene {
		c, hit := v.Scene.ControlAt(p)
		return RoomControl{Square: c}, hit
	}
	return RoomControl{}, false
}

// page binds live frame inputs without rebuilding the page's geometry or layers.
func (v TownRoomView) page() RoomPage {
	page := v.Page
	if page.ViewInputs {
		page.Font, page.Cells = v.Font, v.Cells
		page.Panel.Style, page.Panel.Values = v.ButtonStyle, v.Buttons
	}
	if v.Pane || page.ViewInputs {
		hero := v.Hero
		page.Pane = &hero
	}
	page.Book = v.BookView != nil
	if page.Layers == nil {
		page.Scene, page.HitScene = v.Scene, true
	} else {
		page.PaneBlocks = true
	}
	return page
}

func (v TownRoomView) ControlAt(p image.Point) (RoomControl, bool) {
	return v.page().ControlAt(p)
}

func (v TownRoomView) SurfaceControlAt(p image.Point) (TownSurfaceControl, bool) {
	v.Pane = true
	c, hit := v.ControlAt(p)
	return c.Surface, hit
}

// ComposeTownRoom paints one room and the overlays every room shares.
func ComposeTownRoom(v TownRoomView) *image.RGBA {
	page := v.page()
	size := image.Pt(640, 480)
	if v.Scene != nil && v.Page.Layers == nil {
		size = v.Scene.Size()
	}
	dst := image.NewRGBA(image.Rectangle{Max: size})
	if v.Scene == nil && v.Page.Layers == nil {
		return dst
	}
	if v.Page.Layers != nil {
		page.Phase = v.AnimationFrame
		page.Pointer = RoomPointer{At: v.Hover, Hover: v.HasHover, Press: v.Press, Cell: v.HoverCell}
		DrawRoomPage(dst, page)
	} else if v.Scene != nil {
		v.Scene.Paint(dst)
	}
	if page.Pane != nil {
		DrawTownCharacterRegion(dst, *page.Pane)
	}
	drawTownShellText(dst, v.Font, v.Message, townSurfaceMessageRect(), townShellText)
	if v.BookView != nil {
		drawTownBook(dst, *v.BookView)
	}
	ComposeTipPanel(dst, v.Tip)
	if v.HasHover && !v.SuppressHover {
		doll := v.candidateDollHover()
		if lines, ok := doll.LinesAt(v.Hover); ok {
			doll.Tip.DrawBeside(dst, v.Font, lines, v.Hover)
		}
	}
	return dst
}
