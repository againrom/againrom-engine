package ui

import (
	"fmt"
	"slices"
	"strconv"
	"strings"
)

// The Add window's script steps the editor offers and checks itself, from
// the wizard's item table and the script it holds: a condition is a wizard
// item's check placed in a trigger's first free pair as the item's row, a
// condition no pair names goes into a trigger the same way, and a variable
// is a labelled register the host's allocator gives.

// InspectionVariableLabels is implemented by a host that keeps the
// mapmaker's labels of variable registers. NewVariable labels the register
// the host offers as the new variable and answers it; LabelVariable sets a
// variable's label, an empty label clearing it. Each is one history entry.
type InspectionVariableLabels interface {
	NewVariable(label string) (*InspectionDocument, uint32, error)
	LabelVariable(register uint32, label string) (*InspectionDocument, error)
}

func (e *MapEditor) variableLabels() (InspectionVariableLabels, bool) {
	if !e.editable() {
		return nil, false
	}
	h, ok := e.doc.Edits.(InspectionVariableLabels)
	return h, ok
}

// addKindOpen reports whether the Add window adds kind on this map: a
// condition needs the script the editor reads, a variable also a host that
// keeps labels; every other kind is the host's.
func (e *MapEditor) addKindOpen(kind RecordKind) bool {
	switch kind {
	case KindCondition:
		return e.doc != nil && e.doc.Script != nil
	case KindVariable:
		_, ok := e.variableLabels()
		return ok && e.doc.Script != nil
	}
	return true
}

// AddListLines are the rows of the open Add window's list as text, in the
// font's bytes, for a headless driver; nil while it shows none.
func (e *MapEditor) AddListLines() []string {
	r, ok := e.addRunning()
	if !ok || r.list == nil {
		return nil
	}
	var out []string
	for _, row := range r.list.Rows() {
		out = append(out, row.Text)
	}
	return out
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

// triggerFull is, in the editor's words, why trigger takes no new condition
// pair: it has none free. It is empty for a trigger that takes one and for a
// new trigger (-1).
func (e *MapEditor) triggerFull(trigger int) string {
	if e.doc.Script != nil && trigger >= 0 && trigger < len(e.doc.Script.Triggers) && !freePair(e.doc.Script.Triggers[trigger]) {
		return fmt.Sprintf(e.word("editor.area.full"), trigger+1)
	}
	return ""
}

// addStepText is the text of r's step key, "" when r has none.
func (e *MapEditor) addStepText(r *addRun, key string) string {
	for i, st := range r.steps {
		if st.Key == key {
			return e.addText(r, i)
		}
	}
	return ""
}

// addItemOf is the condition item r's item step names, -1 for none.
func (e *MapEditor) addItemOf(r *addRun) int {
	for i, st := range r.steps {
		if st.Control != addItem {
			continue
		}
		if n, err := strconv.Atoi(strings.TrimSpace(e.addText(r, i))); err == nil && n >= 0 && n < len(wizConditions) {
			return n
		}
	}
	return -1
}

// addParamOf is the parameter of r's item step i asks, false for a step
// that asks none.
func addParamOf(st addStep) (int, bool) {
	k, err := strconv.Atoi(strings.TrimPrefix(st.Key, "p"))
	return k, err == nil && strings.HasPrefix(st.Key, "p")
}

// itemSteps are r's steps once its item step at names an item: the steps up
// to it, the item's parameters and comparison as its wizard row asks them,
// then the kind's steps after the item step.
func (e *MapEditor) itemSteps(r *addRun, at int) []addStep {
	out := append([]addStep(nil), r.steps[:at+1]...)
	if item := e.addItemOf(r); item >= 0 {
		it := wizConditions[item]
		defaults := e.wizardDefaults(true, item)
		for k, p := range it.params {
			control := addChoice
			if e.paramField(true, item, k, defaults[k+1]).Text {
				control = addName
			}
			out = append(out, addStep{Key: fmt.Sprintf("p%d", k), Word: p.labelWord, Control: control, Editor: true})
		}
		if it.chooseCmp {
			out = append(out, addStep{Key: "cmp", Word: "editor.trigger.comparison", Control: addChoice, Editor: true})
		}
	}
	row, _ := addKindOf(r.kind)
	for k, st := range row.Steps {
		if st.Control == addItem {
			return append(out, row.Steps[k+1:]...)
		}
	}
	return out
}

// scriptStepOffer is what the editor offers its step i of r: the wizard's
// condition items under their headings, a parameter or the comparison as
// the item's wizard row offers it with the row's default, the triggers, or
// the variable's label.
func (e *MapEditor) scriptStepOffer(r *addRun, i int) InspectionNewStep {
	st := r.steps[i]
	switch {
	case st.Control == addItem:
		choices, shut := e.itemChoices(&wizardState{rows: []wizRow{{condition: true, item: -2, origItem: -2}}}, 0)
		offer := InspectionNewStep{Choices: choices, Shut: shut}
		for _, c := range choices {
			if !slices.Contains(shut, c.Value) {
				offer.Value = c.Value
				break
			}
		}
		return offer
	case st.Key == addTrigger.Key:
		return InspectionNewStep{Choices: e.triggerChoices(), Value: -1}
	case st.Key == addLabel.Key:
		if r.relabel != nil {
			for _, v := range e.scriptVariables() {
				if v.register == *r.relabel {
					return InspectionNewStep{Text: v.label}
				}
			}
		}
		return InspectionNewStep{Text: e.plainWord("editor.add.variable")}
	}
	item := e.addItemOf(r)
	if item < 0 {
		return InspectionNewStep{}
	}
	defaults := e.wizardDefaults(true, item)
	f := e.cmpField(defaults[len(defaults)-1])
	if k, ok := addParamOf(st); ok && k < len(wizConditions[item].params) {
		f = e.paramField(true, item, k, defaults[k+1])
	}
	return InspectionNewStep{Choices: f.Choices, Value: f.Value, Text: f.TextValue}
}

// scriptStepRefusal is why the editor refuses text for its step i of r, in
// its words; empty when the step takes it. A parameter is checked as the
// wizard's Apply checks it, a trigger as the area tool does.
func (e *MapEditor) scriptStepRefusal(r *addRun, i int, text string) string {
	st := r.steps[i]
	n, err := strconv.ParseInt(strings.TrimSpace(text), 10, 64)
	choose := e.word("editor.add.choose")
	switch {
	case st.Control == addItem:
		if err != nil || n < 0 || n >= int64(len(wizConditions)) || slices.Contains(r.offer.Shut, n) {
			return choose
		}
	case st.Key == addTrigger.Key:
		if err != nil || n < -1 || e.doc.Script == nil || n >= int64(len(e.doc.Script.Triggers)) {
			return choose
		}
		return e.triggerFull(int(n))
	case st.Key == addLabel.Key:
	case st.Key == "cmp":
		if err != nil || n < 0 || n > 5 {
			return choose
		}
	default:
		item := e.addItemOf(r)
		k, ok := addParamOf(st)
		if item < 0 || !ok || k >= len(wizConditions[item].params) {
			return choose
		}
		if _, err := e.wizardParam(wizConditions[item], k, text); err != nil {
			return err.Error()
		}
	}
	return ""
}

// scriptAddFinish ends a run the editor finishes itself, a condition or a
// variable, and reports false for any other run.
func (e *MapEditor) scriptAddFinish(r *addRun) bool {
	switch r.kind {
	case KindCondition:
		e.addConditionFinish(r)
	case KindVariable:
		e.addVariableFinish(r)
	default:
		return false
	}
	return true
}

// addConditionFinish places the run's condition in the chosen trigger. A
// refusal keeps the window on its last step with the reason.
func (e *MapEditor) addConditionFinish(r *addRun) {
	trigger, _ := strconv.Atoi(strings.TrimSpace(e.addStepText(r, addTrigger.Key)))
	note := e.triggerFull(trigger)
	if note == "" && r.listed != 0 {
		note = e.listedToTrigger(r.listed, trigger)
	} else if note == "" {
		item := e.addItemOf(r)
		if item < 0 {
			r.note = e.word("editor.add.choose")
			return
		}
		it := wizConditions[item]
		row := wizRow{condition: true, item: item, origItem: -2, vals: []string{strconv.Itoa(item)}}
		for k := range it.params {
			row.vals = append(row.vals, e.addStepText(r, fmt.Sprintf("p%d", k)))
		}
		if it.chooseCmp {
			row.vals = append(row.vals, e.addStepText(r, "cmp"))
		}
		note = e.placeCheck(trigger, row)
	}
	if note != "" {
		r.note = note
	}
}

// placeCheck writes row into trigger's first free pair, -1 a new trigger,
// as the wizard's Apply writes a new row of its item, then shows the
// trigger. It answers the refusal, shown as it stands.
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
// with a new constant, and the trigger then shows its Details.
func (e *MapEditor) listedToTrigger(id uint32, trigger int) string {
	n, ok := e.nodeByID(true, id)
	if !ok {
		return e.word("editor.add.choose")
	}
	if row, ok := e.loneRow(n); ok {
		return e.placeCheck(trigger, row)
	}
	pair := 0
	if trigger >= 0 {
		t := e.doc.Script.Triggers[trigger]
		for pair < 2 && (t.Left[pair] != 0 || t.Right[pair] != 0) {
			pair++
		}
	}
	values := []InspectionFieldValue{{Key: fmt.Sprintf("left%d", pair), Value: int64(id)},
		{Key: fmt.Sprintf("cmp%d", pair), Value: 0}, {Key: fmt.Sprintf("right%d", pair), Value: -1, New: true}}
	var ch editorChange
	var err error
	if trigger < 0 {
		ch, err = e.commandAddRecord(InspectionNewRecord{Kind: KindTrigger, Values: values})
	} else {
		ch, err = e.commandSetFields(recordFields{Kind: KindTrigger, Index: trigger, Values: values})
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
	return v.node != nil && v.condition && v.node.ID != 0 && slices.Contains(e.scriptSections()[2], v.node.Record)
}

// addToTrigger opens the Add window on the trigger a listed condition goes
// into.
func (e *MapEditor) addToTrigger(v scriptView) {
	if e.objectDirty() {
		e.obj.note = e.word("editor.trigger.applyfirst")
		return
	}
	run := &addRun{kind: KindCondition, title: e.word("editor.scripts.totrigger"), listed: v.node.ID}
	run.setSteps([]addStep{addTrigger})
	e.openAddRun(run)
}

// addVariableFinish makes the run's new variable, or labels the variable it
// relabels, and shows it.
func (e *MapEditor) addVariableFinish(r *addRun) {
	_, register, err := e.commandLabelVariable(r.relabel, e.addStepText(r, addLabel.Key))
	if err != nil {
		r.note = e.editorHostText(err.Error())
		return
	}
	e.leave(leavePrompt)
	e.selectVariable(register)
}

// commandLabelVariable labels register, nil the host's new variable, and
// answers the register labelled.
func (e *MapEditor) commandLabelVariable(register *uint32, label string) (editorChange, uint32, error) {
	host, ok := e.variableLabels()
	if !ok {
		return editorChange{}, 0, errNoEditHost
	}
	var next *InspectionDocument
	var err error
	var r uint32
	if register == nil {
		next, r, err = host.NewVariable(label)
	} else {
		r = *register
		next, err = host.LabelVariable(r, label)
	}
	if err != nil {
		return editorChange{}, 0, err
	}
	ch, err := e.adopted(next, nil)
	return ch, r, err
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
// asks the label in the Add window, Clear label clears it, Cancel closes.
func (e *MapEditor) variableButton(v scriptVariableView, n int) {
	if _, ok := e.variableLabels(); !ok {
		e.cancelPopup()
		return
	}
	switch n {
	case variableLabelButton:
		register := v.register
		run := &addRun{kind: KindVariable, title: fmt.Sprintf(e.word("editor.labels.title"), v.register), relabel: &register}
		run.setSteps([]addStep{addLabel})
		e.openAddRun(run)
	case variableClearButton:
		if v.label == "" {
			return
		}
		if _, _, err := e.commandLabelVariable(&v.register, ""); err != nil {
			e.message = err.Error()
			return
		}
		e.selectVariable(v.register)
	default:
		e.cancelPopup()
	}
}
