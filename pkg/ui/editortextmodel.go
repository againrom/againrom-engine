package ui

import (
	"strings"
	"time"
	"unicode"
)

type TextSelection uint8

const (
	TextCaretSelection TextSelection = iota
	TextReplaceAllSelection
	TextStickyAllSelection
	TextDetachedSelection
)

type TextInputRules struct {
	ByteUnits, AppendOnly, SplitLineBreaks, LiteralArrows bool
	Limit, ByteLimit                                      int
	Selection                                             TextSelection
	Accept                                                func(rune) bool
	AcceptBatch                                           func(string) bool
	Encode                                                func(rune) (byte, bool)
}

type TextInputKeys struct {
	Typed                                     string
	Backspace, Delete, Home, End, Left, Right bool
	SelectAll, Shift, Ctrl, Clamp             bool
	MoveTo                                    *int
}

// EditText applies a field's accepted keys through the shared text model.
func EditText(state TextInputState, keys TextInputKeys, rules TextInputRules) TextInputState {
	m := state
	m.rules = rules
	m.TypedBelowLimit, m.Handled = false, false
	n := len(m.units())
	if rules.Selection != TextDetachedSelection {
		m.clamp()
	}
	if rules.Selection == TextCaretSelection {
		m.Anchor = max(0, min(m.Anchor, n))
	} else {
		m.Anchor = m.Caret
	}
	if rules.AppendOnly {
		m.Caret, m.Anchor, m.SelectedAll = n, n, false
	}
	m.edit(keys)
	return m
}

func (m *TextInputState) units() []rune {
	if m.rules.ByteUnits {
		r := make([]rune, len(m.Text))
		for i, b := range []byte(m.Text) {
			r[i] = rune(b)
		}
		return r
	}
	return []rune(m.Text)
}

func (m *TextInputState) join(r []rune) string {
	if m.rules.ByteUnits {
		b := make([]byte, len(r))
		for i, c := range r {
			b[i] = byte(c)
		}
		return string(b)
	}
	return string(r)
}

func (m *TextInputState) edit(k TextInputKeys) {
	if k.Clamp {
		m.clamp()
	}
	if !m.rules.AppendOnly {
		if k.SelectAll {
			m.selectAll()
			return
		}
		key := ""
		switch {
		case k.MoveTo != nil:
			m.moveTo(*k.MoveTo, k.Shift)
		case k.Home:
			key = "home"
		case k.End:
			key = "end"
		case k.Left && m.Caret > 0:
			key = "left"
		case k.Right && m.Caret < len(m.units()):
			key = "right"
		}
		if key != "" {
			m.move(key, k.Ctrl || (key == "home" || key == "end") && m.rules.Selection != TextCaretSelection,
				k.Shift && (m.rules.Selection == TextCaretSelection || m.rules.Selection == TextDetachedSelection))
		}
		if m.rules.Selection == TextReplaceAllSelection && (k.Home || k.End || k.Left || k.Right) {
			m.SelectedAll = false
		}
	}
	if !k.Backspace && (!k.Delete || m.rules.AppendOnly) && k.Typed == "" {
		return
	}
	if m.SelectedAll {
		m.replace("")
		m.SelectedAll = false
	} else if k.Backspace && (m.Caret > 0 || m.rules.Selection == TextDetachedSelection) {
		m.erase(true)
	} else if k.Delete && !m.rules.AppendOnly {
		m.erase(false)
	}
	m.typeText(k.Typed)
}

func (m *TextInputState) typeText(s string) bool {
	if m.rules.AcceptBatch != nil && !m.rules.AcceptBatch(s) {
		return false
	}
	for _, r := range s {
		if m.rules.ByteUnits && m.rules.AppendOnly {
			if m.rules.Limit > 0 && len(m.units()) >= m.rules.Limit {
				continue
			}
			m.TypedBelowLimit = true
		}
		if m.rules.Accept != nil && !m.rules.Accept(r) || m.rules.Accept == nil && m.rules.Encode == nil && unicode.IsControl(r) {
			continue
		}
		token := string(r)
		if m.rules.Encode != nil {
			b, ok := m.rules.Encode(r)
			if !ok || b < 0x20 {
				continue
			}
			token = string([]byte{b})
		}
		if m.rules.Selection == TextDetachedSelection {
			m.clamp()
			if lo, hi := m.selection(); lo < hi {
				m.replace("")
			}
		}
		if m.rules.Limit > 0 && len(m.units()) >= m.rules.Limit || m.rules.ByteLimit > 0 && len(m.Text)+len(token) > m.rules.ByteLimit {
			continue
		}
		m.replace(token)
	}
	return true
}

func textInputKeys(in appInput) TextInputKeys {
	return TextInputKeys{Typed: in.Typed, Backspace: in.Backspace, Delete: in.Delete,
		Home: in.Home, End: in.End, Left: in.Left, Right: in.Right, SelectAll: in.SelectText, Ctrl: in.Viewer.Ctrl}
}

const (
	editorRepeatDelay = 400 * time.Millisecond
	editorRepeatRate  = 40 * time.Millisecond
)

type TextInputState struct {
	Text                                  string
	Caret, Anchor                         int
	repeating                             map[string]time.Time
	rules                                 TextInputRules
	SelectedAll, TypedBelowLimit, Handled bool
	Start, End                            int
}

func (m *TextInputState) reset(s string) {
	rules := m.rules
	*m = TextInputState{Text: s, rules: rules}
	m.Caret, m.Anchor = len(m.units()), len(m.units())
}

func (m *TextInputState) selection() (int, int) {
	if m.SelectedAll {
		return 0, len(m.units())
	}
	if m.rules.Selection == TextDetachedSelection {
		if m.End > m.Start {
			return m.Start, m.End
		}
		return m.Caret, m.Caret
	}
	return min(m.Caret, m.Anchor), max(m.Caret, m.Anchor)
}

func (m *TextInputState) selectAll() {
	if m.rules.Selection == TextCaretSelection {
		m.Anchor, m.Caret = 0, len(m.units())
	} else {
		m.SelectedAll = true
	}
}

func editorWord(r rune) bool { return unicode.IsLetter(r) || unicode.IsNumber(r) || r == '_' }

func editorLineEnd(r []rune, at int) bool {
	return r[at] == '\n' || r[at] == '\r' && (at+1 == len(r) || r[at+1] == '\n')
}

func (m *TextInputState) move(key string, ctrl, shift bool) {
	if m.rules.LiteralArrows && (key == "left" || key == "right") {
		ctrl = false
	}
	r := m.units()
	at := m.Caret
	lo, hi := m.selection()
	if m.rules.Selection == TextCaretSelection && !shift && lo != hi && !ctrl && (key == "left" || key == "right") {
		at = lo
		if key == "right" {
			at = hi
		}
	} else {
		switch key {
		case "left":
			at = max(0, at-1)
			if !m.rules.SplitLineBreaks && !ctrl && at > 0 && r[at] == '\n' && r[at-1] == '\r' {
				at--
			}
			if ctrl {
				for at > 0 && !editorWord(r[at]) {
					at--
				}
				for at > 0 && editorWord(r[at-1]) {
					at--
				}
			}
		case "right":
			if ctrl {
				for at < len(r) && editorWord(r[at]) {
					at++
				}
				for at < len(r) && !editorWord(r[at]) {
					at++
				}
			} else {
				if !m.rules.SplitLineBreaks && at+1 < len(r) && r[at] == '\r' && r[at+1] == '\n' {
					at += 2
				} else {
					at = min(len(r), at+1)
				}
			}
		case "home":
			if ctrl {
				at = 0
			} else {
				for at > 0 && r[at-1] != '\n' {
					at--
				}
			}
		case "end":
			if ctrl {
				at = len(r)
			} else {
				for at < len(r) && !editorLineEnd(r, at) {
					at++
				}
			}
		case "up", "down":
			start := at
			for start > 0 && r[start-1] != '\n' {
				start--
			}
			column := at - start
			if key == "up" {
				at = max(0, start-1)
				if at > 0 && r[at-1] == '\r' {
					at--
				}
				start = at
				for start > 0 && r[start-1] != '\n' {
					start--
				}
				at = min(at, start+column)
			} else {
				for at < len(r) && r[at] != '\n' {
					at++
				}
				start = min(len(r), at+1)
				at = start
				for at < len(r) && !editorLineEnd(r, at) && at-start < column {
					at++
				}
			}
		}
	}
	m.moveTo(at, shift)
}

func (m *TextInputState) moveTo(at int, shift bool) {
	if m.rules.Selection == TextDetachedSelection {
		m.clamp()
	}
	old := m.Caret
	m.Caret = max(0, min(at, len(m.units())))
	if m.rules.Selection == TextDetachedSelection {
		if !shift {
			m.End = m.Start
			return
		}
		if m.Start == m.End {
			m.Start, m.End = old, old
		}
		switch {
		case old == m.End:
			m.End = m.Caret
		case old == m.Start:
			m.Start = m.Caret
		default:
			m.Start, m.End = min(old, m.Caret), max(old, m.Caret)
		}
		m.Start, m.End = min(m.Start, m.End), max(m.Start, m.End)
		return
	}
	if !shift {
		m.Anchor = m.Caret
	}
}

func (m *TextInputState) replace(s string) {
	r := m.units()
	lo, hi := m.selection()
	m.Text = m.join(r[:lo]) + s + m.join(r[hi:])
	n := len([]rune(s))
	if m.rules.ByteUnits {
		n = len(s)
	}
	m.Caret = lo + n
	m.Anchor = m.Caret
	if m.rules.Selection == TextDetachedSelection {
		m.Start, m.End = m.Caret, m.Caret
	}
}

func (m *TextInputState) erase(back bool) {
	if m.rules.Selection == TextDetachedSelection {
		m.clamp()
	}
	if m.rules.Selection == TextDetachedSelection && back {
		if m.Caret == 0 {
			return
		}
		start, end := m.Start, m.End
		m.Start, m.End = m.Caret-1, m.Caret
		m.replace("")
		m.Start, m.End, m.Handled = start, end, true
		return
	}
	lo, hi := m.selection()
	if lo == hi {
		r := m.units()
		if back {
			m.Anchor = max(0, m.Caret-1)
			if !m.rules.SplitLineBreaks && m.Caret >= 2 && r[m.Caret-2] == '\r' && r[m.Caret-1] == '\n' {
				m.Anchor--
			}
		} else {
			m.Anchor = min(len(r), m.Caret+1)
			if !m.rules.SplitLineBreaks && m.Caret+1 < len(r) && r[m.Caret] == '\r' && r[m.Caret+1] == '\n' {
				m.Anchor++
			}
		}
		if m.rules.Selection == TextDetachedSelection {
			start, end := m.Start, m.End
			m.Start, m.End = m.Caret, m.Anchor
			m.replace("")
			m.Start, m.End = start, end
			return
		}
	}
	m.replace("")
	m.Handled = true
}

func (m *TextInputState) clamp() {
	n := len(m.units())
	m.Caret = max(0, min(m.Caret, n))
	m.Start, m.End = max(0, min(m.Start, n)), max(0, min(m.End, n))
}

func (m *TextInputState) selected() string {
	lo, hi := m.selection()
	return m.join(m.units()[lo:hi])
}

func (m *TextInputState) drag(at int, extend bool) {
	m.Caret = max(0, min(len(m.units()), at))
	if !extend {
		m.Anchor = m.Caret
	}
}

func (m *TextInputState) repeat(now time.Time, held map[string]bool) []string {
	if m.repeating == nil {
		m.repeating = make(map[string]time.Time)
	}
	var out []string
	for _, key := range []string{"backspace", "delete", "left", "right", "up", "down", "home", "end"} {
		if !held[key] {
			delete(m.repeating, key)
			continue
		}
		next, exists := m.repeating[key]
		if !exists {
			out = append(out, key)
			m.repeating[key] = now.Add(editorRepeatDelay)
		} else if !now.Before(next) {
			out = append(out, key)
			m.repeating[key] = next.Add((now.Sub(next)/editorRepeatRate + 1) * editorRepeatRate)
		}
	}
	return out
}

func editorSingleLine(s string, multiline bool) string {
	s = strings.ReplaceAll(strings.ReplaceAll(s, "\r\n", "\n"), "\r", "\n")
	if !multiline {
		s = strings.ReplaceAll(s, "\n", " ")
	}
	return s
}
