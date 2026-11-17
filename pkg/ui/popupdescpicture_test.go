package ui

import (
	"bytes"
	"image"
	"image/color"
	"strings"
	"testing"
)

// pictureDesc is a window with a picture row, a row of icons after its
// value, a window button whose caption takes a word and a fitted label
// column.
const pictureDesc = `{
  "window": "probe",
  "title": "editor.group.number",
  "titleArgs": ["group"],
  "width": 420,
  "fit": true,
  "rows": [
    {"kind": "text", "picture": "preview"},
    {"kind": "text", "label": "editor.group.members", "value": "count", "icons": "pictures"}
  ],
  "buttons": [{"word": "editor.view.switch", "args": ["view"]}],
  "sample": {
    "values": {"group": "2", "count": "2", "preview": "record:1", "pictures": "item:1 item:2"},
    "words": {"view": "editor.view.basic"}
  }
}`

var pictureRed = color.RGBA{R: 250, A: 255}

func redPicture(name string) *image.RGBA {
	if name != "record:1" {
		return nil
	}
	p := image.NewRGBA(image.Rect(0, 0, 8, 8))
	for i := 0; i < len(p.Pix); i += 4 {
		p.Pix[i], p.Pix[i+3] = pictureRed.R, pictureRed.A
	}
	return p
}

func TestDescriptionPictureRowShowsTheNamedPicture(t *testing.T) {
	m, err := newEditorMockup([]byte(pictureDesc), "en", image.Pt(800, 600), wordsFont("en"), wordsFont("en"), nil, mockupPerson{picture: redPicture})
	if err != nil {
		t.Fatal(err)
	}
	if m.plain.Title != "Group 2" || !m.plain.Fit || len(m.plain.Buttons) != 1 || m.plain.Buttons[0] != "View: Basic" {
		t.Fatalf("title %q fit %v buttons %q", m.plain.Title, m.plain.Fit, m.plain.Buttons)
	}
	if !pictureRow(m.paint.Rows[0]) || m.paint.Rows[0].Lines != 3 || len(m.paint.Rows[1].Icons) != 2 {
		t.Fatalf("picture row %v lines %d, %d icons", pictureRow(m.paint.Rows[0]), m.paint.Rows[0].Lines, len(m.paint.Rows[1].Icons))
	}
	box := rowPortraitBox(m.layout.Rows[0])
	p := m.Picture()
	inside, outside := 0, 0
	for y := p.Rect.Min.Y; y < p.Rect.Max.Y; y++ {
		for x := p.Rect.Min.X; x < p.Rect.Max.X; x++ {
			if p.RGBAAt(x, y) != pictureRed {
				continue
			}
			if image.Pt(x, y).In(box) {
				inside++
			} else {
				outside++
			}
		}
	}
	if inside == 0 || outside != 0 {
		t.Errorf("%d picture pixels in the label column %v, %d outside", inside, box, outside)
	}
	page := m.Page()
	for _, name := range []string{">record:1<", ">item:1<", ">item:2<"} {
		if !bytes.Contains(page, []byte(name)) {
			t.Errorf("the page names no picture %s", name)
		}
	}
	if n := bytes.Count(page, []byte(`data-part="picture"`)); n != 3 {
		t.Errorf("the page holds %d pictures, not 3", n)
	}
}

func TestDescriptionRefusesAPictureOnAControlRow(t *testing.T) {
	bad := strings.Replace(pictureDesc, `{"kind": "text", "picture"`, `{"kind": "edit", "picture"`, 1)
	if _, err := parsePopupDesc([]byte(bad)); err == nil {
		t.Error("an edit row holding a picture was read")
	}
}
