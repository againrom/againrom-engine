package ui

import (
	"image"
	"image/color"
	"image/draw"

	"againrom/pkg/render/text"
)

type editorTextSpan struct{ from, to int }

func (s editorTextSpan) lastCaret(r []rune) int {
	if s.to < len(r) && r[s.to] == '\r' {
		return s.to + 1
	}
	return s.to
}

type editorTextView struct {
	text          string
	caret, anchor int
	span          editorTextSpan
	encode        func(string) string
	multiline     bool
	first         int
}

func editorTextSpans(s string, width int, measure func(string) int) []editorTextSpan {
	r := []rune(s)
	var spans []editorTextSpan
	start, at, space, size := 0, 0, -1, 0
	emit := func(end int) {
		if end > start && r[end-1] == '\r' && (end == len(r) || r[end] == '\n') {
			end--
		}
		spans = append(spans, editorTextSpan{start, end})
	}
	for at < len(r) {
		if r[at] == '\r' && (at+1 == len(r) || r[at+1] == '\n') {
			at++
			continue
		}
		if r[at] == '\n' {
			emit(at)
			at++
			start, space, size = at, -1, 0
			continue
		}
		w := measure(string(r[at]))
		if size+w > width && at > start {
			end := at
			if space >= start {
				end = space + 1
			}
			emit(end)
			start, at, space, size = end, end, -1, 0
			continue
		}
		if r[at] == ' ' {
			space = at
		}
		size += w
		at++
	}
	emit(len(r))
	return spans
}

func (e *MapEditor) textMeasure(s string) int {
	if f := e.bodyFont(); f != nil {
		return f.Advance(e.editorHostText(s))
	}
	return 0
}

func (e *MapEditor) inputView(span editorTextSpan, multiline bool) *editorTextView {
	m, _ := e.inputModel()
	if m == nil {
		return nil
	}
	return &editorTextView{text: m.Text, caret: m.Caret, anchor: m.Anchor, span: span, multiline: multiline, encode: e.editorHostText}
}

func (e *MapEditor) promptInputRows(width int) []popupRow {
	m, _ := e.inputModel()
	if m == nil {
		return nil
	}
	spans := editorTextSpans(m.Text, width, e.textMeasure)
	r := []rune(m.Text)
	caretLine := len(spans) - 1
	for i, span := range spans {
		if m.Caret <= span.lastCaret(r) {
			caretLine = i
			break
		}
	}
	start := max(0, caretLine-5)
	end := min(len(spans), start+6)
	var rows []popupRow
	for i := start; i < end; i++ {
		span := spans[i]
		value := e.editorHostText(string(r[span.from:span.to]))
		row := popupRow{Kind: popupEdit, Value: value, Focus: i == caretLine, Edit: e.inputView(span, false)}
		row.Caret = e.textMeasure(string(r[span.from:max(span.from, min(span.to, m.Caret))]))
		rows = append(rows, row)
	}
	return rows
}

func (e *MapEditor) objectInputRows(s popupSpec) popupSpec {
	if !e.objectTyping() {
		return s
	}
	m, _ := e.inputModel()
	if m == nil {
		return s
	}
	for i := range s.Rows {
		row := &s.Rows[i]
		if e.obj.rowField[i] != e.obj.focus {
			continue
		}
		if _, exists := e.obj.rowField[i]; !exists {
			continue
		}
		if row.Kind == popupEdit || row.Kind == popupPick || row.Kind == popupMultiline || row.Kind == popupChoice && m.Caret != m.Anchor {
			row.Edit = e.inputView(editorTextSpan{0, len([]rune(m.Text))}, row.Kind == popupMultiline)
			row.Edit.first = row.First
			row.Focus = true
		}
	}
	return s
}

func (e *MapEditor) addInputRows(s popupSpec) popupSpec {
	if e.prompt.kind != promptAdd && e.prompt.kind != promptNewMode {
		return s
	}
	m, _ := e.inputModel()
	if m == nil {
		return s
	}
	for i := range s.Rows {
		row := &s.Rows[i]
		if row.Focus && (row.Kind == popupEdit || row.Kind == popupPick || row.Kind == popupChoice && m.Caret != m.Anchor) {
			row.Edit = e.inputView(editorTextSpan{0, len([]rune(m.Text))}, false)
		}
	}
	return s
}

func (v *editorTextView) spans(box image.Rectangle, f *text.Font) ([]editorTextSpan, int) {
	if !v.multiline {
		return []editorTextSpan{v.span}, 0
	}
	spans := editorTextSpans(v.text, box.Dx()-8, func(s string) int { return f.Advance(v.encode(s)) })
	r := []rune(v.text)
	visible := max(1, (box.Dy()-8)/fontHeight(f, 15))
	line := len(spans) - 1
	for i, span := range spans {
		if v.caret <= span.lastCaret(r) {
			line = i
			break
		}
	}
	start := max(0, min(v.first, len(spans)-visible))
	if line < start {
		start = line
	}
	if line >= start+visible {
		start = line - visible + 1
	}
	return spans[start:min(len(spans), start+visible)], start
}

func (v *editorTextView) lineOffsets(span editorTextSpan, box image.Rectangle, f *text.Font) (string, int, int, int, int) {
	r := []rune(v.text)
	lo, hi := min(v.caret, v.anchor), max(v.caret, v.anchor)
	measure := func(at int) int { return f.Advance(v.encode(string(r[span.from:max(span.from, min(span.to, at))]))) }
	caret := measure(v.caret)
	shift := 0
	if !v.multiline {
		shift = max(0, caret-(box.Dx()-12))
	}
	return v.encode(string(r[span.from:span.to])), caret - shift, measure(lo) - shift, measure(hi) - shift, shift
}

func drawEditorInput(dst *image.RGBA, f *text.Font, row popupRow, box image.Rectangle, ink popupInk, value color.RGBA) {
	v := row.Edit
	if f == nil {
		if v.multiline {
			drawFrame(dst, panelFrame(box, ink.Ground, ink.Edge))
		} else {
			drawEditField(dst, editField{Rect: box, TextH: 15, Focus: row.Focus, Phase: true, Caret: row.Caret}, nil)
		}
		return
	}
	spans, _ := v.spans(box, f)
	if !v.multiline {
		label, caret, lo, hi, shift := v.lineOffsets(spans[0], box, f)
		drawEditField(dst, editField{Rect: box, TextH: fontHeight(f, 15), Focus: row.Focus, Phase: true, Caret: caret, SelFrom: lo, SelTo: hi}, func(at image.Point) {
			if label == "" && row.Hint != "" {
				label, value = row.Hint, popupHintInk
			}
			f.Draw(dst.SubImage(box.Inset(2)).(*image.RGBA), label, at.X-shift, at.Y, value)
		})
		return
	}
	drawFrame(dst, panelFrame(box, ink.Ground, ink.Edge))
	clip := dst.SubImage(box.Inset(3)).(*image.RGBA)
	y := box.Min.Y + 4
	lo, hi := min(v.caret, v.anchor), max(v.caret, v.anchor)
	runes := []rune(v.text)
	for _, span := range spans {
		label, caret, from, to, _ := v.lineOffsets(span, box, f)
		if lo != hi && hi > span.to && lo <= span.to {
			to += 5
		}
		if to > from {
			r := image.Rect(box.Min.X+4+from, y, box.Min.X+4+to, y+fontHeight(f, 15)).Intersect(clip.Bounds())
			draw.Draw(clip, r, image.NewUniform(color.RGBA{0x55, 0x65, 0x8c, 255}), image.Point{}, draw.Src)
		}
		f.Draw(clip, label, box.Min.X+4, y, value)
		if row.Focus && v.caret >= span.from && v.caret <= span.lastCaret(runes) {
			for yy := y; yy < y+fontHeight(f, 15); yy++ {
				if p := image.Pt(box.Min.X+4+caret, yy); p.In(clip.Bounds()) {
					clip.SetRGBA(p.X, p.Y, value)
				}
			}
		}
		y += fontHeight(f, 15)
	}
}

func (v *editorTextView) hit(p image.Point, box image.Rectangle, f *text.Font) int {
	if f == nil {
		return v.span.to
	}
	spans, _ := v.spans(box, f)
	line := 0
	if v.multiline {
		line = max(0, min(len(spans)-1, (p.Y-box.Min.Y-4)/fontHeight(f, 15)))
	}
	span := spans[line]
	_, _, _, _, shift := v.lineOffsets(span, box, f)
	x := p.X - box.Min.X - 4 + shift
	r, used := []rune(v.text), 0
	for at := span.from; at < span.to; at++ {
		w := f.Advance(v.encode(string(r[at])))
		if x < used+w/2 {
			return at
		}
		used += w
	}
	return span.to
}

func (e *MapEditor) inputPointer(in Input) bool {
	p := image.Pt(in.CursorX, in.CursorY)
	if e.input.drag {
		if !in.PrimaryDown {
			e.input.drag = false
			e.down, e.uiDown = false, false
			return true
		}
		if v, box := e.inputAt(p, false); v != nil {
			e.input.model.drag(v.hit(p, box, e.bodyFont()), true)
		}
		e.down = true
		return true
	}
	if !in.PrimaryDown || e.down || e.menu != "" || e.dropdownOpen() {
		return false
	}
	if v, box := e.inputAt(p, true); v != nil {
		e.input.model.drag(v.hit(p, box, e.bodyFont()), in.Shift)
		e.input.drag = true
		e.down, e.uiDown = true, false
		return true
	}
	return false
}

func (e *MapEditor) inputAt(p image.Point, press bool) (*editorTextView, image.Rectangle) {
	if e.prompt.open() {
		s, ok := e.promptSpec()
		if !ok {
			return nil, image.Rectangle{}
		}
		l := layoutPopup(s, e.font, e.bodyFont())
		var nearest *editorTextView
		var nearestBox image.Rectangle
		distance := int(^uint(0) >> 1)
		for _, rr := range l.Rows {
			row := s.Rows[rr.Index]
			if row.Edit == nil {
				continue
			}
			if p.In(rr.Control) {
				return row.Edit, rr.Control
			}
			if d := max(rr.Control.Min.Y-p.Y, p.Y-rr.Control.Max.Y); d < distance {
				distance, nearest, nearestBox = d, row.Edit, rr.Control
			}
		}
		if !press {
			return nearest, nearestBox
		}
	} else if s, _, ok := e.objectSpec(); ok {
		l := layoutPopup(s, e.font, e.bodyFont())
		for _, rr := range l.Rows {
			row := s.Rows[rr.Index]
			field, exists := e.obj.rowField[rr.Index]
			if !exists || row.Kind != popupEdit && row.Kind != popupMultiline && row.Kind != popupPick {
				continue
			}
			if !p.In(rr.Control) && (press || field != e.obj.focus) {
				continue
			}
			if press && field != e.obj.focus {
				e.obj.focus, e.obj.open = field, -1
				row.Edit = e.inputView(editorTextSpan{0, len([]rune(e.obj.buf[field]))}, row.Kind == popupMultiline)
			}
			if row.Edit != nil {
				return row.Edit, rr.Control
			}
		}
	}
	return nil, image.Rectangle{}
}
