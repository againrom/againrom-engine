package ui

import (
	"image/color"
	"slices"
	"strconv"
	"strings"
)

type choiceListEntry struct {
	index int
	path  []string
	label string
}

type choiceList struct {
	field   InspectionField
	path    []string
	back    string
	plain   bool
	entries []choiceListEntry
}

func (l *choiceList) rows() []PickerRow {
	l.entries = nil
	grouped := map[int64]bool{}
	for _, g := range l.field.Groups {
		for _, v := range g.Values {
			grouped[v] = true
		}
	}
	for i, c := range l.field.Choices {
		if !grouped[c.Value] {
			l.entries = append(l.entries, choiceListEntry{index: i})
		}
	}
	if len(l.path) > 0 {
		l.entries = append(l.entries, choiceListEntry{index: -1, path: slices.Clone(l.path[:len(l.path)-1]), label: l.back})
	}
	seen := map[string]bool{}
	for _, g := range l.field.Groups {
		switch len(l.path) {
		case 0:
			if !seen[g.Block] {
				l.entries = append(l.entries, choiceListEntry{index: -1, path: []string{g.Block}, label: g.Block + " >"})
				seen[g.Block] = true
			}
		case 1:
			if g.Block == l.path[0] && !seen[g.Head] {
				l.entries = append(l.entries, choiceListEntry{index: -1, path: []string{g.Block, g.Head}, label: g.Head + " >"})
				seen[g.Head] = true
			}
		case 2:
			if g.Block == l.path[0] && g.Head == l.path[1] {
				for i, c := range l.field.Choices {
					if slices.Contains(g.Values, c.Value) {
						l.entries = append(l.entries, choiceListEntry{index: i})
					}
				}
			}
		}
	}
	f := l.field
	f.Choices = nil
	var pictured []InspectionChoice
	for _, entry := range l.entries {
		c := InspectionChoice{Label: entry.label}
		if entry.index >= 0 {
			c = l.field.Choices[entry.index]
			if grouped[c.Value] {
				pictured = append(pictured, c)
			}
		}
		f.Choices = append(f.Choices, c)
	}
	if f.Pictures != nil && len(pictured) > 0 {
		f.Icons = f.Pictures(pictured)
	}
	rows := choiceRows(f)
	for i, entry := range l.entries {
		if l.plain && len(l.field.Groups) == 0 {
			rows[i].Ink = color.RGBA{}
		}
		if entry.index < 0 {
			rows[i].Choosable, rows[i].Icon = true, nil
		}
	}
	return rows
}

func (l *choiceList) index(row int) int {
	if row < 0 || row >= len(l.entries) {
		return -1
	}
	return l.entries[row].index
}

func (l *choiceList) take(row int) (int, bool) {
	if row < 0 || row >= len(l.entries) {
		return -1, false
	}
	entry := l.entries[row]
	if entry.index < 0 {
		l.path = entry.path
		return -1, true
	}
	return entry.index, false
}

// openChoiceList is the editor's one open list of field f's choices, the
// object window's and a field prompt's: its entries and its picker over the
// rows window gives for the row count, the choice whose value text holds
// selected.
func (e *MapEditor) openChoiceList(f InspectionField, text string, window func(rows int) int) (choiceList, *Picker) {
	l := choiceList{field: f, back: e.word("editor.button.back")}
	rows := l.rows()
	sel := 0
	for k, entry := range l.entries {
		if entry.index >= 0 && strconv.FormatInt(f.Choices[entry.index].Value, 10) == strings.TrimSpace(text) {
			sel = k
		}
	}
	n := window(len(rows))
	p := NewPicker(rows)
	if n > pickerVisible {
		p.SetRowLimit(n)
	}
	p.SetWindow(n)
	p.Select(sel)
	return l, p
}
