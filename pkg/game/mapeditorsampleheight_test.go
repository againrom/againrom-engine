package game

import (
	"bytes"
	"fmt"
	"image"
	"strings"
	"testing"
	"time"

	"againrom/internal/synth"
	"againrom/pkg/ui"
)

// The height brush's Sample height row on a map whose heights differ from
// cell to cell: the press on cell (20,20) samples its height, which the
// status line shows while the button is held, a drag to (26,20) sets every
// cell within the brush's radius of the path to it, no other cell changes,
// and one undo restores the bytes.
func TestMapEditorSampleHeightLevelsThePathToThePressedHeight(t *testing.T) {
	altitudes := make([]byte, 40*40)
	for i := range altitudes {
		altitudes[i] = byte(40 + (i%40+3*(i/40))%12)
	}
	e, _ := openGoalMap(t, synth.ALM(synth.ALMOptions{Width: 40, Height: 40, Altitudes: altitudes, Type7Payload: make([]byte, 12)}))
	from, to := image.Pt(20, 20), image.Pt(26, 20) // clear of the brush list
	e.Fit()
	_ = e.Key("height")
	for range 3 {
		_ = e.Key("down")
	}
	start := bytes.Clone(e.Document().SourceBytes)
	was, _ := editedMap(t, e)
	sample := was.Altitudes[from.Y*40+from.X]
	now := time.Unix(1, 0)
	for x := from.X; x <= to.X; x++ {
		p := e.CellPoint(image.Pt(x, from.Y))
		e.Pointer(ui.Input{CursorX: p.X, CursorY: p.Y, PrimaryDown: true}, now)
	}
	if s := e.StatusLine(); !strings.HasPrefix(s, fmt.Sprintf("%d,%d", to.X, to.Y)) || !strings.Contains(s, fmt.Sprintf("target %d", int(sample)-ui.HeightLabelZero)) {
		t.Fatalf("the status line %q does not show the sample while the button is held", s)
	}
	if !bytes.Equal(e.Document().SourceBytes, start) {
		t.Fatal("the map changed before the release")
	}
	p := e.CellPoint(to)
	e.Pointer(ui.Input{CursorX: p.X, CursorY: p.Y}, now)
	got, _ := editedMap(t, e)
	const radius = 2 // the brush's starting size
	for i := range got.Altitudes {
		c := image.Pt(i%40, i/40)
		near := false
		for x := from.X; x <= to.X; x++ {
			dx, dy := c.X-x, c.Y-from.Y
			near = near || dx*dx+dy*dy <= radius*radius
		}
		want := was.Altitudes[i]
		if near {
			want = sample
		}
		if got.Altitudes[i] != want {
			t.Fatalf("cell %v is at %d, want %d (sample %d)", c, got.Altitudes[i], want, sample)
		}
	}
	if err := e.Key("undo"); err != nil || !bytes.Equal(e.Document().SourceBytes, start) {
		t.Fatalf("one undo did not restore the map (%v)", err)
	}
}
