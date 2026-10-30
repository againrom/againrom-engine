package ui

import "fmt"

// InspectionModSave is implemented by edits whose host saves into a mod
// folder. While ModTarget names a mod, Save (Ctrl+S) writes the map to the
// path the mod's mission entry for it names instead of asking for a new path.
type InspectionModSave interface {
	ModTarget() string
	SaveIntoMod() (string, error)
}

// modSave is the mod save of the open document, when its host names a mod.
func (e *MapEditor) modSave() (InspectionModSave, bool) {
	if !e.editable() {
		return nil, false
	}
	m, ok := e.doc.Edits.(InspectionModSave)
	if !ok || m.ModTarget() == "" {
		return nil, false
	}
	return m, true
}

// saveIntoMod is Save into the host's mod; a refusal stays on screen.
func (e *MapEditor) saveIntoMod(m InspectionModSave) error {
	if e.deferSave(func() error { return e.saveIntoMod(m) }) {
		return nil
	}
	then, errs := e.saveThen, e.saveErrors
	e.saveThen, e.saveErrors = nil, 0
	e.leave(leaveDrag | leavePick | leavePrompt)
	source, err := e.writeSave(m.SaveIntoMod)
	if err != nil {
		if r, ok := refusal(err); ok {
			if r.Ask {
				e.askSaveWithErrors(r, func() error {
					e.saveThen = then
					return e.saveIntoMod(m)
				})
				return err
			}
			e.refuseSave(r)
			return err
		}
		e.message = err.Error()
		return err
	}
	e.doc.Source = source
	e.leave(leavePrompt)
	e.message = ""
	e.afterSave()
	e.noteSave(errs)
	return e.leaveSaved(then)
}

// saveHelp is the help line of the history and save keys; Save goes into the
// mod when one is named.
func (e *MapEditor) saveHelp() string {
	if m, ok := e.modSave(); ok {
		return editorClip(e.font, fmt.Sprintf(e.word("editor.keys.modsave"), m.ModTarget()), 300)
	}
	return "Ctrl+Z / Ctrl+Y / Ctrl+S"
}
