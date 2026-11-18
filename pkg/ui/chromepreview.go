package ui

import (
	"fmt"
	"image"
	"image/color"
	"strings"

	"againrom/pkg/render/text"
	"againrom/pkg/words"
)

type chromePreview struct {
	Maps      []string `json:"maps"`
	Floating  bool     `json:"floating,omitempty"`
	HoverTool string   `json:"hoverTool,omitempty"`
	Menu      string   `json:"menu,omitempty"`
	Query     *string  `json:"query,omitempty"`
}

type mockupChrome struct {
	editor *MapEditor
	decode func(string) string
	menu   dropdown
	search popupLayout
	tip    popupTip
}

func newMockupChrome(p chromePreview, lang string, window image.Point, title, body *text.Font, encode, decode func(string) string) (*mockupChrome, error) {
	if window.X < 1920 || window.Y < 1080 {
		return nil, fmt.Errorf("editor chrome needs at least 1920x1080")
	}
	if len(p.Maps) < 1 || len(p.Maps) > maxOpenMaps {
		return nil, fmt.Errorf("chrome preview needs 1 to %d map names", maxOpenMaps)
	}
	if p.Query != nil && (p.Menu != "" || p.HoverTool != "") {
		return nil, fmt.Errorf("search, an open menu and a tool hover are separate preview states")
	}
	if title == nil {
		title = &text.Font{Glyphs: make([]text.Glyph, 256-text.FirstChar)}
		for i := range title.Glyphs {
			title.Glyphs[i] = text.Glyph{Advance: 8, Height: 15}
		}
	}
	if decode == nil {
		decode = func(s string) string { return strings.ToValidUTF8(s, "") }
	}
	e := NewMapEditor(title, nil, nil)
	e.window, e.body = window, body
	e.book, e.hasBook = words.For(lang), true
	e.SetHostTextEncoder(encode, decode)
	e.catalog = false
	e.doc = &InspectionDocument{Name: p.Maps[0]}
	if len(p.Maps) > 1 {
		for _, name := range p.Maps {
			e.openMaps = append(e.openMaps, &openMap{doc: &InspectionDocument{Name: name}})
		}
	}
	e.panels.floats[popToolbar] = p.Floating
	if p.Floating {
		size := e.toolbarLayout().Rect.Size()
		at := e.centreCorner(size, openPlace{})
		e.floatPlace(popToolbar, image.Rectangle{Min: at, Max: at.Add(size)})
	}
	c := &mockupChrome{editor: e, decode: decode}
	bounds := image.Rectangle{Max: window}
	if p.Menu != "" {
		e.menu = p.Menu
		b, ok := e.openMenuEntry()
		if !ok {
			return nil, fmt.Errorf("unknown bar menu %q", p.Menu)
		}
		c.menu = e.menuDropdown(b, true)
	}
	if p.HoverTool != "" {
		r, ok := e.controlToolbarRect(p.HoverTool)
		if !ok {
			return nil, fmt.Errorf("unknown picture button %q", p.HoverTool)
		}
		e.pointer, e.pointerOK = r.Min.Add(r.Size().Div(2)), true
		c.tip, _ = popupTipAt(bounds, e.font, e.bodyFont(), popupSpec{}, e.toolbarLayout(), e.pointer, true)
	}
	if p.Query != nil {
		e.search.text = *p.Query
		e.search.typing = true
		e.searchFollow()
		c.search = e.searchLayout()
		r := e.searchEntry().rect
		c.tip, _ = popupTipAt(bounds, e.font, e.bodyFont(), popupSpec{}, c.search, r.Min.Add(r.Size().Div(2)), true)
	}
	return c, nil
}

func (c *mockupChrome) draw(dst *image.RGBA) {
	e := c.editor
	e.drawTopBar(dst)
	e.drawMapTabs(dst)
	e.drawToolbar(dst, false)
	if !c.search.Rect.Empty() {
		e.drawBar(dst, c.search)
	}
	e.drawToolbar(dst, true)
	if c.menu.Open {
		e.drawEditorDropdown(dst, e.font, c.menu)
	}
	if !c.tip.Rect.Empty() {
		drawPopupTip(dst, e.bodyFont(), c.tip, editorPopupInk)
	}
}

func (c *mockupChrome) writePage(write pagePartWriter) {
	e := c.editor
	ink := editorPopupInk
	none := color.RGBA{}
	part := func(class, name string, row int, r image.Rectangle, ink color.RGBA, txt string) {
		write(class, name, row, r, ink, txt, "")
	}
	part("window", "top-bar", -1, e.topBarLayout().Rect, none, "")
	for i, b := range e.barEntries() {
		label := c.decode(b.label)
		if b.menu {
			label += " v"
		}
		part(chromeButtonClass(b.on, b.disabled), "bar-entry", i, b.rect, none, label)
	}
	t, tabs := e.mapTabsLayout()
	part("window", "map-tabs", -1, tabs.Rect, none, "")
	for i, r := range tabs.Tabs {
		part(chromeButtonClass(i == t.Active, false), "map-tab", i, r, none, c.decode(t.Captions[i]))
	}
	part("button", "map-tabs-close", -1, tabs.Close, none, "x")
	if s, ok := e.toolbarSpec(); ok {
		l := layoutPopup(s, e.font, e.bodyFont())
		part("window", "toolbar", -1, l.Rect, none, "")
		part("title", "toolbar-drag", -1, l.Title, none, "")
		part("button", "toolbar-dock", -1, l.Close, none, "x")
	} else {
		part("window", "toolbar", -1, e.toolbarRect(), none, "")
		part("control", "toolbar-grip", -1, e.toolbarGrip(), none, "")
	}
	for _, tool := range e.toolbarEntries() {
		write(chromeButtonClass(tool.shown, tool.closed), "tool-"+tool.key, -1, tool.rect, none, "", c.decode(e.toolWord(tool.toolButton)))
	}
	if len(c.search.Bar.Cells) > 0 {
		cell := c.search.Bar.Cells[0]
		part("window", "search-strip", -1, c.search.Rect, none, "")
		shown, hint := editFieldText(e.bodyFont(), *cell.Edit, cell.Rect)
		valueInk := ink.Value
		if hint {
			valueInk = ink.Hint
		}
		part("control", "search-field", -1, cell.Rect, valueInk, c.decode(shown))
	}
	if c.menu.Open {
		l := c.menu.listBox(e.font)
		part("window", "menu-list", -1, l.Rect.Union(l.Bar()).Inset(-2), none, "")
		for i, row := range c.menu.List.Rows() {
			class := "label"
			if row.Heading {
				class = "heading"
			}
			part(class, "menu-row", i, l.Row(i), row.Ink, c.decode(row.Text))
		}
	}
	write.tip(c.tip, c.decode, c.decode(c.tip.Text))
}

func chromeButtonClass(on, disabled bool) string {
	class := "button"
	if on {
		class += " pressed"
	}
	if disabled {
		class += " off"
	}
	return class
}
