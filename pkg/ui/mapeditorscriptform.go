package ui

import (
	"errors"
	"image"
)

// The script form question. An edit on an original map that only the
// Againrom script record holds (a fourth pair, an OR group, a fifth action,
// a variable past the free registers, a trigger past 1000) comes back from
// the host as *ScriptFormNeeded. The editor asks once per opened map: Yes
// extends the script as its own history entry and runs the edit again; No
// changes nothing. A save of an extended map under an .alm name that the
// original records hold again asks to fold the script first.

// formPrompt is what the question holds: the host's answer and the edit or
// save it runs again once the form changed.
type formPrompt struct {
	need  *ScriptFormNeeded
	again func()
}

// askScriptForm takes err when it is the host's form answer: on an original
// map already agreed to extend it extends at once and runs again, else it
// asks. It reports whether it took err.
func (e *MapEditor) askScriptForm(err error, again func()) bool {
	var need *ScriptFormNeeded
	if !errors.As(err, &need) {
		return false
	}
	if !need.Fold && e.formAgreed {
		e.changeScriptForm(&formPrompt{need: need, again: again})
		return true
	}
	e.openPrompt(editorPrompt{kind: promptScriptForm, form: &formPrompt{need: need, again: again}})
	return true
}

// againNote shows note, the refusal of an edit run again once the form
// changed, as the editor's message.
func (e *MapEditor) againNote(note string) {
	if note != "" {
		e.message = note
	}
}

// scriptFormSpec is the question: what the change of form means for the
// original game, in the editor's words.
func (e *MapEditor) scriptFormSpec() popupSpec {
	if e.prompt.form.need.Fold {
		return e.questionSpec(e.word("editor.scriptform.foldtitle"), []string{e.word("editor.scriptform.foldtext")},
			[]string{e.word("editor.scriptform.foldgo"), e.word("editor.button.cancel")})
	}
	return e.questionSpec(e.word("editor.scriptform.title"), []string{e.word("editor.scriptform.text")},
		[]string{e.word("editor.scriptform.go"), e.word("editor.button.cancel")})
}

// scriptFormPress is a press on the question.
func (e *MapEditor) scriptFormPress(p image.Point) {
	s, _ := e.promptSpec()
	switch part, n := layoutPopup(s, e.font, e.bodyFont()).hit(p); {
	case part == popupClose || part == popupButton && n == 1:
		e.cancelPopup()
	case part == popupButton && n == 0:
		e.scriptFormGo()
	}
}

// scriptFormGo is Yes: the form changes and the held edit runs again.
func (e *MapEditor) scriptFormGo() {
	q := e.prompt.form
	e.leave(leavePrompt)
	e.changeScriptForm(q)
}

// changeScriptForm changes the map's script form as q asks and runs its
// edit again; a refusal is the message.
func (e *MapEditor) changeScriptForm(q *formPrompt) {
	if _, err := e.commandScriptForm(q.need.Fold); err != nil {
		e.message = e.editorHostText(err.Error())
		return
	}
	if !q.need.Fold {
		e.formAgreed = true
	}
	q.again()
}
