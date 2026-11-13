package ui

import (
	"path/filepath"
	"strings"

	"againrom/pkg/namefold"
)

// InspectionTextsInside is implemented by edits whose host can save the
// map's pages inside the map, in the texts record of an extended map. It is
// the switch the File menu and the save window show, off unless the opened
// map held its pages inside; it takes no undo step, and a map whose switch
// differs from its file's is changed. SetTextsInside(true) on an original
// map answers *ScriptFormNeeded.
type InspectionTextsInside interface {
	TextsInside() bool
	SetTextsInside(on bool) error
}

func (e *MapEditor) textsInsideHost() (InspectionTextsInside, bool) {
	if e.doc == nil || e.doc.Edits == nil {
		return nil, false
	}
	host, ok := e.doc.Edits.(InspectionTextsInside)
	return host, ok
}

// textsInsideKey is the File menu's row of the switch.
const textsInsideKey = "textsinside"

// textsInsideOn is the switch's state: the menu row's mark and the save
// window row's word.
func (e *MapEditor) textsInsideOn() bool {
	host, ok := e.textsInsideHost()
	return ok && host.TextsInside()
}

// textsInsideRow is the save window's switch row, when the host has one.
func (e *MapEditor) textsInsideRow() (popupRow, bool) {
	if _, ok := e.textsInsideHost(); !ok || e.prompt.kind != promptSave {
		return popupRow{}, false
	}
	value := e.word("editor.save.textsinside.off")
	if e.textsInsideOn() {
		value = e.word("editor.save.textsinside.on")
	}
	return popupRow{Kind: popupSwitch, Value: value}, true
}

// switchTextsInside turns the switch over, for the File menu's row and the
// save window's alike. On an original map the script form question asks
// first: Yes extends the map as one edit and turns the switch on; from the
// save window it shows the window again with the path typed, an .alm name
// made .almx.
func (e *MapEditor) switchTextsInside() {
	host, ok := e.textsInsideHost()
	window := e.prompt.kind == promptSave
	if !ok || !window && e.prompt.open() {
		return
	}
	path := e.prompt.path
	err := host.SetTextsInside(!host.TextsInside())
	if e.askScriptForm(err, func() {
		if ext := filepath.Ext(path); namefold.HostEqual(ext, ".alm") {
			path = strings.TrimSuffix(path, ext) + ".almx"
		}
		if window {
			e.openPrompt(editorPrompt{kind: promptSave, path: path})
		}
		if err := host.SetTextsInside(true); err != nil {
			e.message = e.editorHostText(err.Error())
		}
	}) {
		return
	}
	if err != nil {
		e.message = e.editorHostText(err.Error())
	}
}
