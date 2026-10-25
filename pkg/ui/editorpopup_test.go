package ui

import (
	"image"
	"image/color"
	"testing"
)

func popupFixture() popupSpec {
	rows := []popupRow{
		{Kind: popupHeading, Label: "Unit"},
		{Kind: popupText, Label: "Cell", Value: "10,10"},
		{Kind: popupEdit, Label: "Hit points", Value: "-1", Focus: true},
		{Kind: popupChoice, Label: "Owner", Value: "1: Self"},
		{Kind: popupPick, Label: "Target", Value: "4,4"},
		{Kind: popupLink, Label: "Group", Value: "Group 2"},
	}
	return popupSpec{Title: "Unit #0", At: image.Pt(600, 40), Width: 400, Bounds: image.Rect(0, 26, 1024, 742),
		Rows: rows, Notes: []string{"refused"}, Buttons: []string{"Cancel", "Apply"}, PickLabel: "Pick"}
}

func TestPopupLayoutIsThePaintAndTheHitTest(t *testing.T) {
	f := panelFont()
	s := popupFixture()
	l := layoutPopup(s, f, f)
	if len(l.Rows) != len(s.Rows) || len(l.Buttons) != 2 || len(l.Notes) != 1 {
		t.Fatalf("layout rows %d buttons %d notes %d", len(l.Rows), len(l.Buttons), len(l.Notes))
	}
	if !l.Rect.In(s.Bounds) {
		t.Fatalf("popup %v leaves the window %v", l.Rect, s.Bounds)
	}
	center := func(r image.Rectangle) image.Point { return r.Min.Add(r.Size().Div(2)) }
	cases := []struct {
		p     image.Point
		part  popupPart
		index int
	}{
		{center(l.Close), popupClose, 0},
		{l.Title.Min.Add(image.Pt(10, 4)), popupTitle, 0},
		{center(l.Rows[2].Control), popupRowControl, 2},
		{center(l.Rows[3].Label), popupRowLine, 3},
		{center(l.Rows[4].Button), popupRowButton, 4},
		{center(l.Buttons[1]), popupButton, 1},
		{l.Rect.Min.Sub(image.Pt(1, 1)), popupMiss, 0},
	}
	for _, c := range cases {
		if part, i := l.hit(c.p); part != c.part || i != c.index {
			t.Errorf("hit %v = %v %d, want %v %d", c.p, part, i, c.part, c.index)
		}
	}
	var calls []widgetCall
	widgetRecord = func(c widgetCall) { calls = append(calls, c) }
	defer func() { widgetRecord = nil }()
	pic := image.NewRGBA(image.Rect(0, 0, 1024, 768))
	drawPopup(pic, f, f, nil, s, l, popupInk{Ground: editorPanel, Edge: editorGold, TitleGround: editorInk, Title: editorGold, Label: editorPaper, Value: editorLight}, image.Point{}, false)
	kinds := map[widgetKind]int{}
	for _, c := range calls {
		kinds[c.kind]++
		if !c.rect.In(l.Rect) {
			t.Errorf("widget %v at %v outside the popup %v", c.kind, c.rect, l.Rect)
		}
	}
	// frame twice and a heading band, close + pick + two buttons + drop-down arrow, edit x2 (+ the drop-down's well)
	if kinds[widgetFrame] != 3 || kinds[widgetPushButton] != 5 || kinds[widgetEdit] != 3 || kinds[widgetDropdown] != 1 {
		t.Fatalf("popup built %v", kinds)
	}
	if c := pic.RGBAAt(l.Rect.Min.X+1, l.Rect.Max.Y-2); c != (color.RGBA{0x30, 0x29, 0x1f, 255}) {
		t.Fatalf("popup ground %v", c)
	}
}

func TestPopupRowsScrollAndTheListSits(t *testing.T) {
	f := panelFont()
	s := popupFixture()
	s.Top, s.MaxRows = 2, 2
	var rows []PickerRow
	for i := 0; i < 30; i++ {
		rows = append(rows, PickerRow{Text: "row", Choosable: true})
	}
	s.List, s.ListRows = NewPicker(rows).SetWindow(8), 8
	l := layoutPopup(s, f, f)
	if len(l.Rows) != 2 || l.Rows[0].Index != 2 || !l.HasList || l.List.Rows != 8 {
		t.Fatalf("rows %v list %v", l.Rows, l.HasList)
	}
	if part, i := l.hit(l.List.Row(3).Min.Add(image.Pt(2, 2))); part != popupListRow || i != 3 {
		t.Fatalf("list row hit %v %d", part, i)
	}
	if part, _ := l.hit(l.List.Bar().Min.Add(image.Pt(2, 2))); part != popupListBar {
		t.Fatalf("bar hit %v", part)
	}
}
