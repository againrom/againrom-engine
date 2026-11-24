package ui

import (
	"fmt"
	"slices"
	"strconv"
	"strings"

	"againrom/pkg/scriptprofile"
)

// The script records the editor places itself: a condition no pair names
// goes into a trigger from its window's Trigger row, and a variable's label
// is asked in a field prompt.

// InspectionVariableLabels is implemented by a host that keeps the
// mapmaker's labels of variable registers. LabelVariable sets a variable's
// label, an empty label clearing it, as one history entry.
type InspectionVariableLabels interface {
	LabelVariable(register uint32, label string) (*InspectionDocument, error)
}

func (e *MapEditor) variableLabels() (InspectionVariableLabels, bool) {
	if !e.editable() {
		return nil, false
	}
	h, ok := Seam[InspectionVariableLabels](e.doc.Edits)
	return h, ok
}

// addKindOpen reports whether a one-click add adds kind on this map: a
// condition needs the script the editor reads; every other kind is the
// host's.
func (e *MapEditor) addKindOpen(kind RecordKind) bool {
	if !e.offersKind(kind) {
		return false
	}
	switch kind {
	case KindCondition:
		return e.doc != nil && e.doc.Script != nil
	}
	return true
}

// triggerChoices are the triggers a new condition pair can go into: a new
// trigger (-1) first, then each trigger by number and name.
func (e *MapEditor) triggerChoices() []InspectionChoice {
	out := []InspectionChoice{{-1, e.word("editor.area.newtrigger")}}
	if e.doc.Script != nil {
		for i, t := range e.doc.Script.Triggers {
			out = append(out, InspectionChoice{int64(i), fmt.Sprintf("T%d %s", i+1, e.editorHostText(t.Name))})
		}
	}
	return out
}

// triggerFull is, in the editor's words, why trigger takes no new condition:
// it is an extended map's trigger holding all the record does. On an original
// map a full trigger is no refusal, since the host asks to extend the map. It
// is empty for a trigger that takes one and for a new trigger (-1).
func (e *MapEditor) triggerFull(trigger int) string {
	s := e.doc.Script
	if s != nil && s.Profile().Groups && trigger >= 0 && trigger < len(s.Triggers) && !freePair(s.Triggers[trigger], s) {
		return fmt.Sprintf(e.word("editor.area.full"), trigger+1)
	}
	return ""
}

// placeCheck writes row into trigger's first free pair, -1 a new trigger,
// as the wizard's Apply writes a new row of its item, then shows the
// trigger. It answers the refusal, shown as it stands. A pair the original
// record cannot hold asks to extend the map first.
func (e *MapEditor) placeCheck(trigger int, row wizRow) string {
	var t WizardTrigger
	var err error
	if trigger < 0 {
		t, err = e.rowsPlan(WizardTrigger{Index: -1, Once: 1}, []wizRow{row})
	} else {
		t, err = e.keptPlan(trigger, row)
	}
	if err != nil {
		return err.Error() // the editor's words, encoded
	}
	ch, err := e.commandWriteTrigger(t)
	if e.askScriptForm(err, func() { e.againNote(e.placeCheck(trigger, row)) }) {
		return ""
	}
	if err != nil {
		return e.editorHostText(err.Error())
	}
	e.leave(leavePrompt)
	if ch.record != nil {
		e.selectTrigger(ch.record.Index, false)
	}
	return ""
}

// listedToTrigger puts condition id, which no pair names, into trigger's
// first free pair, -1 a new trigger: as the row of the wizard item that
// reads it when one does, else as Details enters a pair, id compared by ==
// with a new constant, and the trigger then shows its Details. A pair the
// original record cannot hold asks to extend the map first.
func (e *MapEditor) listedToTrigger(id uint32, trigger int) string {
	n, ok := e.nodeByID(true, id)
	if !ok {
		return e.word("editor.trigger.choosefirst")
	}
	if row, ok := e.loneRow(n); ok {
		return e.placeCheck(trigger, row)
	}
	pair := 0
	if trigger >= 0 {
		t := e.doc.Script.Triggers[trigger]
		for pair < len(t.Left) && (t.Left[pair] != 0 || t.Right[pair] != 0) {
			pair++
		}
	}
	values := []InspectionFieldValue{{Key: scriptprofile.LeftKey(pair), Value: int64(id)},
		{Key: scriptprofile.CmpKey(pair), Value: 0}, {Key: scriptprofile.RightKey(pair), Value: -1, New: true}}
	var ch editorChange
	var err error
	if trigger < 0 {
		ch, err = e.commandAddRecord(InspectionNewRecord{Kind: KindTrigger, Values: values})
	} else {
		ch, err = e.commandSetFields(recordFields{Kind: KindTrigger, Index: trigger, Values: values})
	}
	if e.askScriptForm(err, func() { e.againNote(e.listedToTrigger(id, trigger)) }) {
		return ""
	}
	if err != nil {
		return e.editorHostText(err.Error())
	}
	e.leave(leavePrompt)
	if ch.record != nil {
		e.selectTrigger(ch.record.Index, false)
		e.ShowTriggerDetails(true)
	}
	return ""
}

// listedCondition reports a condition the Conditions list shows: no pair
// names it and no variable is preset by it.
func (e *MapEditor) listedCondition(v scriptView) bool {
	return v.node != nil && v.condition && v.node.ID != 0 && e.nodeRule().listed(e, v)
}

// triggerRowPlacing is what Apply does once the fields of v are written: a
// condition no pair names goes into the trigger its Trigger row names
// (listedToTrigger); any other record does nothing more. The row is read
// before the fields are written, since writing them reloads the window.
func (e *MapEditor) triggerRowPlacing(v scriptView) func() {
	o := &e.obj
	at := slices.IndexFunc(o.fields, func(f InspectionField) bool { return f.Key == nodeTriggerKey })
	if at < 0 || v.node == nil || e.doc.Script == nil {
		return func() {}
	}
	trigger, err := strconv.Atoi(strings.TrimSpace(o.buf[at]))
	if err != nil || trigger < -1 || trigger >= len(e.doc.Script.Triggers) {
		return func() {}
	}
	id := v.node.ID
	return func() {
		note := e.triggerFull(trigger)
		if note == "" {
			note = e.listedToTrigger(id, trigger)
		}
		if note != "" {
			e.loadObject()
			e.obj.note = note
		}
	}
}

// labelVariablePrompt labels the variable the label prompt asks for with
// the typed text and shows it; a refusal keeps the prompt with the reason.
func (e *MapEditor) labelVariablePrompt() error {
	_, register, err := e.commandLabelVariable(e.prompt.register, e.prompt.path)
	if err != nil {
		e.message = e.editorHostText(err.Error())
		return nil
	}
	e.leave(leavePrompt)
	e.selectVariable(register)
	return nil
}

// commandLabelVariable labels register and answers it.
func (e *MapEditor) commandLabelVariable(register uint32, label string) (editorChange, uint32, error) {
	host, ok := e.variableLabels()
	if !ok {
		return editorChange{}, 0, errNoEditHost
	}
	next, err := host.LabelVariable(register, label)
	if err != nil {
		return editorChange{}, 0, err
	}
	ch, err := e.adopted(next, nil)
	return ch, register, err
}

// selectVariable selects the Variables row of register, or nothing when no
// row lists it.
func (e *MapEditor) selectVariable(register uint32) {
	for i, v := range e.scriptVariables() {
		if v.register == register {
			e.Select(-2-i, false)
			return
		}
	}
	e.selected = -1
}

// The variable window's buttons, in order, on a host that keeps labels.
const (
	variableLabelButton = iota
	variableClearButton
)

// variableButton is a press on the variable window's button row: Label
// asks the label in a field prompt, Clear label clears it, Cancel closes.
func (e *MapEditor) variableButton(v scriptVariableView, n int) {
	if _, ok := e.variableLabels(); !ok {
		e.cancelPopup()
		return
	}
	switch n {
	case variableLabelButton:
		e.leave(leaveDrag | leavePick | leavePrompt)
		e.openPrompt(editorPrompt{kind: promptVariableLabel, register: v.register, path: v.label})
	case variableClearButton:
		if v.label == "" {
			return
		}
		if _, _, err := e.commandLabelVariable(v.register, ""); err != nil {
			e.message = err.Error()
			return
		}
		e.selectVariable(v.register)
	default:
		e.cancelPopup()
	}
}
