package ui

import "image"

// MissionScreen is one game's mission screen layout data. The one mission
// screen builder reads it; its zero value is the first game's layout.
type MissionScreen struct {
	// ColumnW is the right column's width: the map view ends ColumnW pixels
	// left of the frame's right edge. The first game's is MissionPanelW
	// (SESS-VIEW-028), and its column art's 16-pixel seam stands over the map;
	// the second game's map view ends at the seam, 176 pixels from the edge
	// at 1024x768 (R2-ENGINE-588). The column art, the inventory row and the
	// spellbook row keep their places in both.
	ColumnW int
	// NoticeOnScreen centres a mission notice on the whole frame, the 640x480
	// place plus (192,144) at 1024x768 (R2-ENGINE-588), instead of on the map
	// view.
	NoticeOnScreen bool
}

// SetMissionScreen adopts the game's mission screen layout data.
func (v *Viewer) SetMissionScreen(s MissionScreen) {
	v.missionScreen = s
	v.syncMapViewport()
}

// columnW is the right column's width the map view stops at.
func (v *Viewer) columnW() int {
	if v.missionScreen.ColumnW > 0 {
		return v.missionScreen.ColumnW
	}
	return MissionPanelW
}

// viewportSize is the map view at the live frame: the frame minus the
// game's right column. A frame narrower than the column has no viewport and
// yields a zero width rather than a negative one.
func (v *Viewer) viewportSize() image.Point {
	return image.Pt(max(0, v.frameW-v.columnW()), v.frameH)
}

// SelectUnit makes the one selectable entity id, owned by the local
// participant, the whole selection, as a plain click on it would without its
// voice reply. It reports false and leaves the selection alone for any other
// id.
func (v *Viewer) SelectUnit(id uint32) bool {
	for _, e := range v.entities {
		if e.ID != id || !selectableEntity(e) || v.localOwner != 0 && e.Owner != v.localOwner {
			continue
		}
		v.sel = selection{id}
		return true
	}
	return false
}

// HoldMinimapContent keeps a newly opened mission's minimap to its
// background and view outline until the open notice closes, or until
// ReleaseMinimapContent finds none open. Both originals show the opening
// dialogue over a minimap with nothing explored on it and fill it after the
// dialogue closes, in their first-mission captures (DIV-2998).
func (v *Viewer) HoldMinimapContent() { v.minimapHeld = true }

// ReleaseMinimapContent ends a hold when no notice is open.
func (v *Viewer) ReleaseMinimapContent() {
	if !v.noticeOpen {
		v.minimapHeld = false
	}
}

// noticeArea is where a mission notice is centred: the frame for a game whose
// notices stand on the screen, the map view otherwise.
func (v *Viewer) noticeArea() image.Point {
	if v.missionScreen.NoticeOnScreen && !v.editorView {
		return image.Pt(v.frameW, v.frameH)
	}
	return v.noticeViewportSize()
}
