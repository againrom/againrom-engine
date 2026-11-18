package ui

import (
	"bytes"
	"encoding/json"
	"image"
	"strings"

	"againrom/pkg/render/text"
	"againrom/pkg/render/textfit"
	"againrom/pkg/words"
)

type popupPreview struct {
	Width     int                `json:"width,omitempty"`
	Top       int                `json:"top,omitempty"`
	Dock      string             `json:"dock,omitempty"`
	DockSize  int                `json:"dockSize,omitempty"`
	MaxWidth  int                `json:"maxWidth,omitempty"`
	MaxHeight int                `json:"maxHeight,omitempty"`
	Hover     *popupPreviewHover `json:"hover,omitempty"`
}

func (p *popupPreview) UnmarshalJSON(data []byte) error {
	type fields popupPreview
	wire := struct {
		*fields
		BodyTop     int  `json:"bodyTop"`
		HoverRow    *int `json:"hoverRow"`
		HoverList   bool `json:"hoverList"`
		HoverColumn int  `json:"hoverColumn"`
	}{fields: (*fields)(p)}
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&wire); err != nil {
		return err
	}
	if p.Top == 0 {
		p.Top = wire.BodyTop
	}
	if p.Hover == nil && wire.HoverRow != nil {
		part := "row"
		if wire.HoverList {
			part = "list"
		}
		p.Hover = &popupPreviewHover{Part: part, Row: *wire.HoverRow, Column: wire.HoverColumn}
	}
	return nil
}

func (p *popupPreview) panelPlacement() bool {
	return p.Dock != "" || p.MaxWidth > 0 || p.MaxHeight > 0
}

type popupPreviewHover struct {
	Part   string `json:"part"`
	Row    int    `json:"row,omitempty"`
	Column int    `json:"column,omitempty"`
}

type pageTip struct {
	popupTip
	Full string
}

func (m *EditorMockup) placePreview(p *popupPreview, word func(string) string) {
	e := NewMapEditor(m.title, nil, nil)
	e.catalog = false
	e.SetHostTextEncoder(m.encode)
	e.SetWords(words.For(m.lang))
	e.SetBodyFont(m.body)
	e.Layout(m.window.X, m.window.Y)
	switch p.Dock {
	case "left", "bottom":
		i := panelLeft
		if p.Dock == "bottom" {
			i = panelBottom
		}
		e.panels.state[i] = panelState{Restored: true, Width: p.DockSize}
		m.paint = e.inPanel(m.paint, i)
		m.plain.CloseLabel = word(sidePanels[i].minimizeWord)
		m.paint.CloseLabel = m.encode(m.plain.CloseLabel)
	default:
		s := &m.paint
		room := e.popupBounds()
		s.Bounds = room
		if p.MaxWidth > 0 {
			s.Width = min(s.Width, p.MaxWidth)
		}
		if p.MaxHeight > 0 {
			s.Bounds.Max.Y = min(s.Bounds.Max.Y, s.Bounds.Min.Y+p.MaxHeight)
		}
		s.Fit, s.Page = true, max(0, p.Top)
		e.openCentred(popList, s)
		if p.MaxHeight > 0 {
			s.Bounds.Min.Y = s.At.Y
			s.Bounds.Max.Y = min(room.Max.Y, s.At.Y+p.MaxHeight)
		}
	}
	m.layout = layoutPopup(m.paint, m.title, m.body)
	if m.paint.List != nil {
		m.paint.List.SetWindow(m.layout.ListRows)
		m.plain.List.SetWindow(m.layout.ListRows)
	}
}

func (m *EditorMockup) previewFit(face *text.Font, value string, width int) string {
	if face == nil {
		return value
	}
	fit := textfit.Spec{Measure: func(s string) int { return face.Advance(m.encode(s)) }}
	return fit.Fit(value, width)
}

func (m *EditorMockup) previewHelp() string {
	if fold := len(m.plain.Notes) - len(m.layout.Notes); fold > 0 {
		return strings.TrimSpace(m.plain.TitleHelp + " " + strings.Join(m.plain.Notes[:fold], " "))
	}
	return m.plain.TitleHelp
}

func (m *EditorMockup) previewTip() pageTip {
	if m.desc.Preview == nil || m.desc.Preview.Hover == nil || m.body == nil {
		return pageTip{}
	}
	h, l := m.desc.Preview.Hover, m.layout
	bounds := image.Rectangle{Max: m.window}
	if h.Part != "list" && !m.desc.Preview.panelPlacement() {
		bounds = m.paint.Bounds
	}
	read := func(r image.Rectangle) (popupTip, bool) {
		return popupTipAt(bounds, m.title, m.body, m.paint, l, r.Min.Add(r.Size().Div(2)), true)
	}
	switch h.Part {
	case "titleHelp":
		if !l.TitleHelp.Empty() {
			tip, _ := read(l.TitleHelp)
			return pageTip{tip, m.previewHelp()}
		}
	case "row":
		for _, r := range l.Rows {
			if r.Index == h.Row && !r.Control.Empty() {
				if tip, ok := read(r.Control); ok {
					row := m.plain.Rows[r.Index]
					return pageTip{tip, strings.TrimSpace(row.Label + " " + row.Value)}
				}
			}
		}
	case "list":
		if m.paint.List == nil {
			break
		}
		top, shown := m.paint.List.Visible()
		if i := h.Row - top; i >= 0 && i < min(shown, l.List.Rows) {
			r, full := l.List.Row(i), m.plain.List.Rows()[h.Row].Text
			if cols := len(l.List.Columns); cols > 0 {
				cells := strings.SplitN(full, "\t", cols)
				if h.Column < 0 || h.Column >= len(cells) {
					break
				}
				r, full = l.List.Cell(i, h.Column), cells[h.Column]
			}
			if tip, ok := read(r); ok {
				return pageTip{tip, full}
			}
		}
	}
	return pageTip{}
}
