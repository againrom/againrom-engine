package ui

import (
	"image"
	"reflect"
	"regexp"
	"strconv"
	"testing"
)

var propertyPageBox = regexp.MustCompile(`data-part="([a-z-]+)"(?: data-row="(\d+)")?(?: title="[^"]*")? data-box="(-?\d+) (-?\d+) (\d+) (\d+)"`)

// checkWindowProperties holds a window spec to the four layout properties at
// one size: every element inside the window, every label whole (at the
// largest size), hit returning each element at its rectangle, and the HTML
// page's rectangles equal to the layout's (a card and its figure aside).
// A row placed on a card is not held to the label check.
func checkWindowProperties(t *testing.T, e *MapEditor, s popupSpec, size image.Point) {
	t.Helper()
	body := e.bodyFont()
	l := layoutPopup(s, e.font, body)
	window := image.Rectangle{Max: size}
	if !l.Rect.In(window) {
		t.Errorf("%v: the window stands at %v, outside the screen", size, l.Rect)
	}
	if len(l.Rows) == 0 {
		t.Fatalf("%v: no rows are laid out", size)
	}
	for _, r := range l.Rows {
		if !r.Rect.In(l.Rect) {
			t.Errorf("%v: row %d at %v lies outside the window", size, r.Index, r.Rect)
		}
		if size.X >= 1920 && s.Rows[r.Index].Label != "" && !s.Rows[r.Index].placed() {
			room := r.Label
			if s.Rows[r.Index].Kind == popupHeading || room.Empty() {
				room = r.Rect
			}
			if w := body.Advance(s.Rows[r.Index].Label); w > room.Dx() {
				t.Errorf("%v: row %d label %q is %d wide in a %d wide place", size, r.Index, s.Rows[r.Index].Label, w, room.Dx())
			}
		}
		centre := r.Rect.Min.Add(r.Rect.Size().Div(2))
		if part, index := l.hit(centre); index != r.Index || part == popupMiss || part == popupBody {
			t.Errorf("%v: hit at the middle of row %d gives part %d, index %d", size, r.Index, part, index)
		}
	}
	for i, b := range l.Buttons {
		if !b.In(l.Rect) {
			t.Errorf("%v: button %d at %v lies outside the window", size, i, b)
		}
		if part, index := l.hit(b.Min.Add(b.Size().Div(2))); part != popupButton || index != i {
			t.Errorf("%v: hit at the middle of button %d gives part %d, index %d", size, i, part, index)
		}
	}
	for i, c := range l.GridCells {
		if !c.In(l.Rect) {
			t.Errorf("%v: grid cell %d at %v lies outside the window", size, i, c)
		}
		if l.GridInert[i] {
			continue
		}
		if part, index := l.hit(c.Min.Add(c.Size().Div(2))); part != popupGridPart || index != l.gridIndex(i) {
			t.Errorf("%v: hit at the middle of grid cell %d gives part %d, index %d", size, i, part, index)
		}
	}
	pictures := pagePictures(body, l, s.Rows, nil)
	page := string(popupPage(popupDesc{}, specWindow(s), l, image.Rectangle{}, pictures, "en", size))
	var got []pagePart
	for _, p := range propertyPageBox.FindAllStringSubmatch(page, -1) {
		n := func(i int) int { v, _ := strconv.Atoi(p[i]); return v }
		if p[1] != "card" && p[1] != "figure" {
			got = append(got, pagePart{p[1], image.Rect(n(3), n(4), n(3)+n(5), n(4)+n(6))})
		}
	}
	want := layoutParts(s, l)
	for _, p := range pictures {
		want = append(want, pagePart{"picture", p.box})
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("%v: page parts\n%v\nlayout\n%v", size, got, want)
	}
}

// The variable window takes its width from its own text, and holds the four
// layout properties at the three sizes with links enough to page.
func TestEditorVariableWindowFitsItsWindowAtThreeSizes(t *testing.T) {
	v := scriptVariableView{register: 1, constant: 3, shared: -1, addressed: true, label: "Gold"}
	for id := 0; id < 15; id++ {
		v.writers = append(v.writers, id)
		v.readers = append(v.readers, 15+id)
	}
	for _, size := range []image.Point{{640, 480}, {800, 600}, {1920, 1080}} {
		e := newMapEditorFixture(t)
		e.Layout(size.X, size.Y)
		s, _, _ := e.variableSpec(v)
		if s.Width > size.X-16 {
			t.Errorf("%v: the variable window asks for width %d, wider than the window allows", size, s.Width)
		}
		checkWindowProperties(t, e, s, size)
	}
}
