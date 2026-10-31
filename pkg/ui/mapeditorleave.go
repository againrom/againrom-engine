package ui

import (
	"fmt"
	"image"
	"path/filepath"

	"github.com/hajimehoshi/ebiten/v2"
)

// leaveStep retains a close, map-open or mod-open continuation.
type leaveStep struct {
	open      bool
	source    string
	workspace bool
}

// The question's buttons, in order.
const (
	leaveSave = iota
	leaveGo
	leaveStay
)

// askLeave asks what to do with the unsaved changes before step runs.
func (e *MapEditor) askLeave(step leaveStep) {
	e.leave(leaveDrag | leavePick)
	e.openPrompt(editorPrompt{kind: promptLeave, then: step})
}

// runLeave leaves the question and does the step it held: it closes the
// window or opens the map, without saving.
func (e *MapEditor) runLeave(step leaveStep) error {
	e.leave(leavePrompt)
	e.saveThen = nil
	if step.workspace {
		return e.modAction("folder-open", step.source)
	}
	if step.open {
		return e.openMap(step.source)
	}
	e.exitAsked = true
	return ebiten.Termination
}

// discardLeave answers an open question that holds a map open as its open
// button does: the map opens and the unsaved changes are dropped.
func (e *MapEditor) discardLeave() error {
	if !e.prompt.open() || e.prompt.kind != promptLeave || !e.prompt.then.open {
		return nil
	}
	return e.runLeave(e.prompt.then)
}

// leaveSaveAs starts the existing Save As route and runs the step once the
// map is saved.
func (e *MapEditor) leaveSaveAs() error {
	step := e.prompt.then
	e.leave(leavePrompt)
	e.saveThen = &step
	return e.keySaveAs("")
}

// leaveSaved runs the step a Save As carried, when it had one.
func (e *MapEditor) leaveSaved(step *leaveStep) error {
	if step == nil {
		return nil
	}
	return e.runLeave(*step)
}

// leaveSpec is the question as a popup description.
func (e *MapEditor) leaveSpec() popupSpec {
	title, proceed := e.word("editor.leave.title.close"), e.word("editor.leave.close")
	if e.prompt.then.open {
		title, proceed = e.word("editor.leave.title.open"), e.word("editor.leave.open")
	}
	name := ""
	if e.doc != nil {
		name = e.leaveName(e.doc.Source)
	}
	return e.questionSpec(title, []string{fmt.Sprintf(e.word("editor.leave.text"), name)},
		[]string{e.word("editor.leave.saveas"), proceed, e.word("editor.button.cancel")})
}

// leaveName is the map as the question names it: a file by its own name, a
// scenario or new map by its source.
func (e *MapEditor) leaveName(source string) string {
	if k := ParseMapSource(source).Kind; k != SourceScenario && k != SourceNew {
		source = filepath.Base(filepath.FromSlash(source))
	}
	return e.editorHostText(source)
}

// leavePress is a press on the question.
func (e *MapEditor) leavePress(p image.Point) {
	s, _ := e.promptSpec()
	part, n := layoutPopup(s, e.font, e.bodyFont()).hit(p)
	switch {
	case part == popupClose || part == popupButton && n == leaveStay:
		e.cancelPopup()
	case part == popupButton && n == leaveSave:
		_ = e.leaveSaveAs()
	case part == popupButton && n == leaveGo:
		_ = e.runLeave(e.prompt.then)
	}
}
