package ui

import (
	"fmt"
	"time"
)

type QuickSaveControls struct {
	Save func(onMap bool) error
	Load func() (MapOpener, bool, error)
}

func (a *App) SetQuickSaveControls(c QuickSaveControls) { a.flow.loadUI.quick = c }

func (v *Viewer) cancelPointerGesture() {
	v.dragging, v.held, v.boxing = false, false, false
	v.dragMoved = 0
	v.primaryDown, v.secondaryDown = false, false
	v.rightDragging, v.rightPanned = false, false
	v.paneSecondaryGrab, v.paneCornerGrab, v.minimapGrab = false, false, false
	v.cmdDragCell = -1
	v.invGrab, v.invEquipTap = false, false
	v.invClickFrames, v.invWornClickFrames = 0, 0
	v.dragActive, v.dragCandKind, v.dragIcon = false, dragNone, nil
	v.cancelGold()
	v.SetDollSuppressedFigure(0, 0, nil, nil)
}

func (a *App) stepQuickSave(save bool) {
	f := a.flow
	var err error
	message := f.words.SaveAcknowledgement
	if save {
		if f.loadUI.quick.Save == nil {
			err = fmt.Errorf("quicksave is unavailable")
		} else {
			err = f.loadUI.quick.Save(f.screen == ScreenMap)
		}
	} else {
		message = "Quickload complete"
		if f.loadUI.quick.Load == nil {
			err = fmt.Errorf("quickload is unavailable")
		} else {
			var open MapOpener
			var town bool
			open, town, err = f.loadUI.quick.Load()
			if err == nil {
				err = f.loadPrepared(open, town)
			}
		}
	}
	if err != nil {
		message = "Quicksave failed: " + err.Error()
		if !save {
			message = "Quickload failed: " + err.Error()
		}
	}
	if f.screen == ScreenMap && f.viewer != nil {
		f.viewer.PostMessage(message, MessageWhite, 15*time.Second)
	} else {
		f.msg = message
	}
	a.syncViewerLayout()
}
