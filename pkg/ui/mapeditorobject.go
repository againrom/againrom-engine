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

// InspectionFieldValue is one field's new value for SetFields.
type InspectionFieldValue struct {
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

// objectEdit is the object popup's edit state: the fields read for one record
// of one document, each field's typed text, the focused field and an open
// drop-down.
type objectEdit struct {
	move   *image.Point
	unit   unitSettingsView
	doc    *InspectionDocument
	record int
	fields []InspectionField
	buf    []string
	focus  int
	open   int
	choice *Picker
	note   string
	first  int // the spec row of the first field
	data   int // the spec row of the data heading
	cell   int // the spec row of the cell, -1 when it is not movable
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
}

// loadObject reads the selected record's fields when the record or the
// document changed since they were read.
func (e *MapEditor) loadObject() {
	o := &e.obj
	if o.doc == e.doc && o.record == e.selected {
		return
	}
	*o = objectEdit{doc: e.doc, record: e.selected, focus: -1, open: -1, cell: -1, pickField: -1}
	if e.messagePage != 0 {
		if host, ok := e.doc.Edits.(InspectionMessages); ok {
			o.fields = host.MessageFields(e.messagePage)
			for _, f := range o.fields {
				o.buf = append(o.buf, fieldInput(f))
			}
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
	r := e.doc.Records[e.selected]
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
	s.Rows, *links = append(s.Rows, popupRow{Kind: popupHeading, Label: e.word("editor.object.fields")}), append(*links, 0)
	o.first = len(s.Rows)
	o.rowField = map[int]int{}
	card, fields := e.unitSettings()
	if card != nil {
		s.Card = card
		s.CardFont = e.cardFont
	}
	group := ""
	for _, i := range fieldOrder(o.fields) {
		f := o.fields[i]
		if f.Group != group {
			group = f.Group
			s.Rows, *links = append(s.Rows, popupRow{Kind: popupHeading, Label: e.editorHostText(f.Group)}), append(*links, 0)
			if note := e.groupNote(f.Group); note != "" {
				for _, part := range editorWrap(e.bodyFont(), note, 378) {
					s.Rows, *links = append(s.Rows, popupRow{Kind: popupText, Label: part}), append(*links, 0)
				}
			}
		}
		o.rowField[len(s.Rows)] = i
		row := popupRow{Kind: popupEdit, Label: editorClip(e.bodyFont(), e.editorHostText(f.Label), 150), Value: o.buf[i], Focus: i == o.focus}
		if len(f.Choices) > 0 {
			row.Kind, row.Value = popupChoice, choiceText(f, o.buf[i])
		} else if f.Text {
			row.Value = e.editorHostText(o.buf[i])
		}
		meaning := f.Meaning
		if fields == nil && o.buf[i] != fieldInput(f) {
			meaning = ""
		}
		inherited := f.Inherited
		for _, current := range fields {
			if current.Key == f.Key {
				meaning, inherited = current.Meaning, current.Inherited
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
			for _, part := range editorWrap(e.bodyFont(), e.editorHostText(meaning), 378) {
				s.Rows = append(s.Rows, popupRow{Kind: popupText, Label: part})
				*links = append(*links, 0)
			}
		}
	}
	s.Buttons = []string{e.word("editor.button.apply"), e.word("editor.button.cancel")}
	s.Disabled = []bool{!e.objectDirty(), false}
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
	if e.obj.move != nil {
		return true
	}
	for i, f := range e.obj.fields {
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
	order := fieldOrder(e.obj.fields)
	n := len(order)
	at := slices.Index(order, i)
	for k := 1; k <= n; k++ {
		if j := order[(at+k+n)%n]; e.fieldRow(j) >= 0 {
			return j
		}
	}
	return order[(at+1+n)%n]
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
		f := o.fields[i]
		rows := make([]PickerRow, len(f.Choices))
		sel := 0
		for k, c := range f.Choices {
			rows[k] = PickerRow{Text: c.Label, Choosable: true}
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
			return dropdown{Rect: rr.Control, Text: s.Rows[rr.Index].Value, Open: true, List: o.choice, Rows: min(10, o.choice.Len()), Focus: true, Fill: editorPanel, Ink: editorLight}, true
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
	e.obj.choice.Move(editorWheelRows(&e.listWheel, wheel))
	return true
}

// chooseObjectChoice sets the open drop-down's field to choice row and
// closes it.
func (e *MapEditor) chooseObjectChoice(row int) {
	o := &e.obj
	if o.open < 0 || o.open >= len(o.fields) || row < 0 || row >= len(o.fields[o.open].Choices) {
		return
	}
	if o.fields[o.open].Choices[row].Value == NewMessageChoice && e.messageField(o.open) {
		field := o.open
		o.open = -1
		e.newMessage(field)
		return
	}
	o.buf[o.open] = strconv.FormatInt(o.fields[o.open].Choices[row].Value, 10)
	o.choice.Select(row)
	o.open = -1
	e.retypeNode()
}

// chooseObjectChoiceKeep sets the open drop-down's field to its highlighted
// choice and leaves it open: the arrow keys over an open list.
func (e *MapEditor) chooseObjectChoiceKeep() {
	o := &e.obj
	if o.open >= 0 && o.open < len(o.fields) && o.choice != nil {
		if k := o.choice.Selection(); k >= 0 && k < len(o.fields[o.open].Choices) {
			o.buf[o.open] = strconv.FormatInt(o.fields[o.open].Choices[k].Value, 10)
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
		rows[k] = PickerRow{Text: c.Label, Choosable: true}
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
		editorListStep(m, delta)
	}
	o.buf[o.focus] = strconv.FormatInt(f.Choices[m.Selection()].Value, 10)
	e.retypeNode()
	return true
}

// dropdownOpen reports that a drop-down of the shown object popup is open.
func (e *MapEditor) dropdownOpen() bool {
	return e.obj.open >= 0 && e.obj.doc == e.doc && e.obj.record == e.selected
}

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

// objectText edits the focused field's text: s is typed, back removes a rune.
func (e *MapEditor) objectText(s string, back bool) {
	o := &e.obj
	if len(o.fields[o.focus].Choices) > 0 && !back {
		if strings.TrimLeft(s, "-0123456789") != "" {
			return
		}
	}
	if back {
		r := []rune(o.buf[o.focus])
		if len(r) > 0 {
			o.buf[o.focus] = string(r[:len(r)-1])
		}
		return
	}
	o.buf[o.focus] += s
}

// objectButton is Apply (0) or Cancel (1).
func (e *MapEditor) objectButton(n int) {
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
	for i, f := range o.fields {
		if o.buf[i] == fieldInput(f) {
			continue
		}
		v := InspectionFieldValue{Key: f.Key, Text: f.Text, TextValue: o.buf[i]}
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
	_ = e.adoptEdit(next, nil)
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
	e.messagePage = 0
	e.objectHidden = true
	e.obj.open, e.obj.focus = -1, -1
	e.groupSel = editorGroup{}
	e.objectBack = nil
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

// chooseGroup highlights a group's members, frames them and opens the group
// popup.
func (e *MapEditor) chooseGroup(g editorGroup) {
	e.messagePage = 0
	e.selected, e.focusedReference = -1, 0
	e.objectBack = nil
	e.groupSel, e.objectHidden, e.objectTop = g, false, 0
	if len(g.members) > 0 {
		e.centerOn(e.doc.Records[g.members[0]].Cell)
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
