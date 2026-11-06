package ui

import (
	"bytes"
	"image"
	"strings"
	"testing"
)

// A block of the plain view that opens and closes is a heading that takes
// the tree's fold state: its label is the block's word alone, its Fold
// follows the block, and the mark PopupLines writes and the painter draws
// is rowFoldMarks', so a change of the tree's marks changes the block's.
func TestMapEditorPlainBlocksTakeTheTreeFold(t *testing.T) {
	marks := rowFoldMarks
	defer func() { rowFoldMarks = marks }()
	for _, record := range []int{viewsUnit, viewsStructure, viewsSack} {
		e := wideEditor(t, "en", 520, "plain", nil)
		e.Select(record, false)
		s, _, _ := e.objectSpec()
		blocks := 0
		for n, row := range s.Rows {
			id, ok := e.obj.rowBlock[n]
			if !ok {
				continue
			}
			blocks++
			b := plainBlock{id: id, fold: true}
			want := FoldClosed
			if e.blockOpen(b) {
				want = FoldOpen
			}
			if row.Kind != popupHeading || row.Fold != want {
				t.Errorf("%s block %s: kind %d fold %d, want a heading with fold %d", e.doc.Records[record].Kind, id, row.Kind, row.Fold, want)
			}
			for _, mark := range []string{"+", "-"} {
				if strings.HasPrefix(row.Label, mark) {
					t.Errorf("%s block %s: label %q carries a mark", e.doc.Records[record].Kind, id, row.Label)
				}
			}
			for _, m := range [][3]string{{"", "-", "+"}, {"", "v", ">"}} {
				rowFoldMarks = m
				if line := e.PopupLines()[n]; line != rowFoldMarks[want]+" "+row.Label {
					t.Errorf("%s block %s: line %q under marks %q", e.doc.Records[record].Kind, id, line, m)
				}
			}
			rowFoldMarks = marks
			e.toggleBlock(id)
			s2, _, _ := e.objectSpec()
			if got := s2.Rows[n].Fold; got == want {
				t.Errorf("%s block %s: fold stays %d after a press", e.doc.Records[record].Kind, id, got)
			}
			e.toggleBlock(id)
		}
		// A unit's basic view is the card and the rows over it: nothing
		// opens and closes.
		shape := blocks > 0
		if record == viewsUnit {
			shape = blocks == 0
		}
		if !shape {
			t.Errorf("%s: %d blocks that open and close", e.doc.Records[record].Kind, blocks)
		}
	}
	// The painter draws the mark from rowFoldMarks: other marks, other
	// pixels.
	s := popupSpec{Title: "t", At: image.Pt(0, 0), Width: 300, Rows: []popupRow{blockRow("Details", false)}}
	paint := func() []byte {
		dst := image.NewRGBA(image.Rect(0, 0, 320, 120))
		drawPopup(dst, wordsFont("en"), wordsFont("en"), nil, s, layoutPopup(s, wordsFont("en"), wordsFont("en")), popupInk{Ground: editorPaper, Edge: editorWarning, TitleGround: editorPaper, Title: editorWarning, Label: editorWarning, Value: editorWarning}, image.Point{}, false)
		return dst.Pix
	}
	before := paint()
	rowFoldMarks = [...]string{FoldOpen: "-", FoldClosed: "#"}
	if bytes.Equal(before, paint()) {
		t.Error("the block's mark is not drawn from rowFoldMarks")
	}
}
