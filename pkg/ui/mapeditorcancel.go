package ui

func (e *MapEditor) rememberCancel() {
	if e.cancelRestore != nil {
		return
	}
	e.cancelRestore = e.cancelCheckpoint()
}

func (e *MapEditor) cancelCheckpoint() func() {
	mode, chrome := e.editorMode, e.editorChrome
	mode.brush.endStroke()
	previousCancel, previousList := e.cancelRestore, e.listRestore
	var restoreCamera func()
	if e.doc != nil {
		view := e.doc.Viewer
		cam := *view.cam
		restoreCamera = func() { *view.cam = cam }
	}
	selected, filter, focused := e.selected, e.filter, e.focusedReference
	message := e.message
	if chrome.model != nil {
		model := *chrome.model
		chrome.model = &model
	}
	chrome.obj.buf = append([]string(nil), chrome.obj.buf...)
	if chrome.obj.choice != nil {
		choice := *chrome.obj.choice
		chrome.obj.choice = &choice
	}
	return func() {
		e.restoreMode(mode)
		e.editorChrome = chrome
		e.cancelRestore, e.listRestore = previousCancel, previousList
		e.selected, e.filter, e.focusedReference = selected, filter, focused
		e.message = message
		if restoreCamera != nil {
			restoreCamera()
		}
		e.scroll = editorScroll{}
		e.drag, e.dragEdge, e.uiDown = popNone, 0, false
	}
}

func (e *MapEditor) rememberListCancel(kind string) {
	if e.listRestore != nil && e.listKind() == kind {
		return
	}
	e.listRestore = e.cancelCheckpoint()
}

func (e *MapEditor) cancelPopup(target ...editorPopupID) {
	if len(target) > 0 && target[0] == popPrompt && !e.prompt.open() {
		e.cancelRestore, e.saveThen = nil, nil
		e.message = ""
		return
	}
	if len(target) > 0 && target[0] == popList {
		e.closeList()
		if restore := e.listRestore; restore != nil {
			e.listRestore = nil
			restore()
			return
		}
		return
	}
	if e.cancelRestore != nil || e.prompt.open() || e.pathDialog.pending != nil {
		if restore := e.cancelRestore; restore != nil {
			e.cancelRestore = nil
			restore()
		} else {
			e.leave(leavePick | leavePrompt)
			e.message = ""
		}
		e.saveThen = nil
		return
	}
	e.obj.doc = nil
	e.loadObject()
	if len(e.objectBack) > 0 {
		e.backObject()
		return
	}
	e.closeObject()
}
