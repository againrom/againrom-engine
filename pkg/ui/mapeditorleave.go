package ui

import (
	"fmt"
	"image"
	"path/filepath"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
)

// leaveStep is what the editor was about to do when unsaved changes stopped
// it: close the window, or open the map named source.
type leaveStep struct {
	open   bool
	source string
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
	if step.open {
		return e.openMap(step.source)
	}
	e.exitAsked = true
	return ebiten.Termination
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
	w := min(560, e.window.X-16)
	s := popupSpec{Title: title, Width: w, Bounds: e.popupBounds(), NoteInk: editorWarning, LabelWidth: 1}
	s.Buttons = []string{e.word("editor.leave.saveas"), proceed, e.word("editor.button.cancel")}
	if f := e.bodyFont(); f != nil {
		for _, b := range s.Buttons {
			s.ButtonW = max(s.ButtonW, f.Advance(b)+16)
		}
		s.Width = min(max(s.Width, 3*s.ButtonW+4*popupPad), e.window.X-16)
	}
	name := ""
	if e.doc != nil {
		name = e.leaveName(e.doc.Source)
	}
	text := s.Width - 2*popupPad - 60
	s.Notes = append(s.Notes, editorWrap(e.bodyFont(), fmt.Sprintf(e.word("editor.leave.text"), name), text)...)
	l := layoutPopup(s, e.font, e.bodyFont())
	s.At = image.Pt((e.window.X-l.Rect.Dx())/2, max(e.barHeight()+8, (e.window.Y-l.Rect.Dy())/3))
	return s
}

// leaveName is the map as the question names it: a file by its own name, a
// scenario or new map by its source.
func (e *MapEditor) leaveName(source string) string {
	if !strings.HasPrefix(source, "scenario/") && !strings.HasPrefix(source, "new/") {
		source = filepath.Base(filepath.FromSlash(source))
	}
	return e.editorHostText(source)
}

// leavePress is a press on the question.
func (e *MapEditor) leavePress(p image.Point) {
	l := layoutPopup(e.leaveSpec(), e.font, e.bodyFont())
	part, n := l.hit(p)
	switch {
	case part == popupClose || part == popupButton && n == leaveStay:
		_ = e.Key("escape")
	case part == popupButton && n == leaveSave:
		_ = e.leaveSaveAs()
	case part == popupButton && n == leaveGo:
		_ = e.runLeave(e.prompt.then)
	}
}
