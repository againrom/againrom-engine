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
	y := e.barHeight() + 3
	return barEntry{key: searchKey, label: e.word("editor.bar.search"), menu: true, field: true,
		rect: image.Rect(4, y, 4+searchFieldW, y+e.barRowHeight()-6)}
}

// searchRows is the most hits the open list shows at once.
const searchRows = 16

// barSearch is the search field's text and whether it takes the typed text.
type barSearch struct {
	text   string
	typing bool
}

// searchTyping reports that the search field takes the typed text.
func (e *MapEditor) searchTyping() bool {
	return e.search.typing && e.doc != nil && !e.prompt.open() && !e.objectTyping() && e.pathDialog.pending == nil
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
	e.search.typing = true
	if m, _ := e.inputModel(); m != nil {
		m.selectAll()
	}
	e.searchFollow()
	return nil
}

// searchLeave ends the field's typing and closes its list; clear also
// empties its text, as Esc does.
func (e *MapEditor) searchLeave(clear bool) {
	e.search.typing = false
	if clear {
		e.search.text = ""
	}
	if e.menu == searchKey {
		e.menu = ""
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
	row := popupRow{Kind: popupEdit, Value: e.editorHostText(e.search.text), Hint: e.word("editor.bar.search")}
	if e.searchTyping() {
		row.Focus = true
		row.Edit = e.inputView(editorTextSpan{0, len([]rune(e.search.text))}, false)
	}
	return row
}

// searchInputAt is the search field's text view under p: a press there makes
// the field take the typed text; while it does, a drag keeps it.
func (e *MapEditor) searchInputAt(p image.Point, press bool) (*editorTextView, image.Rectangle, bool) {
	if !e.searchTyping() {
		return nil, image.Rectangle{}, false
	}
	box := e.searchEntry().rect
	if press {
		if !p.In(box) || e.doc == nil || e.prompt.open() {
			return nil, image.Rectangle{}, false
		}
		if hit := e.chromeHit(p); hit != popBar && hit != popMenu {
			return nil, image.Rectangle{}, false
		}
		if !e.search.typing {
			_ = e.keySearch(searchKey)
		}
	} else if !e.searchTyping() {
		return nil, image.Rectangle{}, false
	}
	v := e.inputView(editorTextSpan{0, len([]rune(e.search.text))}, false)
	return v, box, v != nil
}

// searchPressOff ends the field's typing on a press at p outside the field
// and its open list.
func (e *MapEditor) searchPressOff(p image.Point) {
	if !e.search.typing {
		return
	}
	if p.In(e.searchEntry().rect) {
		return
	}
	if b, ok := e.openMenuEntry(); ok && b.key == searchKey && e.menuDropdown(b, true).Covers(e.font, p) {
		return
	}
	e.searchLeave(false)
}
