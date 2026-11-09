package widgetkit

import (
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
	m.Rules = rules
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
	if m.Rules.ByteUnits {
		r := make([]rune, len(m.Text))
		for i, b := range []byte(m.Text) {
			r[i] = rune(b)
		}
		return r
	}
	return []rune(m.Text)
}

func (m *TextInputState) join(r []rune) string {
	if m.Rules.ByteUnits {
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
	if !m.Rules.AppendOnly {
		if k.SelectAll {
			m.SelectAll()
			return
		}
		key := ""
		switch {
		case k.MoveTo != nil:
			m.MoveTo(*k.MoveTo, k.Shift)
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
			m.Move(key, k.Ctrl || (key == "home" || key == "end") && m.Rules.Selection != TextCaretSelection,
				k.Shift && (m.Rules.Selection == TextCaretSelection || m.Rules.Selection == TextDetachedSelection))
		}
		if m.Rules.Selection == TextReplaceAllSelection && (k.Home || k.End || k.Left || k.Right) {
			m.SelectedAll = false
		}
	}
	if !k.Backspace && (!k.Delete || m.Rules.AppendOnly) && k.Typed == "" {
		return
	}
	if m.SelectedAll {
		m.Replace("")
		m.SelectedAll = false
	} else if k.Backspace && (m.Caret > 0 || m.Rules.Selection == TextDetachedSelection) {
		m.Erase(true)
	} else if k.Delete && !m.Rules.AppendOnly {
		m.Erase(false)
	}
	m.TypeText(k.Typed)
}

func (m *TextInputState) TypeText(s string) bool {
	if m.Rules.AcceptBatch != nil && !m.Rules.AcceptBatch(s) {
		return false
	}
	for _, r := range s {
		if m.Rules.ByteUnits && m.Rules.AppendOnly {
			if m.Rules.Limit > 0 && len(m.units()) >= m.Rules.Limit {
				continue
			}
			m.TypedBelowLimit = true
		}
		if m.Rules.Accept != nil && !m.Rules.Accept(r) || m.Rules.Accept == nil && m.Rules.Encode == nil && unicode.IsControl(r) {
			continue
		}
		token := string(r)
		if m.Rules.Encode != nil {
			b, ok := m.Rules.Encode(r)
			if !ok || b < 0x20 {
				continue
			}
			token = string([]byte{b})
		}
		if m.Rules.Selection == TextDetachedSelection {
			m.clamp()
			if lo, hi := m.Selection(); lo < hi {
				m.Replace("")
			}
		}
		if m.Rules.Limit > 0 && len(m.units()) >= m.Rules.Limit || m.Rules.ByteLimit > 0 && len(m.Text)+len(token) > m.Rules.ByteLimit {
			continue
		}
		m.Replace(token)
	}
	return true
}

const (
	editorRepeatDelay = 400 * time.Millisecond
	editorRepeatRate  = 40 * time.Millisecond
)

type TextInputState struct {
	Text                                  string
	Caret, Anchor                         int
	Repeating                             map[string]time.Time
	Rules                                 TextInputRules
	SelectedAll, TypedBelowLimit, Handled bool
	Start, End                            int
}

func (m *TextInputState) Reset(s string) {
	rules := m.Rules
	*m = TextInputState{Text: s, Rules: rules}
	m.Caret, m.Anchor = len(m.units()), len(m.units())
}

func (m *TextInputState) Selection() (int, int) {
	if m.SelectedAll {
		return 0, len(m.units())
	}
	if m.Rules.Selection == TextDetachedSelection {
		if m.End > m.Start {
			return m.Start, m.End
		}
		return m.Caret, m.Caret
	}
	return min(m.Caret, m.Anchor), max(m.Caret, m.Anchor)
}

func (m *TextInputState) SelectAll() {
	if m.Rules.Selection == TextCaretSelection {
		m.Anchor, m.Caret = 0, len(m.units())
	} else {
		m.SelectedAll = true
	}
}

func editorWordClass(r rune) int {
	switch {
	case unicode.IsSpace(r):
		return 0
	case unicode.IsLetter(r) || unicode.IsDigit(r):
		return 1
	default:
		return 2
	}
}

func editorLineEnd(r []rune, at int) bool {
	return r[at] == '\n' || r[at] == '\r' && (at+1 == len(r) || r[at+1] == '\n')
}

func (m *TextInputState) Move(key string, ctrl, shift bool) {
	if m.Rules.LiteralArrows && (key == "left" || key == "right") {
		ctrl = false
	}
	r := m.units()
	at := m.Caret
	lo, hi := m.Selection()
	if m.Rules.Selection == TextCaretSelection && !shift && lo != hi && !ctrl && (key == "left" || key == "right") {
		at = lo
		if key == "right" {
			at = hi
		}
	} else {
		switch key {
		case "left":
			if ctrl {
				at = m.DeleteWordEdge(true)
			} else {
				at = max(0, at-1)
				if !m.Rules.SplitLineBreaks && at > 0 && r[at] == '\n' && r[at-1] == '\r' {
					at--
				}
			}
		case "right":
			if ctrl {
				at = m.DeleteWordEdge(false)
			} else {
				if !m.Rules.SplitLineBreaks && at+1 < len(r) && r[at] == '\r' && r[at+1] == '\n' {
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
	m.MoveTo(at, shift)
}

func (m *TextInputState) MoveTo(at int, shift bool) {
	if m.Rules.Selection == TextDetachedSelection {
		m.clamp()
	}
	old := m.Caret
	m.Caret = max(0, min(at, len(m.units())))
	if m.Rules.Selection == TextDetachedSelection {
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

func (m *TextInputState) Replace(s string) {
	r := m.units()
	lo, hi := m.Selection()
	m.Text = m.join(r[:lo]) + s + m.join(r[hi:])
	n := len([]rune(s))
	if m.Rules.ByteUnits {
		n = len(s)
	}
	m.Caret = lo + n
	m.Anchor = m.Caret
	if m.Rules.Selection == TextDetachedSelection {
		m.Start, m.End = m.Caret, m.Caret
	}
}

func (m *TextInputState) DeleteWordEdge(back bool) int {
	r := m.units()
	at, end, step, offset := m.Caret, len(r), 1, 0
	if back {
		end, step, offset = 0, -1, -1
	}
	for at != end && editorWordClass(r[at+offset]) == 0 {
		at += step
	}
	if at != end {
		kind := editorWordClass(r[at+offset])
		for at != end && editorWordClass(r[at+offset]) == kind {
			at += step
		}
	}
	return at
}

func (m *TextInputState) Erase(back bool) {
	if m.Rules.Selection == TextDetachedSelection {
		m.clamp()
	}
	if m.Rules.Selection == TextDetachedSelection && back {
		if m.Caret == 0 {
			return
		}
		start, end := m.Start, m.End
		m.Start, m.End = m.Caret-1, m.Caret
		m.Replace("")
		m.Start, m.End, m.Handled = start, end, true
		return
	}
	lo, hi := m.Selection()
	if lo == hi {
		r := m.units()
		if back {
			m.Anchor = max(0, m.Caret-1)
			if !m.Rules.SplitLineBreaks && m.Caret >= 2 && r[m.Caret-2] == '\r' && r[m.Caret-1] == '\n' {
				m.Anchor--
			}
		} else {
			m.Anchor = min(len(r), m.Caret+1)
			if !m.Rules.SplitLineBreaks && m.Caret+1 < len(r) && r[m.Caret] == '\r' && r[m.Caret+1] == '\n' {
				m.Anchor++
			}
		}
		if m.Rules.Selection == TextDetachedSelection {
			start, end := m.Start, m.End
			m.Start, m.End = m.Caret, m.Anchor
			m.Replace("")
			m.Start, m.End = start, end
			return
		}
	}
	m.Replace("")
	m.Handled = true
}

func (m *TextInputState) clamp() {
	n := len(m.units())
	m.Caret = max(0, min(m.Caret, n))
	m.Start, m.End = max(0, min(m.Start, n)), max(0, min(m.End, n))
}

func (m *TextInputState) Selected() string {
	lo, hi := m.Selection()
	return m.join(m.units()[lo:hi])
}

func (m *TextInputState) Drag(at int, extend bool) {
	m.Caret = max(0, min(len(m.units()), at))
	if !extend {
		m.Anchor = m.Caret
	}
}

func (m *TextInputState) Repeat(now time.Time, held map[string]bool, reached func(deadline, now time.Time) bool) []string {
	if m.Repeating == nil {
		m.Repeating = make(map[string]time.Time)
	}
	var out []string
	for _, key := range []string{"backspace", "delete", "left", "right", "up", "down", "home", "end"} {
		if !held[key] {
			delete(m.Repeating, key)
			continue
		}
		next, exists := m.Repeating[key]
		if !exists {
			out = append(out, key)
			m.Repeating[key] = now.Add(editorRepeatDelay)
		} else if reached(next, now) {
			out = append(out, key)
			m.Repeating[key] = next.Add((now.Sub(next)/editorRepeatRate + 1) * editorRepeatRate)
		}
	}
	return out
}
