package ui

import (
	"bytes"
	"image"
	"testing"

	"againrom/pkg/render/text"
)

// heavyFont is panelFont with a second painted pixel beside each glyph's
// first: a face that draws differently from the plain one.
func heavyFont() *text.Font {
	f := panelFont()
	for k := range f.Glyphs {
		g := &f.Glyphs[k]
		for i, p := range g.Pixels {
			if p.Painted && i%g.Width+1 < g.Width {
				g.Pixels[i+1] = text.Pixel{Level: text.MaxLevel, Painted: true}
				break
			}
		}
	}
	return f
}

// A list box draws a row marked Heading in its heading face and every other
// row in the list's own.
func TestListBoxDrawsHeadingRowsInTheHeadingFace(t *testing.T) {
	body, heavy := panelFont(), heavyFont()
	m := NewPicker([]PickerRow{{Text: "Triggers", Heading: true}, {Text: "Trigger one", Choosable: true}})
	box := newListBox(image.Rect(10, 10, 210, 10), 2, body)
	render := func(face *text.Font) *image.RGBA {
		l := box
		l.HeadingFace = face
		dst := image.NewRGBA(image.Rect(0, 0, 260, 120))
		drawListBox(dst, body, nil, l, m, func(row, _ int) string { return m.Rows()[row].Text }, image.Point{}, false)
		return dst
	}
	plain, bold := render(nil), render(heavy)
	rowPix := func(pic *image.RGBA, n int) []byte {
		r := box.Row(n).Inset(1)
		var out []byte
		for y := r.Min.Y; y < r.Max.Y; y++ {
			out = append(out, pic.Pix[pic.PixOffset(r.Min.X, y):pic.PixOffset(r.Max.X, y)]...)
		}
		return out
	}
	if bytes.Equal(rowPix(plain, 0), rowPix(bold, 0)) {
		t.Error("the heading row looks the same with the heading face")
	}
	if !bytes.Equal(rowPix(plain, 1), rowPix(bold, 1)) {
		t.Error("the row under the heading changed with the heading face")
	}
}

// The tree's Scripts branch marks its four group headings and no member
// row.
func TestScriptsListMarksItsGroupHeadings(t *testing.T) {
	e, _ := newModesEditor(t, "full")
	e.showRecords(editorScriptsFilter)
	kinds := e.treeRowsNow()
	rows := e.tree.model.Rows()
	headings := 0
	for i, k := range kinds {
		if k.branch != "" || k.member && k.of != "scripts" {
			continue
		}
		r := rows[i]
		if r.Heading != !r.Choosable || r.Heading == k.member {
			t.Errorf("row %d %q: Heading %v, Choosable %v", i, r.Text, r.Heading, r.Choosable)
		}
		if r.Heading {
			headings++
		}
	}
	if headings != 4 {
		t.Errorf("%d heading rows, want 4 (triggers, variables, conditions, actions)", headings)
	}
}
