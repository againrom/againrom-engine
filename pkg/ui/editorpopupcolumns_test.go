package ui

import (
	"image"
	"testing"
)

func TestPopupColumnsTabsAndRowsUseOneLayout(t *testing.T) {
	s := popupSpec{Width: 1200, Bounds: image.Rect(0, 0, 1280, 800), Columns: 3, MaxRows: 18, Tabs: []string{"Identity", "Equipment", "Bag", "Stats"}, Buttons: []string{"Apply", "Cancel"}}
	for c := 0; c < 3; c++ {
		for i := 0; i < 16-c; i++ {
			s.Rows = append(s.Rows, popupRow{Kind: popupChoice, Column: c, Label: "label", Value: "item"})
		}
	}
	l := layoutPopup(s, nil, nil)
	if popupRowCount(s) != 16 || len(l.Rows) != 45 || len(l.Tabs) != 4 || !l.Rect.In(s.Bounds) {
		t.Fatal("column layout", l.Rect, len(l.Rows))
	}
	for i, rr := range l.Rows {
		if !rr.Rect.In(l.Rect) || rr.Control.Empty() {
			t.Fatal("row outside popup", rr)
		}
		p := rr.Control.Min.Add(rr.Control.Size().Div(2))
		part, row := l.hit(p)
		if part != popupRowControl || row != rr.Index {
			t.Fatal("row hit differs from draw", part, row, rr.Index)
		}
		for _, other := range l.Rows[:i] {
			if rr.Rect.Overlaps(other.Rect) {
				t.Fatal("columns overlap", rr, other)
			}
		}
	}
	for i, tab := range l.Tabs {
		p := tab.Min.Add(tab.Size().Div(2))
		part, row := l.hit(p)
		if part != popupTab || row != i {
			t.Fatal("tab hit differs from draw", i, part, row)
		}
	}
	s.Top, s.MaxRows = 8, 5
	l = layoutPopup(s, nil, nil)
	if len(l.Rows) != 15 || l.RowBar.Empty() {
		t.Fatal("column scrolling does not share the visible rows")
	}
}

func TestPickerSearchSkipsUnavailableChoicesAndKeepsTheFullLabel(t *testing.T) {
	p := NewPicker([]PickerRow{{Text: "Empty NPC", Choosable: false}, {Text: "0x101: Long Sword", Choosable: true}, {Text: "Броня", Choosable: true}}).SetWindow(1)
	if p.Find("empty", nil) || !p.Find("sWoRd", nil) || p.Selection() != 1 || !p.Find("БРОНЯ", nil) || p.Selection() != 2 {
		t.Fatal("choice search", p.Selection())
	}
	if p.Rows()[1].Text != "0x101: Long Sword" {
		t.Fatal("search changed item labels")
	}
	p.MoveChoice(-1)
	p.MoveChoice(-1)
	if p.Selection() != 1 {
		t.Fatal("keyboard selected an unavailable choice")
	}
}
