package ui

import (
	"bytes"
	"image"
	"strings"
	"testing"
)

// listDesc is a brush window: a picture row of heads, a grid with a ticked
// cell, a list whose chosen row's picture stands beside it, and a switch
// button.
const listDesc = `{
  "window": "probe",
  "title": "editor.bar.objects",
  "width": 420,
  "labelWidth": 90,
  "rows": [{"kind": "text", "label": "editor.brush.size", "value": "size"}],
  "grid": {"each": "modes", "picture": "icon", "tip": "name", "ticked": "ticked", "side": 34, "rows": 1,
    "rowHeads": ["editor.brush.mode"]},
  "list": {"each": "objects", "bind": "name", "chosen": "chosen",
    "aside": {"picture": "chosen.picture", "width": 40}},
  "switch": {"word": "editor.place.ruined", "on": "ruined"},
  "sample": {
    "values": {"size": "2", "chosen.picture": "record:1"},
    "lists": {
      "modes": [
        {"words": {"name": "editor.brush.mode.step"}},
        {"words": {"name": "editor.brush.mode.level"}, "flags": ["ticked"]}
      ],
      "objects": [
        {"words": {"name": "editor.brush.noobject"}},
        {"values": {"name": "Rock 1"}, "flags": ["chosen"]},
        {"values": {"name": "Rock 2"}}
      ]
    }
  }
}`

func TestDescriptionListStandsItsPictureBesideIt(t *testing.T) {
	m, err := newEditorMockup([]byte(listDesc), "en", image.Pt(800, 600), wordsFont("en"), wordsFont("en"), nil, mockupPerson{picture: redPicture})
	if err != nil {
		t.Fatal(err)
	}
	g := m.plain.Grid
	if g == nil || len(g.Cells) != 2 || g.Cells[0].Ticked || !g.Cells[1].Ticked || len(g.RowHeads) != 1 || g.RowHeads[0] != "Mode" || g.Cols != 2 {
		t.Fatalf("grid %+v", g)
	}
	if m.plain.List == nil || m.plain.List.Len() != 3 || m.plain.List.Selection() != 1 || m.plain.ListRows != 3 {
		t.Fatalf("list rows %d shown %d", m.plain.List.Len(), m.plain.ListRows)
	}
	if !strings.HasSuffix(m.plain.SwitchButton, "Ruined") {
		t.Errorf("switch button %q", m.plain.SwitchButton)
	}
	box := m.layout.Aside
	if box.Empty() || m.plain.ListPicture == nil {
		t.Fatalf("aside %v picture %v", box, m.plain.ListPicture != nil)
	}
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
		t.Errorf("%d picture pixels beside the list %v, %d outside", inside, box, outside)
	}
	page := m.Page()
	for part, want := range map[string]int{`data-part="list-row"`: 3, `data-part="switch-button"`: 1, ` ticked"`: 1, `>record:1<`: 1} {
		if n := bytes.Count(page, []byte(part)); n != want {
			t.Errorf("the page holds %d %s, not %d", n, part, want)
		}
	}
}
