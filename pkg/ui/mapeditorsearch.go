package ui

import (
	"fmt"
	"image"
	"strings"
)

// File's Search row and Ctrl+F open the same field below File. Its hits use
// the bar's drop-down and choose the same member as the tree.

// searchKey names the field's drop-down, key row and text input.
const searchKey = "search"

const searchFieldW = 220

func (e *MapEditor) searchEntry() barEntry {
	l := e.searchLayout()
	entry := barEntry{key: searchKey, menu: true, field: true}
	if len(l.Bar.Cells) > 0 {
		entry.rect = l.Bar.Cells[0].Rect
	}
	return entry
}

func (e *MapEditor) searchLayout() popupLayout {
	top := e.toolbarRect().Max.Y
	bar := popupBar{Strip: true, Pad: popupPad, Height: fontHeight(e.bodyFont(), 15) + popupRowExtra}
	for _, cell := range e.topBarLayout().Bar.Cells {
		if barKeys[cell.Index].key == "file" {
			bar.Inset = cell.Rect.Min.X
			break
		}
	}
	row := e.searchFieldRow()
	bar.Groups = []popupBarGroup{{Entries: []popupBarEntry{{Edit: &row, PreferredWidth: searchFieldW}}}}
	bounds := image.Rect(0, top, e.window.X, max(top, e.mapBottom(top)-mapLeastHeight))
	return layoutPopup(popupSpec{Bar: &bar, Bounds: bounds}, e.font, e.bodyFont())
}

func (e *MapEditor) chromeContentTop() int {
	if e.searchRowShown() {
		return e.searchLayout().Rect.Max.Y
	}
	return e.toolbarRect().Max.Y
}

func (e *MapEditor) drawSearchBar(pic *image.RGBA) {
	if !e.searchRowShown() {
		return
	}
	l := e.searchLayout()
	if len(l.Bar.Cells) > 0 {
		row := e.searchRow()
		l.Bar.Cells[0].Edit = &row
		e.noteWindow(l.Rect)
		e.drawBar(pic, l)
	}
}

// searchRows is the most hits the open list shows at once.
const searchRows = 16

type barSearch struct {
	text           string
	typing         bool
	closeOnRelease bool
}

func (e *MapEditor) searchRowShown() bool {
	return e.search.closeOnRelease || e.searchTyping()
}

func (e *MapEditor) searchTyping() bool {
	return e.search.typing && !e.search.closeOnRelease && e.doc != nil && !e.prompt.open() && !e.objectTyping() && e.pathDialog.pending == nil
}

// searchHits are the members the search text finds, category by category in
// the bar's order, each in its tree order; none for a blank text.
func (e *MapEditor) searchHits() []treeMember {
	if e.doc == nil || strings.TrimSpace(e.search.text) == "" {
		return nil
	}
	var out []treeMember
	for _, c := range editorCategories {
		for _, m := range e.treeMatches(c, e.search.text) {
			if m.row.member {
				out = append(out, m)
			}
		}
	}
	return out
}

// searchItems are the open list's rows: each hit's category word, then its
// tree row's text.
func (e *MapEditor) searchItems() []menuItem {
	var out []menuItem
	for _, m := range e.searchHits() {
		label := m.text
		if c, ok := categoryByKey(m.row.of); ok {
			label = fmt.Sprintf("%s: %s", e.word(c.word), m.text)
		}
		out = append(out, menuItem{key: searchKey, label: label})
	}
	return out
}

// searchFollow keeps the field open while typing, with the first hit
// highlighted when there are hits.
func (e *MapEditor) searchFollow() {
	switch {
	case e.searchTyping():
		e.menu, e.menuModel = searchKey, nil
	case e.menu == searchKey:
		e.menu = ""
	}
}

// keySearch is Ctrl+F: the search field takes the typed text, its text
// selected so typing replaces it, and its hits listed again.
func (e *MapEditor) keySearch(string) error {
	if e.doc == nil {
		return nil
	}
	e.endFieldTyping()
	e.menu = ""
	e.tree.typing = false
	e.search.typing, e.search.closeOnRelease = true, false
	if m, _ := e.inputModel(); m != nil {
		m.SelectAll()
	}
	e.searchFollow()
	e.relayout()
	return nil
}

// searchLeave ends the field's typing and closes its list; clear also
// empties its text, as Esc does.
func (e *MapEditor) searchLeave(clear bool) {
	showing := e.searchRowShown()
	e.search.typing, e.search.closeOnRelease = false, false
	if clear {
		e.search.text = ""
	}
	if e.menu == searchKey {
		e.menu = ""
	}
	if showing {
		e.relayout()
	}
}

// searchModel is the open hit list's model, nil while the list is closed.
func (e *MapEditor) searchModel() *Picker {
	if b, ok := e.openMenuEntry(); ok && b.key == searchKey {
		e.menuDropdown(b, true)
		return e.menuModel
	}
	return nil
}

// chooseHit chooses hit n: the field leaves typing, its text kept, and the
// member is chosen as a press on its tree row chooses it, a group by
// chooseGroup and any other member by chooseListed.
func (e *MapEditor) chooseHit(n int) {
	hits := e.searchHits()
	if n < 0 || n >= len(hits) {
		return
	}
	e.searchLeave(false)
	if r := hits[n].row; r.group.set {
		e.chooseGroup(r.group, true)
	} else {
		e.chooseListed(r.record)
	}
}

// searchRow is the search field as a text field row: its text, its hint
// while empty and, while it takes the typed text, its text view.
func (e *MapEditor) searchRow() popupRow {
	row := e.searchFieldRow()
	if e.searchTyping() {
		row.Focus = true
		row.Edit = e.inputView(editorTextSpan{0, len([]rune(e.search.text))}, false)
	}
	return row
}

func (e *MapEditor) searchFieldRow() popupRow {
	return popupRow{Kind: popupEdit, Value: e.editorHostText(e.search.text), Hint: e.word("editor.bar.search"), FitEdit: true}
}

// searchInputAt is the search field's text view under p: a press there makes
// the field take the typed text; while it does, a drag keeps it.
func (e *MapEditor) searchInputAt(p image.Point, press bool) (*editorTextView, image.Rectangle, bool) {
	if !e.searchTyping() {
		return nil, image.Rectangle{}, false
	}
	l := e.searchLayout()
	if len(l.Bar.Cells) == 0 {
		return nil, image.Rectangle{}, false
	}
	box := l.Bar.Cells[0].Rect
	if press {
		if part, _ := l.hit(p); part != popupRowControl || e.chromeHit(p) != popMenu {
			return nil, image.Rectangle{}, false
		}
	}
	v := e.inputView(editorTextSpan{0, len([]rune(e.search.text))}, false)
	return v, box, v != nil
}

// searchPressOff yields input while the row holds its space through release.
func (e *MapEditor) searchPressOff(target editorPopupID) {
	if !e.search.typing || target == popMenu && e.menu == searchKey {
		return
	}
	if !e.searchTyping() {
		e.searchLeave(false)
		return
	}
	e.search.closeOnRelease = true
	if e.menu == searchKey {
		e.menu = ""
	}
}
