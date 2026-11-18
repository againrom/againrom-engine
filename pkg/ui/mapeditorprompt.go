package ui

import (
	"image"
	"slices"
	"strconv"
	"strings"
)

// promptWords are the word ids of a typed prompt's title, the name of what
// it asks for, and of its title help, what to type; promptHelp puts the
// keys after the help. A field prompt not listed takes both from its field,
// and the seed prompt its title from the command (seedTitle).
var promptWords = map[promptKind]struct{ title, help string }{
	promptOpen:        {"editor.prompt.open", "editor.prompt.open.help"},
	promptSave:        {"editor.prompt.save", "editor.prompt.save.help"},
	promptItemRecipe:  {"editor.prompt.recipe", "editor.prompt.recipe.help"},
	promptRectangle:   {"editor.prompt.newmap", "editor.prompt.newmap.help"},
	promptSpeaker:     {"editor.speaker.add.name", "editor.speaker.add.name.help"},
	promptForestShare: {"editor.prompt.share", "editor.prompt.share.help"},
	promptSeed:        {"", "editor.prompt.seed.help"},
	promptMapsFolder:  {"editor.mod.addmaps.title", ""},
}

// promptTitle is a typed prompt's title: the name of what it asks for.
func (e *MapEditor) promptTitle() string {
	if r, ok := e.addRunning(); ok {
		return r.title
	}
	if e.prompt.kind == promptSeed {
		return e.seedTitle()
	}
	if w, ok := promptWords[e.prompt.kind]; ok {
		return e.word(w.title)
	}
	if e.prompt.fields() {
		return e.fieldTitle()
	}
	return e.word("editor.prompt.open")
}

// promptHelp is a typed prompt's title help: what to type, the field's
// meaning for a field prompt, then a multiline field's line keys and last
// the prompt's keys, stated once here and in no title or note.
func (e *MapEditor) promptHelp() string {
	var parts []string
	keys := e.word("editor.prompt.keys")
	if w := promptWords[e.prompt.kind]; w.help != "" {
		parts = append(parts, e.word(w.help))
	} else if fields := e.promptFields(); e.prompt.fields() && e.prompt.field < len(fields) {
		if m := fields[e.prompt.field].Meaning; m != "" {
			parts = append(parts, e.editorHostText(m))
		}
	}
	if e.prompt.fields() && e.prompt.kind != promptItemRecipe {
		keys = e.word("editor.prompt.keys.field")
	}
	if e.multilinePrompt() {
		parts = append(parts, e.word("editor.prompt.multiline"))
	}
	return strings.Join(append(parts, keys), " ")
}

// promptRows are a typed prompt's rows, its text wrapped to text pixels: a
// field with choices shows its choice row, any other prompt the typed text's
// edit rows. The save window adds its switch row; a prompt of two rows
// labels each, a prompt of one row has no label.
func (e *MapEditor) promptRows(s *popupSpec, text int) {
	if f, ok := e.promptChoiceField(); ok {
		e.promptChoiceRows(s, f, text)
		return
	}
	switched, ok := e.textsInsideRow()
	if !ok {
		s.Rows = e.promptInputRows(text)
		return
	}
	path := e.word("editor.prompt.path")
	s.LabelWidth = 160
	if body := e.bodyFont(); body != nil {
		pitch := fontHeight(body, 15) + popupRowExtra
		s.LabelWidth = max(body.Advance(path), body.Advance(switched.Label)+helpMarkW(switched, pitch)) + 12
	}
	rows := e.promptInputRows(text - s.LabelWidth)
	if len(rows) > 0 {
		rows[0].Label = path
	}
	s.Rows = append(rows, switched)
}

// promptChoiceField is the field a field prompt sets when that field has
// choices.
func (e *MapEditor) promptChoiceField() (InspectionField, bool) {
	if !e.prompt.fields() || e.prompt.kind == promptAdd {
		return InspectionField{}, false
	}
	fields := e.promptFields()
	if e.prompt.field >= len(fields) || len(fields[e.prompt.field].Choices) == 0 {
		return InspectionField{}, false
	}
	return fields[e.prompt.field], true
}

// picturedChoices reports that every choice of f carries its picture and
// none stands in a group: the prompt shows them as a picture grid. A
// grouped catalogue opens as the drop-down, whose rows carry the pictures
// of the group shown.
func picturedChoices(f InspectionField) bool {
	if len(f.Groups) > 0 || len(f.Icons) == 0 {
		return false
	}
	for _, c := range f.Choices {
		if f.Icons[c.Value] == nil {
			return false
		}
	}
	return true
}

// promptChoiceRows are the choice row of field f: a drop-down showing the
// chosen choice, or for pictured choices a row with the chosen choice's
// name and picture over the grid of every choice, its whole name in the
// cell's tip. Typing the value and Up and Down still choose.
func (e *MapEditor) promptChoiceRows(s *popupSpec, f InspectionField, text int) {
	value := choiceText(f, e.prompt.path)
	if !picturedChoices(f) {
		s.Rows = []popupRow{{Kind: popupChoice, Value: editorClip(e.bodyFont(), value, text), Focus: true}}
		return
	}
	row := popupRow{Kind: popupText, Value: editorClip(e.bodyFont(), value, text)}
	chosen, err := strconv.ParseInt(strings.TrimSpace(e.prompt.path), 10, 64)
	g := &popupGrid{Side: gridCellSide, Top: e.prompt.gridTop, LeastRows: 1}
	for _, c := range f.Choices {
		icon := f.Icons[c.Value]
		g.Side = max(g.Side, icon.Bounds().Dx()+10, icon.Bounds().Dy()+10)
		on := err == nil && c.Value == chosen
		if on {
			row.Icons = []*image.RGBA{icon}
		}
		g.Cells = append(g.Cells, popupGridCell{Icon: icon, Tip: c.Label, Pressed: on})
	}
	s.Rows, s.Grid = []popupRow{row}, g
}

// promptDropdown is the prompt's one drop-down on its choice row: the area
// prompt's trigger choice, or a field prompt's choices while their list is
// open.
func (e *MapEditor) promptDropdown() (dropdown, bool) {
	var list *Picker
	open := false
	switch {
	case e.prompt.kind == promptAreaTrigger && e.prompt.area != nil:
		list, open = e.prompt.area.choice, e.prompt.area.open
	case e.prompt.list != nil && e.prompt.fields():
		list, open = e.prompt.list, true
	default:
		return dropdown{}, false
	}
	base, ok := e.promptSpecBase()
	if !ok {
		return dropdown{}, false
	}
	s := e.sizedPrompt(base)
	for _, rr := range layoutPopup(s, e.font, e.bodyFont()).Rows {
		if s.Rows[rr.Index].Kind == popupChoice {
			return e.rowDropdown(rr.Control, s.Rows[rr.Index].Value, list, min(10, list.Len()), open), true
		}
	}
	return dropdown{}, false
}

// promptDropdownHit reports that p lies on the prompt's open list.
func (e *MapEditor) promptDropdownHit(p image.Point) bool {
	d, ok := e.promptDropdown()
	return ok && d.Open && p.In(d.Bounds(e.bodyFont()))
}

// openPromptList opens or closes the list of a field prompt's choice row.
func (e *MapEditor) openPromptList() {
	f, ok := e.promptChoiceField()
	if !ok || e.prompt.list != nil {
		e.prompt.list = nil
		return
	}
	choices, list := e.openChoiceList(f, e.prompt.path, func(n int) int { return min(10, n) })
	e.prompt.choices, e.prompt.list = &choices, list
}

// promptListPress is a press while a field prompt's list is open: a row
// chooses, the box closes the list, the bar scrolls it, and a press
// elsewhere closes it and goes on to what lies under it.
func (e *MapEditor) promptListPress(p image.Point) bool {
	if e.prompt.list == nil {
		return false
	}
	d, ok := e.promptDropdown()
	if !ok {
		e.prompt.list = nil
		return false
	}
	switch part, row := d.hit(e.bodyFont(), p); part {
	case dropdownRow:
		e.choosePromptRow(row)
		return true
	case dropdownToggle:
		e.prompt.list = nil
		return true
	case dropdownBar:
		return true
	}
	e.prompt.list = nil
	return false
}

// choosePromptRow takes row of the field prompt's open list: a group opens
// its choices, a choice becomes the typed value and closes the list.
func (e *MapEditor) choosePromptRow(row int) {
	index, navigate := e.prompt.choices.take(row)
	if navigate {
		rows := e.prompt.choices.rows()
		e.prompt.list = NewPicker(rows).SetWindow(min(10, len(rows)))
		return
	}
	f, ok := e.promptChoiceField()
	if !ok || index < 0 || index >= len(f.Choices) || slices.Contains(f.Shut, f.Choices[index].Value) {
		return
	}
	e.choosePromptChoice(index)
}

// choosePromptChoice makes choice n of the field prompt's field its typed
// value: a row of its list or a cell of its picture grid.
func (e *MapEditor) choosePromptChoice(n int) {
	if f, ok := e.promptChoiceField(); ok && n >= 0 && n < len(f.Choices) {
		e.prompt.path, e.prompt.list = strconv.FormatInt(f.Choices[n].Value, 10), nil
	}
}

// keptError shows err as the message of an error that left the open map and
// its edit history as they were; the window says so under it.
func (e *MapEditor) keptError(err error) {
	e.message, e.messageKept = err.Error(), ""
	if e.doc != nil {
		e.messageKept = e.message
	}
}
