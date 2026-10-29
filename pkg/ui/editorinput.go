package ui

import (
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"
)

type editorInputPolicy struct {
	multiline  bool
	panArrows  bool
	choiceKeys bool
	limit      int
	bytes      func(rune) int
}

type InspectionInputRules interface {
	InputRule(scope, kind, key string) (int, func(rune) int)
}

func editorNumericBytes(r rune) int {
	if strings.ContainsRune("0123456789-+", r) {
		return 1
	}
	return 0
}

func editorCoordinateBytes(r rune) int {
	if r == ',' || r == ' ' {
		return 1
	}
	return editorNumericBytes(r)
}

func editorHexBytes(r rune) int {
	if strings.ContainsRune("abcdefABCDEFxX", r) {
		return 1
	}
	return editorNumericBytes(r)
}

type editorInputState struct {
	model  TextInputState
	target *string
	key    string
	drag   bool
	// before is the model as the last editing call found it; undo and redo
	// hold the steps this input has taken, and end with the input's focus.
	before     typingStep
	undo, redo []typingStep
}

// typingStep is the text and selection an input held between two edits.
type typingStep struct {
	text          string
	caret, anchor int
}

// typingHistoryLimit bounds the steps one input remembers.
const typingHistoryLimit = 256

func stepOf(m *TextInputState) typingStep { return typingStep{m.Text, m.Caret, m.Anchor} }

// edited records the step that just changed the text, and forgets the redo
// steps it cuts off.
func (s *editorInputState) edited() {
	if s.model.Text == s.before.text {
		return
	}
	s.undo = append(s.undo, s.before)
	if len(s.undo) > typingHistoryLimit {
		s.undo = s.undo[1:]
	}
	s.redo = nil
	s.before = stepOf(&s.model)
}

// typingHistory undoes or redoes the focused input's last typing step.
func (e *MapEditor) typingHistory(redo bool) {
	m, _ := e.inputModel()
	if m == nil {
		return
	}
	from, to := &e.input.undo, &e.input.redo
	if redo {
		from, to = to, from
	}
	if len(*from) == 0 {
		return
	}
	step := (*from)[len(*from)-1]
	*from = (*from)[:len(*from)-1]
	*to = append(*to, stepOf(m))
	m.Text, m.Caret, m.Anchor, m.SelectedAll = step.text, step.caret, step.anchor, false
	e.input.before = step
	e.inputPropagate()
}

func (e *MapEditor) inputTarget() (*string, editorInputPolicy, string) {
	p := editorInputPolicy{limit: 1 << 20, bytes: func(r rune) int {
		if unicode.IsControl(r) {
			return 0
		}
		return utf8.RuneLen(r)
	}}
	var field *InspectionField
	scope, kind, name := "", "", ""
	key := ""
	var target *string
	if e.pathDialog.pending != nil {
		return nil, p, key
	}
	if e.prompt.open() {
		switch e.prompt.kind {
		case promptOpen, promptSave, promptRectangle, promptForestShare, promptSeed, promptNewMode, promptField, promptItemRecipe, promptWorkspace, promptAdd:
		default:
			return nil, p, key
		}
		target = &e.prompt.path
		key = fmt.Sprintf("%d/%s/%d", e.prompt.kind, e.prompt.scope, e.prompt.field)
		if r, ok := e.addRunning(); ok {
			p.panArrows = e.addPicking()
			p.choiceKeys = r.steps[r.at].Control != addCell && len(r.offer.Choices) > 0
			key += fmt.Sprintf("/%p/%d", r, r.at)
			kind, name = r.kind, r.steps[r.at].Key
			switch r.steps[r.at].Control {
			case addChoice, addSprites, addNumber, addKind:
				p.limit, p.bytes = 64, editorNumericBytes
			case addCell:
				p.limit, p.bytes = 64, editorCoordinateBytes
			case addMany:
				p.bytes = editorCoordinateBytes
			}
		}
		if e.prompt.kind == promptForestShare || e.prompt.kind == promptSeed || e.prompt.kind == promptNewMode {
			p.limit, p.bytes = 64, editorNumericBytes
		}
		if e.prompt.kind == promptRectangle {
			p.limit = 64
			p.bytes = func(r rune) int {
				if r == 'x' || r == 'X' || r == ' ' {
					return 1
				}
				return editorNumericBytes(r)
			}
		}
		var fields []InspectionField
		if e.doc != nil {
			fields = e.promptFields()
		}
		if e.prompt.field < len(fields) {
			field = &fields[e.prompt.field]
		}
		p.multiline = e.multilinePrompt()
		scope = e.prompt.scope
	} else if e.objectTyping() {
		target, field = &e.obj.buf[e.obj.focus], &e.obj.fields[e.obj.focus]
		p.multiline = e.messageTextField(e.obj.focus)
		kind, _ = e.selectedRecord()
		if e.messagePage != 0 {
			scope = "pages"
		}
	} else {
		return nil, p, key
	}
	if field != nil {
		if field.Action {
			return nil, p, key
		}
		name = field.Key
		if kind == "" {
			kind, _ = e.selectedRecord()
		}
		if !field.Text {
			p.limit = 64
			p.bytes = editorNumericBytes
			if scope == "metadata" || scope == "package" && name != "source" {
				p.bytes = editorHexBytes
			}
		}
	}
	if e.doc != nil {
		if host, ok := e.doc.Edits.(InspectionInputRules); ok {
			limit, bytes := host.InputRule(scope, kind, name)
			if limit > 0 {
				p.limit = limit
			}
			if bytes != nil {
				p.bytes = bytes
			}
		}
	}
	return target, p, key
}

func (p editorInputPolicy) modifiedKey(key string, ctrl, shift bool) string {
	if key == "up" || key == "down" {
		ctrl = ctrl && p.multiline
		shift = shift && !p.choiceKeys
	}
	if (key == "home" || key == "end") && p.choiceKeys {
		ctrl = false
	}
	if shift {
		key = "shift+" + key
	}
	if ctrl {
		key = "ctrl+" + key
	}
	return key
}

func (e *MapEditor) inputModel() (*TextInputState, editorInputPolicy) {
	target, p, key := e.inputTarget()
	if target == nil {
		e.input = editorInputState{}
		return nil, p
	}
	if e.input.target != target || e.input.key != key {
		e.input = editorInputState{target: target, key: key}
		e.input.model.reset(*target)
	} else if e.input.model.Text != *target {
		repeating := e.input.model.repeating
		e.input.model.reset(*target)
		e.input.model.repeating = repeating
		e.input.undo, e.input.redo = nil, nil
	}
	e.input.before = stepOf(&e.input.model)
	e.input.model.rules = TextInputRules{Accept: func(rune) bool { return true }}
	if e.objectTyping() && len(e.obj.fields[e.obj.focus].Choices) > 0 {
		e.input.model.rules.AcceptBatch = func(s string) bool { return strings.TrimLeft(s, "-0123456789") == "" }
	}
	return &e.input.model, p
}

func (e *MapEditor) inputChanged() {
	e.input.edited()
	e.inputPropagate()
}

func (e *MapEditor) inputPropagate() {
	*e.input.target = e.input.model.Text
	if e.prompt.open() {
		e.addFollow()
	}
}

func (e *MapEditor) insertText(s string) {
	if !e.prompt.open() && !e.objectHidden && e.searchableObjectChoices() && e.dropdownOpen() && s != "" {
		e.obj.search += s
		e.obj.choice.Find(e.obj.search, e.editorSearchText)
		return
	}
	if m, _ := e.inputModel(); m != nil && s != "" {
		if m.typeText(s) {
			e.inputChanged()
		}
	}
}

func editorPaste(m *TextInputState, p editorInputPolicy, s string) string {
	lo, hi := m.selection()
	r := []rune(m.Text)
	used := 0
	for i, c := range r {
		if i < lo || i >= hi {
			n := p.bytes(c)
			if n <= 0 {
				n = utf8.RuneLen(c)
			}
			used += n
		}
	}
	var out strings.Builder
	for _, c := range editorSingleLine(s, p.multiline) {
		if unicode.IsControl(c) && !(c == '\n' && p.multiline) {
			continue
		}
		n := p.bytes(c)
		if c == '\n' && p.multiline {
			n = 1
		}
		if n <= 0 {
			continue
		}
		if used+n > p.limit {
			break
		}
		out.WriteRune(c)
		used += n
	}
	return out.String()
}

func (e *MapEditor) textKey(key string) bool {
	ctrl, shift := strings.Contains(key, "ctrl+"), strings.Contains(key, "shift+")
	key = strings.TrimPrefix(strings.TrimPrefix(strings.TrimPrefix(key, "ctrl+"), "shift+"), "ctrl+")
	if !ctrl && !shift && (key == "left" || key == "right") && e.searchableObjectChoices() && e.objectTyping() && len(e.obj.fields[e.obj.focus].Choices) > 0 {
		delta := 1
		if key == "left" {
			delta = -1
		}
		e.objectStep(delta)
		return true
	}
	switch key {
	case "left", "right", "home", "end", "up", "down", "backspace", "delete", "selectall", "copy", "cut", "paste", "textundo", "textredo":
	default:
		return false
	}
	m, policy := e.inputModel()
	if m == nil {
		return false
	}
	if key == "textundo" || key == "textredo" {
		e.typingHistory(key == "textredo")
		return true
	}
	if key == "up" || key == "down" {
		if !policy.multiline && !shift {
			return false
		}
	}
	if (key == "home" || key == "end") && policy.choiceKeys && !ctrl && !shift {
		return false
	}
	var err error
	switch key {
	case "left", "right", "home", "end", "up", "down":
		m.move(key, ctrl, shift)
	case "selectall":
		m.selectAll()
	case "backspace", "delete":
		m.erase(key == "backspace")
		e.inputChanged()
	case "copy", "cut":
		if m.Caret != m.Anchor {
			err = textClipboard.write(m.selected())
			if err == nil && key == "cut" {
				m.replace("")
				e.inputChanged()
			}
		}
	case "paste":
		var s string
		s, err = textClipboard.read()
		if err == nil {
			filtered := editorPaste(m, policy, s)
			if filtered != "" {
				m.replace(filtered)
				e.inputChanged()
			}
		}
	}
	if err != nil {
		if e.objectTyping() {
			e.obj.note = err.Error()
		} else {
			e.message = err.Error()
		}
	}
	return true
}

func (e *MapEditor) keyText(key string) error { e.textKey(key); return nil }
