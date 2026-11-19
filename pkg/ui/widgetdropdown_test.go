package ui

import (
	"image"
	"image/color"
	"strings"
	"testing"
)

func dropdownFixture(open bool) dropdown {
	var rows []PickerRow
	for i := 0; i < 12; i++ {
		rows = append(rows, PickerRow{Text: string(rune('a' + i)), Choosable: true})
	}
	m := NewPicker(rows).SetWindow(5)
	return dropdown{Rect: image.Rect(10, 10, 210, 30), Text: "a", Open: open, List: m, Rows: 5,
		Fill: color.RGBA{48, 41, 31, 255}, Ink: color.RGBA{243, 229, 190, 255}}
}

func TestDropdownOpensScrollsAndChooses(t *testing.T) {
	f := panelFont()
	d := dropdownFixture(false)
	if h, _ := d.hit(f, image.Pt(100, 20)); h != dropdownToggle {
		t.Fatalf("press on the closed box: %v", h)
	}
	if h, _ := d.hit(f, image.Pt(100, 60)); h != dropdownMiss {
		t.Fatalf("press under a closed box hit %v", h)
	}
	d.Open = true
	l := d.listBox(f)
	if l.Rows != 5 || !d.Bounds(f).Eq(d.Bounds(f).Union(l.Bar())) {
		t.Fatalf("open list rows %d", l.Rows)
	}
	h, row := d.hit(f, l.Row(2).Min.Add(image.Pt(3, 1)))
	if h != dropdownRow || row != 2 {
		t.Fatalf("third visible row: %v %d", h, row)
	}
	d.List.Scroll(4)
	if _, row = d.hit(f, l.Row(0).Min.Add(image.Pt(3, 1))); row != 4 {
		t.Fatalf("after scrolling four rows the first visible row is %d", row)
	}
	if d.List.Selection() != 0 {
		t.Fatalf("scrolling moved the selection to %d", d.List.Selection())
	}
	d.List.Scroll(100)
	if top, n := d.List.Visible(); top != 7 || n != 5 {
		t.Fatalf("scroll clamps to the last page: top %d n %d", top, n)
	}
	if h, _ := d.hit(f, l.Bar().Min.Add(image.Pt(2, 2))); h != dropdownBar {
		t.Fatalf("bar press: %v", h)
	}
	var calls []widgetCall
	widgetRecord = func(c widgetCall) { calls = append(calls, c) }
	defer func() { widgetRecord = nil }()
	pic := image.NewRGBA(image.Rect(0, 0, 300, 200))
	drawDropdown(pic, f, nil, d, func(i, _ int) string { return d.List.Rows()[i].Text }, image.Point{}, false)
	kinds := map[widgetKind]int{}
	for _, c := range calls {
		kinds[c.kind]++
	}
	if kinds[widgetDropdown] != 1 || kinds[widgetListBox] != 1 || kinds[widgetPushButton] != 1 || kinds[widgetVScrollBar] != 1 {
		t.Fatalf("open drop-down built %v", kinds)
	}
}

// One press serves every drop-down: a row chooses, the box toggles, the bar
// only scrolls, and a press beside it calls Off and is not taken.
func TestDropdownPressChoosesTogglesAndLeaves(t *testing.T) {
	f := panelFont()
	d := dropdownFixture(true)
	l := d.listBox(f)
	var got []string
	act := dropdownActs{
		Choose: func(n int) { got = append(got, "choose", string(rune('0'+n))) },
		Toggle: func() { got = append(got, "toggle") },
		Off:    func() { got = append(got, "off") },
	}
	d.List.Scroll(2)
	rowPoint := l.Row(1).Min.Add(image.Pt(3, 1))
	for _, c := range []struct {
		p     image.Point
		taken bool
		want  string
	}{
		{rowPoint, true, "choose 3"},
		{image.Pt(100, 20), true, "toggle"},
		{l.Bar().Min.Add(image.Pt(2, 2)), true, ""},
		{image.Pt(280, 190), false, "off"},
	} {
		got = nil
		if taken := d.press(f, c.p, act); taken != c.taken || strings.Join(got, " ") != c.want {
			t.Errorf("press at %v: taken %v did %q, want %v %q", c.p, taken, got, c.taken, c.want)
		}
	}
	if row, ok := d.rowAt(f, rowPoint); !ok || row != 3 {
		t.Errorf("rowAt %d %v, want row 3", row, ok)
	}
	if !d.over(f, rowPoint) || d.over(f, image.Pt(280, 190)) {
		t.Error("over does not follow the open list")
	}
	act.Off = nil
	if d.press(f, image.Pt(280, 190), act) {
		t.Error("a press beside the drop-down was taken")
	}
	d.Open = false
	if _, ok := d.rowAt(f, rowPoint); ok || d.over(f, image.Pt(100, 20)) {
		t.Error("a closed drop-down answers for its list")
	}
}

func TestDropdownStepWrapsAsTheChoiceKeysDid(t *testing.T) {
	d := dropdownFixture(false)
	dropdownStep(d.List, -1)
	if d.List.Selection() != 11 {
		t.Fatalf("up from the first choice: %d", d.List.Selection())
	}
	dropdownStep(d.List, 1)
	if d.List.Selection() != 0 {
		t.Fatalf("down from the last choice: %d", d.List.Selection())
	}
	if dropdownStep(nil, 1) || dropdownStep(NewPicker(nil), 1) {
		t.Fatal("an empty drop-down stepped")
	}
}

func TestPickerSetRowsKeepsSelectionInside(t *testing.T) {
	d := dropdownFixture(false)
	d.List.Select(11)
	d.List.SetRows(d.List.Rows()[:3])
	if d.List.Selection() != 2 {
		t.Fatalf("selection %d after the list shrank to 3", d.List.Selection())
	}
	if top, n := d.List.Visible(); top != 0 || n != 3 {
		t.Fatalf("window %d+%d", top, n)
	}
}
