package ui

import (
	"image"
	"image/color"
	"strings"
)

type chromeHitValue struct {
	Category string `json:"category"`
	Name     string `json:"name"`
}

func (c *mockupChrome) layoutChromeExtras(sample chromePreview) {
	e := c.editor
	_, c.tabLayout = e.mapTabStrip()
	if sample.Hover == "tab" && len(c.tabLayout.rects) > 0 {
		l := popupLayout{TabControls: &c.tabLayout}
		c.tip, _ = popupTipAt(c.bounds, e.font, c.body, popupSpec{}, l, c.tabLayout.rects[c.tabLayout.active].Min, true)
	}
	if len(sample.Status) > 0 {
		parts := make([]string, len(sample.Status))
		for i, part := range sample.Status {
			parts[i] = e.editorHostText(part)
		}
		c.statusSpec = statusStripSpec(e.statusRect(), statusSummary(parts), e.helpLines())
		c.status = layoutPopup(c.statusSpec, e.font, c.body)
		r := c.status.Rows[0]
		switch sample.Hover {
		case "status":
			c.tip, _ = popupTipAt(c.bounds, e.font, c.body, c.statusSpec, c.status, r.Label.Min, true)
		case "help":
			c.tip, _ = popupTipAt(c.bounds, e.font, c.body, c.statusSpec, c.status, r.Help.Min, true)
		}
	}
	var rows []popupRow
	for i, label := range sample.Players[:min(len(sample.Players), len(editorPlayerTints))] {
		rows = append(rows, playerSwatchRow(i, sample.FocusPlayer, e.editorHostText(label)))
	}
	if len(rows) > 0 {
		c.legendSpec = playerLegendSpec(e.mapArea(), rows, editorMeasure(c.body), 0)
		c.legend = layoutPopup(c.legendSpec, e.font, c.body)
	}
	if len(sample.Hits) > 0 && sample.Query != nil {
		rows := make([]PickerRow, 0, len(sample.Hits))
		for _, hit := range sample.Hits {
			label := searchItemLabel(hit.Category, e.editorHostText(hit.Name), e.word)
			rows = append(rows, menuPickerRow(menuItem{key: searchKey, label: label}))
		}
		list := NewPicker(rows)
		list.Select(sample.SelectedHit)
		c.menu = e.fitMenuList(e.menuDropdown(e.searchEntry(), false), list, true)
	}
	if sample.ListRows > 0 && c.menu.Open && sample.ListRows < c.menu.Rows {
		c.menu = c.menu.fitList(c.bounds, c.body, 0, sample.ListRows)
	}
}

func (c *mockupChrome) drawChromeExtras(dst *image.RGBA) {
	e := c.editor
	for _, w := range []tipWindow{{c.statusSpec, c.status}, {c.legendSpec, c.legend}} {
		if !w.layout.Rect.Empty() {
			drawPopup(dst, e.font, c.body, nil, w.spec, w.layout, editorPopupInk, image.Point{}, false)
		}
	}
}

func (c *mockupChrome) writeChromeExtras(writer pagePartWriter) {
	none := color.RGBA{}
	part := func(class, name string, row int, r image.Rectangle, ink color.RGBA, caption string) {
		writer(class, name, row, r, ink, c.decode(caption), "")
	}
	if len(c.tabLayout.rects) > 0 {
		part("window", "map-tabs", -1, c.editor.mapTabsRect(), none, "")
	}
	l := c.tabLayout
	for i, r := range l.rects {
		class := "button"
		if i == l.active {
			class += " pressed"
		}
		part(class, "map-tab", l.first+i, r, none, l.captions[i])
		writer("button", "close-map", l.first+i, l.closes[i], none, "x", c.decode(l.closeTip))
	}
	for i, box := range l.page {
		class := "button"
		if target := l.pageTarget(i); target < 0 || target >= l.count {
			class += " off"
		}
		writer(class, "tab-page", i, box, none, tabPageMarks[i], c.decode(l.pageTips[i]))
	}
	if len(c.status.Rows) > 0 {
		part("window", "status-strip", -1, c.status.Rect, none, "")
		r := c.status.Rows[0]
		part("label", "status", -1, r.Label, editorPaper, statusRowText(c.body, c.statusSpec.Rows[0].Label, r.Label))
		writer("button", "status-help", -1, r.Help, none, "?", c.decode(c.statusSpec.Rows[0].Help))
	}
	part("window", "legend", -1, c.legend.Rect, none, "")
	for _, r := range c.legend.Rows {
		row := c.legendSpec.Rows[r.Index]
		class := "label"
		if row.Swatch.focus {
			class += " ticked"
		}
		part(class, "legend-row", r.Index, r.Rect, none, "")
		part("swatch", "legend-swatch", r.Index, r.Swatch, row.Swatch.color, "")
		part("label", "legend-label", r.Index, r.Label, row.Ink, strings.Join(r.LabelLines, "\n"))
	}
	part("control", "legend-scroll", -1, c.legend.RowBar, none, "")
}
