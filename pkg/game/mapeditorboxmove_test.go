package game

import (
	"bytes"
	"encoding/binary"
	"image"
	"strings"
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/formats/alm"
)

// boxMoveMap is a 40x40 map of land and road columns whose heights rise
// along both axes, with objects at three cells of the box (10,10)-(16,16)
// and one outside it, a unit inside the box and one outside it.
func boxMoveMap(t *testing.T) []byte {
	t.Helper()
	tiles := make([]byte, 2*40*40)
	overlay := make([]byte, 40*40)
	altitudes := make([]byte, 40*40)
	for i := range 40 * 40 {
		w := uint16(forestLand)
		if i%40%3 == 0 {
			w = forestRoad
		}
		binary.LittleEndian.PutUint16(tiles[2*i:], w)
		altitudes[i] = byte(40 + i%40 + i/40)
	}
	for _, c := range []image.Point{{11, 11}, {13, 12}, {15, 15}, {30, 30}} {
		overlay[c.Y*40+c.X] = forestOakish
	}
	overlay[12*40+14] = forestOther
	o := synth.ALMOptions{Width: 40, Height: 40, Type1Payload: tiles, Altitudes: altitudes, Overlay: overlay, Type7Payload: make([]byte, 12)}
	o.Units = []synth.ALMUnit{
		{X: 12*256 + 100, Y: 14*256 + 60, ClassID: 40, ClassSubID: 2},
		{X: 30*256 + 128, Y: 20*256 + 128, ClassID: 40, ClassSubID: 2},
	}
	return built(t, o, nil, nil).Bytes()
}

// A selection box drawn over trees and a unit, then dragged from a tree
// inside it, moves the trees and the unit by the drag's offset as one
// undo step: the cells the box left hold no object, every other object
// stays, and the tiles and heights stay where they were.
func TestMapEditorBoxDragMovesItsObjectsAndRecords(t *testing.T) {
	e, _ := openGoalMap(t, boxMoveMap(t))
	e.Fit()
	start := bytes.Clone(e.Document().SourceBytes)
	old, _ := editedMap(t, e)
	dragCell(e, image.Pt(10, 10), image.Pt(15, 15))
	if got := e.SelectedRecords(); len(got) != 1 || got[0].Kind != "Unit" || got[0].Index != 0 {
		t.Fatalf("the box selected %v", got)
	}
	dragCell(e, image.Pt(11, 11), image.Pt(21, 14))
	now, _ := editedMap(t, e)
	box, offset := image.Rect(10, 10, 16, 16), image.Pt(10, 3)
	for i := range 40 * 40 {
		c := image.Pt(i%40, i/40)
		want := old.Overlay[i]
		switch {
		case c.In(box.Add(offset)):
			from := c.Sub(offset)
			want = old.Overlay[from.Y*40+from.X]
		case c.In(box):
			want = 0
		}
		if now.Overlay[i] != want {
			t.Errorf("cell %v holds object %d, want %d", c, now.Overlay[i], want)
		}
		if now.Tiles[i] != old.Tiles[i] || now.Altitudes[i] != old.Altitudes[i] {
			t.Fatalf("cell %v changed its land", c)
		}
	}
	for i, d := range []image.Point{offset, {}} {
		if now.Units[i].X != old.Units[i].X+uint32(d.X*256) || now.Units[i].Y != old.Units[i].Y+uint32(d.Y*256) {
			t.Errorf("unit %d stands at %d,%d, from %d,%d", i, now.Units[i].X, now.Units[i].Y, old.Units[i].X, old.Units[i].Y)
		}
	}
	if s := e.StatusLine(); !strings.Contains(s, "objects moved: 4") {
		t.Errorf("the status line %q does not count the moved objects", s)
	}
	if err := e.Key("undo"); err != nil || !bytes.Equal(e.Document().SourceBytes, start) {
		t.Fatalf("one undo did not give back the map (%v)", err)
	}
	if m, err := alm.Open(e.Document().SourceBytes); err != nil || m.Overlay[11*40+11] != forestOakish {
		t.Fatalf("after the undo the tree at 11,11 is gone (%v)", err)
	}
}
