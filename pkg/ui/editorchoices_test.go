package ui

import (
	"image"
	"reflect"
	"strings"
	"testing"
)

func TestChoiceListBrowsesGroupsAndLoadsOnlyTheirPictures(t *testing.T) {
	icon := image.NewRGBA(image.Rect(0, 0, 3, 4))
	var loaded []int64
	l := choiceList{back: "Back", field: InspectionField{
		Choices: []InspectionChoice{{0, "Remove"}, {-1, "Duplicate"}, {1, "Sword"}, {2, "Steel sword"}, {3, "Potion"}},
		Groups:  []InspectionChoiceGroup{{"Swords", "Iron", []int64{1}}, {"Swords", "Steel", []int64{2}}, {"Potions", "None", []int64{3}}},
		Pictures: func(choices []InspectionChoice) map[int64]*image.RGBA {
			out := map[int64]*image.RGBA{}
			for _, c := range choices {
				loaded = append(loaded, c.Value)
				out[c.Value] = icon
			}
			return out
		},
	}}
	rows := l.rows()
	if len(rows) != 4 || rows[0].Text != "Remove" || rows[1].Text != "Duplicate" || rows[2].Text != "Swords >" || len(loaded) != 0 {
		t.Fatalf("root rows=%+v loaded=%v", rows, loaded)
	}
	if _, navigate := l.take(2); !navigate {
		t.Fatal("kind was taken as an item")
	}
	rows = l.rows()
	if len(rows) != 5 || rows[2].Text != "Back" || rows[3].Text != "Iron >" || rows[4].Text != "Steel >" || len(loaded) != 0 {
		t.Fatalf("material rows=%+v loaded=%v", rows, loaded)
	}
	l.take(4)
	rows = l.rows()
	if len(rows) != 4 || rows[3].Text != "Steel sword" || rows[3].Icon != icon || !reflect.DeepEqual(loaded, []int64{2}) {
		t.Fatalf("leaf rows=%+v loaded=%v", rows, loaded)
	}
	if index, navigate := l.take(3); navigate || l.field.Choices[index].Value != 2 {
		t.Fatal("leaf changed the stored item value")
	}
	l.take(2)
	if rows = l.rows(); len(rows) != 5 || rows[3].Text != "Iron >" {
		t.Fatal("Back did not return to materials")
	}
}

func TestChoiceListBuildsFlatListsUnchanged(t *testing.T) {
	f := InspectionField{Choices: []InspectionChoice{{0, "One"}, {1, "Two"}}, Shut: []int64{1}, Icons: map[int64]*image.RGBA{0: image.NewRGBA(image.Rect(0, 0, 1, 1))}}
	l := choiceList{field: f}
	if got, want := l.rows(), choiceRows(f); !reflect.DeepEqual(got, want) {
		t.Fatalf("rows=%+v want=%+v", got, want)
	}
}

func TestChoiceRowDropdownOpensInsideTheWindow(t *testing.T) {
	e := NewMapEditor(nil, nil, nil)
	e.SetBodyFont(wordsFont("en"))
	e.Layout(640, 260)
	rows := make([]PickerRow, 40)
	for i := range rows {
		rows[i] = PickerRow{Text: "trigger " + strings.Repeat("x", i%7), Choosable: true}
	}
	e.openPrompt(editorPrompt{kind: promptAreaTrigger, area: &areaPrompt{box: image.Rect(1, 1, 4, 4), choice: NewPicker(rows), open: true}})
	d, ok := e.promptDropdown()
	if !ok || !d.Open {
		t.Fatal("the area prompt shows no open trigger list")
	}
	if want := e.rowDropdown(d.Rect, d.Text, d.List, d.Rows, true); d != want {
		t.Fatalf("the area prompt's drop-down %+v is not the choice row's %+v", d, want)
	}
	if box := d.listBox(e.bodyFont()); box.Rect.Max.Y > e.window.Y || box.Rect.Min.Y < 0 || d.Rows != 10 {
		t.Errorf("the open list of %d rows stands at %v in a window %v high", d.Rows, box.Rect, e.window.Y)
	}
}

// Every button of a question takes its width from the one measure: its
// caption's width with pad, at least popupButtonW, no screen setting a
// width of its own.
func TestQuestionButtonsTakeTheOneMeasure(t *testing.T) {
	e := NewMapEditor(nil, nil, nil)
	body := wordsFont("en")
	e.SetBodyFont(body)
	e.Layout(800, 600)
	s := e.questionSpec(question{title: "Leave", notes: []string{"The map has changes."}, buttons: []string{"OK", "Leave without saving the map", "Cancel"}})
	if s.ButtonW != 0 {
		t.Fatalf("the question sets its own button width %d", s.ButtonW)
	}
	want := popupButtonWidths(specWindow(s), body)
	l := layoutPopup(s, e.font, body)
	for i, b := range l.Buttons {
		if b.Dx() != want[i] || b.Dx() < popupButtonW || !b.In(l.Rect) {
			t.Errorf("button %q is %d wide at %v in %v; the measure gives %d", s.Buttons[i], b.Dx(), b, l.Rect, want[i])
		}
	}
	if want[1] <= want[0] || want[0] != popupButtonW {
		t.Errorf("the widths %v do not follow the captions", want)
	}
}

// The picture element stands a picture in its box as its fit says and shows
// only its part inside the clip.
func TestEditorPictureFitsItsBox(t *testing.T) {
	solid := func(w, h int) *image.RGBA {
		p := image.NewRGBA(image.Rect(0, 0, w, h))
		for i := 3; i < len(p.Pix); i += 4 {
			p.Pix[i-3], p.Pix[i] = 255, 255
		}
		return p
	}
	painted := func(dst *image.RGBA) image.Rectangle {
		var r image.Rectangle
		for y := 0; y < dst.Bounds().Dy(); y++ {
			for x := 0; x < dst.Bounds().Dx(); x++ {
				if dst.RGBAAt(x, y).A != 0 {
					r = r.Union(image.Rect(x, y, x+1, y+1))
				}
			}
		}
		return r
	}
	box := image.Rect(10, 10, 54, 54)
	for _, c := range []struct {
		name string
		p    editorPicture
		want image.Rectangle
	}{
		{"a small sprite grows by whole factors, centred", editorPicture{Pic: solid(10, 10), Box: box, Clip: box, Fit: pictureCentre}, image.Rect(12, 12, 52, 52)},
		{"a large picture shrinks to the box's top right", editorPicture{Pic: solid(88, 22), Box: box, Clip: box, Fit: pictureCorner}, image.Rect(10, 10, 54, 21)},
		{"a figure fills its box", editorPicture{Pic: solid(10, 20), Box: box, Clip: box, Fit: pictureFill}, box},
		{"the clip cuts it", editorPicture{Pic: solid(10, 20), Box: box, Clip: image.Rect(0, 0, 30, 30), Fit: pictureFill}, image.Rect(10, 10, 30, 30)},
		{"no picture draws nothing", editorPicture{Box: box, Clip: box}, image.Rectangle{}},
	} {
		dst := image.NewRGBA(image.Rect(0, 0, 64, 64))
		drawEditorPicture(dst, c.p)
		if got := painted(dst); got != c.want {
			t.Errorf("%s: painted %v, want %v", c.name, got, c.want)
		}
	}
}
