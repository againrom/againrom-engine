package ui

import (
	"fmt"
	"strconv"
)

// firstGlobal is the text of the first global the mod declares for parameter
// k of item it, 0 when it declares none.
func (e *MapEditor) firstGlobal(condition bool, it wizItem, k int) string {
	for _, c := range e.registryChoices(condition, it, k, 0) {
		if c.Value != createChoice(ScriptGlobalParameter) && c.Value != 0 {
			return strconv.FormatInt(c.Value, 10)
		}
	}
	return "0"
}

// globalChoiceField reports that field f offers the choice that adds a global.
func (e *MapEditor) globalChoiceField(f int) bool {
	o := &e.obj
	if f < 0 || f >= len(o.fields) {
		return false
	}
	for _, c := range o.fields[f].Choices {
		if c.Value == createChoice(ScriptGlobalParameter) {
			return true
		}
	}
	return false
}

// newGlobal adds a global to the mod through the Globals page's own adder
// and puts it in field f; the other global fields list it too. The page
// renames it.
func (e *MapEditor) newGlobal(f int) error {
	had := map[uint32]bool{}
	for _, t := range e.doc.Script.Targets[ScriptGlobalParameter] {
		had[t.Value] = true
	}
	next, err := e.commandAddGlobal()
	if err != nil {
		e.obj.note = e.editorHostText(err.Error())
		return err
	}
	if next == nil || next.Script == nil {
		return errNoEditHost
	}
	targets := next.Script.Targets[ScriptGlobalParameter]
	e.doc.Script.Targets[ScriptGlobalParameter] = targets
	var made *InspectionScriptTarget
	for i, t := range targets {
		if t.NewChoice != createChoice(ScriptGlobalParameter) && !had[t.Value] {
			made = &targets[i]
		}
	}
	if made == nil {
		return errNoEditHost
	}
	o := &e.obj
	for i := range o.fields {
		if e.globalChoiceField(i) {
			cur, _ := strconv.ParseInt(o.buf[i], 10, 64)
			o.fields[i].Choices = e.parameterChoices(ScriptGlobalParameter, cur)
		}
	}
	o.buf[f] = strconv.FormatUint(uint64(made.Value), 10)
	o.note = e.editorHostText(fmt.Sprintf(e.word("editor.globals.added"), made.Name))
	return nil
}
