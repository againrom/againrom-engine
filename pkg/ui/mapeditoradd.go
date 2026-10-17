package ui

import (
	"fmt"
	"image"
	"strconv"
	"strings"

	"againrom/pkg/render/textfit"
)

// The Add window is the one builder of every new record. A record kind's
// steps are data (addKinds); one popup asks them top to bottom, each step one
// value with one control. The host offers each step's choices and default,
// checks each value, and adds the record from every value as one history
// entry. The window then closes and the new record's popup opens.

// InspectionNewRecord is a new record as the Add window has asked it so far:
// its kind, the cell once a step chose one, and every other step's value in
// step order. A step that chooses several values gives one entry each.
type InspectionNewRecord struct {
	Kind    string
	Cell    image.Point
	HasCell bool
	Values  []InspectionFieldValue
}

// InspectionNewStep is what the host offers one step: its choices, none for
// a typed value, and the value or text the step starts at.
type InspectionNewStep struct {
	Choices []InspectionChoice
	Value   int64
	Text    string
}

// InspectionNewRecords is implemented by a host that adds records through
// the Add window. NewRecordStep offers step key given the values before it;
// CheckNewRecord refuses the value r holds for key, with the reason;
// NewRecordPreview is a choice's picture, nil for none; AddRecord adds r as
// one history entry and changes nothing when it refuses.
type InspectionNewRecords interface {
	NewRecordStep(r InspectionNewRecord, key string) InspectionNewStep
	CheckNewRecord(r InspectionNewRecord, key string) error
	NewRecordPreview(kind, key string, value int64) *image.RGBA
	AddRecord(r InspectionNewRecord) (*InspectionDocument, error)
}

// addControl is the control a step asks its value with.
type addControl uint8

const (
	addChoice  addControl = iota // one of the host's choices, from a list
	addSprites                   // a choice whose highlighted row shows its picture
	addNumber                    // a typed whole number
	addName                      // typed text
	addCell                      // a cell picked on the map or typed x,y
	addMany                      // one or more of the host's choices
	addKind                      // the record kind the window goes on as
)

// addStep is one step: the host's key for its value, its label's word and
// its control. An optional step is left out when the host offers it fewer
// than two choices; a kind step offers Kinds.
type addStep struct {
	Key      string
	Word     string
	Control  addControl
	Optional bool
	Kinds    []string
}

// addKindRow is one record kind the window adds: the kind the host adds, the
// title word, the word a kind step offers it by, the list views whose Add
// button opens it and its steps.
type addKindRow struct {
	Kind, Word string
	ChoiceWord string
	Filters    []int
	Steps      []addStep
}

var (
	addOwner = addStep{Key: "owner", Word: "editor.add.owner", Control: addChoice}
	addAt    = addStep{Key: "cell", Word: "editor.object.cell", Control: addCell}
	addNamed = addStep{Key: "name", Word: "editor.add.name", Control: addName}
)

// addKinds is every kind the Add window creates, as data.
var addKinds = []addKindRow{
	{Kind: "Unit", Word: "editor.add.unit", Filters: []int{1}, Steps: []addStep{
		{Key: "class", Word: "editor.add.class", Control: addSprites},
		{Key: "tier", Word: "editor.unit.tier", Control: addChoice, Optional: true},
		addOwner, addAt}},
	{Kind: "Structure", Word: "editor.add.structure", Filters: []int{2}, Steps: []addStep{
		{Key: "kind", Word: "editor.add.kind", Control: addSprites}, addOwner, addAt}},
	{Kind: "Sack", Word: "editor.add.sack", Filters: []int{3}, Steps: []addStep{
		addAt, {Key: "item", Word: "editor.add.items", Control: addMany}}},
	{Kind: "Group", Word: "editor.add.group", Filters: []int{editorGroupsFilter}, Steps: []addStep{
		addOwner, {Key: "group", Word: "editor.add.number", Control: addNumber},
		{Key: "member", Word: "editor.group.members", Control: addMany}}},
	{Kind: "Player", Word: "editor.add.player", Filters: []int{editorPlayersFilter}, Steps: []addStep{
		addNamed, {Key: "colour", Word: "editor.add.colour", Control: addNumber},
		{Key: "control", Word: "editor.add.control", Control: addChoice}}},
	{Kind: "Trigger", Word: "editor.add.trigger", ChoiceWord: "editor.add.kind.trigger", Filters: []int{7}, Steps: []addStep{
		addNamed, {Key: "once", Word: "editor.trigger.once", Control: addChoice}}},
	{Kind: "Script", Word: "editor.add.script", Filters: []int{editorScriptsFilter}, Steps: []addStep{
		{Key: "kind", Word: "editor.add.record", Control: addKind, Kinds: []string{"Action", "Condition", "Trigger"}}}},
	{Kind: "Action", Word: "editor.add.action", ChoiceWord: "editor.add.kind.action"},
	{Kind: "Condition", Word: "editor.add.condition", ChoiceWord: "editor.add.kind.condition"},
}

func addKindOf(kind string) (addKindRow, bool) {
	for _, k := range addKinds {
		if k.Kind == kind {
			return k, true
		}
	}
	return addKindRow{}, false
}

// addKindFor is the kind the list view filter adds.
func addKindFor(filter int) (addKindRow, bool) {
	for _, k := range addKinds {
		for _, f := range k.Filters {
			if f == filter {
				return k, true
			}
		}
	}
	return addKindRow{}, false
}

// addRun is an open Add window: the kind it adds, its steps, the step asked,
// each step's text (the asked step's is the prompt's), each many step's
// chosen values, the labels a kept step's choices read as, the steps left
// out, the asked step's offer and list, and why the last step did not go on.
type addRun struct {
	kind       string
	title      string
	steps      []addStep
	at         int
	texts      []string
	many       [][]int64
	labels     []string
	manyLabels [][]string
	skip       []bool
	offer      InspectionNewStep
	list       *Picker
	note       string
}

func (e *MapEditor) addHost() (InspectionNewRecords, bool) {
	if !e.editable() || e.catalog {
		return nil, false
	}
	h, ok := e.doc.Edits.(InspectionNewRecords)
	return h, ok
}

// openAdd opens the Add window on kind's first step.
func (e *MapEditor) openAdd(kind string) {
	row, ok := addKindOf(kind)
	if _, host := e.addHost(); !ok || !host {
		return
	}
	e.leave(leaveDrag | leavePick | leavePrompt | leaveBrush)
	run := &addRun{kind: row.Kind, title: e.word(row.Word)}
	run.setSteps(row.Steps)
	e.openPrompt(editorPrompt{kind: promptAdd, add: run})
	e.enterAddStep(0, 1)
}

func (r *addRun) setSteps(steps []addStep) {
	r.steps = append([]addStep(nil), steps...)
	n := len(r.steps)
	for len(r.texts) < n {
		r.texts, r.many, r.skip = append(r.texts, ""), append(r.many, nil), append(r.skip, false)
		r.labels, r.manyLabels = append(r.labels, ""), append(r.manyLabels, nil)
	}
	r.texts, r.many, r.skip, r.labels, r.manyLabels = r.texts[:n], r.many[:n], r.skip[:n], r.labels[:n], r.manyLabels[:n]
}

func (e *MapEditor) addRunning() (*addRun, bool) {
	r := e.prompt.add
	return r, e.prompt.kind == promptAdd && r != nil
}

// addText is step i's text.
func (e *MapEditor) addText(r *addRun, i int) string {
	if i == r.at {
		return e.prompt.path
	}
	return r.texts[i]
}

// addRecord is the record of every step before upTo that the window kept.
func (e *MapEditor) addRecord(r *addRun, upTo int) InspectionNewRecord {
	out := InspectionNewRecord{Kind: r.kind}
	for i := 0; i < upTo && i < len(r.steps); i++ {
		st, text := r.steps[i], strings.TrimSpace(e.addText(r, i))
		if r.skip[i] {
			continue
		}
		switch st.Control {
		case addKind:
		case addCell:
			if c, ok := parseCell(text); ok {
				out.Cell, out.HasCell = c, true
			}
		case addName:
			out.Values = append(out.Values, InspectionFieldValue{Key: st.Key, Text: true, TextValue: e.addText(r, i)})
		case addMany:
			for _, v := range r.many[i] {
				out.Values = append(out.Values, InspectionFieldValue{Key: st.Key, Value: v})
			}
		default:
			if v, err := strconv.ParseInt(text, 10, 64); err == nil {
				out.Values = append(out.Values, InspectionFieldValue{Key: st.Key, Value: v})
			}
		}
	}
	return out
}

func parseCell(s string) (image.Point, bool) {
	a, b, ok := strings.Cut(s, ",")
	x, ex := strconv.Atoi(strings.TrimSpace(a))
	y, ey := strconv.Atoi(strings.TrimSpace(b))
	return image.Pt(x, y), ok && ex == nil && ey == nil
}

// enterAddStep asks step i, or the next step in direction dir the host
// offers enough choices: its offer, its list and its text, the kept text or
// the offer's default.
func (e *MapEditor) enterAddStep(i, dir int) {
	r, ok := e.addRunning()
	host, hok := e.addHost()
	if !ok || !hok {
		return
	}
	for i >= 0 && i < len(r.steps) {
		st := r.steps[i]
		r.at = i
		var offer InspectionNewStep
		if st.Control == addKind {
			for n, k := range st.Kinds {
				row, _ := addKindOf(k)
				offer.Choices = append(offer.Choices, InspectionChoice{Value: int64(n), Label: e.word(row.ChoiceWord)})
			}
		} else {
			offer = host.NewRecordStep(e.addRecord(r, i), st.Key)
		}
		r.skip[i] = st.Optional && len(offer.Choices) < 2
		if !r.skip[i] {
			r.offer, r.note = offer, ""
			e.prompt.path = r.texts[i]
			if e.prompt.path == "" && st.Control != addCell && st.Control != addMany {
				e.prompt.path = strconv.FormatInt(offer.Value, 10)
				if st.Control == addName {
					e.prompt.path = offer.Text
				}
			}
			e.addList()
			return
		}
		r.texts[i] = ""
		if i += dir; i < 0 {
			i, dir = 0, 1
		}
	}
}

// addList builds the asked step's list, its highlight on the step's value.
func (e *MapEditor) addList() {
	r, _ := e.addRunning()
	r.list = nil
	if len(r.offer.Choices) == 0 {
		return
	}
	rows := make([]PickerRow, len(r.offer.Choices))
	sel := 0
	for k, c := range r.offer.Choices {
		rows[k] = PickerRow{Text: c.Label, Choosable: true}
		if strconv.FormatInt(c.Value, 10) == strings.TrimSpace(e.prompt.path) {
			sel = k
		}
	}
	r.list = NewPicker(rows).SetWindow(min(addListRows, len(rows)))
	r.list.Select(sel)
}

const addListRows = 8

// addChoiceAt is the asked step's choice whose value text holds.
func (r *addRun) choiceAt(text string) (InspectionChoice, bool) {
	v, err := strconv.ParseInt(strings.TrimSpace(text), 10, 64)
	if err != nil {
		return InspectionChoice{}, false
	}
	for _, c := range r.offer.Choices {
		if c.Value == v {
			return c, true
		}
	}
	return InspectionChoice{}, false
}

// addNext checks the asked step's value and asks the next step, or adds the
// record after the last. A refused value names its reason and the step
// stays. On a many step, a typed or chosen value is added to the step and an
// empty text goes on.
func (e *MapEditor) addNext() {
	r, ok := e.addRunning()
	host, hok := e.addHost()
	if !ok || !hok {
		return
	}
	e.message = ""
	st, text := r.steps[r.at], strings.TrimSpace(e.prompt.path)
	switch st.Control {
	case addKind:
		if _, ok := r.choiceAt(text); !ok {
			r.note = e.word("editor.add.choose")
			return
		}
	case addChoice, addSprites:
		// A typed value off the list stands when the host's check passes it.
		if _, err := strconv.ParseInt(text, 10, 64); err != nil {
			r.note = e.word("editor.add.choose")
			return
		}
	case addNumber:
		if _, err := strconv.ParseInt(text, 10, 64); err != nil {
			r.note = e.word("editor.add.whole")
			return
		}
	case addCell:
		c, ok := parseCell(text)
		if !ok || !c.In(image.Rect(0, 0, e.doc.Width, e.doc.Height)) {
			r.note = e.word("editor.add.cellbad")
			return
		}
	case addMany:
		if text != "" {
			e.addManyValue(text)
			return
		}
		if len(r.many[r.at]) == 0 {
			r.note = e.word("editor.add.atleast")
			return
		}
	}
	if st.Control != addKind && st.Control != addMany {
		if err := host.CheckNewRecord(e.addRecord(r, r.at+1), st.Key); err != nil {
			r.note = e.editorHostText(err.Error())
			return
		}
	}
	if r.texts[r.at] != e.prompt.path {
		for i := r.at + 1; i < len(r.steps); i++ {
			r.texts[i], r.many[i] = "", nil
		}
	}
	r.texts[r.at] = e.prompt.path
	r.labels[r.at], r.manyLabels[r.at] = e.addValueText(r, r.at), nil
	for _, v := range r.many[r.at] {
		r.manyLabels[r.at] = append(r.manyLabels[r.at], r.labelOf(v))
	}
	if st.Control == addKind {
		c, _ := r.choiceAt(text)
		next, _ := addKindOf(st.Kinds[c.Value])
		r.kind, r.title = next.Kind, e.word(next.Word)
		r.setSteps(append(append([]addStep(nil), r.steps[:r.at+1]...), next.Steps...))
	}
	if r.at+1 < len(r.steps) {
		e.enterAddStep(r.at+1, 1)
		return
	}
	e.addFinish()
}

// addManyValue adds the value text names to the asked many step, once.
func (e *MapEditor) addManyValue(text string) {
	r, _ := e.addRunning()
	host, _ := e.addHost()
	v, err := strconv.ParseInt(strings.TrimSpace(text), 10, 64)
	if err != nil {
		r.note = e.word("editor.add.choose")
		return
	}
	for _, have := range r.many[r.at] {
		if have == v {
			e.prompt.path = ""
			return
		}
	}
	r.many[r.at] = append(r.many[r.at], v)
	if err := host.CheckNewRecord(e.addRecord(r, r.at+1), r.steps[r.at].Key); err != nil {
		r.many[r.at] = r.many[r.at][:len(r.many[r.at])-1]
		r.note = e.editorHostText(err.Error())
		return
	}
	e.prompt.path, r.note = "", ""
}

// addBack asks the step before the asked one.
func (e *MapEditor) addBack(to int) {
	r, ok := e.addRunning()
	if !ok || to < 0 || to >= r.at {
		return
	}
	r.texts[r.at] = e.prompt.path
	e.enterAddStep(to, -1)
}

// addFirst is the first step the window asks.
func (r *addRun) first() int {
	for i := range r.steps {
		if !r.skip[i] {
			return i
		}
	}
	return 0
}

// addFinish adds the record through the host as one history entry, closes
// the window and opens the new record's popup. A refusal keeps the window on
// its last step with the reason.
func (e *MapEditor) addFinish() {
	r, _ := e.addRunning()
	host, _ := e.addHost()
	rec := e.addRecord(r, len(r.steps))
	next, err := host.AddRecord(rec)
	if err != nil {
		r.note = e.editorHostText(err.Error())
		return
	}
	if e.adoptEdit(next, nil) != nil || next == nil {
		return
	}
	e.leave(leavePrompt)
	if r.kind != "Group" {
		e.selectNewest(r.kind)
		return
	}
	var owner, group int64
	for _, v := range rec.Values {
		switch v.Key {
		case "owner":
			owner = v.Value
		case "group":
			group = v.Value
		}
	}
	for _, g := range e.groups() {
		if int64(g.player) == owner && int64(g.group) == group {
			e.chooseGroup(g)
		}
	}
}

// addMove steps the asked step's list by delta and takes its value.
func (e *MapEditor) addMove(delta int) {
	r, ok := e.addRunning()
	if !ok || r.list == nil {
		return
	}
	if _, ok := r.choiceAt(e.prompt.path); ok || r.steps[r.at].Control == addMany && e.prompt.path != "" {
		dropdownStep(r.list, delta)
	}
	e.addTake(r.list.Selection())
}

// addTake makes choice row k the asked step's value.
func (e *MapEditor) addTake(k int) {
	r, _ := e.addRunning()
	if k < 0 || k >= len(r.offer.Choices) {
		return
	}
	r.list.Select(k)
	e.prompt.path, r.note = strconv.FormatInt(r.offer.Choices[k].Value, 10), ""
}

// addFollow moves the list's highlight to the choice a typed value names.
func (e *MapEditor) addFollow() {
	r, ok := e.addRunning()
	if !ok || r.list == nil {
		return
	}
	for k, c := range r.offer.Choices {
		if strconv.FormatInt(c.Value, 10) == strings.TrimSpace(e.prompt.path) {
			r.list.Select(k)
			return
		}
	}
}

// addPicked takes a cell clicked on the map as the cell step's value.
func (e *MapEditor) addPicked(cell image.Point) {
	if _, ok := e.addRunning(); ok {
		e.message = ""
		e.prompt.path = fmt.Sprintf("%d,%d", cell.X, cell.Y)
	}
}

// addRowAct is what a press on a row of the window does: ask a step again,
// pick its cell, add the highlighted value, or remove a chosen value.
type addRowAct struct {
	step   int
	item   int // a chosen value's position, -1 for the step's own row
	asked  bool
	button string
}

// labelOf is the asked step's label of choice value v.
func (r *addRun) labelOf(v int64) string {
	for _, c := range r.offer.Choices {
		if c.Value == v {
			return c.Label
		}
	}
	return strconv.FormatInt(v, 10)
}

// addValueText is how a step's value reads in its row.
func (e *MapEditor) addValueText(r *addRun, i int) string {
	st, text := r.steps[i], e.addText(r, i)
	switch st.Control {
	case addName, addCell, addNumber:
		if st.Control == addName {
			return e.editorHostText(text)
		}
		return text
	case addMany:
		return fmt.Sprintf("%d", len(r.many[i]))
	}
	if i == r.at {
		if c, ok := r.choiceAt(text); ok {
			return c.Label
		}
		return text
	}
	return r.labels[i]
}

// addSpec is the Add window as data, with what each row does.
func (e *MapEditor) addSpec() (popupSpec, []addRowAct) {
	r, _ := e.addRunning()
	w := min(480, e.window.X-16)
	s := popupSpec{Title: r.title, Width: w, Bounds: e.popupBounds(), NoteInk: editorWarning, PickLabel: e.word("editor.object.pick")}
	var acts []addRowAct
	body := e.bodyFont()
	for i, st := range r.steps {
		if r.skip[i] {
			continue
		}
		row := popupRow{Kind: popupText, Label: e.word(st.Word)}
		act := addRowAct{step: i, item: -1}
		switch {
		case i < r.at:
			row.Value = e.addValueText(r, i)
		case i == r.at:
			act.asked = true
			row.Focus = true
			row.Value = e.addValueText(r, i)
			switch st.Control {
			case addChoice, addSprites, addKind:
				row.Kind = popupChoice
			case addCell:
				row.Kind = popupPick
				act.button = "pick"
			case addMany:
				row.Kind, row.Value = popupEdit, e.prompt.path
				row.Buttons, act.button = []string{e.word("editor.button.add")}, "add"
			default:
				row.Kind = popupEdit
			}
		}
		if row.Value == "" && row.Kind == popupText {
			row.Value = " "
		}
		_, chosen := r.choiceAt(e.prompt.path)
		if row.Focus && !(row.Kind == popupChoice && chosen) && body != nil {
			// A typed value keeps its end, where the caret stands.
			row.Value = textfit.Spec{Measure: body.Advance, Unit: textfit.Byte, Side: textfit.Start}.Fit(row.Value, w/2)
			row.Caret = body.Advance(row.Value)
		} else {
			row.Value = editorClip(body, row.Value, w/2)
		}
		s.Rows, acts = append(s.Rows, row), append(acts, act)
		if st.Control == addMany && i <= r.at {
			for k, v := range r.many[i] {
				label := r.labelOf(v)
				if i < r.at && k < len(r.manyLabels[i]) {
					label = r.manyLabels[i][k]
				}
				item := popupRow{Kind: popupText, Label: editorClip(body, "  "+label, w-140)}
				if i == r.at {
					item.Buttons = []string{e.word("editor.button.remove")}
				}
				s.Rows, acts = append(s.Rows, item), append(acts, addRowAct{step: i, item: k, button: "remove"})
			}
		}
	}
	if r.list != nil {
		s.List, s.ListRows = r.list, min(addListRows, r.list.Len())
		list := r.list
		s.ListText = func(row, width int) string {
			rows := list.Rows()
			if row < 0 || row >= len(rows) {
				return ""
			}
			return editorClip(body, rows[row].Text, width)
		}
		if r.steps[r.at].Control == addSprites {
			s.ListAside = previewW + popupPad
		}
	}
	for _, note := range []string{r.note, e.editorHostText(e.message)} {
		if note != "" {
			s.Notes = append(s.Notes, editorWrap(body, note, w-2*popupPad-8)...)
		}
	}
	finish := e.word("editor.button.next")
	if r.at == len(r.steps)-1 && r.steps[r.at].Control != addKind {
		finish = e.word("editor.button.finish")
	}
	s.Buttons = []string{e.word("editor.button.back"), finish, e.word("editor.button.cancel")}
	s.Disabled = []bool{r.at <= r.first(), false, false}
	l := layoutPopup(s, e.font, body)
	// The top stays put while the steps grow the window downward.
	s.At = image.Pt((e.window.X-l.Rect.Dx())/2, max(e.barHeight()+8, e.window.Y/8))
	return s, acts
}

// AddStep is the key of the step the open Add window asks and the reason its
// last value was refused; ok is false while no Add window is open. A headless
// driver reads it to give each step its value.
func (e *MapEditor) AddStep() (key, note string, ok bool) {
	r, ok := e.addRunning()
	if !ok {
		return "", "", false
	}
	return r.steps[r.at].Key, r.note, true
}

// addSpecOK is the Add window as data while it is open, else no acts.
func (e *MapEditor) addSpecOK() (popupSpec, []addRowAct) {
	if _, ok := e.addRunning(); !ok || e.addPicking() {
		return popupSpec{}, nil
	}
	return e.addSpec()
}

// addPress is a press on the Add window.
func (e *MapEditor) addPress(p image.Point, double bool) {
	r, _ := e.addRunning()
	s, acts := e.addSpec()
	l := layoutPopup(s, e.font, e.bodyFont())
	switch part, n := l.hit(p); part {
	case popupClose:
		e.leave(leavePrompt)
	case popupButton:
		switch {
		case n == 0 && !s.Disabled[0]:
			e.addBack(e.addPrevious())
		case n == 1:
			e.addNext()
		case n == 2:
			e.leave(leavePrompt)
		}
	case popupRowButton:
		a := acts[n]
		switch a.button {
		case "pick":
			e.armPick(pickAddCell)
		case "add":
			if strings.TrimSpace(e.prompt.path) == "" && r.list != nil {
				e.addTake(r.list.Selection())
			}
			e.addManyValue(e.prompt.path)
		case "remove":
			if a.step == r.at && a.item >= 0 && a.item < len(r.many[a.step]) {
				r.many[a.step] = append(r.many[a.step][:a.item:a.item], r.many[a.step][a.item+1:]...)
			}
		}
	case popupRowLine, popupRowControl:
		if a := acts[n]; !a.asked && a.item < 0 && a.step < r.at {
			e.addBack(a.step)
		}
	case popupListRow:
		top, count := r.list.Visible()
		if n >= count {
			return
		}
		e.addTake(top + n)
		if double {
			e.addNext()
		}
	case popupListBar:
		if req, _ := listBar(l.List, r.list).press(p); req != barNone {
			listBarRequest(r.list, req, 0)
		}
	}
}

// addPrevious is the step before the asked one that the window asks.
func (e *MapEditor) addPrevious() int {
	r, _ := e.addRunning()
	for i := r.at - 1; i >= 0; i-- {
		if !r.skip[i] {
			return i
		}
	}
	return -1
}

// addWheel steps the Add window's list as Up and Down do.
func (e *MapEditor) addWheel(wheel float64) {
	if r, ok := e.addRunning(); ok && r.list != nil {
		if rows := editorWheelRows(&e.listWheel, wheel); rows != 0 {
			e.addMove(rows)
		}
	}
}

// drawAddPreview draws the highlighted choice's picture beside the list of a
// step whose choices have pictures.
func (e *MapEditor) drawAddPreview(pic *image.RGBA, s popupSpec) {
	r, ok := e.addRunning()
	host, hok := e.addHost()
	if !ok || !hok || r.list == nil || r.steps[r.at].Control != addSprites {
		return
	}
	k := r.list.Selection()
	if k < 0 || k >= len(r.offer.Choices) {
		return
	}
	preview := host.NewRecordPreview(r.kind, r.steps[r.at].Key, r.offer.Choices[k].Value)
	if preview == nil || preview.Bounds().Empty() {
		return
	}
	box := layoutPopup(s, e.font, e.bodyFont()).Aside
	editorBlit(pic.SubImage(box).(*image.RGBA), preview, addPreviewRect(preview.Bounds().Size(), box), false)
}

// addPreviewRect fits a picture of size into box, centred: a large picture
// shrinks to fit, a small sprite grows by the largest whole factor that fits.
func addPreviewRect(size image.Point, box image.Rectangle) screenRect {
	scale := min(float64(box.Dx())/float64(size.X), float64(box.Dy())/float64(size.Y))
	if scale >= 1 {
		scale = float64(int(scale))
	}
	w, h := float64(size.X)*scale, float64(size.Y)*scale
	return screenRect{X: float64(box.Min.X) + (float64(box.Dx())-w)/2, Y: float64(box.Min.Y) + (float64(box.Dy())-h)/2, W: w, H: h}
}
