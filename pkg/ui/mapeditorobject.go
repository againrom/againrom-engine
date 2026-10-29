package ui

import (
	"fmt"
	"image"
	"slices"
	"sort"
	"strconv"
	"strings"
)

// The object popup edits every field of the selected record at once. Its
// rows come from the host's field table (InspectionEdits.Fields) and its
// choices from the same table; Apply writes every changed field as one edit.

// InspectionFieldValue is one edit. Equal Step values form a batch;
// rising steps replay a draft within one history entry.
type InspectionFieldValue struct {
	Step      uint32
	Key       string
	Value     int64
	Text      bool
	TextValue string
	// New asks for a new script node of the key's kind, whose id the key
	// then takes: a trigger's condition or action slot.
	New bool
}

// InspectionFieldSet is implemented by edits that write several fields of
// one record as one history entry, through the same checks as SetField and
// SetTextField. A refused field refuses the whole set and changes nothing.
type InspectionFieldSet interface {
	SetFields(kind string, index int, values []InspectionFieldValue) (*InspectionDocument, error)
}

type InspectionObjectSet interface {
	SetObject(string, int, []InspectionFieldValue, *image.Point) (*InspectionDocument, error)
}

// objectEdit holds the popup's draft, fields, focus and open picker.
type objectEdit struct {
	pending []InspectionFieldValue
	section int
	detail  string
	move    *image.Point
	unit    unitSettingsView
	doc     *InspectionDocument
	record  int
	fields  []InspectionField
	buf     []string
	focus   int
	open    int
	choice  *Picker
	search  string
	note    string
	first   int // the spec row of the first field
	data    int // the spec row of the data heading
	cell    int // the spec row of the cell, -1 when it is not movable
	// rowField maps a spec row to the field it edits; rowActs are the
	// actions of a script popup row's buttons.
	rowField map[int]int
	rowActs  map[int][]string
	rowOpen  map[int]int
	// added marks a trigger's pairs (0..2) and action slots (3..6) shown
	// before a node is chosen for them; types are a node's slot types as
	// its chosen operation sets them; pickField is the field a cell pick
	// fills, -1 for none.
	added     [7]bool
	types     [10]uint32
	pickField int
	// signature is the operation and group command types was set for.
	signature [2]int64
	// wiz is the trigger wizard's rows, nil when the popup is no wizard.
	wiz            *wizardState
	messageParts   []InspectionMessagePart
	messageChanged bool
	// messageGone are the page numbers whose cards are struck until Apply.
	messageGone map[int]bool
}

// loadObject reads the selected record's fields when the record or the
// document changed since they were read.
func (e *MapEditor) loadObject() {
	o := &e.obj
	if o.doc == e.doc && o.record == e.selected {
		return
	}
	*o = objectEdit{doc: e.doc, record: e.selected, focus: -1, open: -1, cell: -1, pickField: -1}
	if e.modWindow {
		if host, ok := e.workspace(); ok {
			o.fields = host.WorkspaceFields("package")
			for _, f := range o.fields {
				o.buf = append(o.buf, fieldInput(f))
			}
		}
		return
	}
	if e.messagePage != 0 {
		if host, ok := e.doc.Edits.(InspectionMessages); ok {
			e.loadMessageFields(host.MessageParts(e.messagePage))
		}
		return
	}
	kind, index := e.selectedRecord()
	if index < 0 || !e.editable() {
		return
	}
	if v, ok := e.scriptPopup(); ok {
		e.loadScriptObject(v)
		return
	}
	o.fields = e.doc.Edits.Fields(kind, index)
	for _, f := range o.fields {
		o.buf = append(o.buf, fieldInput(f))
	}
}

func (e *MapEditor) objectHasFields() bool {
	e.loadObject()
	return len(e.obj.fields) > 0
}

// objectFieldRows adds the cell's pick row and a row per field to the object
// popup, and its Apply and Cancel buttons.
func (e *MapEditor) objectFieldRows(s *popupSpec, links *[]int) {
	e.loadObject()
	o := &e.obj
	r := InspectionRecord{Kind: "Mod"}
	if !e.modWindow {
		r = e.doc.Records[e.selected]
	}
	if r.Spatial && e.editable() && dragRank[r.Kind] > 0 && len(s.Rows) > 0 {
		s.Rows[len(s.Rows)-1].Kind = popupPick
		s.PickLabel = e.word("editor.object.pick")
		o.cell = len(s.Rows) - 1
	}
	if len(o.fields) == 0 {
		if r.Kind == "Scenery" || r.Kind == "Section" || !fieldKind(r.Kind) {
			s.Notes = append(s.Notes, e.word("editor.object.readonly"))
		}
		return
	}
	if !e.modWindow {
		s.Rows, *links = append(s.Rows, popupRow{Kind: popupHeading, Label: e.word("editor.object.fields")}), append(*links, 0)
	}
	o.first = len(s.Rows)
	o.rowField = map[int]int{}
	card, fields := e.unitSettings()
	native := e.nativeUnitWindow()
	if card != nil {
		s.Card = card
		s.CardFont = e.cardFont
	}
	if native {
		s.Width = max(420, e.unitWindowWidth()-32)
		if s.Card != nil {
			s.Width -= compactPanelW + popupPad
		}
		s.At = image.Pt(16, e.barHeight()+8)
		if e.slack[popObject] || e.sizes[popObject] != (image.Point{}) {
			s.At = e.objectAt
		}
		s.Tabs = []string{e.word("editor.person.identity"), e.word("editor.person.equipment"), e.word("editor.person.inventory"), e.word("editor.person.stats")}
		s.ActiveTab = o.section
		if e.unitWindowWidth() >= 1100 {
			s.Columns = 3
		}
	}
	textWidth := 378
	if native {
		textWidth = max(100, s.Width/max(1, s.Columns)-2*popupPad-20)
	}
	group := ""
	order := fieldOrder(o.fields)
	if native {
		sort.SliceStable(order, func(i, j int) bool { return o.fields[order[i]].Parent == "" && o.fields[order[j]].Parent != "" })
	}
	for _, i := range order {
		f := o.fields[i]
		if e.modWindow && f.Action && f.Column == 0 {
			continue
		}
		if e.modWindow && f.Parent != "" && f.Parent != o.detail {
			continue
		}
		if native {
			section := map[string]int{"identity": 0, "equipment": 1, "inventory": 2, "stats": 3}[f.Section]
			if s.Columns == 1 && section != o.section || s.Columns == 0 && section != o.section {
				continue
			}
			if s.Columns > 1 && section >= 2 && section != max(2, o.section) {
				continue
			}
			if f.Parent != "" && f.Parent != o.detail {
				continue
			}
		}
		start := len(s.Rows)
		deferColumn := func() {
			if !native && s.Columns > 1 {
				for k := start; k < len(s.Rows); k++ {
					s.Rows[k].Column = f.Column
				}
			}
			if native && s.Columns > 1 {
				column := map[string]int{"identity": 0, "equipment": 1, "inventory": 2, "stats": 2}[f.Section]
				for k := start; k < len(s.Rows); k++ {
					s.Rows[k].Column = column
				}
			}
		}
		if fields != nil && !slices.ContainsFunc(fields, func(current InspectionField) bool { return current.Key == f.Key }) {
			continue
		}
		if f.Group != group {
			group = f.Group
			s.Rows, *links = append(s.Rows, popupRow{Kind: popupHeading, Label: e.editorHostText(f.Group)}), append(*links, 0)
			if note := e.groupNote(f.Group); note != "" {
				for _, part := range editorWrap(e.bodyFont(), note, 378) {
					s.Rows, *links = append(s.Rows, popupRow{Kind: popupText, Label: part}), append(*links, 0)
				}
			}
		}
		if f.ReadOnly {
			for _, current := range fields {
				if current.Key == f.Key {
					f = current
					break
				}
			}
			for _, line := range f.Lines {
				kind := popupText
				if line.Role == "heading" {
					kind = popupHeading
				}
				for _, part := range editorWrap(e.bodyFont(), line.Text, textWidth) {
					s.Rows, *links = append(s.Rows, popupRow{Kind: kind, Label: part}), append(*links, 0)
				}
			}
			deferColumn()
			continue
		}
		o.rowField[len(s.Rows)] = i
		label := e.editorHostText(f.Label)
		if !native {
			label = editorClip(e.bodyFont(), label, 150)
		}
		row := popupRow{Kind: popupEdit, Label: label, Value: o.buf[i], Focus: i == o.focus}
		if len(f.Choices) > 0 {
			row.Kind, row.Value = popupChoice, choiceText(f, o.buf[i])
		} else if f.Text {
			row.Value = e.editorHostText(o.buf[i])
		}
		if f.Action {
			row.Kind = popupChoice
			row.Label = ""
			row.Value = ""
			row.Buttons = []string{e.editorHostText(f.Label)}
		}
		if e.modWindow && strings.HasSuffix(f.Key, ".character") {
			row.Buttons = []string{"..."}
		}
		if e.modWindow && f.Key == "source" {
			row.Buttons = []string{e.word("editor.mod.up"), e.word("editor.mod.down")}
		}
		meaning := f.Meaning
		if native && !f.Inherit && f.Key != o.detail && i != o.focus {
			meaning = ""
		}
		if fields == nil && o.buf[i] != fieldInput(f) {
			meaning = ""
		}
		inherited := f.Inherited
		for _, current := range fields {
			if current.Key == f.Key {
				inherited = current.Inherited
				if !native || f.Inherit || f.Key == o.detail || i == o.focus {
					meaning = current.Meaning
				}
				break
			}
		}
		if f.Inherit {
			if row.Value == "" {
				row.Hint = inherited
			} else {
				row.Ink, row.Buttons = editorGold, []string{e.word("editor.unit.clear")}
			}
		}
		if row.Focus && row.Kind == popupEdit {
			if b := e.bodyFont(); b != nil {
				row.Caret = b.Advance(row.Value)
			}
		}
		s.Rows, *links = append(s.Rows, row), append(*links, 0)
		if meaning != "" {
			for _, part := range editorWrap(e.bodyFont(), e.editorHostText(meaning), textWidth) {
				s.Rows = append(s.Rows, popupRow{Kind: popupText, Label: part})
				*links = append(*links, 0)
			}
		}
		deferColumn()
	}
	s.Buttons = []string{e.word("editor.button.apply"), e.word("editor.button.cancel")}
	s.Disabled = []bool{!e.objectDirty(), false}
	if native {
		s.Buttons = append(s.Buttons, e.word("editor.person.reset"))
		if body := e.bodyFont(); body != nil {
			for _, label := range s.Buttons {
				s.ButtonW = max(s.ButtonW, body.Advance(label)+20)
			}
		}
		count := 0
		for _, f := range o.fields {
			if f.Inherit && f.Value != f.Absent {
				count++
			}
		}
		s.Notes = append(s.Notes, fmt.Sprintf(e.word("editor.person.overrides"), count))
		s.Notes = append(s.Notes, editorWrap(e.bodyFont(), e.word("editor.person.keys"), s.Width-20)...)
	}
	if o.note != "" {
		s.Notes = append(s.Notes, editorWrap(e.bodyFont(), e.editorHostText(o.note), 400)...)
	}
}

// fieldOrder is the order the popup shows fields in: the fields of no group
// first, then each group's fields together, groups in the order they first
// appear.
func fieldOrder(fields []InspectionField) []int {
	var order []int
	var groups []string
	for i, f := range fields {
		if f.Group == "" {
			order = append(order, i)
		} else if !slices.Contains(groups, f.Group) {
			groups = append(groups, f.Group)
		}
	}
	for _, g := range groups {
		for i, f := range fields {
			if f.Group == g {
				order = append(order, i)
			}
		}
	}
	return order
}

// groupNote says under a group's heading what an empty field and a 0 mean
// for its values.
func (e *MapEditor) groupNote(group string) string {
	note := ""
	for _, f := range e.obj.fields {
		if f.Group != group || !f.Inherit {
			continue
		}
		if f.Absent == 0 {
			return e.word("editor.unit.zero.refused")
		}
		note = e.word("editor.unit.zero.allowed")
	}
	return note
}

// choiceText is the label of the choice whose value the text holds.
func choiceText(f InspectionField, text string) string {
	if v, err := strconv.ParseInt(strings.TrimSpace(text), 10, 64); err == nil {
		for _, c := range f.Choices {
			if c.Value == v {
				return c.Label
			}
		}
	}
	return text
}

func (e *MapEditor) objectDirty() bool {
	if len(e.obj.pending) > 0 {
		return true
	}
	if e.obj.messageChanged || e.messagePage != 0 && e.messageFor != nil && e.messagePage != e.messageFrom {
		return true
	}
	if e.obj.move != nil {
		return true
	}
	for i, f := range e.obj.fields {
		if f.ReadOnly {
			continue
		}
		if e.obj.buf[i] != fieldInput(f) {
			return true
		}
	}
	if v, ok := e.scriptPopup(); ok {
		return e.scriptDirty(v)
	}
	return false
}

// fieldRow is the spec row of field i, -1 when no row shows it.
func (e *MapEditor) fieldRow(i int) int {
	for r, f := range e.obj.rowField {
		if f == i {
			return r
		}
	}
	return -1
}

// nextObjectField is the field after i, in the order the rows show them,
// that a row shows.
func (e *MapEditor) nextObjectField(i int) int {
	var rows []int
	for row := range e.obj.rowField {
		rows = append(rows, row)
	}
	sort.Ints(rows)
	if len(rows) == 0 {
		return -1
	}
	for at, row := range rows {
		if e.obj.rowField[row] == i {
			return e.obj.rowField[rows[(at+1)%len(rows)]]
		}
	}
	return e.obj.rowField[rows[0]]
}

func (e *MapEditor) focusObjectField(i int) {
	e.obj.focus, e.obj.open = i, -1
	if i >= 0 && i < len(e.obj.fields) && e.nativeUnitWindow() {
		f := e.obj.fields[i]
		e.obj.detail = f.Key
		if f.Parent != "" {
			e.obj.detail = f.Parent
		}
	}
	s, _, ok := e.objectSpec()
	if !ok {
		return
	}
	row := e.fieldRow(e.obj.focus)
	if row < 0 {
		return
	}
	_, line := popupRowPosition(s, row)
	if line < s.Top {
		e.objectTop = line
	} else if s.MaxRows > 0 && line >= s.Top+s.MaxRows {
		e.objectTop = line - s.MaxRows + 1
	}
}

// objectRowPress is a press on a field row or the cell's pick button.
func (e *MapEditor) objectRowPress(part popupPart, n int) {
	o := &e.obj
	if n == o.cell && part == popupRowButton {
		e.leave(leaveDrag)
		e.armPick(pickMove)
		return
	}
	i, ok := o.rowField[n]
	if len(o.fields) == 0 || !ok || i < 0 || i >= len(o.fields) {
		return
	}
	if part == popupRowButton && o.fields[i].Inherit {
		o.buf[i], o.focus, o.note = "", -1, ""
		return
	}
	o.focus = i
	if e.modWindow && o.fields[i].Action {
		_ = e.modAction(o.fields[i].Key, "")
		return
	}
	if e.modWindow && part == popupRowButton && strings.HasSuffix(o.fields[i].Key, ".character") {
		o.detail = strings.TrimSuffix(o.fields[i].Key, "character")
		e.focusObjectField(min(i+1, len(o.fields)-1))
		return
	}
	if o.fields[i].Section != "" {
		o.detail = o.fields[i].Key
		if o.fields[i].Parent != "" {
			o.detail = o.fields[i].Parent
		}
	}
	if _, script := e.scriptPopup(); script && i == trigOnce && o.fields[i].Key == "once" {
		if strings.TrimSpace(o.buf[i]) == "0" {
			o.buf[i] = "1"
		} else {
			o.buf[i] = "0"
		}
		return
	}
	if len(o.fields[i].Choices) > 0 {
		if o.open == i {
			o.open = -1
			return
		}
		o.open = i
		o.search = ""
		f := o.fields[i]
		rows := make([]PickerRow, len(f.Choices))
		sel := 0
		for k, c := range f.Choices {
			rows[k] = PickerRow{Text: c.Label, Choosable: true, Icon: f.Icons[c.Value]}
			if slices.Contains(f.Shut, c.Value) {
				rows[k].Choosable, rows[k].Ink = false, popupHintInk
			}
			if strconv.FormatInt(c.Value, 10) == strings.TrimSpace(o.buf[i]) {
				sel = k
			}
		}
		o.choice = NewPicker(rows).SetWindow(min(10, len(rows)))
		o.choice.Select(sel)
	}
}

// openDropdown is the object popup's open drop-down, placed on its row.
func (e *MapEditor) openDropdown() (dropdown, bool) {
	o := &e.obj
	if o.open < 0 || o.choice == nil {
		return dropdown{}, false
	}
	s, _, ok := e.objectSpec()
	if !ok {
		return dropdown{}, false
	}
	for _, rr := range layoutPopup(s, e.font, e.bodyFont()).Rows {
		if rr.Index == e.fieldRow(o.open) {
			d := dropdown{Rect: rr.Control, Text: s.Rows[rr.Index].Value, Open: true, List: o.choice, Rows: min(10, o.choice.Len()), Focus: true, Fill: editorPanel, Ink: editorLight}
			if o.search != "" {
				d.Text = e.editorHostText(o.search)
			}
			// A list with a choice it shuts is as wide as its longest row, so
			// the reason reads whole.
			if body := e.bodyFont(); body != nil && len(o.fields[o.open].Shut) > 0 {
				for _, row := range o.choice.Rows() {
					d.ListWidth = max(d.ListWidth, body.Advance(row.Text)+2*listTextX+widgetSpriteSize+8)
				}
				d.ListWidth = min(d.ListWidth, max(rr.Control.Dx(), e.window.X-rr.Control.Min.X-8))
			}
			return d, true
		}
	}
	return dropdown{}, false
}

// drawOpenDropdown paints the object popup's open drop-down over everything
// but an open menu.
func (e *MapEditor) drawOpenDropdown(pic *image.RGBA) {
	d, ok := e.openDropdown()
	if !ok {
		return
	}
	rows := d.List.Rows()
	drawDropdown(pic, e.bodyFont(), nil, d, func(row, width int) string { return editorClip(e.bodyFont(), rows[row].Text, width) }, e.pointer, e.pointerOK)
}

func (e *MapEditor) openDropdownHit(p image.Point) bool {
	d, ok := e.openDropdown()
	return ok && p.In(d.Bounds(e.bodyFont()))
}

// openDropdownPress is a press while a drop-down is open: a row chooses, the
// box closes it, the bar scrolls it, and a press elsewhere closes it and goes
// on to what lies under it.
func (e *MapEditor) openDropdownPress(p image.Point) bool {
	d, ok := e.openDropdown()
	if !ok {
		return false
	}
	o := &e.obj
	switch part, row := d.hit(e.bodyFont(), p); part {
	case dropdownRow:
		e.chooseObjectChoice(row)
		return true
	case dropdownToggle:
		o.open = -1
		return true
	case dropdownBar:
		return true
	}
	o.open = -1
	return false
}

func (e *MapEditor) objectWheelDropdown(wheel float64) bool {
	if !e.openDropdownHit(e.pointer) {
		return false
	}
	e.obj.choice.MoveChoice(editorWheelRows(&e.listWheel, wheel))
	return true
}

// chooseObjectChoice sets the open drop-down's field to choice row and
// closes it.
func (e *MapEditor) chooseObjectChoice(row int) {
	o := &e.obj
	if o.open < 0 || o.open >= len(o.fields) || row < 0 || row >= len(o.fields[o.open].Choices) {
		return
	}
	if slices.Contains(o.fields[o.open].Shut, o.fields[o.open].Choices[row].Value) {
		return
	}
	if f := e.dialogChoiceField(); f >= 0 && o.open == f {
		e.chooseDialog(o.fields[f].Choices[row].Value)
		return
	}
	if o.fields[o.open].Choices[row].Value == NewMessageChoice && e.messageField(o.open) {
		field := o.open
		o.open = -1
		e.newMessage(field)
		return
	}
	field, value := o.open, o.fields[o.open].Choices[row].Value
	commit := e.messageChoiceAlone(field, value)
	o.buf[field] = strconv.FormatInt(value, 10)
	o.choice.Select(row)
	o.open = -1
	e.retypeNode()
	if commit {
		e.applyObject()
	}
	if e.modWindow {
		_ = e.applyModField()
	}
}

// chooseObjectChoiceKeep sets the open drop-down's field to its highlighted
// choice and leaves it open: the arrow keys over an open list.
func (e *MapEditor) chooseObjectChoiceKeep() {
	o := &e.obj
	if o.open >= 0 && o.open < len(o.fields) && o.choice != nil {
		if k := o.choice.Selection(); k >= 0 && k < len(o.fields[o.open].Choices) {
			if !slices.Contains(o.fields[o.open].Shut, o.fields[o.open].Choices[k].Value) {
				o.buf[o.open] = strconv.FormatInt(o.fields[o.open].Choices[k].Value, 10)
			}
		}
	}
	e.retypeNode()
}

// objectStep moves the focused choice field by delta, clamping: the
// drop-down's keyboard path.
func (e *MapEditor) objectStep(delta int) bool {
	o := &e.obj
	if o.focus < 0 || o.focus >= len(o.fields) || len(o.fields[o.focus].Choices) == 0 {
		return false
	}
	f := o.fields[o.focus]
	rows := make([]PickerRow, len(f.Choices))
	at := -1
	for k, c := range f.Choices {
		rows[k] = PickerRow{Text: c.Label, Choosable: !slices.Contains(f.Shut, c.Value)}
		if strconv.FormatInt(c.Value, 10) == strings.TrimSpace(o.buf[o.focus]) {
			at = k
		}
	}
	m := o.choice
	if o.open != o.focus || m == nil {
		m = NewPicker(rows)
	}
	if at < 0 {
		m.Select(0)
	} else {
		m.Select(at)
		m.MoveChoice(delta)
	}
	o.buf[o.focus] = strconv.FormatInt(f.Choices[m.Selection()].Value, 10)
	e.retypeNode()
	return true
}

// dropdownOpen reports that a drop-down of the shown object popup is open.
func (e *MapEditor) dropdownOpen() bool {
	return e.obj.open >= 0 && e.obj.doc == e.doc && e.obj.record == e.selected
}

func (e *MapEditor) searchableObjectChoices() bool { return e.modWindow || e.nativeUnitWindow() }

// objectTyping reports that the keyboard edits an object popup field.
func (e *MapEditor) objectTyping() bool {
	return !e.prompt.open() && e.obj.focus >= 0 && e.obj.focus < len(e.obj.fields) && e.obj.doc == e.doc && e.obj.record == e.selected
}

// endFieldTyping ends field typing as Esc does, without applying the typed
// value. A click on the bar or a menu row calls it before acting.
func (e *MapEditor) endFieldTyping() {
	if e.objectTyping() {
		e.obj.focus = -1
	}
}

// objectButton is Apply (0) or Cancel (1).
func (e *MapEditor) objectButton(n int) {
	if e.modWindow {
		e.modButton(n)
		return
	}
	if n == 2 && e.nativeUnitWindow() {
		for i, f := range e.obj.fields {
			if f.Inherit {
				e.obj.buf[i] = ""
			}
		}
		return
	}
	if n == 1 {
		e.cancelPopup()
		return
	}
	e.applyObject()
}

// applyObject writes every changed field as one edit. A value that does not
// parse, or a field the host refuses, shows its reason in the popup and
// changes nothing.
func (e *MapEditor) applyObject() {
	if e.modWindow {
		_ = e.applyModField()
		return
	}
	if e.messagePage != 0 {
		e.applyMessage()
		return
	}
	for i := range e.obj.fields {
		if e.obj.buf[i] == strconv.FormatInt(NewMessageChoice, 10) && e.messageField(i) {
			e.newMessage(i)
			return
		}
	}
	o := &e.obj
	kind, index := e.selectedRecord()
	if index < 0 || len(o.fields) == 0 {
		return
	}
	if v, ok := e.scriptPopup(); ok && v.trigger != nil && o.wiz != nil {
		e.applyWizard(v)
		return
	}
	if v, ok := e.scriptPopup(); ok {
		values, err := e.scriptValues(v)
		if err == nil && len(values) == 0 {
			return
		}
		var next *InspectionDocument
		if err == nil {
			next, err = e.setFields(kind, index, values)
		}
		if err != nil {
			o.note = err.Error()
			return
		}
		o.open, o.note = -1, ""
		_ = e.adoptEdit(next, nil)
		return
	}
	var values []InspectionFieldValue
	values = append(values, o.pending...)
	for i, f := range o.fields {
		if f.ReadOnly {
			continue
		}
		if o.buf[i] == fieldInput(f) {
			continue
		}
		v := InspectionFieldValue{Key: f.Key, Text: f.Text, TextValue: o.buf[i]}
		if e.nativeUnitWindow() {
			v.Step = uint32(len(o.pending) + 1)
		}
		if !f.Text {
			n, err := fieldNumber(f, o.buf[i])
			if err != nil {
				o.note = e.fieldNumberNote(f, err)
				return
			}
			v.Value = n
		}
		values = append(values, v)
	}
	if len(values) == 0 && o.move == nil {
		return
	}
	var next *InspectionDocument
	var err error
	if host, ok := e.doc.Edits.(InspectionObjectSet); ok {
		next, err = host.SetObject(kind, index, values, o.move)
	} else if o.move != nil {
		next, err = e.doc.Edits.Move(kind, index, *o.move)
	} else {
		next, err = e.setFields(kind, index, values)
	}
	if err != nil {
		o.note = err.Error()
		return
	}
	o.open, o.note = -1, ""
	o.move = nil
	section, detail, native := o.section, o.detail, e.nativeUnitWindow()
	_ = e.adoptEdit(next, nil)
	if native {
		e.obj.doc = nil
		e.loadObject()
		e.obj.section, e.obj.detail = section, detail
	}
}

func (e *MapEditor) ObjectFields() []InspectionField {
	e.objectSpec()
	return slices.Clone(e.obj.fields)
}

// setFields writes values through the host's one-edit seam, or field by
// field on a host without it.
func (e *MapEditor) setFields(kind string, index int, values []InspectionFieldValue) (*InspectionDocument, error) {
	if host, ok := e.doc.Edits.(InspectionFieldSet); ok {
		return host.SetFields(kind, index, values)
	}
	var next *InspectionDocument
	for _, v := range values {
		var d *InspectionDocument
		var err error
		if v.Text {
			host, ok := e.doc.Edits.(InspectionTextFields)
			if !ok {
				return nil, fmt.Errorf("This host cannot edit text fields.")
			}
			d, err = host.SetTextField(kind, index, v.Key, v.TextValue)
		} else {
			d, err = e.doc.Edits.SetField(kind, index, v.Key, v.Value)
		}
		if err != nil {
			return next, err
		}
		if d != nil {
			next = d
		}
	}
	return next, nil
}

// closeObject closes the object popup; the selection stays.
func (e *MapEditor) closeObject() {
	if e.modWindow {
		e.obj.doc = nil
	}
	e.modWindow = false
	e.modConfirm = ""
	e.messagePage = 0
	e.objectHidden = true
	e.obj.open, e.obj.focus = -1, -1
	e.groupSel = editorGroup{}
	e.objectBack = nil
	e.selection.hidden = true
}

// followReference frames reference n and, when it names one record's cell,
// selects that record and opens its popup.
func (e *MapEditor) followReference(n int) {
	if !e.FocusReference(n) {
		return
	}
	targets := e.doc.Records[e.selected].References[n-1].Targets
	if len(targets) != 1 {
		return
	}
	for i, r := range e.doc.Records {
		if i != e.selected && r.Spatial && r.Cell == targets[0].Cell && dragRank[r.Kind] > 0 {
			e.openObjectRecord(i, false)
			return
		}
	}
}

// editorGroupsFilter lists the units' groups.
const editorGroupsFilter = 10

// editorGroup is one group of units: a player's group number and its
// members' records.
type editorGroup struct {
	player, group int
	members       []int
	set           bool
}

// groups are the map's unit groups, by player then group number. A unit
// with group 0 is in no group.
func (e *MapEditor) groups() []editorGroup {
	if e.doc == nil {
		return nil
	}
	at := map[[2]int]int{}
	var out []editorGroup
	for i, r := range e.doc.Records {
		if r.Kind != "Unit" || r.Group == 0 {
			continue
		}
		k := [2]int{r.Player, r.Group}
		n, ok := at[k]
		if !ok {
			n = len(out)
			at[k] = n
			out = append(out, editorGroup{player: r.Player, group: r.Group, set: true})
		}
		out[n].members = append(out[n].members, i)
	}
	sort.SliceStable(out, func(a, b int) bool {
		if out[a].player != out[b].player {
			return out[a].player < out[b].player
		}
		return out[a].group < out[b].group
	})
	return out
}

func (e *MapEditor) groupLabel(g editorGroup) string {
	return fmt.Sprintf("%s %d  P%d  %d %s", e.word("editor.group.group"), g.group, g.player, len(g.members), e.word("editor.group.units"))
}

// chooseGroup selects a group's members, highlights them and opens the
// group popup; with frame set the view centres on the members' middle.
func (e *MapEditor) chooseGroup(g editorGroup, frame bool) {
	e.messagePage = 0
	e.selected, e.focusedReference = -1, 0
	e.objectBack = nil
	e.groupSel, e.objectHidden, e.objectTop = g, false, 0
	e.selection.refs, e.selection.marquee, e.selection.hidden, e.selection.box = nil, nil, false, image.Rectangle{}
	var box image.Rectangle
	for k, id := range g.members {
		r := e.doc.Records[id]
		e.selection.refs = append(e.selection.refs, RecordRef{r.Kind, r.Index})
		if cell := (image.Rectangle{Min: r.Cell, Max: r.Cell.Add(image.Pt(1, 1))}); k == 0 {
			box = cell
		} else {
			box = box.Union(cell)
		}
	}
	if frame && len(g.members) > 0 {
		e.centerOn(image.Pt((box.Min.X+box.Max.X-1)/2, (box.Min.Y+box.Max.Y-1)/2))
	}
}

// groupSpec is the chosen group's popup: a row per member that opens it.
func (e *MapEditor) groupSpec() (popupSpec, []int, bool) {
	g := e.groupSel
	if !g.set || e.doc == nil || e.catalog || e.objectHidden || e.pick != pickNone || e.prompt.kind == promptAdd || e.selected >= 0 {
		return popupSpec{}, nil, false
	}
	e.loadObject()
	if !e.objPlaced {
		e.objectAt, e.objPlaced = image.Pt(e.window.X-8-420, e.barHeight()+8), true
	}
	s := popupSpec{Title: e.groupLabel(g), At: e.objectAt, Width: e.popupWide(popObject, 420), Bounds: e.popupBounds()}
	links := []int{0, 0}
	s.Rows = []popupRow{{Kind: popupText, Label: e.word("editor.group.player"), Value: strconv.Itoa(g.player)}, {Kind: popupHeading, Label: e.word("editor.group.members")}}
	for _, id := range g.members {
		if id < len(e.doc.Records) {
			s.Rows = append(s.Rows, popupRow{Kind: popupLink, Label: editorClip(e.bodyFont(), e.recordSummary(id), s.Width-2*popupPad-8)})
			links = append(links, -(id + 1))
		}
	}
	e.scriptLinkSection(&s, &links, e.word("editor.usedby.title"), e.usedByRows())
	s.MaxRows = e.objectRowsShown(len(e.objectBack) > 0)
	e.objectTop = max(0, min(e.objectTop, len(s.Rows)-s.MaxRows))
	s.Top = e.objectTop
	return s, links, true
}

// groupMarkerLines mark the chosen group's members.
func (e *MapEditor) groupMarkerLines() []editorLine {
	if !e.groupSel.set || e.selected >= 0 {
		return nil
	}
	v := e.doc.Viewer
	var out []editorLine
	for _, id := range e.groupSel.members {
		if id >= len(e.doc.Records) {
			continue
		}
		wx, wy := e.cellCenter(e.doc.Records[id].Cell)
		x, y := v.cam.WorldToScreen(wx, wy)
		for _, d := range [][4]float32{{-7, -7, 7, -7}, {7, -7, 7, 7}, {7, 7, -7, 7}, {-7, 7, -7, -7}} {
			out = append(out, editorLine{float32(x) + d[0], float32(y) + d[1], float32(x) + d[2], float32(y) + d[3], editorCondition})
		}
	}
	return out
}

// GroupMembers are the records the chosen group highlights.
func (e *MapEditor) GroupMembers() []int {
	if !e.groupSel.set {
		return nil
	}
	return append([]int(nil), e.groupSel.members...)
}
